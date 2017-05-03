// Copyright 2016 The go-ethereum Authors
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

package adapters

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/docker/docker/pkg/reexec"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rpc"
)

// Node represents a node in a simulation network which is created by a
// NodeAdapter, for example:
//
// * SimNode    - An in-memory node
// * ExecNode   - A child process node
// * DockerNode - A docker container node
//
type Node interface {
	// Addr returns the node's address (e.g. an Enode URL)
	Addr() []byte

	// Client returns the RPC client which is created once the node is
	// up and running
	Client() (*rpc.Client, error)

	// Start starts the node
	Start() error

	// Stop stops the node
	Stop() error

	// NodeInfo returns information about the node
	NodeInfo() *p2p.NodeInfo
}

// NodeAdapter is an object which creates Nodes to be used in a simulation
// network
type NodeAdapter interface {
	// Name returns the name of the adapter for logging purposes
	Name() string

	// NewNode creates a new node with the given configuration
	NewNode(config *NodeConfig) (Node, error)
}

// RunProtocol is a function which runs a p2p protocol (see p2p.Protocol.Run)
type RunProtocol func(*p2p.Peer, p2p.MsgReadWriter) error

// NodeId wraps a discover.NodeID with some convenience methods
type NodeId struct {
	discover.NodeID
}

func NewNodeId(id []byte) *NodeId {
	var n discover.NodeID
	copy(n[:], id)
	return &NodeId{n}
}

func NewNodeIdFromHex(s string) *NodeId {
	id := discover.MustHexID(s)
	return &NodeId{id}
}

func (self *NodeId) Bytes() []byte {
	return self.NodeID[:]
}

func (self *NodeId) Label() string {
	return self.String()[:4]
}

// NodeConfig is the configuration used to start a node in a simulation
// network
type NodeConfig struct {
	Id         *NodeId
	PrivateKey *ecdsa.PrivateKey

	// Service is the name of the service which should be run when starting
	// the node (for SimNodes it should be the name of a service contained
	// in SimAdapter.services, for other nodes it should be a service
	// registered by calling the RegisterService function)
	Service string
}

// nodeConfigJSON is used to encode and decode NodeConfig as JSON by converting
// all fields to strings
type nodeConfigJSON struct {
	Id         string `json:"id"`
	PrivateKey string `json:"private_key"`
	Service    string `json:"service"`
}

func (n *NodeConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(nodeConfigJSON{
		n.Id.String(),
		hex.EncodeToString(crypto.FromECDSA(n.PrivateKey)),
		n.Service,
	})
}

func (n *NodeConfig) UnmarshalJSON(data []byte) error {
	var confJSON nodeConfigJSON
	if err := json.Unmarshal(data, &confJSON); err != nil {
		return err
	}

	nodeID, err := discover.HexID(confJSON.Id)
	if err != nil {
		return err
	}
	n.Id = &NodeId{NodeID: nodeID}

	key, err := hex.DecodeString(confJSON.PrivateKey)
	if err != nil {
		return err
	}
	n.PrivateKey = crypto.ToECDSA(key)

	n.Service = confJSON.Service

	return nil
}

// RandomNodeConfig returns node configuration with a randomly generated ID and
// PrivateKey
func RandomNodeConfig() *NodeConfig {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("unable to generate key")
	}
	var id discover.NodeID
	pubkey := crypto.FromECDSAPub(&key.PublicKey)
	copy(id[:], pubkey[1:])
	return &NodeConfig{
		Id:         &NodeId{NodeID: id},
		PrivateKey: key,
	}
}

// Services is a collection of services which can be run in a simulation
type Services map[string]ServiceFunc

// ServiceFunc returns a node.Service which can be used to boot devp2p nodes
type ServiceFunc func(id *NodeId) node.Service

// serviceFuncs is a map of registered services which are used to boot devp2p
// nodes
var serviceFuncs = make(Services)

// RegisterServices registers the given ServiceFuncs which can then be used to
// start devp2p nodes
func RegisterServices(services Services) {
	for name, f := range services {
		if _, exists := serviceFuncs[name]; exists {
			panic(fmt.Sprintf("node service already exists: %q", name))
		}
		serviceFuncs[name] = f
	}

	// now we have registered the services, run reexec.Init() which will
	// potentially start one of the services if the current binary has
	// been exec'd as a p2p-node
	if reexec.Init() {
		os.Exit(0)
	}
}
