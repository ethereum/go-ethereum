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
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
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
	peerMap map[discover.NodeID]int
	peers   []*Peer
	Run     ProtoCall
}

func NewSimNode(id *NodeId, n Network) *SimNode {
	return &SimNode{
		Id:      id,
		network: n,
		peerMap: make(map[discover.NodeID]int),
	}
}

func (self *SimNode) LocalAddr() []byte {
	return self.Id.Bytes()
}

func (self *SimNode) ParseAddr(p []byte, s string) ([]byte, error) {
	return p, nil
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

func (self *SimNode) Disconnect(rid []byte) error {
	self.lock.Lock()
	defer self.lock.Unlock()
	id := NewNodeId(rid)
	peer := self.getPeer(id)
	if peer == nil || peer.MsgPipeRW == nil {
		return fmt.Errorf("already disconnected")
	}
	peer.MsgPipeRW.Close()
	peer.MsgPipeRW = nil
	// na := self.network.GetNodeAdapter(id)
	// peer = na.(*SimNode).GetPeer(self.Id)
	// peer.RW = nil
	log.Trace(fmt.Sprintf("dropped peer %v", id))

	return nil
}

func (self *SimNode) Connect(rid []byte) error {
	self.lock.Lock()
	defer self.lock.Unlock()
	id := NewNodeId(rid)
	na := self.network.GetNodeAdapter(id)
	if na == nil {
		return fmt.Errorf("node adapter for %v is missing", id)
	}
	rw, rrw := p2p.MsgPipe()
	// // run protocol on remote node with self as peer
	peer := self.getPeer(id)
	if peer != nil && peer.MsgPipeRW != nil {
		return fmt.Errorf("already connected %v to peer %v", self.Id, id)
	}
	peer = self.setPeer(id, rrw)
	close(peer.Connc)
	defer close(peer.Readyc)
	err := na.(ProtocolRunner).RunProtocol(self.Id, rrw, rw, peer)
	if err != nil {
		return fmt.Errorf("cannot run protocol (%v -> %v) %v", self.Id, id, err)
	}

	// run protocol on remote node with self as peer
	err = self.RunProtocol(id, rw, rrw, peer)
	if err != nil {
		return fmt.Errorf("cannot run protocol (%v -> %v): %v", id, self.Id, err)
	}
	return nil
}

func (self *SimNode) RunProtocol(id *NodeId, rw, rrw p2p.MsgReadWriter, peer *Peer) error {
	if self.Run == nil {
		log.Trace(fmt.Sprintf("no protocol starting on peer %v (connection with %v)", self.Id, id))
		return nil
	}
	log.Trace(fmt.Sprintf("protocol starting on peer %v (connection with %v)", self.Id, id))
	p := p2p.NewPeer(id.NodeID, Name(id.Bytes()), []p2p.Cap{})
	go func() {
		self.network.DidConnect(self.Id, id)
		err := self.Run(p, rw)
		<-peer.Readyc
		self.Disconnect(id.Bytes())
		peer.Errc <- err
		log.Trace(fmt.Sprintf("protocol quit on peer %v (connection with %v broken: %v)", self.Id, id, err))
		self.network.DidDisconnect(self.Id, id)
	}()
	return nil
}
