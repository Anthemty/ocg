package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	WorkspaceID string `json:"workspace_id"`
	AuthCookie  string `json:"auth_cookie"`
}

type UsageData struct {
	Rolling   Meter  `json:"rolling"`
	Weekly    Meter  `json:"weekly"`
	Monthly   Meter  `json:"monthly"`
	Plan      string `json:"plan"`
	FetchedAt string `json:"fetched_at"`
}

type Meter struct {
	Percent    int    `json:"percent"`
	ResetInSec int    `json:"reset_in_sec"`
	Status     string `json:"status"`
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "ocg")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func loadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func formatDuration(sec int) string {
	d := time.Duration(sec) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func bar(pct, width int) string {
	filled := pct * width / 100
	b := make([]rune, width)
	for i := 0; i < width; i++ {
		if i < filled {
			b[i] = '█'
		} else {
			b[i] = '░'
		}
	}
	return string(b)
}
