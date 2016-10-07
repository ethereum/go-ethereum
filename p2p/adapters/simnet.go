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

	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

type NetAdapter interface {
	Connect([]byte) error
	Disconnect(*p2p.Peer, p2p.MsgReadWriter)
	LocalAddr() []byte
	ParseAddr([]byte, string) ([]byte, error)
}

func newPeer(rw p2p.MsgReadWriter) *Peer {
	return &Peer{
		RW:     rw,
		Errc:   make(chan error, 1),
		Flushc: make(chan bool),
		Onc:    make(chan bool),
	}
}

type Peer struct {
	RW     p2p.MsgReadWriter
	Errc   chan error
	Flushc chan bool
	Onc    chan bool
}

// Network interface to retrieve protocol runner to launch upon peer
// connection
type Network interface {
	Protocol(id *discover.NodeID) ProtoCall
}

type Messenger interface {
	SendMsg(p2p.MsgWriter, uint64, interface{}) error
	ReadMsg(p2p.MsgReader) (p2p.Msg, error)
	NewPipe() (p2p.MsgReadWriter, p2p.MsgReadWriter)
	ClosePipe(rw p2p.MsgReadWriter)
}

type ProtoCall func(*p2p.Peer, p2p.MsgReadWriter) error

func NewSimNet(id *discover.NodeID, n Network, m Messenger) *SimNet {
	return &SimNet{
		ID:        id,
		Network:   n,
		Messenger: m,
		PeerMap:   make(map[discover.NodeID]int),
	}
}

// Simnet is the network adapter that
type SimNet struct {
	ID *discover.NodeID
	Network
	Messenger
	Run     ProtoCall
	PeerMap map[discover.NodeID]int
	Peers   []*Peer
	lock    sync.RWMutex
}

func Key(id []byte) string {
	return string(id)
}

func Name(id []byte) string {
	return fmt.Sprintf("test-%08x", id)
}

func (self *SimNet) LocalAddr() []byte {
	return self.ID[:]
}

func (self *SimNet) ParseAddr(p []byte, s string) ([]byte, error) {
	return p, nil
}

func (self *SimNet) GetPeer(id *discover.NodeID) *Peer {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.getPeer(id)
}

func (self *SimNet) getPeer(id *discover.NodeID) *Peer {
	i, found := self.PeerMap[*id]
	if !found {
		return nil
	}
	return self.Peers[i]
}

func (self *SimNet) SetPeer(id *discover.NodeID, rw p2p.MsgReadWriter) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.setPeer(id, rw)
}

func (self *SimNet) setPeer(id *discover.NodeID, rw p2p.MsgReadWriter) {
	i, found := self.PeerMap[*id]
	if !found {
		i = len(self.Peers)
		self.PeerMap[*id] = i
		self.Peers = append(self.Peers, newPeer(rw))
		return
	}
	if self.Peers[i] != nil && rw != nil {
		panic(fmt.Sprintf("pipe for %v already set", id))
	}
	// legit reconnect reset disconnection error,
	self.Peers[i].RW = rw
}

func (self *SimNet) Disconnect(p *p2p.Peer, rw p2p.MsgReadWriter) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.ClosePipe(rw)
	id := p.ID()
	self.getPeer(&id).RW = nil
	glog.V(6).Infof("dropped peer %v", id)
}

func (self *SimNet) Connect(rid []byte) error {
	self.lock.Lock()
	defer self.lock.Unlock()
	var id discover.NodeID
	copy(id[:], rid)
	peer := self.getPeer(&id)
	if peer != nil {
		return fmt.Errorf("already connected")
	}
	run := self.Protocol(&id)
	rw, rrw := self.NewPipe()
	glog.V(6).Infof("connect to peer %v, setting pipe", id)
	self.setPeer(&id, rrw)
	if run != nil {
		p := p2p.NewPeer(*self.ID, Name(self.ID[:]), []p2p.Cap{})
		go run(p, rrw)
	}
	peer = self.getPeer(&id)
	go func() {
		glog.V(6).Infof("simnet connect to %v", id)
		p := p2p.NewPeer(id, Name(id[:]), []p2p.Cap{})
		err := self.Run(p, rw)
		peer.Errc <- err
	}()
	return nil
}
