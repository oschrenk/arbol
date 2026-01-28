# arbol

Managing git repositories using declarative TOML configuration and JSON output

## Features

- **JSON by default** - Pipe `arbol status` directly into `jq` for scripting
- **Declarative config** - Define all your repositories in a single TOML file
- **Multiple accounts** - Use different profiles for different machines (home, work, etc.)
- **Smart status** - See branch, dirty files, and sync state with remote at a glance
- **Shell completion** - Tab-complete repository paths in Fish shell

## Installation

### From Source

Requires Go 1.23+ and [go-task](https://taskfile.dev/).

```bash
git clone https://github.com/oschrenk/arbol.git
cd arbol
task install
```

This installs the binary to `$GOPATH/bin/arbol` and Fish completions to `~/.config/fish/completions/`.

### Manual Build

```bash
go build -o arbol ./cmd/arbol
```

## Quick Start

1. Create a starter config:

```bash
arbol init
```

2. Edit `~/.config/arbol/config.toml`:

```toml
[accounts.default]
default = true
root = "~/Projects"

repos.work.backend = [
  { url = "git@github.com:company/api.git" },
  { url = "git@github.com:company/worker.git" },
]

repos.personal = [
  { url = "git@github.com:me/dotfiles.git" },
]
```

3. Clone your repositories:

```bash
arbol sync
```

4. Check status:

```bash
arbol status
```

Output is JSON by default, ready for `jq`:

```bash
$ arbol status | jq '.[0]'
{
  "id": "work.backend.api",
  "path": "/home/user/Projects/work/backend/api",
  "branch": { "name": "main", "detached": false },
  "changes": { "dirty": false, "files": 0, "last_commit": "2025-01-15T10:30:00Z" },
  "remote": { "ahead": 0, "behind": 0, "diverged": false, "tracking": true }
}
```

Use `--plain` for a human-readable table.

## Commands

### `arbol sync [path]`

Clone missing repositories. Skips repos that already exist.

```bash
arbol sync                    # Sync all repos
arbol sync work.backend       # Sync repos under work.backend
arbol sync --fetch            # Also fetch updates for existing repos
```

**Flags:**
- `--fetch` - Run `git fetch --all --tags` on existing repos

### `arbol status [path]`

Show status of repositories. Outputs JSON by default for easy scripting and piping to tools like `jq`.

```bash
arbol status                  # JSON output (default)
arbol status personal         # Filter repos under personal
arbol status --plain          # Table output
arbol status | jq '.[] | select(.changes.dirty)'  # Filter dirty repos
```

**JSON schema:**

Each entry in the output array has the following structure. The `branch`, `changes`, and `remote` fields are omitted for repos that are not cloned or have errors.

```json
{
  "id": "work.backend.api",
  "path": "/home/user/Projects/work/backend/api",
  "branch": {
    "name": "main",
    "detached": false
  },
  "changes": {
    "dirty": true,
    "files": 3,
    "last_commit": "2025-01-15T10:30:00Z"
  },
  "remote": {
    "ahead": 2,
    "behind": 0,
    "diverged": true,
    "tracking": true
  }
}
```

**Flags:**
- `--plain` - Show table output instead of JSON
- `--no-color` - Disable colored output (only with `--plain`)
- `--no-headers` - Hide column headers (only with `--plain`)
- `--path-width N` - Width of PATH column, default: 30 (only with `--plain`)
- `--branch-width N` - Width of BRANCH column, default: 15 (only with `--plain`)

### `arbol init`

Create a starter configuration file at `~/.config/arbol/config.toml`.

### `arbol version`

Print version, commit hash, and build date.

### `arbol completion [shell]`

Generate shell completion scripts. Supports: bash, zsh, fish, powershell.

```bash
arbol completion fish > ~/.config/fish/completions/arbol.fish
```

## Global Flags

- `--account`, `-a` - Use a specific account instead of the default

## Configuration

Config location: `$XDG_CONFIG_HOME/arbol/config.toml` (defaults to `~/.config/arbol/config.toml`)

### Basic Structure

```toml
[accounts.<name>]
default = true              # Optional: mark as default account
root = "~/Projects"         # Required: base directory for repos

repos.<path> = [
  { url = "git@github.com:user/repo.git" },
  { url = "git@github.com:user/other.git", name = "custom-dir" },
]
```

### Path Mapping

Config paths map directly to filesystem directories:

| Config | Filesystem |
|--------|------------|
| `repos.work.backend` with `api` | `~/Projects/work/backend/api/` |
| `repos.personal` with `dotfiles` | `~/Projects/personal/dotfiles/` |

### Nested Paths

When a directory contains both repos and subdirectories, use `"/"`:

```toml
# Repos directly in ~/Projects/personal/
repos.personal."/" = [
  { url = "git@github.com:me/dotfiles.git" },
]

# Repos in ~/Projects/personal/golang/
repos.personal.golang = [
  { url = "git@github.com:me/arbol.git" },
]
```

### Multiple Accounts

Define different repo sets for different machines:

```toml
[accounts.home]
default = true
root = "~/Projects"

repos.personal = [
  { url = "git@github.com:me/dotfiles.git" },
]
repos.work.backend = [
  { url = "git@github.com:company/api.git" },
]

[accounts.work-laptop]
root = "~/Code"

# Only work repos on work laptop
repos.work.backend = [
  { url = "git@github.com:company/api.git" },
]
```

Use with: `arbol sync --account work-laptop`

See [EXAMPLES.md](EXAMPLES.md) for more `jq` recipes.

## License

MIT
