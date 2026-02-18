---
title: "Agent Memory"
weight: 7
---

Agentary gives each agent persistent memory so they can learn from past tasks and follow team conventions.

## Journal (per agent)

Each agent has a **journal** stored as `journal.md` under the agent directory (`<home>/teams/<team>/agents/<agent>/journal.md`). After each turn, the workflow engine can append an entry with task ID, outcome, decisions, and patterns. This context is available to the agent on the next run (e.g. via summary or last N characters).

- **Append:** Done by the workflow engine after a successful turn.
- **Read:** Use `GET /teams/:team/agents/:agent/journal` (optional `?limit=` in bytes).
- **Summary:** A short summary can be generated for injection into agent context.

## Notes directory

Each agent has a `notes/` subdirectory under their agent dir for arbitrary files (e.g. scratch files, saved snippets). The write guard controls what paths the agent can write to.

## Team shared directory

Under `<home>/teams/<team>/shared/` the team can store files that all agents can read. Use this for shared conventions, examples, or reference docs.

## Team charter

The **charter** is a markdown file at `<home>/teams/<team>/charter.md`. It describes the team’s mission, coding style, and conventions. You can edit it in the web UI (Charter view) or via `GET`/`PUT` `/teams/:team/charter`. The charter is included in context so agents follow team rules.

## Per-agent config (config.yaml)

Each agent can have a `config.yaml` in their agent directory with model settings (e.g. `model`, `max_tokens`). This is loaded by the runtime when running a turn. Use `GET /teams/:team/agents/:agent/config` to read it from the API.

## How memory accumulates

- **Journal:** Grows with each task turn; entries are appended. Reading with a limit returns the most recent content.
- **Charter:** Edited by humans or API; single file.
- **Config:** Edited by humans or tooling; single file per agent.
- **Notes / shared:** Written by agents (within sandbox) or by users.

All of this lives under `AGENTARY_HOME`. See [Data directory]({{< ref "docs/data-directory" >}}) for the full layout.
