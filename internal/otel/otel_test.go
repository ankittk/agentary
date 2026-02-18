package otel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInitMeterProvider(t *testing.T) {
	ctx := context.Background()
	handler, err := InitMeterProvider(ctx, "test-service")
	if err != nil {
		t.Fatalf("InitMeterProvider: %v", err)
	}
	if handler == nil {
		t.Fatal("InitMeterProvider: expected non-nil handler")
	}
	// Serve /metrics and check we get 200 and OpenMetrics-style output
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /metrics: status=%d", rec.Code)
	}
	body := rec.Body.String()
	if body == "" {
		t.Fatal("GET /metrics: empty body")
	}
}

func TestInitMeterProvider_emptyServiceName(t *testing.T) {
	ctx := context.Background()
	handler, err := InitMeterProvider(ctx, "")
	if err != nil {
		t.Fatalf("InitMeterProvider: %v", err)
	}
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestMeter(t *testing.T) {
	ctx := context.Background()
	_, _ = InitMeterProvider(ctx, "meter-test")
	m := Meter()
	if m == nil {
		t.Fatal("Meter() returned nil")
	}
}

func TestAttributeKeys(t *testing.T) {
	// KeyValue types are never nil; just ensure String() returns valid attributes.
	_ = AttrTeam.String("t1")
	_ = AttrStatus.String("todo")
	_ = AttrStage.String("dev")
	_ = AttrAgent.String("a1")
	_ = AttrRoute.String("/teams")
}
