package cli

import (
	"fmt"

	"github.com/ankittk/agentary/internal/config"
	"github.com/ankittk/agentary/internal/git"
	"github.com/ankittk/agentary/internal/store"
	"github.com/spf13/cobra"
)

func newTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks",
	}
	cmd.AddCommand(newTaskRequeueCmd())
	cmd.AddCommand(newTaskAssignCmd())
	cmd.AddCommand(newTaskStatusCmd())
	cmd.AddCommand(newTaskCancelCmd())
	cmd.AddCommand(newTaskRetryCmd())
	cmd.AddCommand(newTaskCompleteCmd())
	cmd.AddCommand(newTaskForceTransitionCmd())
	cmd.AddCommand(newTaskRewindCmd())
	return cmd
}

func newTaskRequeueCmd() *cobra.Command {
	var team string
	var taskID int64

	cmd := &cobra.Command{
		Use:   "requeue",
		Short: "Requeue a task (set status to todo, clear assignee)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if team == "" {
				return fmt.Errorf("--team is required")
			}
			if taskID <= 0 {
				return fmt.Errorf("--id must be a positive task ID")
			}
			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			if err := st.RequeueTask(cmd.Context(), team, taskID); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Requeued task %d in team %q\n", taskID, team)
			return nil
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "Team name")
	cmd.Flags().Int64Var(&taskID, "id", 0, "Task ID")
	_ = cmd.MarkFlagRequired("team")
	return cmd
}

func newTaskAssignCmd() *cobra.Command {
	var team string
	var taskID int64
	var assignee string

	cmd := &cobra.Command{
		Use:   "assign",
		Short: "Assign a task to an agent or human",
		RunE: func(cmd *cobra.Command, args []string) error {
			if team == "" || assignee == "" || taskID <= 0 {
				return fmt.Errorf("--team, --id, and --assignee are required")
			}
			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			if err := st.UpdateTask(cmd.Context(), taskID, "", &assignee); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Assigned task %d to %q\n", taskID, assignee)
			return nil
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "Team name")
	cmd.Flags().Int64Var(&taskID, "id", 0, "Task ID")
	cmd.Flags().StringVar(&assignee, "assignee", "", "Assignee name")
	return cmd
}

func newTaskStatusCmd() *cobra.Command {
	var team string
	var taskID int64
	var status string
	var assignee string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Set task status (and optionally assignee)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if team == "" || taskID <= 0 || status == "" {
				return fmt.Errorf("--team, --id, and --status are required")
			}
			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			var a *string
			if assignee != "" {
				a = &assignee
			}
			if err := st.UpdateTask(cmd.Context(), taskID, status, a); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Task %d status set to %q\n", taskID, status)
			return nil
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "Team name")
	cmd.Flags().Int64Var(&taskID, "id", 0, "Task ID")
	cmd.Flags().StringVar(&status, "status", "", "New status (todo, in_progress, done, failed, cancelled)")
	cmd.Flags().StringVar(&assignee, "assignee", "", "Optional assignee")
	return cmd
}

func newTaskCancelCmd() *cobra.Command {
	var team string
	var taskID int64

	cmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a task (terminal status); cleans up git worktree if present",
		RunE: func(cmd *cobra.Command, args []string) error {
			if team == "" || taskID <= 0 {
				return fmt.Errorf("--team and --id are required")
			}
			ctx := cmd.Context()
			home := config.MustHomeFrom(ctx)
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			task, err := st.GetTaskByIDAndTeam(ctx, team, taskID)
			if err != nil {
				return err
			}
			if task == nil {
				return fmt.Errorf("task %d not found in team %q", taskID, team)
			}
			if task.WorktreePath != nil && *task.WorktreePath != "" {
				_ = git.DeleteWorktree(ctx, *task.WorktreePath)
				_ = st.ClearTaskGitFields(ctx, taskID)
			}
			if err := st.SetTaskCancelled(ctx, team, taskID); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cancelled task %d\n", taskID)
			return nil
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "Team name")
	cmd.Flags().Int64Var(&taskID, "id", 0, "Task ID")
	return cmd
}

func newTaskRetryCmd() *cobra.Command {
	var team string
	var taskID int64

	cmd := &cobra.Command{
		Use:   "retry",
		Short: "Retry a task (requeue: set status to todo, clear assignee)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if team == "" || taskID <= 0 {
				return fmt.Errorf("--team and --id are required")
			}
			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			if err := st.RequeueTask(cmd.Context(), team, taskID); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Retried (requeued) task %d\n", taskID)
			return nil
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "Team name")
	cmd.Flags().Int64Var(&taskID, "id", 0, "Task ID")
	return cmd
}

func newTaskCompleteCmd() *cobra.Command {
	var team string
	var taskID int64

	cmd := &cobra.Command{
		Use:   "complete",
		Short: "Mark a task as done",
		RunE: func(cmd *cobra.Command, args []string) error {
			if team == "" || taskID <= 0 {
				return fmt.Errorf("--team and --id are required")
			}
			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			if err := st.UpdateTask(cmd.Context(), taskID, "done", nil); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Task %d marked done\n", taskID)
			return nil
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "Team name")
	cmd.Flags().Int64Var(&taskID, "id", 0, "Task ID")
	return cmd
}

func newTaskForceTransitionCmd() *cobra.Command {
	var team string
	var taskID int64
	var stage string

	cmd := &cobra.Command{
		Use:   "force-transition",
		Short: "Force task to a workflow stage (bypass normal transitions)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if team == "" || taskID <= 0 || stage == "" {
				return fmt.Errorf("--team, --id, and --stage are required")
			}
			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			if err := st.UpdateTaskStage(cmd.Context(), taskID, stage); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Task %d stage set to %q\n", taskID, stage)
			return nil
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "Team name")
	cmd.Flags().Int64Var(&taskID, "id", 0, "Task ID")
	cmd.Flags().StringVar(&stage, "stage", "", "Stage name")
	return cmd
}

func newTaskRewindCmd() *cobra.Command {
	var team string
	var taskID int64

	cmd := &cobra.Command{
		Use:   "rewind",
		Short: "Rewind task to initial stage (todo, clear assignee, reset stage)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if team == "" || taskID <= 0 {
				return fmt.Errorf("--team and --id are required")
			}
			home := config.MustHomeFrom(cmd.Context())
			st, err := store.Open(home)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()

			if err := st.RewindTask(cmd.Context(), team, taskID); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Rewound task %d\n", taskID)
			return nil
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "Team name")
	cmd.Flags().Int64Var(&taskID, "id", 0, "Task ID")
	return cmd
}
