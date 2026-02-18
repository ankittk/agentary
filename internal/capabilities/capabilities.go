package capabilities

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// Capability is an integration that can notify or post (e.g. Slack, GitHub).
type Capability interface {
	Name() string
	// Notify sends a message to the default target (e.g. Slack channel, GitHub repo).
	Notify(ctx context.Context, message string) error
}

// Registry holds loaded capabilities by name.
type Registry struct {
	mu   sync.RWMutex
	caps map[string]Capability
}

func NewRegistry() *Registry {
	return &Registry{caps: make(map[string]Capability)}
}

func (r *Registry) Register(name string, c Capability) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.caps[name] = c
}

func (r *Registry) Get(name string) Capability {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.caps[name]
}

func (r *Registry) Notify(ctx context.Context, name, message string) error {
	c := r.Get(name)
	if c == nil {
		return fmt.Errorf("capability %q not found", name)
	}
	return c.Notify(ctx, message)
}

// SlackWebhook sends messages to a Slack channel via incoming webhook URL.
type SlackWebhook struct {
	WebhookURL string
	Channel    string // optional override
	Username   string // optional
}

func (s SlackWebhook) Name() string { return "slack" }

func (s SlackWebhook) Notify(ctx context.Context, message string) error {
	if s.WebhookURL == "" {
		return fmt.Errorf("slack webhook URL not set")
	}
	payload := map[string]any{"text": message}
	if s.Channel != "" {
		payload["channel"] = s.Channel
	}
	if s.Username != "" {
		payload["username"] = s.Username
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.WebhookURL, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned %d", resp.StatusCode)
	}
	return nil
}

// GitHubNotifier can create issues or comment (stub for now; requires token and API calls).
type GitHubNotifier struct {
	Token     string
	OwnerRepo string // e.g. "owner/repo"
}

func (g GitHubNotifier) Name() string { return "github" }

func (g GitHubNotifier) Notify(ctx context.Context, message string) error {
	if g.Token == "" || g.OwnerRepo == "" {
		return fmt.Errorf("github token or owner/repo not set")
	}
	// Stub: could POST to GitHub API to create an issue or comment.
	_ = ctx
	_ = message
	return nil
}
