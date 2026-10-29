package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"ts4ccmanager/internal/mod"
)

var updateCmd = &cobra.Command{
	Use:   "update <id|name> <new-file-path>",
	Short: "Update an existing mod with a new file",
	Long:  `Replaces an existing mod's file on disk, updates checksums, and refreshes metadata in the database.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		identifier := args[0]
		newFilePath := args[1]

		svc, repo, _, err := initModService()
		if err != nil {
			return err
		}
		defer repo.Close()

		name, _ := cmd.Flags().GetString("name")
		version, _ := cmd.Flags().GetString("version")
		author, _ := cmd.Flags().GetString("author")

		updatedMod, err := svc.Update(identifier, newFilePath, mod.UpdateOptions{
			Name:    name,
			Version: version,
			Author:  author,
		})
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated mod %q (ID: %d)\n", updatedMod.Name, updatedMod.ID)
		fmt.Fprintf(cmd.OutOrStdout(), "  New File:    %s\n", updatedMod.FileName)
		fmt.Fprintf(cmd.OutOrStdout(), "  Destination: %s\n", updatedMod.FilePath)
		if updatedMod.Version != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  Version:     %s\n", updatedMod.Version)
		}
		if updatedMod.Author != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  Author:      %s\n", updatedMod.Author)
		}

		return nil
	},
}

func init() {
	updateCmd.Flags().StringP("name", "n", "", "Update display name for the mod")
	updateCmd.Flags().StringP("version", "v", "", "Update mod version string")
	updateCmd.Flags().StringP("author", "a", "", "Update mod author/creator name")
	rootCmd.AddCommand(updateCmd)
}
