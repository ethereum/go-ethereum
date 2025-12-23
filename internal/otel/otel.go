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

// Package otel provides OpenTelemetry tracing configuration for geth.
package otel

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

var (
	tracerProvider *sdktrace.TracerProvider
)

// Setup initializes OpenTelemetry tracing based on CLI flags.
func Setup(ctx *cli.Context) error {
	if !ctx.Bool(utils.OTELTracingFlag.Name) {
		return nil
	}
	endpoint := ctx.String(utils.OTELEndpointFlag.Name)
	protocol := ctx.String(utils.OTELProtocolFlag.Name)
	insecure := ctx.Bool(utils.OTELInsecureFlag.Name)
	serviceName := ctx.String(utils.OTELServiceNameFlag.Name)

	setupCtx := ctx.Context
	if setupCtx == nil {
		setupCtx = context.Background()
	}

	// Create resource with service information
	res, err := resource.New(setupCtx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create otel resource: %w", err)
	}

	// Create exporter based on protocol
	var exporter sdktrace.SpanExporter
	switch protocol {
	case "grpc":
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(endpoint),
		}
		if insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		exporter, err = otlptracegrpc.New(setupCtx, opts...)
	case "http":
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(endpoint),
		}
		if insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		exporter, err = otlptracehttp.New(setupCtx, opts...)
	default:
		return fmt.Errorf("unsupported otel protocol: %s (use 'grpc' or 'http')", protocol)
	}
	if err != nil {
		return fmt.Errorf("failed to create otel exporter: %w", err)
	}

	// Create TracerProvider
	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set as global provider
	otel.SetTracerProvider(tracerProvider)

	// Set global propagator for context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Info("OpenTelemetry tracing enabled", "endpoint", endpoint, "protocol", protocol, "service", serviceName)
	return nil
}

// Exit shuts down the OpenTelemetry TracerProvider, flushing any remaining spans.
func Exit() {
	if tracerProvider != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tracerProvider.Shutdown(ctx); err != nil {
			log.Error("Failed to shutdown OpenTelemetry tracer provider", "err", err)
		}
	}
}
