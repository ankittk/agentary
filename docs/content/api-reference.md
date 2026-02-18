---
title: "API reference"
permalink: "/docs/api-reference/"
description: "HTTP endpoints"
weight: 3
---

All endpoints are HTTP/JSON. Base URL is the server (e.g. `http://localhost:3548`). Optional: set `X-API-Key` or query `api_key` when the server is configured with an API key.

## Health and config

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check; returns `{"ok": true}`. |
| GET | `/metrics` | Prometheus metrics (or legacy task gauges). |
| GET | `/config` | Config blob (human_name, hc_home, bootstrap_id). |
| GET | `/bootstrap` | Full bootstrap: config, teams, initial_team, tasks, agents, repos, workflows, network allowlist. |

## Teams

| Method | Path | Description |
|--------|------|-------------|
| GET | `/teams` | List teams. |
| POST | `/teams` | Create team; body `{"name": "..."}`. |

## Team-scoped

All under `/teams/{team}/`. Replace `{team}` with the team name.

### Tasks

| Method | Path | Description |
|--------|------|-------------|
| GET | `/teams/{team}/tasks` | List tasks (optional query `limit`). |
| POST | `/teams/{team}/tasks` | Create task; body `{"title": "...", "status": "todo" \| "in_progress"}`. |
| GET | `/teams/{team}/tasks/{id}` | Get one task. |
| PATCH | `/teams/{team}/tasks/{id}` | Update task; body `{"status": "...", "assignee": "..."}`. |
| GET | `/teams/{team}/tasks/{id}/comments` | List comments. |
| POST | `/teams/{team}/tasks/{id}/comments` | Add comment; body `{"author": "...", "body": "..."}`. |
| GET | `/teams/{team}/tasks/{id}/attachments` | List attachments. |
| POST | `/teams/{team}/tasks/{id}/attachments` | Add attachment; body `{"file_path": "..."}`. |
| DELETE | `/teams/{team}/tasks/{id}/attachments?file_path=...` | Remove attachment. |
| GET | `/teams/{team}/tasks/{id}/dependencies` | List dependencies. |
| POST | `/teams/{team}/tasks/{id}/dependencies` | Add dependency; body `{"depends_on_task_id": id}`. |
| GET | `/teams/{team}/tasks/{id}/diff` | Git diff for review (worktree). |
| GET | `/teams/{team}/tasks/{id}/reviews` | List reviews. |
| POST | `/teams/{team}/tasks/{id}/approve` | Approve/review outcome; body `{"outcome": "approved" \| "changes_requested"}`. |
| POST | `/teams/{team}/tasks/{id}/request-review` | Move to InReview and assign reviewer. |
| POST | `/teams/{team}/tasks/{id}/submit-review` | Submit review; body `{"reviewer_agent", "outcome", "comments"}`. |

### Agents, charter, repos, workflows, messages

| Method | Path | Description |
|--------|------|-------------|
| GET | `/teams/{team}/agents` | List agents. |
| POST | `/teams/{team}/agents` | Create agent; body `{"name": "...", "role": "..."}`. |
| GET | `/teams/{team}/charter` | Get charter content. |
| PUT | `/teams/{team}/charter` | Set charter; body `{"content": "..."}`. |
| GET | `/teams/{team}/repos` | List repos. |
| POST | `/teams/{team}/repos` | Create repo; body `{"name", "source", "approval", "test_cmd"}`. |
| GET | `/teams/{team}/workflows` | List workflows. |
| POST | `/teams/{team}/workflows` | Create workflow; body `{"name", "version", "source"}`. |
| POST | `/teams/{team}/workflows/init` | Init default workflow. |
| GET | `/teams/{team}/messages?recipient=...` | List messages (inbox for recipient). |
| POST | `/teams/{team}/messages` | Send message; body `{"sender", "recipient", "content"}`. |

## Network

| Method | Path | Description |
|--------|------|-------------|
| GET | `/network` | Get allowlist. |
| POST | `/network/reset` | Reset allowlist. |
| POST | `/network/allow` | Allow domain; body `{"domain": "..."}`. |
| POST | `/network/disallow` | Disallow domain; body `{"domain": "..."}`. |

## SSE

| Method | Path | Description |
|--------|------|-------------|
| GET | `/stream` | Server-Sent Events stream. Sends `connected` and then events (e.g. `task_update`, `team_update`, `message`). |

## Errors

JSON error responses use `{"error": "message"}` with an appropriate HTTP status (400, 404, 405, 500).
