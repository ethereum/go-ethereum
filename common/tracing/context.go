package tracing

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type tracerKey struct{}

type Option func(context.Context, trace.Span)

func WithTracer(ctx context.Context, tr trace.Tracer) context.Context {
	return context.WithValue(ctx, tracerKey{}, tr)
}

func FromContext(ctx context.Context) trace.Tracer {
	tr, _ := ctx.Value(tracerKey{}).(trace.Tracer)

	return tr
}

func StartSpan(ctx context.Context, snapName string) (context.Context, trace.Span) {
	tr := FromContext(ctx)

	if tr == nil {
		return ctx, nil
	}

	ctx, span := tr.Start(ctx, snapName)
	ctx = WithTracer(ctx, tr)

	return ctx, span
}

func EndSpan(span trace.Span) {
	if span != nil {
		span.End()
	}
}

func Trace(ctx context.Context, spanName string) (context.Context, trace.Span) {
	tr := FromContext(ctx)

	if tr == nil {
		return ctx, nil
	}

	return tr.Start(ctx, spanName)
}

func Exec(ctx context.Context, spanName string, opts ...Option) {
	var span trace.Span

	tr := FromContext(ctx)

	if tr != nil {
		ctx, span = tr.Start(ctx, spanName)
	}

	for _, optFn := range opts {
		optFn(ctx, span)
	}

	if tr != nil {
		span.End()
	}
}

func WithTime(fn func(context.Context, trace.Span)) Option {
	return func(ctx context.Context, span trace.Span) {
		ElapsedTime(ctx, span, "elapsed", fn)
	}
}

func ElapsedTime(ctx context.Context, span trace.Span, msg string, fn func(context.Context, trace.Span)) {
	var now time.Time

	if span != nil {
		now = time.Now()
	}

	fn(ctx, span)

	if span != nil {
		span.SetAttributes(attribute.Int(msg, int(time.Since(now).Milliseconds())))
	}
}

func SetAttributes(span trace.Span, kvs ...attribute.KeyValue) {
	if span != nil {
		span.SetAttributes(kvs...)
	}
}
