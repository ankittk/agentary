---
title: "CLI reference"
permalink: "/docs/cli-reference/"
description: "Commands and flags"
weight: 4
---

The CLI is the `agentary` binary. All commands respect `--home` (or `AGENTARY_HOME`) for the data directory.

## Global flag

| Flag | Description |
|------|-------------|
| `--home` | Override Agentary home directory (default: `~/.agentary`). |

## Commands

### Lifecycle

| Command | Description |
|---------|-------------|
| `agentary start` | Start the server and scheduler. |
| `agentary start --foreground` | Run in the foreground (no daemonize). |
| `agentary stop` | Stop the daemon. |
| `agentary status` | Show whether Agentary is running and the UI URL. |
| `agentary doctor` | Run health checks (home, DB, etc.). |
| `agentary nuke` | Remove home directory and all data (destructive). |

### Teams and agents

| Command | Description |
|---------|-------------|
| `agentary team add --name <name>` | Create a team. |
| `agentary team list` | List teams. |
| `agentary team remove --name <name>` | Remove a team. |
| `agentary agent add <team> <name> [--role engineer\|manager]` | Create an agent. |

### Repos and workflows

| Command | Description |
|---------|-------------|
| `agentary repo add --team <team> --name <name> --source <path>` | Add a repo. |
| `agentary repo list --team <team>` | List repos. |
| `agentary workflow show --team <team>` | Show workflow for team. |

### Network, identity, and API key

| Command | Description |
|---------|-------------|
| `agentary network allow --domain <domain>` | Allow a domain in the network allowlist. |
| `agentary network disallow --domain <domain>` | Disallow a domain. |
| `agentary network show` | Show current allowlist. |
| `agentary identity detect [--repo <path>]` | Detect git user name/email and save to members. |
| `agentary apikey generate` | Generate a random API key and print usage. Use `--env .env` to append to a file. |

### Start flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | 3548 | Port for the web UI. |
| `--foreground` | false | Run in foreground. |
| `--dev` | false | Enable dev mode (CORS for Vite). |
| `--otel` | true | Enable OpenTelemetry metrics. |
| `--runtime` | stub | Runtime: `stub`, `subprocess`, or `grpc`. |
| `--db-driver` | sqlite | Store driver: `sqlite` or `postgres`. |
| `--db-url` | "" | PostgreSQL connection string (or `DATABASE_URL`). |
| `--env-file` | "" | Load env vars from file. |
| `--pprof` | "" | Enable pprof on address (e.g. `127.0.0.1:6060`). |
| `--interval` | 1.0 | Scheduler poll interval (seconds). |
| `--max-concurrent` | 32 | Max concurrent agent turns. |
| `--subprocess-cmd` | "" | Command for subprocess runtime. |
| `--subprocess-args` | [] | Args for subprocess runtime. |
| `--grpc-addr` | "" | gRPC server address for `runtime=grpc`. |

Run `agentary --help` and `agentary <command> --help` for the full list.
