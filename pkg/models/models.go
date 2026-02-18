// Package models provides shared types for the Agentary HTTP API and external tools.
// These types mirror the API JSON and are stable for use by pkg/client and other consumers.
package models

import "time"

// Team is a named group of agents and tasks.
type Team struct {
	TeamID     string    `json:"team_id,omitempty"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
	AgentCount int       `json:"agent_count,omitempty"`
	TaskCount  int       `json:"task_count,omitempty"`
}

// Agent is a team member (e.g. manager or engineer) that can be assigned tasks.
type Agent struct {
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// Task is a work item with status, assignee, workflow stage, and optional git worktree info.
type Task struct {
	TaskID       int64     `json:"task_id"`
	Title        string    `json:"title"`
	Status       string    `json:"status"`
	Assignee     *string   `json:"assignee,omitempty"`
	DRI          *string   `json:"dri,omitempty"`
	AttemptCount int       `json:"attempt_count,omitempty"`
	WorkflowID   *string   `json:"workflow_id,omitempty"`
	CurrentStage *string   `json:"current_stage,omitempty"`
	WorktreePath *string   `json:"worktree_path,omitempty"`
	BranchName   *string   `json:"branch_name,omitempty"`
	BaseSHA      *string   `json:"base_sha,omitempty"`
	RepoName     *string   `json:"repo_name,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
}

// TaskComment is a comment on a task.
type TaskComment struct {
	CommentID int64     `json:"comment_id"`
	TaskID    int64     `json:"task_id"`
	TeamID    string    `json:"team_id,omitempty"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// TaskAttachment is a file attachment on a task.
type TaskAttachment struct {
	AttachmentID int64     `json:"attachment_id"`
	TaskID       int64     `json:"task_id"`
	FilePath     string    `json:"file_path"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
}

// Repo is a git repository linked to a team.
type Repo struct {
	Name      string     `json:"name"`
	Source    string     `json:"source"`
	Approval  string     `json:"approval"`
	TestCmd   *string    `json:"test_cmd,omitempty"`
	CreatedAt time.Time  `json:"created_at,omitempty"`
}

// Workflow is a named workflow definition.
type Workflow struct {
	Name       string    `json:"name"`
	Version    int       `json:"version"`
	SourcePath string    `json:"source_path,omitempty"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
}

// WorkflowStage is a stage in a workflow (agent, human, auto, terminal).
type WorkflowStage struct {
	WorkflowID      string `json:"workflow_id"`
	StageName       string `json:"stage_name"`
	StageType       string `json:"stage_type"`
	Outcomes        string `json:"outcomes,omitempty"`
	CandidateAgents string `json:"candidate_agents,omitempty"`
}

// WorkflowTransition is (from_stage, outcome) -> to_stage.
type WorkflowTransition struct {
	WorkflowID string `json:"workflow_id"`
	FromStage  string `json:"from_stage"`
	Outcome    string `json:"outcome"`
	ToStage    string `json:"to_stage"`
}

// TaskReview is a review submission for a task.
type TaskReview struct {
	ReviewID      int64     `json:"review_id"`
	TaskID        int64     `json:"task_id"`
	TeamID        string    `json:"team_id,omitempty"`
	ReviewerAgent string    `json:"reviewer_agent"`
	Outcome       string    `json:"outcome"`
	Comments      string    `json:"comments,omitempty"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
}

// Message is used for agent↔agent or human↔manager communication.
type Message struct {
	MessageID   int64      `json:"message_id"`
	TeamID      string     `json:"team_id,omitempty"`
	Sender      string     `json:"sender"`
	Recipient   string     `json:"recipient"`
	Content     string     `json:"content"`
	CreatedAt   time.Time  `json:"created_at,omitempty"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

// Config is the /config API response.
type Config struct {
	HumanName   string `json:"human_name,omitempty"`
	HcHome      string `json:"hc_home,omitempty"`
	BootstrapID string `json:"bootstrap_id,omitempty"`
}

// Bootstrap is the /bootstrap API response.
type Bootstrap struct {
	Config      Config    `json:"config"`
	Teams       []Team    `json:"teams"`
	InitialTeam *string   `json:"initial_team,omitempty"`
	Tasks       []Task    `json:"tasks,omitempty"`
	Agents      []Agent   `json:"agents,omitempty"`
	Repos       []Repo    `json:"repos,omitempty"`
	Workflows   []Workflow `json:"workflows,omitempty"`
	Network     struct {
		Allowlist []string `json:"allowlist,omitempty"`
	} `json:"network"`
}
