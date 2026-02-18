package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ankittk/agentary/internal/httpapi"
)

// LLMOpts configures the LLM-backed manager (OpenAI-compatible API).
type LLMOpts struct {
	BaseURL string // e.g. https://api.openai.com
	APIKey  string
	Model   string // e.g. gpt-4o-mini
}

// RunLLM subscribes to the hub and handles task_update and message events using an LLM with tools:
// create_task, advance_task, assign_task, reply, message_agent. When opts.APIKey is empty, returns immediately.
func RunLLM(ctx context.Context, app *httpapi.App, opts LLMOpts) {
	if opts.APIKey == "" || opts.BaseURL == "" {
		return
	}
	if opts.Model == "" {
		opts.Model = "gpt-4o-mini"
	}
	ch := app.Hub.Subscribe()
	defer app.Hub.Unsubscribe(ch)

	tools := []map[string]any{
		{
			"type": "function",
			"function": map[string]any{
				"name":        "create_task",
				"description": "Create a new task in the team",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"team":   map[string]any{"type": "string", "description": "Team name"},
						"title":  map[string]any{"type": "string", "description": "Task title"},
						"status": map[string]any{"type": "string", "description": "todo or in_progress"},
					},
					"required": []string{"team", "title"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        "advance_task",
				"description": "Set task status (todo, in_progress, done, failed)",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"task_id": map[string]any{"type": "integer"},
						"status":  map[string]any{"type": "string"},
					},
					"required": []string{"task_id", "status"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        "assign_task",
				"description": "Assign a task to an agent",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"task_id":  map[string]any{"type": "integer"},
						"assignee": map[string]any{"type": "string"},
					},
					"required": []string{"task_id", "assignee"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]any{
				"name":        "reply",
				"description": "Send a message reply to a recipient (e.g. human)",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"team":      map[string]any{"type": "string"},
						"recipient": map[string]any{"type": "string"},
						"content":   map[string]any{"type": "string"},
					},
					"required": []string{"team", "recipient", "content"},
				},
			},
		},
	}

	for {
		select {
		case <-ctx.Done():
			return
		case raw, ok := <-ch:
			if !ok {
				return
			}
			var payload map[string]any
			if err := json.Unmarshal(raw, &payload); err != nil {
				continue
			}
			typ, _ := payload["type"].(string)
			if typ != "task_update" && typ != "message" {
				continue
			}
			handleLLMEvent(ctx, app, opts, tools, payload)
		}
	}
}

func handleLLMEvent(ctx context.Context, app *httpapi.App, opts LLMOpts, tools []map[string]any, payload map[string]any) {
	team, _ := payload["team"].(string)
	if team == "" {
		return
	}
	var content string
	if payload["type"] == "task_update" {
		status, _ := payload["status"].(string)
		taskID, _ := payload["task_id"].(float64)
		content = fmt.Sprintf("Event: task_update team=%s task_id=%.0f status=%s", team, taskID, status)
	} else {
		from, _ := payload["from"].(string)
		to, _ := payload["to"].(string)
		content = fmt.Sprintf("Event: message team=%s from=%s to=%s", team, from, to)
	}
	messages := []map[string]any{
		{"role": "system", "content": "You are a manager agent. Use the provided tools to create/advance/assign tasks or reply to messages. Prefer brevity."},
		{"role": "user", "content": content},
	}
	reqBody := map[string]any{
		"model":       opts.Model,
		"messages":    messages,
		"tools":       tools,
		"tool_choice": "auto",
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return
	}
	url := strings.TrimSuffix(opts.BaseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+opts.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Warn("LLM manager request failed", "err", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		slog.Warn("LLM manager API returned non-200", "status", resp.StatusCode)
		return
	}
	var apiResp struct {
		Choices []struct {
			Message struct {
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return
	}
	if len(apiResp.Choices) == 0 {
		return
	}
	for _, tc := range apiResp.Choices[0].Message.ToolCalls {
		executeLLMTool(ctx, app, team, tc.Function.Name, tc.Function.Arguments)
	}
}

func executeLLMTool(ctx context.Context, app *httpapi.App, defaultTeam, name, argsJSON string) {
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return
	}
	switch name {
	case "create_task":
		team, _ := args["team"].(string)
		if team == "" {
			team = defaultTeam
		}
		title, _ := args["title"].(string)
		status, _ := args["status"].(string)
		if status == "" {
			status = "todo"
		}
		if id, err := CreateTaskForTeam(ctx, app.Store, team, title, status); err == nil {
			slog.Info("LLM manager created task", "team", team, "task_id", id)
			app.Hub.PublishJSON(map[string]any{"type": "task_update", "team": team, "task_id": id})
		}
	case "advance_task":
		taskID, _ := args["task_id"].(float64)
		status, _ := args["status"].(string)
		if err := app.Store.UpdateTask(ctx, int64(taskID), status, nil); err == nil {
			app.Hub.PublishJSON(map[string]any{"type": "task_update", "task_id": int64(taskID), "status": status})
		}
	case "assign_task":
		taskID, _ := args["task_id"].(float64)
		assignee, _ := args["assignee"].(string)
		a := &assignee
		if err := app.Store.UpdateTask(ctx, int64(taskID), "", a); err == nil {
			app.Hub.PublishJSON(map[string]any{"type": "task_update", "task_id": int64(taskID), "assignee": assignee})
		}
	case "reply":
		team, _ := args["team"].(string)
		if team == "" {
			team = defaultTeam
		}
		recipient, _ := args["recipient"].(string)
		content, _ := args["content"].(string)
		if _, err := app.Store.CreateMessage(ctx, team, DefaultManagerRecipient, recipient, content); err == nil {
			app.Hub.PublishJSON(map[string]any{"type": "message", "team": team, "from": DefaultManagerRecipient, "to": recipient})
		}
	case "message_agent":
		team, _ := args["team"].(string)
		if team == "" {
			team = defaultTeam
		}
		recipient, _ := args["recipient"].(string)
		content, _ := args["content"].(string)
		if _, err := app.Store.CreateMessage(ctx, team, DefaultManagerRecipient, recipient, content); err == nil {
			app.Hub.PublishJSON(map[string]any{"type": "message", "team": team, "from": DefaultManagerRecipient, "to": recipient})
		}
	}
}
