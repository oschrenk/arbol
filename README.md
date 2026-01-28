# arbol

Managing git repositories (across multiple machines) using declarative TOML configuration.

## Features

- **Declarative config** - Define all your repositories in a single TOML file
- **Tree structure** - Organize repos in paths that map to filesystem directories
- **Multiple accounts** - Use different profiles for different machines (home, work-laptop, etc.)
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

Show status of repositories in a table format.

```bash
arbol status                  # Status of all repos
arbol status personal         # Status of repos under personal
```

**Output columns:**
- **PATH** - Repository path in config
- **BRANCH** - Current branch (cyan if detached HEAD)
- **WORK** - Working tree status: green checkmark if clean, yellow dot with count if dirty
- **REMOTE** - Sync with remote: arrows show commits ahead/behind
- **COMMENTS** - Additional context (dirty file count, diverged, etc.)

**Flags:**
- `--no-color` - Disable colored output
- `--no-headers` - Hide column headers
- `--path-width N` - Width of PATH column (default: 30)
- `--branch-width N` - Width of BRANCH column (default: 15)

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

## Status Output Example

```
PATH                            BRANCH           WORK   REMOTE    COMMENTS
work.backend.api                main             ✔      ✔
work.backend.worker             feature/auth     ● 3    ↑2        3 dirty files, 2 unpushed commits
personal.dotfiles               main             ✔      ↓5        5 commits behind origin
personal.golang.arbol           main             ✔      ↓2 ↑1     diverged
external.archived               a]b5c2d          ✔      ✔         detached HEAD
```

## License

MIT
