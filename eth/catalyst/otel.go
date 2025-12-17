package catalyst

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	tracerInitOnce sync.Once

	engineTracer    trace.TracerProvider = noop.NewTracerProvider()
	engineSDKTracer *sdktrace.TracerProvider
)

// initEngineTelemetry initializes the OpenTelemetry tracing for the engine API.
func initEngineTelemetry() {

	// TODO(jrhea): allow caller to provide init context once lifecycle plumbing exists.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// TODO(jrhea): make endpoint configurable via flags.
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("localhost:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		log.Warn("OpenTelemetry exporter init failed, using no-op tracer", "err", err)
		return
	}
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("geth"),
			semconv.ServiceNamespace("engine"),
		),
	)
	if err != nil {
		log.Warn("OpenTelemetry resource init failed", "err", err)
		res = resource.Empty()
	}

	// TODO(jrhea): Intentionally use a synchronous exporter + AlwaysSample
	// for initial testing. This will be replaced with a batched exporter
	// and production-ready sampling.
	tracer := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSyncer(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tracer)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	engineTracer = tracer
	engineSDKTracer = tracer
}

// TODO(jrhea): wire into geth/node lifecycle.
func shutdownEngineTelemetry(ctx context.Context) error {
	if engineSDKTracer != nil {
		return engineSDKTracer.Shutdown(ctx)
	}
	return nil
}

func getEngineTracer() trace.Tracer {
	tracerInitOnce.Do(initEngineTelemetry)
	return engineTracer.Tracer("geth/engine")
}
