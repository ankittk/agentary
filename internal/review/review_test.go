package review

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ankittk/agentary/internal/store"
)

func TestPickReviewer(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()

	_, _ = st.CreateTeam(ctx, "t1")
	_ = st.CreateAgent(ctx, "t1", "a1", "engineer")
	_ = st.CreateAgent(ctx, "t1", "a2", "engineer")
	wfID, _ := st.CreateWorkflow(ctx, "t1", "wf", 1, "builtin:wf")
	taskID, _ := st.CreateTask(ctx, "t1", "task", "todo", &wfID)
	task, _ := st.GetTaskByIDAndTeam(ctx, "t1", taskID)
	if task == nil {
		t.Fatal("task nil")
	}
	agents, _ := st.ListAgents(ctx, "t1")
	got := PickReviewer(ctx, st, "t1", task, agents)
	if got == "" {
		t.Fatal("PickReviewer: expected non-empty")
	}
}

func TestSubmitReview(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()

	_, _ = st.CreateTeam(ctx, "t1")
	_ = st.CreateAgent(ctx, "t1", "a1", "engineer")
	stages := []store.WorkflowStage{
		{StageName: "InReview", StageType: "agent", Outcomes: "approved", CandidateAgents: ""},
		{StageName: "Done", StageType: "terminal", Outcomes: "", CandidateAgents: ""},
	}
	transitions := []store.WorkflowTransition{
		{FromStage: "InReview", Outcome: "approved", ToStage: "Done"},
	}
	wfID, _ := st.CreateWorkflowWithStages(ctx, "t1", "wf", 1, "builtin:wf", stages, transitions)
	taskID, _ := st.CreateTask(ctx, "t1", "task", "todo", &wfID)
	task, _ := st.GetTaskByIDAndTeam(ctx, "t1", taskID)
	if task == nil {
		t.Fatal("task nil")
	}
	if err := SubmitReview(ctx, st, "t1", taskID, "a1", "approved", "looks good"); err != nil {
		t.Fatalf("SubmitReview: %v", err)
	}
	updated, _ := st.GetTaskByIDAndTeam(ctx, "t1", taskID)
	if updated == nil || updated.Status != "done" {
		t.Fatalf("SubmitReview: expected done, got %+v", updated)
	}
}

func TestSubmitReview_changesRequested(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), "home")
	st, err := store.Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	_, _ = st.CreateTeam(ctx, "t1")
	_ = st.CreateAgent(ctx, "t1", "a1", "engineer")
	stages := []store.WorkflowStage{
		{StageName: "InReview", StageType: "agent", Outcomes: "approved,changes_requested", CandidateAgents: ""},
		{StageName: "InProgress", StageType: "agent", Outcomes: "done", CandidateAgents: ""},
		{StageName: "Done", StageType: "terminal", Outcomes: "", CandidateAgents: ""},
	}
	transitions := []store.WorkflowTransition{
		{FromStage: "InReview", Outcome: "approved", ToStage: "Done"},
		{FromStage: "InReview", Outcome: "changes_requested", ToStage: "InProgress"},
		{FromStage: "InProgress", Outcome: "done", ToStage: "Done"},
	}
	wfID, _ := st.CreateWorkflowWithStages(ctx, "t1", "wf", 1, "builtin:wf", stages, transitions)
	taskID, _ := st.CreateTask(ctx, "t1", "task", "todo", &wfID)
	_ = st.SetTaskWorkflowAndStage(ctx, taskID, wfID, "InReview")
	dri := "a1"
	_ = st.UpdateTask(ctx, taskID, "", &dri)
	if err := SubmitReview(ctx, st, "t1", taskID, "a1", "changes_requested", "fix it"); err != nil {
		t.Fatalf("SubmitReview changes_requested: %v", err)
	}
	updated, _ := st.GetTaskByIDAndTeam(ctx, "t1", taskID)
	if updated == nil {
		t.Fatal("task nil")
	}
	if updated.CurrentStage != nil && *updated.CurrentStage == "Done" {
		t.Fatalf("changes_requested should return to InProgress, got stage %v", updated.CurrentStage)
	}
}

func TestPickReviewer_singleAgent(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), "home")
	st, err := store.Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	_, _ = st.CreateTeam(ctx, "t1")
	_ = st.CreateAgent(ctx, "t1", "alice", "engineer")
	wfID, _ := st.CreateWorkflow(ctx, "t1", "wf", 1, "builtin:wf")
	taskID, _ := st.CreateTask(ctx, "t1", "task", "todo", &wfID)
	task, _ := st.GetTaskByIDAndTeam(ctx, "t1", taskID)
	agents, _ := st.ListAgents(ctx, "t1")
	got := PickReviewer(ctx, st, "t1", task, agents)
	// Single agent: may return that agent (self-review fallback) or empty
	if len(agents) == 1 && got != "" && got != "alice" {
		t.Errorf("PickReviewer single agent: got %q", got)
	}
}
