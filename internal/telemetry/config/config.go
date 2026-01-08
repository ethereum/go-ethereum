// Copyright 2025 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/internal/version"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
)

// Provider wraps a TracerProvider and exposes a Shutdown method.
type Provider struct {
	tracerProvider *sdktrace.TracerProvider
}

// Shutdown shuts down the tracer provider.
func (p *Provider) Shutdown(ctx context.Context) error {
	return p.tracerProvider.Shutdown(ctx)
}

// Setup initializes OpenTelemetry tracing with the given parameters.
func Setup(ctx context.Context, endpoint string) (*Provider, error) {
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

	// Create resource with service and client information
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("geth"),
		attribute.String("client.name", version.ClientName("geth")),
	)

	// Create TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set as global provider
	otel.SetTracerProvider(tp)

	// Set global propagator for context propagation
	// Note: This is needed for distributed tracing
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Provider{tracerProvider: tp}, nil
}
