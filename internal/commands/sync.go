package commands

import (
	"fmt"
	"strings"

	"github.com/oschrenk/arbol/internal/config"
	"github.com/oschrenk/arbol/internal/git"
	"github.com/spf13/cobra"
)

var fetchFlag bool

var syncCmd = &cobra.Command{
	Use:   "sync [path]",
	Short: "Clone missing repositories",
	Long: `Clone missing repositories from the configuration.

Without a path argument, syncs all repos in the account.
With a path, syncs only repos under that path.

Use --fetch to also fetch updates for existing repositories.

Examples:
  arbol sync                    # sync all repos
  arbol sync work.backend       # sync repos under work.backend
  arbol sync personal.dotfiles  # sync single repo
  arbol sync --fetch            # sync all and fetch existing`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		account, accountName, err := getAccount()
		if err != nil {
			return err
		}

		pathFilter := ""
		if len(args) > 0 {
			pathFilter = args[0]
		}

		repos := account.GetRepos(pathFilter)
		if len(repos) == 0 {
			if pathFilter != "" {
				fmt.Printf("No repos found matching '%s' in account '%s'\n", pathFilter, accountName)
			} else {
				fmt.Printf("No repos configured in account '%s'\n", accountName)
			}
			return nil
		}

		var cloned, fetched, skipped, failed int

		for _, repo := range repos {
			displayPath := repo.Path + "." + repo.Name

			if git.Exists(repo.FullPath) {
				if fetchFlag {
					fmt.Printf("  fetch %s\n", displayPath)
					if err := git.Fetch(repo.FullPath); err != nil {
						fmt.Printf("  error %s: %v\n", displayPath, err)
						failed++
						continue
					}
					fetched++
				} else {
					fmt.Printf("  skip  %s (already exists)\n", displayPath)
					skipped++
				}
				continue
			}

			fmt.Printf("  clone %s\n", displayPath)
			if err := git.Clone(repo.Repo.URL, repo.FullPath); err != nil {
				fmt.Printf("  error %s: %v\n", displayPath, err)
				failed++
				continue
			}
			cloned++
		}

		// Build summary based on what was done
		var summary []string
		if cloned > 0 {
			summary = append(summary, fmt.Sprintf("%d cloned", cloned))
		}
		if fetched > 0 {
			summary = append(summary, fmt.Sprintf("%d fetched", fetched))
		}
		if skipped > 0 {
			summary = append(summary, fmt.Sprintf("%d skipped", skipped))
		}
		if failed > 0 {
			summary = append(summary, fmt.Sprintf("%d failed", failed))
		}
		if len(summary) == 0 {
			summary = append(summary, "nothing to do")
		}
		fmt.Printf("\nSummary: %s\n", strings.Join(summary, ", "))
		return nil
	},
	ValidArgsFunction: completeRepoPath,
}

func init() {
	syncCmd.Flags().BoolVar(&fetchFlag, "fetch", false, "Fetch updates for existing repos")
	rootCmd.AddCommand(syncCmd)
}

// completeRepoPath provides completion for repo paths
func completeRepoPath(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var account *config.Account
	if accountFlag != "" {
		account, err = cfg.GetAccount(accountFlag)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	} else {
		account, _, err = cfg.DefaultAccount()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}

	return account.RepoPaths(), cobra.ShellCompDirectiveNoFileComp
}
