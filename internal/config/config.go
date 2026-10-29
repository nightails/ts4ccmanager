package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"ts4ccmanager/internal/detector"
)

const (
	AppName        = "ts4ccmanager"
	ConfigFileName = "config.json"
	DBFileName     = "mods.db"

	KeyModsDir = "mods_dir"
	KeyDBPath  = "db_path"
)

// Config holds user configuration settings for ts4ccmanager.
type Config struct {
	ModsDir string `json:"mods_dir"`
	DBPath  string `json:"db_path"`
}

// DefaultConfigDir returns the default configuration directory (~/.config/ts4ccmanager on Unix).
func DefaultConfigDir() (string, error) {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to determine user home or config directory: %w", err)
		}
		baseDir = filepath.Join(home, ".config")
	}
	return filepath.Join(baseDir, AppName), nil
}

// DefaultConfigPath returns the full path to config.json.
func DefaultConfigPath() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigFileName), nil
}

// DefaultDBPath returns the full path to the default SQLite database file.
func DefaultDBPath() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DBFileName), nil
}

// Load reads the configuration from the default location, or returns defaults if the file does not exist.
func Load() (*Config, error) {
	path, err := DefaultConfigPath()
	if err != nil {
		return nil, err
	}
	return LoadFrom(path)
}

// LoadFrom reads configuration from a specific file path.
func LoadFrom(path string) (*Config, error) {
	cfg := &Config{}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	return cfg, nil
}

// Save writes the configuration to the default config file path.
func (c *Config) Save() error {
	path, err := DefaultConfigPath()
	if err != nil {
		return err
	}
	return c.SaveTo(path)
}

// SaveTo writes the configuration to a specific file path.
func (c *Config) SaveTo(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Get retrieves the value for a given configuration key.
func (c *Config) Get(key string) (string, error) {
	switch key {
	case KeyModsDir:
		return c.ModsDir, nil
	case KeyDBPath:
		return c.DBPath, nil
	default:
		return "", fmt.Errorf("unknown configuration key: %q (valid keys: %s, %s)", key, KeyModsDir, KeyDBPath)
	}
}

// Set updates the value for a given configuration key.
func (c *Config) Set(key, value string) error {
	switch key {
	case KeyModsDir:
		c.ModsDir = value
	case KeyDBPath:
		c.DBPath = value
	default:
		return fmt.Errorf("unknown configuration key: %q (valid keys: %s, %s)", key, KeyModsDir, KeyDBPath)
	}
	return nil
}

// ResolveModsDir returns the explicitly configured mods directory or attempts Steam detection.
func (c *Config) ResolveModsDir() (string, error) {
	if c.ModsDir != "" {
		if _, err := os.Stat(c.ModsDir); os.IsNotExist(err) {
			return c.ModsDir, fmt.Errorf("configured mods_dir does not exist: %s", c.ModsDir)
		}
		return c.ModsDir, nil
	}

	detected, err := detector.DetectSteamModsPath()
	if err != nil {
		return "", fmt.Errorf("mods directory is not configured and automatic Steam detection failed: %w (use 'ts4ccmanager config set mods_dir <path>' to set it manually)", err)
	}

	return detected, nil
}

// ResolveDBPath returns the configured db path or defaults to ~/.config/ts4ccmanager/mods.db.
func (c *Config) ResolveDBPath() (string, error) {
	if c.DBPath != "" {
		return c.DBPath, nil
	}
	return DefaultDBPath()
}
