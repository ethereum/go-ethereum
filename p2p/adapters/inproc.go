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
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rpc"
)

func newPeer(rw *p2p.MsgPipeRW) *Peer {
	return &Peer{
		MsgPipeRW: rw,
		Errc:      make(chan error, 1),
		Connc:     make(chan bool),
		Readyc:    make(chan bool),
	}
}

type Peer struct {
	*p2p.MsgPipeRW
	Connc  chan bool
	Readyc chan bool
	Errc   chan error
}

// Network interface to retrieve protocol runner to launch upon peer
// connection
type Network interface {
	GetNodeAdapter(id *NodeId) NodeAdapter
	Reporter
}

// SimNode is the network adapter that
type SimNode struct {
	lock    sync.RWMutex
	Id      *NodeId
	network Network
	service node.Service
	peerMap map[discover.NodeID]int
	peers   []*Peer
	client  *rpc.Client
}

func NewSimNode(id *NodeId, svc node.Service, n Network) *SimNode {
	// for simplicity, only support single protocol services
	if len(svc.Protocols()) != 1 {
		panic("service must have a single protocol")
	}

	return &SimNode{
		Id:      id,
		network: n,
		service: svc,
		peerMap: make(map[discover.NodeID]int),
	}
}

// Addr returns the node's address
func (self *SimNode) Addr() []byte {
	return []byte(self.Node().String())
}

func (self *SimNode) Node() *discover.Node {
	return discover.NewNode(self.Id.NodeID, nil, 0, 0)
}

func (self *SimNode) Client() (*rpc.Client, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.client == nil {
		return nil, errors.New("RPC not started")
	}
	return self.client, nil
}

// Start starts the RPC handler and the underlying service
func (self *SimNode) Start() error {
	if err := self.startRPC(); err != nil {
		return err
	}
	return self.service.Start(self)
}

// Stop stops the RPC handler and the underlying service
func (self *SimNode) Stop() error {
	self.stopRPC()
	return self.service.Stop()
}

func (self *SimNode) startRPC() error {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.client != nil {
		return errors.New("RPC already started")
	}

	// add SimAdminAPI so that the network can call the AddPeer
	// and RemovePeer RPC methods
	apis := append(self.service.APIs(), rpc.API{
		Namespace: "admin",
		Version:   "1.0",
		Service:   &SimAdminAPI{self},
	})

	// start the RPC handler
	handler := rpc.NewServer()
	for _, api := range apis {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return fmt.Errorf("error registering RPC: %s", err)
		}
	}

	// create an in-process RPC client
	self.client = rpc.DialInProc(handler)

	return nil
}

func (self *SimNode) stopRPC() {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.client != nil {
		self.client.Close()
		self.client = nil
	}
}

func (self *SimNode) GetPeer(id *NodeId) *Peer {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.getPeer(id)
}

func (self *SimNode) getPeer(id *NodeId) *Peer {
	i, found := self.peerMap[id.NodeID]
	if !found {
		return nil
	}
	return self.peers[i]
}

func (self *SimNode) setPeer(id *NodeId, rw *p2p.MsgPipeRW) *Peer {
	i, found := self.peerMap[id.NodeID]
	if !found {
		i = len(self.peers)
		self.peerMap[id.NodeID] = i
		p := newPeer(rw)
		self.peers = append(self.peers, p)
		return p
	}
	// if self.peers[i] != nil && m != nil {
	// 	panic(fmt.Sprintf("pipe for %v already set", id))
	// }
	// legit reconnect reset disconnection error,
	p := self.peers[i]
	p.MsgPipeRW = rw
	p.Connc = make(chan bool)
	p.Readyc = make(chan bool)
	return p
}

func (self *SimNode) RemovePeer(node *discover.Node) {
	self.lock.Lock()
	defer self.lock.Unlock()
	id := &NodeId{node.ID}
	peer := self.getPeer(id)
	if peer == nil || peer.MsgPipeRW == nil {
		return
	}
	peer.MsgPipeRW.Close()
	peer.MsgPipeRW = nil
	// na := self.network.GetNodeAdapter(id)
	// peer = na.(*SimNode).GetPeer(self.Id)
	// peer.RW = nil
	log.Trace(fmt.Sprintf("dropped peer %v", id))
}

func (self *SimNode) AddPeer(node *discover.Node) {
	self.lock.Lock()
	defer self.lock.Unlock()
	id := &NodeId{node.ID}
	na := self.network.GetNodeAdapter(id)
	if na == nil {
		panic(fmt.Sprintf("node adapter for %v is missing", id))
	}
	rw, rrw := p2p.MsgPipe()
	// // run protocol on remote node with self as peer
	peer := self.getPeer(id)
	if peer != nil && peer.MsgPipeRW != nil {
		return
	}
	peer = self.setPeer(id, rrw)
	close(peer.Connc)
	defer close(peer.Readyc)
	na.(*SimNode).RunProtocol(self, rrw, rw, peer)

	// run protocol on remote node with self as peer
	self.RunProtocol(na.(*SimNode), rw, rrw, peer)
}

func (self *SimNode) PeerCount() int {
	self.lock.Lock()
	defer self.lock.Unlock()
	return len(self.peers)
}

func (self *SimNode) NodeInfo() *p2p.NodeInfo {
	return &p2p.NodeInfo{ID: self.Id.String()}
}

func (self *SimNode) PeersInfo() (info []*p2p.PeerInfo) {
	return nil
}

func (self *SimNode) RunProtocol(node *SimNode, rw, rrw p2p.MsgReadWriter, peer *Peer) {
	id := node.Id
	protocol := self.service.Protocols()[0]
	if protocol.Run == nil {
		log.Trace(fmt.Sprintf("no protocol starting on peer %v (connection with %v)", self.Id, id))
		return
	}
	log.Trace(fmt.Sprintf("protocol starting on peer %v (connection with %v)", self.Id, id))
	p := p2p.NewPeer(id.NodeID, id.Label(), []p2p.Cap{})
	go func() {
		self.network.DidConnect(self.Id, id)
		err := protocol.Run(p, rw)
		<-peer.Readyc
		self.RemovePeer(node.Node())
		peer.Errc <- err
		log.Trace(fmt.Sprintf("protocol quit on peer %v (connection with %v broken: %v)", self.Id, id, err))
		self.network.DidDisconnect(self.Id, id)
	}()
}

// SimAdminAPI implements the AddPeer and RemovePeer RPC methods (API
// compatible with node.PrivateAdminAPI)
type SimAdminAPI struct {
	*SimNode
}

func (api *SimAdminAPI) AddPeer(url string) (bool, error) {
	node, err := discover.ParseNode(url)
	if err != nil {
		return false, fmt.Errorf("invalid enode: %v", err)
	}
	api.SimNode.AddPeer(node)
	return true, nil
}

func (api *SimAdminAPI) RemovePeer(url string) (bool, error) {
	node, err := discover.ParseNode(url)
	if err != nil {
		return false, fmt.Errorf("invalid enode: %v", err)
	}
	api.SimNode.RemovePeer(node)
	return true, nil
}
