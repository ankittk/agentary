package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestIntegrationBootstrapConfigAgentsMessages exercises bootstrap, config, agents, and messages APIs
// against a real NewApp (SQLite store, SSE hub). Runs with unit tests; use -short to skip if needed.
func TestIntegrationBootstrapConfigAgentsMessages(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	app, err := NewApp(ServerOptions{Home: home, Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	ts := httptest.NewServer(app.Server.Handler)
	t.Cleanup(ts.Close)

	// Bootstrap (teams + agents + workflows)
	resp, err := http.Get(ts.URL + "/bootstrap")
	if err != nil {
		t.Fatalf("GET /bootstrap: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /bootstrap status=%d", resp.StatusCode)
	}
	var bootstrap map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&bootstrap); err != nil {
		t.Fatalf("decode /bootstrap: %v", err)
	}
	if bootstrap["teams"] == nil {
		t.Fatal("bootstrap missing teams")
	}

	// Config
	resp2, err := http.Get(ts.URL + "/config")
	if err != nil {
		t.Fatalf("GET /config: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("GET /config status=%d", resp2.StatusCode)
	}
	var config map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&config); err != nil {
		t.Fatalf("decode /config: %v", err)
	}

	// Ensure we have a team (from SeedDemo) and list agents
	_, _ = http.Post(ts.URL+"/teams", "application/json", strings.NewReader(`{"name":"int-team"}`))
	agentsResp, err := http.Get(ts.URL + "/teams/int-team/agents")
	if err != nil {
		t.Fatalf("GET /teams/int-team/agents: %v", err)
	}
	if agentsResp.StatusCode != http.StatusOK {
		t.Fatalf("GET agents status=%d", agentsResp.StatusCode)
	}
	var agents []any
	if err := json.NewDecoder(agentsResp.Body).Decode(&agents); err != nil {
		t.Fatalf("decode agents: %v", err)
	}

	// Messages: send and list
	sendBody := `{"recipient":"bob","content":"hello from integration"}`
	sendResp, err := http.Post(ts.URL+"/teams/int-team/messages", "application/json", strings.NewReader(sendBody))
	if err != nil {
		t.Fatalf("POST /teams/int-team/messages: %v", err)
	}
	if sendResp.StatusCode != http.StatusOK {
		t.Fatalf("POST messages status=%d", sendResp.StatusCode)
	}
	msgsResp, err := http.Get(ts.URL + "/teams/int-team/messages?recipient=bob")
	if err != nil {
		t.Fatalf("GET /teams/int-team/messages: %v", err)
	}
	if msgsResp.StatusCode != http.StatusOK {
		t.Fatalf("GET messages status=%d", msgsResp.StatusCode)
	}
	var msgs []map[string]any
	if err := json.NewDecoder(msgsResp.Body).Decode(&msgs); err != nil {
		t.Fatalf("decode messages: %v", err)
	}
	if len(msgs) < 1 {
		t.Fatalf("expected at least one message for bob, got %d", len(msgs))
	}
}
