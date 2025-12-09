package cmd

import (
"fmt"

"github.com/OneManRepo/notion2memos/internal/config"
"github.com/spf13/cobra"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset migration state",
	Long:  `Clears the migration state file, allowing pages to be migrated again.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.ClearStateFile(); err != nil {
			return err
		}
		fmt.Println("Migration state has been reset")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)
}
