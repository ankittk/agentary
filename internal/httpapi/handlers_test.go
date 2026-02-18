package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHandlers exercises many server routes to improve coverage of server.go.
func TestHandlers(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	app, err := NewApp(ServerOptions{Home: home, Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	ts := httptest.NewServer(app.Server.Handler)
	t.Cleanup(ts.Close)

	// POST team with empty name
	resp, _ := http.Post(ts.URL+"/teams", "application/json", strings.NewReader(`{"name":""}`))
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if resp != nil && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST /teams empty name: status=%d", resp.StatusCode)
	}

	// Create team and task for sub-routes
	createTeamResp, _ := http.Post(ts.URL+"/teams", "application/json", strings.NewReader(`{"name":"h1"}`))
	if createTeamResp != nil {
		_ = createTeamResp.Body.Close()
	}
	taskResp, _ := http.Post(ts.URL+"/teams/h1/tasks", "application/json", strings.NewReader(`{"title":"t1"}`))
	if taskResp != nil {
		defer func() { _ = taskResp.Body.Close() }()
	}
	var taskBody struct {
		TaskID int64 `json:"task_id"`
	}
	_ = json.NewDecoder(taskResp.Body).Decode(&taskBody)
	taskID := taskBody.TaskID
	if taskID == 0 {
		t.Fatal("expected non-zero task_id from POST task")
	}

	// GET/POST agents
	agentsResp, _ := http.Get(ts.URL + "/teams/h1/agents")
	if agentsResp != nil {
		defer func() { _ = agentsResp.Body.Close() }()
	}
	if agentsResp != nil && agentsResp.StatusCode != http.StatusOK {
		t.Fatalf("GET agents: %d", agentsResp.StatusCode)
	}
	postAgentsResp, _ := http.Post(ts.URL+"/teams/h1/agents", "application/json", strings.NewReader(`{"name":"a1","role":"eng"}`))
	if postAgentsResp != nil {
		_ = postAgentsResp.Body.Close()
	}

	// Charter GET/PUT
	charterGet, _ := http.Get(ts.URL + "/teams/h1/charter")
	if charterGet != nil {
		defer func() { _ = charterGet.Body.Close() }()
	}
	if charterGet != nil && charterGet.StatusCode != http.StatusOK {
		t.Fatalf("GET charter: %d", charterGet.StatusCode)
	}
	putReq, _ := http.NewRequest(http.MethodPut, ts.URL+"/teams/h1/charter", strings.NewReader(`{"content":"# Charter"}`))
	putReq.Header.Set("Content-Type", "application/json")
	putResp, _ := http.DefaultClient.Do(putReq)
	if putResp != nil {
		defer func() { _ = putResp.Body.Close() }()
	}
	if putResp != nil && putResp.StatusCode != http.StatusOK {
		t.Fatalf("PUT charter: %d", putResp.StatusCode)
	}

	// Repos GET/POST
	reposResp, _ := http.Get(ts.URL + "/teams/h1/repos")
	if reposResp != nil {
		defer func() { _ = reposResp.Body.Close() }()
	}
	if reposResp != nil && reposResp.StatusCode != http.StatusOK {
		t.Fatalf("GET repos: %d", reposResp.StatusCode)
	}
	postReposResp, _ := http.Post(ts.URL+"/teams/h1/repos", "application/json", strings.NewReader(`{"name":"r1","source":"/tmp","approval":"manual"}`))
	if postReposResp != nil {
		_ = postReposResp.Body.Close()
	}

	// Workflows GET/POST, init
	wfResp, _ := http.Get(ts.URL + "/teams/h1/workflows")
	if wfResp != nil {
		defer func() { _ = wfResp.Body.Close() }()
	}
	if wfResp != nil && wfResp.StatusCode != http.StatusOK {
		t.Fatalf("GET workflows: %d", wfResp.StatusCode)
	}
	wfPostResp, _ := http.Post(ts.URL+"/teams/h1/workflows", "application/json", strings.NewReader(`{"name":"w1","version":1,"source":"builtin:w1"}`))
	if wfPostResp != nil {
		_ = wfPostResp.Body.Close()
	}
	wfInitResp, _ := http.Post(ts.URL+"/teams/h1/workflows/init", "application/json", nil)
	if wfInitResp != nil {
		_ = wfInitResp.Body.Close()
	}

	// Task comments GET/POST
	commentsResp, _ := http.Get(fmt.Sprintf("%s/teams/h1/tasks/%d/comments", ts.URL, taskID))
	if commentsResp.StatusCode != http.StatusOK {
		t.Fatalf("GET comments: %d", commentsResp.StatusCode)
	}
	_, _ = http.Post(fmt.Sprintf("%s/teams/h1/tasks/%d/comments", ts.URL, taskID), "application/json", strings.NewReader(`{"body":"c1"}`))

	// Task attachments GET/POST/DELETE
	attResp, _ := http.Get(fmt.Sprintf("%s/teams/h1/tasks/%d/attachments", ts.URL, taskID))
	if attResp.StatusCode != http.StatusOK {
		t.Fatalf("GET attachments: %d", attResp.StatusCode)
	}
	_, _ = http.Post(fmt.Sprintf("%s/teams/h1/tasks/%d/attachments", ts.URL, taskID), "application/json", strings.NewReader(`{"file_path":"/f"}`))
	delReq, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/teams/h1/tasks/%d/attachments?file_path=/f", ts.URL, taskID), nil)
	_, _ = http.DefaultClient.Do(delReq)

	// Task dependencies GET/POST
	taskID2Resp, _ := http.Post(ts.URL+"/teams/h1/tasks", "application/json", strings.NewReader(`{"title":"t2"}`))
	var t2 struct {
		TaskID int64 `json:"task_id"`
	}
	_ = json.NewDecoder(taskID2Resp.Body).Decode(&t2)
	depsResp, _ := http.Get(fmt.Sprintf("%s/teams/h1/tasks/%d/dependencies", ts.URL, taskID))
	if depsResp.StatusCode != http.StatusOK {
		t.Fatalf("GET dependencies: %d", depsResp.StatusCode)
	}
	_, _ = http.Post(fmt.Sprintf("%s/teams/h1/tasks/%d/dependencies", ts.URL, taskID), "application/json", bytes.NewReader([]byte(fmt.Sprintf(`{"depends_on_task_id":%d}`, t2.TaskID))))

	// Task diff (may 500 if git not available in path)
	diffResp, _ := http.Get(fmt.Sprintf("%s/teams/h1/tasks/%d/diff", ts.URL, taskID))
	_ = diffResp.StatusCode // 200 or 500

	// Task reviews GET
	revResp, _ := http.Get(fmt.Sprintf("%s/teams/h1/tasks/%d/reviews", ts.URL, taskID))
	if revResp.StatusCode != http.StatusOK {
		t.Fatalf("GET reviews: %d", revResp.StatusCode)
	}

	// Network GET, reset, allow, disallow
	netResp, _ := http.Get(ts.URL + "/network")
	if netResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /network: %d", netResp.StatusCode)
	}
	_, _ = http.Post(ts.URL+"/network/reset", "application/json", nil)
	_, _ = http.Post(ts.URL+"/network/allow", "application/json", strings.NewReader(`{"domain":"test.com"}`))
	_, _ = http.Post(ts.URL+"/network/disallow", "application/json", strings.NewReader(`{"domain":"test.com"}`))

	// GET /config and GET /metrics (legacy handler when MetricsHandler not set)
	configResp, _ := http.Get(ts.URL + "/config")
	if configResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /config: %d", configResp.StatusCode)
	}
	metricsResp, _ := http.Get(ts.URL + "/metrics")
	if metricsResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /metrics: %d", metricsResp.StatusCode)
	}
	teamsListResp, _ := http.Get(ts.URL + "/teams")
	if teamsListResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /teams: %d", teamsListResp.StatusCode)
	}

	// PATCH task invalid status
	patchBad, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/teams/h1/tasks/%d", ts.URL, taskID), strings.NewReader(`{"status":"invalid"}`))
	patchBad.Header.Set("Content-Type", "application/json")
	badResp, _ := http.DefaultClient.Do(patchBad)
	if badResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("PATCH invalid status: %d", badResp.StatusCode)
	}

	// POST attachments without file_path
	attBad, _ := http.Post(fmt.Sprintf("%s/teams/h1/tasks/%d/attachments", ts.URL, taskID), "application/json", strings.NewReader(`{}`))
	if attBad.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST attachments no file_path: %d", attBad.StatusCode)
	}

	// POST dependencies with invalid depends_on_task_id
	depBad, _ := http.Post(fmt.Sprintf("%s/teams/h1/tasks/%d/dependencies", ts.URL, taskID), "application/json", strings.NewReader(`{"depends_on_task_id":0}`))
	if depBad.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST dependencies invalid id: %d", depBad.StatusCode)
	}

	// GET single task
	getTask, _ := http.Get(fmt.Sprintf("%s/teams/h1/tasks/%d", ts.URL, taskID))
	if getTask.StatusCode != http.StatusOK {
		t.Fatalf("GET task: %d", getTask.StatusCode)
	}

	// request-review and approve (workflow task via API: init default workflow then create task)
	_, _ = http.Post(ts.URL+"/teams", "application/json", strings.NewReader(`{"name":"wfteam"}`))
	_, _ = http.Post(ts.URL+"/teams/wfteam/workflows/init", "application/json", nil)
	wfTaskResp, _ := http.Post(ts.URL+"/teams/wfteam/tasks", "application/json", strings.NewReader(`{"title":"wf task"}`))
	var wfTaskBody struct{ TaskID int64 `json:"task_id"` }
	_ = json.NewDecoder(wfTaskResp.Body).Decode(&wfTaskBody)
	wfTaskID := wfTaskBody.TaskID
	if wfTaskID > 0 {
		reqRevResp, _ := http.Post(fmt.Sprintf("%s/teams/wfteam/tasks/%d/request-review", ts.URL, wfTaskID), "application/json", nil)
		_ = reqRevResp.StatusCode
		approveResp, _ := http.Post(fmt.Sprintf("%s/teams/wfteam/tasks/%d/approve", ts.URL, wfTaskID), "application/json", strings.NewReader(`{"outcome":"approved"}`))
		_ = approveResp.StatusCode
	}

	// Method not allowed
	postHealth, _ := http.Post(ts.URL+"/health", "application/json", nil)
	_ = postHealth.StatusCode

	// submit-review endpoint
	submitRevResp, _ := http.Post(fmt.Sprintf("%s/teams/h1/tasks/%d/submit-review", ts.URL, taskID), "application/json", strings.NewReader(`{"reviewer_agent":"a1","outcome":"approved","comments":"ok"}`))
	if submitRevResp != nil {
		defer func() { _ = submitRevResp.Body.Close() }()
	}
	_ = submitRevResp.StatusCode // 200 or 400 if task has no workflow

	// GET /bootstrap full
	bootstrapResp, _ := http.Get(ts.URL + "/bootstrap")
	if bootstrapResp != nil {
		defer func() { _ = bootstrapResp.Body.Close() }()
	}
	if bootstrapResp != nil && bootstrapResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /bootstrap: %d", bootstrapResp.StatusCode)
	}
	var bootstrap map[string]any
	if bootstrapResp != nil {
		_ = json.NewDecoder(bootstrapResp.Body).Decode(&bootstrap)
	}
	if bootstrap != nil && bootstrap["teams"] == nil {
		t.Error("bootstrap should have teams")
	}
}

func TestAPIKeyMiddleware(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	app, err := NewApp(ServerOptions{Home: home, Addr: ":0", APIKey: "secret"})
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	ts := httptest.NewServer(app.Server.Handler)
	t.Cleanup(ts.Close)

	// /health and /metrics exempt
	healthResp, _ := http.Get(ts.URL + "/health")
	if healthResp != nil {
		defer func() { _ = healthResp.Body.Close() }()
	}
	if healthResp != nil && healthResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health without key: %d", healthResp.StatusCode)
	}
	metricsResp, _ := http.Get(ts.URL + "/metrics")
	if metricsResp != nil {
		defer func() { _ = metricsResp.Body.Close() }()
	}
	if metricsResp != nil && metricsResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /metrics without key: %d", metricsResp.StatusCode)
	}

	// /teams without key -> 401
	teamsResp, _ := http.Get(ts.URL + "/teams")
	if teamsResp != nil {
		defer func() { _ = teamsResp.Body.Close() }()
	}
	if teamsResp != nil && teamsResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /teams without key: %d", teamsResp.StatusCode)
	}

	// /teams with X-API-Key -> 200
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/teams", nil)
	req.Header.Set("X-API-Key", "secret")
	resp, _ := http.DefaultClient.Do(req)
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if resp != nil && resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /teams with key: %d", resp.StatusCode)
	}

	// /teams with query api_key -> 200
	req2, _ := http.NewRequest(http.MethodGet, ts.URL+"/teams?api_key=secret", nil)
	resp2, _ := http.DefaultClient.Do(req2)
	if resp2 != nil {
		defer func() { _ = resp2.Body.Close() }()
	}
	if resp2 != nil && resp2.StatusCode != http.StatusOK {
		t.Fatalf("GET /teams with api_key query: %d", resp2.StatusCode)
	}

	// invalid key -> 401
	req3, _ := http.NewRequest(http.MethodGet, ts.URL+"/teams", nil)
	req3.Header.Set("X-API-Key", "wrong")
	resp3, _ := http.DefaultClient.Do(req3)
	if resp3 != nil {
		defer func() { _ = resp3.Body.Close() }()
	}
	if resp3 != nil && resp3.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /teams with wrong key: %d", resp3.StatusCode)
	}
}

func TestDeleteTeam_cascade(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	app, err := NewApp(ServerOptions{Home: home, Addr: ":0"})
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	ts := httptest.NewServer(app.Server.Handler)
	t.Cleanup(ts.Close)
	ctx := context.Background()

	// Create team and task
	_, _ = http.Post(ts.URL+"/teams", "application/json", strings.NewReader(`{"name":"cascade"}`))
	_, _ = http.Post(ts.URL+"/teams/cascade/tasks", "application/json", strings.NewReader(`{"title":"t1"}`))
	teamsBefore, _ := app.Store.ListTeams(ctx)
	var found bool
	for _, tt := range teamsBefore {
		if tt.Name == "cascade" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("team cascade not created")
	}
	// DELETE team
	delReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/teams/cascade", nil)
	delResp, _ := http.DefaultClient.Do(delReq)
	if delResp != nil {
		defer func() { _ = delResp.Body.Close() }()
	}
	if delResp != nil && delResp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE /teams/cascade: %d", delResp.StatusCode)
	}
	teamsAfter, _ := app.Store.ListTeams(ctx)
	for _, tt := range teamsAfter {
		if tt.Name == "cascade" {
			t.Fatal("team cascade should be deleted")
		}
	}
}
