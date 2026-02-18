package manager

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ankittk/agentary/internal/httpapi"
	"github.com/ankittk/agentary/internal/store"
	"github.com/ankittk/agentary/pkg/models"
)

func testApp(t *testing.T) (*httpapi.App, context.Context) {
	t.Helper()
	home := filepath.Join(t.TempDir(), "home")
	st, err := store.Open(home)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	app, err := httpapi.NewApp(httpapi.ServerOptions{Home: home, Addr: ":0"})
	if err != nil {
		_ = st.Close()
		t.Fatalf("NewApp: %v", err)
	}
	return app, context.Background()
}

func TestCreateTaskForTeam(t *testing.T) {
	app, ctx := testApp(t)
	defer func() { _ = app.Store.Close() }()

	if _, err := app.Store.CreateTeam(ctx, "team1"); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if _, err := app.Store.CreateWorkflow(ctx, "team1", "default", 1, "builtin:default"); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	id, err := CreateTaskForTeam(ctx, app.Store, "team1", "Fix bug", models.StatusTodo)
	if err != nil {
		t.Fatalf("CreateTaskForTeam: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected task id > 0, got %d", id)
	}
	task, err := app.Store.GetTaskByIDAndTeam(ctx, "team1", id)
	if err != nil || task == nil {
		t.Fatalf("GetTaskByIDAndTeam: %v", err)
	}
	if task.Title != "Fix bug" || task.Status != models.StatusTodo {
		t.Fatalf("task: title=%q status=%q", task.Title, task.Status)
	}
}

func TestAssignTask(t *testing.T) {
	app, ctx := testApp(t)
	defer func() { _ = app.Store.Close() }()

	app.Store.CreateTeam(ctx, "team1")
	app.Store.CreateAgent(ctx, "team1", "alice", "engineer")
	id, _ := app.Store.CreateTask(ctx, "team1", "Task", models.StatusTodo, nil)

	if err := AssignTask(ctx, app.Store, "team1", id, "alice"); err != nil {
		t.Fatalf("AssignTask: %v", err)
	}
	task, _ := app.Store.GetTaskByIDAndTeam(ctx, "team1", id)
	if task == nil || task.Assignee == nil || *task.Assignee != "alice" {
		t.Fatalf("expected assignee alice, got %+v", task)
	}
}

func TestAdvanceTask(t *testing.T) {
	app, ctx := testApp(t)
	defer func() { _ = app.Store.Close() }()

	app.Store.CreateTeam(ctx, "team1")
	id, _ := app.Store.CreateTask(ctx, "team1", "Task", models.StatusTodo, nil)

	if err := AdvanceTask(ctx, app.Store, id, models.StatusInProgress, nil); err != nil {
		t.Fatalf("AdvanceTask in_progress: %v", err)
	}
	task, _ := app.Store.GetTaskByIDAndTeam(ctx, "team1", id)
	if task == nil || task.Status != models.StatusInProgress {
		t.Fatalf("expected status in_progress, got %+v", task)
	}

	if err := AdvanceTask(ctx, app.Store, id, models.StatusDone, nil); err != nil {
		t.Fatalf("AdvanceTask done: %v", err)
	}
	task, _ = app.Store.GetTaskByIDAndTeam(ctx, "team1", id)
	if task == nil || task.Status != models.StatusDone {
		t.Fatalf("expected status done, got %+v", task)
	}

	// Invalid status should be ignored (returns nil, no error).
	if err := AdvanceTask(ctx, app.Store, id, "invalid", nil); err != nil {
		t.Fatalf("AdvanceTask invalid: %v", err)
	}
}

func TestHandleEvent_taskUpdateDone_createsFollowUpTask(t *testing.T) {
	app, ctx := testApp(t)
	defer func() { _ = app.Store.Close() }()

	app.Store.CreateTeam(ctx, "team1")
	app.Store.CreateWorkflow(ctx, "team1", "default", 1, "builtin:default")
	taskID, _ := app.Store.CreateTask(ctx, "team1", "Original task", models.StatusTodo, nil)
	_ = app.Store.UpdateTask(ctx, taskID, models.StatusDone, nil)

	handleEvent(ctx, app, map[string]any{
		"type":   "task_update",
		"team":   "team1",
		"task_id": float64(taskID),
		"status": models.StatusDone,
	})

	tasks, _ := app.Store.ListTasks(ctx, "team1", 10)
	var found bool
	for _, tk := range tasks {
		if tk.Title == "Review: Original task" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected follow-up Review task, got %d tasks: %+v", len(tasks), tasks)
	}
}

func TestHandleEvent_ignoresNonTaskUpdate(t *testing.T) {
	app, ctx := testApp(t)
	defer func() { _ = app.Store.Close() }()

	app.Store.CreateTeam(ctx, "team1")
	before, _ := app.Store.ListTasks(ctx, "team1", 10)

	// Publish non-task_update event; handleEvent should ignore.
	app.Hub.PublishJSON(map[string]any{"type": "message", "team": "team1"})
	app.Hub.PublishJSON(map[string]any{"type": "team_update", "team": "team1"})

	after, _ := app.Store.ListTasks(ctx, "team1", 10)
	if len(after) != len(before) {
		t.Fatalf("expected no new tasks from non-task_update events, before=%d after=%d", len(before), len(after))
	}
}

func TestShouldRequeueFailed_returnsFalse(t *testing.T) {
	if shouldRequeueFailed(map[string]any{"status": "failed"}) {
		t.Fatal("shouldRequeueFailed should return false by default")
	}
}
