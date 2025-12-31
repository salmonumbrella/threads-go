package config

import (
	"os"
	"path/filepath"
	"runtime"
)

const appName = "threads-cli"

// ConfigDir returns the configuration directory path
func ConfigDir() string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", appName)
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, appName)
	}
	return filepath.Join(os.Getenv("HOME"), ".config", appName)
}

// DataDir returns the data directory path
func DataDir() string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", appName)
	}
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, appName)
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "share", appName)
}

// CacheDir returns the cache directory path
func CacheDir() string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(os.Getenv("HOME"), "Library", "Caches", appName)
	}
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, appName)
	}
	return filepath.Join(os.Getenv("HOME"), ".cache", appName)
}

// EnsureConfigDir creates the config directory if it doesn't exist
func EnsureConfigDir() error {
	return os.MkdirAll(ConfigDir(), 0o700)
}
