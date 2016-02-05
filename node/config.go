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

package node

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/nat"
)

var (
	datadirPrivateKey   = "nodekey"            // Path within the datadir to the node's private key
	datadirStaticNodes  = "static-nodes.json"  // Path within the datadir to the static node list
	datadirTrustedNodes = "trusted-nodes.json" // Path within the datadir to the trusted node list
	datadirNodeDatabase = "nodes"              // Path within the datadir to store the node infos
)

// Config represents a small collection of configuration values to fine tune the
// P2P network layer of a protocol stack. These values can be further extended by
// all registered services.
type Config struct {
	// DataDir is the file system folder the node should use for any data storage
	// requirements. The configured data directory will not be directly shared with
	// registered services, instead those can use utility methods to create/access
	// databases or flat files. This enables ephemeral nodes which can fully reside
	// in memory.
	DataDir string

	// IpcPath is the requested location to place the IPC endpoint. If the path is
	// a simple file name, it is placed inside the data directory (or on the root
	// pipe path on Windows), whereas if it's a resolvable path name (absolute or
	// relative), then that specific path is enforced. An empty path disables IPC.
	IpcPath string

	// This field should be a valid secp256k1 private key that will be used for both
	// remote peer identification as well as network traffic encryption. If no key
	// is configured, the preset one is loaded from the data dir, generating it if
	// needed.
	PrivateKey *ecdsa.PrivateKey

	// Name sets the node name of this server. Use common.MakeName to create a name
	// that follows existing conventions.
	Name string

	// NoDiscovery specifies whether the peer discovery mechanism should be started
	// or not. Disabling is usually useful for protocol debugging (manual topology).
	NoDiscovery bool

	// Bootstrap nodes used to establish connectivity with the rest of the network.
	BootstrapNodes []*discover.Node

	// Network interface address on which the node should listen for inbound peers.
	ListenAddr string

	// If set to a non-nil value, the given NAT port mapper is used to make the
	// listening port available to the Internet.
	NAT nat.Interface

	// If Dialer is set to a non-nil value, the given Dialer is used to dial outbound
	// peer connections.
	Dialer *net.Dialer

	// If NoDial is true, the node will not dial any peers.
	NoDial bool

	// MaxPeers is the maximum number of peers that can be connected. If this is
	// set to zero, then only the configured static and trusted peers can connect.
	MaxPeers int

	// MaxPendingPeers is the maximum number of peers that can be pending in the
	// handshake phase, counted separately for inbound and outbound connections.
	// Zero defaults to preset values.
	MaxPendingPeers int

	// HttpHost is the host interface on which to start the HTTP RPC server. If this
	// field is empty, no HTTP API endpoint will be started.
	HttpHost string

	// HttpPort is the TCP port number on which to start the HTTP RPC server. The
	// default zero value is/ valid and will pick a port number randomly (useful
	// for ephemeral nodes).
	HttpPort int

	// HttpCors is the Cross-Origin Resource Sharing header to send to requesting
	// clients. Please be aware that CORS is a browser enforced security, it's fully
	// useless for custom HTTP clients.
	HttpCors string

	// HttpModules is a list of API modules to expose via the HTTP RPC interface.
	// If the module list is empty, all RPC API endpoints designated public will be
	// exposed.
	HttpModules []string

	// WsHost is the host interface on which to start the websocket RPC server. If
	// this field is empty, no websocket API endpoint will be started.
	WsHost string

	// WsPort is the TCP port number on which to start the websocket RPC server. The
	// default zero value is/ valid and will pick a port number randomly (useful for
	// ephemeral nodes).
	WsPort int

	// WsCors is the Cross-Origin Resource Sharing header to send to requesting clients.
	// Please be aware that CORS is a browser enforced security, it's fully useless
	// for custom websocket clients.
	WsCors string

	// WsModules is a list of API modules to expose via the websocket RPC interface.
	// If the module list is empty, all RPC API endpoints designated public will be
	// exposed.
	WsModules []string
}

// IpcEndpoint resolves an IPC endpoint based on a configured value, taking into
// account the set data folders as well as the designated platform we're currently
// running on.
func (c *Config) IpcEndpoint() string {
	// Short circuit if IPC has not been enabled
	if c.IpcPath == "" {
		return ""
	}
	// On windows we can only use plain top-level pipes
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(c.IpcPath, `\\.\pipe\`) {
			return c.IpcPath
		}
		return `\\.\pipe\` + c.IpcPath
	}
	// Resolve names into the data directory full paths otherwise
	if filepath.Base(c.IpcPath) == c.IpcPath {
		if c.DataDir == "" {
			return filepath.Join(os.TempDir(), c.IpcPath)
		}
		return filepath.Join(c.DataDir, c.IpcPath)
	}
	return c.IpcPath
}

// DefaultIpcEndpoint returns the IPC path used by default.
func DefaultIpcEndpoint() string {
	config := &Config{DataDir: common.DefaultDataDir(), IpcPath: common.DefaultIpcSocket}
	return config.IpcEndpoint()
}

// HttpEndpoint resolves an HTTP endpoint based on the configured host interface
// and port parameters.
func (c *Config) HttpEndpoint() string {
	if c.HttpHost == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", c.HttpHost, c.HttpPort)
}

// DefaultHttpEndpoint returns the HTTP endpoint used by default.
func DefaultHttpEndpoint() string {
	config := &Config{HttpHost: common.DefaultHttpHost, HttpPort: common.DefaultHttpPort}
	return config.HttpEndpoint()
}

// WsEndpoint resolves an websocket endpoint based on the configured host interface
// and port parameters.
func (c *Config) WsEndpoint() string {
	if c.WsHost == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", c.WsHost, c.WsPort)
}

// DefaultWsEndpoint returns the websocket endpoint used by default.
func DefaultWsEndpoint() string {
	config := &Config{WsHost: common.DefaultWsHost, WsPort: common.DefaultWsPort}
	return config.WsEndpoint()
}

// NodeKey retrieves the currently configured private key of the node, checking
// first any manually set key, falling back to the one found in the configured
// data folder. If no key can be found, a new one is generated.
func (c *Config) NodeKey() *ecdsa.PrivateKey {
	// Use any specifically configured key
	if c.PrivateKey != nil {
		return c.PrivateKey
	}
	// Generate ephemeral key if no datadir is being used
	if c.DataDir == "" {
		key, err := crypto.GenerateKey()
		if err != nil {
			glog.Fatalf("Failed to generate ephemeral node key: %v", err)
		}
		return key
	}
	// Fall back to persistent key from the data directory
	keyfile := filepath.Join(c.DataDir, datadirPrivateKey)
	if key, err := crypto.LoadECDSA(keyfile); err == nil {
		return key
	}
	// No persistent key found, generate and store a new one
	key, err := crypto.GenerateKey()
	if err != nil {
		glog.Fatalf("Failed to generate node key: %v", err)
	}
	if err := crypto.SaveECDSA(keyfile, key); err != nil {
		glog.V(logger.Error).Infof("Failed to persist node key: %v", err)
	}
	return key
}

// StaticNodes returns a list of node enode URLs configured as static nodes.
func (c *Config) StaticNodes() []*discover.Node {
	return c.parsePersistentNodes(datadirStaticNodes)
}

// TrusterNodes returns a list of node enode URLs configured as trusted nodes.
func (c *Config) TrusterNodes() []*discover.Node {
	return c.parsePersistentNodes(datadirTrustedNodes)
}

// parsePersistentNodes parses a list of discovery node URLs loaded from a .json
// file from within the data directory.
func (c *Config) parsePersistentNodes(file string) []*discover.Node {
	// Short circuit if no node config is present
	if c.DataDir == "" {
		return nil
	}
	path := filepath.Join(c.DataDir, file)
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	// Load the nodes from the config file
	blob, err := ioutil.ReadFile(path)
	if err != nil {
		glog.V(logger.Error).Infof("Failed to access nodes: %v", err)
		return nil
	}
	nodelist := []string{}
	if err := json.Unmarshal(blob, &nodelist); err != nil {
		glog.V(logger.Error).Infof("Failed to load nodes: %v", err)
		return nil
	}
	// Interpret the list as a discovery node array
	var nodes []*discover.Node
	for _, url := range nodelist {
		if url == "" {
			continue
		}
		node, err := discover.ParseNode(url)
		if err != nil {
			glog.V(logger.Error).Infof("Node URL %s: %v\n", url, err)
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes
}
