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

package telemetry

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// newTestTracer creates a TracerProvider backed by an in-memory exporter.
func newTestTracer(t *testing.T) (trace.Tracer, *sdktrace.TracerProvider, *tracetest.InMemoryExporter) {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	return tp.Tracer("test"), tp, exporter
}

func TestStartSpanWithTracer_NoParent(t *testing.T) {
	t.Parallel()
	tracer, tp, exporter := newTestTracer(t)

	// Create a span without a parent.
	ctx := context.Background()
	retCtx, _, endSpan := StartSpanWithTracer(ctx, tracer, "should-not-exist")
	endSpan(nil)

	// The returned context should be the original context (unchanged).
	if retCtx != ctx {
		t.Fatal("expected original context to be returned unchanged")
	}

	// Flush and verify no spans were recorded.
	if err := tp.ForceFlush(context.Background()); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	spans := exporter.GetSpans()
	if len(spans) != 0 {
		t.Fatalf("expected no spans, got %d", len(spans))
	}
}

func TestStartSpanWithTracer_WithParent(t *testing.T) {
	t.Parallel()
	tracer, tp, exporter := newTestTracer(t)

	// Create a parent span to establish a valid span context.
	ctx, parentSpan := tracer.Start(context.Background(), "parent")
	defer parentSpan.End()

	// Should create a real child span.
	_, _, endSpan := StartSpanWithTracer(ctx, tracer, "child")
	endSpan(nil)

	// Flush and verify the child span was recorded.
	parentSpan.End()
	if err := tp.ForceFlush(context.Background()); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	spans := exporter.GetSpans()
	var childSpan *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "child" {
			childSpan = &spans[i]
			break
		}
	}
	if childSpan == nil {
		t.Fatal("child span not found")
	}

	// Verify it is parented to the correct trace.
	if childSpan.Parent.TraceID() != parentSpan.SpanContext().TraceID() {
		t.Errorf("trace ID mismatch: got %s, want %s",
			childSpan.Parent.TraceID(), parentSpan.SpanContext().TraceID())
	}
	if childSpan.SpanKind != trace.SpanKindInternal {
		t.Errorf("expected SpanKindInternal, got %v", childSpan.SpanKind)
	}
}
