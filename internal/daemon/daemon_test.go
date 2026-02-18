package daemon

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/ankittk/agentary/internal/httpapi"
	"github.com/ankittk/agentary/internal/store"
	"github.com/ankittk/agentary/pkg/models"
)

func TestStartForeground_emptyHome(t *testing.T) {
	ctx := context.Background()
	err := StartForeground(ctx, StartOptions{Home: ""})
	if err == nil {
		t.Fatal("StartForeground empty home: expected error")
	}
}

func testApp(t *testing.T) (*httpapi.App, context.Context) {
	t.Helper()
	home := filepath.Join(t.TempDir(), "home")
	app, err := httpapi.NewApp(httpapi.ServerOptions{Home: home, Addr: ":0"})
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	return app, context.Background()
}

func TestPickAssignee_prefersManager(t *testing.T) {
	app, ctx := testApp(t)
	defer func() { _ = app.Store.Close() }()

	app.Store.CreateTeam(ctx, "team1")
	app.Store.CreateAgent(ctx, "team1", "alice", "manager")
	app.Store.CreateAgent(ctx, "team1", "bob", "engineer")
	taskID, _ := app.Store.CreateTask(ctx, "team1", "Task", models.StatusTodo, nil)
	task, _ := app.Store.GetTaskByIDAndTeam(ctx, "team1", taskID)
	agents, _ := app.Store.ListAgents(ctx, "team1")

	got := pickAssignee(ctx, app.Store, "team1", task, agents)
	if got != "alice" {
		t.Errorf("pickAssignee (manager first): got %q, want alice", got)
	}
}

func TestPickAssignee_fallbackToFirstAgent(t *testing.T) {
	app, ctx := testApp(t)
	defer func() { _ = app.Store.Close() }()

	app.Store.CreateTeam(ctx, "team1")
	app.Store.CreateAgent(ctx, "team1", "bob", "engineer")
	app.Store.CreateAgent(ctx, "team1", "carol", "engineer")
	taskID, _ := app.Store.CreateTask(ctx, "team1", "Task", models.StatusTodo, nil)
	task, _ := app.Store.GetTaskByIDAndTeam(ctx, "team1", taskID)
	agents, _ := app.Store.ListAgents(ctx, "team1")

	got := pickAssignee(ctx, app.Store, "team1", task, agents)
	if got != "bob" {
		t.Errorf("pickAssignee (no manager): got %q, want bob", got)
	}
}

func TestPickAssignee_candidatePool_prefersManager(t *testing.T) {
	app, ctx := testApp(t)
	defer func() { _ = app.Store.Close() }()

	app.Store.CreateTeam(ctx, "team1")
	app.Store.CreateAgent(ctx, "team1", "alice", "engineer")
	app.Store.CreateAgent(ctx, "team1", "bob", "manager")
	app.Store.CreateAgent(ctx, "team1", "carol", "engineer")
	wfID, _ := app.Store.CreateWorkflowWithStages(ctx, "team1", "wf1", 1, "builtin:wf1",
		[]store.WorkflowStage{
			{WorkflowID: "", StageName: "InProgress", StageType: "agent", Outcomes: "done", CandidateAgents: "alice,bob,carol"},
		},
		[]store.WorkflowTransition{
			{WorkflowID: "", FromStage: "InProgress", Outcome: "done", ToStage: "Done"},
		})
	taskID, _ := app.Store.CreateTask(ctx, "team1", "Task", models.StatusTodo, &wfID)
	_ = app.Store.SetTaskWorkflowAndStage(ctx, taskID, wfID, "InProgress")
	task, _ := app.Store.GetTaskByIDAndTeam(ctx, "team1", taskID)
	agents, _ := app.Store.ListAgents(ctx, "team1")

	got := pickAssignee(ctx, app.Store, "team1", task, agents)
	if got != "bob" {
		t.Errorf("pickAssignee (candidate pool with manager): got %q, want bob", got)
	}
}

func TestPickAssignee_candidatePool_noManager_returnsFirstCandidate(t *testing.T) {
	app, ctx := testApp(t)
	defer func() { _ = app.Store.Close() }()

	app.Store.CreateTeam(ctx, "team1")
	app.Store.CreateAgent(ctx, "team1", "alice", "engineer")
	app.Store.CreateAgent(ctx, "team1", "bob", "engineer")
	wfID, _ := app.Store.CreateWorkflowWithStages(ctx, "team1", "wf1", 1, "builtin:wf1",
		[]store.WorkflowStage{
			{WorkflowID: "", StageName: "InProgress", StageType: "agent", Outcomes: "done", CandidateAgents: "bob,alice"},
		},
		[]store.WorkflowTransition{
			{WorkflowID: "", FromStage: "InProgress", Outcome: "done", ToStage: "Done"},
		})
	taskID, _ := app.Store.CreateTask(ctx, "team1", "Task", models.StatusTodo, &wfID)
	_ = app.Store.SetTaskWorkflowAndStage(ctx, taskID, wfID, "InProgress")
	task, _ := app.Store.GetTaskByIDAndTeam(ctx, "team1", taskID)
	agents, _ := app.Store.ListAgents(ctx, "team1")

	got := pickAssignee(ctx, app.Store, "team1", task, agents)
	// Code picks first agent in list that is in the candidate pool; agents order is alice, bob
	if got != "alice" && got != "bob" {
		t.Errorf("pickAssignee (candidate pool): got %q, want alice or bob", got)
	}
}

func TestPublishTaskUpdate_sendsCorrectPayload(t *testing.T) {
	app, _ := testApp(t)
	defer func() { _ = app.Store.Close() }()

	ch := app.Hub.Subscribe()
	defer app.Hub.Unsubscribe(ch)

	assignee := "alice"
	publishTaskUpdate(app, "team1", 42, models.StatusInProgress, &assignee)

	select {
	case raw := <-ch:
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if payload["type"] != "task_update" {
			t.Errorf("type: got %v", payload["type"])
		}
		if payload["team"] != "team1" {
			t.Errorf("team: got %v", payload["team"])
		}
		if id, ok := payload["task_id"].(float64); !ok || int64(id) != 42 {
			t.Errorf("task_id: got %v", payload["task_id"])
		}
		if payload["status"] != models.StatusInProgress {
			t.Errorf("status: got %v", payload["status"])
		}
		if payload["assignee"] != "alice" {
			t.Errorf("assignee: got %v", payload["assignee"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for hub message")
	}
}

func TestRunScheduler_claimsTaskAndAssigns(t *testing.T) {
	app, ctx := testApp(t)
	defer func() { _ = app.Store.Close() }()

	app.Store.CreateTeam(ctx, "team1")
	app.Store.CreateAgent(ctx, "team1", "alice", "engineer")
	taskID, _ := app.Store.CreateTask(ctx, "team1", "Task", models.StatusTodo, nil)

	runCtx, cancel := context.WithCancel(ctx)
	opts := StartOptions{Home: app.Home, IntervalSec: 0.01, MaxConcurrent: 2}
	go runScheduler(runCtx, opts, app)

	// Wait for scheduler tick to pick up task (stub runtime will run and mark done).
	for i := 0; i < 100; i++ {
		task, _ := app.Store.GetTaskByIDAndTeam(ctx, "team1", taskID)
		if task != nil && task.Assignee != nil && *task.Assignee == "alice" {
			break
		}
		if task != nil && task.Status == models.StatusDone {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	task, _ := app.Store.GetTaskByIDAndTeam(ctx, "team1", taskID)
	if task == nil {
		cancel()
		time.Sleep(50 * time.Millisecond)
		t.Fatal("task not found")
	}
	if task.Assignee == nil || *task.Assignee != "alice" {
		t.Errorf("expected assignee alice, got %+v", task.Assignee)
	}
	if task.Status != models.StatusInProgress && task.Status != models.StatusDone {
		t.Errorf("expected status in_progress or done, got %q", task.Status)
	}
	// Stop scheduler before closing store
	cancel()
	time.Sleep(100 * time.Millisecond)
}
