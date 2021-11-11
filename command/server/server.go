package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/command/server/proto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethstats"
	"github.com/ethereum/go-ethereum/graphql"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/influxdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"google.golang.org/grpc"
)

type Server struct {
	proto.UnimplementedBorServer
	node       *node.Node
	backend    *eth.Ethereum
	grpcServer *grpc.Server
}

func NewServer(config *Config) (*Server, error) {
	srv := &Server{}

	// start the logger
	setupLogger(config.LogLevel)

	if err := srv.setupGRPCServer(config.GRPC.Addr); err != nil {
		return nil, err
	}

	// load the chain genesis
	if err := config.loadChain(); err != nil {
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
	srv.node = stack

	// register the ethereum backend
	ethCfg, err := config.buildEth()
	if err != nil {
		return nil, err
	}
	backend, err := eth.New(stack, ethCfg)
	if err != nil {
		return nil, err
	}
	srv.backend = backend

	// debug tracing is enabled by default
	stack.RegisterAPIs(tracers.APIs(backend.APIBackend))

	// graphql is started from another place
	if config.JsonRPC.Graphql.Enabled {
		if err := graphql.New(stack, backend.APIBackend, config.JsonRPC.Cors, config.JsonRPC.Modules); err != nil {
			return nil, fmt.Errorf("failed to register the GraphQL service: %v", err)
		}
	}

	// register ethash service
	if config.Ethstats != "" {
		if err := ethstats.New(stack, backend.APIBackend, backend.Engine(), config.Ethstats); err != nil {
			return nil, err
		}
	}

	// setup account manager (only keystore)
	{
		keydir := stack.KeyStoreDir()
		n, p := keystore.StandardScryptN, keystore.StandardScryptP
		if config.Accounts.UseLightweightKDF {
			n, p = keystore.LightScryptN, keystore.LightScryptP
		}
		stack.AccountManager().AddBackend(keystore.NewKeyStore(keydir, n, p))
	}

	// sealing (if enabled)
	if config.Sealer.Enabled {
		if err := backend.StartMining(1); err != nil {
			return nil, err
		}
	}

	if err := srv.setupMetrics(config.Telemetry); err != nil {
		return nil, err
	}

	// start the node
	if err := srv.node.Start(); err != nil {
		return nil, err
	}
	return srv, nil
}

func (s *Server) Stop() {
	s.node.Close()
}

func (s *Server) setupMetrics(config *TelemetryConfig) error {
	metrics.Enabled = config.Enabled
	metrics.EnabledExpensive = config.Expensive

	if !metrics.Enabled {
		// metrics are disabled, do not set up any sink
		return nil
	}

	log.Info("Enabling metrics collection")

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

	return nil
}

func (s *Server) setupGRPCServer(addr string) error {
	s.grpcServer = grpc.NewServer(s.withLoggingUnaryInterceptor())
	proto.RegisterBorServer(s.grpcServer, s)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			log.Error("failed to serve grpc server", "err", err)
		}
	}()

	log.Info("GRPC Server started", "addr", addr)
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

func setupLogger(logLevel string) {
	output := io.Writer(os.Stderr)
	usecolor := (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
	if usecolor {
		output = colorable.NewColorableStderr()
	}
	ostream := log.StreamHandler(output, log.TerminalFormat(usecolor))
	glogger := log.NewGlogHandler(ostream)

	// logging
	lvl, err := log.LvlFromString(strings.ToLower(logLevel))
	if err == nil {
		glogger.Verbosity(lvl)
	} else {
		glogger.Verbosity(log.LvlInfo)
	}
	log.Root().SetHandler(glogger)
}
