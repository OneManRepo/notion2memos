package cmd

import (
"fmt"
"os"
"path/filepath"

"github.com/OneManRepo/notion2memos/internal/config"
"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long:  `Creates a configuration file template at ~/.notion2memos/config.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, err := config.GetConfigPath()
		if err != nil {
			return err
		}

		// Check if config already exists
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("config file already exists at %s", configPath)
		}

		// Create config directory
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// Create config template
		template := `# Notion2Memos Configuration
# Get your Notion token from https://www.notion.so/my-integrations
notion_token: "YOUR_NOTION_TOKEN_HERE"

# Your Memos instance URL (e.g., https://memos.example.com)
memos_url: "YOUR_MEMOS_URL_HERE"

# Get your Memos token from Settings -> Access Tokens in Memos
memos_token: "YOUR_MEMOS_TOKEN_HERE"
`

		if err := os.WriteFile(configPath, []byte(template), 0600); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		fmt.Printf("Configuration file created at: %s\n", configPath)
		fmt.Println("\nPlease edit the file and add your tokens before running migrations.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
