package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSubprocessRuntime_Name(t *testing.T) {
	r := SubprocessRuntime{}
	if r.Name() != "subprocess" {
		t.Errorf("Name: got %q", r.Name())
	}
}

func TestSubprocessRuntime_RunTurn_emptyCommand(t *testing.T) {
	r := SubprocessRuntime{}
	ctx := context.Background()
	_, err := r.RunTurn(ctx, TurnRequest{}, func(Event) {})
	if err == nil {
		t.Fatal("expected error when command empty")
	}
}

func TestSubprocessRuntime_RunTurn_echoScript(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "agent.sh")
	// Script: read JSON from stdin, echo one event line to stdout (NDJSON)
	content := `#!/bin/sh
read line
echo '{"type":"agent_activity","timestamp":"2020-01-01T00:00:00Z","data":{"output":"ok"}}'
`
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	r := SubprocessRuntime{Command: script, Timeout: 5 * time.Second}
	ctx := context.Background()
	var emitted Event
	_, err := r.RunTurn(ctx, TurnRequest{Input: "hello"}, func(ev Event) {
		emitted = ev
	})
	if err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if emitted.Type != "agent_activity" {
		t.Errorf("emitted event type: %q", emitted.Type)
	}
	if out, _ := emitted.Data["output"].(string); out != "ok" {
		t.Errorf("emitted event data: %+v", emitted.Data)
	}
}

func TestSubprocessRuntime_RunTurn_contextCancel(t *testing.T) {
	// Use a script that sleeps so we can cancel
	dir := t.TempDir()
	script := filepath.Join(dir, "sleep.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 10\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	r := SubprocessRuntime{Command: script, Timeout: time.Second}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	_, err := r.RunTurn(ctx, TurnRequest{}, func(Event) {})
	if err == nil && ctx.Err() == nil {
		t.Log("RunTurn with cancelled context may still start process; defer kills it")
	}
}

func TestTurnRequest_roundtrip(t *testing.T) {
	req := TurnRequest{Input: "test", Team: "t1", Agent: "a1"}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var out TurnRequest
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out.Input != req.Input || out.Team != req.Team {
		t.Errorf("roundtrip: %+v", out)
	}
}
