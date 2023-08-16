// Copyright 2023 The go-ethereum Authors
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

package trie

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// Trie defines the interface that various tree implementations must adhere to.
// These implementations are used by the stateDB to access both account data and
// contract storage data within the Ethereum state. The primary responsibility
// of a Trie is to serve as the underlying structure for managing Ethereum's state.
type Trie interface {
	// GetKey returns the sha3 preimage of a hashed key that was previously used
	// to store a value.
	//
	// TODO(fjl): remove this when StateTrie is removed
	GetKey([]byte) []byte

	// GetAccount abstracts an account read from the trie. It retrieves the
	// account blob from the trie with provided account address and decodes it
	// with associated decoding algorithm. If the specified account is not in
	// the trie, nil will be returned. If the trie is corrupted(e.g. some nodes
	// are missing or the account blob is incorrect for decoding), an error will
	// be returned.
	GetAccount(address common.Address) (*types.StateAccount, error)

	// GetStorage returns the value for key stored in the trie. The value bytes
	// must not be modified by the caller. If a node was not found in the database,
	// a trie.MissingNodeError is returned.
	GetStorage(addr common.Address, key []byte) ([]byte, error)

	// UpdateAccount abstracts an account write to the trie. It encodes the
	// provided account object with associated algorithm and then updates it
	// in the trie with provided address.
	UpdateAccount(address common.Address, account *types.StateAccount) error

	// UpdateStorage associates key with value in the trie. If value has length zero,
	// any existing value is deleted from the trie. The value bytes must not be modified
	// by the caller while they are stored in the trie. If a node was not found in the
	// database, a trie.MissingNodeError is returned.
	UpdateStorage(addr common.Address, key, value []byte) error

	// DeleteAccount abstracts an account deletion from the trie.
	DeleteAccount(address common.Address) error

	// DeleteStorage removes any existing value for key from the trie. If a node
	// was not found in the database, a trie.MissingNodeError is returned.
	DeleteStorage(addr common.Address, key []byte) error

	// UpdateContractCode abstracts code write to the trie. It is expected
	// to be moved to the stateWriter interface when the latter is ready.
	UpdateContractCode(address common.Address, codeHash common.Hash, code []byte) error

	// Hash returns the root hash of the trie. It does not write to the database and
	// can be used even if the trie doesn't have one.
	Hash() common.Hash

	// Commit gathers changed nodes in the trie, replacing them with corresponding
	// node hashes. It creates a set of these changed nodes, including modified
	// leaf if the "collectLeaf" parameter is set to true. This set is returned as
	// the "nodeset," which can be empty (nil) if no changes exist.
	//
	// Once committed, the trie becomes unusable and requires a new trie with a
	// fresh root and updated database for further use.
	Commit(collectLeaf bool) (common.Hash, *trienode.NodeSet, error)

	// NodeIterator returns an iterator that returns nodes of the trie. Iteration
	// starts at the key after the given start key. And error will be returned
	// if fails to create node iterator.
	NodeIterator(startKey []byte) (NodeIterator, error)

	// Prove constructs a Merkle proof for key. The result contains all encoded nodes
	// on the path to the value at key. The value itself is also included in the last
	// node and can be retrieved by verifying the proof.
	//
	// If the trie does not contain a value for key, the returned proof contains all
	// nodes of the longest existing prefix of the key (at least the root), ending
	// with the node that proves the absence of the key.
	Prove(key []byte, proofDb ethdb.KeyValueWriter) error
}
