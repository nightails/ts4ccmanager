package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"ts4ccmanager/internal/db"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tracked mods",
	Long:  `Displays all mods and custom content currently registered in the ts4ccmanager database.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, _, err := initDB()
		if err != nil {
			return err
		}
		defer repo.Close()

		mods, err := repo.ListMods()
		if err != nil {
			return fmt.Errorf("failed to list mods: %w", err)
		}

		isJSON, _ := cmd.Flags().GetBool("json")
		if isJSON {
			output := mods
			if output == nil {
				output = []*db.Mod{}
			}
			data, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to format JSON: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}

		if len(mods) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No mods currently installed.")
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tFILE NAME\tVERSION\tAUTHOR\tSTATUS\tINSTALLED AT")
		for _, m := range mods {
			installedStr := m.InstalledAt.Format("2006-01-02 15:04:05")
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
				m.ID, m.Name, m.FileName, m.Version, m.Author, m.Status, installedStr)
		}
		_ = w.Flush()

		return nil
	},
}

func init() {
	listCmd.Flags().Bool("json", false, "Output list formatted as JSON")
	rootCmd.AddCommand(listCmd)
}
