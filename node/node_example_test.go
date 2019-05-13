// Copyright 2015 The go-ethereum Authors
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

package node_test

import (
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

// SampleService is a trivial network service that can be attached to a node for
// life cycle management.
//
// The following methods are needed to implement a node.Service:
//  - Protocols() []p2p.Protocol - devp2p protocols the service can communicate on
//  - APIs() []rpc.API           - api methods the service wants to expose on rpc channels
//  - Start() error              - method invoked when the node is ready to start the service
//  - Stop() error               - method invoked when the node terminates the service
type SampleService struct{}

func (s *SampleService) Protocols() []p2p.Protocol { return nil }
func (s *SampleService) APIs() []rpc.API           { return nil }
func (s *SampleService) Start(*p2p.Server) error   { fmt.Println("Service starting..."); return nil }
func (s *SampleService) Stop() error               { fmt.Println("Service stopping..."); return nil }

func ExampleService() {
	// Create a network node to run protocols with the default values.
	stack, err := node.New(&node.Config{})
	if err != nil {
		log.Fatalf("Failed to create network node: %v", err)
	}
	defer stack.Close()

	// Create and register a simple network service. This is done through the definition
	// of a node.ServiceConstructor that will instantiate a node.Service. The reason for
	// the factory method approach is to support service restarts without relying on the
	// individual implementations' support for such operations.
	constructor := func(context *node.ServiceContext) (node.Service, error) {
		return new(SampleService), nil
	}
	if err := stack.Register(constructor); err != nil {
		log.Fatalf("Failed to register service: %v", err)
	}
	// Boot up the entire protocol stack, do a restart and terminate
	if err := stack.Start(); err != nil {
		log.Fatalf("Failed to start the protocol stack: %v", err)
	}
	if err := stack.Restart(); err != nil {
		log.Fatalf("Failed to restart the protocol stack: %v", err)
	}
	if err := stack.Stop(); err != nil {
		log.Fatalf("Failed to stop the protocol stack: %v", err)
	}
	// Output:
	// Service starting...
	// Service stopping...
	// Service starting...
	// Service stopping...
}
