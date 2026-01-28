package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oschrenk/arbol/internal/config"
	"github.com/spf13/cobra"
)

var completePathCmd = &cobra.Command{
	Use:    "__complete-path [prefix]",
	Short:  "Output path completions (for shell completion)",
	Hidden: true,
	Args:   cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfigForCompletion()
		if err != nil {
			return nil // Silent fail for completion
		}

		var account *config.Account
		if accountFlag != "" {
			account, err = cfg.GetAccount(accountFlag)
		} else {
			account, _, err = cfg.DefaultAccount()
		}
		if err != nil {
			return nil // Silent fail for completion
		}

		prefix := ""
		if len(args) > 0 {
			prefix = args[0]
		}

		paths := account.RepoPaths()
		sort.Strings(paths)

		for _, path := range paths {
			if prefix == "" || strings.HasPrefix(path, prefix) {
				fmt.Println(path)
			}
		}

		return nil
	},
}

var completeAccountCmd = &cobra.Command{
	Use:    "__complete-account [prefix]",
	Short:  "Output account completions (for shell completion)",
	Hidden: true,
	Args:   cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config directly since we're completing accounts
		cfg, err := loadConfigForCompletion()
		if err != nil {
			return nil // Silent fail for completion
		}

		prefix := ""
		if len(args) > 0 {
			prefix = args[0]
		}

		names := cfg.AccountNames()
		sort.Strings(names)

		for _, name := range names {
			if prefix == "" || strings.HasPrefix(name, prefix) {
				fmt.Println(name)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(completePathCmd)
	rootCmd.AddCommand(completeAccountCmd)
}
