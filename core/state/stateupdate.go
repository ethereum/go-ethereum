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
	"fmt"
	"maps"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
)

// contractCode represents a contract code with associated metadata.
type contractCode struct {
	hash   common.Hash // hash is the cryptographic hash of the contract code.
	blob   []byte      // blob is the binary representation of the contract code.
	exists bool        // flag whether the code has been existent
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
	address    common.Address         // address is the unique account identifier
	data       []byte                 // data is the slim-RLP encoded account data.
	origin     []byte                 // origin is the original value of account data in slim-RLP encoding.
	code       *contractCode          // code represents mutated contract code; nil means it's not modified.
	codeOrigin *contractCode          // codeOrigin is the original value of contract code; nil means it does not exist before the update.
	storages   map[common.Hash][]byte // storages stores mutated slots in prefix-zero-trimmed RLP format.

	// storagesOriginByKey and storagesOriginByHash both store the original values
	// of mutated slots in prefix-zero-trimmed RLP format. The difference is that
	// storagesOriginByKey uses the **raw** storage slot key as the map ID, while
	// storagesOriginByHash uses the **hash** of the storage slot key instead.
	storagesOriginByKey  map[common.Hash][]byte
	storagesOriginByHash map[common.Hash][]byte
}

// stateUpdate represents the difference between two states resulting from state
// execution. It contains information about mutated contract codes, accounts,
// and storage slots, along with their original values.
type stateUpdate struct {
	originRoot  common.Hash // hash of the state before applying mutation
	root        common.Hash // hash of the state after applying mutation
	blockNumber uint64      // Associated block number

	accounts       map[common.Hash][]byte    // accounts stores mutated accounts in 'slim RLP' encoding
	accountsOrigin map[common.Address][]byte // accountsOrigin stores the original values of mutated accounts in 'slim RLP' encoding

	// storages stores mutated slots in 'prefix-zero-trimmed' RLP format.
	// The value is keyed by account hash and **storage slot key hash**.
	storages map[common.Hash]map[common.Hash][]byte

	// storagesOrigin stores the original values of mutated slots in
	// 'prefix-zero-trimmed' RLP format.
	// (a) the value is keyed by account hash and **storage slot key** if rawStorageKey is true;
	// (b) the value is keyed by account hash and **storage slot key hash** if rawStorageKey is false;
	storagesOrigin map[common.Address]map[common.Hash][]byte
	rawStorageKey  bool

	codes       map[common.Address]*contractCode // codes contains the set of dirty codes
	codesOrigin map[common.Address]*contractCode // codesOrigin contains the original code data of the mutated accounts
	nodes       *trienode.MergedNodeSet          // Aggregated dirty nodes caused by state changes
}

// empty returns a flag indicating the state transition is empty or not.
func (sc *stateUpdate) empty() bool {
	return sc.originRoot == sc.root
}

// newStateUpdate constructs a state update object by identifying the differences
// between two states through state execution. It combines the specified account
// deletions and account updates to create a complete state update.
//
// rawStorageKey is a flag indicating whether to use the raw storage slot key or
// the hash of the slot key for constructing state update object.
func newStateUpdate(rawStorageKey bool, originRoot common.Hash, root common.Hash, blockNumber uint64, deletes map[common.Hash]*accountDelete, updates map[common.Hash]*accountUpdate, nodes *trienode.MergedNodeSet) *stateUpdate {
	var (
		accounts       = make(map[common.Hash][]byte)
		accountsOrigin = make(map[common.Address][]byte)
		storages       = make(map[common.Hash]map[common.Hash][]byte)
		storagesOrigin = make(map[common.Address]map[common.Hash][]byte)
		codes          = make(map[common.Address]*contractCode)
		codesOrigin    = make(map[common.Address]*contractCode)
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
			codes[addr] = op.code
		}
		if op.codeOrigin != nil {
			codesOrigin[addr] = op.codeOrigin
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
	return &stateUpdate{
		originRoot:     originRoot,
		root:           root,
		blockNumber:    blockNumber,
		accounts:       accounts,
		accountsOrigin: accountsOrigin,
		storages:       storages,
		storagesOrigin: storagesOrigin,
		rawStorageKey:  rawStorageKey,
		codes:          codes,
		codesOrigin:    codesOrigin,
		nodes:          nodes,
	}
}

// stateSet converts the current stateUpdate object into a triedb.StateSet
// object. This function extracts the necessary data from the stateUpdate
// struct and formats it into the StateSet structure consumed by the triedb
// package.
func (sc *stateUpdate) stateSet() *triedb.StateSet {
	return &triedb.StateSet{
		Accounts:       sc.accounts,
		AccountsOrigin: sc.accountsOrigin,
		Storages:       sc.storages,
		StoragesOrigin: sc.storagesOrigin,
		RawStorageKey:  sc.rawStorageKey,
	}
}

// markCodeExistence determines whether each piece of contract code referenced
// in this state update actually exists.
//
// Note: This operation is expensive and not needed during normal state transitions.
// It is only required when SizeTracker is enabled to produce accurate state
// statistics.
func (sc *stateUpdate) markCodeExistence(reader ContractCodeReader) {
	cache := make(map[common.Hash]bool)
	for addr, code := range sc.codes {
		if exists, ok := cache[code.hash]; ok {
			code.exists = exists
			continue
		}
		res := reader.Has(addr, code.hash)
		cache[code.hash] = res
		code.exists = res
	}
}

// ToTracingUpdate converts the internal stateUpdate to an exported tracing.StateUpdate.
func (sc *stateUpdate) ToTracingUpdate() (*tracing.StateUpdate, error) {
	update := &tracing.StateUpdate{
		OriginRoot:     sc.originRoot,
		Root:           sc.root,
		BlockNumber:    sc.blockNumber,
		AccountChanges: make(map[common.Address]*tracing.AccountChange, len(sc.accountsOrigin)),
		StorageChanges: make(map[common.Address]map[common.Hash]*tracing.StorageChange),
		CodeChanges:    make(map[common.Address]*tracing.CodeChange, len(sc.codes)),
		TrieChanges:    make(map[common.Hash]map[string]*tracing.TrieNodeChange),
	}

	for addr, prev := range sc.accountsOrigin {
		addrHash := crypto.Keccak256Hash(addr.Bytes())
		new, exists := sc.accounts[addrHash]
		if !exists {
			return nil, fmt.Errorf("account %x not found", addr)
		}

		change := &tracing.AccountChange{}

		if len(prev) > 0 {
			if acct, err := types.FullAccount(prev); err == nil {
				change.Prev = &types.StateAccount{
					Nonce:    acct.Nonce,
					Balance:  acct.Balance,
					Root:     acct.Root,
					CodeHash: acct.CodeHash,
				}
			}
		}

		if len(new) > 0 {
			if acct, err := types.FullAccount(new); err == nil {
				change.New = &types.StateAccount{
					Nonce:    acct.Nonce,
					Balance:  acct.Balance,
					Root:     acct.Root,
					CodeHash: acct.CodeHash,
				}
			}
		}

		update.AccountChanges[addr] = change
	}

	for addr, slots := range sc.storagesOrigin {
		addrHash := crypto.Keccak256Hash(addr.Bytes())
		subset, exists := sc.storages[addrHash]
		if !exists {
			return nil, fmt.Errorf("storage %x not found", addr)
		}

		storageMap := make(map[common.Hash]*tracing.StorageChange, len(slots))
		for key, encPrev := range slots {
			// Get new value - handle both raw and hashed key formats
			var (
				exists  bool
				encNew  []byte
				decPrev []byte
				decNew  []byte
				err     error
			)
			if sc.rawStorageKey {
				encNew, exists = subset[crypto.Keccak256Hash(key.Bytes())]
			} else {
				encNew, exists = subset[key]
			}
			if !exists {
				return nil, fmt.Errorf("storage slot %x-%x not found", addr, key)
			}

			// Decode the prev and new values
			if len(encPrev) > 0 {
				_, decPrev, _, err = rlp.Split(encPrev)
				if err != nil {
					return nil, fmt.Errorf("failed to decode prevValue: %v", err)
				}
			}
			if len(encNew) > 0 {
				_, decNew, _, err = rlp.Split(encNew)
				if err != nil {
					return nil, fmt.Errorf("failed to decode newValue: %v", err)
				}
			}

			storageMap[key] = &tracing.StorageChange{
				Prev: common.BytesToHash(decPrev),
				New:  common.BytesToHash(decNew),
			}
		}
		update.StorageChanges[addr] = storageMap
	}

	for addr, code := range sc.codes {
		change := &tracing.CodeChange{
			New: &tracing.ContractCode{
				Hash:   code.hash,
				Code:   code.blob,
				Exists: code.exists,
			},
		}
		if origin, ok := sc.codesOrigin[addr]; ok {
			change.Prev = &tracing.ContractCode{
				Hash:   origin.hash,
				Code:   origin.blob,
				Exists: origin.exists,
			}
		}
		update.CodeChanges[addr] = change
	}

	if sc.nodes != nil {
		for owner, subset := range sc.nodes.Sets {
			nodeMap := make(map[string]*tracing.TrieNodeChange, len(subset.Origins))
			for path, oldNode := range subset.Origins {
				newNode, exists := subset.Nodes[path]
				if !exists {
					return nil, fmt.Errorf("node %x-%v not found", owner, path)
				}
				nodeMap[path] = &tracing.TrieNodeChange{
					Prev: &trienode.Node{
						Hash: crypto.Keccak256Hash(oldNode),
						Blob: oldNode,
					},
					New: &trienode.Node{
						Hash: newNode.Hash,
						Blob: newNode.Blob,
					},
				}
			}
			update.TrieChanges[owner] = nodeMap
		}
	}

	return update, nil
}
