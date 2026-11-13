package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <id|name>",
	Short: "Remove a mod from the game and database",
	Long:  `Deletes the specified mod file or directory from the Sims 4 Mods directory and removes its entry from the database.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		identifier := args[0]

		svc, repo, _, err := initModService()
		if err != nil {
			return err
		}
		defer repo.Close()

		if err := svc.Remove(identifier); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully removed mod %q\n", identifier)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
