{
  description = "development flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    _1password-shell-plugins.url = "github:1Password/shell-plugins";
    gomod2nix = {
      url = "github:tweag/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      gomod2nix,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          config = {
            allowUnfree = true;
          };
          system = system;
          overlays = [
            gomod2nix.overlays.default
            (self: super: {
              go = super.go_1_23;
              python = super.python3.withPackages (
                subpkgs: with subpkgs; [
                  openapi-spec-validator
                  detect-secrets
                  requests
                  python-dotenv
                ]
              );
            })
          ];
        };

        pkglist = with pkgs; [
          toybox
          gnumake
          openssh
          git
          bash
          sudo
          curl
          wget
          which
          gawk

          _1password
          gibberish-detector
          addlicense
          shfmt
          git
          pre-commit
          shellcheck
          act
          gcc
          ruby

          nodejs
          nodePackages.npm
          nodePackages.prettier
          nodePackages.sql-formatter

          python

          go
          gopls
          gotools
          go-tools
          gomod2nix.packages.${system}.default
          sqlite-interactive

          delve
          golangci-lint
          goreleaser
          go-licenses

          # postgresql

          sqlfluff
          squawk
          postgresql
          pgformatter
          go-migrate
        ];
      in
      {
        packages.default = pkgs.buildGoApplication {
          pname = "void";
          version = "0.0.11";

          pwd = ./.;
          src = ./.;
          modules = ./gomod2nix.toml;
          buildInputs = pkglist;

          buildPhase = ''
            make build
          '';
          installPhase = ''
            make install
          '';
        };

        devShells.default = pkgs.mkShell {
          buildInputs = pkglist;
          shellHook = ''
            echo "Post-load script executed!"
            #export MY_VAR="hello"
            npm i prettier-plugin-sql
          '';
        };
      }
    );
}
