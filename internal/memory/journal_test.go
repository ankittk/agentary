package memory

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestJournal_AppendAndRead(t *testing.T) {
	dir := t.TempDir()
	teamDir := filepath.Join(dir, "teams", "acme")
	agentDir := AgentDir(teamDir, "alice")
	_ = os.MkdirAll(agentDir, 0o755)

	j := &Journal{AgentName: "alice", TeamDir: teamDir}
	ctx := context.Background()

	ts, _ := time.Parse(time.RFC3339, "2025-01-15T10:00:00Z")
	err := j.Append(ctx, JournalEntry{
		TaskID:    1,
		TaskTitle: "Add feature",
		Outcome:   "done",
		CreatedAt: ts,
	})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	content, err := j.Read(ctx, 0)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if content == "" || len(content) < 20 {
		t.Fatalf("Read: expected non-empty content, got %q", content)
	}
	if !strings.Contains(content, "Add feature") || !strings.Contains(content, "done") {
		t.Fatalf("Read: expected task title and outcome in content, got %q", content)
	}

	sum, err := j.Summary(ctx, 500)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if sum == "" || sum == "(no journal entries yet)" {
		t.Fatalf("Summary: expected content, got %q", sum)
	}
}

func TestJournal_Append_createsDirectory(t *testing.T) {
	dir := t.TempDir()
	teamDir := filepath.Join(dir, "teams", "newteam")
	// Agent dir does not exist yet; Append should create it.
	j := &Journal{AgentName: "bob", TeamDir: teamDir}
	ctx := context.Background()
	err := j.Append(ctx, JournalEntry{TaskID: 1, TaskTitle: "T1", Outcome: "ok", CreatedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	agentDir := AgentDir(teamDir, "bob")
	if _, err := os.Stat(agentDir); os.IsNotExist(err) {
		t.Fatalf("Append should create agent dir %q", agentDir)
	}
}

func TestJournal_Read_limitBytes(t *testing.T) {
	dir := t.TempDir()
	teamDir := filepath.Join(dir, "teams", "t1")
	agentDir := AgentDir(teamDir, "alice")
	_ = os.MkdirAll(agentDir, 0o755)
	j := &Journal{AgentName: "alice", TeamDir: teamDir}
	ctx := context.Background()
	_ = j.Append(ctx, JournalEntry{TaskID: 1, TaskTitle: "Long title here", Outcome: "done", CreatedAt: time.Now().UTC()})
	content, err := j.Read(ctx, 20)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(content) > 20 {
		t.Fatalf("Read limitBytes=20: got len %d", len(content))
	}
}

func TestJournal_Summary_emptyJournal(t *testing.T) {
	dir := t.TempDir()
	teamDir := filepath.Join(dir, "teams", "empty")
	j := &Journal{AgentName: "nobody", TeamDir: teamDir}
	ctx := context.Background()
	sum, err := j.Summary(ctx, 500)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if sum != "(no journal entries yet)" {
		t.Fatalf("Summary empty: got %q", sum)
	}
}

func TestEnsureTeamDirs(t *testing.T) {
	dir := t.TempDir()
	teamDir := filepath.Join(dir, "teams", "acme")
	if err := EnsureTeamDirs(teamDir); err != nil {
		t.Fatalf("EnsureTeamDirs: %v", err)
	}
	shared := filepath.Join(teamDir, "shared")
	if _, err := os.Stat(shared); os.IsNotExist(err) {
		t.Fatalf("EnsureTeamDirs should create shared/ %q", shared)
	}
}
