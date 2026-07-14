{
  description = "Arbol - manage git repositories across machines via declarative TOML";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  # Offer prebuilt binaries from the Cachix cache so `nix profile install`
  # downloads instead of compiling. Consumers are prompted to trust these.
  nixConfig = {
    extra-substituters = [ "https://oschrenk.cachix.org" ];
    extra-trusted-public-keys = [
      "oschrenk.cachix.org-1:3JOMfkq2vFiLw4UsCVwzu8kWFBkuS/3DD5AojcO9pks="
    ];
  };

  outputs =
    { self, nixpkgs }:
    let
      # Single source of truth for the version: ./VERSION holds a bare semver
      # (e.g. 0.2.2); the "v" prefix is added here and in the taskfile release flow.
      version = "v${nixpkgs.lib.fileContents ./VERSION}";

      systems = [
        "aarch64-darwin"
        "aarch64-linux"
        "x86_64-linux"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      packages = forAllSystems (pkgs: rec {
        arbol = pkgs.buildGoModule {
          pname = "arbol";
          inherit version;
          src = self;

          # Regenerate after changing go.mod/go.sum: set to lib.fakeHash,
          # run `nix build`, then paste the expected hash from the error.
          vendorHash = "sha256-dICoy7X//t/wsqd4CD9TtRyJ5GpdsthhfmtIx18/U0c=";

          subPackages = [ "cmd/arbol" ];

          ldflags =
            let
              p = "github.com/oschrenk/arbol/internal/commands";
            in
            [
              "-s"
              "-w"
              "-X ${p}.Version=${version}"
              "-X ${p}.Commit=${self.shortRev or self.dirtyShortRev or "unknown"}"
              "-X ${p}.BuildDate=${self.lastModifiedDate}"
            ];

          # Generate + install the fish completion the taskfile ships.
          nativeBuildInputs = [ pkgs.installShellFiles ];
          postInstall = ''
            installShellCompletion --cmd arbol \
              --bash <($out/bin/arbol completion bash) \
              --zsh <($out/bin/arbol completion zsh) \
              --fish <($out/bin/arbol completion fish)
          '';

          meta = {
            description = "Manage git repositories across machines via declarative TOML";
            homepage = "https://github.com/oschrenk/arbol";
            mainProgram = "arbol";
          };
        };
        default = arbol;
      });

      apps = forAllSystems (pkgs: rec {
        arbol = {
          type = "app";
          program = "${self.packages.${pkgs.stdenv.hostPlatform.system}.arbol}/bin/arbol";
        };
        default = arbol;
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go # go, language
            golangci-lint # go, linter runner
            gopls # go, lsp
          ];
        };
      });
    };
}
