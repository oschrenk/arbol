// Package git provides git operations for arbol.
//
// # Performance
//
// This package shells out to the git CLI for status operations rather than
// using go-git's pure Go implementation. This is significantly faster because:
//
//   - git status --porcelain: Uses git's optimized filesystem caching and stat
//     info rather than walking every file in Go
//   - git rev-list --count: Uses git's native graph algorithms to count commits
//     between refs, avoiding expensive ancestor traversal in Go
//
// go-git is still used for Clone (benefits from its SSH agent handling) and
// Exists (simple check).
package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// RepoStatus represents the status of a git repository
type RepoStatus struct {
	Branch         string
	IsDetached     bool // true if HEAD is detached (Branch will be short hash)
	IsDirty        bool
	DirtyFiles     int
	Behind         int       // commits current branch is behind origin
	Ahead          int       // commits current branch is ahead of origin (unpushed)
	NoTracking     bool      // true if no remote tracking branch
	LastCommitTime time.Time // time of the most recent commit
}

// Clone clones a git repository to the specified path
func Clone(url, path string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// Get SSH authentication
	auth := getSSHAuth()

	_, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:  url,
		Auth: auth,
	})
	return err
}

// getSSHAuth returns SSH authentication, trying agent first then key files
func getSSHAuth() transport.AuthMethod {
	// Try SSH agent first
	auth, err := ssh.NewSSHAgentAuth("git")
	if err == nil {
		return auth
	}

	// Fall back to default SSH key
	home, _ := os.UserHomeDir()
	keyPath := filepath.Join(home, ".ssh", "id_rsa")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		keyPath = filepath.Join(home, ".ssh", "id_ed25519")
	}

	keyAuth, err := ssh.NewPublicKeysFromFile("git", keyPath, "")
	if err == nil {
		return keyAuth
	}

	// No auth available (for public repos)
	return nil
}

// Status returns the status of a git repository
// Uses git CLI for speed
func Status(path string) (*RepoStatus, error) {
	result := &RepoStatus{}

	// Get current branch or commit hash if detached
	branch, err := gitCommand(path, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	branch = strings.TrimSpace(branch)

	if branch == "HEAD" {
		// Detached HEAD - get short hash
		hash, err := gitCommand(path, "rev-parse", "--short", "HEAD")
		if err != nil {
			return nil, err
		}
		result.Branch = strings.TrimSpace(hash)
		result.IsDetached = true
	} else {
		result.Branch = branch
		result.IsDetached = false
	}

	// Get dirty files count using git status --porcelain
	status, err := gitCommand(path, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	if status == "" {
		result.DirtyFiles = 0
	} else {
		result.DirtyFiles = len(strings.Split(strings.TrimSpace(status), "\n"))
	}
	result.IsDirty = result.DirtyFiles > 0

	// Check ahead/behind for current branch
	if !result.IsDetached {
		result.Ahead, result.Behind, result.NoTracking = getAheadBehind(path, result.Branch)
	}

	// Get last commit time
	result.LastCommitTime = getLastCommitTime(path)

	return result, nil
}

// getLastCommitTime returns the time of the most recent commit
func getLastCommitTime(repoPath string) time.Time {
	// Use %ct for committer date as Unix timestamp (faster to parse)
	output, err := gitCommand(repoPath, "log", "-1", "--format=%ct")
	if err != nil {
		return time.Time{}
	}
	timestamp, err := strconv.ParseInt(strings.TrimSpace(output), 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(timestamp, 0)
}

// gitCommand runs a git command and returns stdout
func gitCommand(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// getAheadBehind returns how many commits the current branch is ahead/behind its remote tracking branch
// Uses git rev-list --count for efficiency
func getAheadBehind(repoPath, branchName string) (ahead, behind int, noTracking bool) {
	remote := "origin/" + branchName

	// Check if remote tracking branch exists
	cmd := exec.Command("git", "rev-parse", "--verify", remote)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return 0, 0, true
	}

	// Count commits ahead (local commits not in remote)
	ahead = revListCount(repoPath, remote+"..HEAD")

	// Count commits behind (remote commits not in local)
	behind = revListCount(repoPath, "HEAD.."+remote)

	return ahead, behind, false
}

// revListCount runs git rev-list --count and returns the count
func revListCount(repoPath, revRange string) int {
	cmd := exec.Command("git", "rev-list", "--count", revRange)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	count, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0
	}
	return count
}

// Exists checks if a path is a git repository
func Exists(path string) bool {
	_, err := git.PlainOpen(path)
	return err == nil
}

// Fetch fetches all remotes and tags for a repository
// Uses --progress to show output even when not a tty
func Fetch(path string) error {
	cmd := exec.Command("git", "fetch", "--all", "--tags", "--progress")
	cmd.Dir = path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
