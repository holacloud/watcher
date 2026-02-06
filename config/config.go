package config

import (
	"time"

	"github.com/holacloud/watcher/telegram"
)

var VERSION = "dev"

type Config struct {
	Unit     string        `usage:"systemd unit name (e.g. nginx.service)"`
	State    string        `usage:"path to state file"`
	Timeout  time.Duration `usage:"timeout for systemctl command"`
	Cooldown time.Duration `usage:"minimum time between alerts"`
	DryRun   bool          `usage:"do not send telegram, only print"`
	Telegram telegram.Config

	ShowConfig bool `json:"show_config"`
	Version    bool `json:"version"`
}
