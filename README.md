# autodocs

automatic documentation builder

A multi-language monorepo with TypeScript/JavaScript, Python, and Go applications.

## Development Setup

This project uses [pre-commit.com](https://pre-commit.com/) for managing git hooks across multiple languages.

### Prerequisites

- Nix with flakes enabled (provides pre-commit, nixfmt, and other tools)
- Node.js 22+ with pnpm
- Go 1.24+
- uv (Python package manager)

### Installation

1. Install dependencies:

```bash
# Install Python dependencies
uv sync

# Install Node.js dependencies
pnpm install

# Install pre-commit hooks (pre-commit is provided by Nix)
pre-commit install
pre-commit install --hook-type pre-push
```

### Pre-commit Hooks

The following hooks run automatically on commit:

- **General**: Trailing whitespace, end-of-file fixing, merge conflict detection
- **Nix**: Nix file formatting with nixfmt
- **TypeScript/JavaScript**: Biome formatting, TypeScript type checking
- **Python**: Ruff linting and formatting (apps/agent)
- **Go**: Formatting, vetting, and mod tidying (apps/crawler)

### Pre-push Hooks

The following hooks run on push:

- **Node.js**: All tests via `pnpm run test`
- **Go**: Tests via `go test ./...` in apps/crawler

### Manual Commands

```bash
# Run all hooks on all files
pre-commit run --all-files

# Run specific hook
pre-commit run go-fmt

# Run pre-push hooks manually
pre-commit run --hook-stage pre-push

# Update hook versions
pre-commit autoupdate
```
