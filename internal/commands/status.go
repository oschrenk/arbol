package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/oschrenk/arbol/internal/git"
	"github.com/spf13/cobra"
)

var (
	noColor     bool
	noHeaders   bool
	pathWidth   int
	branchWidth int
)

// ANSI color codes
const (
	colorReset   = "\033[0m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorRed     = "\033[31m"
	colorCyan    = "\033[36m"
	colorMagenta = "\033[35m"
	colorGray    = "\033[90m"
)

var statusCmd = &cobra.Command{
	Use:   "status [path]",
	Short: "Show status of repositories",
	Long: `Show the status of repositories including branch and sync state.

Without a path argument, shows status of all repos in the account.
With a path, shows only repos under that path.

Examples:
  arbol status                  # status of all repos
  arbol status work.backend     # status of repos under work.backend
  arbol status --account spare  # use specific account`,
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

		// Sort repos by display path for consistent output
		sort.Slice(repos, func(i, j int) bool {
			pathI := repos[i].Path + "." + repos[i].Name
			pathJ := repos[j].Path + "." + repos[j].Name
			return pathI < pathJ
		})

		// Fixed column widths (pathWidth and branchWidth come from flags)
		const workWidth = 5
		const remoteWidth = 8

		// Print header
		if !noHeaders {
			fmt.Printf("%-*s  %-*s  %-*s  %-*s  %s\n",
				pathWidth, "PATH",
				branchWidth, "BRANCH",
				workWidth, "WORK",
				remoteWidth, "REMOTE",
				"COMMENTS")
		}

		for _, repo := range repos {
			displayPath := repo.Path + "." + repo.Name
			printRepoStatus(displayPath, repo.FullPath, pathWidth, branchWidth, workWidth, remoteWidth)
		}

		return nil
	},
	ValidArgsFunction: completeRepoPath,
}

func init() {
	statusCmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	statusCmd.Flags().BoolVar(&noHeaders, "no-headers", false, "Hide column headers")
	statusCmd.Flags().IntVar(&pathWidth, "path-width", 30, "Width of PATH column")
	statusCmd.Flags().IntVar(&branchWidth, "branch-width", 15, "Width of BRANCH column")
	rootCmd.AddCommand(statusCmd)
}

func printRepoStatus(displayPath, fullPath string, pathWidth, branchWidth, workWidth, remoteWidth int) {
	// Truncate path if needed
	pathText := truncate(displayPath, pathWidth)

	// Check if repo exists
	if !git.Exists(fullPath) {
		path := padRight(pathText, pathWidth)
		branch := padRight(colorize(colorGray, "—"), branchWidth)
		work := padRight(colorize(colorGray, "—"), workWidth)
		remote := padRight(colorize(colorGray, "—"), remoteWidth)
		comment := colorize(colorGray, "not cloned")
		fmt.Printf("%s  %s  %s  %s  %s\n", path, branch, work, remote, comment)
		return
	}

	status, err := git.Status(fullPath)
	if err != nil {
		path := padRight(pathText, pathWidth)
		branch := padRight(colorize(colorGray, "?"), branchWidth)
		work := padRight(colorize(colorGray, "?"), workWidth)
		remote := padRight(colorize(colorGray, "?"), remoteWidth)
		comment := colorize(colorGray, err.Error())
		fmt.Printf("%s  %s  %s  %s  %s\n", path, branch, work, remote, comment)
		return
	}

	// Format path
	path := padRight(pathText, pathWidth)

	// Format branch (truncate with ellipsis if too long)
	branchText := truncate(status.Branch, branchWidth)
	var branch string
	if status.IsDetached {
		branch = padRight(colorize(colorCyan, branchText), branchWidth)
	} else {
		branch = padRight(branchText, branchWidth)
	}

	// Format work status
	var workText string
	if status.IsDirty {
		workText = fmt.Sprintf("● %d", status.DirtyFiles)
	} else {
		workText = "✔"
	}
	var work string
	if status.IsDirty {
		work = padRight(colorize(colorYellow, workText), workWidth)
	} else {
		work = padRight(colorize(colorGreen, workText), workWidth)
	}

	// Format remote status
	var remoteText string
	var comments []string

	if status.IsDetached {
		remoteText = "✔"
		comments = append(comments, "detached HEAD")
	} else if status.NoTracking {
		remoteText = "↑?"
		comments = append(comments, "no tracking branch")
	} else if status.Ahead > 0 && status.Behind > 0 {
		remoteText = fmt.Sprintf("↓%d ↑%d", status.Behind, status.Ahead)
		comments = append(comments, "diverged")
	} else if status.Behind > 0 {
		remoteText = fmt.Sprintf("↓%d", status.Behind)
		comments = append(comments, fmt.Sprintf("%d commits behind origin", status.Behind))
	} else if status.Ahead > 0 {
		remoteText = fmt.Sprintf("↑%d", status.Ahead)
		comments = append(comments, fmt.Sprintf("%d unpushed commits", status.Ahead))
	} else {
		remoteText = "✔"
	}

	var remote string
	if status.IsDetached || (!status.NoTracking && status.Ahead == 0 && status.Behind == 0) {
		remote = padRight(colorize(colorGreen, remoteText), remoteWidth)
	} else if status.NoTracking || status.Ahead > 0 && status.Behind == 0 {
		remote = padRight(colorize(colorYellow, remoteText), remoteWidth)
	} else if status.Behind > 0 && status.Ahead == 0 {
		remote = padRight(colorize(colorRed, remoteText), remoteWidth)
	} else {
		remote = padRight(colorize(colorMagenta, remoteText), remoteWidth)
	}

	// Add dirty files comment
	if status.IsDirty {
		comments = append([]string{fmt.Sprintf("%d dirty files", status.DirtyFiles)}, comments...)
	}

	comment := colorize(colorGray, strings.Join(comments, ", "))

	fmt.Printf("%s  %s  %s  %s  %s\n", path, branch, work, remote, comment)
}

// truncate shortens a string to maxLen, adding ellipsis if truncated
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

// padRight pads a string to width, accounting for ANSI color codes
func padRight(s string, width int) string {
	// Count visible length (excluding ANSI codes)
	visible := 0
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
		} else if inEscape && r == 'm' {
			inEscape = false
		} else if !inEscape {
			visible++
		}
	}
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}

func colorize(color, text string) string {
	if noColor || !isTerminal() {
		// Strip any existing ANSI codes and return plain text
		return stripAnsi(text)
	}
	return color + text + colorReset
}

func isTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func stripAnsi(s string) string {
	// Simple approach - if no color, we won't have added any codes
	return s
}
