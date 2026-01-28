package commands

import (
	"fmt"
	"os"

	"github.com/oschrenk/arbol/internal/config"
	"github.com/spf13/cobra"
)

var (
	accountFlag string
	cfg         *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "arbol",
	Short: "Manage git repositories across machines",
	Long: `Arbol is a CLI tool for managing git repositories across multiple machines
using a declarative TOML configuration.

Define repositories in a config file, organize them in a tree structure,
and use accounts as machine profiles.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for these commands
		switch cmd.Name() {
		case "init", "completion", "version", "__complete-path", "__complete-account":
			return nil
		}
		if cmd.Parent() != nil && cmd.Parent().Name() == "completion" {
			return nil
		}

		// Load config
		var err error
		cfg, err = config.Load()
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&accountFlag, "account", "a", "", "Use specific account instead of default")

	// Register custom completion for --account flag
	rootCmd.RegisterFlagCompletionFunc("account", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cfg, err := config.Load()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return cfg.AccountNames(), cobra.ShellCompDirectiveNoFileComp
	})
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// getAccount returns the account to use based on flags or default
func getAccount() (*config.Account, string, error) {
	if accountFlag != "" {
		account, err := cfg.GetAccount(accountFlag)
		if err != nil {
			return nil, "", err
		}
		return account, accountFlag, nil
	}
	return cfg.DefaultAccount()
}

// loadConfigForCompletion loads config for completion commands
// These commands skip PersistentPreRunE so need to load config themselves
func loadConfigForCompletion() (*config.Config, error) {
	if cfg != nil {
		return cfg, nil
	}
	return config.Load()
}
