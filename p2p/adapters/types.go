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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

const lablen = 4

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

type ProtoCall func(*p2p.Peer, p2p.MsgReadWriter) error

func (self *NodeId) Bytes() []byte {
	return self.NodeID[:]
}

func (self *NodeId) MarshalJSON() (out []byte, err error) {
	return []byte(`"` + self.String() + `"`), nil
}

func (self *NodeId) UnmarshalJSON(value []byte) error {
	s := string(value)
	h, err := discover.HexID(s)
	if err != nil {
		return err
	}
	*self = NodeId{h}
	return nil
}

func (self *NodeId) Label() string {
	return self.String()[:lablen]
}

type Messenger interface {
	SendMsg(uint64, interface{}) error
	ReadMsg() (p2p.Msg, error)
	Close()
}

type NodeAdapter interface {
	Connect([]byte) error
	Disconnect([]byte) error
	// Disconnect(*p2p.Peer, p2p.MsgReadWriter)
	LocalAddr() []byte
	ParseAddr([]byte, string) ([]byte, error)
	// Messenger() Messenger  <<<... old version
	Messenger(p2p.MsgReadWriter) Messenger
}

type ProtocolRunner interface {
	RunProtocol(id *NodeId, rw, rrw p2p.MsgReadWriter, p *Peer) error
}

type StartAdapter interface {
	Start() error
	Stop() error
}

type Reporter interface {
	DidConnect(*NodeId, *NodeId) error
	DidDisconnect(*NodeId, *NodeId) error
}

func RandomNodeId() *NodeId {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("unable to generate key")
	}
	var id discover.NodeID
	pubkey := crypto.FromECDSAPub(&key.PublicKey)
	copy(id[:], pubkey[1:])
	return &NodeId{id}
}

func RandomNodeIds(n int) []*NodeId {
	var ids []*NodeId
	for i := 0; i < n; i++ {
		ids = append(ids, RandomNodeId())
	}
	return ids
}
