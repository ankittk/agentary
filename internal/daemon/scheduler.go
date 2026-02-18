package daemon

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	agentrt "github.com/ankittk/agentary/internal/agent/runtime"
	agentrtgrpc "github.com/ankittk/agentary/internal/agent/runtime/grpc"
	"github.com/ankittk/agentary/internal/httpapi"
	"github.com/ankittk/agentary/internal/otel"
	"github.com/ankittk/agentary/internal/review"
	"github.com/ankittk/agentary/internal/store"
	"github.com/ankittk/agentary/internal/workflow"
	"github.com/ankittk/agentary/pkg/models"
)

// runScheduler periodically picks runnable tasks (todo/in_progress) per team, assigns an agent, runs a turn via the stub runtime, and publishes SSE events (including task_update).
func runScheduler(ctx context.Context, opts StartOptions, app *httpapi.App) {
	interval := time.Duration(opts.IntervalSec * float64(time.Second))
	if interval <= 0 {
		interval = 1 * time.Second
	}
	max := opts.MaxConcurrent
	if max <= 0 {
		max = models.DefaultSchedulerChanSize
	}

	sem := make(chan struct{}, max)
	baseRt := agentrt.Runtime(agentrt.StubRuntime{})
	switch {
	case opts.Runtime == "grpc" && opts.GrpcAddr != "":
		baseRt = &agentrtgrpc.Client{Addr: opts.GrpcAddr}
	case opts.Runtime == "subprocess" && opts.SubprocessCmd != "":
		baseRt = agentrt.SubprocessRuntime{
			Command:        opts.SubprocessCmd,
			Args:           opts.SubprocessArgs,
			SandboxHome:    opts.SandboxHome,
			SandboxTeamDir: "", // set per-team below
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			teams, err := app.Store.ListTeams(ctx)
			if err != nil {
				slog.Error("scheduler list teams failed", "err", err)
				continue
			}

			var wg sync.WaitGroup
			for _, t := range teams {
				task, err := app.Store.NextRunnableTaskForTeam(ctx, t.Name)
				if err != nil || task == nil {
					continue
				}

				agents, err := app.Store.ListAgents(ctx, t.Name)
				if err != nil || len(agents) == 0 {
					continue
				}

				// Build runtime for this team so sandbox can restrict writes to team dir only.
				rt := baseRt
				if sub, ok := baseRt.(agentrt.SubprocessRuntime); ok && sub.SandboxHome != "" && opts.Home != "" {
					sub.SandboxTeamDir = filepath.Join(opts.Home, "teams", t.Name)
					rt = sub
				}

				// Candidate pool: if task has workflow + current_stage with candidate_agents, pick assignee from that pool; else prefer manager, then first agent.
				agentName := pickAssignee(ctx, app.Store, t.Name, task, agents)

				select {
				case sem <- struct{}{}:
				case <-ctx.Done():
					continue
				}

				wg.Add(1)
				taskCopy := *task
				taskID := task.TaskID
				taskTitle := task.Title
				agentsCopy := make([]store.Agent, len(agents))
				copy(agentsCopy, agents)
				go func(teamName, agent string, tid int64, title string, tk *store.Task, runtime agentrt.Runtime, agentsList []store.Agent) {
					defer wg.Done()
					defer func() { <-sem }()

					// Claim only if still todo (prevents double-processing)
					claimed, err := app.Store.ClaimTask(ctx, teamName, tid, agent)
					if err != nil {
						slog.Error("scheduler claim task failed", "task_id", tid, "err", err)
						return
					}
					if !claimed {
						return // another worker got it or it's no longer todo
					}
					otel.RecordTaskOp(ctx, "claim", teamName, "in_progress")
					publishTaskUpdate(app, teamName, tid, "in_progress", &agent)

					turnStart := time.Now()
					eng := &workflow.Engine{Store: app.Store, Home: opts.Home}
					handled, err := eng.RunTurn(ctx, teamName, tk, runtime, func(ev agentrt.Event) {
						if ev.Timestamp.IsZero() {
							ev.Timestamp = time.Now().UTC()
						}
						app.Hub.PublishJSON(ev)
					})
					if handled {
						otel.RecordAgentTurn(ctx, teamName, agent, time.Since(turnStart))
						if err != nil {
							_ = app.Store.SetTaskFailed(ctx, tid)
							app.Hub.PublishJSON(map[string]any{
								"type":      "agent_activity",
								"team":      teamName,
								"agent":     agent,
								"task_id":   tid,
								"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
								"tool":      "error",
								"error":     err.Error(),
							})
							publishTaskUpdate(app, teamName, tid, "failed", nil)
						} else {
							updated, _ := app.Store.GetTaskByIDAndTeam(ctx, teamName, tid)
							if updated != nil {
								// When transitioned to InReview, assign a reviewer (different from DRI)
								if updated.CurrentStage != nil && *updated.CurrentStage == "InReview" && len(agentsList) > 0 {
									reviewer := review.PickReviewer(ctx, app.Store, teamName, updated, agentsList)
									if reviewer != "" {
										_ = app.Store.UpdateTask(ctx, tid, "", &reviewer)
										updated.Assignee = &reviewer
									}
								}
								publishTaskUpdate(app, teamName, tid, updated.Status, updated.Assignee)
							}
						}
						return
					}

					allowlist, _ := app.Store.ListAllowedDomains(ctx)
					_, err = runtime.RunTurn(ctx, agentrt.TurnRequest{
						Team:             teamName,
						Agent:            agent,
						TaskID:           &tid,
						Input:            title,
						NetworkAllowlist: allowlist,
					}, func(ev agentrt.Event) {
						if ev.Timestamp.IsZero() {
							ev.Timestamp = time.Now().UTC()
						}
						app.Hub.PublishJSON(ev)
					})
					otel.RecordAgentTurn(ctx, teamName, agent, time.Since(turnStart))
					if err != nil {
						_ = app.Store.SetTaskFailed(ctx, tid)
						app.Hub.PublishJSON(map[string]any{
							"type":      "agent_activity",
							"team":      teamName,
							"agent":     agent,
							"task_id":   tid,
							"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
							"tool":      "error",
							"error":     err.Error(),
						})
						publishTaskUpdate(app, teamName, tid, "failed", nil)
						return
					}

					if err := app.Store.UpdateTask(ctx, tid, models.StatusDone, nil); err != nil {
						slog.Error("scheduler update task to done failed", "task_id", tid, "err", err)
						return
					}
					publishTaskUpdate(app, teamName, tid, models.StatusDone, nil)
				}(t.Name, agentName, taskID, taskTitle, &taskCopy, rt, agentsCopy)
			}
			wg.Wait()
		}
	}
}

func pickAssignee(ctx context.Context, st store.Store, teamName string, task *store.Task, agents []store.Agent) string {
	if task.WorkflowID != nil && *task.WorkflowID != "" && task.CurrentStage != nil && *task.CurrentStage != "" {
		stages, err := st.GetWorkflowStages(ctx, *task.WorkflowID)
		if err == nil {
			for _, s := range stages {
				if s.StageName == *task.CurrentStage && strings.TrimSpace(s.CandidateAgents) != "" {
					pool := strings.Split(s.CandidateAgents, ",")
					set := make(map[string]bool)
					for _, p := range pool {
						set[strings.TrimSpace(p)] = true
					}
					var candidates []store.Agent
					for _, a := range agents {
						if set[a.Name] {
							candidates = append(candidates, a)
						}
					}
					if len(candidates) > 0 {
						for _, a := range candidates {
							if a.Role == "manager" {
								return a.Name
							}
						}
						return candidates[0].Name
					}
				}
			}
		}
	}
	// Default: prefer manager, else first agent.
	for _, a := range agents {
		if a.Role == "manager" {
			return a.Name
		}
	}
	return agents[0].Name
}

func publishTaskUpdate(app *httpapi.App, team string, taskID int64, status string, assignee *string) {
	payload := map[string]any{"type": "task_update", "team": team, "task_id": taskID, "status": status}
	if assignee != nil {
		payload["assignee"] = *assignee
	}
	app.Hub.PublishJSON(payload)
}
