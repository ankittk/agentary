package otel

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	otelglobal "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

const meterName = "github.com/ankittk/agentary"

// InitMeterProvider initializes the global MeterProvider with a Prometheus exporter
// and returns an http.Handler that serves /metrics. Call once at daemon startup.
// If init fails, returns (nil, err); caller can fall back to non-OTel /metrics.
func InitMeterProvider(ctx context.Context, serviceName string) (http.Handler, error) {
	if serviceName == "" {
		serviceName = "agentary"
	}
	reg := prometheus.NewRegistry()
	exporter, err := otelprom.New(otelprom.WithRegisterer(reg))
	if err != nil {
		return nil, err
	}
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
		sdkmetric.WithResource(res),
	)
	otelglobal.SetMeterProvider(provider)
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{EnableOpenMetrics: true}), nil
}

// Meter returns the global meter for agentary (after InitMeterProvider).
func Meter() metric.Meter {
	return otelglobal.Meter(meterName)
}

// Common attribute keys for metrics.
var (
	AttrTeam   = attribute.Key("team")
	AttrStatus = attribute.Key("status")
	AttrStage  = attribute.Key("stage")
	AttrAgent  = attribute.Key("agent")
	AttrRoute  = attribute.Key("http.route")
)
