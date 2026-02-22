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

package rpc

import (
	"context"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// attributeMap converts a slice of attributes to a map.
func attributeMap(attrs []attribute.KeyValue) map[string]string {
	m := make(map[string]string)
	for _, a := range attrs {
		switch a.Value.Type() {
		case attribute.STRING:
			m[string(a.Key)] = a.Value.AsString()
		case attribute.BOOL:
			if a.Value.AsBool() {
				m[string(a.Key)] = "true"
			} else {
				m[string(a.Key)] = "false"
			}
		default:
			m[string(a.Key)] = a.Value.Emit()
		}
	}
	return m
}

// newTracingServer creates a new server with tracing enabled.
func newTracingServer(t *testing.T) (*Server, *sdktrace.TracerProvider, *tracetest.InMemoryExporter) {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	server := newTestServer()
	server.setTracerProvider(tp)
	t.Cleanup(server.Stop)
	return server, tp, exporter
}

// TestTracingHTTP verifies that RPC spans are emitted when processing HTTP requests.
func TestTracingHTTP(t *testing.T) {
	// Not parallel: this test modifies the global otel TextMapPropagator.

	// Set up a propagator to extract W3C Trace Context headers.
	originalPropagator := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(originalPropagator) })

	server, tracer, exporter := newTracingServer(t)
	httpsrv := httptest.NewServer(server)
	t.Cleanup(httpsrv.Close)

	// Define the expected trace and span IDs for context propagation.
	const (
		traceID      = "4bf92f3577b34da6a3ce929d0e0e4736"
		parentSpanID = "00f067aa0ba902b7"
		traceparent  = "00-" + traceID + "-" + parentSpanID + "-01"
	)

	client, err := DialHTTP(httpsrv.URL)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	t.Cleanup(client.Close)

	// Set trace context headers.
	client.SetHeader("traceparent", traceparent)

	// Make a successful RPC call.
	var result echoResult
	if err := client.Call(&result, "test_echo", "hello", 42, &echoArgs{S: "world"}); err != nil {
		t.Fatalf("RPC call failed: %v", err)
	}

	// Flush and verify that we emitted the expected span.
	if err := tracer.ForceFlush(context.Background()); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("no spans were emitted")
	}
	var rpcSpan *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "jsonrpc.test/echo" {
			rpcSpan = &spans[i]
			break
		}
	}
	if rpcSpan == nil {
		t.Fatalf("jsonrpc.test/echo span not found")
	}

	// Verify span attributes.
	attrs := attributeMap(rpcSpan.Attributes)
	if attrs["rpc.system"] != "jsonrpc" {
		t.Errorf("expected rpc.system=jsonrpc, got %v", attrs["rpc.system"])
	}
	if attrs["rpc.service"] != "test" {
		t.Errorf("expected rpc.service=test, got %v", attrs["rpc.service"])
	}
	if attrs["rpc.method"] != "echo" {
		t.Errorf("expected rpc.method=echo, got %v", attrs["rpc.method"])
	}
	if _, ok := attrs["rpc.jsonrpc.request_id"]; !ok {
		t.Errorf("expected rpc.jsonrpc.request_id attribute to be set")
	}

	// Verify the span's parent matches the traceparent header values.
	if got := rpcSpan.Parent.TraceID().String(); got != traceID {
		t.Errorf("parent trace ID mismatch: got %s, want %s", got, traceID)
	}
	if got := rpcSpan.Parent.SpanID().String(); got != parentSpanID {
		t.Errorf("parent span ID mismatch: got %s, want %s", got, parentSpanID)
	}
	if !rpcSpan.Parent.IsRemote() {
		t.Error("expected parent span context to be marked as remote")
	}
}

// TestTracingErrorRecording verifies that errors are recorded on spans.
func TestTracingHTTPErrorRecording(t *testing.T) {
	t.Parallel()
	server, tracer, exporter := newTracingServer(t)
	httpsrv := httptest.NewServer(server)
	t.Cleanup(httpsrv.Close)
	client, err := DialHTTP(httpsrv.URL)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	t.Cleanup(client.Close)

	// Call a method that returns an error.
	var result any
	err = client.Call(&result, "test_returnError")
	if err == nil {
		t.Fatal("expected error from test_returnError")
	}

	// Flush and verify spans recorded the error.
	if err := tracer.ForceFlush(context.Background()); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	spans := exporter.GetSpans()

	// Only the runMethod span should have error status.
	if len(spans) == 0 {
		t.Fatal("no spans were emitted")
	}
	for _, span := range spans {
		switch span.Name {
		case "rpc.runMethod":
			if span.Status.Code != codes.Error {
				t.Errorf("expected %s span status Error, got %v", span.Name, span.Status.Code)
			}
		default:
			if span.Status.Code == codes.Error {
				t.Errorf("unexpected error status on span %s", span.Name)
			}
		}
	}
}

// TestTracingBatchHTTP verifies that RPC spans are emitted for batched JSON-RPC calls over HTTP.
func TestTracingBatchHTTP(t *testing.T) {
	t.Parallel()
	server, tracer, exporter := newTracingServer(t)
	httpsrv := httptest.NewServer(server)
	t.Cleanup(httpsrv.Close)
	client, err := DialHTTP(httpsrv.URL)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	t.Cleanup(client.Close)

	// Make a successful batch RPC call.
	batch := []BatchElem{
		{
			Method: "test_echo",
			Args:   []any{"hello", 42, &echoArgs{S: "world"}},
			Result: new(echoResult),
		},
		{
			Method: "test_echo",
			Args:   []any{"your", 7, &echoArgs{S: "mom"}},
			Result: new(echoResult),
		},
	}
	if err := client.BatchCall(batch); err != nil {
		t.Fatalf("batch RPC call failed: %v", err)
	}

	// Flush and verify we emitted spans for each batch element.
	if err := tracer.ForceFlush(context.Background()); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("no spans were emitted")
	}
	var found int
	for i := range spans {
		if spans[i].Name == "jsonrpc.test/echo" {
			attrs := attributeMap(spans[i].Attributes)
			if attrs["rpc.system"] == "jsonrpc" &&
				attrs["rpc.service"] == "test" &&
				attrs["rpc.method"] == "echo" &&
				attrs["rpc.batch"] == "true" {
				found++
			}
		}
	}
	if found != len(batch) {
		t.Fatalf("expected %d matching batch spans, got %d", len(batch), found)
	}
}

// TestTracingSubscribeUnsubscribe verifies that subscribe and unsubscribe calls
// do not emit any spans.
// Note: This works because client.newClientConn() passes nil as the tracer provider.
func TestTracingSubscribeUnsubscribe(t *testing.T) {
	t.Parallel()
	server, tracer, exporter := newTracingServer(t)
	client := DialInProc(server)
	t.Cleanup(client.Close)

	// Subscribe to notifications.
	sub, err := client.Subscribe(context.Background(), "nftest", make(chan int), "someSubscription", 1, 1)
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	// Unsubscribe.
	sub.Unsubscribe()

	// Flush and check that no spans were emitted.
	if err := tracer.ForceFlush(context.Background()); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	spans := exporter.GetSpans()
	if len(spans) != 0 {
		t.Errorf("expected no spans for subscribe/unsubscribe, got %d", len(spans))
	}
}
