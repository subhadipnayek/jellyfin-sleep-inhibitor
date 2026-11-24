package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Jellyfin.ServerURL != "http://localhost:8096" {
		t.Errorf("expected default server url http://localhost:8096, got %s", cfg.Jellyfin.ServerURL)
	}
	if cfg.Power.IdleGracePeriodMinutes != 5 {
		t.Errorf("expected default idle grace period 5, got %d", cfg.Power.IdleGracePeriodMinutes)
	}
	if cfg.Jellyfin.PollingIntervalSeconds != 15 {
		t.Errorf("expected default polling interval 15, got %d", cfg.Jellyfin.PollingIntervalSeconds)
	}
}

func TestLoadConfig(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")

	content := `{
		"jellyfin": {
			"server_url": "http://192.168.1.50:8096",
			"api_key": "test_api_token_123",
			"polling_interval_seconds": 20,
			"request_timeout_seconds": 5
		},
		"power": {
			"idle_grace_period_minutes": 10,
			"ignore_paused": true,
			"paused_grace_period_minutes": 5,
			"keep_awake_on_network_error": false,
			"require_away_mode": false
		},
		"logging": {
			"level": "debug",
			"log_to_file": false,
			"log_file_path": "test.log"
		}
	}`

	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Jellyfin.ServerURL != "http://192.168.1.50:8096" {
		t.Errorf("unexpected server_url: %s", cfg.Jellyfin.ServerURL)
	}
	if cfg.Jellyfin.APIKey != "test_api_token_123" {
		t.Errorf("unexpected api_key: %s", cfg.Jellyfin.APIKey)
	}
	if cfg.Power.IdleGracePeriodMinutes != 10 {
		t.Errorf("unexpected idle_grace_period_minutes: %d", cfg.Power.IdleGracePeriodMinutes)
	}
	if !cfg.Power.IgnorePaused {
		t.Errorf("expected ignore_paused to be true")
	}
}

func TestLoadConfigMissingAPIKey(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")

	content := `{
		"jellyfin": {
			"server_url": "http://localhost:8096",
			"api_key": ""
		}
	}`
	_ = os.WriteFile(cfgPath, []byte(content), 0644)

	_, err := LoadConfig(cfgPath)
	if err == nil {
		t.Errorf("expected error for empty api_key, got nil")
	}
}
