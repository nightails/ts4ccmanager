package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"ts4ccmanager/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage ts4ccmanager configuration",
	Long:  `View and modify configuration settings such as the Sims 4 Mods directory and SQLite database path.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration and resolved paths",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		cfgPath, _ := config.DefaultConfigPath()
		resolvedModsDir, modsDirErr := cfg.ResolveModsDir()
		resolvedDBPath, _ := cfg.ResolveDBPath()

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
		fmt.Fprintf(w, "CONFIG FILE:\t%s\n", cfgPath)
		fmt.Fprintf(w, "MODS DIRECTORY (CONFIGURED):\t%s\n", cfg.ModsDir)
		if modsDirErr != nil {
			fmt.Fprintf(w, "MODS DIRECTORY (RESOLVED):\t<not found: %v>\n", modsDirErr)
		} else {
			fmt.Fprintf(w, "MODS DIRECTORY (RESOLVED):\t%s\n", resolvedModsDir)
		}
		fmt.Fprintf(w, "DB PATH (CONFIGURED):\t%s\n", cfg.DBPath)
		fmt.Fprintf(w, "DB PATH (RESOLVED):\t%s\n", resolvedDBPath)
		_ = w.Flush()

		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		key := args[0]
		val, err := cfg.Get(key)
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), val)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		key := args[0]
		val := args[1]

		if err := cfg.Set(key, val); err != nil {
			return err
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully set %s = %s\n", key, val)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}
