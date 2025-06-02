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
        nodeVersion = pkgs.nodejs_22; # Choose a specific Node.js version

      in
      {
        devShells.default = pkgs.mkShell {
          name = "autodocs-dev-shell";
          packages = [
            # Python tools
            pythonVersion
            pkgs.uv # Fast Python package installer and resolver
            pkgs.ruff # Python linter/formatter

            # Node.js tools
            nodeVersion
            pkgs.pnpm # PNPM package manager
            pkgs.nodePackages.typescript # Global TypeScript for LSP
            pkgs.biome # Biome for JS/TS linting/formatting
            pkgs.turbo # Turborepo CLI

            # Other tools
            pkgs.pre-commit # For pre-commit hooks
            pkgs.git
            pkgs.docker-compose # For Docker Compose
            pkgs.opentofu # Open-source Terraform alternative
            pkgs.openssl # Often a dependency for various tools
            pkgs.pkg-config # For building some native extensions
            pkgs.glibcLocales # For locale settings that some tools might need
          ];

          shellHook = ''
            export PATH="$PWD/node_modules/.bin:$PATH"
            export PYTHONIOENCODING=UTF-8
            export LANG=en_US.UTF-8
            export LC_ALL=en_US.UTF-8

            # Python virtual environment setup with uv
            if [ ! -d ".venv" ]; then
              ${pythonVersion}/bin/python -m venv .venv
              echo "Python virtual environment created with ${pythonVersion}."
            fi
            source .venv/bin/activate

            # uv is now available natively from Nix
            echo "uv is available: $(uv --version)"

            # Install Python dependencies from workspace
            echo "Syncing Python workspace dependencies with uv..."
            uv pip sync apps/agent/pyproject.toml --python .venv/bin/python
            uv pip sync apps/bots/whatsapp/pyproject.toml --python .venv/bin/python
            uv pip sync apps/bots/telegram/pyproject.toml --python .venv/bin/python
            # Consider a workspace-level sync if your uv version supports it well for all projects


            # Install Node.js dependencies
            echo "Installing Node.js dependencies with pnpm..."
            pnpm install --frozen-lockfile

            echo "Development environment ready!"
            echo "--------------------------------------------------"
            echo "Key tools available:"
            echo "- Python: $(${pythonVersion}/bin/python --version)"
            echo "- uv: $(uv --version)"
            echo "- Ruff: $(ruff --version)"
            echo "- Node.js: $(node --version)"
            echo "- pnpm: $(pnpm --version)"
            echo "- TypeScript: $(tsc --version)"
            echo "- Biome: $(biome --version)"
            echo "- Turborepo: $(turbo --version)"
            echo "- Docker Compose: $(docker-compose --version)"
            echo "- OpenTofu: $(tofu --version)"
            echo "- pre-commit: $(pre-commit --version)"
            echo "--------------------------------------------------"
            echo "Run 'exit' to leave this Nix shell."
          '';

          # Environment variables for VSCode or other tools
          # These help tools find the correct interpreters and configurations
          RUST_SRC_PATH = "${pkgs.rustPlatform.rustLibSrc}";

        };
      }
    );
}