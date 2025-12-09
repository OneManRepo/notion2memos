package cmd

import (
"github.com/OneManRepo/notion2memos/internal/config"
"github.com/OneManRepo/notion2memos/internal/migrate"
"github.com/spf13/cobra"
)

var (
resume       bool
filterTitles []string
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate pages from Notion to Memos",
	Long: `Searches for all pages in Notion and migrates them to Memos.
Supports filtering by exact page titles and resuming interrupted migrations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return err
		}

		// Create migrator
		migrator, err := migrate.NewMigrator(cfg, dryRun)
		if err != nil {
			return err
		}

		// Run migration
		opts := migrate.MigrateOptions{
			Resume:       resume,
			FilterTitles: filterTitles,
		}

		return migrator.Migrate(opts)
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().BoolVar(&resume, "resume", false, "resume migration from where it left off")
	migrateCmd.Flags().StringSliceVar(&filterTitles, "filter-title", []string{}, "filter pages by exact title (can be specified multiple times)")
}
