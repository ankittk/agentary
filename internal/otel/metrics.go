package otel

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	initMetricsOnce sync.Once
	taskOpsCounter  metric.Int64Counter
	agentTurnsCounter metric.Int64Counter
	agentTurnDuration metric.Float64Histogram
	workflowTurnDuration metric.Float64Histogram
	sseConnectionsGauge metric.Int64ObservableGauge
	sseEventsCounter   metric.Int64Counter
	sseConnections     int64
	sseConnectionsMu   sync.Mutex
)

// InitMetrics creates the meter instruments. Safe to call multiple times; only runs once.
// Call after InitMeterProvider.
func InitMetrics(ctx context.Context) error {
	var err error
	initMetricsOnce.Do(func() {
		m := Meter()
		taskOpsCounter, err = m.Int64Counter("agentary_task_operations_total", metric.WithDescription("Total task operations (create, update, claim, etc.)"))
		if err != nil {
			return
		}
		agentTurnsCounter, err = m.Int64Counter("agentary_agent_turns_total", metric.WithDescription("Total agent turns executed"))
		if err != nil {
			return
		}
		agentTurnDuration, err = m.Float64Histogram("agentary_agent_turn_duration_seconds", metric.WithDescription("Agent turn duration in seconds"))
		if err != nil {
			return
		}
		workflowTurnDuration, err = m.Float64Histogram("agentary_workflow_turn_duration_seconds", metric.WithDescription("Workflow turn duration in seconds"))
		if err != nil {
			return
		}
		sseEventsCounter, err = m.Int64Counter("agentary_sse_events_total", metric.WithDescription("Total SSE events published"))
		if err != nil {
			return
		}
		sseConnectionsGauge, err = m.Int64ObservableGauge("agentary_sse_connections", metric.WithDescription("Current SSE subscriber count"))
		if err != nil {
			return
		}
		_, err = m.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
			sseConnectionsMu.Lock()
			n := sseConnections
			sseConnectionsMu.Unlock()
			o.ObserveInt64(sseConnectionsGauge, n)
			return nil
		}, sseConnectionsGauge)
		if err != nil {
			return
		}
	})
	return err
}

// RecordTaskOp records a task operation (create, update, claim, etc.).
func RecordTaskOp(ctx context.Context, op string, team string, status string) {
	if taskOpsCounter == nil {
		return
	}
	taskOpsCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("operation", op),
		AttrTeam.String(team),
		AttrStatus.String(status),
	))
}

// RecordAgentTurn records an agent turn and its duration.
func RecordAgentTurn(ctx context.Context, team, agent string, duration time.Duration) {
	if agentTurnsCounter != nil {
		agentTurnsCounter.Add(ctx, 1, metric.WithAttributes(AttrTeam.String(team), AttrAgent.String(agent)))
	}
	if agentTurnDuration != nil {
		agentTurnDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(AttrTeam.String(team), AttrAgent.String(agent)))
	}
}

// RecordWorkflowTurn records a workflow turn duration.
func RecordWorkflowTurn(ctx context.Context, team, stage string, duration time.Duration) {
	if workflowTurnDuration != nil {
		workflowTurnDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(AttrTeam.String(team), AttrStage.String(stage)))
	}
}

// RecordSSEEvent records one SSE event published.
func RecordSSEEvent(ctx context.Context) {
	if sseEventsCounter != nil {
		sseEventsCounter.Add(ctx, 1)
	}
}

// AddSSEConnection adds 1 to the SSE connection gauge (call on subscribe).
func AddSSEConnection() {
	sseConnectionsMu.Lock()
	sseConnections++
	sseConnectionsMu.Unlock()
}

// RemoveSSEConnection subtracts 1 from the SSE connection gauge (call on unsubscribe).
func RemoveSSEConnection() {
	sseConnectionsMu.Lock()
	sseConnections--
	if sseConnections < 0 {
		sseConnections = 0
	}
	sseConnectionsMu.Unlock()
}

// TaskCountFunc returns (todo, in_progress, done, failed) counts. Used for agentary_tasks_total gauge.
type TaskCountFunc func() (todo, inProgress, done, failed int64)

// InitMetricsWithTaskCount creates instruments and optionally registers a callback for task gauges.
// Call after InitMeterProvider. If taskCount is nil, task gauges are not reported.
func InitMetricsWithTaskCount(ctx context.Context, taskCount TaskCountFunc) error {
	if err := InitMetrics(ctx); err != nil {
		return err
	}
	if taskCount == nil {
		return nil
	}
	m := Meter()
	tasksGauge, err := m.Float64ObservableGauge("agentary_tasks_total", metric.WithDescription("Number of tasks by status"))
	if err != nil {
		return err
	}
	_, err = m.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		todo, inProgress, done, failed := taskCount()
		o.ObserveFloat64(tasksGauge, float64(todo), metric.WithAttributes(AttrStatus.String("todo")))
		o.ObserveFloat64(tasksGauge, float64(inProgress), metric.WithAttributes(AttrStatus.String("in_progress")))
		o.ObserveFloat64(tasksGauge, float64(done), metric.WithAttributes(AttrStatus.String("done")))
		o.ObserveFloat64(tasksGauge, float64(failed), metric.WithAttributes(AttrStatus.String("failed")))
		return nil
	}, tasksGauge)
	return err
}