package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ankittk/agentary/internal/capabilities"
	"github.com/ankittk/agentary/internal/git"
	"github.com/ankittk/agentary/internal/memory"
	"github.com/ankittk/agentary/internal/otel"
	"github.com/ankittk/agentary/internal/review"
	"github.com/ankittk/agentary/internal/store"
	"github.com/ankittk/agentary/internal/store/postgres"
	"github.com/ankittk/agentary/internal/ui"
	"github.com/ankittk/agentary/pkg/models"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// defaultMaxRequestBodyBytes is the default limit for request body size (1 MiB) to prevent OOM.
const defaultMaxRequestBodyBytes = 1 << 20

// limitBody wraps r.Body with http.MaxBytesReader so handlers cannot read more than maxBytes.
// Call this for requests that have a body (e.g. POST, PUT, PATCH) before decoding JSON.
func limitBody(w http.ResponseWriter, r *http.Request, maxBytes int64) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
}

// bodyLimitMiddleware limits request body size for POST, PUT, PATCH to prevent OOM.
func bodyLimitMiddleware(maxBytes int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			limitBody(w, r, maxBytes)
		}
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware sets CORS headers for dev mode (Vite dev server on different origin).
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ServerOptions configures the HTTP server (home dir, listen addr, API key, DB, metrics).
type ServerOptions struct {
	Home           string
	Addr           string
	Dev            bool
	APIKey         string       // if set, require X-API-Key header or query api_key
	DBDriver       string       // "sqlite" (default) or "postgres"
	DBURL          string       // for postgres: connection string (or set DATABASE_URL env)
	MetricsHandler http.Handler // if set, used for /metrics (e.g. OTel Prometheus handler)
	UseOtelHTTP    bool         // if true, wrap handler with otelhttp for request metrics
}

// App holds the HTTP server, SSE hub, store, capabilities registry, and home path.
type App struct {
	Server       *http.Server
	Hub          *SSEHub
	Store        store.Store
	Capabilities *capabilities.Registry // optional; loaded from env (e.g. SLACK_WEBHOOK_URL)
	Home         string                 // data directory; for team/agent dirs and charter
}

// NewServer builds an HTTP server from options; kept for backward compatibility (prefer NewApp).
func NewServer(opts ServerOptions) *http.Server {
	app, err := NewApp(opts)
	if err != nil {
		panic(err)
	}
	return app.Server
}

// NewApp creates the HTTP app (server, hub, store, capabilities) and registers all routes.
func NewApp(opts ServerOptions) (*App, error) {
	hub := NewSSEHub()
	mux := http.NewServeMux()

	var st store.Store
	var err error
	if opts.DBDriver == "postgres" {
		st, err = postgres.Open(opts.DBURL)
	} else {
		st, err = store.Open(opts.Home)
	}
	if err != nil {
		return nil, err
	}
	_ = st.SeedDemo(context.Background())

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	if opts.MetricsHandler != nil {
		mux.Handle("/metrics", opts.MetricsHandler)
	} else {
		mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			teams, _ := st.ListTeams(r.Context())
			var todo, inProgress, done, failed int64
			for _, t := range teams {
				tasks, _ := st.ListTasks(r.Context(), t.Name, 0)
				for _, tk := range tasks {
					switch tk.Status {
					case models.StatusTodo:
						todo++
					case models.StatusInProgress:
						inProgress++
					case models.StatusDone:
						done++
					case models.StatusFailed:
						failed++
					}
				}
			}
			_, _ = fmt.Fprintf(w, "# TYPE agentary_tasks_total gauge\n")
			_, _ = fmt.Fprintf(w, "agentary_tasks_total{status=\"todo\"} %d\n", todo)
			_, _ = fmt.Fprintf(w, "agentary_tasks_total{status=\"in_progress\"} %d\n", inProgress)
			_, _ = fmt.Fprintf(w, "agentary_tasks_total{status=\"done\"} %d\n", done)
			_, _ = fmt.Fprintf(w, "agentary_tasks_total{status=\"failed\"} %d\n", failed)
		})
	}

	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"human_name":   "human",
			"hc_home":      opts.Home,
			"bootstrap_id": getBootstrapID(opts.Home),
		})
	})

	mux.HandleFunc("/bootstrap", func(w http.ResponseWriter, r *http.Request) {
		handleBootstrap(w, r, st, opts.Home)
	})

	mux.HandleFunc("/stream", hub.Handler())

	// --- Teams ---
	mux.HandleFunc("/teams", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			teams, err := st.ListTeams(r.Context())
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, teams)
			return
		case http.MethodPost:
			var body struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid json")
				return
			}
			if body.Name == "" {
				writeJSONError(w, http.StatusBadRequest, "name required")
				return
			}
			t, err := st.CreateTeam(r.Context(), body.Name)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, err.Error())
				return
			}
			if opts.Home != "" {
				_ = memory.EnsureTeamDirs(memory.TeamDir(opts.Home, t.Name))
			}
			hub.PublishJSON(map[string]any{"type": "team_update", "team": t.Name})
			writeJSON(w, t)
			return
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
	})

	// --- Team-scoped endpoints ---
	mux.HandleFunc("/teams/", func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/teams/")
		parts := strings.Split(rest, "/")
		if len(parts) < 1 {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
		team := parts[0]

		// /teams/{team}
		if len(parts) == 1 || parts[1] == "" {
			if r.Method == http.MethodDelete {
				if err := st.DeleteTeam(r.Context(), team); err != nil {
					writeJSONError(w, http.StatusNotFound, err.Error())
					return
				}
				hub.PublishJSON(map[string]any{"type": "team_update", "team": team})
				writeJSON(w, map[string]any{"ok": true})
				return
			}
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}

		switch parts[1] {
		case "tasks":
			// /teams/{team}/tasks/{id} or /teams/{team}/tasks/{id}/comments|attachments|dependencies
			if len(parts) >= 3 && parts[2] != "" {
				var taskID int64
				if _, err := fmt.Sscanf(parts[2], "%d", &taskID); err != nil {
					writeJSONError(w, http.StatusBadRequest, "invalid task id")
					return
				}
				task, err := st.GetTaskByIDAndTeam(r.Context(), team, taskID)
				if err != nil {
					writeJSONError(w, http.StatusInternalServerError, err.Error())
					return
				}
				if task == nil {
					writeJSONError(w, http.StatusNotFound, "task not found")
					return
				}
				// /teams/{team}/tasks/{id}/comments
				if len(parts) >= 4 && parts[3] == "comments" {
					switch r.Method {
					case http.MethodGet:
						comments, err := st.ListTaskComments(r.Context(), team, taskID)
						if err != nil {
							writeJSONError(w, http.StatusInternalServerError, err.Error())
							return
						}
						writeJSON(w, comments)
						return
					case http.MethodPost:
						var body struct {
							Author string `json:"author"`
							Body   string `json:"body"`
						}
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							writeJSONError(w, http.StatusBadRequest, "invalid json")
							return
						}
						if body.Author == "" {
							body.Author = "api"
						}
						id, err := st.CreateTaskComment(r.Context(), team, taskID, body.Author, body.Body)
						if err != nil {
							writeJSONError(w, http.StatusBadRequest, err.Error())
							return
						}
						writeJSON(w, map[string]any{"comment_id": id})
						return
					default:
						writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
						return
					}
				}
				// /teams/{team}/tasks/{id}/attachments
				if len(parts) >= 4 && parts[3] == "attachments" {
					switch r.Method {
					case http.MethodGet:
						attachments, err := st.ListTaskAttachments(r.Context(), team, taskID)
						if err != nil {
							writeJSONError(w, http.StatusInternalServerError, err.Error())
							return
						}
						writeJSON(w, attachments)
						return
					case http.MethodPost:
						var body struct {
							FilePath string `json:"file_path"`
						}
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							writeJSONError(w, http.StatusBadRequest, "invalid json")
							return
						}
						if body.FilePath == "" {
							writeJSONError(w, http.StatusBadRequest, "file_path required")
							return
						}
						if err := st.AddTaskAttachment(r.Context(), team, taskID, body.FilePath); err != nil {
							writeJSONError(w, http.StatusBadRequest, err.Error())
							return
						}
						writeJSON(w, map[string]any{"ok": true})
						return
					case http.MethodDelete:
						filePath := r.URL.Query().Get("file_path")
						if filePath == "" {
							writeJSONError(w, http.StatusBadRequest, "file_path query required")
							return
						}
						if err := st.RemoveTaskAttachment(r.Context(), team, taskID, filePath); err != nil {
							writeJSONError(w, http.StatusBadRequest, err.Error())
							return
						}
						writeJSON(w, map[string]any{"ok": true})
						return
					default:
						writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
						return
					}
				}
				// /teams/{team}/tasks/{id}/dependencies
				if len(parts) >= 4 && parts[3] == "dependencies" {
					switch r.Method {
					case http.MethodGet:
						deps, err := st.ListTaskDependencies(r.Context(), team, taskID)
						if err != nil {
							writeJSONError(w, http.StatusInternalServerError, err.Error())
							return
						}
						writeJSON(w, map[string]any{"depends_on": deps})
						return
					case http.MethodPost:
						var body struct {
							DependsOnTaskID int64 `json:"depends_on_task_id"`
						}
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							writeJSONError(w, http.StatusBadRequest, "invalid json")
							return
						}
						if body.DependsOnTaskID <= 0 {
							writeJSONError(w, http.StatusBadRequest, "depends_on_task_id required")
							return
						}
						if err := st.AddTaskDependency(r.Context(), team, taskID, body.DependsOnTaskID); err != nil {
							writeJSONError(w, http.StatusBadRequest, err.Error())
							return
						}
						writeJSON(w, map[string]any{"ok": true})
						return
					default:
						writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
						return
					}
				}
				// /teams/{team}/tasks/{id}/approve — POST review outcome (e.g. approved, changes_requested) to advance human/Review stage
				if len(parts) >= 4 && parts[3] == "approve" {
					if r.Method != http.MethodPost {
						writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
						return
					}
					var body struct {
						Outcome string `json:"outcome"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						writeJSONError(w, http.StatusBadRequest, "invalid json")
						return
					}
					if body.Outcome == "" {
						body.Outcome = "approved"
					}
					if task.WorkflowID == nil || *task.WorkflowID == "" || task.CurrentStage == nil {
						writeJSONError(w, http.StatusBadRequest, "task has no workflow or current stage")
						return
					}
					transitions, err := st.GetWorkflowTransitions(r.Context(), *task.WorkflowID)
					if err != nil {
						writeJSONError(w, http.StatusInternalServerError, err.Error())
						return
					}
					var nextStage string
					for _, tr := range transitions {
						if tr.FromStage == *task.CurrentStage && tr.Outcome == body.Outcome {
							nextStage = tr.ToStage
							break
						}
					}
					if nextStage == "" {
						writeJSONError(w, http.StatusBadRequest, "no transition for stage "+*task.CurrentStage+" with outcome "+body.Outcome)
						return
					}
					if err := st.UpdateTaskStage(r.Context(), taskID, nextStage); err != nil {
						writeJSONError(w, http.StatusInternalServerError, err.Error())
						return
					}
					stages, _ := st.GetWorkflowStages(r.Context(), *task.WorkflowID)
					isTerminal := false
					for _, s := range stages {
						if s.StageName == nextStage && s.StageType == "terminal" {
							isTerminal = true
							break
						}
					}
					if isTerminal {
						_ = st.UpdateTask(r.Context(), taskID, models.StatusDone, nil)
					}
					hub.PublishJSON(map[string]any{"type": "task_update", "team": team, "task_id": taskID, "current_stage": nextStage})
					writeJSON(w, map[string]any{"ok": true, "current_stage": nextStage})
					return
				}
				// /teams/{team}/tasks/{id}/diff — GET diff (base_sha → branch tip) for review UI
				if len(parts) >= 4 && parts[3] == "diff" {
					if r.Method != http.MethodGet {
						writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
						return
					}
					worktreePath := ""
					if task.WorktreePath != nil {
						worktreePath = *task.WorktreePath
					}
					baseSHA := "HEAD~1"
					if task.BaseSHA != nil && *task.BaseSHA != "" {
						baseSHA = *task.BaseSHA
					}
					headRef := "HEAD"
					if task.BranchName != nil && *task.BranchName != "" {
						headRef = *task.BranchName
					}
					diffOut, err := git.Diff(r.Context(), worktreePath, baseSHA, headRef)
					if err != nil {
						writeJSONError(w, http.StatusInternalServerError, err.Error())
						return
					}
					writeJSON(w, map[string]any{"diff": diffOut})
					return
				}
				// /teams/{team}/tasks/{id}/reviews — GET list of reviews
				if len(parts) >= 4 && parts[3] == "reviews" {
					if r.Method != http.MethodGet {
						writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
						return
					}
					reviews, err := st.ListTaskReviews(r.Context(), team, taskID)
					if err != nil {
						writeJSONError(w, http.StatusInternalServerError, err.Error())
						return
					}
					writeJSON(w, map[string]any{"reviews": reviews})
					return
				}
				// /teams/{team}/tasks/{id}/submit-review — POST submit review (reviewer_agent, outcome, comments)
				if len(parts) >= 4 && parts[3] == "submit-review" {
					if r.Method != http.MethodPost {
						writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
						return
					}
					var body struct {
						ReviewerAgent string `json:"reviewer_agent"`
						Outcome       string `json:"outcome"`
						Comments      string `json:"comments"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						writeJSONError(w, http.StatusBadRequest, "invalid json")
						return
					}
					if body.Outcome == "" {
						writeJSONError(w, http.StatusBadRequest, "outcome required (e.g. approved, changes_requested)")
						return
					}
					if err := review.SubmitReview(r.Context(), st, team, taskID, body.ReviewerAgent, body.Outcome, body.Comments); err != nil {
						writeJSONError(w, http.StatusBadRequest, err.Error())
						return
					}
					updated, _ := st.GetTaskByIDAndTeam(r.Context(), team, taskID)
					if updated != nil {
						hub.PublishJSON(map[string]any{"type": "task_update", "team": team, "task_id": taskID, "current_stage": updated.CurrentStage, "status": updated.Status})
					}
					writeJSON(w, map[string]any{"ok": true})
					return
				}
				// /teams/{team}/tasks/{id}/request-review — POST move to InReview and assign reviewer
				if len(parts) >= 4 && parts[3] == "request-review" {
					if r.Method != http.MethodPost {
						writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
						return
					}
					if task.WorkflowID == nil || *task.WorkflowID == "" {
						writeJSONError(w, http.StatusBadRequest, "task has no workflow")
						return
					}
					currentStage := ""
					if task.CurrentStage != nil {
						currentStage = *task.CurrentStage
					}
					transitions, err := st.GetWorkflowTransitions(r.Context(), *task.WorkflowID)
					if err != nil {
						writeJSONError(w, http.StatusInternalServerError, err.Error())
						return
					}
					var nextStage string
					for _, tr := range transitions {
						if tr.FromStage == currentStage && tr.Outcome == "submit_for_review" {
							nextStage = tr.ToStage
							break
						}
					}
					if nextStage == "" {
						writeJSONError(w, http.StatusBadRequest, "no submit_for_review transition from current stage")
						return
					}
					if err := st.SetTaskWorkflowAndStage(r.Context(), taskID, *task.WorkflowID, nextStage); err != nil {
						writeJSONError(w, http.StatusInternalServerError, err.Error())
						return
					}
					agents, _ := st.ListAgents(r.Context(), team)
					updated, _ := st.GetTaskByIDAndTeam(r.Context(), team, taskID)
					if updated != nil && len(agents) > 0 && nextStage == "InReview" {
						reviewer := review.PickReviewer(r.Context(), st, team, updated, agents)
						if reviewer != "" {
							_ = st.UpdateTask(r.Context(), taskID, "", &reviewer)
							updated.Assignee = &reviewer
						}
					}
					if updated == nil {
						updated, _ = st.GetTaskByIDAndTeam(r.Context(), team, taskID)
					}
					if updated != nil {
						hub.PublishJSON(map[string]any{"type": "task_update", "team": team, "task_id": taskID, "current_stage": &nextStage, "assignee": updated.Assignee})
					}
					writeJSON(w, map[string]any{"ok": true, "current_stage": nextStage})
					return
				}
				// /teams/{team}/tasks/{id} — GET or PATCH single task
				switch r.Method {
				case http.MethodGet:
					writeJSON(w, task)
					return
				case http.MethodPatch:
					var body struct {
						Status   *string `json:"status"`
						Assignee *string `json:"assignee"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						writeJSONError(w, http.StatusBadRequest, "invalid json")
						return
					}
					status := ""
					if body.Status != nil {
						status = *body.Status
						allowed := map[string]bool{
							models.StatusTodo: true, models.StatusInProgress: true, models.StatusInReview: true,
							models.StatusInApproval: true, models.StatusMerging: true,
							models.StatusDone: true, models.StatusFailed: true, models.StatusCancelled: true,
						}
						if status != "" && !allowed[status] {
							writeJSONError(w, http.StatusBadRequest, "status must be todo, in_progress, in_review, in_approval, merging, done, failed, or cancelled")
							return
						}
					}
					if err := st.UpdateTask(r.Context(), taskID, status, body.Assignee); err != nil {
						writeJSONError(w, http.StatusBadRequest, err.Error())
						return
					}
					updated, _ := st.GetTaskByIDAndTeam(r.Context(), team, taskID)
					if updated != nil {
						payload := map[string]any{"type": "task_update", "team": team, "task_id": taskID, "status": updated.Status}
						if updated.Assignee != nil {
							payload["assignee"] = *updated.Assignee
						}
						hub.PublishJSON(payload)
						writeJSON(w, updated)
					} else {
						writeJSON(w, map[string]any{"task_id": taskID, "ok": true})
					}
					return
				default:
					writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
					return
				}
			}
			// /teams/{team}/tasks — list or create
			switch r.Method {
			case http.MethodGet:
				limit := 0
				if l := r.URL.Query().Get("limit"); l != "" {
					if n, _ := fmt.Sscanf(l, "%d", &limit); n == 1 && limit > 0 {
						if limit > models.DefaultTaskListLimit {
							limit = models.DefaultTaskListLimit
						}
					}
				}
				tasks, err := st.ListTasks(r.Context(), team, limit)
				if err != nil {
					writeJSONError(w, http.StatusNotFound, err.Error())
					return
				}
				writeJSON(w, tasks)
				return
			case http.MethodPost:
				var body struct {
					Title  string `json:"title"`
					Status string `json:"status"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					writeJSONError(w, http.StatusBadRequest, "invalid json")
					return
				}
				if body.Title == "" {
					writeJSONError(w, http.StatusBadRequest, "title required")
					return
				}
				if body.Status != "" && body.Status != models.StatusTodo && body.Status != models.StatusInProgress {
					writeJSONError(w, http.StatusBadRequest, "status must be todo or in_progress")
					return
				}
				var wfID *string
				if defaultWF, _ := st.GetWorkflowIDByTeamAndName(r.Context(), team, "default", 1); defaultWF != "" {
					wfID = &defaultWF
				}
				id, err := st.CreateTask(r.Context(), team, body.Title, body.Status, wfID)
				if err != nil {
					writeJSONError(w, http.StatusBadRequest, err.Error())
					return
				}
				otel.RecordTaskOp(r.Context(), "create", team, body.Status)
				hub.PublishJSON(map[string]any{"type": "task_update", "team": team, "task_id": id})
				writeJSON(w, map[string]any{"task_id": id})
				return
			default:
				writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}

		case "agents":
			// /teams/{team}/agents/{agent}/journal — GET agent journal (optional ?limit= bytes)
			if len(parts) >= 4 && parts[3] == "journal" && parts[2] != "" {
				if r.Method != http.MethodGet {
					writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
					return
				}
				agentName := parts[2]
				if opts.Home == "" {
					writeJSONError(w, http.StatusBadRequest, "home not configured")
					return
				}
				teamDir := memory.TeamDir(opts.Home, team)
				j := &memory.Journal{AgentName: agentName, TeamDir: teamDir}
				limitBytes := 0
				if l := r.URL.Query().Get("limit"); l != "" {
					if n, _ := fmt.Sscanf(l, "%d", &limitBytes); n == 1 && limitBytes > 0 {
						if limitBytes > 512*1024 {
							limitBytes = 512 * 1024
						}
					}
				}
				content, err := j.Read(r.Context(), limitBytes)
				if err != nil {
					writeJSONError(w, http.StatusInternalServerError, err.Error())
					return
				}
				writeJSON(w, map[string]any{"content": content})
				return
			}
			// /teams/{team}/agents/{agent}/config — GET agent config (config.yaml)
			if len(parts) >= 4 && parts[3] == "config" && parts[2] != "" {
				if r.Method != http.MethodGet {
					writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
					return
				}
				agentName := parts[2]
				if opts.Home == "" {
					writeJSONError(w, http.StatusBadRequest, "home not configured")
					return
				}
				teamDir := memory.TeamDir(opts.Home, team)
				agentDir := memory.AgentDir(teamDir, agentName)
				cfg, err := memory.LoadAgentConfig(agentDir)
				if err != nil {
					writeJSONError(w, http.StatusInternalServerError, err.Error())
					return
				}
				if cfg == nil {
					writeJSON(w, map[string]any{"model": "", "max_tokens": 0})
					return
				}
				writeJSON(w, map[string]any{"model": cfg.Model, "max_tokens": cfg.MaxTokens})
				return
			}
			switch r.Method {
			case http.MethodGet:
				agents, err := st.ListAgents(r.Context(), team)
				if err != nil {
					writeJSONError(w, http.StatusNotFound, err.Error())
					return
				}
				writeJSON(w, agents)
				return
			case http.MethodPost:
				var body struct {
					Name string `json:"name"`
					Role string `json:"role"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					writeJSONError(w, http.StatusBadRequest, "invalid json")
					return
				}
				if err := st.CreateAgent(r.Context(), team, body.Name, body.Role); err != nil {
					writeJSONError(w, http.StatusBadRequest, err.Error())
					return
				}
				if opts.Home != "" {
					teamDir := memory.TeamDir(opts.Home, team)
					_ = memory.EnsureAgentDir(teamDir, body.Name)
				}
				hub.PublishJSON(map[string]any{"type": "agent_update", "team": team, "agent": body.Name})
				writeJSON(w, map[string]any{"ok": true})
				return
			default:
				writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}

		case "charter":
			teamDir := memory.TeamDir(opts.Home, team)
			switch r.Method {
			case http.MethodGet:
				content, err := memory.ReadCharter(teamDir)
				if err != nil {
					writeJSONError(w, http.StatusInternalServerError, err.Error())
					return
				}
				writeJSON(w, map[string]any{"content": content})
				return
			case http.MethodPut:
				var body struct {
					Content string `json:"content"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					writeJSONError(w, http.StatusBadRequest, "invalid json")
					return
				}
				if err := memory.WriteCharter(teamDir, body.Content); err != nil {
					writeJSONError(w, http.StatusInternalServerError, err.Error())
					return
				}
				writeJSON(w, map[string]any{"ok": true})
				return
			default:
				writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}

		case "repos":
			switch r.Method {
			case http.MethodGet:
				repos, err := st.ListRepos(r.Context(), team)
				if err != nil {
					writeJSONError(w, http.StatusNotFound, err.Error())
					return
				}
				writeJSON(w, repos)
				return
			case http.MethodPost:
				var body struct {
					Name     string  `json:"name"`
					Source   string  `json:"source"`
					Approval string  `json:"approval"`
					TestCmd  *string `json:"test_cmd"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					writeJSONError(w, http.StatusBadRequest, "invalid json")
					return
				}
				if body.Approval != "" && body.Approval != "auto" && body.Approval != "manual" {
					writeJSONError(w, http.StatusBadRequest, "approval must be auto or manual")
					return
				}
				if err := st.CreateRepo(r.Context(), team, body.Name, body.Source, body.Approval, body.TestCmd); err != nil {
					writeJSONError(w, http.StatusBadRequest, err.Error())
					return
				}
				hub.PublishJSON(map[string]any{"type": "repo_update", "team": team, "repo": body.Name})
				writeJSON(w, map[string]any{"ok": true})
				return
			default:
				writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}

		case "workflows":
			// /teams/{team}/workflows/init
			if len(parts) >= 3 && parts[2] == "init" {
				if r.Method != http.MethodPost {
					writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
					return
				}
				_, _ = st.CreateWorkflow(r.Context(), team, "default", 1, "builtin:default")
				hub.PublishJSON(map[string]any{"type": "workflow_update", "team": team, "workflow": "default"})
				writeJSON(w, map[string]any{"ok": true})
				return
			}
			switch r.Method {
			case http.MethodGet:
				wfs, err := st.ListWorkflows(r.Context(), team)
				if err != nil {
					writeJSONError(w, http.StatusNotFound, err.Error())
					return
				}
				writeJSON(w, wfs)
				return
			case http.MethodPost:
				var body struct {
					Name       string `json:"name"`
					Version    int    `json:"version"`
					SourcePath string `json:"source"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					writeJSONError(w, http.StatusBadRequest, "invalid json")
					return
				}
				if _, err := st.CreateWorkflow(r.Context(), team, body.Name, body.Version, body.SourcePath); err != nil {
					writeJSONError(w, http.StatusBadRequest, err.Error())
					return
				}
				hub.PublishJSON(map[string]any{"type": "workflow_update", "team": team, "workflow": body.Name})
				writeJSON(w, map[string]any{"ok": true})
				return
			default:
				writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}

		case "messages":
			// GET /teams/{team}/messages?recipient=X (inbox for X); POST send message
			switch r.Method {
			case http.MethodGet:
				recipient := r.URL.Query().Get("recipient")
				limit := 0
				if l := r.URL.Query().Get("limit"); l != "" {
					if n, _ := fmt.Sscanf(l, "%d", &limit); n == 1 && limit > 0 {
						if limit > models.DefaultMessageListLimit {
							limit = models.DefaultMessageListLimit
						}
					}
				}
				msgs, err := st.ListMessages(r.Context(), team, recipient, limit)
				if err != nil {
					writeJSONError(w, http.StatusInternalServerError, err.Error())
					return
				}
				writeJSON(w, msgs)
				return
			case http.MethodPost:
				var body struct {
					Sender    string `json:"sender"`
					Recipient string `json:"recipient"`
					Content   string `json:"content"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					writeJSONError(w, http.StatusBadRequest, "invalid json")
					return
				}
				if body.Sender == "" {
					body.Sender = "api"
				}
				if body.Recipient == "" {
					writeJSONError(w, http.StatusBadRequest, "recipient required")
					return
				}
				id, err := st.CreateMessage(r.Context(), team, body.Sender, body.Recipient, body.Content)
				if err != nil {
					writeJSONError(w, http.StatusBadRequest, err.Error())
					return
				}
				hub.PublishJSON(map[string]any{"type": "message", "team": team, "message_id": id})
				writeJSON(w, map[string]any{"message_id": id})
				return
			default:
				writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}

		default:
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
	})

	// --- Global network endpoints ---
	mux.HandleFunc("/network", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		domains, err := st.ListAllowedDomains(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, map[string]any{"allowlist": domains})
	})
	mux.HandleFunc("/network/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if err := st.ResetAllowlist(r.Context()); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		hub.PublishJSON(map[string]any{"type": "network_update"})
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/network/allow", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var body struct {
			Domain string `json:"domain"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid json")
			return
		}
		if body.Domain == "" {
			writeJSONError(w, http.StatusBadRequest, "domain required")
			return
		}
		if err := st.AllowDomain(r.Context(), body.Domain); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		hub.PublishJSON(map[string]any{"type": "network_update"})
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/network/disallow", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var body struct {
			Domain string `json:"domain"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid json")
			return
		}
		if body.Domain == "" {
			writeJSONError(w, http.StatusBadRequest, "domain required")
			return
		}
		if err := st.DisallowDomain(r.Context(), body.Domain); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		hub.PublishJSON(map[string]any{"type": "network_update"})
		writeJSON(w, map[string]any{"ok": true})
	})

	// UI: embedded React SPA (web/dist via go:embed)
	mux.Handle("/", ui.Handler())

	var handler http.Handler = mux
	handler = bodyLimitMiddleware(defaultMaxRequestBodyBytes, handler)
	if opts.Dev {
		handler = corsMiddleware(handler)
	}
	if opts.APIKey != "" {
		handler = apiKeyMiddleware(opts.APIKey, handler)
	}
	handler = requestLogMiddleware(handler)
	if opts.UseOtelHTTP {
		handler = otelhttp.NewHandler(handler, "agentary")
	}
	srv := &http.Server{
		Addr:              opts.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	srv.RegisterOnShutdown(func() {
		_ = st.Close()
	})

	reg := capabilities.NewRegistry()
	if u := os.Getenv("SLACK_WEBHOOK_URL"); u != "" {
		reg.Register("slack", capabilities.SlackWebhook{WebhookURL: u})
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		if repo := os.Getenv("GITHUB_OWNER_REPO"); repo != "" {
			reg.Register("github", capabilities.GitHubNotifier{Token: token, OwnerRepo: repo})
		}
	}
	return &App{Server: srv, Hub: hub, Store: st, Capabilities: reg, Home: opts.Home}, nil
}

// responseRecorder captures status code for logging and forwards Flusher if supported.
type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func apiKeyMiddleware(apiKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/health" || path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		key := r.Header.Get("X-API-Key")
		if key == "" {
			key = r.URL.Query().Get("api_key")
		}
		if key != apiKey {
			writeJSONError(w, http.StatusUnauthorized, "invalid or missing API key")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, req)
		slog.Info("request",
			"method", req.Method,
			"path", req.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds())
	})
}

func getBootstrapID(home string) string {
	protected := filepath.Join(home, "protected")
	_ = os.MkdirAll(protected, 0o755)
	path := filepath.Join(protected, "bootstrap_id")
	if b, err := os.ReadFile(path); err == nil {
		if s := string(bytesTrimSpace(b)); s != "" {
			return s
		}
	}
	id := randomHex(16)
	_ = os.WriteFile(path, []byte(id+"\n"), 0o644)
	return id
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// fallback: time-based
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(b)
}

func bytesTrimSpace(b []byte) []byte {
	i := 0
	j := len(b)
	for i < j && (b[i] == ' ' || b[i] == '\n' || b[i] == '\r' || b[i] == '\t') {
		i++
	}
	for j > i && (b[j-1] == ' ' || b[j-1] == '\n' || b[j-1] == '\r' || b[j-1] == '\t') {
		j--
	}
	return b[i:j]
}

func handleBootstrap(w http.ResponseWriter, r *http.Request, st store.Store, home string) {
	teams, _ := st.ListTeams(r.Context())
	initialTeam := ""
	if len(teams) > 0 {
		initialTeam = teams[0].Name
	}
	var tasks any = []any{}
	var agents any = []any{}
	var repos any = []any{}
	var workflows any = []any{}
	allowlist, _ := st.ListAllowedDomains(r.Context())
	if initialTeam != "" {
		if t, err := st.ListTasks(r.Context(), initialTeam, 0); err == nil {
			tasks = t
		}
		if a, err := st.ListAgents(r.Context(), initialTeam); err == nil {
			agents = a
		}
		if rp, err := st.ListRepos(r.Context(), initialTeam); err == nil {
			repos = rp
		}
		if wf, err := st.ListWorkflows(r.Context(), initialTeam); err == nil {
			workflows = wf
		}
	}
	writeJSON(w, map[string]any{
		"config": map[string]any{
			"human_name":   "human",
			"hc_home":      home,
			"bootstrap_id": getBootstrapID(home),
		},
		"teams":        teams,
		"initial_team": nilIfEmpty(initialTeam),
		"tasks":        tasks,
		"agents":       agents,
		"repos":        repos,
		"workflows":    workflows,
		"network": map[string]any{
			"allowlist": allowlist,
		},
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

// writeJSONError sends a JSON body {"error": "message"} with the given status code.
func writeJSONError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": message})
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
