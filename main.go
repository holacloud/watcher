package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fulldump/goconfig"

	"github.com/holacloud/watcher/config"
	"github.com/holacloud/watcher/telegram"
)

type State struct {
	LastUptimeMS int64 `json:"last_uptime_ms"`
	LastAlertMS  int64 `json:"last_alert_ms"`
}

func main() {

	c := &config.Config{}

	goconfig.Read(&c)

	if c.Version {
		fmt.Println(config.VERSION)
		os.Exit(0)
	}

	if c.ShowConfig {
		e := json.NewEncoder(os.Stdout)
		e.SetIndent("", "  ")
		e.Encode(c)
		os.Exit(0)
	}

	if strings.TrimSpace(c.Unit) == "" {
		fmt.Fprintln(os.Stderr, "missing Unit (provide -unit=... or UNIT env var)")
		os.Exit(2)
	}

	now := time.Now()

	st, _ := loadState(c.State)
	tg := telegram.New(c.Telegram)

	uptimeMS, activeState, err := getUptimeMS(c.Unit, c.Timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: getUptimeMS: %v\n", err)
		os.Exit(1)
	}

	// Opcional: alerta si no está active
	if activeState != "active" {
		msg := fmt.Sprintf("⚠️ Service %s state=%s (uptime=%s)", c.Unit, activeState, formatDurMS(uptimeMS))
		if shouldAlert(st, now, c.Cooldown) {
			alert(c.DryRun, tg, msg)
			st.LastAlertMS = now.UnixMilli()
		}
		st.LastUptimeMS = uptimeMS
		_ = saveState(c.State, st)
		return
	}

	const toleranceMS int64 = 250 // jitter
	if st.LastUptimeMS > 0 && uptimeMS+toleranceMS < st.LastUptimeMS {
		msg := fmt.Sprintf("🚨 RESTART detected: %s uptime dropped %s → %s (state=%s)",
			c.Unit, formatDurMS(st.LastUptimeMS), formatDurMS(uptimeMS), activeState)

		if shouldAlert(st, now, c.Cooldown) {
			alert(c.DryRun, tg, msg)
			st.LastAlertMS = now.UnixMilli()
		} else {
			fmt.Println("restart detected but in cooldown; skipping alert")
		}
	}

	st.LastUptimeMS = uptimeMS
	if err := saveState(c.State, st); err != nil {
		fmt.Fprintf(os.Stderr, "WARN: saveState: %v\n", err)
	}
}

func alert(dry bool, tg *telegram.Telegram, msg string) {
	if dry {
		fmt.Println("[DRY-RUN]", msg)
		return
	}
	if err := sendTelegram(tg, msg); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: sendTelegram: %v\n", err)
	}
}

// getUptimeMS returns (uptimeMillisSinceActive, ActiveState, error).
func getUptimeMS(unit string, timeout time.Duration) (int64, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", "show", unit,
		"-p", "ActiveEnterTimestampMonotonic",
		"-p", "ActiveState",
	)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return 0, "", fmt.Errorf("systemctl show failed: %v: %s", err, strings.TrimSpace(string(ee.Stderr)))
		}
		return 0, "", fmt.Errorf("systemctl show failed: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	var enterUS int64 = -1
	activeState := ""
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		if strings.HasPrefix(ln, "ActiveEnterTimestampMonotonic=") {
			v := strings.TrimPrefix(ln, "ActiveEnterTimestampMonotonic=")
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return 0, "", fmt.Errorf("parse ActiveEnterTimestampMonotonic=%q: %w", v, err)
			}
			enterUS = n
		} else if strings.HasPrefix(ln, "ActiveState=") {
			activeState = strings.TrimPrefix(ln, "ActiveState=")
		}
	}

	if enterUS < 0 {
		return 0, activeState, fmt.Errorf("missing ActiveEnterTimestampMonotonic in systemctl output")
	}
	if activeState == "" {
		return 0, activeState, fmt.Errorf("missing ActiveState in systemctl output")
	}

	if activeState != "active" || enterUS == 0 {
		return 0, activeState, nil
	}

	bootUS, err := readProcUptimeUS()
	if err != nil {
		return 0, activeState, err
	}

	uptimeUS := bootUS - enterUS
	if uptimeUS < 0 {
		uptimeUS = 0
	}
	return uptimeUS / 1000, activeState, nil
}

func readProcUptimeUS() (int64, error) {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, fmt.Errorf("read /proc/uptime: %w", err)
	}
	fields := strings.Fields(string(b))
	if len(fields) < 1 {
		return 0, fmt.Errorf("unexpected /proc/uptime format: %q", strings.TrimSpace(string(b)))
	}
	f, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("parse /proc/uptime seconds=%q: %w", fields[0], err)
	}
	return int64(f * 1_000_000), nil
}

func shouldAlert(st State, now time.Time, cooldown time.Duration) bool {
	if st.LastAlertMS == 0 {
		return true
	}
	last := time.UnixMilli(st.LastAlertMS)
	return now.Sub(last) >= cooldown
}

func formatDurMS(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.2fh", d.Hours())
}

func loadState(path string) (State, error) {
	var st State
	b, err := os.ReadFile(path)
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return State{}, err
	}
	return st, nil
}

func saveState(path string, st State) error {
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0o755)

	tmp := path + ".tmp"
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func sendTelegram(tg *telegram.Telegram, text string) error {
	return tg.SendMessageSync(text)
}
