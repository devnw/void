{
  description = "development flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    _1password-shell-plugins.url = "github:1Password/shell-plugins";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          system = system;
          overlays = [
            (self: super: {
              go = super.go_1_23;
              python = super.python3.withPackages (subpkgs: with subpkgs; [
                openapi-spec-validator
                detect-secrets
                requests
                python-dotenv
              ]);
            })
          ];
        };

        pkglist = with pkgs; [
          addlicense
          shfmt
          git
          pre-commit
          shellcheck
          automake
          act
          gcc

          python

          go
          delve
          golangci-lint
          goreleaser
          go-licenses
        ];
      in
      {
        packages = {
          default = pkgs.stdenv.mkDerivation {
            name = "cap";
            src = ./.;
            buildInputs = pkglist;
            buildPhase = ''
              make build
            '';
            installPhase = ''
              make install
            '';
          };
        };

        devShells = {
          default = pkgs.mkShell {
            buildInputs = pkglist;
          };
        };
      }
    );
}
