package server

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/consensus/beacon" //nolint:typecheck
	"github.com/ethereum/go-ethereum/consensus/bor"    //nolint:typecheck
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethstats"
	"github.com/ethereum/go-ethereum/graphql"
	"github.com/ethereum/go-ethereum/internal/cli/server/pprof"
	"github.com/ethereum/go-ethereum/internal/cli/server/proto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/influxdb"
	"github.com/ethereum/go-ethereum/metrics/prometheus"
	"github.com/ethereum/go-ethereum/node"

	// Force-load the tracer engines to trigger registration
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
)

type Server struct {
	proto.UnimplementedBorServer
	node       *node.Node
	backend    *eth.Ethereum
	grpcServer *grpc.Server
	tracer     *sdktrace.TracerProvider
	config     *Config

	// tracerAPI to trace block executions
	tracerAPI *tracers.API
}

type serverOption func(srv *Server, config *Config) error

var glogger *log.GlogHandler

func init() {
	handler := log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, false)
	log.SetDefault(log.NewLogger(handler))
}

func WithGRPCAddress() serverOption {
	return func(srv *Server, config *Config) error {
		return srv.gRPCServerByAddress(config.GRPC.Addr)
	}
}

func WithGRPCListener(lis net.Listener) serverOption {
	return func(srv *Server, _ *Config) error {
		return srv.gRPCServerByListener(lis)
	}
}

func VerbosityIntToString(verbosity int) string {
	mapIntToString := map[int]string{
		5: "trace",
		4: "debug",
		3: "info",
		2: "warn",
		1: "error",
		0: "crit",
	}

	return mapIntToString[verbosity]
}

func VerbosityStringToInt(loglevel string) int {
	mapStringToInt := map[string]int{
		"trace": 5,
		"debug": 4,
		"info":  3,
		"warn":  2,
		"error": 1,
		"crit":  0,
	}

	return mapStringToInt[loglevel]
}

//nolint:gocognit
func NewServer(config *Config, opts ...serverOption) (*Server, error) {
	// start pprof
	if config.Pprof.Enabled {
		pprof.SetMemProfileRate(config.Pprof.MemProfileRate)
		pprof.SetSetBlockProfileRate(config.Pprof.BlockProfileRate)
		pprof.StartPProf(fmt.Sprintf("%s:%d", config.Pprof.Addr, config.Pprof.Port))
	}

	runtime.SetMutexProfileFraction(5)

	srv := &Server{
		config: config,
	}

	// start the logger
	setupLogger(config.Verbosity, *config.Logging)

	var err error

	for _, opt := range opts {
		err = opt(srv, config)
		if err != nil {
			return nil, err
		}
	}

	// load the chain genesis
	if err = config.loadChain(); err != nil {
		return nil, err
	}

	// create the node/stack
	nodeCfg, err := config.buildNode()
	if err != nil {
		return nil, err
	}

	stack, err := node.New(nodeCfg)
	if err != nil {
		return nil, err
	}

	// setup account manager (only keystore)
	// create a new account manager, only for the scope of this function
	accountManager := accounts.NewManager(&accounts.Config{})

	// register backend to account manager with keystore for signing
	keydir := stack.KeyStoreDir()

	n, p := keystore.StandardScryptN, keystore.StandardScryptP
	if config.Accounts.UseLightweightKDF {
		n, p = keystore.LightScryptN, keystore.LightScryptP
	}

	// proceed to authorize the local account manager in any case
	accountManager.AddBackend(keystore.NewKeyStore(keydir, n, p))

	// flag to set if we're authorizing consensus here
	authorized := false

	var ethCfg *ethconfig.Config

	// check if personal wallet endpoints are disabled or not
	// nolint:nestif
	if !config.Accounts.DisableBorWallet {
		// add keystore globally to the node's account manager if personal wallet is enabled
		stack.AccountManager().AddBackend(keystore.NewKeyStore(keydir, n, p))

		// register the ethereum backend
		ethCfg, err = config.buildEth(stack, stack.AccountManager())
		if err != nil {
			return nil, err
		}

		backend, err := eth.New(stack, ethCfg)
		if err != nil {
			return nil, err
		}

		srv.backend = backend
	} else {
		// register the ethereum backend (with temporary created account manager)
		ethCfg, err = config.buildEth(stack, accountManager)
		if err != nil {
			return nil, err
		}

		backend, err := eth.New(stack, ethCfg)
		if err != nil {
			return nil, err
		}

		srv.backend = backend

		// authorize only if mining or in developer mode
		if config.Sealer.Enabled || config.Developer.Enabled {
			// get the etherbase
			eb, err := srv.backend.Etherbase()
			if err != nil {
				log.Error("Cannot start mining without etherbase", "err", err)

				return nil, fmt.Errorf("etherbase missing: %v", err)
			}

			// Authorize the clique consensus (if chosen) to sign using wallet signer
			var cli *clique.Clique
			if c, ok := srv.backend.Engine().(*clique.Clique); ok {
				cli = c
			} else if cl, ok := srv.backend.Engine().(*beacon.Beacon); ok {
				if c, ok := cl.InnerEngine().(*clique.Clique); ok {
					cli = c
				}
			}

			if cli != nil {
				wallet, err := accountManager.Find(accounts.Account{Address: eb})
				if wallet == nil || err != nil {
					log.Error("Etherbase account unavailable locally", "err", err)
					return nil, fmt.Errorf("signer missing: %v", err)
				}

				cli.Authorize(eb, wallet.SignData)

				authorized = true
			}

			// Authorize the bor consensus (if chosen) to sign using wallet signer
			if bor, ok := srv.backend.Engine().(*bor.Bor); ok {
				wallet, err := accountManager.Find(accounts.Account{Address: eb})
				if wallet == nil || err != nil {
					log.Error("Etherbase account unavailable locally", "err", err)
					return nil, fmt.Errorf("signer missing: %v", err)
				}

				bor.Authorize(eb, wallet.SignData)

				authorized = true
			}
		}
	}

	// set the auth status in backend
	srv.backend.SetAuthorized(authorized)

	filterSystem := utils.RegisterFilterAPI(stack, srv.backend.APIBackend, ethCfg)

	// debug tracing is enabled by default
	stack.RegisterAPIs(tracers.APIs(srv.backend.APIBackend))
	srv.tracerAPI = tracers.NewAPI(srv.backend.APIBackend)

	// graphql is started from another place
	if config.JsonRPC.Graphql.Enabled {
		if err := graphql.New(stack, srv.backend.APIBackend, filterSystem, config.JsonRPC.Graphql.Cors, config.JsonRPC.Graphql.VHost); err != nil {
			return nil, fmt.Errorf("failed to register the GraphQL service: %v", err)
		}
	}

	// register ethash service
	if config.Ethstats != "" {
		if err := ethstats.New(stack, srv.backend.APIBackend, srv.backend.Engine(), config.Ethstats); err != nil {
			return nil, err
		}
	}

	// sealing (if enabled) or in dev mode
	if config.Sealer.Enabled || config.Developer.Enabled {
		if err := srv.backend.StartMining(); err != nil {
			return nil, err
		}
	}

	if err := srv.setupMetrics(config.Telemetry, config.Identity); err != nil {
		return nil, err
	}

	// Set the node instance
	srv.node = stack

	// start the node
	if err := srv.node.Start(); err != nil {
		return nil, err
	}

	return srv, nil
}

func (s *Server) Stop() {
	if s.node != nil {
		s.node.Close()
	}

	if s.grpcServer != nil {
		s.grpcServer.Stop()
	}

	// shutdown the tracer
	if s.tracer != nil {
		if err := s.tracer.Shutdown(context.Background()); err != nil {
			log.Error("Failed to shutdown open telemetry tracer")
		}
	}
}

func (s *Server) setupMetrics(config *TelemetryConfig, serviceName string) error {
	// Check the global metrics if they're matching with the provided config
	if metrics.Enabled != config.Enabled || metrics.EnabledExpensive != config.Expensive {
		log.Warn(
			"Metric misconfiguration, some of them might not be visible",
			"metrics", metrics.Enabled,
			"config.metrics", config.Enabled,
			"expensive", metrics.EnabledExpensive,
			"config.expensive", config.Expensive,
		)
	}

	// Update the values anyways (for services which don't need immediate attention)
	metrics.Enabled = config.Enabled
	metrics.EnabledExpensive = config.Expensive

	if !metrics.Enabled {
		// metrics are disabled, do not set up any sink
		return nil
	}

	log.Info("Enabling metrics collection")

	if metrics.EnabledExpensive {
		log.Info("Enabling expensive metrics collection")
	}

	// influxdb
	if v1Enabled, v2Enabled := config.InfluxDB.V1Enabled, config.InfluxDB.V2Enabled; v1Enabled || v2Enabled {
		if v1Enabled && v2Enabled {
			return fmt.Errorf("both influx v1 and influx v2 cannot be enabled")
		}

		cfg := config.InfluxDB
		tags := cfg.Tags
		endpoint := cfg.Endpoint

		if v1Enabled {
			log.Info("Enabling metrics export to InfluxDB (v1)")

			go influxdb.InfluxDBWithTags(metrics.DefaultRegistry, 10*time.Second, endpoint, cfg.Database, cfg.Username, cfg.Password, "geth.", tags)
		}

		if v2Enabled {
			log.Info("Enabling metrics export to InfluxDB (v2)")

			go influxdb.InfluxDBV2WithTags(metrics.DefaultRegistry, 10*time.Second, endpoint, cfg.Token, cfg.Bucket, cfg.Organization, "geth.", tags)
		}
	}

	// Start system runtime metrics collection
	go metrics.CollectProcessMetrics(3 * time.Second)

	if config.PrometheusAddr != "" {
		prometheusMux := http.NewServeMux()

		prometheusMux.Handle("/debug/metrics/prometheus", prometheus.Handler(metrics.DefaultRegistry))

		promServer := &http.Server{
			Addr:    config.PrometheusAddr,
			Handler: prometheusMux,
		}

		go func() {
			if err := promServer.ListenAndServe(); err != nil {
				log.Error("Failure in running Prometheus server", "err", err)
			}
		}()

		log.Info("Enabling metrics export to prometheus", "path", fmt.Sprintf("http://%s/debug/metrics/prometheus", config.PrometheusAddr))
	}

	if config.OpenCollectorEndpoint != "" {
		// setup open collector tracer
		ctx := context.Background()

		res, err := resource.New(ctx,
			resource.WithAttributes(
				// the service name used to display traces in backends
				semconv.ServiceNameKey.String(serviceName),
			),
		)
		if err != nil {
			return fmt.Errorf("failed to create open telemetry resource for service: %v", err)
		}

		// Set up a trace exporter
		traceExporter, err := otlptracegrpc.New(
			ctx,
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(config.OpenCollectorEndpoint),
		)
		if err != nil {
			return fmt.Errorf("failed to create open telemetry tracer exporter for service: %v", err)
		}

		// Register the trace exporter with a TracerProvider, using a batch
		// span processor to aggregate spans before export.
		bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
		tracerProvider := sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithResource(res),
			sdktrace.WithSpanProcessor(bsp),
		)
		otel.SetTracerProvider(tracerProvider)

		// set global propagator to tracecontext (the default is no-op).
		otel.SetTextMapPropagator(propagation.TraceContext{})

		// set the tracer
		s.tracer = tracerProvider

		log.Info("Open collector tracing started", "address", config.OpenCollectorEndpoint)
	}

	return nil
}

func (s *Server) gRPCServerByAddress(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	return s.gRPCServerByListener(lis)
}

func (s *Server) gRPCServerByListener(listener net.Listener) error {
	s.grpcServer = grpc.NewServer(s.withLoggingUnaryInterceptor())
	proto.RegisterBorServer(s.grpcServer, s)

	go func() {
		if err := s.grpcServer.Serve(listener); err != nil {
			log.Error("failed to serve grpc server", "err", err)
		}
	}()

	log.Info("GRPC Server started", "addr", listener.Addr())

	return nil
}

func (s *Server) withLoggingUnaryInterceptor() grpc.ServerOption {
	return grpc.UnaryInterceptor(s.loggingServerInterceptor)
}

func (s *Server) loggingServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	h, err := handler(ctx, req)

	log.Trace("Request", "method", info.FullMethod, "duration", time.Since(start), "error", err)

	return h, err
}

func setupLogger(logLevel int, loggingInfo LoggingConfig) {
	output := io.Writer(os.Stderr)

	if loggingInfo.Json {
		glogger = log.NewGlogHandler(log.JSONHandler(os.Stderr))
	} else {
		usecolor := (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
		if usecolor {
			output = colorable.NewColorableStderr()
		}

		glogger = log.NewGlogHandler(log.NewTerminalHandler(output, usecolor))
	}

	// logging
	lvl := log.FromLegacyLevel(logLevel)
	glogger.Verbosity(lvl)

	if loggingInfo.Vmodule != "" {
		if err := glogger.Vmodule(loggingInfo.Vmodule); err != nil {
			log.Error("failed to set Vmodule", "err", err)
		}
	}

	log.SetDefault(log.NewLogger(glogger))
}

func (s *Server) GetLatestBlockNumber() *big.Int {
	return s.backend.BlockChain().CurrentBlock().Number
}

func (s *Server) GetGrpcAddr() string {
	return s.config.GRPC.Addr[1:]
}
