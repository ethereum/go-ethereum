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

package state

import (
	"bytes"

	"github.com/ethereum/go-ethereum/access"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
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
	r := &TrieRequest{ctx: ctx, root: self.root, key: key}
	return self.ca.Retrieve(r) == nil
}

// OdrEnabled returns true if this TrieAccess is capable of doing network requests
func (self *TrieAccess) OdrEnabled() bool {
	return self.ca.OdrEnabled()
}

// TrieRequest is the ODR request type for state/storage trie entries
type TrieRequest struct {
	access.Request
	ctx   context.Context
	root  common.Hash
	key   []byte
	proof trie.MerkleProof
}

func (req *TrieRequest) Ctx() context.Context { return req.ctx }

func (req *TrieRequest) StoreResult(db access.Database) {
	trie.StoreProof(db, req.proof)
}

// NodeDataRequest is the ODR request type for node data (used for retrieving contract code)
type NodeDataRequest struct {
	access.Request
	ctx  context.Context
	hash common.Hash
	data []byte
}

func (req *NodeDataRequest) Ctx() context.Context { return req.ctx }

func (req *NodeDataRequest) GetData() []byte {
	return req.data
}

func (req *NodeDataRequest) StoreResult(db access.Database) {
	db.Put(req.hash[:], req.GetData())
}

var sha3_nil = sha3.NewKeccak256().Sum(nil)

func RetrieveNodeData(ctx context.Context, ca *access.ChainAccess, hash common.Hash) []byte {
	if bytes.Compare(hash[:], sha3_nil) == 0 {
		return nil
	}
	res, _ := ca.Db().Get(hash[:])
	if res != nil || !access.IsOdrContext(ctx) {
		return res
	}
	r := &NodeDataRequest{ctx: ctx, hash: hash}
	ca.Retrieve(r)
	return r.GetData()
}
