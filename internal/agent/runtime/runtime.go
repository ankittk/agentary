package runtime

import (
	"context"
	"time"
)

type Event struct {
	Type      string         `json:"type"`
	Team      string         `json:"team,omitempty"`
	Agent     string         `json:"agent,omitempty"`
	TaskID    *int64         `json:"task_id,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data,omitempty"`
}

type TurnRequest struct {
	Team             string
	Agent            string
	TaskID           *int64
	Input            string
	NetworkAllowlist []string // If non-empty, agent should only allow outbound to these domains (or "*" = unrestricted).
	// Per-agent model config (from agents/<name>/config.yaml); optional.
	Model     string // e.g. claude-sonnet
	MaxTokens int    // 0 = use default
}

type TurnResult struct {
	Output string
}

type Runtime interface {
	Name() string
	RunTurn(ctx context.Context, req TurnRequest, emit func(Event)) (TurnResult, error)
}
