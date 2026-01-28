# Claude Context for Arbol

Arbol is a CLI tool for managing git repositories across multiple machines using declarative TOML configuration.

## Project Structure

```
arbol/
├── cmd/arbol/main.go           # Entry point
├── internal/
│   ├── commands/               # Cobra commands
│   │   ├── root.go             # Root command, --account flag, config loading
│   │   ├── sync.go             # Clone missing repos, --fetch flag
│   │   ├── status.go           # Show repo status with colors
│   │   ├── init.go             # Create starter config
│   │   ├── version.go          # Version info (ldflags)
│   │   ├── completion.go       # Shell completion (custom Fish script)
│   │   └── complete.go         # Hidden completion helper commands
│   ├── config/
│   │   └── config.go           # TOML parsing, account/repo structs, validation
│   └── git/
│       └── git.go              # Git operations (clone via go-git, status via CLI)
├── taskfile.yml                # Build tasks (task build, test, install, etc.)
├── SPEC.md                     # Full specification
├── DEVELOPMENT.md              # Development setup
└── README.md                   # User documentation
```

## Key Design Decisions

### Config Format

Uses dotted TOML keys for paths: `repos.work.backend = [...]`. When a path has both repos and subpaths, use `"/"` suffix:

```toml
repos.personal."/" = [{ url = "..." }]      # repos at personal/
repos.personal.golang = [{ url = "..." }]   # repos at personal/golang/
```

### Performance

Git status operations shell out to `git` CLI instead of using go-git's pure Go implementation:
- `git status --porcelain` for dirty files
- `git rev-list --count` for ahead/behind counts

This is significantly faster than go-git's commit graph traversal.

### Shell Completion

Fish completion uses hidden commands (`__complete-path`, `__complete-account`) rather than Cobra's built-in completion, for better control over suggestions.

## Common Tasks

### Building

```bash
task build          # Build to bin/arbol
task install        # Install to $GOPATH/bin + fish completion
task test           # Run tests
```

### Testing Commands

```bash
./bin/arbol status                    # All repos
./bin/arbol status work.backend       # Filtered
./bin/arbol sync --fetch              # Clone missing, fetch existing
./bin/arbol init                      # Create starter config
```

## Code Conventions

- Commands use Cobra with `RunE` for error handling
- Config validation happens at load time in `config.Load()`
- Colors use ANSI codes with terminal detection (`isTerminal()`)
- Column alignment accounts for ANSI escape sequences (`padRight()`)

## Config Location

`$XDG_CONFIG_HOME/arbol/config.toml` (defaults to `~/.config/arbol/config.toml`)
