package commands

import (
	"os"

	"github.com/spf13/cobra"
)

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
	Hidden:                true,
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
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
