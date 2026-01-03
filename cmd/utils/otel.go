// Copyright 2025 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/internal/version"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// otelService wraps the OpenTelemetry TracerProvider to implement node.Lifecycle.
type otelService struct {
	tracerProvider *sdktrace.TracerProvider
}

// Start implements node.Lifecycle.
func (o *otelService) Start() error {
	return nil // Provider is already started during setup
}

// Stop implements node.Lifecycle
func (o *otelService) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := o.tracerProvider.Shutdown(ctx); err != nil {
		log.Error("Failed to stop OTEL Service", "err", err)
		return err
	}
	log.Info("OTEL Service stopped")
	return nil
}

// SetupOTEL initializes OpenTelemetry tracing based on CLI flags.
func SetupOTEL(ctx *cli.Context) (*otelService, error) {
	if !ctx.Bool(OTELEnabledFlag.Name) {
		return nil, nil
	}
	endpoint := ctx.String(OTELEndpointFlag.Name)
	if endpoint == "" {
		return nil, nil
	}
	serviceName := ctx.String(OTELServiceNameFlag.Name)
	setupCtx := ctx.Context
	if setupCtx == nil {
		setupCtx = context.Background()
	}

	// Create exporter based on endpoint URL
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid otel endpoint URL: %w", err)
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
		exporter, err = otlptracehttp.New(setupCtx, opts...)
	default:
		return nil, fmt.Errorf("unsupported otel url scheme: %s", u.Scheme)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create otel exporter: %w", err)
	}

	// Create resource with service information
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
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
	// Note: This should enable distributed tracing
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Info("OpenTelemetry tracing enabled", "endpoint", endpoint, "service", serviceName)
	return &otelService{tracerProvider: tp}, nil
}

// RegisterOTELService registers the otelService with the node so its lifecycle is managed by the node.
func RegisterOTELService(otelService *otelService, stack *node.Node) {
	stack.RegisterLifecycle(otelService)
}
