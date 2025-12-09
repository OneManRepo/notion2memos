package cmd

import (
"os"

"github.com/spf13/cobra"
)

var (
cfgFile string
dryRun  bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "notion2memos",
	Short: "Migrate notes from Notion to Memos",
	Long: `notion2memos is a CLI tool to migrate your notes from Notion to Memos.
It supports filtering by title, resume capability, and dry-run mode.`,
}

// Execute adds all child commands to the root command and sets flags appropriately
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.notion2memos/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "run without actually creating memos (saves to ./dry-run-output/)")
}
