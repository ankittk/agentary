package capabilities

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegistry_RegisterGet(t *testing.T) {
	reg := NewRegistry()
	c := SlackWebhook{WebhookURL: "https://example.com"}
	reg.Register("slack", c)
	got := reg.Get("slack")
	if got != c {
		t.Fatalf("Get(slack): got %+v", got)
	}
	if reg.Get("nonexistent") != nil {
		t.Fatal("Get(nonexistent) should be nil")
	}
}

func TestSlackWebhook_Notify_mockHTTP(t *testing.T) {
	var received string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		received = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := SlackWebhook{WebhookURL: srv.URL}
	ctx := context.Background()
	if err := c.Notify(ctx, "hello"); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if received != "" {
		t.Logf("request received at %s", received)
	}
}

func TestSlackWebhook_Notify_emptyURL(t *testing.T) {
	c := SlackWebhook{}
	ctx := context.Background()
	if err := c.Notify(ctx, "msg"); err == nil {
		t.Fatal("expected error when webhook URL empty")
	}
}

func TestGitHubNotifier_Notify(t *testing.T) {
	g := GitHubNotifier{Token: "x", OwnerRepo: "owner/repo"}
	ctx := context.Background()
	if err := g.Notify(ctx, "msg"); err != nil {
		t.Fatalf("Notify: %v", err)
	}
}

func TestGitHubNotifier_Notify_missingConfig(t *testing.T) {
	g := GitHubNotifier{}
	ctx := context.Background()
	if err := g.Notify(ctx, "msg"); err == nil {
		t.Fatal("expected error when token or owner/repo not set")
	}
}
