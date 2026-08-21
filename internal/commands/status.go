package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/oschrenk/arbol/internal/config"
	"github.com/oschrenk/arbol/internal/git"
	"github.com/spf13/cobra"
)

var (
	noColor     bool
	noHeaders   bool
	pathWidth   int
	branchWidth int
	plainOutput bool
	stateFilter []string
)

// Repository states accepted by --state. A repo matches the filter if it is in
// any of the requested states (OR).
const (
	// stateAttention is a superstate expanding to every state below: anything
	// that is not clean and in sync with its upstream.
	stateAttention = "attention"

	stateDirty      = "dirty"
	stateAhead      = "ahead"
	stateBehind     = "behind"
	stateDiverged   = "diverged"
	stateDetached   = "detached"
	stateMissing    = "missing"
	stateNoUpstream = "no-upstream"
)

var validStates = []string{
	stateDirty,
	stateAhead,
	stateBehind,
	stateDiverged,
	stateDetached,
	stateMissing,
	stateNoUpstream,
}

// stateKeys is what --state accepts, i.e. the concrete states plus the
// attention superstate.
var stateKeys = append([]string{stateAttention}, validStates...)

// repoState pairs a repo with its resolved git status, so the state filter and
// the renderers work from a single snapshot instead of running git twice.
type repoState struct {
	displayPath string
	fullPath    string
	exists      bool
	status      *git.RepoStatus
	err         error
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

--state narrows the output to repos in any of the given states:

  attention    any of the states below, i.e. not clean and in sync
  dirty        uncommitted changes in the working tree
  ahead        unpushed commits
  behind       commits behind the upstream branch
  diverged     both ahead and behind
  detached     detached HEAD
  missing      not cloned yet
  no-upstream  branch has no upstream to compare against

Repos whose status cannot be read are always shown, since they cannot be
classified.

Examples:
  arbol status                       # status of all repos
  arbol status work.backend          # status of repos under work.backend
  arbol status --account spare       # use specific account
  arbol status --state dirty         # only repos with uncommitted changes
  arbol status --state dirty,ahead   # dirty or unpushed
  arbol status --state attention     # anything not clean and in sync`,
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

		keys, err := parseStateFilter(stateFilter)
		if err != nil {
			return err
		}

		states := collectStates(repos)
		if len(keys) > 0 {
			states = filterStates(states, keys)
		}

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
	statusCmd.Flags().StringSliceVar(&stateFilter, "state", nil,
		"Only show repos in any of these states: "+strings.Join(stateKeys, ", "))
	_ = statusCmd.RegisterFlagCompletionFunc("state",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return stateKeys, cobra.ShellCompDirectiveNoFileComp
		})
	rootCmd.AddCommand(statusCmd)
}

// parseStateFilter validates and normalizes the --state values.
func parseStateFilter(values []string) ([]string, error) {
	var keys []string
	seen := make(map[string]bool)

	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if key == "" {
			continue
		}
		if !slices.Contains(stateKeys, key) {
			return nil, fmt.Errorf("unknown state '%s' (valid states: %s)", value, strings.Join(stateKeys, ", "))
		}

		// attention is a superstate: expand it to every concrete state.
		expanded := []string{key}
		if key == stateAttention {
			expanded = validStates
		}
		for _, k := range expanded {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}

	return keys, nil
}

// collectStates resolves the git status of each repo once.
func collectStates(repos []config.RepoWithPath) []repoState {
	states := make([]repoState, 0, len(repos))

	for _, repo := range repos {
		state := repoState{
			displayPath: repo.Path + "." + repo.Name,
			fullPath:    repo.FullPath,
			exists:      git.Exists(repo.FullPath),
		}
		if state.exists {
			state.status, state.err = git.Status(repo.FullPath)
		}
		states = append(states, state)
	}

	return states
}

func filterStates(states []repoState, keys []string) []repoState {
	filtered := make([]repoState, 0, len(states))
	for _, state := range states {
		if state.matches(keys) {
			filtered = append(filtered, state)
		}
	}
	return filtered
}

// matches reports whether the repo is in any of the requested states.
func (s repoState) matches(keys []string) bool {
	// A repo whose status could not be read cannot be classified. Show it
	// rather than silently dropping a repo that may well need attention.
	if s.exists && s.err != nil {
		return true
	}

	for _, key := range keys {
		if key == stateMissing {
			if !s.exists {
				return true
			}
			continue
		}
		if s.status == nil {
			continue
		}

		switch key {
		case stateDirty:
			if s.status.IsDirty {
				return true
			}
		case stateAhead:
			if s.status.Ahead > 0 {
				return true
			}
		case stateBehind:
			if s.status.Behind > 0 {
				return true
			}
		case stateDiverged:
			if s.status.Ahead > 0 && s.status.Behind > 0 {
				return true
			}
		case stateDetached:
			if s.status.IsDetached {
				return true
			}
		case stateNoUpstream:
			if s.status.NoTracking {
				return true
			}
		}
	}

	return false
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

	for _, state := range states {
		printRepoStatus(state, pathWidth, branchWidth, workWidth, remoteWidth, ageWidth)
	}
	return nil
}

func printJSONStatus(states []repoState) error {
	results := []jsonRepo{}

	for _, state := range states {
		entry := jsonRepo{
			ID:   state.displayPath,
			Path: state.fullPath,
		}

		if !state.exists || state.err != nil {
			results = append(results, entry)
			continue
		}
		status := state.status

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

func printRepoStatus(state repoState, pathWidth, branchWidth, workWidth, remoteWidth, ageWidth int) {
	// Truncate path if needed
	pathText := truncate(state.displayPath, pathWidth)

	// Check if repo exists
	if !state.exists {
		path := padRight(pathText, pathWidth)
		branch := padRight(colorize(colorGray, "—"), branchWidth)
		work := padRight(colorize(colorGray, "—"), workWidth)
		remote := padRight(colorize(colorGray, "—"), remoteWidth)
		age := padRight(colorize(colorGray, "—"), ageWidth)
		comment := colorize(colorGray, "not cloned")
		fmt.Printf("%s  %s  %s  %s  %s  %s\n", path, branch, work, remote, age, comment)
		return
	}

	status, err := state.status, state.err
	if err != nil {
		path := padRight(pathText, pathWidth)
		branch := padRight(colorize(colorGray, "?"), branchWidth)
		work := padRight(colorize(colorGray, "?"), workWidth)
		remote := padRight(colorize(colorGray, "?"), remoteWidth)
		age := padRight(colorize(colorGray, "?"), ageWidth)
		comment := colorize(colorGray, err.Error())
		fmt.Printf("%s  %s  %s  %s  %s  %s\n", path, branch, work, remote, age, comment)
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
