package simulation

import (
	"io"
)

// Node is a node within a simulation
type Node interface {
	Info() NodeInfo
	// Start starts the node
	Start() error
	// Stop stops the node
	Stop() error
	// Snapshot returns a snapshot of the node
	Snapshot() (NodeSnapshot, error)
}

// Adapter can handle Node creation
type Adapter interface {
	// NewNode creates a new node based on the NodeConfig
	NewNode(config NodeConfig) Node
	// Snapshot returns a snapshot of the adapter
	Snapshot() AdapterSnapshot
}

// NodeID is the node identifier within a simulation. This can be an arbitrary string.
type NodeID string

// NodeConfig is the configuration of a specific node
type NodeConfig struct {
	// Arbitrary string used to identify a node
	ID NodeID `json:"id"`
	// Command line arguments
	Args []string `json:"args"`
	// Environment variables
	Env []string `json:"env,omitempty"`
	// Stdout and Stderr specify the nodes' standard output and error
	Stdout io.Writer `json:"-"`
	Stderr io.Writer `json:"-"`
}

// NodeInfo contains the nodes information and connections strings
type NodeInfo struct {
	ID      NodeID
	Enode   string
	BzzAddr string

	RPCListen   string // RPC listener address. Should be a valid ipc or websocket path
	HTTPListen  string // HTTP listener address: e.g. http://localhost:8500
	PprofListen string // PProf listener address: e.g http://localhost:6060
}

// Snapshot is a snapshot of a simulation. It contains snapshots of:
// - the default adapter that the simulation was initialized with
// - the list of nodes that were created within the simulation
// - the list of connections between nodes
type Snapshot struct {
	DefaultAdapter *AdapterSnapshot     `json:"defaultAdapter"`
	Nodes          []NodeSnapshot       `json:"nodes"`
	Connections    []ConnectionSnapshot `json:"connections"`
}

// NodeSnapshot is a snapshot of the node, it contains the node configuration and an adapter snapshot
type NodeSnapshot struct {
	Config  NodeConfig       `json:"config"`
	Adapter *AdapterSnapshot `json:"adapter,omitempty"`
}

// ConnectionSnapshot is a snapshot of a connection between peers
type ConnectionSnapshot struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// AdapterSnapshot is a snapshot of the configuration of an adapter
// - The type can be an arbitrary strings, e.g. "exec", "docker", etc.
// - The config will depend on the type, as every adapter has different configuration options
type AdapterSnapshot struct {
	Type   string      `json:"type"`
	Config interface{} `json:"config"`
}
