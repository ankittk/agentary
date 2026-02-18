package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBranchName(t *testing.T) {
	got := BranchName("tid-1", "team one", 42)
	if got != "agentary/tid-1/team-one/T42" {
		t.Errorf("BranchName: got %q", got)
	}
}

func TestWorktreePath(t *testing.T) {
	got := WorktreePath("/home", "team one", "repo one", 1)
	want := filepath.Join("/home", "protected", "teams", "team_one", "worktrees", "repo_one-T1")
	if got != want {
		t.Errorf("WorktreePath: got %q, want %q", got, want)
	}
}

func TestDeleteWorktree_emptyAndMissing(t *testing.T) {
	ctx := context.Background()
	if err := DeleteWorktree(ctx, ""); err != nil {
		t.Errorf("DeleteWorktree empty: %v", err)
	}
	if err := DeleteWorktree(ctx, filepath.Join(t.TempDir(), "nonexistent")); err != nil {
		t.Errorf("DeleteWorktree missing: %v", err)
	}
}

func TestDiff_emptyPath(t *testing.T) {
	ctx := context.Background()
	out, err := Diff(ctx, "", "HEAD~1", "HEAD")
	if err != nil {
		t.Errorf("Diff empty path: %v", err)
	}
	if out != "" {
		t.Errorf("Diff empty path: got %q", out)
	}
}

func TestMergeInWorktree_empty(t *testing.T) {
	ctx := context.Background()
	if err := MergeInWorktree(ctx, "", "branch"); err != nil {
		t.Errorf("MergeInWorktree empty path: %v", err)
	}
	if err := MergeInWorktree(ctx, "/path", ""); err != nil {
		t.Errorf("MergeInWorktree empty branch: %v", err)
	}
}

func TestRunTestCmd_empty(t *testing.T) {
	ctx := context.Background()
	if err := RunTestCmd(ctx, "", "echo 1"); err != nil {
		t.Errorf("RunTestCmd empty path: %v", err)
	}
	if err := RunTestCmd(ctx, t.TempDir(), ""); err != nil {
		t.Errorf("RunTestCmd empty cmd: %v", err)
	}
}

func TestRebaseOntoMain_empty(t *testing.T) {
	ctx := context.Background()
	if err := RebaseOntoMain(ctx, "", "branch"); err != nil {
		t.Errorf("RebaseOntoMain empty path: %v", err)
	}
	if err := RebaseOntoMain(ctx, "/path", ""); err != nil {
		t.Errorf("RebaseOntoMain empty branch: %v", err)
	}
}

func TestCreateWorktree_validation(t *testing.T) {
	ctx := context.Background()
	_, err := CreateWorktree(ctx, "", "http://x", "branch")
	if err == nil {
		t.Fatal("CreateWorktree empty path: expected error")
	}
	_, err = CreateWorktree(ctx, t.TempDir(), "", "branch")
	if err == nil {
		t.Fatal("CreateWorktree empty sourceURL: expected error")
	}
}

func TestDeleteWorktree_existingDir(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "wt")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := DeleteWorktree(ctx, dir); err != nil {
		t.Errorf("DeleteWorktree: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("DeleteWorktree: dir should be removed")
	}
}
