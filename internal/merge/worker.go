package merge

import (
	"context"
	"log/slog"
	"time"

	"github.com/ankittk/agentary/internal/git"
	"github.com/ankittk/agentary/internal/store"
	"github.com/ankittk/agentary/pkg/models"
)

// Worker polls for tasks in "Merging" stage, rebases onto main, runs test_cmd, merges, marks done, and cleans worktree.
type Worker struct {
	Store store.Store
	// Interval between poll rounds
	Interval time.Duration
	// RebaseBeforeMerge runs rebase onto origin/main before merge when true
	RebaseBeforeMerge bool
}

const defaultMergeInterval = 15 * time.Second

// Run runs the merge worker until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	interval := w.Interval
	if interval <= 0 {
		interval = defaultMergeInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *Worker) runOnce(ctx context.Context) {
	teams, err := w.Store.ListTeams(ctx)
	if err != nil {
		slog.Error("merge worker list teams failed", "err", err)
		return
	}
	for _, t := range teams {
		tasks, err := w.Store.ListTasksInStage(ctx, t.Name, "Merging", 20)
		if err != nil {
			slog.Error("merge worker list tasks in stage failed", "team", t.Name, "err", err)
			continue
		}
		for _, task := range tasks {
			w.processTask(ctx, t.Name, &task)
		}
	}
}

func (w *Worker) processTask(ctx context.Context, teamName string, task *store.Task) {
	worktreePath := ""
	if task.WorktreePath != nil {
		worktreePath = *task.WorktreePath
	}
	branchName := ""
	if task.BranchName != nil {
		branchName = *task.BranchName
	}
	wfID := ""
	if task.WorkflowID != nil {
		wfID = *task.WorkflowID
	}
	if wfID == "" {
		return
	}

	if worktreePath != "" && branchName != "" && w.RebaseBeforeMerge {
		if err := git.RebaseOntoMain(ctx, worktreePath, branchName); err != nil {
			slog.Error("merge worker rebase failed", "task_id", task.TaskID, "err", err)
			_ = w.Store.SetTaskFailed(ctx, task.TaskID)
			return
		}
	}

	if worktreePath != "" {
		repos, _ := w.Store.ListRepos(ctx, teamName)
		var repo *store.Repo
		for i := range repos {
			if task.RepoName != nil && repos[i].Name == *task.RepoName {
				repo = &repos[i]
				break
			}
		}
		if repo == nil && len(repos) > 0 {
			repo = &repos[0]
		}
		if repo != nil && repo.TestCmd != nil && *repo.TestCmd != "" {
			if err := git.RunTestCmd(ctx, worktreePath, *repo.TestCmd); err != nil {
				slog.Error("merge worker test failed", "task_id", task.TaskID, "err", err)
				_ = w.Store.SetTaskFailed(ctx, task.TaskID)
				return
			}
		}

		if branchName != "" {
			if err := git.MergeInWorktree(ctx, worktreePath, branchName); err != nil {
				slog.Error("merge worker merge failed", "task_id", task.TaskID, "err", err)
				_ = w.Store.SetTaskFailed(ctx, task.TaskID)
				return
			}
		}
	}

	_ = w.Store.SetTaskWorkflowAndStage(ctx, task.TaskID, wfID, "Done")
	_ = w.Store.UpdateTask(ctx, task.TaskID, models.StatusDone, nil)
	_ = w.Store.ClearTaskGitFields(ctx, task.TaskID)
	if worktreePath != "" {
		_ = git.DeleteWorktree(ctx, worktreePath)
	}
	slog.Info("merge worker completed task", "task_id", task.TaskID, "team", teamName)
}
