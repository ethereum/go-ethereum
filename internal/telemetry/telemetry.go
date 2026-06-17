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
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
	"go.opentelemetry.io/otel/trace"
)

// Attribute is an alias for attribute.KeyValue.
type Attribute = attribute.KeyValue

// StringAttribute creates an attribute with a string value.
func StringAttribute(key, val string) Attribute {
	return attribute.String(key, val)
}

// Int64Attribute creates an attribute with an int64 value.
func Int64Attribute(key string, val int64) Attribute {
	return attribute.Int64(key, val)
}

// IntAttribute creates an attribute with an int value.
func IntAttribute(key string, val int) Attribute {
	return attribute.Int(key, val)
}

// BoolAttribute creates an attribute with a bool value.
func BoolAttribute(key string, val bool) Attribute {
	return attribute.Bool(key, val)
}

// IsRecording reports whether the context carries a valid parent span, i.e.
// whether a span started now would actually be exported. Hot paths can use this
// to avoid building span attributes (which may allocate) when tracing is off.
func IsRecording(ctx context.Context) bool {
	return trace.SpanFromContext(ctx).SpanContext().IsValid()
}

// StartSpan creates a SpanKind=INTERNAL span.
func StartSpan(ctx context.Context, spanName string, attributes ...Attribute) (context.Context, trace.Span, func(*error)) {
	return StartSpanWithTracer(ctx, otel.Tracer(""), spanName, attributes...)
}

// StartSpanWithTracer requires a tracer to be passed in and creates a SpanKind=INTERNAL span.
func StartSpanWithTracer(ctx context.Context, tracer trace.Tracer, name string, attributes ...Attribute) (context.Context, trace.Span, func(*error)) {
	// Don't create a span if there's no parent span in the context.
	parent := trace.SpanFromContext(ctx)
	if !parent.SpanContext().IsValid() {
		return ctx, parent, func(*error) {}
	}
	return startSpan(ctx, tracer, trace.SpanKindInternal, name, attributes...)
}

// TracerFromContext returns a Tracer from the TracerProvider associated with the
// parent span in ctx. If ctx has no parent span, the returned tracer comes from
// the no-op provider, so spans created with it will not be exported.
func TracerFromContext(ctx context.Context) trace.Tracer {
	return trace.SpanFromContext(ctx).TracerProvider().Tracer("")
}

// RPCInfo contains information about the RPC request.
type RPCInfo struct {
	System    string
	Service   string
	Method    string
	RequestID string
}

// StartCallServerSpan creates a SpanKind=SERVER span for a JSON-RPC call.
// The span name is formatted as $rpcSystem.$rpcService/$rpcMethod
// (e.g. "jsonrpc.engine/newPayloadV4") which follows the Open Telemetry
// semantic convensions: https://opentelemetry.io/docs/specs/semconv/rpc/rpc-spans/#span-name.
func StartCallServerSpan(ctx context.Context, tracer trace.Tracer, rpc RPCInfo, others ...Attribute) (context.Context, func(*error)) {
	var (
		name       = fmt.Sprintf("%s.%s/%s", rpc.System, rpc.Service, rpc.Method)
		attributes = append([]Attribute{
			semconv.RPCSystemKey.String(rpc.System),
			semconv.RPCServiceKey.String(rpc.Service),
			semconv.RPCMethodKey.String(rpc.Method),
			semconv.RPCJSONRPCRequestID(rpc.RequestID),
		},
			others...,
		)
	)
	ctx, _, end := startSpan(ctx, tracer, trace.SpanKindServer, name, attributes...)
	return ctx, end
}

// StartBatchServerSpan creates a SpanKind=SERVER span representing a batched request.
// The span name is "$system.batch" (e.g. "jsonrpc.batch") and per-call spans are nested under it.
// batchSize is exposed as rpc.batch.size.
func StartBatchServerSpan(ctx context.Context, tracer trace.Tracer, system string, batchSize int, others ...Attribute) (context.Context, func(*error)) {
	attributes := append([]Attribute{
		semconv.RPCSystemKey.String(system),
		IntAttribute("rpc.batch.size", batchSize),
	}, others...)
	ctx, _, end := startSpan(ctx, tracer, trace.SpanKindServer, system+".batch", attributes...)
	return ctx, end
}

// StartBatchCallSpan creates a SpanKind=INTERNAL span for an individual RPC call as part of a batch.
// This carries the same name and attributes as StartCallServerSpan.
func StartBatchCallSpan(ctx context.Context, tracer trace.Tracer, rpc RPCInfo, others ...Attribute) (context.Context, func(*error)) {
	var (
		name       = fmt.Sprintf("%s.%s/%s", rpc.System, rpc.Service, rpc.Method)
		attributes = append([]Attribute{
			semconv.RPCSystemKey.String(rpc.System),
			semconv.RPCServiceKey.String(rpc.Service),
			semconv.RPCMethodKey.String(rpc.Method),
			semconv.RPCJSONRPCRequestID(rpc.RequestID),
		},
			others...,
		)
	)
	ctx, _, end := startSpan(ctx, tracer, trace.SpanKindInternal, name, attributes...)
	return ctx, end
}

// startSpan creates a span with the given kind.
func startSpan(ctx context.Context, tracer trace.Tracer, kind trace.SpanKind, spanName string, attributes ...Attribute) (context.Context, trace.Span, func(*error)) {
	ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(kind))
	if len(attributes) > 0 {
		span.SetAttributes(attributes...)
	}
	return ctx, span, endSpan(span)
}

// endSpan ends the span and handles error recording.
func endSpan(span trace.Span) func(*error) {
	return func(err *error) {
		if err != nil && *err != nil {
			span.RecordError(*err)
			span.SetStatus(codes.Error, (*err).Error())
		}
		span.End()
	}
}
