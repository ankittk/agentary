package mcp

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ankittk/agentary/internal/store"
	"github.com/ankittk/agentary/pkg/models"
)

func TestCreateTask(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	st, err := store.Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()

	st.CreateTeam(ctx, "team1")
	tk := &MCPToolkit{Store: st, AgentName: "alice", TeamName: "team1"}

	id, err := tk.CreateTask(ctx, "New task")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected task id > 0, got %d", id)
	}
	task, _ := st.GetTaskByIDAndTeam(ctx, "team1", id)
	if task == nil || task.Title != "New task" || task.Status != models.StatusTodo {
		t.Fatalf("task: %+v", task)
	}
}

func TestSendMessage(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	st, err := store.Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()

	st.CreateTeam(ctx, "team1")
	tk := &MCPToolkit{Store: st, AgentName: "alice", TeamName: "team1"}

	id, err := tk.SendMessage(ctx, "bob", "Hello")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected message id > 0, got %d", id)
	}
	msgs, _ := st.ListMessages(ctx, "team1", "bob", 10)
	if len(msgs) == 0 || msgs[0].Sender != "alice" || msgs[0].Recipient != "bob" {
		t.Fatalf("ListMessages: %+v", msgs)
	}
}

func TestListTasks(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	st, err := store.Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()

	st.CreateTeam(ctx, "team1")
	st.CreateTask(ctx, "team1", "T1", models.StatusTodo, nil)
	st.CreateTask(ctx, "team1", "T2", models.StatusTodo, nil)

	tk := &MCPToolkit{Store: st, AgentName: "alice", TeamName: "team1"}
	tasks, err := tk.ListTasks(ctx, 5)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestListMessages(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	st, err := store.Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()

	st.CreateTeam(ctx, "team1")
	st.CreateMessage(ctx, "team1", "bob", "alice", "Hi")

	tk := &MCPToolkit{Store: st, AgentName: "alice", TeamName: "team1"}
	msgs, err := tk.ListMessages(ctx, "alice", 10)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Recipient != "alice" || msgs[0].Sender != "bob" {
		t.Fatalf("ListMessages: %+v", msgs)
	}
}
