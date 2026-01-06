package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Attribute = attribute.KeyValue

func StringAttribute(key, val string) Attribute {
	return attribute.String(key, val)
}

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
			// Error occurred, record it and set status on span and parent
			span.RecordError(*err)
			span.SetStatus(codes.Error, (*err).Error())
		}
		span.End()
	}
	return ctx, spanEnd
}
