---
title: "Getting Started"
description: "Install and run in a few minutes"
weight: 1
permalink: "/docs/getting-started/"
---

## Install

### From binary (recommended)

Download from [GitHub Releases](https://github.com/ankittk/agentary/releases) for your platform (macOS, Linux, Windows).

### From source

```bash
go install github.com/ankittk/agentary/cmd/agentary@latest
```

### Docker

```bash
docker run -p 3548:3548 -v agentary-data:/data ghcr.io/ankittk/agentary:latest
```

## Start

```bash
agentary start --foreground
```

Open **http://localhost:3548**.

## Create your first team

```bash
agentary team add --name myteam
agentary agent add myteam alice --role engineer
agentary agent add myteam bob --role engineer
agentary repo add --team myteam --name myrepo --source /path/to/repo
```

## Send your first task

Open http://localhost:3548, select **myteam**, type a task title, and click **Create**.

Or via API:

```bash
curl -X POST http://localhost:3548/teams/myteam/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title": "Add /health endpoint"}'
```

## What happens next

1. The scheduler picks up the task and assigns it to an available agent.
2. The agent works in a git worktree, writes code, and submits for review.
3. A reviewer agent checks the diff and approves or requests changes.
4. You approve the merge in the web UI (or set auto-approval).
5. The merge worker rebases onto main, runs tests, and merges.

## Web UI

The web dashboard is built into the binary - just open http://localhost:3548. It includes a Kanban board, agent monitor, workflow visualizer, charts, chat, reviews, network config, charter editor, and agent memory. No Node or npm is needed to run; everything is compiled into the single binary.
