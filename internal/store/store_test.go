package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrationsAndBasicCRUD(t *testing.T) {
	t.Parallel()

	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}

	st, err := Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()

	ctx := context.Background()

	// Network allowlist should exist (migration 002).
	domains, err := st.ListAllowedDomains(ctx)
	if err != nil {
		t.Fatalf("ListAllowedDomains: %v", err)
	}
	if len(domains) == 0 {
		t.Fatalf("expected non-empty allowlist")
	}

	_, err = st.CreateTeam(ctx, "t1")
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	if err := st.CreateAgent(ctx, "t1", "a1", "engineer"); err != nil {
		t.Fatalf("CreateAgent: %v", err)
	}

	if err := st.CreateRepo(ctx, "t1", "r1", "/tmp", "manual", nil); err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}

	if _, err := st.CreateWorkflow(ctx, "t1", "wf", 1, "builtin:wf"); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	id1, err := st.CreateTask(ctx, "t1", "hello", "todo", nil)
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	_, _ = st.CreateTask(ctx, "t1", "world", "todo", nil)

	// NextRunnableTaskForTeam returns oldest runnable task
	next, err := st.NextRunnableTaskForTeam(ctx, "t1")
	if err != nil {
		t.Fatalf("NextRunnableTaskForTeam: %v", err)
	}
	if next == nil || next.TaskID != id1 || next.Status != "todo" {
		t.Fatalf("NextRunnableTaskForTeam: got %+v, want task id=%d status=todo", next, id1)
	}

	if err := st.UpdateTask(ctx, id1, "in_progress", ptr("a1")); err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	next, _ = st.NextRunnableTaskForTeam(ctx, "t1")
	if next == nil {
		t.Fatal("expected a runnable task after update")
	}
	if next.TaskID == id1 && (next.Status != "in_progress" || next.Assignee == nil || *next.Assignee != "a1") {
		t.Fatalf("task id1 should be in_progress with assignee a1, got %+v", next)
	}
	if err := st.UpdateTask(ctx, id1, "done", nil); err != nil {
		t.Fatalf("UpdateTask done: %v", err)
	}
	next, _ = st.NextRunnableTaskForTeam(ctx, "t1")
	if next == nil || next.Title != "world" {
		t.Fatalf("after first task done, next should be 'world', got %+v", next)
	}

	if err := st.DeleteTeam(ctx, "t1"); err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}
}

func TestSetRepoApproval(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()

	_, _ = st.CreateTeam(ctx, "t1")
	if err := st.CreateRepo(ctx, "t1", "r1", "/tmp", "manual", nil); err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}
	if err := st.SetRepoApproval(ctx, "t1", "r1", "auto"); err != nil {
		t.Fatalf("SetRepoApproval: %v", err)
	}
	repos, _ := st.ListRepos(ctx, "t1")
	if len(repos) != 1 || repos[0].Approval != "auto" {
		t.Fatalf("expected repo r1 approval=auto, got %+v", repos)
	}
	if err := st.SetRepoApproval(ctx, "t1", "nonexistent", "auto"); err == nil {
		t.Fatal("expected error for nonexistent repo")
	}
}

func TestMessages(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()

	_, _ = st.CreateTeam(ctx, "t1")
	id, err := st.CreateMessage(ctx, "t1", "alice", "bob", "hello")
	if err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive message_id")
	}
	msgs, err := st.ListMessages(ctx, "t1", "bob", 10)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Sender != "alice" || msgs[0].Recipient != "bob" || msgs[0].Content != "hello" {
		t.Fatalf("ListMessages: got %+v", msgs)
	}
	if err := st.MarkMessageProcessed(ctx, id); err != nil {
		t.Fatalf("MarkMessageProcessed: %v", err)
	}
}

func TestRewindAndClearTaskGitFields(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()

	_, _ = st.CreateTeam(ctx, "t1")
	_ = st.CreateAgent(ctx, "t1", "a1", "engineer")
	wfID, _ := st.CreateWorkflow(ctx, "t1", "default", 1, "builtin:default")
	taskID, err := st.CreateTask(ctx, "t1", "task1", "todo", &wfID)
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	wp := "/tmp/wt"
	bn := "agentary/team1/team1/T1"
	if err := st.UpdateTaskGitFields(ctx, taskID, &wp, &bn, nil, nil); err != nil {
		t.Fatalf("UpdateTaskGitFields: %v", err)
	}
	task, _ := st.GetTaskByIDAndTeam(ctx, "t1", taskID)
	if task == nil || task.WorktreePath == nil || *task.WorktreePath != wp {
		t.Fatalf("expected worktree_path set, got %+v", task)
	}
	if err := st.ClearTaskGitFields(ctx, taskID); err != nil {
		t.Fatalf("ClearTaskGitFields: %v", err)
	}
	task, _ = st.GetTaskByIDAndTeam(ctx, "t1", taskID)
	if task.WorktreePath != nil || task.BranchName != nil {
		t.Fatalf("expected git fields cleared, got %+v", task)
	}
	if err := st.RewindTask(ctx, "t1", taskID); err != nil {
		t.Fatalf("RewindTask: %v", err)
	}
	task, _ = st.GetTaskByIDAndTeam(ctx, "t1", taskID)
	if task.Status != "todo" || task.Assignee != nil {
		t.Fatalf("RewindTask: expected todo and no assignee, got %+v", task)
	}
}

func ptr(s string) *string { return &s }

func TestStoreExtendedCRUD(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()

	_, _ = st.CreateTeam(ctx, "t1")
	_ = st.CreateAgent(ctx, "t1", "a1", "engineer")
	wfID, _ := st.CreateWorkflow(ctx, "t1", "wf1", 1, "builtin:wf1")
	taskID, _ := st.CreateTask(ctx, "t1", "task1", "todo", &wfID)
	taskID2, _ := st.CreateTask(ctx, "t1", "task2", "todo", nil)

	// ListTasks with limit
	tasks, _ := st.ListTasks(ctx, "t1", 1)
	if len(tasks) != 1 {
		t.Fatalf("ListTasks limit 1: got %d", len(tasks))
	}

	// ListTasksInStage (task1 has workflow+stage after CreateWorkflow)
	stages, _ := st.GetWorkflowStages(ctx, wfID)
	if len(stages) > 0 {
		stageName := stages[0].StageName
		_ = st.UpdateTaskStage(ctx, taskID, stageName)
		inStage, _ := st.ListTasksInStage(ctx, "t1", stageName, 10)
		if len(inStage) < 1 {
			t.Fatalf("ListTasksInStage: expected at least 1, got %d", len(inStage))
		}
	}

	// SetTaskFailed, RequeueTask
	if err := st.SetTaskFailed(ctx, taskID); err != nil {
		t.Fatalf("SetTaskFailed: %v", err)
	}
	task, _ := st.GetTaskByIDAndTeam(ctx, "t1", taskID)
	if task == nil || task.Status != "failed" {
		t.Fatalf("SetTaskFailed: got %+v", task)
	}
	if err := st.RequeueTask(ctx, "t1", taskID); err != nil {
		t.Fatalf("RequeueTask: %v", err)
	}
	task, _ = st.GetTaskByIDAndTeam(ctx, "t1", taskID)
	if task == nil || task.Status != "todo" {
		t.Fatalf("RequeueTask: got %+v", task)
	}

	// SetTaskCancelled
	if err := st.SetTaskCancelled(ctx, "t1", taskID2); err != nil {
		t.Fatalf("SetTaskCancelled: %v", err)
	}

	// CreateTaskComment, ListTaskComments
	cid, err := st.CreateTaskComment(ctx, "t1", taskID, "alice", "a comment")
	if err != nil {
		t.Fatalf("CreateTaskComment: %v", err)
	}
	if cid <= 0 {
		t.Fatal("expected positive comment_id")
	}
	comments, _ := st.ListTaskComments(ctx, "t1", taskID)
	if len(comments) != 1 || comments[0].Body != "a comment" {
		t.Fatalf("ListTaskComments: got %+v", comments)
	}

	// AddTaskAttachment, ListTaskAttachments, RemoveTaskAttachment
	if err := st.AddTaskAttachment(ctx, "t1", taskID, "/path/to/file"); err != nil {
		t.Fatalf("AddTaskAttachment: %v", err)
	}
	attachments, _ := st.ListTaskAttachments(ctx, "t1", taskID)
	if len(attachments) != 1 {
		t.Fatalf("ListTaskAttachments: got %d", len(attachments))
	}
	if err := st.RemoveTaskAttachment(ctx, "t1", taskID, "/path/to/file"); err != nil {
		t.Fatalf("RemoveTaskAttachment: %v", err)
	}
	attachments, _ = st.ListTaskAttachments(ctx, "t1", taskID)
	if len(attachments) != 0 {
		t.Fatalf("after Remove: got %d", len(attachments))
	}

	// AddTaskDependency, ListTaskDependencies
	if err := st.AddTaskDependency(ctx, "t1", taskID, taskID2); err != nil {
		t.Fatalf("AddTaskDependency: %v", err)
	}
	deps, _ := st.ListTaskDependencies(ctx, "t1", taskID)
	if len(deps) != 1 || deps[0] != taskID2 {
		t.Fatalf("ListTaskDependencies: got %v", deps)
	}

	// CreateTaskReview, ListTaskReviews
	rid, err := st.CreateTaskReview(ctx, "t1", taskID, "a1", "approved", "looks good")
	if err != nil {
		t.Fatalf("CreateTaskReview: %v", err)
	}
	if rid <= 0 {
		t.Fatal("expected positive review_id")
	}
	reviews, _ := st.ListTaskReviews(ctx, "t1", taskID)
	if len(reviews) != 1 || reviews[0].Outcome != "approved" {
		t.Fatalf("ListTaskReviews: got %+v", reviews)
	}

	// ListWorkflows, GetWorkflowIDByTeamAndName, GetWorkflowStages, GetWorkflowTransitions, GetWorkflowInitialStage
	workflows, _ := st.ListWorkflows(ctx, "t1")
	if len(workflows) < 1 {
		t.Fatalf("ListWorkflows: got %d", len(workflows))
	}
	wid, _ := st.GetWorkflowIDByTeamAndName(ctx, "t1", "wf1", 1)
	if wid != wfID {
		t.Fatalf("GetWorkflowIDByTeamAndName: got %q", wid)
	}
	_, _ = st.GetWorkflowStages(ctx, wfID)
	_, _ = st.GetWorkflowTransitions(ctx, wfID)
	initial, errInit := st.GetWorkflowInitialStage(ctx, wfID)
	if errInit != nil && len(stages) > 0 {
		t.Fatalf("GetWorkflowInitialStage: %v", errInit)
	}
	_ = initial

	// AllowDomain, DisallowDomain, ResetAllowlist
	_ = st.AllowDomain(ctx, "example.com")
	domains, _ := st.ListAllowedDomains(ctx)
	found := false
	for _, d := range domains {
		if d == "example.com" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("AllowDomain: example.com not in list")
	}
	_ = st.DisallowDomain(ctx, "example.com")
	domains, _ = st.ListAllowedDomains(ctx)
	for _, d := range domains {
		if d == "example.com" {
			t.Fatal("DisallowDomain: example.com still in list")
		}
	}
	_ = st.ResetAllowlist(ctx)

	// ListUnprocessedMessages
	_, _ = st.CreateMessage(ctx, "t1", "x", "bob", "unread")
	unproc, _ := st.ListUnprocessedMessages(ctx, "t1", "bob", 10)
	if len(unproc) < 1 {
		t.Fatalf("ListUnprocessedMessages: got %d", len(unproc))
	}
}

func TestStoreErrorPaths(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()

	_, err = st.CreateTeam(ctx, "")
	if err == nil {
		t.Fatal("CreateTeam empty name: expected error")
	}
	_, _ = st.CreateTeam(ctx, "t1")
	if err := st.CreateAgent(ctx, "t1", "", "role"); err == nil {
		t.Fatal("CreateAgent empty name: expected error")
	}
	_, err = st.CreateTask(ctx, "t1", "", "todo", nil)
	if err == nil {
		t.Fatal("CreateTask empty title: expected error")
	}
	_, err = st.GetTeamByName(ctx, "nonexistent")
	if err == nil {
		t.Fatal("GetTeamByName nonexistent: expected error")
	}
	task, _ := st.GetTaskByIDAndTeam(ctx, "t1", 99999)
	if task != nil {
		t.Fatalf("GetTaskByIDAndTeam nonexistent: expected nil, got %+v", task)
	}
}

func TestOpenWithOptions(t *testing.T) {
	t.Parallel()
	_, err := OpenWithOptions(OpenOptions{Driver: "postgres"})
	if err == nil {
		t.Fatal("OpenWithOptions postgres: expected error")
	}
	dir := t.TempDir()
	st, err := OpenWithOptions(OpenOptions{Driver: "sqlite", Home: dir})
	if err != nil {
		t.Fatalf("OpenWithOptions sqlite: %v", err)
	}
	_ = st.Close()
	// DSN path
	st2, err := OpenWithOptions(OpenOptions{Driver: "sqlite", Home: "", DSN: "file:" + filepath.Join(dir, "protected", "db.sqlite")})
	if err != nil {
		t.Fatalf("OpenWithOptions DSN: %v", err)
	}
	_ = st2.Close()
}

func TestUpdateTask_assigneeOnly(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	_, _ = st.CreateTeam(ctx, "t1")
	taskID, _ := st.CreateTask(ctx, "t1", "task", "todo", nil)
	if err := st.UpdateTask(ctx, taskID, "", ptr("alice")); err != nil {
		t.Fatalf("UpdateTask assignee only: %v", err)
	}
	task, _ := st.GetTaskByIDAndTeam(ctx, "t1", taskID)
	if task == nil || task.Assignee == nil || *task.Assignee != "alice" {
		t.Fatalf("UpdateTask assignee only: got %+v", task)
	}
}

func TestSeedDemo(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), "home")
	st, err := Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	if err := st.SeedDemo(ctx); err != nil {
		t.Fatalf("SeedDemo: %v", err)
	}
	teams, _ := st.ListTeams(ctx)
	if len(teams) == 0 {
		t.Fatal("SeedDemo: expected teams")
	}
}

func TestSeedDemo_idempotency(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), "home")
	st, err := Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	if err := st.SeedDemo(ctx); err != nil {
		t.Fatalf("SeedDemo first: %v", err)
	}
	if err := st.SeedDemo(ctx); err != nil {
		t.Fatalf("SeedDemo second (idempotent): %v", err)
	}
	teams, _ := st.ListTeams(ctx)
	if len(teams) == 0 {
		t.Fatal("SeedDemo: expected teams")
	}
}

func TestListTasksInStage(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), "home")
	st, err := Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	_, _ = st.CreateTeam(ctx, "t1")
	stages := []WorkflowStage{
		{StageName: "InProgress", StageType: "agent", Outcomes: "done", CandidateAgents: ""},
		{StageName: "Merging", StageType: "merge", Outcomes: "", CandidateAgents: ""},
	}
	transitions := []WorkflowTransition{
		{FromStage: "InProgress", Outcome: "done", ToStage: "Merging"},
	}
	wfID, _ := st.CreateWorkflowWithStages(ctx, "t1", "wf", 1, "builtin:wf", stages, transitions)
	taskID, _ := st.CreateTask(ctx, "t1", "task", "todo", &wfID)
	_ = st.SetTaskWorkflowAndStage(ctx, taskID, wfID, "Merging")
	inStage, err := st.ListTasksInStage(ctx, "t1", "Merging", 10)
	if err != nil {
		t.Fatalf("ListTasksInStage: %v", err)
	}
	if len(inStage) != 1 || inStage[0].TaskID != taskID {
		t.Fatalf("ListTasksInStage Merging: got %+v", inStage)
	}
	empty, _ := st.ListTasksInStage(ctx, "t1", "Nonexistent", 10)
	if len(empty) != 0 {
		t.Fatalf("ListTasksInStage Nonexistent: got %d", len(empty))
	}
}

func TestConcurrentCreateTask(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), "home")
	st, err := Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	_, _ = st.CreateTeam(ctx, "t1")
	const n = 20
	done := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(j int) {
			_, err := st.CreateTask(ctx, "t1", fmt.Sprintf("task %d", j), "todo", nil)
			done <- err
		}(i)
	}
	for i := 0; i < n; i++ {
		if err := <-done; err != nil {
			t.Fatalf("concurrent CreateTask: %v", err)
		}
	}
	tasks, _ := st.ListTasks(ctx, "t1", 100)
	if len(tasks) != n {
		t.Fatalf("expected %d tasks, got %d", n, len(tasks))
	}
}

func BenchmarkCreateTaskAndListTasks(b *testing.B) {
	home := filepath.Join(b.TempDir(), "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		b.Fatal(err)
	}
	st, err := Open(home)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	_, _ = st.CreateTeam(ctx, "t1")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = st.CreateTask(ctx, "t1", "bench task", "todo", nil)
		_, _ = st.ListTasks(ctx, "t1", 100)
	}
}

func BenchmarkGetTaskByIDAndTeam(b *testing.B) {
	home := filepath.Join(b.TempDir(), "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		b.Fatal(err)
	}
	st, err := Open(home)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	_, _ = st.CreateTeam(ctx, "t1")
	id, _ := st.CreateTask(ctx, "t1", "task", "todo", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = st.GetTaskByIDAndTeam(ctx, "t1", id)
	}
}

func BenchmarkListTasks(b *testing.B) {
	home := filepath.Join(b.TempDir(), "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		b.Fatal(err)
	}
	st, err := Open(home)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	_, _ = st.CreateTeam(ctx, "t1")
	for i := 0; i < 50; i++ {
		_, _ = st.CreateTask(ctx, "t1", "task", "todo", nil)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = st.ListTasks(ctx, "t1", 100)
	}
}

func BenchmarkNextRunnableTaskForTeam(b *testing.B) {
	home := filepath.Join(b.TempDir(), "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		b.Fatal(err)
	}
	st, err := Open(home)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	_, _ = st.CreateTeam(ctx, "t1")
	_, _ = st.CreateTask(ctx, "t1", "task", "todo", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = st.NextRunnableTaskForTeam(ctx, "t1")
	}
}
