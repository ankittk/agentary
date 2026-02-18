package runtime

import (
	"context"
	"testing"
	"time"
)

func TestStubRuntime_Name(t *testing.T) {
	var r StubRuntime
	if got := r.Name(); got != "stub" {
		t.Errorf("Name(): got %q, want stub", got)
	}
}

func TestStubRuntime_RunTurn(t *testing.T) {
	ctx := context.Background()
	var r StubRuntime
	events := 0
	emit := func(ev Event) {
		events++
		if ev.Team != "t1" || ev.Agent != "a1" {
			t.Errorf("event Team/Agent: got %q/%q", ev.Team, ev.Agent)
		}
	}
	req := TurnRequest{Team: "t1", Agent: "a1", Input: "hello"}
	result, err := r.RunTurn(ctx, req, emit)
	if err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if result.Output != "stub: ok" {
		t.Errorf("RunTurn Output: got %q", result.Output)
	}
	if events < 2 {
		t.Errorf("expected at least 2 events, got %d", events)
	}
}

func TestStubRuntime_RunTurn_contextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var r StubRuntime
	_, err := r.RunTurn(ctx, TurnRequest{Team: "t1", Agent: "a1", Input: "x"}, func(Event) {})
	if err != nil {
		t.Fatalf("RunTurn with cancelled context: %v", err)
	}
	// Stub may return quickly when ctx is done (sleep exits early)
	_ = time.Millisecond
}
