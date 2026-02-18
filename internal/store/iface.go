package store

import "context"

// Store is the persistence interface for teams, tasks, workflows, messages, and network allowlist.
// Implementations: *sqlite.Store (SQLite) and *postgres.Store (PostgreSQL).
type Store interface {
	// Teams
	ListTeams(ctx context.Context) ([]Team, error)
	GetTeamByName(ctx context.Context, name string) (Team, error)
	CreateTeam(ctx context.Context, name string) (Team, error)
	DeleteTeam(ctx context.Context, name string) error

	// Agents
	ListAgents(ctx context.Context, teamName string) ([]Agent, error)
	CreateAgent(ctx context.Context, teamName, name, role string) error

	// Tasks
	ListTasks(ctx context.Context, teamName string, limit int) ([]Task, error)
	ListTasksInStage(ctx context.Context, teamName, stage string, limit int) ([]Task, error)
	CreateTask(ctx context.Context, teamName, title, status string, workflowID *string) (int64, error)
	UpdateTask(ctx context.Context, taskID int64, status string, assignee *string) error
	ClaimTask(ctx context.Context, teamName string, taskID int64, assignee string) (bool, error)
	SetTaskFailed(ctx context.Context, taskID int64) error
	RequeueTask(ctx context.Context, teamName string, taskID int64) error
	SetTaskCancelled(ctx context.Context, teamName string, taskID int64) error
	ClearTaskGitFields(ctx context.Context, taskID int64) error
	UpdateTaskGitFields(ctx context.Context, taskID int64, worktreePath, branchName, baseSHA, repoName *string) error
	RewindTask(ctx context.Context, teamName string, taskID int64) error
	CreateTaskComment(ctx context.Context, teamName string, taskID int64, author, body string) (int64, error)
	ListTaskComments(ctx context.Context, teamName string, taskID int64) ([]TaskComment, error)
	AddTaskAttachment(ctx context.Context, teamName string, taskID int64, filePath string) error
	RemoveTaskAttachment(ctx context.Context, teamName string, taskID int64, filePath string) error
	ListTaskAttachments(ctx context.Context, teamName string, taskID int64) ([]TaskAttachment, error)
	AddTaskDependency(ctx context.Context, teamName string, taskID, dependsOnTaskID int64) error
	ListTaskDependencies(ctx context.Context, teamName string, taskID int64) ([]int64, error)
	NextRunnableTaskForTeam(ctx context.Context, teamName string) (*Task, error)
	GetTaskByIDAndTeam(ctx context.Context, teamName string, taskID int64) (*Task, error)
	UpdateTaskStage(ctx context.Context, taskID int64, stage string) error
	SetTaskWorkflowAndStage(ctx context.Context, taskID int64, workflowID, stage string) error

	// Task reviews (agent-to-agent or human)
	CreateTaskReview(ctx context.Context, teamName string, taskID int64, reviewerAgent, outcome, comments string) (int64, error)
	ListTaskReviews(ctx context.Context, teamName string, taskID int64) ([]TaskReview, error)

	// Repos
	ListRepos(ctx context.Context, teamName string) ([]Repo, error)
	CreateRepo(ctx context.Context, teamName, name, source, approval string, testCmd *string) error
	SetRepoApproval(ctx context.Context, teamName, repoName, approval string) error

	// Workflows
	ListWorkflows(ctx context.Context, teamName string) ([]Workflow, error)
	CreateWorkflow(ctx context.Context, teamName, name string, version int, sourcePath string) (string, error)
	CreateWorkflowWithStages(ctx context.Context, teamName, name string, version int, sourcePath string, stages []WorkflowStage, transitions []WorkflowTransition) (string, error)
	GetWorkflowStages(ctx context.Context, workflowID string) ([]WorkflowStage, error)
	GetWorkflowTransitions(ctx context.Context, workflowID string) ([]WorkflowTransition, error)
	GetWorkflowIDByTeamAndName(ctx context.Context, teamName, name string, version int) (string, error)
	GetWorkflowInitialStage(ctx context.Context, workflowID string) (string, error)

	// Network allowlist
	ListAllowedDomains(ctx context.Context) ([]string, error)
	ResetAllowlist(ctx context.Context) error
	AllowDomain(ctx context.Context, domain string) error
	DisallowDomain(ctx context.Context, domain string) error

	// Messages
	CreateMessage(ctx context.Context, teamName, sender, recipient, content string) (int64, error)
	ListMessages(ctx context.Context, teamName string, recipient string, limit int) ([]Message, error)
	ListUnprocessedMessages(ctx context.Context, teamName string, recipient string, limit int) ([]Message, error)
	MarkMessageProcessed(ctx context.Context, messageID int64) error

	// Lifecycle
	SeedDemo(ctx context.Context) error
	Close() error
}
