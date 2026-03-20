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
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
)

// contractCode encapsulates contract bytecode and its associated metadata.
type contractCode struct {
	hash       common.Hash // hash is the cryptographic hash of the current contract code.
	blob       []byte      // blob is the raw byte representation of the current contract code.
	originHash common.Hash // originHash is the cryptographic hash of the code prior to mutation.

	// Derived fields, populated only when state tracking is enabled.
	duplicate  bool   // duplicate indicates whether the updated code already exists.
	originBlob []byte // originBlob is the original byte representation of the contract code.
}

// accountDelete represents a deletion operation for an Ethereum account.
type accountDelete struct {
	address common.Address // address uniquely identifies the account.
	origin  Account        // origin is the account state prior to deletion.

	storages       map[common.Hash]common.Hash // storages contains mutated storage slots.
	storagesOrigin map[common.Hash]common.Hash // storagesOrigin holds original values of mutated slots; keys are hashes of raw storage slot keys.
}

// accountUpdate represents an update operation for an Ethereum account.
type accountUpdate struct {
	address  common.Address              // address uniquely identifies the account.
	data     *Account                    // data is the updated account state; nil indicates deletion.
	origin   *Account                    // origin is the previous account state; nil indicates non-existence.
	code     *contractCode               // code contains updated contract code; nil if unchanged.
	storages map[common.Hash]common.Hash // storages contains updated storage slots.

	// storagesOriginByKey and storagesOriginByHash both record original values
	// of mutated storage slots:
	// - storagesOriginByKey uses raw storage slot keys.
	// - storagesOriginByHash uses hashed storage slot keys.
	storagesOriginByKey  map[common.Hash]common.Hash
	storagesOriginByHash map[common.Hash]common.Hash
}

// stateUpdate captures the difference between two states resulting from
// execution. It records all mutated accounts, contract codes, and storage
// slots, along with their original values.
type stateUpdate struct {
	originRoot  common.Hash // originRoot is the state root before applying changes.
	root        common.Hash // root is the state root after applying changes.
	blockNumber uint64      // blockNumber is the associated block height.

	accounts       map[common.Hash]*Account    // accounts contains mutated accounts, keyed by account hash.
	accountsOrigin map[common.Address]*Account // accountsOrigin holds original values of mutated accounts, keyed by address.

	// storages contains mutated storage slots, keyed by account hash and
	// storage slot key hash.
	storages map[common.Hash]map[common.Hash]common.Hash

	// storagesOrigin holds original values of mutated storage slots.
	// The key format depends on rawStorageKey:
	// - if true:  keyed by account address and raw storage slot key.
	// - if false: keyed by account address and storage slot key hash.
	storagesOrigin map[common.Address]map[common.Hash]common.Hash
	rawStorageKey  bool

	codes           map[common.Address]*contractCode // codes contains mutated contract codes, keyed by address.
	nodes           *trienode.MergedNodeSet          // nodes aggregates all dirty trie nodes produced by the update.
	secondaryHashes map[common.Address]SecondaryHash // hashes of secondary tries
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
func newStateUpdate(rawStorageKey bool, originRoot common.Hash, root common.Hash, blockNumber uint64, deletes map[common.Hash]*accountDelete, updates map[common.Hash]*accountUpdate, nodes *trienode.MergedNodeSet, secondaryHashes map[common.Address]SecondaryHash) *stateUpdate {
	var (
		accounts       = make(map[common.Hash]*Account)
		accountsOrigin = make(map[common.Address]*Account)
		storages       = make(map[common.Hash]map[common.Hash]common.Hash)
		storagesOrigin = make(map[common.Address]map[common.Hash]common.Hash)
		codes          = make(map[common.Address]*contractCode)
	)
	// Since some accounts might be destroyed and recreated within the same
	// block, deletions must be aggregated first.
	for addrHash, op := range deletes {
		addr := op.address
		accounts[addrHash] = nil
		accountsOrigin[addr] = &op.origin

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
		originRoot:      originRoot,
		root:            root,
		blockNumber:     blockNumber,
		accounts:        accounts,
		accountsOrigin:  accountsOrigin,
		storages:        storages,
		storagesOrigin:  storagesOrigin,
		rawStorageKey:   rawStorageKey,
		codes:           codes,
		nodes:           nodes,
		secondaryHashes: secondaryHashes,
	}
}

// stateSet converts the current stateUpdate object into a triedb.StateSet
// object. This function extracts the necessary data from the stateUpdate
// struct and formats it into the StateSet structure consumed by the triedb
// package.
func (sc *stateUpdate) stateSet() *triedb.StateSet {
	return nil
	//return &triedb.StateSet{
	//	Accounts:       sc.accounts,
	//	AccountsOrigin: sc.accountsOrigin,
	//	Storages:       sc.storages,
	//	StoragesOrigin: sc.storagesOrigin,
	//	RawStorageKey:  sc.rawStorageKey,
	//}
}

// deriveCodeFields derives the missing fields of contract code changes
// such as original code value.
//
// Note: This operation is expensive and not needed during normal state
// transitions. It is only required when SizeTracker or StateUpdate hook
// is enabled to produce accurate state statistics.
func (sc *stateUpdate) deriveCodeFields(reader ContractCodeReader) error {
	cache := make(map[common.Hash]bool)
	for addr, code := range sc.codes {
		if code.originHash != types.EmptyCodeHash {
			blob := reader.Code(addr, code.originHash)
			if len(blob) == 0 {
				return fmt.Errorf("original code of %x is empty", addr)
			}
			code.originBlob = blob
		}
		if exists, ok := cache[code.hash]; ok {
			code.duplicate = exists
			continue
		}
		res := reader.Has(addr, code.hash)
		cache[code.hash] = res
		code.duplicate = res
	}
	return nil
}

// ToTracingUpdate converts the internal stateUpdate to an exported tracing.StateUpdate.
func (sc *stateUpdate) ToTracingUpdate() (*tracing.StateUpdate, error) {
	return nil, nil
	//update := &tracing.StateUpdate{
	//	OriginRoot:     sc.originRoot,
	//	Root:           sc.root,
	//	BlockNumber:    sc.blockNumber,
	//	AccountChanges: make(map[common.Address]*tracing.AccountChange, len(sc.accountsOrigin)),
	//	StorageChanges: make(map[common.Address]map[common.Hash]*tracing.StorageChange),
	//	CodeChanges:    make(map[common.Address]*tracing.CodeChange, len(sc.codes)),
	//	TrieChanges:    make(map[common.Hash]map[string]*tracing.TrieNodeChange),
	//}
	//// Gather all account changes
	//for addr, oldData := range sc.accountsOrigin {
	//	addrHash := crypto.Keccak256Hash(addr.Bytes())
	//	newData, exists := sc.accounts[addrHash]
	//	if !exists {
	//		return nil, fmt.Errorf("account %x not found", addr)
	//	}
	//	change := &tracing.AccountChange{}
	//
	//	if len(oldData) > 0 {
	//		acct, err := types.FullAccount(oldData)
	//		if err != nil {
	//			return nil, err
	//		}
	//		change.Prev = &types.StateAccount{
	//			Nonce:    acct.Nonce,
	//			Balance:  acct.Balance,
	//			Root:     acct.Root,
	//			CodeHash: acct.CodeHash,
	//		}
	//	}
	//	if len(newData) > 0 {
	//		acct, err := types.FullAccount(newData)
	//		if err != nil {
	//			return nil, err
	//		}
	//		change.New = &types.StateAccount{
	//			Nonce:    acct.Nonce,
	//			Balance:  acct.Balance,
	//			Root:     acct.Root,
	//			CodeHash: acct.CodeHash,
	//		}
	//	}
	//	update.AccountChanges[addr] = change
	//}
	//
	//// Gather all storage slot changes
	//for addr, slots := range sc.storagesOrigin {
	//	addrHash := crypto.Keccak256Hash(addr.Bytes())
	//	subset, exists := sc.storages[addrHash]
	//	if !exists {
	//		return nil, fmt.Errorf("storage %x not found", addr)
	//	}
	//	storageChanges := make(map[common.Hash]*tracing.StorageChange, len(slots))
	//
	//	for key, encPrev := range slots {
	//		// Get new value - handle both raw and hashed key formats
	//		var (
	//			exists  bool
	//			encNew  []byte
	//			decPrev []byte
	//			decNew  []byte
	//			err     error
	//		)
	//		if sc.rawStorageKey {
	//			encNew, exists = subset[crypto.Keccak256Hash(key.Bytes())]
	//		} else {
	//			encNew, exists = subset[key]
	//		}
	//		if !exists {
	//			return nil, fmt.Errorf("storage slot %x-%x not found", addr, key)
	//		}
	//
	//		// Decode the prev and new values
	//		if len(encPrev) > 0 {
	//			_, decPrev, _, err = rlp.Split(encPrev)
	//			if err != nil {
	//				return nil, fmt.Errorf("failed to decode prevValue: %v", err)
	//			}
	//		}
	//		if len(encNew) > 0 {
	//			_, decNew, _, err = rlp.Split(encNew)
	//			if err != nil {
	//				return nil, fmt.Errorf("failed to decode newValue: %v", err)
	//			}
	//		}
	//		storageChanges[key] = &tracing.StorageChange{
	//			Prev: common.BytesToHash(decPrev),
	//			New:  common.BytesToHash(decNew),
	//		}
	//	}
	//	update.StorageChanges[addr] = storageChanges
	//}
	//
	//// Gather all contract code changes
	//for addr, code := range sc.codes {
	//	change := &tracing.CodeChange{
	//		New: &tracing.ContractCode{
	//			Hash:   code.hash,
	//			Code:   code.blob,
	//			Exists: code.duplicate,
	//		},
	//	}
	//	if code.originHash != types.EmptyCodeHash {
	//		change.Prev = &tracing.ContractCode{
	//			Hash:   code.originHash,
	//			Code:   code.originBlob,
	//			Exists: true,
	//		}
	//	}
	//	update.CodeChanges[addr] = change
	//}
	//
	//// Gather all trie node changes
	//if sc.nodes != nil {
	//	for owner, subset := range sc.nodes.Sets {
	//		nodeChanges := make(map[string]*tracing.TrieNodeChange, len(subset.Origins))
	//		for path, oldNode := range subset.Origins {
	//			newNode, exists := subset.Nodes[path]
	//			if !exists {
	//				return nil, fmt.Errorf("node %x-%v not found", owner, path)
	//			}
	//			nodeChanges[path] = &tracing.TrieNodeChange{
	//				Prev: &trienode.Node{
	//					Hash: crypto.Keccak256Hash(oldNode),
	//					Blob: oldNode,
	//				},
	//				New: &trienode.Node{
	//					Hash: newNode.Hash,
	//					Blob: newNode.Blob,
	//				},
	//			}
	//		}
	//		update.TrieChanges[owner] = nodeChanges
	//	}
	//}
	//return update, nil
}
