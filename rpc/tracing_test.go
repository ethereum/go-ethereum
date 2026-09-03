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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
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
	var writeJSONSpan *tracetest.SpanStub
	var httpWriteSpan *tracetest.SpanStub
	for i := range spans {
		switch spans[i].Name {
		case "jsonrpc.test/echo":
			rpcSpan = &spans[i]
		case "rpc.writeJSON":
			writeJSONSpan = &spans[i]
		case "rpc.httpWrite":
			httpWriteSpan = &spans[i]
		}
	}
	if rpcSpan == nil {
		t.Fatalf("jsonrpc.test/echo span not found")
	}
	if writeJSONSpan == nil {
		t.Fatalf("rpc.writeJSON span not found")
	}
	if httpWriteSpan == nil {
		t.Fatalf("rpc.httpWrite span not found")
	}
	if got, want := httpWriteSpan.Parent.SpanID(), writeJSONSpan.SpanContext.SpanID(); got != want {
		t.Errorf("rpc.httpWrite parent: got %s, want rpc.writeJSON (%s)", got, want)
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

	// The runMethod span and the SERVER span should both have error status.
	if len(spans) == 0 {
		t.Fatal("no spans were emitted")
	}
	for _, span := range spans {
		switch span.Name {
		case "rpc.runMethod", "jsonrpc.test/returnError":
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

	// Flush and verify the batch trace shape:
	//   jsonrpc.batch (SERVER, rpc.batch.size=N)
	//     - jsonrpc.test/echo (INTERNAL, x N)
	//     - rpc.writeJSONBatch (INTERNAL)
	//          - rpc.httpWriteResult (INTERNAL)
	if err := tracer.ForceFlush(context.Background()); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("no spans were emitted")
	}
	var (
		batchSpan          *tracetest.SpanStub
		callSpans          []*tracetest.SpanStub
		writeJSONBatchSpan *tracetest.SpanStub
		httpWriteSpan      *tracetest.SpanStub
	)
	for i := range spans {
		switch spans[i].Name {
		case "jsonrpc.batch":
			batchSpan = &spans[i]
		case "jsonrpc.test/echo":
			callSpans = append(callSpans, &spans[i])
		case "rpc.writeJSONBatch":
			writeJSONBatchSpan = &spans[i]
		case "rpc.httpWrite":
			httpWriteSpan = &spans[i]
		}
	}
	if batchSpan == nil {
		t.Fatal("jsonrpc.batch span not found")
	}
	if got, want := len(callSpans), len(batch); got != want {
		t.Fatalf("got %d per-call spans, want %d", got, want)
	}
	if writeJSONBatchSpan == nil {
		t.Fatal("rpc.writeJSONBatch span not found")
	}
	if httpWriteSpan == nil {
		t.Fatal("rpc.httpWrite span not found")
	}

	// Batch span: SERVER kind, rpc.batch.size=N.
	if batchSpan.SpanKind != trace.SpanKindServer {
		t.Errorf("jsonrpc.batch: got kind %v, want SERVER", batchSpan.SpanKind)
	}
	batchAttrs := attributeMap(batchSpan.Attributes)
	if got, want := batchAttrs["rpc.batch.size"], strconv.Itoa(len(batch)); got != want {
		t.Errorf("jsonrpc.batch rpc.batch.size: got %q, want %q", got, want)
	}

	// Per-call spans: INTERNAL kind, parented to the batch span, carry rpc.* attrs.
	for _, s := range callSpans {
		if s.SpanKind != trace.SpanKindInternal {
			t.Errorf("jsonrpc.test/echo: got kind %v, want INTERNAL", s.SpanKind)
		}
		if got, want := s.Parent.SpanID(), batchSpan.SpanContext.SpanID(); got != want {
			t.Errorf("jsonrpc.test/echo parent: got %s, want %s (batch)", got, want)
		}
		attrs := attributeMap(s.Attributes)
		if attrs["rpc.system"] != "jsonrpc" || attrs["rpc.service"] != "test" || attrs["rpc.method"] != "echo" {
			t.Errorf("jsonrpc.test/echo attrs missing rpc.system/service/method: %v", attrs)
		}
	}

	// writeJSONBatch parented to the batch span.
	if got, want := writeJSONBatchSpan.Parent.SpanID(), batchSpan.SpanContext.SpanID(); got != want {
		t.Errorf("rpc.writeJSONBatch parent: got %s, want %s (batch)", got, want)
	}

	// httpWriteResult parented to writeJSONBatch.
	if got, want := httpWriteSpan.Parent.SpanID(), writeJSONBatchSpan.SpanContext.SpanID(); got != want {
		t.Errorf("rpc.httpWriteResult parent: got %s, want %s (rpc.writeJSONBatch)", got, want)
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

// postJSONRPC sends a raw JSON body to the given test server and discards the
// response body. Used to send messages the typed RPC client can't construct,
// like notifications (no "id" field).
func postJSONRPC(t *testing.T, url, body string) {
	t.Helper()
	if err := tryPostJSONRPC(url, body); err != nil {
		t.Fatalf("request: %v", err)
	}
}

// tryPostJSONRPC is like postJSONRPC but returns the transport error instead of
// failing the test. The write-timeout test uses this because the HTTP
// WriteTimeout can drop the connection before the response is flushed.
func tryPostJSONRPC(url, body string) error {
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return nil
}

// TestTracingHTTPNotification verifies that a JSON-RPC notification emits the
// SERVER span (with error captured when applicable) but no rpc.writeJSON span,
// since notifications do not get a response written.
func TestTracingHTTPNotification(t *testing.T) {
	t.Parallel()
	server, tracer, exporter := newTracingServer(t)
	httpsrv := httptest.NewServer(server)
	t.Cleanup(httpsrv.Close)

	// Successful notification (no "id"): should produce a SERVER span without error,
	// and no rpc.writeJSON span.
	postJSONRPC(t, httpsrv.URL, `{"jsonrpc":"2.0","method":"test_echo","params":["hi",1,{"S":"x"}]}`)

	// Notification with unknown method: SERVER span should be present with error status.
	postJSONRPC(t, httpsrv.URL, `{"jsonrpc":"2.0","method":"test_doesNotExist"}`)

	if err := tracer.ForceFlush(context.Background()); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	spans := exporter.GetSpans()

	var (
		echoSpan       *tracetest.SpanStub
		unknownSpan    *tracetest.SpanStub
		writeJSONFound bool
	)
	for i := range spans {
		switch spans[i].Name {
		case "jsonrpc.test/echo":
			echoSpan = &spans[i]
		case "jsonrpc.test/doesNotExist":
			unknownSpan = &spans[i]
		case "rpc.writeJSON":
			writeJSONFound = true
		}
	}
	if echoSpan == nil {
		t.Fatal("jsonrpc.test/echo span not found for successful notification")
	}
	if echoSpan.Status.Code == codes.Error {
		t.Errorf("successful notification: expected no error status, got %v", echoSpan.Status)
	}
	if unknownSpan == nil {
		t.Fatal("jsonrpc.test/doesNotExist span not found for unknown-method notification")
	}
	if unknownSpan.Status.Code != codes.Error {
		t.Errorf("unknown-method notification: expected error status, got %v", unknownSpan.Status.Code)
	}
	if writeJSONFound {
		t.Error("notifications should not produce an rpc.writeJSON span")
	}
}

// TestTracingBatchHTTPErrorCapture verifies that errors on individual calls
// inside a batch are recorded on the per-call INTERNAL span, including the
// pre-dispatch cases (method not found / invalid params) where runMethod
// never runs.
func TestTracingBatchHTTPErrorCapture(t *testing.T) {
	t.Parallel()
	server, tracer, exporter := newTracingServer(t)
	httpsrv := httptest.NewServer(server)
	t.Cleanup(httpsrv.Close)

	// A batch with: one valid call, one unknown method, one method that
	// returns an error from its handler.
	body := `[
		{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["x",1,{"S":"a"}]},
		{"jsonrpc":"2.0","id":2,"method":"test_doesNotExist"},
		{"jsonrpc":"2.0","id":3,"method":"test_returnError"}
	]`
	postJSONRPC(t, httpsrv.URL, body)

	if err := tracer.ForceFlush(context.Background()); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	spans := exporter.GetSpans()

	byName := make(map[string]*tracetest.SpanStub)
	for i := range spans {
		byName[spans[i].Name] = &spans[i]
	}

	if byName["jsonrpc.batch"] == nil {
		t.Fatal("jsonrpc.batch span not found")
	}
	if echo := byName["jsonrpc.test/echo"]; echo == nil {
		t.Fatal("jsonrpc.test/echo span not found")
	} else if echo.Status.Code == codes.Error {
		t.Errorf("test/echo: unexpected error status %v", echo.Status)
	}
	if missing := byName["jsonrpc.test/doesNotExist"]; missing == nil {
		t.Fatal("jsonrpc.test/doesNotExist span not found (method-not-found should still get a per-call span)")
	} else if missing.Status.Code != codes.Error {
		t.Errorf("test/doesNotExist: expected error status, got %v", missing.Status.Code)
	}
	if ret := byName["jsonrpc.test/returnError"]; ret == nil {
		t.Fatal("jsonrpc.test/returnError span not found")
	} else if ret.Status.Code != codes.Error {
		t.Errorf("test/returnError: expected error status, got %v", ret.Status.Code)
	}
}

// TestTracingBatchHTTPEmpty verifies that an empty batch still emits a
// SERVER span, with rpc.batch.size=0 and error status.
func TestTracingBatchHTTPEmpty(t *testing.T) {
	t.Parallel()
	server, tracer, exporter := newTracingServer(t)
	httpsrv := httptest.NewServer(server)
	t.Cleanup(httpsrv.Close)

	postJSONRPC(t, httpsrv.URL, `[]`)

	// Wait for the in-flight request to finish so the deferred spanEnd fires
	// before GetSpans is called.
	httpsrv.Close()

	if err := tracer.ForceFlush(context.Background()); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	spans := exporter.GetSpans()

	var batchSpan *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "jsonrpc.batch" {
			batchSpan = &spans[i]
		}
	}
	if batchSpan == nil {
		t.Fatal("jsonrpc.batch span not found for empty batch")
	}
	if batchSpan.Status.Code != codes.Error {
		t.Errorf("empty batch: expected error status, got %v", batchSpan.Status.Code)
	}
	attrs := attributeMap(batchSpan.Attributes)
	if got, want := attrs["rpc.batch.size"], "0"; got != want {
		t.Errorf("empty batch rpc.batch.size: got %q, want %q", got, want)
	}
}

// TestTracingBatchHTTPTooLarge verifies that a batch exceeding the server's
// item limit emits a SERVER span with rpc.batch.size=N and error status.
func TestTracingBatchHTTPTooLarge(t *testing.T) {
	t.Parallel()
	server, tracer, exporter := newTracingServer(t)
	server.SetBatchLimits(2, 100000) // limit to 2 items
	httpsrv := httptest.NewServer(server)
	t.Cleanup(httpsrv.Close)

	// 3 items > limit of 2.
	body := `[
		{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["a",1,{"S":"x"}]},
		{"jsonrpc":"2.0","id":2,"method":"test_echo","params":["b",2,{"S":"y"}]},
		{"jsonrpc":"2.0","id":3,"method":"test_echo","params":["c",3,{"S":"z"}]}
	]`
	postJSONRPC(t, httpsrv.URL, body)

	// Wait for the in-flight request to finish so the deferred spanEnd fires
	// before GetSpans is called.
	httpsrv.Close()

	if err := tracer.ForceFlush(context.Background()); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	spans := exporter.GetSpans()

	var batchSpan *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "jsonrpc.batch" {
			batchSpan = &spans[i]
		}
	}
	if batchSpan == nil {
		t.Fatal("jsonrpc.batch span not found for too-large batch")
	}
	if batchSpan.Status.Code != codes.Error {
		t.Errorf("batch-too-large: expected error status, got %v", batchSpan.Status.Code)
	}
	attrs := attributeMap(batchSpan.Attributes)
	if got, want := attrs["rpc.batch.size"], "3"; got != want {
		t.Errorf("batch-too-large rpc.batch.size: got %q, want %q", got, want)
	}
}

// newHeaderRecordingServer creates an HTTP test server that responds to any
// JSON-RPC call and records the traceparent header of incoming requests.
func newHeaderRecordingServer(t *testing.T, headerCh chan<- string) *httptest.Server {
	t.Helper()
	httpsrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerCh <- r.Header.Get("traceparent")
		var msg jsonrpcMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			t.Errorf("invalid request body: %v", err)
		}
		resp := jsonrpcMessage{Version: vsn, ID: msg.ID, Result: []byte("null")}
		w.Header().Set("Content-Type", contentType)
		json.NewEncoder(w).Encode(&resp)
	}))
	t.Cleanup(httpsrv.Close)
	return httpsrv
}

// TestTracingClientPropagation verifies that the client injects the W3C
// traceparent header into outgoing HTTP requests when configured with the
// WithTextMapPropagator option.
func TestTracingClientPropagation(t *testing.T) {
	t.Parallel()

	headerCh := make(chan string, 1)
	httpsrv := newHeaderRecordingServer(t, headerCh)

	client, err := DialOptions(context.Background(), httpsrv.URL, WithTextMapPropagator(propagation.TraceContext{}))
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	t.Cleanup(client.Close)

	// Build a context carrying a sampled remote span context.
	const (
		traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
		spanID  = "00f067aa0ba902b7"
	)
	tid, err := trace.TraceIDFromHex(traceID)
	if err != nil {
		t.Fatal(err)
	}
	sid, err := trace.SpanIDFromHex(spanID)
	if err != nil {
		t.Fatal(err)
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	if err := client.CallContext(ctx, nil, "test_foo"); err != nil {
		t.Fatalf("RPC call failed: %v", err)
	}
	want := "00-" + traceID + "-" + spanID + "-01"
	if got := <-headerCh; got != want {
		t.Errorf("traceparent header: got %q, want %q", got, want)
	}

	// A call without a span context in ctx must not produce a traceparent header.
	if err := client.CallContext(context.Background(), nil, "test_foo"); err != nil {
		t.Fatalf("RPC call failed: %v", err)
	}
	if got := <-headerCh; got != "" {
		t.Errorf("traceparent header without span context: got %q, want none", got)
	}
}

// TestTracingHTTPTimeout verifies that when a non-batch call exceeds the HTTP
// server's WriteTimeout, the SERVER span ends with error status (carrying the
// timeout error message).
func TestTracingHTTPTimeout(t *testing.T) {
	t.Parallel()
	server, tracer, exporter := newTracingServer(t)

	// Configure a short WriteTimeout so the internal request timer fires
	// quickly. ContextRequestTimeout subtracts 100ms from WriteTimeout, so
	// 250ms here gives ~150ms before the timeout response is sent.
	httpsrv := httptest.NewUnstartedServer(server)
	httpsrv.Config.WriteTimeout = 250 * time.Millisecond
	httpsrv.Start()
	t.Cleanup(httpsrv.Close)

	// test_block waits on ctx.Done() and returns an error. The internal
	// timer cancels ctx, so test_block unblocks shortly after the timeout
	// response goes out.
	//
	// Ignore the client-side result. Under load the HTTP WriteTimeout can
	// drop the connection before the timeout response is flushed, which the
	// client sees as EOF. The server still records the timeout on its span,
	// which is what we assert below.
	_ = tryPostJSONRPC(httpsrv.URL, `{"jsonrpc":"2.0","id":1,"method":"test_block"}`)

	// Wait for the in-flight request to finish so the deferred spanEnd fires
	// before GetSpans is called.
	httpsrv.Close()

	if err := tracer.ForceFlush(context.Background()); err != nil {
		t.Fatalf("failed to flush: %v", err)
	}
	spans := exporter.GetSpans()

	var serverSpan *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "jsonrpc.test/block" {
			serverSpan = &spans[i]
		}
	}
	if serverSpan == nil {
		t.Fatal("jsonrpc.test/block span not found")
	}
	if serverSpan.Status.Code != codes.Error {
		t.Errorf("timeout: expected SERVER span error status, got %v (%q)", serverSpan.Status.Code, serverSpan.Status.Description)
	}
}
