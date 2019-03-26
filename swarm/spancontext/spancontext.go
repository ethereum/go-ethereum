package spancontext

import (
	"context"

	opentracing "github.com/opentracing/opentracing-go"
)

func WithContext(ctx context.Context, sctx opentracing.SpanContext) context.Context {
	return context.WithValue(ctx, "span_context", sctx)
}

func FromContext(ctx context.Context) opentracing.SpanContext {
	sctx, ok := ctx.Value("span_context").(opentracing.SpanContext)
	if ok {
		return sctx
	}

	return nil
}

func StartSpan(ctx context.Context, name string) (context.Context, opentracing.Span) {
	tracer := opentracing.GlobalTracer()

	sctx := FromContext(ctx)

	var sp opentracing.Span
	if sctx != nil {
		sp = tracer.StartSpan(
			name,
			opentracing.ChildOf(sctx))
	} else {
		sp = tracer.StartSpan(name)
	}

	nctx := context.WithValue(ctx, "span_context", sp.Context())

	return nctx, sp
}

func StartSpanFrom(name string, sctx opentracing.SpanContext) opentracing.Span {
	tracer := opentracing.GlobalTracer()

	sp := tracer.StartSpan(
		name,
		opentracing.ChildOf(sctx))

	return sp
}
