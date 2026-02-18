package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestServerSmoke(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	app, err := NewApp(ServerOptions{Home: home, Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	ts := httptest.NewServer(app.Server.Handler)
	t.Cleanup(ts.Close)

	// health
	r1, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	if r1.StatusCode != 200 {
		t.Fatalf("/health status=%d", r1.StatusCode)
	}

	// create team
	resp, err := http.Post(ts.URL+"/teams", "application/json", strings.NewReader(`{"name":"t1"}`))
	if err != nil {
		t.Fatalf("POST /teams: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("POST /teams status=%d", resp.StatusCode)
	}

	// list teams
	r2, err := http.Get(ts.URL + "/teams")
	if err != nil {
		t.Fatalf("GET /teams: %v", err)
	}
	var teams []any
	if err := json.NewDecoder(r2.Body).Decode(&teams); err != nil {
		t.Fatalf("decode /teams: %v", err)
	}
	if len(teams) == 0 {
		t.Fatalf("expected teams")
	}

	// SSE should produce initial connected event quickly.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/stream", nil)
	sseResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /stream: %v", err)
	}
	defer func() { _ = sseResp.Body.Close() }()

	sc := bufio.NewScanner(sseResp.Body)
	found := false
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "data: ") && strings.Contains(line, `"type":"connected"`) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("did not see connected event")
	}

	// JSON error on not found
	r3, _ := http.Get(ts.URL + "/teams/nonexistent/tasks")
	if r3.StatusCode != 404 {
		t.Fatalf("GET /teams/nonexistent/tasks status=%d", r3.StatusCode)
	}
	var errBody struct{ Error string }
	if err := json.NewDecoder(r3.Body).Decode(&errBody); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if errBody.Error == "" {
		t.Fatalf("expected error message in JSON")
	}

	// create task, GET by id, PATCH
	idResp, _ := http.Post(ts.URL+"/teams/t1/tasks", "application/json", strings.NewReader(`{"title":"test task"}`))
	if idResp.StatusCode != 200 {
		t.Fatalf("POST task status=%d", idResp.StatusCode)
	}
	var createResp struct {
		TaskID int64 `json:"task_id"`
	}
	if err := json.NewDecoder(idResp.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode task_id: %v", err)
	}
	getOne, _ := http.Get(fmt.Sprintf("%s/teams/t1/tasks/%d", ts.URL, createResp.TaskID))
	if getOne.StatusCode != 200 {
		t.Fatalf("GET task by id status=%d", getOne.StatusCode)
	}
	var task map[string]any
	if err := json.NewDecoder(getOne.Body).Decode(&task); err != nil {
		t.Fatalf("decode task: %v", err)
	}
	if task["Title"] != "test task" || task["Status"] != "todo" {
		t.Fatalf("task: got %v", task)
	}
	patchReq, _ := http.NewRequest("PATCH", fmt.Sprintf("%s/teams/t1/tasks/%d", ts.URL, createResp.TaskID),
		strings.NewReader(`{"status":"in_progress","assignee":"alice"}`))
	patchResp, _ := http.DefaultClient.Do(patchReq)
	if patchResp.StatusCode != 200 {
		t.Fatalf("PATCH task status=%d", patchResp.StatusCode)
	}
	var updated map[string]any
	if err := json.NewDecoder(patchResp.Body).Decode(&updated); err != nil {
		t.Fatalf("decode updated task: %v", err)
	}
	if updated["Status"] != "in_progress" || updated["Assignee"] != "alice" {
		t.Fatalf("updated task: got %v", updated)
	}
}
