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
	"github.com/ethereum/go-ethereum/p2p/discover"
	rpc "github.com/ethereum/go-ethereum/rpc/v2"
)

// SampleService is a trivial network service that can be attached to a node for
// life cycle management.
//
// The following methods are needed to implement a node.Service:
//  - Protocols() []p2p.Protocol - devp2p protocols the service can communicate on
//  - Start() error              - method invoked when the node is ready to start the service
//  - Stop() error               - method invoked when the node terminates the service
type SampleService struct{}

func (s *SampleService) Protocols() []p2p.Protocol { return nil }
func (s *SampleService) APIs() []rpc.API           { return nil }
func (s *SampleService) Start(*p2p.Server) error   { fmt.Println("Service starting..."); return nil }
func (s *SampleService) Stop() error               { fmt.Println("Service stopping..."); return nil }

func ExampleUsage() {
	// Create a network node to run protocols with the default values. The below list
	// is only used to display each of the configuration options. All of these could
	// have been ommited if the default behavior is desired.
	nodeConfig := &node.Config{
		DataDir:         "",                 // Empty uses ephemeral storage
		PrivateKey:      nil,                // Nil generates a node key on the fly
		Name:            "",                 // Any textual node name is allowed
		NoDiscovery:     false,              // Can disable discovering remote nodes
		BootstrapNodes:  []*discover.Node{}, // List of bootstrap nodes to use
		ListenAddr:      ":0",               // Network interface to listen on
		NAT:             nil,                // UPnP port mapper to use for crossing firewalls
		Dialer:          nil,                // Custom dialer to use for establishing peer connections
		NoDial:          false,              // Can prevent this node from dialing out
		MaxPeers:        0,                  // Number of peers to allow
		MaxPendingPeers: 0,                  // Number of peers allowed to handshake concurrently
	}
	stack, err := node.New(nodeConfig)
	if err != nil {
		log.Fatalf("Failed to create network node: %v", err)
	}
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
