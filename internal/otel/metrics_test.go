package otel

import (
	"context"
	"testing"
	"time"
)

func TestInitMetrics_RecordTaskOp(t *testing.T) {
	ctx := context.Background()
	_, err := InitMeterProvider(ctx, "metrics-test")
	if err != nil {
		t.Fatalf("InitMeterProvider: %v", err)
	}
	if err := InitMetrics(ctx); err != nil {
		t.Fatalf("InitMetrics: %v", err)
	}
	RecordTaskOp(ctx, "create", "team1", "todo")
	RecordTaskOp(ctx, "claim", "team1", "in_progress")
}

func TestAddSSEConnection_RemoveSSEConnection(t *testing.T) {
	AddSSEConnection()
	AddSSEConnection()
	RemoveSSEConnection()
	RemoveSSEConnection()
	RemoveSSEConnection() // should not go negative
}

func TestRecordAgentTurn_RecordWorkflowTurn_RecordSSEEvent(t *testing.T) {
	ctx := context.Background()
	_, _ = InitMeterProvider(ctx, "record-test")
	_ = InitMetrics(ctx)
	RecordAgentTurn(ctx, "t1", "a1", 100*time.Millisecond)
	RecordWorkflowTurn(ctx, "t1", "dev", 50*time.Millisecond)
	RecordSSEEvent(ctx)
}

func TestInitMetricsWithTaskCount(t *testing.T) {
	ctx := context.Background()
	_, _ = InitMeterProvider(ctx, "taskcount-test")
	err := InitMetricsWithTaskCount(ctx, func() (todo, inProgress, done, failed int64) {
		return 1, 2, 3, 0
	})
	if err != nil {
		t.Fatalf("InitMetricsWithTaskCount: %v", err)
	}
}

func TestInitMetricsWithTaskCount_nilFunc(t *testing.T) {
	ctx := context.Background()
	_, _ = InitMeterProvider(ctx, "taskcount-nil-test")
	err := InitMetricsWithTaskCount(ctx, nil)
	if err != nil {
		t.Fatalf("InitMetricsWithTaskCount(nil): %v", err)
	}
}
