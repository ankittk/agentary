package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/ankittk/agentary/internal/httpapi"
	"github.com/ankittk/agentary/internal/sandbox"
	"github.com/ankittk/agentary/internal/store"
	"github.com/ankittk/agentary/pkg/models"
)

const (
	DefaultManagerRecipient = "manager"
	InboxPollInterval       = 5 * time.Second
)

// Run subscribes to the app's event hub and runs a minimal rule-based manager:
// on task_update with status "done", optionally creates a follow-up "Review: <title>" task.
// It can create/advance/assign tasks via the store (scheduler will assign newly created tasks).
func Run(ctx context.Context, app *httpapi.App) {
	ch := app.Hub.Subscribe()
	defer app.Hub.Unsubscribe(ch)

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
			handleEvent(ctx, app, payload)
		}
	}
}

func handleEvent(ctx context.Context, app *httpapi.App, payload map[string]any) {
	typ, _ := payload["type"].(string)
	if typ != "task_update" {
		return
	}
	status, _ := payload["status"].(string)
	team, _ := payload["team"].(string)
	taskIDVal, ok := payload["task_id"]
	if !ok || team == "" {
		return
	}
	var taskID int64
	switch v := taskIDVal.(type) {
	case float64:
		taskID = int64(v)
	case int64:
		taskID = v
	default:
		return
	}

	switch status {
	case models.StatusDone:
		// Rule: create a follow-up "Review: <title>" task in the same team.
		task, err := app.Store.GetTaskByIDAndTeam(ctx, team, taskID)
		if err != nil || task == nil {
			return
		}
		title := "Review: " + task.Title
		var wfID *string
		if defaultWF, _ := app.Store.GetWorkflowIDByTeamAndName(ctx, team, "default", 1); defaultWF != "" {
			wfID = &defaultWF
		}
		id, err := app.Store.CreateTask(ctx, team, title, models.StatusTodo, wfID)
		if err != nil {
			slog.Warn("manager create follow-up task failed", "team", team, "err", err)
			return
		}
		slog.Info("manager created follow-up review task", "team", team, "task_id", id, "title", title)
		app.Hub.PublishJSON(map[string]any{"type": "task_update", "team": team, "task_id": id})
		if app.Capabilities != nil {
			if c := app.Capabilities.Get("slack"); c != nil {
				_ = c.Notify(ctx, "Task completed: "+task.Title+" (follow-up Review task #"+formatTaskID(id)+" created)")
			}
		}
	case models.StatusFailed:
		// Optional: requeue failed tasks (advance by resetting to todo). Disabled by default.
		if shouldRequeueFailed(payload) {
			_ = app.Store.RequeueTask(ctx, team, taskID)
			slog.Info("manager requeued failed task", "team", team, "task_id", taskID)
			app.Hub.PublishJSON(map[string]any{"type": "task_update", "team": team, "task_id": taskID, "status": models.StatusTodo})
		}
	}
}

// shouldRequeueFailed can be extended with config; for now we do not auto-requeue.
func shouldRequeueFailed(payload map[string]any) bool {
	// Could check payload["attempt_count"] or a config flag.
	_ = payload
	return false
}

// AssignTask assigns a task to an agent (e.g. manager or first worker). Used by rule-based flows.
func AssignTask(ctx context.Context, st store.Store, team string, taskID int64, assignee string) error {
	return st.UpdateTask(ctx, taskID, "", &assignee)
}

// AdvanceTask sets task status (and optionally assignee). Used by rule-based flows.
func AdvanceTask(ctx context.Context, st store.Store, taskID int64, status string, assignee *string) error {
	allowed := map[string]bool{models.StatusTodo: true, models.StatusInProgress: true, models.StatusDone: true, models.StatusFailed: true, models.StatusCancelled: true}
	if !allowed[status] {
		return nil
	}
	return st.UpdateTask(ctx, taskID, status, assignee)
}

// CreateTaskForTeam creates a task in the given team (convenience for manager rules).
func CreateTaskForTeam(ctx context.Context, st store.Store, team, title, status string) (int64, error) {
	var wfID *string
	if w, _ := st.GetWorkflowIDByTeamAndName(ctx, team, "default", 1); w != "" {
		wfID = &w
	}
	if status == "" {
		status = models.StatusTodo
	}
	return st.CreateTask(ctx, team, title, status, wfID)
}

// PollInbox runs in a loop: lists unprocessed messages for recipient "manager" (or managerRecipient), handles each (e.g. /shell, create task, reply), marks processed. Drives manager turns from mailbox.
func PollInbox(ctx context.Context, app *httpapi.App, managerRecipient string, interval time.Duration) {
	if managerRecipient == "" {
		managerRecipient = DefaultManagerRecipient
	}
	if interval <= 0 {
		interval = InboxPollInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			teams, err := app.Store.ListTeams(ctx)
			if err != nil {
				continue
			}
			for _, t := range teams {
				msgs, err := app.Store.ListUnprocessedMessages(ctx, t.Name, managerRecipient, 10)
				if err != nil || len(msgs) == 0 {
					continue
				}
				for _, m := range msgs {
					handleInboxMessage(ctx, app, t.Name, m)
					_ = app.Store.MarkMessageProcessed(ctx, m.MessageID)
				}
			}
		}
	}
}

func handleInboxMessage(ctx context.Context, app *httpapi.App, team string, m store.Message) {
	content := strings.TrimSpace(m.Content)
	// /shell <cmd>: run command and reply with output (human runs commands from chat).
	if strings.HasPrefix(content, "/shell ") {
		cmdLine := strings.TrimSpace(content[len("/shell "):])
		if cmdLine == "" {
			replyToSender(ctx, app, team, m.Sender, "usage: /shell <command>")
			return
		}
		if sandbox.BlockedShellCommand(cmdLine) {
			replyToSender(ctx, app, team, m.Sender, "error: command not allowed")
			return
		}
		out, err := runShellCommand(ctx, cmdLine)
		if err != nil {
			replyToSender(ctx, app, team, m.Sender, "error: "+err.Error())
			return
		}
		replyToSender(ctx, app, team, m.Sender, out)
		return
	}
	// Default: reply with ack; if message looks like a task request, create task and reply with task id.
	var reply string
	if len(content) > 10 && !strings.HasPrefix(content, "/") {
		if id, err := CreateTaskForTeam(ctx, app.Store, team, content, models.StatusTodo); err == nil {
			slog.Info("manager created task from message", "team", team, "task_id", id, "from", m.Sender)
			reply = "Created task #" + formatTaskID(id)
		} else {
			reply = "Got: " + content
		}
	} else {
		reply = "Got: " + content
	}
	replyToSender(ctx, app, team, m.Sender, reply)
	app.Hub.PublishJSON(map[string]any{"type": "message", "team": team, "from": DefaultManagerRecipient, "to": m.Sender})
}

func replyToSender(ctx context.Context, app *httpapi.App, team, sender, body string) {
	_, _ = app.Store.CreateMessage(ctx, team, DefaultManagerRecipient, sender, body)
	app.Hub.PublishJSON(map[string]any{"type": "message", "team": team, "from": DefaultManagerRecipient, "to": sender})
}

func runShellCommand(ctx context.Context, cmdLine string) (string, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdLine)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return strings.TrimSpace(string(out)), nil
}

func formatTaskID(id int64) string {
	return fmt.Sprintf("%d", id)
}
