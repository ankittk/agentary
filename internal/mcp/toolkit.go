package mcp

import (
	"context"

	"github.com/ankittk/agentary/internal/store"
	"github.com/ankittk/agentary/pkg/models"
)

// Store is the minimal store interface required by MCPToolkit. *store.Store implements it.
type Store interface {
	CreateTask(ctx context.Context, teamName, title, status string, workflowID *string) (int64, error)
	ListTasks(ctx context.Context, teamName string, limit int) ([]store.Task, error)
	CreateMessage(ctx context.Context, teamName, sender, recipient, content string) (int64, error)
	ListMessages(ctx context.Context, teamName string, recipient string, limit int) ([]store.Message, error)
}

// MCPToolkit exposes validated tool methods for an agent. Agent identity (AgentName, TeamName)
// is baked into every call so agents cannot impersonate others. Use this when exposing
// task/mailbox operations to agents via MCP or other tool-call interfaces.
type MCPToolkit struct {
	Store     Store
	AgentName string
	TeamName  string
}

// CreateTask creates a new task in the team with the given title. Status is set to "todo".
// The creating agent is not set as assignee; the scheduler assigns later.
func (t *MCPToolkit) CreateTask(ctx context.Context, title string) (int64, error) {
	return t.Store.CreateTask(ctx, t.TeamName, title, models.StatusTodo, nil)
}

// SendMessage sends a message from this agent to the given recipient.
func (t *MCPToolkit) SendMessage(ctx context.Context, recipient, content string) (int64, error) {
	return t.Store.CreateMessage(ctx, t.TeamName, t.AgentName, recipient, content)
}

// ListTasks returns tasks for the team. Limit 0 means no limit (use a reasonable cap in production).
func (t *MCPToolkit) ListTasks(ctx context.Context, limit int) ([]store.Task, error) {
	if limit <= 0 {
		limit = models.DefaultMCPTaskLimit
	}
	return t.Store.ListTasks(ctx, t.TeamName, limit)
}

// ListMessages returns messages for the team, optionally filtered by recipient (inbox for that agent).
// Use recipient == t.AgentName to get this agent's inbox.
func (t *MCPToolkit) ListMessages(ctx context.Context, recipient string, limit int) ([]store.Message, error) {
	if limit <= 0 {
		limit = models.DefaultMCPMessageLimit
	}
	return t.Store.ListMessages(ctx, t.TeamName, recipient, limit)
}
