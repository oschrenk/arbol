package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oschrenk/arbol/internal/config"
	"github.com/spf13/cobra"
)

const starterConfig = `[accounts.default]
default = true
root = "~/Projects"

# Simple path (no subpaths):
# repos.external = [
#   { url = "git@github.com:user/lib.git" },
# ]

# Path with subpaths - use "/" for repos directly in this directory:
# repos.personal."/" = [
#   { url = "git@github.com:user/dotfiles.git" },
# ]
# repos.personal.golang = [
#   { url = "git@github.com:user/myapp.git" },
# ]

# repos.work.backend = [
#   { url = "git@github.com:company/api.git" },
#   { url = "git@github.com:company/worker.git", name = "worker-svc" },
# ]
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a starter configuration file",
	Long: `Create a starter configuration file at the default config location.

The config file will be created at:
  $XDG_CONFIG_HOME/arbol/config.toml (or ~/.config/arbol/config.toml)

This command will fail if a config file already exists.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := config.ConfigPath()

		// Check if config already exists
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("config file already exists at %s", configPath)
		}

		// Create directory if needed
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// Write starter config
		if err := os.WriteFile(configPath, []byte(starterConfig), 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		fmt.Printf("Created config file at %s\n", configPath)
		fmt.Println("\nEdit the file to add your repositories, then run 'arbol sync' to clone them.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
