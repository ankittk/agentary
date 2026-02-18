---
title: "Configuration"
permalink: "/docs/configuration/"
description: "Flags, env vars, config files"
weight: 8
---

This page summarizes how to configure Agentary: CLI flags, environment variables, and config files.

## CLI flags (start command)

| Flag | Default | Description |
|------|---------|-------------|
| `--home` | `~/.agentary` | Data directory. |
| `--port` | 3548 | HTTP port for API and web UI. |
| `--foreground` | false | Run in foreground (no daemon). |
| `--dev` | false | Enable dev mode (e.g. CORS for Vite). |
| `--otel` | true | Enable OpenTelemetry metrics. |
| `--runtime` | stub | Agent runtime: `stub`, `subprocess`, or `grpc`. |
| `--db-driver` | sqlite | Store: `sqlite` or `postgres`. |
| `--db-url` | "" | PostgreSQL URL (or set `DATABASE_URL`). |
| `--env-file` | "" | Load env vars from file. |
| `--pprof` | "" | Enable pprof on address (e.g. `127.0.0.1:6060`). |
| `--interval` | 1.0 | Scheduler poll interval (seconds). |
| `--max-concurrent` | 32 | Max concurrent agent turns. |
| `--subprocess-cmd` | "" | Command for subprocess runtime. |
| `--subprocess-args` | [] | Args for subprocess runtime. |
| `--grpc-addr` | "" | gRPC server address for `runtime=grpc`. |

Run `agentary start --help` for the full list.

## Environment variables

| Variable | Description |
|----------|-------------|
| `AGENTARY_HOME` | Data directory (overrides `--home` default). |
| `AGENTARY_API_KEY` | If set, API requires `X-API-Key` or `api_key` query. |
| `DATABASE_URL` | PostgreSQL connection string when `--db-driver=postgres`. |
| `OPENAI_API_KEY` | Used by manager LLM (task breakdown). Also often used by subprocess/gRPC runtimes for agent turns. |
| `ANTHROPIC_API_KEY` | Used by runtimes that call Claude (e.g. subprocess or gRPC backend). |
| `SLACK_WEBHOOK_URL` | Optional Slack notifications. |
| `GITHUB_TOKEN` | Optional for GitHub notifier. |

Use `--env-file .env` to load from a file.

## Models and API keys

The **stub** runtime does not call any LLM. For **subprocess** or **grpc** runtimes, you supply the model backend; Agentary passes per-agent config (`model`, `max_tokens` from each agent’s `config.yaml`) to the runtime. Set the API key(s) your runtime expects:

| Provider | Env var | Typical models |
|----------|---------|----------------|
| OpenAI | `OPENAI_API_KEY` | gpt-4o, gpt-4o-mini, code-style models |
| Anthropic | `ANTHROPIC_API_KEY` | claude-3-5-sonnet, claude-3-opus |
| OpenRouter / others | As required by your runtime | Depends on runtime implementation |

Per-agent `config.yaml` under `teams/<team>/agents/<agent>/config.yaml` can set `model` and `max_tokens` so different agents use different models (e.g. manager on a stronger model, engineers on a faster one).

## Per-agent config (config.yaml)

Under `<home>/teams/<team>/agents/<agent>/config.yaml`:

```yaml
model: "gpt-4o-mini"
max_tokens: 4096
```

The runtime loads this when running a turn for that agent. Missing file means defaults.

## Team charter

The team charter is stored at `<home>/teams/<team>/charter.md`. It is plain markdown. Edit via the web UI (Charter view) or `GET`/`PUT` `/teams/:team/charter`. No special format; use it for mission, style, and conventions.
