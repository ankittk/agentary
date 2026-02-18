// Package store defines the persistence interface and shared models for teams, tasks, workflows, and messages.
package store

import "time"

// Team is a named group of agents and tasks.
type Team struct {
	TeamID     string
	Name       string
	CreatedAt  time.Time
	AgentCount int
	TaskCount  int
}

// Agent is a team member (e.g. manager or engineer) that can be assigned tasks.
type Agent struct {
	Name      string
	Role      string
	CreatedAt time.Time
}

// Task is a work item with status, assignee, workflow stage, and optional git worktree info.
type Task struct {
	TaskID       int64
	Title        string
	Status       string
	Assignee     *string
	DRI          *string // Directly Responsible Individual (set on first assignment, never changes)
	AttemptCount int
	WorkflowID   *string
	CurrentStage *string
	WorktreePath *string // Git worktree path (e.g. ~/.agentary/teams/<team>/worktrees/<repo>-T<id>)
	BranchName   *string // agentary/<team_id>/<team>/T<NNNN>
	BaseSHA      *string // Base commit when branch was created
	RepoName     *string // Optional repo name for this task
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TaskComment is a comment on a task (author and body).
type TaskComment struct {
	CommentID int64
	TaskID    int64
	TeamID    string
	Author    string
	Body      string
	CreatedAt time.Time
}

// TaskAttachment is a file attachment on a task.
type TaskAttachment struct {
	AttachmentID int64
	TaskID       int64
	FilePath     string
	CreatedAt    time.Time
}

// Repo is a git repository linked to a team (source path, approval mode, optional test command).
type Repo struct {
	Name      string
	Source    string
	Approval  string
	TestCmd   *string
	CreatedAt time.Time
}

// Workflow is a named workflow definition (version and source path or builtin).
type Workflow struct {
	Name       string
	Version    int
	SourcePath string
	CreatedAt  time.Time
}

// WorkflowStage is a stage in a workflow (agent, human, auto, terminal).
type WorkflowStage struct {
	WorkflowID      string
	StageName       string
	StageType       string // agent, human, auto, terminal
	Outcomes        string // comma-separated outcomes, empty for terminal
	CandidateAgents string // comma-separated agent names; if set, scheduler picks assignee from this pool
}

// WorkflowTransition is (from_stage, outcome) -> to_stage.
type WorkflowTransition struct {
	WorkflowID string
	FromStage  string
	Outcome    string
	ToStage    string
}

// TaskReview is an agent-to-agent or human review submission for a task.
type TaskReview struct {
	ReviewID      int64
	TaskID        int64
	TeamID        string
	ReviewerAgent string
	Outcome       string // "approved" or "changes_requested"
	Comments      string
	CreatedAt     time.Time
}

// Message is used for agent↔agent or human↔manager communication (mailbox).
type Message struct {
	MessageID   int64
	TeamID      string
	Sender      string
	Recipient   string
	Content     string
	CreatedAt   time.Time
	ProcessedAt *time.Time
}
