# Contributing to Agentary

Thank you for your interest in contributing to Agentary. This document covers development setup, pull request guidelines, code style, testing, and a brief architecture overview.

## Development setup

### Prerequisites

- **Go 1.21+** – [go.dev/dl](https://go.dev/dl/)
- **Git** – for version control and (optionally) testing git worktree behavior

### Clone and build

```bash
git clone https://github.com/ankittk/agentary.git
cd agentary
go build ./cmd/agentary
```

Use the built binary or run with `go run ./cmd/agentary`.

### Running the daemon locally

```bash
go run ./cmd/agentary start --foreground
```

Optional: copy `.env.example` to `.env` and set `AGENTARY_API_KEY`, `AGENTARY_HOME`, etc. Then:

```bash
go run ./cmd/agentary start --foreground --env-file .env
```

## Pull request guidelines

1. **Open an issue first** for non-trivial changes so we can align on approach.
2. **Keep PRs focused** – one feature or fix per PR when possible.
3. **Update tests** – new behavior should have tests; existing tests must pass.
4. **Update docs** – if you change CLI flags, API, or behavior, update README or `docs/` as needed.
5. **Follow code style** – run `gofmt` and the project’s linter (see below).

## Commit messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

- **feat:** new feature
- **fix:** bug fix
- **docs:** documentation only
- **test:** adding or updating tests
- **chore:** maintenance (deps, CI, etc.)
- **perf:** performance improvement
- **refactor:** code change that neither fixes a bug nor adds a feature

Example: `feat: add agent memory journal browser view`

CI runs commitlint on pull requests to enforce this format.

## Code style

- **Formatting**: Run `gofmt -s -w .` on all Go files. The CI runs format checks.
- **Linting**: We use [golangci-lint](https://golangci-lint.run/). Run `golangci-lint run ./...` before pushing.
- **Dependencies**: Keep dependencies minimal. New imports should be justified in the PR.
- **Naming**: Follow standard Go conventions; prefer short names in small scopes, clear names for exports.

## Testing

- **Unit tests**: `go test ./...`
- **Race detector**: `go test -race ./...` (CI runs this).
- **Coverage**: `go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out`

Add unit tests for new packages and for non-trivial logic. Existing tests live next to the code (e.g. `internal/store/store_test.go`, `internal/httpapi/server_test.go`).

## Architecture overview

- **`cmd/agentary`** – CLI entrypoint (Cobra). Subcommands: `start`, `team`, `task`, `workflow`, `agent`, `repo`, `network`, etc.
- **`internal/`** – All core logic. No public Go API here; use `pkg/` for shared types and client.
  - **`config`** – Home directory, env, and config resolution.
  - **`store`** – SQLite persistence (teams, tasks, workflows, messages, network allowlist). Migrations in `store/migrations/`.
  - **`httpapi`** – HTTP server and SSE hub; serves REST API and `/stream` for live events.
  - **`daemon`** – Scheduler, manager agent, and long-running process orchestration.
  - **`workflow`** – Workflow engine: stages, transitions, and builtin workflows.
  - **`manager`** – Manager (delegate) agent: inbox, task creation, assignment.
  - **`agent/runtime`** – Agent runtimes: stub and subprocess (JSON-lines over stdin/stdout).
  - **`sandbox`** – Execution sandbox (e.g. bubblewrap) for agent subprocesses.
  - **`git`** – Git worktree and branch helpers.
- **`pkg/models`** – Shared data types for external consumers.
- **`pkg/client`** – Go client for the Agentary HTTP API (for use by other tools).

Data and state live under `AGENTARY_HOME` (default `~/.agentary/`). See [docs/content/data-directory.md](docs/content/data-directory.md) for the target layout.

## Questions?

Open a [GitHub Discussion](https://github.com/ankittk/agentary/discussions) or an issue. For security-sensitive topics, see [SECURITY.md](SECURITY.md).
