package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	agentrt "github.com/ankittk/agentary/internal/agent/runtime"
	"github.com/ankittk/agentary/internal/git"
	"github.com/ankittk/agentary/internal/memory"
	"github.com/ankittk/agentary/internal/store"
	"github.com/ankittk/agentary/pkg/models"
)

// Engine runs workflow stages: guard (no-op for minimal), assign (use current or pick), enter (no-op), dispatch, exit (no-op).
// For agent stage: runs runtime once; outcome drives transition. For terminal: marks task done. For human/auto: minimal stub.
// If Home is set, per-agent config is loaded and journal is appended after each agent turn.
type Engine struct {
	Store store.Store
	Home  string // optional: for agent config and journal
}

// RunTurn runs one workflow turn for the task. If task has no workflow_id, returns (false, nil) so caller can use legacy flow.
// Returns (true, nil) if turn was handled; (true, err) if error; (false, nil) if task has no workflow.
func (e *Engine) RunTurn(ctx context.Context, teamName string, task *store.Task, rt agentrt.Runtime, emit func(ev agentrt.Event)) (handled bool, err error) {
	if task.WorkflowID == nil || *task.WorkflowID == "" {
		return false, nil
	}
	wfID := *task.WorkflowID
	stageName := ""
	if task.CurrentStage != nil {
		stageName = *task.CurrentStage
	}
	if stageName == "" {
		initial, err := e.Store.GetWorkflowInitialStage(ctx, wfID)
		if err != nil {
			return true, err
		}
		stageName = initial
		if err := e.Store.SetTaskWorkflowAndStage(ctx, task.TaskID, wfID, stageName); err != nil {
			return true, err
		}
		task.CurrentStage = &stageName
	}

	stages, err := e.Store.GetWorkflowStages(ctx, wfID)
	if err != nil {
		return true, err
	}
	var stage *store.WorkflowStage
	for i := range stages {
		if stages[i].StageName == stageName {
			stage = &stages[i]
			break
		}
	}
	if stage == nil {
		return true, nil
	}

	switch stage.StageType {
	case "terminal":
		_ = e.Store.UpdateTask(ctx, task.TaskID, models.StatusDone, nil)
		return true, nil
	case "agent":
		// Dispatch: run runtime; outcome drives transition
		agentName := ""
		if task.Assignee != nil {
			agentName = *task.Assignee
		}
		allowlist, _ := e.Store.ListAllowedDomains(ctx)
		req := agentrt.TurnRequest{
			Team:             teamName,
			Agent:            agentName,
			TaskID:           &task.TaskID,
			Input:            task.Title,
			NetworkAllowlist: allowlist,
		}
		if e.Home != "" && agentName != "" {
			teamDir := memory.TeamDir(e.Home, teamName)
			agentDir := memory.AgentDir(teamDir, agentName)
			if cfg, _ := memory.LoadAgentConfig(agentDir); cfg != nil {
				req.Model = cfg.Model
				req.MaxTokens = cfg.MaxTokens
			}
		}
		result, runErr := rt.RunTurn(ctx, req, emit)
		if runErr != nil {
			_ = e.Store.SetTaskFailed(ctx, task.TaskID)
			return true, runErr
		}
		outcome := strings.TrimSpace(result.Output)
		if outcome == "" || outcome == "stub: ok" {
			outcome = "done"
		}
		// Append to agent journal after successful turn
		if e.Home != "" && agentName != "" {
			teamDir := memory.TeamDir(e.Home, teamName)
			j := &memory.Journal{AgentName: agentName, TeamDir: teamDir}
			_ = j.Append(ctx, memory.JournalEntry{
				TaskID:    task.TaskID,
				TaskTitle: task.Title,
				Outcome:   outcome,
				CreatedAt: time.Now().UTC(),
			})
		}
		nextStage, err := e.transition(ctx, wfID, stageName, outcome)
		if err != nil {
			return true, err
		}
		if nextStage != "" {
			_ = e.Store.UpdateTaskStage(ctx, task.TaskID, nextStage)
			if e.isTerminalStage(ctx, wfID, nextStage) {
				_ = e.Store.UpdateTask(ctx, task.TaskID, models.StatusDone, nil)
			}
		}
		return true, nil
	case "human", "auto":
		if stage.StageType == "auto" {
			nextStage, _ := e.transition(ctx, wfID, stageName, "done")
			if nextStage != "" {
				_ = e.Store.UpdateTaskStage(ctx, task.TaskID, nextStage)
				if e.isTerminalStage(ctx, wfID, nextStage) {
					_ = e.Store.UpdateTask(ctx, task.TaskID, models.StatusDone, nil)
				}
			}
		}
		return true, nil
	case "merge":
		// Run repo test_cmd in worktree first (CI); then merge.
		if task.WorktreePath != nil && *task.WorktreePath != "" {
			repos, _ := e.Store.ListRepos(ctx, teamName)
			var repo *store.Repo
			for i := range repos {
				if task.RepoName != nil && repos[i].Name == *task.RepoName {
					repo = &repos[i]
					break
				}
			}
			if repo == nil && len(repos) > 0 {
				repo = &repos[0]
			}
			if repo != nil && repo.TestCmd != nil && *repo.TestCmd != "" {
				if err := git.RunTestCmd(ctx, *task.WorktreePath, *repo.TestCmd); err != nil {
					_ = e.Store.SetTaskFailed(ctx, task.TaskID)
					return true, err
				}
			}
		}
		if task.WorktreePath != nil && *task.WorktreePath != "" && task.BranchName != nil && *task.BranchName != "" {
			if err := git.MergeInWorktree(ctx, *task.WorktreePath, *task.BranchName); err != nil {
				return true, fmt.Errorf("merge failed: %w", err)
			}
		}
		nextStage, _ := e.transition(ctx, wfID, stageName, "done")
		if nextStage != "" {
			_ = e.Store.UpdateTaskStage(ctx, task.TaskID, nextStage)
			if e.isTerminalStage(ctx, wfID, nextStage) {
				_ = e.Store.UpdateTask(ctx, task.TaskID, models.StatusDone, nil)
			}
		}
		return true, nil
	default:
		return true, nil
	}
}

func (e *Engine) transition(ctx context.Context, workflowID, fromStage, outcome string) (toStage string, err error) {
	transitions, err := e.Store.GetWorkflowTransitions(ctx, workflowID)
	if err != nil {
		return "", err
	}
	for _, t := range transitions {
		if t.FromStage == fromStage && t.Outcome == outcome {
			return t.ToStage, nil
		}
	}
	return "", nil
}

func (e *Engine) isTerminalStage(ctx context.Context, workflowID, stageName string) bool {
	stages, err := e.Store.GetWorkflowStages(ctx, workflowID)
	if err != nil {
		return false
	}
	for _, s := range stages {
		if s.StageName == stageName && s.StageType == "terminal" {
			return true
		}
	}
	return false
}
