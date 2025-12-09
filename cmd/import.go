package cmd

import (
"fmt"

"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import markdown files to Memos (not implemented yet)",
	Long:  `Import markdown files from local directory to Memos.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("import command is not implemented yet")
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
}
