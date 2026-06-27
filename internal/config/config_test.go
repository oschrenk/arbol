package config

import "testing"

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
