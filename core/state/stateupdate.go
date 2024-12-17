// Copyright 2024 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
)

// contractCode represents a contract code with associated metadata.
type contractCode struct {
	hash common.Hash // hash is the cryptographic hash of the contract code.
	blob []byte      // blob is the binary representation of the contract code.
}

// accountDelete represents an operation for deleting an Ethereum account.
type accountDelete struct {
	address        common.Address         // address is the unique account identifier
	origin         []byte                 // origin is the original value of account data in slim-RLP encoding.
	storagesOrigin map[common.Hash][]byte // storagesOrigin stores the original values of mutated slots in prefix-zero-trimmed RLP format.
}

// accountUpdate represents an operation for updating an Ethereum account.
type accountUpdate struct {
	address        common.Address         // address is the unique account identifier
	data           []byte                 // data is the slim-RLP encoded account data.
	origin         []byte                 // origin is the original value of account data in slim-RLP encoding.
	code           *contractCode          // code represents mutated contract code; nil means it's not modified.
	storages       map[common.Hash][]byte // storages stores mutated slots in prefix-zero-trimmed RLP format.
	storagesOrigin map[common.Hash][]byte // storagesOrigin stores the original values of mutated slots in prefix-zero-trimmed RLP format.
}

// stateUpdate represents the difference between two states resulting from state
// execution. It contains information about mutated contract codes, accounts,
// and storage slots, along with their original values.
type stateUpdate struct {
	originRoot     common.Hash                               // hash of the state before applying mutation
	root           common.Hash                               // hash of the state after applying mutation
	destructs      map[common.Hash]struct{}                  // destructs contains the list of destructed accounts
	accounts       map[common.Hash][]byte                    // accounts stores mutated accounts in 'slim RLP' encoding
	accountsOrigin map[common.Address][]byte                 // accountsOrigin stores the original values of mutated accounts in 'slim RLP' encoding
	storages       map[common.Hash]map[common.Hash][]byte    // storages stores mutated slots in 'prefix-zero-trimmed' RLP format
	storagesOrigin map[common.Address]map[common.Hash][]byte // storagesOrigin stores the original values of mutated slots in 'prefix-zero-trimmed' RLP format
	codes          map[common.Address]contractCode           // codes contains the set of dirty codes
	nodes          *trienode.MergedNodeSet                   // Aggregated dirty nodes caused by state changes
}

// empty returns a flag indicating the state transition is empty or not.
func (sc *stateUpdate) empty() bool {
	return sc.originRoot == sc.root
}

// newStateUpdate constructs a state update object, representing the differences
// between two states by performing state execution. It aggregates the given
// account deletions and account updates to form a comprehensive state update.
func newStateUpdate(originRoot common.Hash, root common.Hash, deletes map[common.Hash]*accountDelete, updates map[common.Hash]*accountUpdate, nodes *trienode.MergedNodeSet) *stateUpdate {
	var (
		destructs      = make(map[common.Hash]struct{})
		accounts       = make(map[common.Hash][]byte)
		accountsOrigin = make(map[common.Address][]byte)
		storages       = make(map[common.Hash]map[common.Hash][]byte)
		storagesOrigin = make(map[common.Address]map[common.Hash][]byte)
		codes          = make(map[common.Address]contractCode)
	)
	// Due to the fact that some accounts could be destructed and resurrected
	// within the same block, the deletions must be aggregated first.
	for addrHash, op := range deletes {
		addr := op.address
		destructs[addrHash] = struct{}{}
		accountsOrigin[addr] = op.origin
		if len(op.storagesOrigin) > 0 {
			storagesOrigin[addr] = op.storagesOrigin
		}
	}
	// Aggregate account updates then.
	for addrHash, op := range updates {
		// Aggregate dirty contract codes if they are available.
		addr := op.address
		if op.code != nil {
			codes[addr] = *op.code
		}
		// Aggregate the account changes. The original account value will only
		// be tracked if it's not present yet.
		accounts[addrHash] = op.data
		if _, found := accountsOrigin[addr]; !found {
			accountsOrigin[addr] = op.origin
		}
		// Aggregate the storage changes. The original storage slot value will
		// only be tracked if it's not present yet.
		if len(op.storages) > 0 {
			storages[addrHash] = op.storages
		}
		if len(op.storagesOrigin) > 0 {
			origin := storagesOrigin[addr]
			if origin == nil {
				storagesOrigin[addr] = op.storagesOrigin
				continue
			}
			for key, slot := range op.storagesOrigin {
				if _, found := origin[key]; !found {
					origin[key] = slot
				}
			}
			storagesOrigin[addr] = origin
		}
	}
	return &stateUpdate{
		originRoot:     types.TrieRootHash(originRoot),
		root:           types.TrieRootHash(root),
		destructs:      destructs,
		accounts:       accounts,
		accountsOrigin: accountsOrigin,
		storages:       storages,
		storagesOrigin: storagesOrigin,
		codes:          codes,
		nodes:          nodes,
	}
}

// stateSet converts the current stateUpdate object into a triedb.StateSet
// object. This function extracts the necessary data from the stateUpdate
// struct and formats it into the StateSet structure consumed by the triedb
// package.
func (sc *stateUpdate) stateSet() *triedb.StateSet {
	return &triedb.StateSet{
		Destructs:      sc.destructs,
		Accounts:       sc.accounts,
		AccountsOrigin: sc.accountsOrigin,
		Storages:       sc.storages,
		StoragesOrigin: sc.storagesOrigin,
	}
}
