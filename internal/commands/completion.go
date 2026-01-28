package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const fishCompletion = `# Fish completion for arbol

# Disable file completions for arbol
complete -c arbol -f

# Commands
complete -c arbol -n "__fish_use_subcommand" -a "sync" -d "Clone missing repositories"
complete -c arbol -n "__fish_use_subcommand" -a "status" -d "Show status of repositories"
complete -c arbol -n "__fish_use_subcommand" -a "init" -d "Create a starter configuration file"
complete -c arbol -n "__fish_use_subcommand" -a "version" -d "Print version information"
complete -c arbol -n "__fish_use_subcommand" -a "completion" -d "Generate shell completion scripts"
complete -c arbol -n "__fish_use_subcommand" -a "help" -d "Help about any command"

# Global flags
complete -c arbol -s a -l account -d "Use specific account instead of default" -xa "(arbol __complete-account)"
complete -c arbol -s h -l help -d "Show help"

# Completion subcommand
complete -c arbol -n "__fish_seen_subcommand_from completion" -a "bash zsh fish powershell"

# Sync flags
complete -c arbol -n "__fish_seen_subcommand_from sync" -l fetch -d "Fetch updates for existing repos"

# Path completions for sync and status
complete -c arbol -n "__fish_seen_subcommand_from sync status" -xa "(arbol __complete-path (commandline -ct))"
`

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for arbol.

To load completions:

Bash:
  $ source <(arbol completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ arbol completion bash > /etc/bash_completion.d/arbol
  # macOS:
  $ arbol completion bash > $(brew --prefix)/etc/bash_completion.d/arbol

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. Execute once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ arbol completion zsh > "${fpath[1]}/_arbol"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ arbol completion fish | source
  # To load completions for each session, execute once:
  $ arbol completion fish > ~/.config/fish/completions/arbol.fish

PowerShell:
  PS> arbol completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, run:
  PS> arbol completion powershell > arbol.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			fmt.Print(fishCompletion)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
