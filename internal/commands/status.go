package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/oschrenk/arbol/internal/config"
	"github.com/oschrenk/arbol/internal/git"
	"github.com/spf13/cobra"
)

var (
	noColor      bool
	noHeaders    bool
	pathWidth    int
	branchWidth  int
	plainOutput  bool
	localDirty   bool
	remoteDirty  bool
)

// repoState pairs a configured repo with its resolved git status.
// status is nil when the repo is not cloned (exists == false) or when
// git.Status returned an error (err != nil).
type repoState struct {
	repo   config.RepoWithPath
	exists bool
	status *git.RepoStatus
	err    error
}

type jsonBranch struct {
	Name     string `json:"name"`
	Detached bool   `json:"detached"`
}

type jsonChanges struct {
	Dirty      bool   `json:"dirty"`
	Files      int    `json:"files"`
	LastCommit string `json:"last_commit"`
}

type jsonRemote struct {
	Ahead    int  `json:"ahead"`
	Behind   int  `json:"behind"`
	Diverged bool `json:"diverged"`
	Tracking bool `json:"tracking"`
}

type jsonRepo struct {
	ID      string       `json:"id"`
	Path    string       `json:"path"`
	Branch  *jsonBranch  `json:"branch,omitempty"`
	Changes *jsonChanges `json:"changes,omitempty"`
	Remote  *jsonRemote  `json:"remote,omitempty"`
}

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
				if plainOutput {
					fmt.Printf("No repos found matching '%s' in account '%s'\n", pathFilter, accountName)
				} else {
					return fmt.Errorf("no repos found matching '%s' in account '%s'", pathFilter, accountName)
				}
			} else {
				if plainOutput {
					fmt.Printf("No repos configured in account '%s'\n", accountName)
				} else {
					return fmt.Errorf("no repos configured in account '%s'", accountName)
				}
			}
			return nil
		}

		// Sort repos by display path for consistent output
		sort.Slice(repos, func(i, j int) bool {
			pathI := repos[i].Path + "." + repos[i].Name
			pathJ := repos[j].Path + "." + repos[j].Name
			return pathI < pathJ
		})

		// Resolve git status once per repo, then optionally filter to dirty.
		states := gatherStates(repos)
		states = filterStates(states, localDirty, remoteDirty)

		if plainOutput {
			return printPlainStatus(states)
		}
		return printJSONStatus(states)
	},
	ValidArgsFunction: completeRepoPath,
}

func init() {
	statusCmd.Flags().BoolVar(&plainOutput, "plain", false, "Show table output instead of JSON")
	statusCmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colored output (only with --plain)")
	statusCmd.Flags().BoolVar(&noHeaders, "no-headers", false, "Hide column headers (only with --plain)")
	statusCmd.Flags().IntVar(&pathWidth, "path-width", 30, "Width of PATH column (only with --plain)")
	statusCmd.Flags().IntVar(&branchWidth, "branch-width", 15, "Width of BRANCH column (only with --plain)")
	statusCmd.Flags().BoolVar(&localDirty, "local-dirty", false, "Only show repos with uncommitted local changes")
	statusCmd.Flags().BoolVar(&remoteDirty, "remote-dirty", false, "Only show repos out of sync with their remote (ahead, behind, or no tracking branch)")
	rootCmd.AddCommand(statusCmd)
}

// gatherStates resolves the git status of each repo exactly once.
func gatherStates(repos []config.RepoWithPath) []repoState {
	states := make([]repoState, 0, len(repos))
	for _, repo := range repos {
		st := repoState{repo: repo}
		if git.Exists(repo.FullPath) {
			st.exists = true
			if status, err := git.Status(repo.FullPath); err != nil {
				st.err = err
			} else {
				st.status = status
			}
		}
		states = append(states, st)
	}
	return states
}

// isLocalDirty reports whether the working tree has uncommitted changes.
func isLocalDirty(s *git.RepoStatus) bool {
	return s.IsDirty
}

// isRemoteDirty reports whether the branch is out of sync with its remote:
// ahead, behind, or without a tracking branch.
func isRemoteDirty(s *git.RepoStatus) bool {
	return s.NoTracking || s.Ahead > 0 || s.Behind > 0
}

// filterStates keeps only dirty repos when --local-dirty and/or --remote-dirty are set.
// When both flags are set the match is a union (local OR remote dirty).
// Repos without a resolved status (not cloned or errored) are excluded while
// filtering, since their dirtiness can't be determined.
func filterStates(states []repoState, local, remote bool) []repoState {
	if !local && !remote {
		return states
	}
	filtered := make([]repoState, 0, len(states))
	for _, st := range states {
		if st.status == nil {
			continue
		}
		if (local && isLocalDirty(st.status)) || (remote && isRemoteDirty(st.status)) {
			filtered = append(filtered, st)
		}
	}
	return filtered
}

func printPlainStatus(states []repoState) error {
	const workWidth = 5
	const remoteWidth = 8
	const ageWidth = 6

	if !noHeaders {
		fmt.Printf("%-*s  %-*s  %-*s  %-*s  %-*s  %s\n",
			pathWidth, "PATH",
			branchWidth, "BRANCH",
			workWidth, "WORK",
			remoteWidth, "REMOTE",
			ageWidth, "AGE",
			"COMMENTS")
	}

	for _, st := range states {
		displayPath := st.repo.Path + "." + st.repo.Name
		printRepoStatus(st, displayPath, pathWidth, branchWidth, workWidth, remoteWidth, ageWidth)
	}
	return nil
}

func printJSONStatus(states []repoState) error {
	var results []jsonRepo

	for _, st := range states {
		displayPath := st.repo.Path + "." + st.repo.Name
		entry := jsonRepo{
			ID:   displayPath,
			Path: st.repo.FullPath,
		}

		if !st.exists || st.err != nil || st.status == nil {
			results = append(results, entry)
			continue
		}

		status := st.status

		entry.Branch = &jsonBranch{
			Name:     status.Branch,
			Detached: status.IsDetached,
		}

		lastCommit := ""
		if !status.LastCommitTime.IsZero() {
			lastCommit = status.LastCommitTime.Format(time.RFC3339)
		}
		entry.Changes = &jsonChanges{
			Dirty:      status.IsDirty,
			Files:      status.DirtyFiles,
			LastCommit: lastCommit,
		}

		entry.Remote = &jsonRemote{
			Ahead:    status.Ahead,
			Behind:   status.Behind,
			Diverged: status.Ahead > 0 || status.Behind > 0,
			Tracking: !status.NoTracking,
		}

		results = append(results, entry)
	}

	output, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

func printRepoStatus(st repoState, displayPath string, pathWidth, branchWidth, workWidth, remoteWidth, ageWidth int) {
	// Truncate path if needed
	pathText := truncate(displayPath, pathWidth)

	// Check if repo exists
	if !st.exists {
		path := padRight(pathText, pathWidth)
		branch := padRight(colorize(colorGray, "—"), branchWidth)
		work := padRight(colorize(colorGray, "—"), workWidth)
		remote := padRight(colorize(colorGray, "—"), remoteWidth)
		age := padRight(colorize(colorGray, "—"), ageWidth)
		comment := colorize(colorGray, "not cloned")
		fmt.Printf("%s  %s  %s  %s  %s  %s\n", path, branch, work, remote, age, comment)
		return
	}

	if st.err != nil || st.status == nil {
		path := padRight(pathText, pathWidth)
		branch := padRight(colorize(colorGray, "?"), branchWidth)
		work := padRight(colorize(colorGray, "?"), workWidth)
		remote := padRight(colorize(colorGray, "?"), remoteWidth)
		age := padRight(colorize(colorGray, "?"), ageWidth)
		comment := colorize(colorGray, st.err.Error())
		fmt.Printf("%s  %s  %s  %s  %s  %s\n", path, branch, work, remote, age, comment)
		return
	}

	status := st.status

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

	// Format age
	var age string
	if status.LastCommitTime.IsZero() {
		age = padRight(colorize(colorGray, "?"), ageWidth)
	} else {
		ageText := formatRelativeTime(status.LastCommitTime)
		age = padRight(colorize(colorGray, ageText), ageWidth)
	}

	comment := colorize(colorGray, strings.Join(comments, ", "))

	fmt.Printf("%s  %s  %s  %s  %s  %s\n", path, branch, work, remote, age, comment)
}

// formatRelativeTime formats a time as a relative age string
func formatRelativeTime(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		return fmt.Sprintf("%dm", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		return fmt.Sprintf("%dh", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	case duration < 30*24*time.Hour:
		weeks := int(duration.Hours() / 24 / 7)
		return fmt.Sprintf("%dw", weeks)
	case duration < 365*24*time.Hour:
		months := int(duration.Hours() / 24 / 30)
		return fmt.Sprintf("%dmo", months)
	default:
		years := int(duration.Hours() / 24 / 365)
		return fmt.Sprintf("%dy", years)
	}
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
