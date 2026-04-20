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
)

// ContractCode represents contract bytecode mutation along with its
// associated metadata.
type ContractCode struct {
	Hash       common.Hash // Hash is the cryptographic hash of the current contract code.
	Blob       []byte      // Blob is the binary representation of the current contract code.
	OriginHash common.Hash // OriginHash is the cryptographic hash of the code before mutation.

	// Derived fields, populated only when state tracking is enabled.
	Duplicate  bool   // Duplicate indicates whether the updated code already exists.
	OriginBlob []byte // OriginBlob is the original binary representation of the contract code.
}

// AccountDelete represents a deletion operation for an Ethereum account.
type AccountDelete struct {
	Address        common.Address              // Address uniquely identifies the account.
	Origin         *types.StateAccount         // Origin is the account state prior to deletion (never be null).
	Storages       map[common.Hash]common.Hash // Storages contains mutated storage slots.
	StoragesOrigin map[common.Hash]common.Hash // StoragesOrigin holds original values of mutated slots; keys are hashes of raw storage slot keys.
}

// AccountUpdate represents an update operation for an Ethereum account.
type AccountUpdate struct {
	Address  common.Address              // Address uniquely identifies the account.
	Data     *types.StateAccount         // Data is the updated account state; nil indicates deletion.
	Origin   *types.StateAccount         // Origin is the previous account state; nil indicates non-existence.
	Code     *ContractCode               // Code contains updated contract code; nil if unchanged.
	Storages map[common.Hash]common.Hash // Storages contains updated storage slots.

	// StoragesOriginByKey and StoragesOriginByHash both record original values
	// of mutated storage slots:
	// - StoragesOriginByKey uses raw storage slot keys.
	// - StoragesOriginByHash uses hashed storage slot keys.
	StoragesOriginByKey  map[common.Hash]common.Hash
	StoragesOriginByHash map[common.Hash]common.Hash
}

// StorageKeyEncoding specifies the encoding scheme of a storage key.
type StorageKeyEncoding int

const (
	// StorageKeyHashed represents a hashed key (e.g. Keccak256).
	StorageKeyHashed StorageKeyEncoding = iota

	// StorageKeyPlain represents a raw (unhashed) key.
	StorageKeyPlain
)

// StateUpdate represents the difference between two states resulting from state
// execution. It contains information about mutated contract codes, accounts,
// and storage slots, along with their original values.
type StateUpdate struct {
	OriginRoot  common.Hash // Hash of the state before applying mutation
	Root        common.Hash // Hash of the state after applying mutation
	BlockNumber uint64      // Associated block number

	// Accounts contains mutated accounts, keyed by address hash.
	Accounts map[common.Hash]*types.StateAccount

	// Storages contains mutated storage slots, keyed by address
	// hash and storage slot key hash.
	Storages map[common.Hash]map[common.Hash]common.Hash

	// AccountsOrigin holds the original values of mutated accounts, keyed by address.
	AccountsOrigin map[common.Address]*types.StateAccount

	// StoragesOrigin holds the original values of mutated storage slots.
	// The key format depends on StorageKeyType:
	// - if StorageKeyType is plain:  keyed by account address and plain storage slot key.
	// - if StorageKeyType is hashed: keyed by account address and storage slot key hash.
	StoragesOrigin map[common.Address]map[common.Hash]common.Hash
	StorageKeyType StorageKeyEncoding

	Codes map[common.Address]*ContractCode // Codes contains the set of dirty codes
	Nodes *trienode.MergedNodeSet          // Aggregated dirty nodes caused by state changes
}

// Empty returns a flag indicating the state transition is empty or not.
func (sc *StateUpdate) Empty() bool {
	return sc.OriginRoot == sc.Root
}

// NewStateUpdate constructs a state update object by identifying the differences
// between two states through state execution. It combines the specified account
// deletions and account updates to create a complete state update.
func NewStateUpdate(typ StorageKeyEncoding, originRoot common.Hash, root common.Hash, blockNumber uint64, deletes map[common.Hash]*AccountDelete, updates map[common.Hash]*AccountUpdate, nodes *trienode.MergedNodeSet) *StateUpdate {
	var (
		accounts       = make(map[common.Hash]*types.StateAccount)
		accountsOrigin = make(map[common.Address]*types.StateAccount)
		storages       = make(map[common.Hash]map[common.Hash]common.Hash)
		storagesOrigin = make(map[common.Address]map[common.Hash]common.Hash)
		codes          = make(map[common.Address]*ContractCode)
	)
	// Since some accounts might be deleted and recreated within the same
	// block, deletions must be aggregated first.
	for addrHash, op := range deletes {
		addr := op.Address
		accounts[addrHash] = nil
		accountsOrigin[addr] = op.Origin

		if len(op.Storages) > 0 {
			storages[addrHash] = op.Storages
		}
		if len(op.StoragesOrigin) > 0 {
			storagesOrigin[addr] = op.StoragesOrigin
		}
	}
	// Aggregate account updates then.
	for addrHash, op := range updates {
		// Aggregate dirty contract codes if they are available.
		addr := op.Address
		if op.Code != nil {
			codes[addr] = op.Code
		}
		accounts[addrHash] = op.Data

		// Aggregate the account original value. If the account is already
		// present in the aggregated AccountsOrigin set, skip it.
		if _, found := accountsOrigin[addr]; !found {
			accountsOrigin[addr] = op.Origin
		}
		// Aggregate the storage mutation list. If a slot in op.storages is
		// already present in aggregated storages set, the value will be
		// overwritten.
		if len(op.Storages) > 0 {
			if _, exist := storages[addrHash]; !exist {
				storages[addrHash] = op.Storages
			} else {
				maps.Copy(storages[addrHash], op.Storages)
			}
		}
		// Aggregate the storage original values. If the slot is already present
		// in aggregated StoragesOrigin set, skip it.
		storageOriginSet := op.StoragesOriginByHash
		if typ == StorageKeyPlain {
			storageOriginSet = op.StoragesOriginByKey
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
		BlockNumber:    blockNumber,
		Accounts:       accounts,
		AccountsOrigin: accountsOrigin,
		Storages:       storages,
		StoragesOrigin: storagesOrigin,
		StorageKeyType: typ,
		Codes:          codes,
		Nodes:          nodes,
	}
}

// encodeSlot encodes the storage slot value by trimming all leading zeros
// and then RLP-encoding the result.
func encodeSlot(value common.Hash) []byte {
	if value == (common.Hash{}) {
		return nil
	}
	blob, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(value[:]))
	return blob
}

// EncodeMPTState encodes all state mutations alongside their original value
// into the Merkle-Patricia-Trie representation.
//
// It transforms account and storage updates into their corresponding MPT-encoded
// key-value mappings, using the same encoding rules as the Ethereum state trie.
func (sc *StateUpdate) EncodeMPTState() (map[common.Hash][]byte, map[common.Address][]byte, map[common.Hash]map[common.Hash][]byte, map[common.Address]map[common.Hash][]byte) {
	var (
		accounts      = make(map[common.Hash][]byte, len(sc.Accounts))
		storages      = make(map[common.Hash]map[common.Hash][]byte, len(sc.Storages))
		accountOrigin = make(map[common.Address][]byte, len(sc.AccountsOrigin))
		storageOrigin = make(map[common.Address]map[common.Hash][]byte, len(sc.StoragesOrigin))
	)
	for addr, prev := range sc.AccountsOrigin {
		if prev == nil {
			accountOrigin[addr] = nil
		} else {
			accountOrigin[addr] = types.SlimAccountRLP(*prev)
		}
	}
	for addrHash, data := range sc.Accounts {
		if data == nil {
			accounts[addrHash] = nil
		} else {
			accounts[addrHash] = types.SlimAccountRLP(*data)
		}
	}
	for addr, slots := range sc.StoragesOrigin {
		subset := make(map[common.Hash][]byte)
		for key, val := range slots {
			subset[key] = encodeSlot(val)
		}
		storageOrigin[addr] = subset
	}
	for addrHash, slots := range sc.Storages {
		subset := make(map[common.Hash][]byte)
		for key, val := range slots {
			subset[key] = encodeSlot(val)
		}
		storages[addrHash] = subset
	}
	return accounts, accountOrigin, storages, storageOrigin
}

// EncodeUBTState encodes all state mutations alongside their original value
// into the Unified-Binary-Trie representation.
//
// It transforms account and storage updates into their corresponding UBT-encoded
// key-value mappings, using the same encoding rules as the Ethereum state trie.
func (sc *StateUpdate) EncodeUBTState() (map[common.Hash][]byte, map[common.Address][]byte, map[common.Hash]map[common.Hash][]byte, map[common.Address]map[common.Hash][]byte) {
	var (
		accounts      = make(map[common.Hash][]byte, len(sc.Accounts))
		storages      = make(map[common.Hash]map[common.Hash][]byte, len(sc.Storages))
		accountOrigin = make(map[common.Address][]byte, len(sc.AccountsOrigin))
		storageOrigin = make(map[common.Address]map[common.Hash][]byte, len(sc.StoragesOrigin))
	)
	for addr, prev := range sc.AccountsOrigin {
		if prev == nil {
			accountOrigin[addr] = nil
		} else {
			accountOrigin[addr] = types.SlimAccountRLP(*prev)
		}
	}
	for addrHash, data := range sc.Accounts {
		if data == nil {
			accounts[addrHash] = nil
		} else {
			accounts[addrHash] = types.SlimAccountRLP(*data)
		}
	}
	for addr, slots := range sc.StoragesOrigin {
		subset := make(map[common.Hash][]byte)
		for key, val := range slots {
			subset[key] = encodeSlot(val)
		}
		storageOrigin[addr] = subset
	}
	for addrHash, slots := range sc.Storages {
		subset := make(map[common.Hash][]byte)
		for key, val := range slots {
			subset[key] = encodeSlot(val)
		}
		storages[addrHash] = subset
	}
	return accounts, accountOrigin, storages, storageOrigin
}

// deriveCodeFields derives the missing fields of contract code changes
// such as original code value.
//
// Note: This operation is expensive and not needed during normal state
// transitions. It is only required when SizeTracker or StateUpdate hook
// is enabled to produce accurate state statistics.
func (sc *StateUpdate) deriveCodeFields(reader ContractCodeReader) error {
	cache := make(map[common.Hash]bool)
	for addr, code := range sc.Codes {
		if code.OriginHash != types.EmptyCodeHash {
			blob := reader.Code(addr, code.OriginHash)
			if len(blob) == 0 {
				return fmt.Errorf("original code of %x is empty", addr)
			}
			code.OriginBlob = blob
		}
		if exists, ok := cache[code.Hash]; ok {
			code.Duplicate = exists
			continue
		}
		res := reader.Has(addr, code.Hash)
		cache[code.Hash] = res
		code.Duplicate = res
	}
	return nil
}

// ToTracingUpdate converts the internal StateUpdate to an exported tracing.StateUpdate.
func (sc *StateUpdate) ToTracingUpdate() (*tracing.StateUpdate, error) {
	update := &tracing.StateUpdate{
		OriginRoot:     sc.OriginRoot,
		Root:           sc.Root,
		BlockNumber:    sc.BlockNumber,
		AccountChanges: make(map[common.Address]*tracing.AccountChange, len(sc.AccountsOrigin)),
		StorageChanges: make(map[common.Address]map[common.Hash]*tracing.StorageChange),
		CodeChanges:    make(map[common.Address]*tracing.CodeChange, len(sc.Codes)),
		TrieChanges:    make(map[common.Hash]map[string]*tracing.TrieNodeChange),
	}
	// Gather all account changes
	for addr, oldData := range sc.AccountsOrigin {
		addrHash := crypto.Keccak256Hash(addr.Bytes())
		newData, exists := sc.Accounts[addrHash]
		if !exists {
			return nil, fmt.Errorf("account %x not found", addr)
		}
		change := &tracing.AccountChange{
			Prev: oldData,
			New:  newData,
		}
		update.AccountChanges[addr] = change
	}

	// Gather all storage slot changes
	for addr, slots := range sc.StoragesOrigin {
		addrHash := crypto.Keccak256Hash(addr.Bytes())
		subset, exists := sc.Storages[addrHash]
		if !exists {
			return nil, fmt.Errorf("storage %x not found", addr)
		}
		storageChanges := make(map[common.Hash]*tracing.StorageChange, len(slots))

		for key, oldData := range slots {
			// Get new value - handle both raw and hashed key formats
			var (
				exists  bool
				newData common.Hash
			)
			if sc.StorageKeyType == StorageKeyPlain {
				newData, exists = subset[crypto.Keccak256Hash(key.Bytes())]
			} else {
				newData, exists = subset[key]
			}
			if !exists {
				return nil, fmt.Errorf("storage slot %x-%x not found", addr, key)
			}
			storageChanges[key] = &tracing.StorageChange{
				Prev: oldData,
				New:  newData,
			}
		}
		update.StorageChanges[addr] = storageChanges
	}

	// Gather all contract code changes
	for addr, code := range sc.Codes {
		change := &tracing.CodeChange{
			New: &tracing.ContractCode{
				Hash:   code.Hash,
				Code:   code.Blob,
				Exists: code.Duplicate,
			},
		}
		if code.OriginHash != types.EmptyCodeHash {
			change.Prev = &tracing.ContractCode{
				Hash:   code.OriginHash,
				Code:   code.OriginBlob,
				Exists: true,
			}
		}
		update.CodeChanges[addr] = change
	}

	// Gather all trie node changes
	if sc.Nodes != nil {
		for owner, subset := range sc.Nodes.Sets {
			nodeChanges := make(map[string]*tracing.TrieNodeChange, len(subset.Origins))
			for path, oldNode := range subset.Origins {
				newNode, exists := subset.Nodes[path]
				if !exists {
					return nil, fmt.Errorf("node %x-%v not found", owner, path)
				}
				nodeChanges[path] = &tracing.TrieNodeChange{
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
			update.TrieChanges[owner] = nodeChanges
		}
	}
	return update, nil
}
