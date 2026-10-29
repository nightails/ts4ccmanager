package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"ts4ccmanager/internal/config"
	"ts4ccmanager/internal/db"
	"ts4ccmanager/internal/mod"
)

var rootCmd = &cobra.Command{
	Use:   "ts4ccmanager",
	Short: "A lightweight mod manager for The Sims 4",
	Long: `ts4ccmanager is a lightweight, terminal-based mod and custom content (CC) manager for The Sims 4.
It simplifies installing, updating, tracking, and removing mods with an embedded SQLite database.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func initDB() (*db.SQLiteRepository, *config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	dbPath, err := cfg.ResolveDBPath()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve database path: %w", err)
	}

	database, err := db.Open(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database at %s: %w", dbPath, err)
	}

	return db.NewRepository(database), cfg, nil
}

func initModService() (*mod.Service, *db.SQLiteRepository, *config.Config, error) {
	repo, cfg, err := initDB()
	if err != nil {
		return nil, nil, nil, err
	}

	modsDir, err := cfg.ResolveModsDir()
	if err != nil {
		_ = repo.Close()
		return nil, nil, nil, err
	}

	svc := mod.NewService(repo, modsDir)
	return svc, repo, cfg, nil
}
