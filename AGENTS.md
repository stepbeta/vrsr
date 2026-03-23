# Copilot Instructions

You are an expert Go developer working on this CLI application built with Cobra.

## Project Overview

**vrsr** (Version Selector Runner) is a CLI tool for managing multiple versions of various command-line tools. It allows users to easily install, list, and switch between different versions of tools like `kubectl`, `helm`, `kind`, `talosctl`, `cilium`, and `hubble`. The tool downloads binaries from GitHub releases and manages them in a configurable storage path.

## Project Structure

- `cmd/` - Cobra commands (root.go, version.go, docs.go)
- `internal/cli/` - Command implementations organized by subcommand
- `internal/cli/tools/` - Tool-specific subcommands (kubectl, helm, kind, etc.)
- `internal/cli/common/` - Shared command utilities (install, list, use, etc.)
- `internal/utils/` - Utility functions (cache, binary management)
- `internal/github/` - GitHub API helpers

## Code Style

- Use `cobra.Command` for CLI commands
- Use `viper` for configuration management
- Use environment variable prefix `VRSR_` (e.g., `VRSR_BIN_PATH`)
- Follow standard Go conventions: `camelCase` for variables, `PascalCase` for exported functions/types
- Use meaningful error messages and wrap errors with `fmt.Errorf("context: %w", err)`
- Use the `command` pattern from `internal/cli/common/command.go` for consistent subcommand structure
- Always format and lint code after a modification.

## Testing

- Write unit tests with the `_test.go` suffix in the same package
- Use table-driven tests where appropriate

## Commands

- Run the CLI: `go run main.go`
- Run tests: `go test ./...`
- Build: `go build -o vrsr`
- Format code: `go fmt -w -s .`
- Lint code: `golangci-lint run`
- Add a new tool subcommand: Follow the pattern in `internal/cli/tools/`

## Constraints

- Never execute destructive commands (rm -rf, dd, mkfs, etc.) unless the user asks explicitly
- If the user asks for a destructive command, always ask for confirmation before executing
- Never modify or delete files outside the project directory without user consent
- Always validate potentially destructive operations with the user before proceeding
- Never run commands that could affect the host system (systemd operations, disk operations, etc.)
- When in doubt, ask the user to confirm before taking any action that modifies system state
- Do not interact with Git or GitHub repositories without explicit user instructions
- Always prioritize user safety and data integrity in all operations
