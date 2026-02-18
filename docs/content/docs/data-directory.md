---
title: "Data directory"
description: "Home directory layout"
weight: 6
---

Agentary stores all runtime data under a single home directory. The default is `~/.agentary/`. Override it with the `AGENTARY_HOME` environment variable or the `--home` CLI flag.

## Target layout

```
~/.agentary/
├── members/                # Human identities (from git config)
│   └── <username>.yaml
├── teams/
│   └── <team-name>/
│       ├── agents/         # Per-agent directories
│       │   ├── agentary/   # Manager agent (the delegate)
│       │   │   ├── journal.md    # Accumulated context
│       │   │   ├── notes/        # Agent scratch notes
│       │   │   └── config.yaml   # Per-agent model config
│       │   ├── alice/      # Engineer agent
│       │   │   ├── journal.md
│       │   │   ├── notes/
│       │   │   └── config.yaml
│       │   └── bob/
│       ├── repos/          # Symlinks to real git repos
│       ├── shared/         # Team-wide shared files (conventions, patterns)
│       ├── charter.md      # Team charter: review standards, norms
│       └── workflows/      # Registered workflow YAML definitions
├── protected/
│   ├── db.sqlite           # Database (outside agent sandbox)
│   ├── network.yaml        # Network allowlist (agents cannot tamper)
│   └── bootstrap_id
└── config.yaml             # Global config
```

## Directories

| Path | Purpose |
|------|---------|
| `members/` | Human identities (e.g. from `git config user.name` / `user.email`) for commit attribution and review approvals. |
| `teams/<team>/agents/` | One directory per agent. Each can have `journal.md`, `notes/`, and `config.yaml` (model, max_tokens). |
| `teams/<team>/repos/` | Symlinks or references to git repositories the team uses. |
| `teams/<team>/shared/` | Team-wide context: coding conventions, API patterns, architecture decisions. |
| `teams/<team>/charter.md` | Team charter: review standards, communication norms, code style, testing requirements. |
| `teams/<team>/workflows/` | Workflow YAML (or builtin) definitions. |
| `protected/` | Data that must not be writable by agent processes: database, network allowlist, bootstrap id. |

**Agent memory** is implemented: creating a team (CLI or API) creates `teams/<team>/` and `teams/<team>/shared/`. Creating an agent creates `teams/<team>/agents/<agent>/` and `agents/<agent>/notes/`. The workflow engine appends to `journal.md` after each agent turn and loads per-agent `config.yaml` (model, max_tokens) when running turns. Use `agentary identity detect` to detect git user name/email and save to `members/<username>.yaml`. Team charter is at `teams/<team>/charter.md` (GET/PUT via API or edit on disk).
