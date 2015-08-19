// Copyright 2014 The go-ethereum Authors
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

// Package state provides a caching layer atop the Ethereum state trie.
package state

import (
	"bytes"
	//"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/access"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/trie"
)

type TrieAccess struct {
	ca     *access.ChainAccess
	root   common.Hash
	trieDb trie.Database
}

func NewTrieAccess(ca *access.ChainAccess, root common.Hash, trieDb trie.Database) *TrieAccess {
	return &TrieAccess{
		ca:     ca,
		root:   root,
		trieDb: trieDb,
	}
}

func (self *TrieAccess) RetrieveKey(key []byte, ctx *access.OdrContext) bool {
	//fmt.Println("request trie %v key %v", self.root, key)
	r := &TrieEntryAccess{root: self.root, trieDb: self.trieDb, key: key}
	return self.ca.Retrieve(r, ctx) == nil
}

func (self *TrieAccess) OdrEnabled() bool {
	return self.ca.OdrEnabled()
}

type TrieEntryAccess struct {
	root       common.Hash
	trieDb     trie.Database
	key, value []byte
	proof      trie.MerkleProof
	skipLevels int // set by DbGet() if unsuccessful
}

func (self *TrieEntryAccess) Request(peer *access.Peer) error {
	glog.V(access.LogLevel).Infof("ODR: requesting trie root %08x key %08x from peer %v", self.root[:4], self.key[:4], peer.Id())
	req := &access.ProofReq{
		Root: self.root,
		Key:  self.key,
	}
	return peer.GetProofs([]*access.ProofReq{req})
}

func (self *TrieEntryAccess) Valid(msg *access.Msg) bool {
	glog.V(access.LogLevel).Infof("ODR: validating trie root %08x key %08x", self.root[:4], self.key[:4])

	if msg.MsgType != access.MsgProofs {
		glog.V(access.LogLevel).Infof("ODR: invalid message type")
		return false
	}
	proofs := msg.Obj.([]trie.MerkleProof)
	if len(proofs) != 1 {
		glog.V(access.LogLevel).Infof("ODR: invalid number of entries: %d", len(proofs))
		return false
	}
	value, err := trie.VerifyProof(self.root, self.key, proofs[0])
	if err != nil {
		glog.V(access.LogLevel).Infof("ODR: merkle proof verification error: %v", err)
		return false
	}
	self.proof = proofs[0]
	self.value = value
	glog.V(access.LogLevel).Infof("ODR: validation successful")
	return true
}

func (self *TrieEntryAccess) DbGet() bool {
	return false // not used
}

func (self *TrieEntryAccess) DbPut() {
	trie.StoreProof(self.trieDb, self.proof)
}

type NodeDataAccess struct {
	db   ethdb.Database
	hash common.Hash
	data []byte
}

func (self *NodeDataAccess) Request(peer *access.Peer) error {
	glog.V(access.LogLevel).Infof("ODR: requesting node data for hash %08x from peer %v", self.hash[:4], peer.Id())
	return peer.GetNodeData([]common.Hash{self.hash})
}

func (self *NodeDataAccess) Valid(msg *access.Msg) bool {
	glog.V(access.LogLevel).Infof("ODR: validating node data for hash %08x", self.hash[:4])
	if msg.MsgType != access.MsgNodeData {
		glog.V(access.LogLevel).Infof("ODR: invalid message type")
		return false
	}
	reply := msg.Obj.([][]byte)
	if len(reply) != 1 {
		glog.V(access.LogLevel).Infof("ODR: invalid number of entries: %d", len(reply))
		return false
	}
	data := reply[0]
	hash := crypto.Sha3Hash(data)
	if bytes.Compare(self.hash[:], hash[:]) != 0 {
		glog.V(access.LogLevel).Infof("ODR: requested hash %08x does not match received data hash %08x", self.hash[:4], hash[:4])
		return false
	}
	self.data = data
	glog.V(access.LogLevel).Infof("ODR: validation successful")
	return true
}

func (self *NodeDataAccess) DbGet() bool {
	data, _ := self.db.Get(self.hash[:])
	if len(data) == 0 {
		return false
	}
	self.data = data
	return true
}

func (self *NodeDataAccess) DbPut() {
	self.db.Put(self.hash[:], self.data)
}

var sha3_nil = sha3.NewKeccak256().Sum(nil)

func RetrieveNodeData(ca *access.ChainAccess, hash common.Hash, ctx *access.OdrContext) []byte {
	//fmt.Println("request node data %v", hash)
	if bytes.Compare(hash[:], sha3_nil) == 0 {
		return nil
	}
	r := &NodeDataAccess{db: ca.Db(), hash: hash}
	ca.Retrieve(r, ctx)
	return r.data
}
