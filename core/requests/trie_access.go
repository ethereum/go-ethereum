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
package requests

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/core/access"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/trie"
	"golang.org/x/net/context"
)

// TrieAccess implements trie.OdrAccess, providing database/network access for
// a trie identified by root hash
type TrieAccess struct {
	trie.OdrAccess
	ca     *access.ChainAccess
	root   common.Hash
	trieDb trie.Database
}

// NewTrieAccess creates a new TrieAccess
func NewTrieAccess(ca *access.ChainAccess, root common.Hash, trieDb trie.Database) *TrieAccess {
	return &TrieAccess{
		ca:     ca,
		root:   root,
		trieDb: trieDb,
	}
}

// RetrieveKey retrieves a single key, returns true and stores nodes in local
// database if successful
func (self *TrieAccess) RetrieveKey(ctx context.Context, key []byte) bool {
	//fmt.Println("request trie %v key %v", self.root, key)
	r := NewTrieEntryAccess(self.root, self.trieDb, key)
	return self.ca.Retrieve(ctx, r) == nil
}

// OdrEnabled returns true if this TrieAccess is capable of doing network requests
func (self *TrieAccess) OdrEnabled() bool {
	return self.ca.OdrEnabled()
}

// TrieEntryAccess is the ODR request type for state/storage trie entries
type TrieEntryAccess struct {
	access.ObjectAccess
	root       common.Hash
	trieDb     trie.Database
	key, value []byte
	proof      trie.MerkleProof
	skipLevels int // set by DbGet() if unsuccessful
}

// NewTrieEntryAccess creates a new TrieEntryAccess request
func NewTrieEntryAccess(root common.Hash, trieDb trie.Database, key []byte) *TrieEntryAccess {
	return &TrieEntryAccess{root: root, trieDb: trieDb, key: key}
}

// Request sends an ODR request to the LES network (implementation of access.ObjectAccess)
func (self *TrieEntryAccess) Request(peer *access.Peer) error {
	glog.V(access.LogLevel).Infof("ODR: requesting trie root %08x key %08x from peer %v", self.root[:4], self.key[:4], peer.Id())
	req := &access.ProofReq{
		Root: self.root,
		Key:  self.key,
	}
	return peer.GetProofs([]*access.ProofReq{req})
}

// Valid processes an ODR request reply message from the LES network
// returns true and stores results in memory if the message was a valid reply
// to the request (implementation of access.ObjectAccess)
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

// DbGet tries to retrieve requested data from the local database, returns
// true and stores results in memory if successful
// (implementation of access.ObjectAccess)
func (self *TrieEntryAccess) DbGet() bool {
	return false // not used
}

// DbPut stores the results of a successful request in the local database
// (implementation of access.ObjectAccess)
func (self *TrieEntryAccess) DbPut() {
	trie.StoreProof(self.trieDb, self.proof)
}

// NodeDataBlockAccess is the ODR request type for node data (used for retrieving contract code)
type NodeDataAccess struct {
	access.ObjectAccess
	db   ethdb.Database
	hash common.Hash
	data []byte
}

// NewNodeDataAccess creates a new NodeDataAccess request
func NewNodeDataAccess(db ethdb.Database, hash common.Hash) *NodeDataAccess {
	return &NodeDataAccess{db: db, hash: hash}
}

// Request sends an ODR request to the LES network (implementation of access.ObjectAccess)
func (self *NodeDataAccess) Request(peer *access.Peer) error {
	glog.V(access.LogLevel).Infof("ODR: requesting node data for hash %08x from peer %v", self.hash[:4], peer.Id())
	return peer.GetNodeData([]common.Hash{self.hash})
}

// Valid processes an ODR request reply message from the LES network
// returns true and stores results in memory if the message was a valid reply
// to the request (implementation of access.ObjectAccess)
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

// DbGet tries to retrieve requested data from the local database, returns
// true and stores results in memory if successful
// (implementation of access.ObjectAccess)
func (self *NodeDataAccess) DbGet() bool {
	data, _ := self.db.Get(self.hash[:])
	if len(data) == 0 {
		return false
	}
	self.data = data
	return true
}

// DbPut stores the results of a successful request in the local database
// (implementation of access.ObjectAccess)
func (self *NodeDataAccess) DbPut() {
	self.db.Put(self.hash[:], self.data)
}

var sha3_nil = sha3.NewKeccak256().Sum(nil)

func RetrieveNodeData(ctx context.Context, ca *access.ChainAccess, hash common.Hash) []byte {
	//fmt.Println("request node data %v", hash)
	if bytes.Compare(hash[:], sha3_nil) == 0 {
		return nil
	}
	r := NewNodeDataAccess(ca.Db(), hash)
	ca.Retrieve(ctx, r)
	return r.data
}
