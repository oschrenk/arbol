package config

import (
	"reflect"
	"testing"
)

func TestMatchesFilter(t *testing.T) {
	cases := []struct {
		name      string
		container string
		repo      string
		filter    string
		want      bool
	}{
		// Empty filter matches everything.
		{"empty filter", "timewax", "backend", "", true},

		// Exact repo filter matches only that repo, not siblings.
		{"exact repo", "timewax", "backend", "timewax.backend", true},
		{"sibling excluded", "timewax", "all-node-apps", "timewax.backend", false},
		{"sibling excluded 2", "timewax", "all-python2-apps", "timewax.backend", false},

		// Container filter matches all repos directly in it.
		{"container matches repo", "timewax", "backend", "timewax", true},
		{"container matches sibling", "timewax", "all-node-apps", "timewax", true},

		// Ancestor directory filter matches nested repos.
		{"ancestor matches nested", "timewax.golang", "tool", "timewax", true},
		{"nested container exact", "timewax.golang", "tool", "timewax.golang", true},

		// Non-matching filters.
		{"unrelated account", "personal", "dotfiles", "timewax", false},
		{"partial segment no match", "timewaxx", "backend", "timewax", false},
		{"deeper filter than repo", "timewax", "backend", "timewax.backend.extra", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := matchesFilter(c.container, c.repo, c.filter); got != c.want {
				t.Errorf("matchesFilter(%q, %q, %q) = %v, want %v",
					c.container, c.repo, c.filter, got, c.want)
			}
		})
	}
}

// AccountNames and RepoPaths back shell completion, which ranges over maps
// (random iteration order). Both must return sorted, deterministic output.

func TestAccountNamesSorted(t *testing.T) {
	cfg := &Config{
		Accounts: map[string]*Account{
			"zeta":  {},
			"alpha": {},
			"mid":   {},
		},
	}
	got := cfg.AccountNames()
	want := []string{"alpha", "mid", "zeta"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("AccountNames() = %v, want %v", got, want)
	}
}

func TestRepoPathsSorted(t *testing.T) {
	acct := &Account{
		Repos: map[string][]Repo{
			"work":     {{URL: "https://example.com/backend.git", Name: "backend"}, {URL: "https://example.com/api.git", Name: "api"}},
			"personal": {{URL: "https://example.com/dotfiles.git", Name: "dotfiles"}},
		},
	}
	got := acct.RepoPaths()
	want := []string{
		"personal",
		"personal.dotfiles",
		"work",
		"work.api",
		"work.backend",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RepoPaths() = %v, want %v", got, want)
	}
}
