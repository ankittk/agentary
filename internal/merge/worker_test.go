package merge

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ankittk/agentary/internal/store"
)

func TestWorker_runOnce_picksMergingTasks(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	st, err := store.Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()

	st.CreateTeam(ctx, "team1")
	st.CreateWorkflow(ctx, "team1", "default", 1, "builtin:default")
	wfID, _ := st.GetWorkflowIDByTeamAndName(ctx, "team1", "default", 1)
	id, _ := st.CreateTask(ctx, "team1", "Task", "todo", &wfID)
	_ = st.SetTaskWorkflowAndStage(ctx, id, wfID, "Merging")

	w := &Worker{Store: st, Interval: time.Hour}
	w.runOnce(ctx)
	// processTask will run; without worktree it will skip merge but still try to update stage
	task, _ := st.GetTaskByIDAndTeam(ctx, "team1", id)
	if task == nil {
		t.Fatal("task should exist")
	}
}

func TestWorker_Run_respectsContextCancellation(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	st, err := store.Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	w := &Worker{Store: st, Interval: 1 * time.Hour}
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()
	cancel()
	select {
	case <-done:
		// Run exited
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit after context cancel")
	}
}
