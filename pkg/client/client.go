// Package client provides a Go SDK for the Agentary HTTP API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ankittk/agentary/pkg/models"
)

// Client calls the Agentary HTTP API. It is safe for concurrent use.
type Client struct {
	BaseURL    string       // e.g. "http://localhost:3548"
	APIKey     string       // optional; set for X-API-Key / api_key
	HTTPClient *http.Client // optional; nil uses http.DefaultClient
}

// New returns a client for the given base URL (e.g. "http://localhost:3548").
// APIKey is optional; when set, requests use X-API-Key header and optionally api_key query.
func New(baseURL, apiKey string) *Client {
	return &Client{BaseURL: baseURL, APIKey: apiKey}
}

func (c *Client) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}
	u := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}
	return c.client().Do(req)
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	resp, err := c.do(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		if errBody.Error != "" {
			return fmt.Errorf("api %s %s: %s", method, path, errBody.Error)
		}
		return fmt.Errorf("api %s %s: status %d", method, path, resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// Health returns the /health response (ok: true).
func (c *Client) Health(ctx context.Context) (ok bool, err error) {
	var out struct {
		OK bool `json:"ok"`
	}
	err = c.doJSON(ctx, http.MethodGet, "/health", nil, &out)
	return out.OK, err
}

// Config returns the /config response.
func (c *Client) Config(ctx context.Context) (*models.Config, error) {
	var out models.Config
	err := c.doJSON(ctx, http.MethodGet, "/config", nil, &out)
	return &out, err
}

// Bootstrap returns the full /bootstrap payload.
func (c *Client) Bootstrap(ctx context.Context) (*models.Bootstrap, error) {
	var out models.Bootstrap
	err := c.doJSON(ctx, http.MethodGet, "/bootstrap", nil, &out)
	return &out, err
}

// ListTeams returns all teams.
func (c *Client) ListTeams(ctx context.Context) ([]models.Team, error) {
	var out []models.Team
	err := c.doJSON(ctx, http.MethodGet, "/teams", nil, &out)
	return out, err
}

// CreateTeam creates a team and returns it.
func (c *Client) CreateTeam(ctx context.Context, name string) (*models.Team, error) {
	var out models.Team
	err := c.doJSON(ctx, http.MethodPost, "/teams", map[string]string{"name": name}, &out)
	return &out, err
}

// DeleteTeam deletes a team by name.
func (c *Client) DeleteTeam(ctx context.Context, team string) error {
	return c.doJSON(ctx, http.MethodDelete, "/teams/"+url.PathEscape(team), nil, nil)
}

// ListTasks returns tasks for a team (limit 0 = default).
func (c *Client) ListTasks(ctx context.Context, team string, limit int) ([]models.Task, error) {
	path := "/teams/" + url.PathEscape(team) + "/tasks"
	if limit > 0 {
		path += "?limit=" + strconv.Itoa(limit)
	}
	var out []models.Task
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

// CreateTask creates a task and returns the task_id.
func (c *Client) CreateTask(ctx context.Context, team, title, status string) (taskID int64, err error) {
	body := map[string]string{"title": title}
	if status != "" {
		body["status"] = status
	}
	var out struct {
		TaskID int64 `json:"task_id"`
	}
	err = c.doJSON(ctx, http.MethodPost, "/teams/"+url.PathEscape(team)+"/tasks", body, &out)
	return out.TaskID, err
}

// GetTask returns a task by team and ID.
func (c *Client) GetTask(ctx context.Context, team string, taskID int64) (*models.Task, error) {
	path := "/teams/" + url.PathEscape(team) + "/tasks/" + strconv.FormatInt(taskID, 10)
	var out models.Task
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)
	return &out, err
}

// UpdateTask updates a task's status and/or assignee (pass empty string to leave unchanged, nil assignee to clear).
func (c *Client) UpdateTask(ctx context.Context, team string, taskID int64, status string, assignee *string) error {
	body := make(map[string]any)
	if status != "" {
		body["status"] = status
	}
	if assignee != nil {
		body["assignee"] = *assignee
	}
	return c.doJSON(ctx, http.MethodPatch, "/teams/"+url.PathEscape(team)+"/tasks/"+strconv.FormatInt(taskID, 10), body, nil)
}

// ListAgents returns agents for a team.
func (c *Client) ListAgents(ctx context.Context, team string) ([]models.Agent, error) {
	var out []models.Agent
	err := c.doJSON(ctx, http.MethodGet, "/teams/"+url.PathEscape(team)+"/agents", nil, &out)
	return out, err
}

// CreateAgent creates an agent (role defaults to "engineer" if empty).
func (c *Client) CreateAgent(ctx context.Context, team, name, role string) error {
	if role == "" {
		role = "engineer"
	}
	return c.doJSON(ctx, http.MethodPost, "/teams/"+url.PathEscape(team)+"/agents", map[string]string{"name": name, "role": role}, nil)
}

// ListMessages returns messages for a team and recipient (inbox).
func (c *Client) ListMessages(ctx context.Context, team, recipient string, limit int) ([]models.Message, error) {
	path := "/teams/" + url.PathEscape(team) + "/messages?recipient=" + url.QueryEscape(recipient)
	if limit > 0 {
		path += "&limit=" + strconv.Itoa(limit)
	}
	var out []models.Message
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

// CreateMessage sends a message.
func (c *Client) CreateMessage(ctx context.Context, team, sender, recipient, content string) (messageID int64, err error) {
	var out struct {
		MessageID int64 `json:"message_id"`
	}
	err = c.doJSON(ctx, http.MethodPost, "/teams/"+url.PathEscape(team)+"/messages", map[string]string{
		"sender": sender, "recipient": recipient, "content": content,
	}, &out)
	return out.MessageID, err
}

// NetworkAllowlist returns the current allowlist (GET /network).
func (c *Client) NetworkAllowlist(ctx context.Context) ([]string, error) {
	var out struct {
		Allowlist []string `json:"allowlist"`
	}
	err := c.doJSON(ctx, http.MethodGet, "/network", nil, &out)
	return out.Allowlist, err
}

// NetworkAllow adds a domain to the allowlist.
func (c *Client) NetworkAllow(ctx context.Context, domain string) error {
	return c.doJSON(ctx, http.MethodPost, "/network/allow", map[string]string{"domain": domain}, nil)
}

// NetworkDisallow removes a domain from the allowlist.
func (c *Client) NetworkDisallow(ctx context.Context, domain string) error {
	return c.doJSON(ctx, http.MethodPost, "/network/disallow", map[string]string{"domain": domain}, nil)
}

// NetworkReset resets the allowlist.
func (c *Client) NetworkReset(ctx context.Context) error {
	return c.doJSON(ctx, http.MethodPost, "/network/reset", nil, nil)
}
