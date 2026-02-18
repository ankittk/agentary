package sandbox

import (
	"path/filepath"
	"testing"
)

func TestWriteGuard_Manager(t *testing.T) {
	teamDir := filepath.Join(t.TempDir(), "teams", "acme")
	guard := &WriteGuard{Role: "manager", TeamDir: teamDir}
	// Manager can write anywhere under team dir
	if !guard.AllowWrite(teamDir) {
		t.Error("manager should allow team dir")
	}
	if !guard.AllowWrite(filepath.Join(teamDir, "agents", "alice", "notes", "x.md")) {
		t.Error("manager should allow any path under team dir")
	}
	if !guard.AllowWrite(filepath.Join(teamDir, "shared", "conv.md")) {
		t.Error("manager should allow shared")
	}
	// Path outside team dir is denied
	if guard.AllowWrite(filepath.Join(filepath.Dir(teamDir), "protected", "db.sqlite")) {
		t.Error("manager should not allow path outside team dir")
	}
}

func TestWriteGuard_Engineer(t *testing.T) {
	base := t.TempDir()
	teamDir := filepath.Join(base, "teams", "acme")
	worktree := filepath.Join(teamDir, "worktrees", "repo-T1")
	guard := &WriteGuard{
		Role:         "engineer",
		AgentName:    "alice",
		TeamDir:      teamDir,
		WorktreeDirs: []string{worktree},
	}
	// Own agent dir
	if !guard.AllowWrite(filepath.Join(teamDir, "agents", "alice", "journal.md")) {
		t.Error("engineer should allow own agent dir")
	}
	if !guard.AllowWrite(filepath.Join(teamDir, "agents", "alice", "notes", "x")) {
		t.Error("engineer should allow own notes")
	}
	// Shared
	if !guard.AllowWrite(filepath.Join(teamDir, "shared", "conv.md")) {
		t.Error("engineer should allow team shared")
	}
	// Worktree
	if !guard.AllowWrite(filepath.Join(worktree, "src", "main.go")) {
		t.Error("engineer should allow worktree path")
	}
	// Denied: other agent's dir
	if guard.AllowWrite(filepath.Join(teamDir, "agents", "bob", "journal.md")) {
		t.Error("engineer should not allow other agent dir")
	}
	// Denied: protected
	if guard.AllowWrite(filepath.Join(base, "protected", "db.sqlite")) {
		t.Error("engineer should not allow protected")
	}
}
