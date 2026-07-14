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

1. Bump `version` in `flake.nix` to the new version; commit and push.
2. `task release -- v0.3.0` - lint, dirty check, tag, build artifacts, create GitHub release.
   Pushing the `v*` tag also triggers `.github/workflows/build.yml`, which builds every
   system and pushes the binaries to the Cachix cache.
3. Update [homebrew-made](https://github.com/oschrenk/homebrew-made) formula

## Fish Completion

Fish completion is generated via `arbol completion fish`. The script is embedded in the binary (`internal/commands/completion.go`) and generated during install and release.

## Nix

The repo is a flake. `nix run github:oschrenk/arbol`, `nix profile install github:oschrenk/arbol`, `nix build`, `nix develop` (dev shell). Use `task build` for the dev loop; Nix is for consuming/reproducible builds.

- **vendorHash**: after changing `go.mod`/`go.sum`, set `vendorHash = nixpkgs.lib.fakeHash`, run `nix build`, paste the `got:` hash. Bump `version` in `flake.nix` on release.
- **Cache**: CI (`.github/workflows/build.yml`) pushes prebuilt binaries to `oschrenk.cachix.org` on push/tag. Needs a `CACHIX_AUTH_TOKEN` repository secret ([Cachix personal token](https://app.cachix.org/personal-auth-tokens)). Pulling needs no token.
- **First-time trust**: the flake's `nixConfig` (cache) needs one-time approval, which `direnv` can't answer (it hangs). Accept it once without opening a shell: `nix develop --command true`, answer `y` to the prompts (persisted to `~/.local/share/nix/trusted-settings.json`), then `direnv reload`.
