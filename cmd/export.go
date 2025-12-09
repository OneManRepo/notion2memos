package cmd

import (
"fmt"

"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export pages from Notion (not implemented yet)",
	Long:  `Export pages from Notion to local files without importing to Memos.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("export command is not implemented yet")
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
}
