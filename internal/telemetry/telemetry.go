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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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

// StartSpan starts a tracing span on the default tracer and returns a function
// to end the span. The function will record errors and set span status based
// on the error value.
func StartSpan(ctx context.Context, spanName string, attributes ...Attribute) (context.Context, func(*error)) {
	return StartSpanWithTracer(ctx, otel.Tracer(""), spanName, attributes...)
}

// StartSpanWithTracer starts a tracing span on the supplied tracer and returns
// a function to end the span. The function will record errors and set span
// status based on the error value.
func StartSpanWithTracer(ctx context.Context, tracer trace.Tracer, spanName string, attributes ...Attribute) (context.Context, func(*error)) {
	ctx, span := tracer.Start(ctx, spanName)

	// Fast path: noop provider or span not sampled
	if !span.IsRecording() {
		return ctx, func(*error) { span.End() }
	}

	// Set span attributes.
	if len(attributes) > 0 {
		span.SetAttributes(attributes...)
	}

	// Define the function to end the span and handle error recording
	spanEnd := func(err *error) {
		if *err != nil {
			// Error occurred, record it and set status on span
			span.RecordError(*err)
			span.SetStatus(codes.Error, (*err).Error())
		}
		span.End()
	}
	return ctx, spanEnd
}
