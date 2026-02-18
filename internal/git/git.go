package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BranchName returns the agentary-style branch name for a task: agentary/<team_id>/<team>/T<NNNN>.
// teamID is the internal team UUID; teamName is the display name; taskID is the task number.
func BranchName(teamID, teamName string, taskID int64) string {
	safe := strings.ReplaceAll(teamName, " ", "-")
	return fmt.Sprintf("agentary/%s/%s/T%d", teamID, safe, taskID)
}

// WorktreePath returns the path for a task worktree under home: <home>/protected/teams/<team>/worktrees/<repo>-T<id>.
func WorktreePath(home, teamName, repoName string, taskID int64) string {
	safeTeam := strings.ReplaceAll(teamName, " ", "_")
	safeRepo := strings.ReplaceAll(repoName, " ", "_")
	return filepath.Join(home, "protected", "teams", safeTeam, "worktrees", fmt.Sprintf("%s-T%d", safeRepo, taskID))
}

// CreateWorktree creates a worktree for the task: clones sourceURL into worktreePath and checks out branch branchName (creating it from main or HEAD).
// Returns baseSHA (commit at branch creation). If the directory already exists, returns the current HEAD there without re-cloning.
func CreateWorktree(ctx context.Context, worktreePath, sourceURL, branchName string) (baseSHA string, err error) {
	if worktreePath == "" || sourceURL == "" || branchName == "" {
		return "", fmt.Errorf("worktree_path, source_url, and branch_name required")
	}
	dir := filepath.Dir(worktreePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	// If worktree already exists, just return HEAD (idempotent).
	if _, err := os.Stat(worktreePath); err == nil {
		cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
		cmd.Dir = worktreePath
		out, runErr := cmd.Output()
		if runErr != nil {
			return "", runErr
		}
		return strings.TrimSpace(string(out)), nil
	}

	// Clone (shallow to save space).
	cloneCmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", sourceURL, worktreePath)
	if out, err := cloneCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git clone: %w: %s", err, string(out))
	}

	// Create and checkout branch.
	checkoutCmd := exec.CommandContext(ctx, "git", "checkout", "-b", branchName)
	checkoutCmd.Dir = worktreePath
	if out, err := checkoutCmd.CombinedOutput(); err != nil {
		_ = os.RemoveAll(worktreePath)
		return "", fmt.Errorf("git checkout -b: %w: %s", err, string(out))
	}

	revCmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	revCmd.Dir = worktreePath
	out, err := revCmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// DeleteWorktree removes the worktree directory and optionally deletes the branch in the main repo.
// If worktreePath is empty or the path doesn't exist, no-op (returns nil).
func DeleteWorktree(ctx context.Context, worktreePath string) error {
	if worktreePath == "" {
		return nil
	}
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(worktreePath)
}

// RebaseOntoMain checks out branchName, fetches origin, and rebases onto origin/main (or origin/master).
// No-op if worktreePath or branchName is empty.
func RebaseOntoMain(ctx context.Context, worktreePath, branchName string) error {
	if worktreePath == "" || branchName == "" {
		return nil
	}
	co := exec.CommandContext(ctx, "git", "checkout", branchName)
	co.Dir = worktreePath
	if out, err := co.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout %s: %w: %s", branchName, err, string(out))
	}
	fetch := exec.CommandContext(ctx, "git", "fetch", "origin")
	fetch.Dir = worktreePath
	if out, err := fetch.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch origin: %w: %s", err, string(out))
	}
	rebase := exec.CommandContext(ctx, "git", "rebase", "origin/main")
	rebase.Dir = worktreePath
	if out, err := rebase.CombinedOutput(); err != nil {
		rebase = exec.CommandContext(ctx, "git", "rebase", "origin/master")
		rebase.Dir = worktreePath
		if out2, err2 := rebase.CombinedOutput(); err2 != nil {
			return fmt.Errorf("git rebase origin/main: %w: %s", err2, string(out2))
		}
		_ = out
	} else {
		_ = out
	}
	return nil
}

// MergeInWorktree runs git checkout main (or master) and git merge branchName in the worktree.
// Used by the merge workflow stage to merge the task branch into main.
func MergeInWorktree(ctx context.Context, worktreePath, branchName string) error {
	if worktreePath == "" || branchName == "" {
		return nil
	}
	// Prefer main, fallback to master.
	checkout := exec.CommandContext(ctx, "git", "checkout", "main")
	checkout.Dir = worktreePath
	if out, err := checkout.CombinedOutput(); err != nil {
		checkout = exec.CommandContext(ctx, "git", "checkout", "master")
		checkout.Dir = worktreePath
		if out2, err2 := checkout.CombinedOutput(); err2 != nil {
			return fmt.Errorf("git checkout main/master: %w: %s", err2, string(out2))
		}
		_ = out
	} else {
		_ = out
	}
	mergeCmd := exec.CommandContext(ctx, "git", "merge", branchName)
	mergeCmd.Dir = worktreePath
	if out, err := mergeCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git merge %s: %w: %s", branchName, err, string(out))
	}
	return nil
}

// RunTestCmd runs testCmd (e.g. from repo.test_cmd) in worktreePath. Uses sh -c for shell semantics.
func RunTestCmd(ctx context.Context, worktreePath, testCmd string) error {
	if worktreePath == "" || testCmd == "" {
		return nil
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", testCmd)
	cmd.Dir = worktreePath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("test_cmd: %w: %s", err, string(out))
	}
	return nil
}

// Diff returns git diff baseSHA..headRef in worktreePath (for review UI). headRef is typically "HEAD" or branch name.
func Diff(ctx context.Context, worktreePath, baseSHA, headRef string) (string, error) {
	if worktreePath == "" {
		return "", nil
	}
	if headRef == "" {
		headRef = "HEAD"
	}
	if baseSHA == "" {
		baseSHA = "HEAD~1" // fallback
	}
	cmd := exec.CommandContext(ctx, "git", "diff", baseSHA+".."+headRef)
	cmd.Dir = worktreePath
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}
	return string(out), nil
}
