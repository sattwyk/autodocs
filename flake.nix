{
  description = "dev environment for autodocs";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };

        # 1. Toolchain Definition
        # We define the core tools here. Nix ensures that every developer
        # gets the exact same version of the compilers, interpreters, and package managers.
        # This is the "Nix Layer" of our dependency management.
        python = pkgs.python311;
        node = pkgs.nodejs_22;
        go = pkgs.go_1_24;
        uv = pkgs.uv;

        # 2. Main Shell Definition
        # This is factored out for clarity. We assemble our tools and define the
        # shell hook that bridges Nix with the language-specific package managers.
        devShell = pkgs.mkShell {
          name = "autodocs-dev-shell";
          packages = [
            # Language Runtimes & Toolchains
            python
            node
            go

            # Language Package Managers
            # Nix provides the *managers*, and the managers handle the libraries.
            pkgs.pnpm
            uv

            # Go Language Server & Debugger
            pkgs.gopls
            pkgs.delve

            # Common System Tools
            pkgs.git
            pkgs.docker-compose
            pkgs.opentofu
            pkgs.glibcLocales
            pkgs.pre-commit
            pkgs.nixfmt-rfc-style
          ];

          shellHook = ''
            # Set a flag to prevent re-running the hook in sub-shells
            if [ -n "$IN_AUTODOCS_SHELL" ]; then
              return
            fi
            export IN_AUTODOCS_SHELL=1

            # Environment variables
            export PATH="$PWD/node_modules/.bin:$PATH"
            export PYTHONIOENCODING=UTF-8
            export LANG=en_US.UTF-8
            export LC_ALL=en_US.UTF-8
            export LOCALE_ARCHIVE="${pkgs.glibcLocales}/lib/locale/locale-archive"

            # Go environment setup
            export XDG_CACHE_HOME="$PWD/.cache"
            export GOCACHE="$XDG_CACHE_HOME/go-build"
            export GOPATH="$PWD/.go"
            export PATH="$GOPATH/bin:$PATH"

            # --- Hybrid Package Management Hooks ---
            # This is where we use the tools provided by Nix to manage
            # dependencies within the project directory. This is the "Language Layer".

            # Node.js: Use pnpm with its lock file.
            echo "Checking Node.js dependencies..."
            pnpm install --frozen-lockfile

            # Python: Use uv to create a venv and sync with pyproject.toml
            echo "Checking Python dependencies..."
            if [ ! -d ".venv" ]; then
              python -m venv .venv
            fi
            source .venv/bin/activate
            uv pip sync apps/agent/pyproject.toml --python .venv/bin/python

            echo -e "\nEnvironment ready!"
          '';
        };
      in
      {
        devShells.default = devShell;
      }
    );
}
