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
	"maps"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
)

// contractCode represents a contract code with associated metadata.
type contractCode struct {
	hash common.Hash // hash is the cryptographic hash of the contract code.
	blob []byte      // blob is the binary representation of the contract code.
}

// CodeLen returns the length of the contract code blob.
func (c *contractCode) CodeLen() int {
	return len(c.blob)
}

// accountDelete represents an operation for deleting an Ethereum account.
type accountDelete struct {
	address common.Address // address is the unique account identifier
	origin  []byte         // origin is the original value of account data in slim-RLP encoding.

	// storages stores mutated slots, the value should be nil.
	storages map[common.Hash][]byte

	// storagesOrigin stores the original values of mutated slots in
	// prefix-zero-trimmed RLP format. The map key refers to the **HASH**
	// of the raw storage slot key.
	storagesOrigin map[common.Hash][]byte
}

// accountUpdate represents an operation for updating an Ethereum account.
type accountUpdate struct {
	address  common.Address         // address is the unique account identifier
	data     []byte                 // data is the slim-RLP encoded account data.
	origin   []byte                 // origin is the original value of account data in slim-RLP encoding.
	code     *contractCode          // code represents mutated contract code; nil means it's not modified.
	storages map[common.Hash][]byte // storages stores mutated slots in prefix-zero-trimmed RLP format.

	// storagesOriginByKey and storagesOriginByHash both store the original values
	// of mutated slots in prefix-zero-trimmed RLP format. The difference is that
	// storagesOriginByKey uses the **raw** storage slot key as the map ID, while
	// storagesOriginByHash uses the **hash** of the storage slot key instead.
	storagesOriginByKey  map[common.Hash][]byte
	storagesOriginByHash map[common.Hash][]byte
}

// StateUpdate represents the difference between two states resulting from state
// execution. It contains information about mutated contract codes, accounts,
// and storage slots, along with their original values.
type StateUpdate struct {
	OriginRoot     common.Hash               // hash of the state before applying mutation
	Root           common.Hash               // hash of the state after applying mutation
	Accounts       map[common.Hash][]byte    // accounts stores mutated accounts in 'slim RLP' encoding
	AccountsOrigin map[common.Address][]byte // accountsOrigin stores the original values of mutated accounts in 'slim RLP' encoding

	// Storages stores mutated slots in 'prefix-zero-trimmed' RLP format.
	// The value is keyed by account hash and **storage slot key hash**.
	Storages map[common.Hash]map[common.Hash][]byte

	// StoragesOrigin stores the original values of mutated slots in
	// 'prefix-zero-trimmed' RLP format.
	// (a) the value is keyed by account hash and **storage slot key** if rawStorageKey is true;
	// (b) the value is keyed by account hash and **storage slot key hash** if rawStorageKey is false;
	StoragesOrigin map[common.Address]map[common.Hash][]byte
	RawStorageKey  bool

	Codes map[common.Address]contractCode // codes contains the set of dirty codes
	Nodes *trienode.MergedNodeSet         // Aggregated dirty nodes caused by state changes
}

// empty returns a flag indicating the state transition is empty or not.
func (sc *StateUpdate) empty() bool {
	return sc.OriginRoot == sc.Root
}

// newStateUpdate constructs a state update object by identifying the differences
// between two states through state execution. It combines the specified account
// deletions and account updates to create a complete state update.
//
// rawStorageKey is a flag indicating whether to use the raw storage slot key or
// the hash of the slot key for constructing state update object.
func newStateUpdate(rawStorageKey bool, originRoot common.Hash, root common.Hash, deletes map[common.Hash]*accountDelete, updates map[common.Hash]*accountUpdate, nodes *trienode.MergedNodeSet) *StateUpdate {
	var (
		accounts       = make(map[common.Hash][]byte)
		accountsOrigin = make(map[common.Address][]byte)
		storages       = make(map[common.Hash]map[common.Hash][]byte)
		storagesOrigin = make(map[common.Address]map[common.Hash][]byte)
		codes          = make(map[common.Address]contractCode)
	)
	// Since some accounts might be destroyed and recreated within the same
	// block, deletions must be aggregated first.
	for addrHash, op := range deletes {
		addr := op.address
		accounts[addrHash] = nil
		accountsOrigin[addr] = op.origin

		// If storage wiping exists, the hash of the storage slot key must be used
		if len(op.storages) > 0 {
			storages[addrHash] = op.storages
		}
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
		accounts[addrHash] = op.data

		// Aggregate the account original value. If the account is already
		// present in the aggregated accountsOrigin set, skip it.
		if _, found := accountsOrigin[addr]; !found {
			accountsOrigin[addr] = op.origin
		}
		// Aggregate the storage mutation list. If a slot in op.storages is
		// already present in aggregated storages set, the value will be
		// overwritten.
		if len(op.storages) > 0 {
			if _, exist := storages[addrHash]; !exist {
				storages[addrHash] = op.storages
			} else {
				maps.Copy(storages[addrHash], op.storages)
			}
		}
		// Aggregate the storage original values. If the slot is already present
		// in aggregated storagesOrigin set, skip it.
		storageOriginSet := op.storagesOriginByHash
		if rawStorageKey {
			storageOriginSet = op.storagesOriginByKey
		}
		if len(storageOriginSet) > 0 {
			origin, exist := storagesOrigin[addr]
			if !exist {
				storagesOrigin[addr] = storageOriginSet
			} else {
				for key, slot := range storageOriginSet {
					if _, found := origin[key]; !found {
						origin[key] = slot
					}
				}
			}
		}
	}
	return &StateUpdate{
		OriginRoot:     originRoot,
		Root:           root,
		Accounts:       accounts,
		AccountsOrigin: accountsOrigin,
		Storages:       storages,
		StoragesOrigin: storagesOrigin,
		RawStorageKey:  rawStorageKey,
		Codes:          codes,
		Nodes:          nodes,
	}
}

// stateSet converts the current stateUpdate object into a triedb.StateSet
// object. This function extracts the necessary data from the stateUpdate
// struct and formats it into the StateSet structure consumed by the triedb
// package.
func (sc *StateUpdate) stateSet() *triedb.StateSet {
	return &triedb.StateSet{
		Accounts:       sc.Accounts,
		AccountsOrigin: sc.AccountsOrigin,
		Storages:       sc.Storages,
		StoragesOrigin: sc.StoragesOrigin,
		RawStorageKey:  sc.RawStorageKey,
	}
}

// StateChangeset represents a state mutations that occurred during the execution of a block.
type StateChangeset struct {
	Accounts     int // Total number of accounts present in the state at this block
	Storages     int // Total number of storage entries across all accounts in the state at this block
	Trienodes    int // Total number of trie nodes present in the state at this block
	Codes        int // Total number of contract codes present in the state at this block, with 32 bytes hash as the identifier
	AccountSize  int // Combined size of all accounts in the state, with 20 bytes address as the identifier
	StorageSize  int // Combined size of all storage entries, with 32 bytes key as the identifier
	TrienodeSize int // Combined size of all trie nodes, with varying size node path as the identifier (up to 64 bytes)
	CodeSize     int // Combined size of all contract codes in the state, with 20 bytes address as the identifier
}

// IntoChangeset converts the current StateUpdate into a StateChangeset.
func (sc *StateUpdate) IntoChangeset() *StateChangeset {
	var (
		accountSize, storageSize, nodeSize, codeSize int
		accounts, storages, nodes, codes             int
	)

	for addr, oldValue := range sc.AccountsOrigin {
		addrHash := crypto.Keccak256Hash(addr.Bytes())
		newValue, exists := sc.Accounts[addrHash]
		if !exists {
			log.Warn("State update missing account", "address", addr)
			continue
		}
		if len(newValue) == 0 {
			accounts -= 1
			accountSize -= common.HashLength
		}
		if len(oldValue) == 0 {
			accounts += 1
			accountSize += common.HashLength
		}
		accountSize += len(newValue) - len(oldValue)
	}
	for addr, slots := range sc.StoragesOrigin {
		addrHash := crypto.Keccak256Hash(addr.Bytes())
		subset, exists := sc.Storages[addrHash]
		if !exists {
			log.Warn("State update missing storage", "address", addr)
			continue
		}
		for key, oldValue := range slots {
			var (
				exists   bool
				newValue []byte
			)
			if sc.RawStorageKey {
				newValue, exists = subset[crypto.Keccak256Hash(key.Bytes())]
			} else {
				newValue, exists = subset[key]
			}
			if !exists {
				log.Warn("State update missing storage slot", "address", addr, "key", key)
				continue
			}
			if len(newValue) == 0 {
				storages -= 1
				storageSize -= common.HashLength
			}
			if len(oldValue) == 0 {
				storages += 1
				storageSize += common.HashLength
			}
			storageSize += len(newValue) - len(oldValue)
		}
	}
	for _, subset := range sc.Nodes.Sets {
		for path, n := range subset.Nodes {
			if len(n.Blob) == 0 {
				nodes -= 1
				nodeSize -= len(path) + common.HashLength
			}
			if n.OriginLen() == 0 {
				nodes += 1
				nodeSize += len(path) + common.HashLength
			}
			nodeSize += len(n.Blob) - n.OriginLen()
		}
	}
	for _, code := range sc.Codes {
		codes += 1
		codeSize += code.CodeLen() + common.HashLength // no deduplication
	}

	return &StateChangeset{
		Accounts:     accounts,
		AccountSize:  accountSize,
		Storages:     storages,
		StorageSize:  storageSize,
		Trienodes:    nodes,
		TrienodeSize: nodeSize,
		Codes:        codes,
		CodeSize:     codeSize,
	}
}
