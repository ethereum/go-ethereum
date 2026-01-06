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

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

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

func newTracingServer(t *testing.T) (*Server, *sdktrace.TracerProvider, *tracetest.InMemoryExporter) {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	server := newTestServer()
	server.SetTracerProvider(tp)
	t.Cleanup(server.Stop)
	return server, tp, exporter
}

// TestTracingHTTP verifies that RPC spans are emitted when processing HTTP requests.
func TestTracingHTTP(t *testing.T) {
	t.Parallel()
	server, tracer, exporter := newTracingServer(t)
	httpsrv := httptest.NewServer(server)
	t.Cleanup(httpsrv.Close)
	client, err := DialHTTP(httpsrv.URL)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	t.Cleanup(client.Close)

	var result echoResult
	if err := client.Call(&result, "test_echo", "hello", 42, &echoArgs{S: "world"}); err != nil {
		t.Fatalf("RPC call failed: %v", err)
	}
	if err := tracer.ForceFlush(context.Background()); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("no spans were emitted")
	}
	var rpcSpan *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "rpc.handleCall" {
			rpcSpan = &spans[i]
			break
		}
	}
	if rpcSpan == nil {
		t.Fatalf("rpc.handleCall span not found.")
	}
	attrs := attributeMap(rpcSpan.Attributes)
	if attrs["rpc.method"] != "test_echo" {
		t.Errorf("expected rpc.method=test_echo, got %v", attrs["rpc.method"])
	}
	if _, ok := attrs["rpc.id"]; !ok {
		t.Errorf("expected rpc.id attribute to be set")
	}
}

func TestTracingHTTPShouldFail(t *testing.T) {
	t.Parallel()
	server, tracer, exporter := newTracingServer(t)
	httpsrv := httptest.NewServer(server)
	t.Cleanup(httpsrv.Close)
	client, err := DialHTTP(httpsrv.URL)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	t.Cleanup(client.Close)

	var result echoResult
	if err := client.Call(&result, "testnonexistent", "hello", 42, &echoArgs{S: "world"}); err == nil {
		t.Fatalf("RPC call should have failed")
	}
	if err := tracer.ForceFlush(context.Background()); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("no spans were emitted")
	}
	var rpcSpan *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "rpc.handleCall" {
			rpcSpan = &spans[i]
			break
		}
	}
	if rpcSpan == nil {
		t.Fatalf("rpc.handleCall span not found.")
	}
	attrs := attributeMap(rpcSpan.Attributes)
	if attrs["rpc.method"] != "testnonexistent" {
		t.Errorf("expected rpc.method=testnonexistent, got %v", attrs["rpc.method"])
	}
	if _, ok := attrs["rpc.id"]; !ok {
		t.Errorf("expected rpc.id attribute to be set")
	}
}

// TestTracingSubscribeUnsubscribe verifies that subscribe and unsubscribe calls
// do not emit any spans.
// Note: This works because client.newClientConn() does not set a tracer provider.
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
