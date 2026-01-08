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

// RPCInfo contains information about the RPC request.
type RPCInfo struct {
	System    string
	Service   string
	Method    string
	RequestID string
}

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

// StartServerSpan creates a SpanKind=SERVER span at the JSON-RPC boundary.
// The span name is formatted as $rpcSystem.$rpcService/$rpcMethod
// (e.g. "jsonrpc.engine/newPayloadV4").
func StartServerSpan(ctx context.Context, tracer trace.Tracer, rpc RPCInfo, others ...Attribute) (context.Context, func(error)) {
	spanName := fmt.Sprintf("%s.%s/%s", rpc.System, rpc.Service, rpc.Method)
	ctx, span := tracer.Start(
		ctx,
		spanName,
		trace.WithSpanKind(trace.SpanKindServer),
	)

	// Fast path: noop provider or span not sampled
	if !span.IsRecording() {
		return ctx, func(error) { span.End() }
	}

	// Define required attributes
	attrs := []Attribute{
		semconv.RPCSystemKey.String(rpc.System),
		semconv.RPCServiceKey.String(rpc.Service),
		semconv.RPCMethodKey.String(rpc.Method),
		semconv.RPCJSONRPCRequestID(rpc.RequestID),
	}

	// Add any additional attributes provided
	if len(others) > 0 {
		attrs = append(attrs, others...)
	}
	span.SetAttributes(attrs...)
	return ctx, endSpan(span)
}

// StartSpan creates a SpanKind=INTERNAL span.
func StartSpan(
	ctx context.Context,
	spanName string,
	attributes ...Attribute,
) (context.Context, trace.Span, func(error)) {
	return StartSpanWithTracer(ctx, otel.Tracer(""), spanName, attributes...)
}

// StartSpanWithTracer requires a tracer to be passed in and creates a SpanKind=INTERNAL span.
func StartSpanWithTracer(
	ctx context.Context,
	tracer trace.Tracer,
	spanName string,
	attributes ...Attribute,
) (context.Context, trace.Span, func(error)) {
	return startSpan(ctx, tracer, spanName, attributes...)
}

// startSpan creates a SpanKind=INTERNAL span.
func startSpan(
	ctx context.Context,
	tracer trace.Tracer,
	spanName string,
	attributes ...Attribute,
) (context.Context, trace.Span, func(error)) {
	ctx, span := tracer.Start(
		ctx,
		spanName,
		trace.WithSpanKind(trace.SpanKindInternal),
	)

	// Fast path
	if !span.IsRecording() {
		return ctx, span, func(error) { span.End() }
	}

	if len(attributes) > 0 {
		span.SetAttributes(attributes...)
	}
	return ctx, span, endSpan(span)
}

// endSpan ends the span and handles error recording.
func endSpan(span trace.Span) func(error) {
	return func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}
}
