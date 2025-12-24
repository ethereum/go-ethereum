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

func spanNames(spans []tracetest.SpanStub) []string {
	names := make([]string, len(spans))
	for i, s := range spans {
		names[i] = s.Name
	}
	return names
}

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
	tracer := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() { _ = tracer.Shutdown(context.Background()) })

	server := newTestServer()
	server.SetTracerProvider(tracer)
	t.Cleanup(server.Stop)

	return server, tracer, exporter
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
		if spans[i].Name == "rpc.call" {
			rpcSpan = &spans[i]
			break
		}
	}
	if rpcSpan == nil {
		t.Fatalf("rpc.call span not found, got %v", spanNames(spans))
	}

	attrs := attributeMap(rpcSpan.Attributes)
	if attrs["rpc.system"] != "jsonrpc" {
		t.Errorf("expected rpc.system=jsonrpc, got %v", attrs["rpc.system"])
	}
	if attrs["rpc.method"] != "test_echo" {
		t.Errorf("expected rpc.method=test_echo, got %v", attrs["rpc.method"])
	}
	if _, ok := attrs["rpc.id"]; !ok {
		t.Errorf("expected rpc.id attribute to be set")
	}
}

// TestTracingInProcess verifies that RPC spans are emitted when processing requests in process.
func TestTracingInProcess(t *testing.T) {
	t.Parallel()
	server, tracer, exporter := newTracingServer(t)

	client := DialInProc(server)
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
		if spans[i].Name == "rpc.call" {
			rpcSpan = &spans[i]
			break
		}
	}
	if rpcSpan == nil {
		t.Fatalf("rpc.call span not found, got %v", spanNames(spans))
	}

	attrs := attributeMap(rpcSpan.Attributes)
	if attrs["rpc.system"] != "jsonrpc" {
		t.Errorf("expected rpc.system=jsonrpc, got %v", attrs["rpc.system"])
	}
	if attrs["rpc.method"] != "test_echo" {
		t.Errorf("expected rpc.method=test_echo, got %v", attrs["rpc.method"])
	}
	if _, ok := attrs["rpc.id"]; !ok {
		t.Errorf("expected rpc.id attribute to be set")
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

	// Flush and verify we emitted the rpc.call spans with batch=true.
	if err := tracer.ForceFlush(context.Background()); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("no spans were emitted")
	}

	var found int
	for i := range spans {
		if spans[i].Name != "rpc.call" {
			continue
		}
		attrs := attributeMap(spans[i].Attributes)
		if attrs["rpc.system"] == "jsonrpc" &&
			attrs["rpc.method"] == "test_echo" &&
			attrs["rpc.batch"] == "true" {
			found++
		}
	}
	if found != len(batch) {
		t.Fatalf("expected %d matching batch spans, got %d", len(batch), found)
	}
}
