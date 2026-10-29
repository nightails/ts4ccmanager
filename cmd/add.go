package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"ts4ccmanager/internal/mod"
)

var addCmd = &cobra.Command{
	Use:   "add <file-path>",
	Short: "Add and install a mod file or archive",
	Long:  `Validates, copies, or extracts a .package, .ts4script, or .zip mod file into the Sims 4 Mods directory and records it in the database.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		svc, repo, _, err := initModService()
		if err != nil {
			return err
		}
		defer repo.Close()

		name, _ := cmd.Flags().GetString("name")
		version, _ := cmd.Flags().GetString("version")
		author, _ := cmd.Flags().GetString("author")

		installedMod, err := svc.Add(filePath, mod.AddOptions{
			Name:    name,
			Version: version,
			Author:  author,
		})
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully added mod %q (ID: %d)\n", installedMod.Name, installedMod.ID)
		fmt.Fprintf(cmd.OutOrStdout(), "  File:        %s\n", installedMod.FileName)
		fmt.Fprintf(cmd.OutOrStdout(), "  Destination: %s\n", installedMod.FilePath)
		if installedMod.Version != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  Version:     %s\n", installedMod.Version)
		}
		if installedMod.Author != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  Author:      %s\n", installedMod.Author)
		}

		return nil
	},
}

func init() {
	addCmd.Flags().StringP("name", "n", "", "Custom display name for the mod")
	addCmd.Flags().StringP("version", "v", "", "Mod version string")
	addCmd.Flags().StringP("author", "a", "", "Mod author/creator name")
	rootCmd.AddCommand(addCmd)
}
