package git

import (
	"errors"
	"strings"
	"testing"
)

func TestHostFromURL(t *testing.T) {
	cases := map[string]string{
		"git@github.com:oschrenk/arbol.git":     "github.com",
		"git@bitbucket.example.com:team/repo":   "bitbucket.example.com",
		"ssh://git@github.com:22/oschrenk/arbol": "github.com",
		"https://github.com/oschrenk/arbol.git": "github.com",
		"not-a-url":                             "not-a-url",
	}
	for url, want := range cases {
		if got := hostFromURL(url); got != want {
			t.Errorf("hostFromURL(%q) = %q, want %q", url, got, want)
		}
	}
}

func TestFriendlyCloneError(t *testing.T) {
	url := "git@github.com:oschrenk/arbol.git"

	network := errors.New("dial tcp 35.241.184.25:22: connect: bad file descriptor")
	got := friendlyCloneError(url, network)
	if !strings.Contains(got.Error(), "cannot reach") || !strings.Contains(got.Error(), "github.com") {
		t.Errorf("expected unreachable-host/host hint, got %q", got.Error())
	}
	if !errors.Is(got, network) {
		t.Error("expected wrapped error to preserve original via errors.Is")
	}

	authErr := errors.New("ssh: handshake failed: permission denied")
	got = friendlyCloneError(url, authErr)
	if !strings.Contains(got.Error(), "authentication") {
		t.Errorf("expected auth hint, got %q", got.Error())
	}

	other := errors.New("repository already exists")
	if friendlyCloneError(url, other) != other {
		t.Error("expected unrecognized errors to pass through unchanged")
	}
}
