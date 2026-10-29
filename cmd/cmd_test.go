package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"ts4ccmanager/internal/config"
	"ts4ccmanager/internal/db"
)

func executeCommand(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	resetFlags(rootCmd)

	err := rootCmd.Execute()
	return buf.String(), err
}

func resetFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
	for _, sub := range cmd.Commands() {
		resetFlags(sub)
	}
}

func TestCLIWorkflow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ts4ccmanager-cli-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	modsDir := filepath.Join(tempDir, "Mods")
	if err := os.MkdirAll(modsDir, 0755); err != nil {
		t.Fatalf("failed to create mods dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test_mods.db")
	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")

	// Set custom config for tests
	cfg := &config.Config{
		ModsDir: modsDir,
		DBPath:  dbPath,
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save test config: %v", err)
	}

	// Override default config paths by setting XDG_CONFIG_HOME
	t.Setenv("XDG_CONFIG_HOME", configDir)

	// Save config to XDG location so config.Load() picks it up
	xdgConfigAppDir := filepath.Join(configDir, config.AppName)
	_ = os.MkdirAll(xdgConfigAppDir, 0755)
	_ = cfg.SaveTo(filepath.Join(xdgConfigAppDir, config.ConfigFileName))

	// 1. Test Config Set / Get
	_, err = executeCommand("config", "set", "mods_dir", modsDir)
	if err != nil {
		t.Fatalf("failed to run config set: %v", err)
	}

	// 2. Test Add Package
	pkgFile := filepath.Join(tempDir, "test_item.package")
	if err := os.WriteFile(pkgFile, []byte("item-pkg-data"), 0644); err != nil {
		t.Fatalf("failed to write test item: %v", err)
	}

	_, err = executeCommand("add", pkgFile, "--name", "Test Item", "--version", "1.0.0", "--author", "ArtistA")
	if err != nil {
		t.Fatalf("failed to add mod via CLI: %v", err)
	}

	// 3. Test List
	listOutput, err := executeCommand("list")
	if err != nil {
		t.Fatalf("failed to list mods via CLI: %v", err)
	}
	if !bytes.Contains([]byte(listOutput), []byte("Test Item")) {
		t.Errorf("expected list output to contain 'Test Item', got: %s", listOutput)
	}

	// 4. Test List JSON
	jsonOutput, err := executeCommand("list", "--json")
	if err != nil {
		t.Fatalf("failed to list mods as JSON: %v", err)
	}

	var mods []*db.Mod
	if err := json.Unmarshal([]byte(jsonOutput), &mods); err != nil {
		t.Fatalf("failed to parse JSON list output: %v, output was: %s", err, jsonOutput)
	}
	if len(mods) != 1 || mods[0].Name != "Test Item" {
		t.Errorf("unexpected JSON mods output: %+v", mods)
	}

	// 5. Test Update
	newPkgFile := filepath.Join(tempDir, "test_item_v2.package")
	if err := os.WriteFile(newPkgFile, []byte("item-pkg-data-updated"), 0644); err != nil {
		t.Fatalf("failed to write new package: %v", err)
	}

	_, err = executeCommand("update", "Test Item", newPkgFile, "--version", "2.0.0")
	if err != nil {
		t.Fatalf("failed to update mod via CLI: %v", err)
	}

	// 6. Test Remove
	_, err = executeCommand("remove", "Test Item")
	if err != nil {
		t.Fatalf("failed to remove mod via CLI: %v", err)
	}

	// Verify empty list
	emptyList, err := executeCommand("list")
	if err != nil {
		t.Fatalf("failed to list after removal: %v", err)
	}
	if !bytes.Contains([]byte(emptyList), []byte("No mods currently installed")) {
		t.Errorf("expected empty list message, got: %s", emptyList)
	}
}
