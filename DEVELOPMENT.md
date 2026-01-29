# Development

## Requirements

- [Go 1.23+](https://go.dev/) - `brew install go`
- [go-task](https://taskfile.dev/) - `brew install go-task`

## Commands

- `task build` - Build binary to `bin/arbol`
- `task run` - Run the application
- `task test` - Run tests
- `task tidy` - Tidy dependencies
- `task lint` - Run linters (go vet, go fmt)
- `task clean` - Remove build artifacts
- `task install` - Install to `$GOPATH/bin/` and fish completion
- `task uninstall` - Remove from `$GOPATH/bin/` and fish completion
- `task artifacts` - Build release artifacts to `.release/`
- `task sha` - Generate SHA256 checksums
- `task release -- v0.2.0` - Lint, dirty check, tag, build, and create GitHub release
- `task updates` - Check for dependency updates

## Version Management

Version is derived from git tags via `git describe --tags`. The version is embedded in the binary at build time via LDFLAGS.

- Tagged commit: `v0.1.0`
- After commits: `v0.1.0-3-gabc1234`
- Dirty working tree: `v0.1.0-dirty`
- No tags: `dev`

## Release Process

1. `task release -- v0.2.0` - lint, dirty check, tag, build artifacts, create GitHub release
2. Update [homebrew-made](https://github.com/oschrenk/homebrew-made) formula

## Fish Completion

Fish completion is generated via `arbol completion fish`. The script is embedded in the binary (`internal/commands/completion.go`) and generated during install and release.
