// Copyright 2024 The go-ethereum Authors
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

package grpc

import (
	"fmt"
	"net"

	"github.com/ethereum/go-ethereum/log"
	"google.golang.org/grpc"
)

// Service implements the node.Lifecycle interface for the gRPC server.
type Service struct {
	backend  Backend
	server   *grpc.Server
	listener net.Listener
	host     string
	port     int
}

// NewService creates a new gRPC service.
func NewService(backend Backend, host string, port int) *Service {
	return &Service{
		backend: backend,
		host:    host,
		port:    port,
	}
}

// Start implements node.Lifecycle, starting the gRPC server.
func (s *Service) Start() error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = lis

	// Create gRPC server with options
	s.server = grpc.NewServer(
		grpc.MaxRecvMsgSize(100*1024*1024), // 100MB for large bundles
		grpc.MaxSendMsgSize(100*1024*1024), // 100MB for large simulation results
	)

	// Register TraderService
	traderServer := NewTraderServer(s.backend)
	RegisterTraderServiceServer(s.server, traderServer)

	// Start serving in a goroutine
	go func() {
		log.Info("gRPC server started", "addr", addr)
		if err := s.server.Serve(lis); err != nil {
			log.Error("gRPC server failed", "err", err)
		}
	}()

	return nil
}

// Stop implements node.Lifecycle, stopping the gRPC server.
func (s *Service) Stop() error {
	if s.server != nil {
		log.Info("gRPC server stopping")
		s.server.GracefulStop()
		log.Info("gRPC server stopped")
	}
	return nil
}

