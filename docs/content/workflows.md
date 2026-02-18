---
title: "Workflows"
permalink: "/docs/workflows/"
weight: 9
---

Workflows define how tasks move through stages (e.g. todo → in progress → review → merge → done).

## Default workflow stages

The built-in default workflow has these stages:

| Stage | Type | Outcomes |
|-------|------|----------|
| Coding | agent | submit_for_review, done |
| InReview | agent | approved, changes_requested |
| InApproval | human | approved, changes_requested |
| Merging | merge | done |
| Done | terminal | - |

- **agent:** An agent runs a turn; outcome is chosen by the runtime (e.g. submit_for_review, done).
- **human:** A human approves or requests changes in the web UI (Reviews panel) or via `POST /teams/:team/tasks/:id/approve`.
- **merge:** The merge worker rebases, runs tests, and merges; then moves to Done.
- **terminal:** No further transitions.

## Transitions

Transitions are stored per workflow (e.g. "from Coding, outcome submit_for_review → to InReview"). The workflow engine uses them to advance the task’s current stage when an outcome is submitted (e.g. after a turn or after human approval).

## Creating custom workflows

Use the API to create workflows and stages:

- `POST /teams/:team/workflows` - create a workflow.
- Stages and transitions are created when the workflow is initialized (e.g. via workflow init or seed).

Each stage has a **type** (agent, human, auto, terminal, merge) and **outcomes** (comma-separated). Transitions map (from_stage, outcome) → to_stage.

## Candidate agent pools

Stages can have candidate agents (e.g. engineers for Coding, reviewers for InReview). The scheduler picks an assignee from the pool when claiming a task. The review module picks a reviewer different from the DRI when moving to InReview.
