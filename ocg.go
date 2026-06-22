package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ---------- Config ----------

type Config struct {
	ActiveProvider string         `json:"active_provider"` // "opencode" | "deepseek" | "minimax"
	OpenCode       OpenCodeConfig `json:"opencode"`
	DeepSeek       DeepSeekConfig `json:"deepseek"`
	Minimax        MinimaxConfig  `json:"minimax"`
}

type OpenCodeConfig struct {
	WorkspaceID string `json:"workspace_id"`
	AuthCookie  string `json:"auth_cookie"`
}

type DeepSeekConfig struct {
	APIKey string `json:"api_key"`
}

type MinimaxConfig struct {
	APIKey string `json:"api_key"`
}

// ---------- OpenCode types (keep for backward compat during migration) ----------

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

// ---------- Unified fetch result ----------

// ProviderFetchResult holds the data for one provider after a fetch cycle.
type ProviderFetchResult struct {
	Criticality int          // 0-100, for icon colour
	Err         error        // non-nil if fetch failed
	Lines       []string     // fallback formatted display lines
	Meters      []UsageMeter // structured usage rows for progress rendering
}

type UsageMeter struct {
	Label   string `json:"label"`
	Percent int    `json:"percent"`
	Detail  string `json:"detail"`
}

// providerCache holds the latest fetch result per provider.
var (
	providerCache = make(map[string]*ProviderFetchResult)
	cacheMu       sync.RWMutex
	lastUpdated   time.Time
)

// ---------- Config file ----------

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
			return &Config{ActiveProvider: "opencode"}, nil
		}
		return nil, err
	}
	// Try new format first.
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err == nil {
		// Detect old-format: top-level fields set.
		var old legacyConfig
		if json.Unmarshal(data, &old) == nil && (old.WorkspaceID != "" || old.AuthCookie != "") {
			cfg.OpenCode.WorkspaceID = old.WorkspaceID
			cfg.OpenCode.AuthCookie = old.AuthCookie
			if cfg.ActiveProvider == "" {
				cfg.ActiveProvider = "opencode"
			}
			// Persist migrated format.
			_ = saveConfig(&cfg)
		}
		if cfg.ActiveProvider == "" {
			cfg.ActiveProvider = "opencode"
		}
		return &cfg, nil
	}
	// If even legacy parse fails, return empty.
	return &Config{ActiveProvider: "opencode"}, nil
}

// legacyConfig mirrors the old single-provider format for migration.
type legacyConfig struct {
	WorkspaceID string `json:"workspace_id"`
	AuthCookie  string `json:"auth_cookie"`
}

func saveConfig(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if cfg.ActiveProvider == "" {
		cfg.ActiveProvider = "opencode"
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// ---------- Helpers ----------

func formatDuration(sec int) string {
	d := time.Duration(sec) * time.Second
	days := int(d.Hours()) / 24
	h := int(d.Hours()) % 24
	m := int(d.Minutes()) % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh", days, h)
	case h > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	default:
		return fmt.Sprintf("%dm", m)
	}
}

// statusDot returns a coloured circle emoji by usage level.
func statusDot(pct int) string {
	switch {
	case pct < 50:
		return "🟢"
	case pct < 85:
		return "🟡"
	default:
		return "🔴"
	}
}
