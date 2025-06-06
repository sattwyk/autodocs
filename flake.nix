{
  description = "dev environment for autodocs";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };
        pythonVersion = pkgs.python311;
        nodeVersion = pkgs.nodejs_22;
        goVersion = pkgs.go_1_23;

      in
      {
        devShells.default = pkgs.mkShell {
          name = "autodocs-dev-shell";
          packages = [
            # Python tools
            pythonVersion
            pkgs.uv
            pkgs.ruff

            # Node.js tools
            nodeVersion
            pkgs.pnpm

            # Go tools
            goVersion
            pkgs.gopls
            pkgs.delve

            # Other tools
            pkgs.git
            pkgs.docker-compose
            pkgs.opentofu
            pkgs.glibcLocales

            # Pre-commit and Nix formatting
            pkgs.pre-commit
            pkgs.nixfmt-rfc-style
          ];

          shellHook = ''
            # Set up environment variables
            export PATH="$PWD/node_modules/.bin:$PATH"
            export PYTHONIOENCODING=UTF-8
            export LANG=en_US.UTF-8
            export LC_ALL=en_US.UTF-8

            # Go environment setup
            export GOPATH="$PWD/.go"
            export GOCACHE="$PWD/.go/cache"
            export PATH="$GOPATH/bin:$PATH"
            mkdir -p "$GOPATH/bin"

            # Python virtual environment setup
            if [ ! -d ".venv" ]; then
              ${pythonVersion}/bin/python -m venv .venv
              echo "Created Python virtual environment"
            fi
            source .venv/bin/activate

            # Only run setup on first load or when forced
            if [ ! -f ".direnv/autodocs-loaded" ] || [ "$AUTODOCS_SHOW_INFO" = "1" ]; then
              # Install Python dependencies if needed
              if [ ! -f ".venv/lib/python3.11/site-packages/ruff/__init__.py" ] || \
                 [ "apps/agent/pyproject.toml" -nt ".venv/pyvenv.cfg" ]; then
                echo "Installing Python dependencies..."
                uv pip sync apps/agent/pyproject.toml --python .venv/bin/python || exit 1
              fi

              # Install Node.js dependencies if needed
              if [ ! -d "node_modules" ] || \
                 [ "pnpm-lock.yaml" -nt "node_modules/.pnpm/lock.yaml" ] 2>/dev/null; then
                echo "Installing Node.js dependencies..."
                pnpm install --frozen-lockfile || exit 1
              fi

              # Show environment info
              echo "Environment ready! Tools:"
              echo "Python $(python --version) | Node $(node --version) | pnpm $(pnpm --version) | Go $(go version)"

              # Mark as loaded
              mkdir -p .direnv
              touch .direnv/autodocs-loaded
            fi
          '';
        };
      }
    );
}
