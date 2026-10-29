package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoadAndSave(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ts4ccmanager-config-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	// Test loading when file does not exist
	cfg, err := LoadFrom(configPath)
	if err != nil {
		t.Fatalf("unexpected error loading non-existent config: %v", err)
	}
	if cfg.ModsDir != "" || cfg.DBPath != "" {
		t.Errorf("expected empty initial config, got %+v", cfg)
	}

	// Test setting values
	if err := cfg.Set("mods_dir", "/custom/mods"); err != nil {
		t.Fatalf("failed to set mods_dir: %v", err)
	}
	if err := cfg.Set("db_path", "/custom/mods.db"); err != nil {
		t.Fatalf("failed to set db_path: %v", err)
	}

	// Test invalid key
	if err := cfg.Set("invalid_key", "value"); err == nil {
		t.Errorf("expected error setting invalid key, got nil")
	}

	// Test saving
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Test reloading
	loadedCfg, err := LoadFrom(configPath)
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}

	val, err := loadedCfg.Get("mods_dir")
	if err != nil || val != "/custom/mods" {
		t.Errorf("expected /custom/mods, got %q (err: %v)", val, err)
	}

	val, err = loadedCfg.Get("db_path")
	if err != nil || val != "/custom/mods.db" {
		t.Errorf("expected /custom/mods.db, got %q (err: %v)", val, err)
	}

	if _, err := loadedCfg.Get("invalid_key"); err == nil {
		t.Errorf("expected error getting invalid key, got nil")
	}
}

func TestResolveDBPath(t *testing.T) {
	cfg := &Config{DBPath: "/my/db.db"}
	resolved, err := cfg.ResolveDBPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != "/my/db.db" {
		t.Errorf("expected /my/db.db, got %q", resolved)
	}

	emptyCfg := &Config{}
	defaultDB, err := emptyCfg.ResolveDBPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if defaultDB == "" {
		t.Errorf("expected non-empty default DB path")
	}
}
