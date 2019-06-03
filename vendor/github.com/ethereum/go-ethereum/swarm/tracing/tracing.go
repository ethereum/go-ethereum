package tracing

import (
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/spancontext"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	// Enabled turns tracing on for the current swarm instance
	Enabled bool = false
	store        = spanStore{}
)

const (
	// TracingEnabledFlag is the CLI flag name to use to enable trace collections.
	TracingEnabledFlag = "tracing"

	// StoreLabelId is the context value key of the name of the span to be saved
	StoreLabelId = "span_save_id"

	// StoreLabelMeta is the context value key that together with StoreLabelId constitutes the retrieval key for saved spans in the span store
	// StartSaveSpan and ShiftSpanByKey
	StoreLabelMeta = "span_save_meta"
)

var (
	Closer io.Closer
)

var (
	TracingFlag = cli.BoolFlag{
		Name:  TracingEnabledFlag,
		Usage: "Enable tracing",
	}
	TracingEndpointFlag = cli.StringFlag{
		Name:  "tracing.endpoint",
		Usage: "Tracing endpoint",
		Value: "0.0.0.0:6831",
	}
	TracingSvcFlag = cli.StringFlag{
		Name:  "tracing.svc",
		Usage: "Tracing service name",
		Value: "swarm",
	}
)

// Flags holds all command-line flags required for tracing collection.
var Flags = []cli.Flag{
	TracingFlag,
	TracingEndpointFlag,
	TracingSvcFlag,
}

// Init enables or disables the open tracing system.
func init() {
	for _, arg := range os.Args {
		if flag := strings.TrimLeft(arg, "-"); flag == TracingEnabledFlag {
			Enabled = true
		}
	}
}

func Setup(ctx *cli.Context) {
	if Enabled {
		log.Info("Enabling opentracing")
		var (
			endpoint = ctx.GlobalString(TracingEndpointFlag.Name)
			svc      = ctx.GlobalString(TracingSvcFlag.Name)
		)

		Closer = initTracer(endpoint, svc)
	}
}

func initTracer(endpoint, svc string) (closer io.Closer) {
	// Sample configuration for testing. Use constant sampling to sample every trace
	// and enable LogSpan to log every span via configured Logger.
	cfg := jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:            true,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  endpoint,
		},
	}

	// Example logger and metrics factory. Use github.com/uber/jaeger-client-go/log
	// and github.com/uber/jaeger-lib/metrics respectively to bind to real logging and metrics
	// frameworks.
	//jLogger := jaegerlog.StdLogger
	//jMetricsFactory := metrics.NullFactory

	// Initialize tracer with a logger and a metrics factory
	closer, err := cfg.InitGlobalTracer(
		svc,
		//jaegercfg.Logger(jLogger),
		//jaegercfg.Metrics(jMetricsFactory),
		//jaegercfg.Observer(rpcmetrics.NewObserver(jMetricsFactory, rpcmetrics.DefaultNameNormalizer)),
	)
	if err != nil {
		log.Error("Could not initialize Jaeger tracer", "err", err)
	}

	return closer
}

// spanStore holds saved spans
type spanStore struct {
	spans sync.Map
}

// StartSaveSpan stores the span specified in the passed context for later retrieval
// The span object but be context value on the key StoreLabelId.
// It will be stored under the the following string key context.Value(StoreLabelId)|.|context.Value(StoreLabelMeta)
func StartSaveSpan(ctx context.Context) context.Context {
	if !Enabled {
		return ctx
	}
	traceId := ctx.Value(StoreLabelId)

	if traceId != nil {
		traceStr := traceId.(string)
		var sp opentracing.Span
		ctx, sp = spancontext.StartSpan(
			ctx,
			traceStr,
		)
		traceMeta := ctx.Value(StoreLabelMeta)
		if traceMeta != nil {
			traceStr = traceStr + "." + traceMeta.(string)
		}
		store.spans.Store(traceStr, sp)
	}
	return ctx
}

// ShiftSpanByKey retrieves the span stored under the key of the string given as argument
// The span is then deleted from the store
func ShiftSpanByKey(k string) opentracing.Span {
	if !Enabled {
		return nil
	}
	span, spanOk := store.spans.Load(k)
	if !spanOk {
		return nil
	}
	store.spans.Delete(k)
	return span.(opentracing.Span)
}

// FinishSpans calls `Finish()` on all stored spans
// It should be called on instance shutdown
func FinishSpans() {
	store.spans.Range(func(_, v interface{}) bool {
		v.(opentracing.Span).Finish()
		return true
	})
}
