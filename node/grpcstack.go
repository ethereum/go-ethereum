package node

import (
	"net"
	"sync"

	"github.com/ethereum/go-ethereum/grpc/execution"
	executionv1 "github.com/ethereum/go-ethereum/grpc/gen/proto/execution/v1"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"google.golang.org/grpc"
)

// GRPCServerHandler is the gRPC server handler.
// It gives us a way to attach the gRPC server to the node so it can be stopped on shutdown.
type GRPCServerHandler struct {
	mu sync.Mutex

	endpoint               string
	server                 *grpc.Server
	executionServiceServer *execution.ExecutionServiceServer
}

// NewServer creates a new gRPC server.
// It registers the execution service server.
// It registers the gRPC server with the node so it can be stopped on shutdown.
func NewGRPCServerHandler(node *Node, backend ethapi.Backend, cfg *Config) error {
	server := grpc.NewServer()

	executionServiceServer := &execution.ExecutionServiceServer{
		Backend: backend,
	}

	log.Info("gRPC server enabled", "endpoint", cfg.GRPCEndpoint())

	serverHandler := &GRPCServerHandler{
		endpoint:               cfg.GRPCEndpoint(),
		server:                 server,
		executionServiceServer: executionServiceServer,
	}

	executionv1.RegisterExecutionServiceServer(server, executionServiceServer)

	node.RegisterGRPCServer(serverHandler)
	return nil
}

// Start starts the gRPC server if it is enabled.
func (handler *GRPCServerHandler) Start() error {
	handler.mu.Lock()
	defer handler.mu.Unlock()

	if handler.endpoint == "" {
		return nil
	}

	// Start the gRPC server
	lis, err := net.Listen("tcp", handler.endpoint)
	if err != nil {
		return err
	}
	go handler.server.Serve(lis)
	log.Info("gRPC server started", "endpoint", handler.endpoint)
	return nil
}

// Stop stops the gRPC server.
func (handler *GRPCServerHandler) Stop() error {
	handler.mu.Lock()
	defer handler.mu.Unlock()

	handler.server.Stop()
	log.Info("gRPC server stopped", "endpoint", handler.endpoint)
	return nil
}
