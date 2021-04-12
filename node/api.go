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
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"
)

// apis returns the collection of built-in RPC APIs.
func (n *Node) apis() []rpc.API {
	return []rpc.API{
		{
			Namespace: "admin",
			Version:   "1.0",
			Service:   &privateAdminAPI{n},
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   &publicAdminAPI{n},
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   debug.Handler,
		}, {
			Namespace: "web3",
			Version:   "1.0",
			Service:   &publicWeb3API{n},
			Public:    true,
		},
	}
}

// privateAdminAPI is the collection of administrative API methods exposed only
// over a secure RPC channel.
type privateAdminAPI struct {
	node *Node // Node interfaced by this API
}

// AddPeer requests connecting to a remote node, and also maintaining the new
// connection at all times, even reconnecting if it is lost.
func (api *privateAdminAPI) AddPeer(url string) (bool, error) {
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
func (api *privateAdminAPI) RemovePeer(url string) (bool, error) {
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
func (api *privateAdminAPI) AddTrustedPeer(url string) (bool, error) {
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
func (api *privateAdminAPI) RemoveTrustedPeer(url string) (bool, error) {
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
func (api *privateAdminAPI) PeerEvents(ctx context.Context) (*rpc.Subscription, error) {
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

// StartHTTP starts the HTTP RPC API server.
func (api *privateAdminAPI) StartHTTP(host *string, port *int, cors *string, apis *string, vhosts *string) (bool, error) {
	api.node.lock.Lock()
	defer api.node.lock.Unlock()

	// Determine host and port.
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

	// Determine config.
	config := httpConfig{
		CorsAllowedOrigins: api.node.config.HTTPCors,
		Vhosts:             api.node.config.HTTPVirtualHosts,
		Modules:            api.node.config.HTTPModules,
	}
	if cors != nil {
		config.CorsAllowedOrigins = nil
		for _, origin := range strings.Split(*cors, ",") {
			config.CorsAllowedOrigins = append(config.CorsAllowedOrigins, strings.TrimSpace(origin))
		}
	}
	if vhosts != nil {
		config.Vhosts = nil
		for _, vhost := range strings.Split(*host, ",") {
			config.Vhosts = append(config.Vhosts, strings.TrimSpace(vhost))
		}
	}
	if apis != nil {
		config.Modules = nil
		for _, m := range strings.Split(*apis, ",") {
			config.Modules = append(config.Modules, strings.TrimSpace(m))
		}
	}

	if err := api.node.http.setListenAddr(*host, *port); err != nil {
		return false, err
	}
	if err := api.node.http.enableRPC(api.node.rpcAPIs, config); err != nil {
		return false, err
	}
	if err := api.node.http.start(); err != nil {
		return false, err
	}
	return true, nil
}

// StartRPC starts the HTTP RPC API server.
// This method is deprecated. Use StartHTTP instead.
func (api *privateAdminAPI) StartRPC(host *string, port *int, cors *string, apis *string, vhosts *string) (bool, error) {
	log.Warn("Deprecation warning", "method", "admin.StartRPC", "use-instead", "admin.StartHTTP")
	return api.StartHTTP(host, port, cors, apis, vhosts)
}

// StopHTTP shuts down the HTTP server.
func (api *privateAdminAPI) StopHTTP() (bool, error) {
	api.node.http.stop()
	return true, nil
}

// StopRPC shuts down the HTTP server.
// This method is deprecated. Use StopHTTP instead.
func (api *privateAdminAPI) StopRPC() (bool, error) {
	log.Warn("Deprecation warning", "method", "admin.StopRPC", "use-instead", "admin.StopHTTP")
	return api.StopHTTP()
}

// StartWS starts the websocket RPC API server.
func (api *privateAdminAPI) StartWS(host *string, port *int, allowedOrigins *string, apis *string) (bool, error) {
	api.node.lock.Lock()
	defer api.node.lock.Unlock()

	// Determine host and port.
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

	// Determine config.
	config := wsConfig{
		Modules: api.node.config.WSModules,
		Origins: api.node.config.WSOrigins,
		// ExposeAll: api.node.config.WSExposeAll,
	}
	if apis != nil {
		config.Modules = nil
		for _, m := range strings.Split(*apis, ",") {
			config.Modules = append(config.Modules, strings.TrimSpace(m))
		}
	}
	if allowedOrigins != nil {
		config.Origins = nil
		for _, origin := range strings.Split(*allowedOrigins, ",") {
			config.Origins = append(config.Origins, strings.TrimSpace(origin))
		}
	}

	// Enable WebSocket on the server.
	server := api.node.wsServerForPort(*port)
	if err := server.setListenAddr(*host, *port); err != nil {
		return false, err
	}
	if err := server.enableWS(api.node.rpcAPIs, config); err != nil {
		return false, err
	}
	if err := server.start(); err != nil {
		return false, err
	}
	api.node.http.log.Info("WebSocket endpoint opened", "url", api.node.WSEndpoint())
	return true, nil
}

// StopWS terminates all WebSocket servers.
func (api *privateAdminAPI) StopWS() (bool, error) {
	api.node.http.stopWS()
	api.node.ws.stop()
	return true, nil
}

// publicAdminAPI is the collection of administrative API methods exposed over
// both secure and unsecure RPC channels.
type publicAdminAPI struct {
	node *Node // Node interfaced by this API
}

// Peers retrieves all the information we know about each individual peer at the
// protocol granularity.
func (api *publicAdminAPI) Peers() ([]*p2p.PeerInfo, error) {
	server := api.node.Server()
	if server == nil {
		return nil, ErrNodeStopped
	}
	return server.PeersInfo(), nil
}

// NodeInfo retrieves all the information we know about the host node at the
// protocol granularity.
func (api *publicAdminAPI) NodeInfo() (*p2p.NodeInfo, error) {
	server := api.node.Server()
	if server == nil {
		return nil, ErrNodeStopped
	}
	return server.NodeInfo(), nil
}

// Datadir retrieves the current data directory the node is using.
func (api *publicAdminAPI) Datadir() string {
	return api.node.DataDir()
}

// publicWeb3API offers helper utils
type publicWeb3API struct {
	stack *Node
}

// ClientVersion returns the node name
func (s *publicWeb3API) ClientVersion() string {
	return s.stack.Server().Name
}

// Sha3 applies the ethereum sha3 implementation on the input.
// It assumes the input is hex encoded.
func (s *publicWeb3API) Sha3(input hexutil.Bytes) hexutil.Bytes {
	return crypto.Keccak256(input)
}
