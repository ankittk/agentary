package runtime

import (
	"context"
	"time"
)

// StubRuntime is a deterministic local runtime that emits plausible events
// without calling any external LLM or spawning subprocesses.
type StubRuntime struct{}

func (StubRuntime) Name() string { return "stub" }

func (StubRuntime) RunTurn(ctx context.Context, req TurnRequest, emit func(Event)) (TurnResult, error) {
	now := time.Now().UTC()
	emit(Event{
		Type:      "turn_started",
		Team:      req.Team,
		Agent:     req.Agent,
		TaskID:    req.TaskID,
		Timestamp: now,
		Data: map[string]any{
			"sender": "system",
		},
	})

	// Simulate some tool activity.
	sleep(ctx, 150*time.Millisecond)
	emit(Event{
		Type:      "agent_activity",
		Team:      req.Team,
		Agent:     req.Agent,
		TaskID:    req.TaskID,
		Timestamp: time.Now().UTC(),
		Data: map[string]any{
			"tool":    "think",
			"summary": "Stub runtime simulated a turn",
		},
	})

	sleep(ctx, 150*time.Millisecond)
	emit(Event{
		Type:      "turn_ended",
		Team:      req.Team,
		Agent:     req.Agent,
		TaskID:    req.TaskID,
		Timestamp: time.Now().UTC(),
	})

	return TurnResult{Output: "stub: ok"}, nil
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return
	case <-t.C:
		return
	}
}
