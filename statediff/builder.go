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

// Contains a batch of utility type declarations used by the tests. As the node
// operates on unique types, a lot of them are needed to check various features.

package statediff

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// Builder interface exposes the method for building a state diff between two blocks
type Builder interface {
	BuildStateDiff(oldStateRoot, newStateRoot common.Hash, blockNumber int64, blockHash common.Hash) (StateDiff, error)
}

type builder struct {
	chainDB    ethdb.Database
	blockChain *core.BlockChain
}

// NewBuilder is used to create a builder
func NewBuilder(db ethdb.Database, blockChain *core.BlockChain) Builder {
	return &builder{
		chainDB:    db,
		blockChain: blockChain,
	}
}

// BuildStateDiff builds a StateDiff object from two blocks
func (sdb *builder) BuildStateDiff(oldStateRoot, newStateRoot common.Hash, blockNumber int64, blockHash common.Hash) (StateDiff, error) {
	// Generate tries for old and new states
	stateCache := sdb.blockChain.StateCache()
	oldTrie, err := stateCache.OpenTrie(oldStateRoot)
	if err != nil {
		log.Error("Error creating trie for oldStateRoot", "error", err)
		return StateDiff{}, err
	}
	newTrie, err := stateCache.OpenTrie(newStateRoot)
	if err != nil {
		log.Error("Error creating trie for newStateRoot", "error", err)
		return StateDiff{}, err
	}

	// Find created accounts
	oldIt := oldTrie.NodeIterator([]byte{})
	newIt := newTrie.NodeIterator([]byte{})
	creations, err := sdb.collectDiffNodes(oldIt, newIt)
	if err != nil {
		log.Error("Error collecting creation diff nodes", "error", err)
		return StateDiff{}, err
	}

	// Find deleted accounts
	oldIt = oldTrie.NodeIterator([]byte{})
	newIt = newTrie.NodeIterator([]byte{})
	deletions, err := sdb.collectDiffNodes(newIt, oldIt)
	if err != nil {
		log.Error("Error collecting deletion diff nodes", "error", err)
		return StateDiff{}, err
	}

	// Find all the diffed keys
	createKeys := sortKeys(creations)
	deleteKeys := sortKeys(deletions)
	updatedKeys := findIntersection(createKeys, deleteKeys)

	// Build and return the statediff
	updatedAccounts, err := sdb.buildDiffIncremental(creations, deletions, updatedKeys)
	if err != nil {
		log.Error("Error building diff for updated accounts", "error", err)
		return StateDiff{}, err
	}
	createdAccounts, err := sdb.buildDiffEventual(creations)
	if err != nil {
		log.Error("Error building diff for created accounts", "error", err)
		return StateDiff{}, err
	}
	deletedAccounts, err := sdb.buildDiffEventual(deletions)
	if err != nil {
		log.Error("Error building diff for deleted accounts", "error", err)
		return StateDiff{}, err
	}

	return StateDiff{
		BlockNumber:     blockNumber,
		BlockHash:       blockHash,
		CreatedAccounts: createdAccounts,
		DeletedAccounts: deletedAccounts,
		UpdatedAccounts: updatedAccounts,
	}, nil
}

func (sdb *builder) collectDiffNodes(a, b trie.NodeIterator) (AccountsMap, error) {
	var diffAccounts = make(AccountsMap)
	it, _ := trie.NewDifferenceIterator(a, b)

	for {
		log.Debug("Current Path and Hash", "path", pathToStr(it), "hashold", it.Hash())
		if it.Leaf() {
			leafProof := make([][]byte, len(it.LeafProof()))
			copy(leafProof, it.LeafProof())
			leafPath := make([]byte, len(it.Path()))
			copy(leafPath, it.Path())
			leafKey := make([]byte, len(it.LeafKey()))
			copy(leafKey, it.LeafKey())
			leafKeyHash := common.BytesToHash(leafKey)
			leafValue := make([]byte, len(it.LeafBlob()))
			copy(leafValue, it.LeafBlob())
			// lookup account state
			var account state.Account
			if err := rlp.DecodeBytes(leafValue, &account); err != nil {
				log.Error("Error looking up account via address", "address", leafKeyHash, "error", err)
				return nil, err
			}
			aw := accountWrapper{
				Account:  account,
				RawKey:   leafKey,
				RawValue: leafValue,
				Proof:    leafProof,
				Path:     leafPath,
			}
			// record account to diffs (creation if we are looking at new - old; deletion if old - new)
			log.Debug("Account lookup successful", "address", leafKeyHash, "account", account)
			diffAccounts[leafKeyHash] = aw
		}
		cont := it.Next(true)
		if !cont {
			break
		}
	}

	return diffAccounts, nil
}

func (sdb *builder) buildDiffEventual(accounts AccountsMap) (AccountDiffsMap, error) {
	accountDiffs := make(AccountDiffsMap)
	for _, val := range accounts {
		storageDiffs, err := sdb.buildStorageDiffsEventual(val.Account.Root)
		if err != nil {
			log.Error("Failed building eventual storage diffs", "Address", common.BytesToHash(val.RawKey), "error", err)
			return nil, err
		}
		accountDiffs[common.BytesToHash(val.RawKey)] = AccountDiff{
			Key:     val.RawKey,
			Value:   val.RawValue,
			Proof:   val.Proof,
			Path:    val.Path,
			Storage: storageDiffs,
		}
	}

	return accountDiffs, nil
}

func (sdb *builder) buildDiffIncremental(creations AccountsMap, deletions AccountsMap, updatedKeys []string) (AccountDiffsMap, error) {
	updatedAccounts := make(AccountDiffsMap)
	for _, val := range updatedKeys {
		createdAcc := creations[common.HexToHash(val)]
		deletedAcc := deletions[common.HexToHash(val)]
		oldSR := deletedAcc.Account.Root
		newSR := createdAcc.Account.Root
		storageDiffs, err := sdb.buildStorageDiffsIncremental(oldSR, newSR)
		if err != nil {
			log.Error("Failed building storage diffs", "Address", val, "error", err)
			return nil, err
		}
		updatedAccounts[common.HexToHash(val)] = AccountDiff{
			Key:     createdAcc.RawKey,
			Value:   createdAcc.RawValue,
			Proof:   createdAcc.Proof,
			Path:    createdAcc.Path,
			Storage: storageDiffs,
		}
		delete(creations, common.HexToHash(val))
		delete(deletions, common.HexToHash(val))
	}

	return updatedAccounts, nil
}

func (sdb *builder) buildStorageDiffsEventual(sr common.Hash) ([]StorageDiff, error) {
	log.Debug("Storage Root For Eventual Diff", "root", sr.Hex())
	stateCache := sdb.blockChain.StateCache()
	sTrie, err := stateCache.OpenTrie(sr)
	if err != nil {
		log.Info("error in build storage diff eventual", "error", err)
		return nil, err
	}
	it := sTrie.NodeIterator(make([]byte, 0))
	storageDiffs := buildStorageDiffsFromTrie(it)
	return storageDiffs, nil
}

func (sdb *builder) buildStorageDiffsIncremental(oldSR common.Hash, newSR common.Hash) ([]StorageDiff, error) {
	log.Debug("Storage Roots for Incremental Diff", "old", oldSR.Hex(), "new", newSR.Hex())
	stateCache := sdb.blockChain.StateCache()

	oldTrie, err := stateCache.OpenTrie(oldSR)
	if err != nil {
		return nil, err
	}
	newTrie, err := stateCache.OpenTrie(newSR)
	if err != nil {
		return nil, err
	}

	oldIt := oldTrie.NodeIterator(make([]byte, 0))
	newIt := newTrie.NodeIterator(make([]byte, 0))
	it, _ := trie.NewDifferenceIterator(oldIt, newIt)
	storageDiffs := buildStorageDiffsFromTrie(it)

	return storageDiffs, nil
}

func buildStorageDiffsFromTrie(it trie.NodeIterator) []StorageDiff {
	storageDiffs := make([]StorageDiff, 0)
	for {
		log.Debug("Iterating over state at path ", "path", pathToStr(it))
		if it.Leaf() {
			log.Debug("Found leaf in storage", "path", pathToStr(it))
			leafProof := make([][]byte, len(it.LeafProof()))
			copy(leafProof, it.LeafProof())
			leafPath := make([]byte, len(it.Path()))
			copy(leafPath, it.Path())
			leafKey := make([]byte, len(it.LeafKey()))
			copy(leafKey, it.LeafKey())
			leafValue := make([]byte, len(it.LeafBlob()))
			copy(leafValue, it.LeafBlob())
			storageDiffs = append(storageDiffs, StorageDiff{
				Key:   leafKey,
				Value: leafValue,
				Path:  leafPath,
				Proof: leafProof,
			})
		}
		cont := it.Next(true)
		if !cont {
			break
		}
	}

	return storageDiffs
}

func (sdb *builder) addressByPath(path []byte) (*common.Address, error) {
	log.Debug("Looking up address from path", "path", hexutil.Encode(append([]byte("secure-key-"), path...)))
	addrBytes, err := sdb.chainDB.Get(append([]byte("secure-key-"), hexToKeyBytes(path)...))
	if err != nil {
		log.Error("Error looking up address via path", "path", hexutil.Encode(append([]byte("secure-key-"), path...)), "error", err)
		return nil, err
	}
	addr := common.BytesToAddress(addrBytes)
	log.Debug("Address found", "Address", addr)
	return &addr, nil
}
