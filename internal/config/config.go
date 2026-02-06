package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
)

const configFileName = "config.json"

// Config represents user-configurable CLI defaults.
type Config struct {
	Account string `json:"account,omitempty"`
	Output  string `json:"output,omitempty"` // text|json
	Color   string `json:"color,omitempty"`  // auto|always|never
	Debug   bool   `json:"debug,omitempty"`
}

// Default returns a Config with default values.
func Default() *Config {
	return &Config{
		Output: "text",
		Color:  "auto",
		Debug:  false,
	}
}

// ConfigPath returns the config file path, honoring THREADS_CONFIG if set.
func ConfigPath() string {
	if path := os.Getenv("THREADS_CONFIG"); path != "" {
		return path
	}
	return filepath.Join(ConfigDir(), configFileName)
}

// Load reads config from disk and applies environment overrides.
func Load() (*Config, error) {
	cfg, err := LoadFile(ConfigPath())
	if err != nil {
		return nil, err
	}
	applyEnv(cfg)
	return cfg, nil
}

// LoadFile reads config from a specific path without applying env overrides.
func LoadFile(path string) (*Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path) //nolint:gosec // The config path is chosen by the local user running the CLI.
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes config to disk, creating the config directory if needed.
func Save(cfg *Config) error {
	if err := EnsureConfigDir(); err != nil {
		return err
	}
	path := ConfigPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// applyEnv overrides config values from environment variables.
func applyEnv(cfg *Config) {
	if val := os.Getenv("THREADS_ACCOUNT"); val != "" {
		cfg.Account = val
	}
	if val := os.Getenv("THREADS_OUTPUT"); val != "" {
		cfg.Output = val
	}
	if val := os.Getenv("THREADS_COLOR"); val != "" {
		cfg.Color = val
	}
	if val := os.Getenv("THREADS_DEBUG"); val != "" {
		if parsed, err := strconv.ParseBool(val); err == nil {
			cfg.Debug = parsed
		} else {
			cfg.Debug = true
		}
	}
	if os.Getenv("NO_COLOR") != "" {
		cfg.Color = "never"
	}
}
