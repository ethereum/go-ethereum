// Copyright 2017 The go-ethereum Authors
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
	"log/slog"
	"os"

	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/node"
	"github.com/XinFinOrg/XDPoSChain/p2p"
	"github.com/XinFinOrg/XDPoSChain/p2p/discover"
	"github.com/XinFinOrg/XDPoSChain/rpc"
	"github.com/docker/docker/pkg/reexec"
	"github.com/gorilla/websocket"
)

// Node represents a node in a simulation network which is created by a
// NodeAdapter, for example:
//
// * SimNode    - An in-memory node
// * ExecNode   - A child process node
// * DockerNode - A Docker container node
type Node interface {
	// Addr returns the node's address (e.g. an Enode URL)
	Addr() []byte

	// Client returns the RPC client which is created once the node is
	// up and running
	Client() (*rpc.Client, error)

	// ServeRPC serves RPC requests over the given connection
	ServeRPC(*websocket.Conn) error

	// Start starts the node with the given snapshots
	Start(snapshots map[string][]byte) error

	// Stop stops the node
	Stop() error

	// NodeInfo returns information about the node
	NodeInfo() *p2p.NodeInfo

	// Snapshots creates snapshots of the running services
	Snapshots() (map[string][]byte, error)
}

// NodeAdapter is used to create Nodes in a simulation network
type NodeAdapter interface {
	// Name returns the name of the adapter for logging purposes
	Name() string

	// NewNode creates a new node with the given configuration
	NewNode(config *NodeConfig) (Node, error)
}

// NodeConfig is the configuration used to start a node in a simulation
// network
type NodeConfig struct {
	// ID is the node's ID which is used to identify the node in the
	// simulation network
	ID discover.NodeID

	// PrivateKey is the node's private key which is used by the devp2p
	// stack to encrypt communications
	PrivateKey *ecdsa.PrivateKey

	// Enable peer events for Msgs
	EnableMsgEvents bool

	// Name is a human friendly name for the node like "node01"
	Name string

	// Lifecycles are the names of the service lifecycles which should be run when
	// starting the node (for SimNodes it should be the names of service lifecycles
	// contained in SimAdapter.lifecycles, for other nodes it should be
	// service lifecycles registered by calling the RegisterLifecycle function)
	Lifecycles []string

	// function to sanction or prevent suggesting a peer
	Reachable func(id discover.NodeID) bool

	// LogFile is the log file name of the p2p node at runtime.
	//
	// The default value is empty so that the default log writer
	// is the system standard output.
	LogFile string

	// LogVerbosity is the log verbosity of the p2p node at runtime.
	//
	// The default verbosity is INFO.
	LogVerbosity slog.Level
}

// nodeConfigJSON is used to encode and decode NodeConfig as JSON by encoding
// all fields as strings
type nodeConfigJSON struct {
	ID           string   `json:"id"`
	PrivateKey   string   `json:"private_key"`
	Name         string   `json:"name"`
	Services     []string `json:"services"`
	LogFile      string   `json:"logfile"`
	LogVerbosity int      `json:"log_verbosity"`
}

// MarshalJSON implements the json.Marshaler interface by encoding the config
// fields as strings
func (n *NodeConfig) MarshalJSON() ([]byte, error) {
	confJSON := nodeConfigJSON{
		ID:           n.ID.String(),
		Name:         n.Name,
		Services:     n.Lifecycles,
		LogFile:      n.LogFile,
		LogVerbosity: int(n.LogVerbosity),
	}
	if n.PrivateKey != nil {
		confJSON.PrivateKey = hex.EncodeToString(crypto.FromECDSA(n.PrivateKey))
	}
	return json.Marshal(confJSON)
}

// UnmarshalJSON implements the json.Unmarshaler interface by decoding the json
// string values into the config fields
func (n *NodeConfig) UnmarshalJSON(data []byte) error {
	var confJSON nodeConfigJSON
	if err := json.Unmarshal(data, &confJSON); err != nil {
		return err
	}

	if confJSON.ID != "" {
		nodeID, err := discover.HexID(confJSON.ID)
		if err != nil {
			return err
		}
		n.ID = nodeID
	}

	if confJSON.PrivateKey != "" {
		key, err := hex.DecodeString(confJSON.PrivateKey)
		if err != nil {
			return err
		}
		privKey, err := crypto.ToECDSA(key)
		if err != nil {
			return err
		}
		n.PrivateKey = privKey
	}

	n.Name = confJSON.Name
	n.Lifecycles = confJSON.Services
	n.LogFile = confJSON.LogFile
	n.LogVerbosity = slog.Level(confJSON.LogVerbosity)

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
		ID:         id,
		PrivateKey: key,
	}
}

// ServiceContext is a collection of options and methods which can be utilised
// when starting services
type ServiceContext struct {
	RPCDialer

	Config   *NodeConfig
	Snapshot []byte
}

// RPCDialer is used when initialising services which need to connect to
// other nodes in the network (for example a simulated Swarm node which needs
// to connect to a Geth node to resolve ENS names)
type RPCDialer interface {
	DialRPC(id discover.NodeID) (*rpc.Client, error)
}

// LifecycleConstructor allows a Lifecycle to be constructed during node start-up.
// While the service-specific package usually takes care of Lifecycle creation and registration,
// for testing purposes, it is useful to be able to construct a Lifecycle on spot.
type LifecycleConstructor func(ctx *ServiceContext, stack *node.Node) (node.Lifecycle, error)

// LifecycleConstructors stores LifecycleConstructor functions to call during node start-up.
type LifecycleConstructors map[string]LifecycleConstructor

// lifecycleConstructorFuncs is a map of registered services which are used to boot devp2p
// nodes
var lifecycleConstructorFuncs = make(LifecycleConstructors)

// RegisterLifecycles registers the given Services which can then be used to
// start devp2p nodes using either the Exec or Docker adapters.
//
// It should be called in an init function so that it has the opportunity to
// execute the services before main() is called.
func RegisterLifecycles(lifecycles LifecycleConstructors) {
	for name, f := range lifecycles {
		if _, exists := lifecycleConstructorFuncs[name]; exists {
			panic(fmt.Sprintf("node service already exists: %q", name))
		}
		lifecycleConstructorFuncs[name] = f
	}

	// now we have registered the services, run reexec.Init() which will
	// potentially start one of the services if the current binary has
	// been exec'd with argv[0] set to "p2p-node"
	if reexec.Init() {
		os.Exit(0)
	}
}
