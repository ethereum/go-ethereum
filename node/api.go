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
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"
)

// apis returns the collection of built-in RPC APIs.
func (n *Node) apis() []rpc.API {
	return []rpc.API{
		{
			Namespace: "admin",
			Service:   &adminAPI{n},
		}, {
			Namespace: "debug",
			Service:   debug.Handler,
		}, {
			Namespace: "debug",
			Service:   &p2pDebugAPI{n},
		}, {
			Namespace: "web3",
			Service:   &web3API{n},
		},
	}
}

// adminAPI is the collection of administrative API methods exposed over
// both secure and unsecure RPC channels.
type adminAPI struct {
	node *Node // Node interfaced by this API
}

// This function sets the param maxPeers for the node. If there are excess peers attached to the node, it will remove the difference.
func (api *adminAPI) SetMaxPeers(maxPeers int) (bool, error) {
	// Make sure the server is running, fail otherwise
	server := api.node.Server()
	if server == nil {
		return false, ErrNodeStopped
	}

	server.SetMaxPeers(maxPeers)

	return true, nil
}

// This function gets the maxPeers param for the node.
func (api *adminAPI) GetMaxPeers() (int, error) {
	// Make sure the server is running, fail otherwise
	server := api.node.Server()
	if server == nil {
		return 0, ErrNodeStopped
	}

	return server.MaxPeers, nil
}

// AddPeer requests connecting to a remote node, and also maintaining the new
// connection at all times, even reconnecting if it is lost.
func (api *adminAPI) AddPeer(url string) (bool, error) {
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
func (api *adminAPI) RemovePeer(url string) (bool, error) {
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
func (api *adminAPI) AddTrustedPeer(url string) (bool, error) {
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
func (api *adminAPI) RemoveTrustedPeer(url string) (bool, error) {
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
func (api *adminAPI) PeerEvents(ctx context.Context) (*rpc.Subscription, error) {
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
func (api *adminAPI) StartHTTP(host *string, port *int, cors *string, apis *string, vhosts *string) (bool, error) {
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
		rpcEndpointConfig: rpcEndpointConfig{
			batchItemLimit:         api.node.config.BatchRequestLimit,
			batchResponseSizeLimit: api.node.config.BatchResponseMaxSize,
		},
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
// Deprecated: use StartHTTP instead.
func (api *adminAPI) StartRPC(host *string, port *int, cors *string, apis *string, vhosts *string) (bool, error) {
	log.Warn("Deprecation warning", "method", "admin.StartRPC", "use-instead", "admin.StartHTTP")
	return api.StartHTTP(host, port, cors, apis, vhosts)
}

// StopHTTP shuts down the HTTP server.
func (api *adminAPI) StopHTTP() (bool, error) {
	api.node.http.stop()
	return true, nil
}

// StopRPC shuts down the HTTP server.
// Deprecated: use StopHTTP instead.
func (api *adminAPI) StopRPC() (bool, error) {
	log.Warn("Deprecation warning", "method", "admin.StopRPC", "use-instead", "admin.StopHTTP")
	return api.StopHTTP()
}

// StartWS starts the websocket RPC API server.
func (api *adminAPI) StartWS(host *string, port *int, allowedOrigins *string, apis *string) (bool, error) {
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
		rpcEndpointConfig: rpcEndpointConfig{
			batchItemLimit:         api.node.config.BatchRequestLimit,
			batchResponseSizeLimit: api.node.config.BatchResponseMaxSize,
		},
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
	server := api.node.wsServerForPort(*port, false)
	if err := server.setListenAddr(*host, *port); err != nil {
		return false, err
	}

	openApis, _ := api.node.getAPIs()
	if err := server.enableWS(openApis, config); err != nil {
		return false, err
	}

	if err := server.start(); err != nil {
		return false, err
	}

	api.node.http.log.Info("WebSocket endpoint opened", "url", api.node.WSEndpoint())

	return true, nil
}

// StopWS terminates all WebSocket servers.
func (api *adminAPI) StopWS() (bool, error) {
	api.node.http.stopWS()
	api.node.ws.stop()

	return true, nil
}

// Peers retrieves all the information we know about each individual peer at the
// protocol granularity.
func (api *adminAPI) Peers() ([]*p2p.PeerInfo, error) {
	server := api.node.Server()
	if server == nil {
		return nil, ErrNodeStopped
	}

	return server.PeersInfo(), nil
}

// NodeInfo retrieves all the information we know about the host node at the
// protocol granularity.
func (api *adminAPI) NodeInfo() (*p2p.NodeInfo, error) {
	server := api.node.Server()
	if server == nil {
		return nil, ErrNodeStopped
	}

	return server.NodeInfo(), nil
}

// Datadir retrieves the current data directory the node is using.
func (api *adminAPI) Datadir() string {
	return api.node.DataDir()
}

// web3API offers helper utils
type web3API struct {
	stack *Node
}

// ClientVersion returns the node name
func (s *web3API) ClientVersion() string {
	return s.stack.Server().Name
}

// Sha3 applies the ethereum sha3 implementation on the input.
// It assumes the input is hex encoded.
func (s *web3API) Sha3(input hexutil.Bytes) hexutil.Bytes {
	return crypto.Keccak256(input)
}

type ExecutionPoolSize struct {
	HttpLimit int
	WSLimit   int
}

type ExecutionPoolRequestTimeout struct {
	HttpLimit time.Duration
	WSLimit   time.Duration
}

func (api *adminAPI) GetExecutionPoolSize() *ExecutionPoolSize {
	var httpLimit int
	if api.node.http.host != "" {
		httpLimit = api.node.http.httpHandler.Load().(*rpcHandler).server.GetExecutionPoolSize()
	}

	var wsLimit int
	if api.node.ws.host != "" {
		wsLimit = api.node.ws.wsHandler.Load().(*rpcHandler).server.GetExecutionPoolSize()
	}

	executionPoolSize := &ExecutionPoolSize{
		HttpLimit: httpLimit,
		WSLimit:   wsLimit,
	}

	return executionPoolSize
}

func (api *adminAPI) GetExecutionPoolRequestTimeout() *ExecutionPoolRequestTimeout {
	var httpLimit time.Duration
	if api.node.http.host != "" {
		httpLimit = api.node.http.httpHandler.Load().(*rpcHandler).server.GetExecutionPoolRequestTimeout()
	}

	var wsLimit time.Duration
	if api.node.ws.host != "" {
		wsLimit = api.node.ws.wsHandler.Load().(*rpcHandler).server.GetExecutionPoolRequestTimeout()
	}

	executionPoolRequestTimeout := &ExecutionPoolRequestTimeout{
		HttpLimit: httpLimit,
		WSLimit:   wsLimit,
	}

	return executionPoolRequestTimeout
}

// func (api *privateAdminAPI) SetWSExecutionPoolRequestTimeout(n int) *ExecutionPoolRequestTimeout {
// 	if api.node.ws.host != "" {
// 		api.node.ws.wsConfig.executionPoolRequestTimeout = time.Duration(n) * time.Millisecond
// 		api.node.ws.wsHandler.Load().(*rpcHandler).server.SetExecutionPoolRequestTimeout(time.Duration(n) * time.Millisecond)
// 		log.Warn("updating ws execution pool request timeout", "timeout", n)
// 	}

// 	return api.GetExecutionPoolRequestTimeout()
// }

// func (api *privateAdminAPI) SetHttpExecutionPoolRequestTimeout(n int) *ExecutionPoolRequestTimeout {
// 	if api.node.http.host != "" {
// 		api.node.http.httpConfig.executionPoolRequestTimeout = time.Duration(n) * time.Millisecond
// 		api.node.http.httpHandler.Load().(*rpcHandler).server.SetExecutionPoolRequestTimeout(time.Duration(n) * time.Millisecond)
// 		log.Warn("updating http execution pool request timeout", "timeout", n)
// 	}

// 	return api.GetExecutionPoolRequestTimeout()
// }

func (api *adminAPI) SetWSExecutionPoolSize(n int) *ExecutionPoolSize {
	if api.node.ws.host != "" {
		api.node.ws.wsConfig.executionPoolSize = uint64(n)
		api.node.ws.wsHandler.Load().(*rpcHandler).server.SetExecutionPoolSize(n)
		log.Warn("updating ws execution pool size", "threads", n)
	}

	return api.GetExecutionPoolSize()
}

func (api *adminAPI) SetHttpExecutionPoolSize(n int) *ExecutionPoolSize {
	if api.node.http.host != "" {
		api.node.http.httpConfig.executionPoolSize = uint64(n)
		api.node.http.httpHandler.Load().(*rpcHandler).server.SetExecutionPoolSize(n)
		log.Warn("updating http execution pool size", "threads", n)
	}

	return api.GetExecutionPoolSize()
}

// p2pDebugAPI provides access to p2p internals for debugging.
type p2pDebugAPI struct {
	stack *Node
}

func (s *p2pDebugAPI) DiscoveryV4Table() [][]discover.BucketNode {
	disc := s.stack.server.DiscoveryV4()
	if disc != nil {
		return disc.TableBuckets()
	}
	return nil
}
