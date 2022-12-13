package server

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common/network"
)

func CreateMockServer(config *Config) (*Server, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// get grpc port and listener
	grpcPort, gRPCListener, err := network.FindAvailablePort()
	if err != nil {
		return nil, err
	}

	// The test uses grpc port from config so setting it here.
	config.GRPC.Addr = fmt.Sprintf(":%d", grpcPort)

	// datadir
	datadir, err := os.MkdirTemp("", "bor-cli-test")
	if err != nil {
		return nil, err
	}

	config.DataDir = datadir
	config.JsonRPC.Http.Port = 0 // It will choose a free/available port

	// start the server
	return NewServer(config, WithGRPCListener(gRPCListener))
}

func CloseMockServer(server *Server) {
	// remove the contents of temp data dir
	os.RemoveAll(server.config.DataDir)

	// close the server
	server.Stop()
}
