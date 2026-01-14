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

package config

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/internal/version"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
)

// TelemetryProvider wraps a TracerProvider and exposes the Shutdown method.
type TelemetryProvider struct {
	tracerProvider *sdktrace.TracerProvider
}

// Shutdown shuts down the TracerProvider.
func (t *TelemetryProvider) Shutdown(ctx context.Context) error {
	return t.tracerProvider.Shutdown(ctx)
}

// Setup initializes telemetry with the given parameters.
func Setup(ctx context.Context, endpoint string, sampleRatio float64) (*TelemetryProvider, error) {
	// Create exporter based on endpoint URL
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid rpc tracing endpoint URL: %w", err)
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
		exporter, err = otlptracehttp.New(ctx, opts...)
	default:
		return nil, fmt.Errorf("unsupported telemetry url scheme: %s", u.Scheme)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create telemetry exporter: %w", err)
	}

	// Define sampler such that if no parent span is available,
	// then sampleRatio of traces are sampled; otherwise, inherit
	// the parent's sampling decision.
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRatio))

	// Define batch span processor options
	batchOpts := []sdktrace.BatchSpanProcessorOption{
		// The maximum number of spans that can be queued before dropping
		sdktrace.WithMaxQueueSize(sdktrace.DefaultMaxExportBatchSize),
		// The maximum number of spans to export in a single batch
		sdktrace.WithMaxExportBatchSize(sdktrace.DefaultMaxExportBatchSize),
		// How long an export operation can take before timing out
		sdktrace.WithExportTimeout(sdktrace.DefaultExportTimeout),
		// How often to export, even if the batch isn't full
		sdktrace.WithBatchTimeout(5 * time.Second), // SDK default is 5s
	}

	// Define resource with service and client information
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("geth"),
		attribute.String("client.name", version.ClientName("geth")),
	)

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

	return &TelemetryProvider{tracerProvider: tp}, nil
}
