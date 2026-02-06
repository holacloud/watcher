package config

import "time"

var VERSION = "dev"

type Config struct {
	Unit     string        `usage:"systemd unit name (e.g. nginx.service)"`
	State    string        `usage:"path to state file"`
	Timeout  time.Duration `usage:"timeout for systemctl command"`
	Cooldown time.Duration `usage:"minimum time between alerts"`
	DryRun   bool          `usage:"do not send telegram, only print"`
	BotToken string        `usage:"telegram bot token"`
	BotID    string        `usage:"telegram bot id"`
	
	ShowConfig bool `json:"show_config"`
	Version    bool `json:"version"`
}
