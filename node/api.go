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
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"
)

// PrivateAdminAPI is the collection of administrative API methods exposed only
// over a secure RPC channel.
type PrivateAdminAPI struct {
	node *Node // Node interfaced by this API
}

// NewPrivateAdminAPI creates a new API definition for the private admin methods
// of the node itself.
func NewPrivateAdminAPI(node *Node) *PrivateAdminAPI {
	return &PrivateAdminAPI{node: node}
}

// AddPeer requests connecting to a remote node, and also maintaining the new
// connection at all times, even reconnecting if it is lost.
func (api *PrivateAdminAPI) AddPeer(url string) (bool, error) {
	// Make sure the server is running, fail otherwise
	server := api.node.Server()
	if server == nil {
		return false, ErrNodeStopped
	}
	// Try to add the url as a static peer and return
	node, err := enode.Parse(enode.ValidSchemes, url)
	if err != nil {
		return false, fmt.Errorf("invalid enode: %v", err)
	}
	server.AddPeer(node)
	return true, nil
}

// RemovePeer disconnects from a remote node if the connection exists
func (api *PrivateAdminAPI) RemovePeer(url string) (bool, error) {
	// Make sure the server is running, fail otherwise
	server := api.node.Server()
	if server == nil {
		return false, ErrNodeStopped
	}
	// Try to remove the url as a static peer and return
	node, err := enode.Parse(enode.ValidSchemes, url)
	if err != nil {
		return false, fmt.Errorf("invalid enode: %v", err)
	}
	server.RemovePeer(node)
	return true, nil
}

// AddTrustedPeer allows a remote node to always connect, even if slots are full
func (api *PrivateAdminAPI) AddTrustedPeer(url string) (bool, error) {
	// Make sure the server is running, fail otherwise
	server := api.node.Server()
	if server == nil {
		return false, ErrNodeStopped
	}
	node, err := enode.Parse(enode.ValidSchemes, url)
	if err != nil {
		return false, fmt.Errorf("invalid enode: %v", err)
	}
	server.AddTrustedPeer(node)
	return true, nil
}

// RemoveTrustedPeer removes a remote node from the trusted peer set, but it
// does not disconnect it automatically.
func (api *PrivateAdminAPI) RemoveTrustedPeer(url string) (bool, error) {
	// Make sure the server is running, fail otherwise
	server := api.node.Server()
	if server == nil {
		return false, ErrNodeStopped
	}
	node, err := enode.Parse(enode.ValidSchemes, url)
	if err != nil {
		return false, fmt.Errorf("invalid enode: %v", err)
	}
	server.RemoveTrustedPeer(node)
	return true, nil
}

// PeerEvents creates an RPC subscription which receives peer events from the
// node's p2p.Server
func (api *PrivateAdminAPI) PeerEvents(ctx context.Context) (*rpc.Subscription, error) {
	// Make sure the server is running, fail otherwise
	server := api.node.Server()
	if server == nil {
		return nil, ErrNodeStopped
	}

	// Create the subscription
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}
	rpcSub := notifier.CreateSubscription()

	go func() {
		events := make(chan *p2p.PeerEvent)
		sub := server.SubscribeEvents(events)
		defer sub.Unsubscribe()

		for {
			select {
			case event := <-events:
				notifier.Notify(rpcSub.ID, event)
			case <-sub.Err():
				return
			case <-rpcSub.Err():
				return
			case <-notifier.Closed():
				return
			}
		}
	}()

	return rpcSub, nil
}

// StartRPC starts the HTTP RPC API server.
func (api *PrivateAdminAPI) StartRPC(host *string, port *int, cors *string, apis *string, vhosts *string) (bool, error) {
	api.node.lock.Lock()
	defer api.node.lock.Unlock()

	if api.node.httpHandler != nil {
		return false, fmt.Errorf("HTTP RPC already running on %s", api.node.httpEndpoint)
	}

	if host == nil {
		h := DefaultHTTPHost
		if api.node.config.HTTPHost != "" {
			h = api.node.config.HTTPHost
		}
		host = &h
	}
	if port == nil {
		port = &api.node.config.HTTPPort
	}

	allowedOrigins := api.node.config.HTTPCors
	if cors != nil {
		allowedOrigins = nil
		for _, origin := range strings.Split(*cors, ",") {
			allowedOrigins = append(allowedOrigins, strings.TrimSpace(origin))
		}
	}

	allowedVHosts := api.node.config.HTTPVirtualHosts
	if vhosts != nil {
		allowedVHosts = nil
		for _, vhost := range strings.Split(*host, ",") {
			allowedVHosts = append(allowedVHosts, strings.TrimSpace(vhost))
		}
	}

	modules := api.node.httpWhitelist
	if apis != nil {
		modules = nil
		for _, m := range strings.Split(*apis, ",") {
			modules = append(modules, strings.TrimSpace(m))
		}
	}

	if err := api.node.startHTTP(fmt.Sprintf("%s:%d", *host, *port), api.node.rpcAPIs, modules, allowedOrigins, allowedVHosts, api.node.config.HTTPTimeouts, api.node.config.WSOrigins); err != nil {
		return false, err
	}
	return true, nil
}

// StopRPC terminates an already running HTTP RPC API endpoint.
func (api *PrivateAdminAPI) StopRPC() (bool, error) {
	api.node.lock.Lock()
	defer api.node.lock.Unlock()

	if api.node.httpHandler == nil {
		return false, fmt.Errorf("HTTP RPC not running")
	}
	api.node.stopHTTP()
	return true, nil
}

// StartWS starts the websocket RPC API server.
func (api *PrivateAdminAPI) StartWS(host *string, port *int, allowedOrigins *string, apis *string) (bool, error) {
	api.node.lock.Lock()
	defer api.node.lock.Unlock()

	if api.node.wsHandler != nil {
		return false, fmt.Errorf("WebSocket RPC already running on %s", api.node.wsEndpoint)
	}

	if host == nil {
		h := DefaultWSHost
		if api.node.config.WSHost != "" {
			h = api.node.config.WSHost
		}
		host = &h
	}
	if port == nil {
		port = &api.node.config.WSPort
	}

	origins := api.node.config.WSOrigins
	if allowedOrigins != nil {
		origins = nil
		for _, origin := range strings.Split(*allowedOrigins, ",") {
			origins = append(origins, strings.TrimSpace(origin))
		}
	}

	modules := api.node.config.WSModules
	if apis != nil {
		modules = nil
		for _, m := range strings.Split(*apis, ",") {
			modules = append(modules, strings.TrimSpace(m))
		}
	}

	if err := api.node.startWS(fmt.Sprintf("%s:%d", *host, *port), api.node.rpcAPIs, modules, origins, api.node.config.WSExposeAll); err != nil {
		return false, err
	}
	return true, nil
}

// StopWS terminates an already running websocket RPC API endpoint.
func (api *PrivateAdminAPI) StopWS() (bool, error) {
	api.node.lock.Lock()
	defer api.node.lock.Unlock()

	if api.node.wsHandler == nil {
		return false, fmt.Errorf("WebSocket RPC not running")
	}
	api.node.stopWS()
	return true, nil
}

// PublicAdminAPI is the collection of administrative API methods exposed over
// both secure and unsecure RPC channels.
type PublicAdminAPI struct {
	node *Node // Node interfaced by this API
}

// NewPublicAdminAPI creates a new API definition for the public admin methods
// of the node itself.
func NewPublicAdminAPI(node *Node) *PublicAdminAPI {
	return &PublicAdminAPI{node: node}
}

// Peers retrieves all the information we know about each individual peer at the
// protocol granularity.
func (api *PublicAdminAPI) Peers() ([]*p2p.PeerInfo, error) {
	server := api.node.Server()
	if server == nil {
		return nil, ErrNodeStopped
	}
	return server.PeersInfo(), nil
}

// NodeInfo retrieves all the information we know about the host node at the
// protocol granularity.
func (api *PublicAdminAPI) NodeInfo() (*p2p.NodeInfo, error) {
	server := api.node.Server()
	if server == nil {
		return nil, ErrNodeStopped
	}
	return server.NodeInfo(), nil
}

// Datadir retrieves the current data directory the node is using.
func (api *PublicAdminAPI) Datadir() string {
	return api.node.DataDir()
}

// PublicWeb3API offers helper utils
type PublicWeb3API struct {
	stack *Node
}

// NewPublicWeb3API creates a new Web3Service instance
func NewPublicWeb3API(stack *Node) *PublicWeb3API {
	return &PublicWeb3API{stack}
}

// ClientVersion returns the node name
func (s *PublicWeb3API) ClientVersion() string {
	return s.stack.Server().Name
}

// Sha3 applies the ethereum sha3 implementation on the input.
// It assumes the input is hex encoded.
func (s *PublicWeb3API) Sha3(input hexutil.Bytes) hexutil.Bytes {
	return crypto.Keccak256(input)
}
