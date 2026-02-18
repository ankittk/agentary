package models

// Task statuses used throughout the codebase.
const (
	StatusTodo       = "todo"
	StatusInProgress = "in_progress"
	StatusInReview   = "in_review"
	StatusInApproval = "in_approval"
	StatusMerging    = "merging"
	StatusDone       = "done"
	StatusFailed     = "failed"
	StatusCancelled  = "cancelled"
)

// Agent roles.
const (
	RoleEngineer = "engineer"
	RoleManager  = "manager"
)

// Stage types for workflow engine.
const (
	StageTypeAgent    = "agent"
	StageTypeHuman    = "human"
	StageTypeAuto     = "auto"
	StageTypeTerminal = "terminal"
	StageTypeMerge    = "merge"
)

// Default limits.
const (
	DefaultMaxRequestBodyBytes = 1 << 20 // 1 MiB
	DefaultTaskListLimit       = 1000
	DefaultMessageListLimit    = 500
	DefaultSSEChannelBuffer    = 256
	DefaultMCPTaskLimit        = 500
	DefaultMCPMessageLimit     = 100
	DefaultSchedulerChanSize   = 32
)
