package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Repo represents a git repository configuration
type Repo struct {
	URL  string `toml:"url"`
	Name string `toml:"name,omitempty"`
}

// Account represents a machine profile with repos
type Account struct {
	Default bool
	Root    string
	Repos   map[string][]Repo // path -> repos (path has "/" stripped)
}

// Config represents the full configuration file
type Config struct {
	Accounts map[string]*Account
}

// RepoWithPath represents a repo with its full path information
type RepoWithPath struct {
	Repo     Repo
	Path     string // e.g., "work.backend"
	FullPath string // e.g., "/Users/user/Projects/work/backend/api"
	Name     string // derived name from URL or explicit name
}

// ConfigPath returns the path to the config file
func ConfigPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "arbol", "config.toml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "arbol", "config.toml")
}

// Load loads the configuration from the default config path
func Load() (*Config, error) {
	configPath := ConfigPath()
	return LoadFromPath(configPath)
}

// LoadFromPath loads the configuration from a specific path
func LoadFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found at %s\nRun 'arbol init' to create a starter config", path)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse into raw structure first
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	config := &Config{
		Accounts: make(map[string]*Account),
	}

	// Extract accounts
	accountsRaw, ok := raw["accounts"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid config: missing 'accounts' section")
	}

	for accountName, accountData := range accountsRaw {
		accountMap, ok := accountData.(map[string]any)
		if !ok {
			continue
		}

		account := &Account{
			Repos: make(map[string][]Repo),
		}

		// Parse default and root
		if def, ok := accountMap["default"].(bool); ok {
			account.Default = def
		}
		if root, ok := accountMap["root"].(string); ok {
			account.Root = root
		}

		// Parse repos - traverse the nested structure
		if reposRaw, ok := accountMap["repos"].(map[string]any); ok {
			parseReposRecursive(reposRaw, "", account.Repos)
		}

		config.Accounts[accountName] = account
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// parseReposRecursive traverses the nested repos structure
func parseReposRecursive(data map[string]any, prefix string, repos map[string][]Repo) {
	for key, value := range data {
		var currentPath string
		if prefix == "" {
			currentPath = key
		} else {
			currentPath = prefix + "." + key
		}

		// Check if this is a repo array or nested path
		switch v := value.(type) {
		case []any:
			// This is an array of repos
			path := currentPath
			// Strip trailing "/" from path (it's just a marker)
			if key == "/" {
				path = prefix
			}

			var repoList []Repo
			for _, item := range v {
				if repoMap, ok := item.(map[string]any); ok {
					repo := Repo{}
					if url, ok := repoMap["url"].(string); ok {
						repo.URL = url
					}
					if name, ok := repoMap["name"].(string); ok {
						repo.Name = name
					}
					repoList = append(repoList, repo)
				}
			}
			repos[path] = repoList

		case map[string]any:
			// This is a nested path, recurse
			parseReposRecursive(v, currentPath, repos)
		}
	}
}

// Validate checks the config for errors
func (c *Config) Validate() error {
	for accountName, account := range c.Accounts {
		if err := account.Validate(accountName); err != nil {
			return err
		}
	}
	return nil
}

// Validate checks an account for path conflicts
func (a *Account) Validate(accountName string) error {
	// Build a set of all path segments that exist as subpaths
	subpaths := make(map[string]map[string]bool) // parent path -> set of child segments

	for path := range a.Repos {
		parts := strings.Split(path, ".")
		for i := 0; i < len(parts)-1; i++ {
			parentPath := strings.Join(parts[:i+1], ".")
			childSegment := parts[i+1]
			if subpaths[parentPath] == nil {
				subpaths[parentPath] = make(map[string]bool)
			}
			subpaths[parentPath][childSegment] = true
		}
	}

	// Check for conflicts: repo names that match sibling path segments
	for path, repos := range a.Repos {
		siblings := subpaths[path]
		if siblings == nil {
			continue
		}

		for _, repo := range repos {
			repoName := RepoName(repo.URL)
			if repo.Name != "" {
				repoName = repo.Name
			}

			if siblings[repoName] {
				conflictPath := path + "." + repoName
				return fmt.Errorf("config conflict in account %q\n  repos.%s.\"/\" contains repo %q\n  repos.%s also exists\n  Both would use path: %s/%s/",
					accountName, path, repoName, conflictPath,
					ExpandPath(a.Root), strings.ReplaceAll(conflictPath, ".", "/"))
			}
		}
	}

	return nil
}

// DefaultAccount returns the default account and its name
func (c *Config) DefaultAccount() (*Account, string, error) {
	for name, account := range c.Accounts {
		if account.Default {
			return account, name, nil
		}
	}
	// If no default, return first account if only one exists
	if len(c.Accounts) == 1 {
		for name, account := range c.Accounts {
			return account, name, nil
		}
	}
	return nil, "", fmt.Errorf("no default account configured and multiple accounts exist")
}

// GetAccount returns a specific account by name
func (c *Config) GetAccount(name string) (*Account, error) {
	account, ok := c.Accounts[name]
	if !ok {
		return nil, fmt.Errorf("account '%s' not found", name)
	}
	return account, nil
}

// GetRepos returns all repos for an account, optionally filtered by path prefix
func (a *Account) GetRepos(pathFilter string) []RepoWithPath {
	var result []RepoWithPath
	rootPath := ExpandPath(a.Root)

	for path, repos := range a.Repos {
		// Check if path matches filter
		if pathFilter != "" {
			if !strings.HasPrefix(path, pathFilter) && !strings.HasPrefix(pathFilter, path) {
				continue
			}
		}

		// Convert dotted path to directory path
		dirPath := strings.ReplaceAll(path, ".", string(filepath.Separator))

		for _, repo := range repos {
			name := RepoName(repo.URL)
			if repo.Name != "" {
				name = repo.Name
			}

			// Check if filtering for a specific repo
			if pathFilter != "" && strings.Contains(pathFilter, ".") {
				fullRepoPath := path + "." + name
				if !strings.HasPrefix(fullRepoPath, pathFilter) && fullRepoPath != pathFilter {
					// Skip if the filter is more specific and doesn't match
					parts := strings.Split(pathFilter, ".")
					filterPath := strings.Join(parts[:len(parts)-1], ".")
					filterName := parts[len(parts)-1]
					if filterPath != path || filterName != name {
						if len(parts) > len(strings.Split(path, "."))+1 {
							continue
						}
					}
				}
			}

			fullPath := filepath.Join(rootPath, dirPath, name)
			result = append(result, RepoWithPath{
				Repo:     repo,
				Path:     path,
				FullPath: fullPath,
				Name:     name,
			})
		}
	}

	return result
}

// AccountNames returns a list of all account names
func (c *Config) AccountNames() []string {
	var names []string
	for name := range c.Accounts {
		names = append(names, name)
	}
	return names
}

// RepoPaths returns all unique repo paths for an account
func (a *Account) RepoPaths() []string {
	var paths []string
	seen := make(map[string]bool)

	for path, repos := range a.Repos {
		if !seen[path] {
			paths = append(paths, path)
			seen[path] = true
		}
		// Also add individual repo paths
		for _, repo := range repos {
			name := RepoName(repo.URL)
			if repo.Name != "" {
				name = repo.Name
			}
			fullPath := path + "." + name
			if !seen[fullPath] {
				paths = append(paths, fullPath)
				seen[fullPath] = true
			}
		}
	}

	return paths
}

// ExpandPath expands ~ to home directory
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// RepoName extracts the repository name from a git URL
func RepoName(url string) string {
	// Handle both git@github.com:user/repo.git and https://github.com/user/repo.git
	base := filepath.Base(url)
	return strings.TrimSuffix(base, ".git")
}
