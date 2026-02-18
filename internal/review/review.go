package review

import (
	"context"
	"strings"

	"github.com/ankittk/agentary/internal/store"
)

// PickReviewer chooses an agent to perform review for the task. Prefers someone other than the DRI (author).
// If the workflow stage "InReview" has candidate_agents, picks from that pool; otherwise any non-DRI engineer.
func PickReviewer(ctx context.Context, st store.Store, teamName string, task *store.Task, agents []store.Agent) string {
	dri := ""
	if task.DRI != nil {
		dri = *task.DRI
	}
	// Prefer InReview stage candidate pool if set
	if task.WorkflowID != nil && *task.WorkflowID != "" {
		stages, err := st.GetWorkflowStages(ctx, *task.WorkflowID)
		if err == nil {
			for _, s := range stages {
				if s.StageName == "InReview" && strings.TrimSpace(s.CandidateAgents) != "" {
					pool := strings.Split(s.CandidateAgents, ",")
					set := make(map[string]bool)
					for _, p := range pool {
						set[strings.TrimSpace(p)] = true
					}
					var candidates []store.Agent
					for _, a := range agents {
						if set[a.Name] && a.Name != dri {
							candidates = append(candidates, a)
						}
					}
					if len(candidates) > 0 {
						return candidates[0].Name
					}
				}
			}
		}
	}
	// Else: any agent that is not the DRI
	for _, a := range agents {
		if a.Name != dri {
			return a.Name
		}
	}
	if len(agents) > 0 {
		return agents[0].Name
	}
	return ""
}

// SubmitReview records a review (approve/changes_requested) and applies the workflow transition.
// If outcome is changes_requested, assignee is set back to the DRI so the task returns to the author.
func SubmitReview(ctx context.Context, st store.Store, teamName string, taskID int64, reviewerAgent, outcome, comments string) error {
	_, err := st.CreateTaskReview(ctx, teamName, taskID, reviewerAgent, outcome, comments)
	if err != nil {
		return err
	}
	task, err := st.GetTaskByIDAndTeam(ctx, teamName, taskID)
	if err != nil || task == nil || task.WorkflowID == nil || *task.WorkflowID == "" {
		return err
	}
	wfID := *task.WorkflowID
	stageName := ""
	if task.CurrentStage != nil {
		stageName = *task.CurrentStage
	}
	transitions, err := st.GetWorkflowTransitions(ctx, wfID)
	if err != nil {
		return err
	}
	var toStage string
	for _, t := range transitions {
		if t.FromStage == stageName && t.Outcome == outcome {
			toStage = t.ToStage
			break
		}
	}
	if toStage == "" {
		return nil
	}
	if err := st.SetTaskWorkflowAndStage(ctx, taskID, wfID, toStage); err != nil {
		return err
	}
	// Return task to author when changes requested
	if outcome == "changes_requested" && task.DRI != nil && *task.DRI != "" {
		_ = st.UpdateTask(ctx, taskID, "in_progress", task.DRI)
	}
	// Mark done if terminal
	stages, _ := st.GetWorkflowStages(ctx, wfID)
	for _, s := range stages {
		if s.StageName == toStage && s.StageType == "terminal" {
			_ = st.UpdateTask(ctx, taskID, "done", nil)
			break
		}
	}
	return nil
}
