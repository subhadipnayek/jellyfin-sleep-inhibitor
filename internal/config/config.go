package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type JellyfinConfig struct {
	ServerURL              string `json:"server_url"`
	APIKey                 string `json:"api_key"`
	PollingIntervalSeconds int    `json:"polling_interval_seconds"`
	RequestTimeoutSeconds  int    `json:"request_timeout_seconds"`
}

type PowerConfig struct {
	IdleGracePeriodMinutes   int  `json:"idle_grace_period_minutes"`
	IgnorePaused             bool `json:"ignore_paused"`
	PausedGracePeriodMinutes int  `json:"paused_grace_period_minutes"`
	KeepAwakeOnNetworkError  bool `json:"keep_awake_on_network_error"`
	RequireAwayMode          bool `json:"require_away_mode"`
}

type LoggingConfig struct {
	Level       string `json:"level"`
	LogToFile   bool   `json:"log_to_file"`
	LogFilePath string `json:"log_file_path"`
}

type Config struct {
	Jellyfin JellyfinConfig `json:"jellyfin"`
	Power    PowerConfig    `json:"power"`
	Logging  LoggingConfig  `json:"logging"`
}

func DefaultConfig() Config {
	return Config{
		Jellyfin: JellyfinConfig{
			ServerURL:              "http://localhost:8096",
			APIKey:                 "",
			PollingIntervalSeconds: 15,
			RequestTimeoutSeconds:  10,
		},
		Power: PowerConfig{
			IdleGracePeriodMinutes:   5,
			IgnorePaused:             false,
			PausedGracePeriodMinutes: 15,
			KeepAwakeOnNetworkError:  true,
			RequireAwayMode:          true,
		},
		Logging: LoggingConfig{
			Level:       "info",
			LogToFile:   true,
			LogFilePath: "jellyfin_sleep_inhibitor.log",
		},
	}
}

// GetExeDir returns directory of current executable
func GetExeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func LoadConfig(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	if configPath == "" {
		// Look in current directory first, then executable directory
		candidates := []string{
			"config.json",
			filepath.Join(GetExeDir(), "config.json"),
		}
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				configPath = candidate
				break
			}
		}
	}

	if configPath == "" {
		return nil, fmt.Errorf("config.json not found in working directory or next to executable")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file (%s): %w", configPath, err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON (%s): %w", configPath, err)
	}

	// Validate configuration
	if cfg.Jellyfin.ServerURL == "" {
		return nil, fmt.Errorf("jellyfin.server_url cannot be empty")
	}
	if cfg.Jellyfin.APIKey == "" || cfg.Jellyfin.APIKey == "YOUR_JELLYFIN_API_KEY_HERE" {
		return nil, fmt.Errorf("jellyfin.api_key must be set with a valid Jellyfin API token")
	}
	if cfg.Jellyfin.PollingIntervalSeconds <= 0 {
		cfg.Jellyfin.PollingIntervalSeconds = 15
	}
	if cfg.Jellyfin.RequestTimeoutSeconds <= 0 {
		cfg.Jellyfin.RequestTimeoutSeconds = 10
	}
	if cfg.Power.IdleGracePeriodMinutes < 0 {
		cfg.Power.IdleGracePeriodMinutes = 0
	}

	return &cfg, nil
}
