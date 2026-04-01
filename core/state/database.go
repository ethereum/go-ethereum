// Copyright 2017 The go-ethereum Authors
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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/trie/transitiontrie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
)

// DatabaseType represents the type of trie backing the state database.
type DatabaseType int

const (
	// TypeMPT indicates a Merkle Patricia Trie (MPT) backed database.
	TypeMPT DatabaseType = iota

	// TypeUBT indicates a Unified Binary Trie (UBT) backed database.
	TypeUBT
)

// Is returns the flag indicating the database type equals to the given one.
func (typ DatabaseType) Is(t DatabaseType) bool {
	return typ == t
}

// Database wraps access to tries and contract code.
type Database interface {
	// Type returns the trie type backing this database (MPT or UBT).
	Type() DatabaseType

	// Reader returns a state reader associated with the specified state root.
	Reader(root common.Hash) (Reader, error)

	// Iteratee returns a state iteratee associated with the specified state root,
	// through which the account iterator and storage iterator can be created.
	Iteratee(root common.Hash) (Iteratee, error)

	// OpenTrie opens the main account trie.
	OpenTrie(root common.Hash) (Trie, error)

	// OpenStorageTrie opens the storage trie of an account.
	OpenStorageTrie(stateRoot common.Hash, address common.Address, root common.Hash, trie Trie) (Trie, error)

	// TrieDB returns the underlying trie database for managing trie nodes.
	TrieDB() *triedb.Database

	// Commit flushes all pending writes and finalizes the state transition,
	// committing the changes to the underlying storage. It returns an error
	// if the commit fails.
	Commit(update *StateUpdate) error
}

// Trie is a Ethereum Merkle Patricia trie.
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

	// PrefetchAccount attempts to resolve specific accounts from the database
	// to accelerate subsequent trie operations.
	PrefetchAccount([]common.Address) error

	// GetStorage returns the value for key stored in the trie. The value bytes
	// must not be modified by the caller. If a node was not found in the database,
	// a trie.MissingNodeError is returned.
	GetStorage(addr common.Address, key []byte) ([]byte, error)

	// PrefetchStorage attempts to resolve specific storage slots from the database
	// to accelerate subsequent trie operations.
	PrefetchStorage(addr common.Address, keys [][]byte) error

	// UpdateAccount abstracts an account write to the trie. It encodes the
	// provided account object with associated algorithm and then updates it
	// in the trie with provided address.
	UpdateAccount(address common.Address, account *types.StateAccount, codeLen int) error

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

	// Commit collects all dirty nodes in the trie and replace them with the
	// corresponding node hash. All collected nodes(including dirty leaves if
	// collectLeaf is true) will be encapsulated into a nodeset for return.
	// The returned nodeset can be nil if the trie is clean(nothing to commit).
	// Once the trie is committed, it's not usable anymore. A new trie must
	// be created with new root and updated trie database for following usage
	Commit(collectLeaf bool) (common.Hash, *trienode.NodeSet)

	// Witness returns a set containing all trie nodes that have been accessed.
	// The returned map could be nil if the witness is empty.
	Witness() map[string][]byte

	// NodeIterator returns an iterator that returns nodes of the trie. Iteration
	// starts at the key after the given start key. And error will be returned
	// if fails to create node iterator.
	NodeIterator(startKey []byte) (trie.NodeIterator, error)

	// Prove constructs a Merkle proof for key. The result contains all encoded nodes
	// on the path to the value at key. The value itself is also included in the last
	// node and can be retrieved by verifying the proof.
	//
	// If the trie does not contain a value for key, the returned proof contains all
	// nodes of the longest existing prefix of the key (at least the root), ending
	// with the node that proves the absence of the key.
	Prove(key []byte, proofDb ethdb.KeyValueWriter) error

	// IsUBT returns true if the trie is unified binary trie based.
	IsUBT() bool
}

// NewDatabase creates a state database with the provided data sources.
//
// Deprecated, please use NewMPTDatabase or NewUBTDatabase directly.
func NewDatabase(tdb *triedb.Database, codedb *CodeDB) Database {
	if tdb.IsUBT() {
		return NewUBTDatabase(tdb, codedb)
	}
	return NewMPTDatabase(tdb, codedb)
}

// NewDatabaseForTesting is similar to NewDatabase, but it initializes the caching
// db by using an ephemeral memory db with default config for testing.
func NewDatabaseForTesting() Database {
	db := rawdb.NewMemoryDatabase()
	return NewDatabase(triedb.NewDatabase(db, nil), NewCodeDB(db))
}

// mustCopyTrie returns a deep-copied trie.
func mustCopyTrie(t Trie) Trie {
	switch t := t.(type) {
	case *trie.StateTrie:
		return t.Copy()
	case *transitiontrie.TransitionTrie:
		return t.Copy()
	case *bintrie.BinaryTrie:
		return t.Copy()
	default:
		panic(fmt.Errorf("unknown trie type %T", t))
	}
}
