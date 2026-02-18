package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	agentrt "github.com/ankittk/agentary/internal/agent/runtime"
	"github.com/ankittk/agentary/internal/store"
)

func TestEngine_RunTurn_noWorkflow(t *testing.T) {
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
	_ = st.CreateAgent(ctx, "t1", "a1", "role")
	taskID, _ := st.CreateTask(ctx, "t1", "task1", "todo", nil)
	task, _ := st.GetTaskByIDAndTeam(ctx, "t1", taskID)
	if task == nil {
		t.Fatal("task nil")
	}

	eng := &Engine{Store: st, Home: ""}
	handled, err := eng.RunTurn(ctx, "t1", task, agentrt.StubRuntime{}, func(agentrt.Event) {})
	if err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if handled {
		t.Fatal("expected handled=false when task has no workflow")
	}
}

func TestEngine_RunTurn_terminalStage(t *testing.T) {
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
	_ = st.CreateAgent(ctx, "t1", "a1", "role")
	stages := []store.WorkflowStage{
		{WorkflowID: "", StageName: "done", StageType: "terminal", Outcomes: "", CandidateAgents: ""},
	}
	wfID, err := st.CreateWorkflowWithStages(ctx, "t1", "wf", 1, "builtin:wf", stages, nil)
	if err != nil {
		t.Fatalf("CreateWorkflowWithStages: %v", err)
	}
	taskID, err := st.CreateTask(ctx, "t1", "task1", "todo", &wfID)
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	task, _ := st.GetTaskByIDAndTeam(ctx, "t1", taskID)
	if task == nil {
		t.Fatal("task nil")
	}

	eng := &Engine{Store: st, Home: ""}
	handled, err := eng.RunTurn(ctx, "t1", task, agentrt.StubRuntime{}, func(agentrt.Event) {})
	if err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if !handled {
		t.Fatal("expected handled=true for workflow with terminal stage")
	}
	updated, _ := st.GetTaskByIDAndTeam(ctx, "t1", taskID)
	if updated == nil || updated.Status != "done" {
		t.Fatalf("expected task status done, got %+v", updated)
	}
}

func TestEngine_transition_isTerminalStage(t *testing.T) {
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
	stages := []store.WorkflowStage{
		{StageName: "start", StageType: "agent", Outcomes: "done", CandidateAgents: ""},
		{StageName: "done", StageType: "terminal", Outcomes: "", CandidateAgents: ""},
	}
	transitions := []store.WorkflowTransition{
		{FromStage: "start", Outcome: "done", ToStage: "done"},
	}
	wfID, err := st.CreateWorkflowWithStages(ctx, "t1", "wf2", 1, "builtin:wf2", stages, transitions)
	if err != nil {
		t.Fatalf("CreateWorkflowWithStages: %v", err)
	}

	eng := &Engine{Store: st, Home: ""}
	to, err := eng.transition(ctx, wfID, "start", "done")
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if to != "done" {
		t.Fatalf("transition start+done: got %q, want done", to)
	}
	if !eng.isTerminalStage(ctx, wfID, "done") {
		t.Fatal("isTerminalStage(done): expected true")
	}
	if eng.isTerminalStage(ctx, wfID, "start") {
		t.Fatal("isTerminalStage(start): expected false")
	}
}

func TestEngine_RunTurn_agentStage(t *testing.T) {
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
	_ = st.CreateAgent(ctx, "t1", "a1", "role")
	stages := []store.WorkflowStage{
		{StageName: "start", StageType: "agent", Outcomes: "done", CandidateAgents: ""},
		{StageName: "done", StageType: "terminal", Outcomes: "", CandidateAgents: ""},
	}
	transitions := []store.WorkflowTransition{
		{FromStage: "start", Outcome: "done", ToStage: "done"},
	}
	wfID, err := st.CreateWorkflowWithStages(ctx, "t1", "wf", 1, "builtin:wf", stages, transitions)
	if err != nil {
		t.Fatalf("CreateWorkflowWithStages: %v", err)
	}
	taskID, err := st.CreateTask(ctx, "t1", "agent task", "todo", &wfID)
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	agentName := "a1"
	_ = st.UpdateTask(ctx, taskID, "todo", &agentName)
	task, _ := st.GetTaskByIDAndTeam(ctx, "t1", taskID)
	if task == nil {
		t.Fatal("task nil")
	}

	eng := &Engine{Store: st, Home: ""}
	handled, err := eng.RunTurn(ctx, "t1", task, agentrt.StubRuntime{}, func(agentrt.Event) {})
	if err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if !handled {
		t.Fatal("expected handled=true for agent stage")
	}
	updated, _ := st.GetTaskByIDAndTeam(ctx, "t1", taskID)
	if updated == nil || updated.Status != "done" {
		t.Fatalf("expected task status done after agent->terminal, got %+v", updated)
	}
}

func TestEngine_RunTurn_autoStage(t *testing.T) {
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
	stages := []store.WorkflowStage{
		{StageName: "auto1", StageType: "auto", Outcomes: "done", CandidateAgents: ""},
		{StageName: "done", StageType: "terminal", Outcomes: "", CandidateAgents: ""},
	}
	transitions := []store.WorkflowTransition{
		{FromStage: "auto1", Outcome: "done", ToStage: "done"},
	}
	wfID, err := st.CreateWorkflowWithStages(ctx, "t1", "wfauto", 1, "builtin:wfauto", stages, transitions)
	if err != nil {
		t.Fatalf("CreateWorkflowWithStages: %v", err)
	}
	taskID, err := st.CreateTask(ctx, "t1", "auto task", "todo", &wfID)
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	task, _ := st.GetTaskByIDAndTeam(ctx, "t1", taskID)
	if task == nil {
		t.Fatal("task nil")
	}

	eng := &Engine{Store: st, Home: ""}
	handled, err := eng.RunTurn(ctx, "t1", task, agentrt.StubRuntime{}, func(agentrt.Event) {})
	if err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if !handled {
		t.Fatal("expected handled=true for auto stage")
	}
	updated, _ := st.GetTaskByIDAndTeam(ctx, "t1", taskID)
	if updated == nil || updated.Status != "done" {
		t.Fatalf("expected task status done after auto->terminal, got %+v", updated)
	}
}

func TestEngine_transition_noMatchingOutcome(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), "home")
	st, err := store.Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	_, _ = st.CreateTeam(ctx, "t1")
	stages := []store.WorkflowStage{
		{StageName: "start", StageType: "agent", Outcomes: "done", CandidateAgents: ""},
		{StageName: "done", StageType: "terminal", Outcomes: "", CandidateAgents: ""},
	}
	transitions := []store.WorkflowTransition{
		{FromStage: "start", Outcome: "done", ToStage: "done"},
	}
	wfID, _ := st.CreateWorkflowWithStages(ctx, "t1", "wf", 1, "builtin:wf", stages, transitions)
	eng := &Engine{Store: st, Home: ""}
	to, err := eng.transition(ctx, wfID, "start", "unknown_outcome")
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if to != "" {
		t.Fatalf("transition no match: got %q, want empty", to)
	}
}

func TestEngine_isTerminalStage_edgeCases(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), "home")
	st, err := store.Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	_, _ = st.CreateTeam(ctx, "t1")
	stages := []store.WorkflowStage{
		{StageName: "done", StageType: "terminal", Outcomes: "", CandidateAgents: ""},
	}
	wfID, _ := st.CreateWorkflowWithStages(ctx, "t1", "wf", 1, "builtin:wf", stages, nil)
	eng := &Engine{Store: st, Home: ""}
	if !eng.isTerminalStage(ctx, wfID, "done") {
		t.Fatal("isTerminalStage(done): expected true")
	}
	if eng.isTerminalStage(ctx, wfID, "nonexistent") {
		t.Fatal("isTerminalStage(nonexistent): expected false")
	}
}
