// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package tracesetup

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/internal/version"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
)

// Service wraps the provider to implement node.Lifecycle.
type Service struct {
	provider *sdktrace.TracerProvider
}

// Start implements node.Lifecycle.
func (t *Service) Start() error {
	return nil // provider is already started during setup
}

// Stop implements node.Lifecycle.
func (t *Service) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := t.provider.Shutdown(ctx); err != nil {
		log.Error("Failed to stop OpenTelemetry service", "err", err)
		return err
	}
	log.Debug("OpenTelemetry stopped")
	return nil
}

// StartTelemetry initializes telemetry with the given parameters.
func StartTelemetry(ctx context.Context, cfg node.OpenTelemetryConfig, stack *node.Node) error {
	if !cfg.Enabled {
		return nil
	}

	if cfg.SampleRatio < 0 || cfg.SampleRatio > 1 {
		return fmt.Errorf("invalid sample ratio: %f", cfg.SampleRatio)
	}

	// Create exporter based on endpoint URL
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return fmt.Errorf("invalid rpc tracing endpoint URL: %w", err)
	}
	var exporter sdktrace.SpanExporter
	switch u.Scheme {
	case "http", "https":
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(u.Host),
		}
		if u.Scheme == "http" {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if u.Path != "" && u.Path != "/" {
			opts = append(opts, otlptracehttp.WithURLPath(u.Path))
		}
		if cfg.AuthUser != "" {
			opts = append(opts, otlptracehttp.WithHeaders(map[string]string{
				"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.AuthUser+":"+cfg.AuthPassword)),
			}))
		}
		exporter, err = otlptracehttp.New(ctx, opts...)
	default:
		return fmt.Errorf("unsupported telemetry url scheme: %s", u.Scheme)
	}
	if err != nil {
		return fmt.Errorf("failed to create telemetry exporter: %w", err)
	}

	// Define sampler such that if no parent span is available,
	// then sampleRatio of traces are sampled; otherwise, inherit
	// the parent's sampling decision.
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))

	// Define batch span processor options
	batchOpts := []sdktrace.BatchSpanProcessorOption{
		// The maximum number of spans that can be queued before dropping
		sdktrace.WithMaxQueueSize(sdktrace.DefaultMaxExportBatchSize),
		// The maximum number of spans to export in a single batch
		sdktrace.WithMaxExportBatchSize(sdktrace.DefaultMaxExportBatchSize),
		// How long an export operation can take before timing out
		sdktrace.WithExportTimeout(time.Duration(sdktrace.DefaultExportTimeout) * time.Millisecond),
		// How often to export, even if the batch isn't full
		sdktrace.WithBatchTimeout(time.Duration(sdktrace.DefaultScheduleDelay) * time.Millisecond),
	}

	// Define resource with service and client information
	var attr = []attribute.KeyValue{
		semconv.ServiceName("geth"),
		attribute.String("client.name", version.ClientName("geth")),
	}
	if cfg.InstanceID != "" {
		attr = append(attr, semconv.ServiceInstanceID(cfg.InstanceID))
	}
	res := resource.NewWithAttributes(semconv.SchemaURL, attr...)

	// Configure TracerProvider and set it as the global tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter, batchOpts...),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	// Set global propagator for context propagation
	// Note: This is needed for distributed tracing
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	service := &Service{provider: tp}
	stack.RegisterLifecycle(service)
	log.Info("OpenTelemetry tracing enabled", "endpoint", cfg.Endpoint)
	return nil
}
