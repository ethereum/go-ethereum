// Copyright 2019 The go-ethereum Authors
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
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

var nullNode = common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000")

// Builder interface exposes the method for building a state diff between two blocks
type Builder interface {
	BuildStateDiff(oldStateRoot, newStateRoot common.Hash, blockNumber *big.Int, blockHash common.Hash) (StateDiff, error)
}

type builder struct {
	chainDB    ethdb.Database
	config     Config
	blockChain *core.BlockChain
	stateCache state.Database
}

// NewBuilder is used to create a state diff builder
func NewBuilder(db ethdb.Database, blockChain *core.BlockChain, config Config) Builder {
	return &builder{
		chainDB:    db,
		config:     config,
		blockChain: blockChain,
	}
}

// BuildStateDiff builds a StateDiff object from two blocks
func (sdb *builder) BuildStateDiff(oldStateRoot, newStateRoot common.Hash, blockNumber *big.Int, blockHash common.Hash) (StateDiff, error) {
	// Generate tries for old and new states
	sdb.stateCache = sdb.blockChain.StateCache()
	oldTrie, err := sdb.stateCache.OpenTrie(oldStateRoot)
	if err != nil {
		return StateDiff{}, fmt.Errorf("error creating trie for oldStateRoot: %v", err)
	}
	newTrie, err := sdb.stateCache.OpenTrie(newStateRoot)
	if err != nil {
		return StateDiff{}, fmt.Errorf("error creating trie for newStateRoot: %v", err)
	}

	// Find created accounts
	oldIt := oldTrie.NodeIterator([]byte{})
	newIt := newTrie.NodeIterator([]byte{})
	creations, err := sdb.collectDiffNodes(oldIt, newIt)
	if err != nil {
		return StateDiff{}, fmt.Errorf("error collecting creation diff nodes: %v", err)
	}

	// Find deleted accounts
	oldIt = oldTrie.NodeIterator([]byte{})
	newIt = newTrie.NodeIterator([]byte{})
	deletions, err := sdb.collectDiffNodes(newIt, oldIt)
	if err != nil {
		return StateDiff{}, fmt.Errorf("error collecting deletion diff nodes: %v", err)
	}

	// Find all the diffed keys
	createKeys := sortKeys(creations)
	deleteKeys := sortKeys(deletions)
	updatedKeys := findIntersection(createKeys, deleteKeys)

	// Build and return the statediff
	updatedAccounts, err := sdb.buildDiffIncremental(creations, deletions, updatedKeys)
	if err != nil {
		return StateDiff{}, fmt.Errorf("error building diff for updated accounts: %v", err)
	}
	createdAccounts, err := sdb.buildDiffEventual(creations)
	if err != nil {
		return StateDiff{}, fmt.Errorf("error building diff for created accounts: %v", err)
	}
	deletedAccounts, err := sdb.buildDiffEventual(deletions)
	if err != nil {
		return StateDiff{}, fmt.Errorf("error building diff for deleted accounts: %v", err)
	}

	return StateDiff{
		BlockNumber:     blockNumber,
		BlockHash:       blockHash,
		CreatedAccounts: createdAccounts,
		DeletedAccounts: deletedAccounts,
		UpdatedAccounts: updatedAccounts,
	}, nil
}

func (sdb *builder) isWatchedAddress(hashKey []byte) bool {
	// If we aren't watching any addresses, we are watching everything
	if len(sdb.config.WatchedAddresses) == 0 {
		return true
	}
	for _, addrStr := range sdb.config.WatchedAddresses {
		addr := common.HexToAddress(addrStr)
		addrHashKey := crypto.Keccak256(addr[:])
		if bytes.Equal(addrHashKey, hashKey) {
			return true
		}
	}
	return false
}

func (sdb *builder) collectDiffNodes(a, b trie.NodeIterator) (AccountsMap, error) {
	var diffAccounts = make(AccountsMap)
	it, _ := trie.NewDifferenceIterator(a, b)
	for {
		log.Debug("Current Path and Hash", "path", pathToStr(it), "old hash", it.Hash())
		if it.Leaf() && sdb.isWatchedAddress(it.LeafKey()) {
			leafKey := make([]byte, len(it.LeafKey()))
			copy(leafKey, it.LeafKey())
			leafKeyHash := common.BytesToHash(leafKey)
			leafValue := make([]byte, len(it.LeafBlob()))
			copy(leafValue, it.LeafBlob())
			// lookup account state
			var account state.Account
			if err := rlp.DecodeBytes(leafValue, &account); err != nil {
				return nil, fmt.Errorf("error looking up account via address %s\r\nerror: %v", leafKeyHash.Hex(), err)
			}
			aw := accountWrapper{
				Leaf:     true,
				Account:  &account,
				RawKey:   leafKey,
				RawValue: leafValue,
			}
			if sdb.config.PathsAndProofs {
				leafProof := make([][]byte, len(it.LeafProof()))
				copy(leafProof, it.LeafProof())
				leafPath := make([]byte, len(it.Path()))
				copy(leafPath, it.Path())
				aw.Proof = leafProof
				aw.Path = leafPath
			}
			// record account to diffs (creation if we are looking at new - old; deletion if old - new)
			log.Debug("Account lookup successful", "address", leafKeyHash, "account", account)
			diffAccounts[leafKeyHash] = aw
		} else if sdb.config.AllNodes && !bytes.Equal(nullNode, it.Hash().Bytes()) {
			nodeKey := it.Hash()
			node, err := sdb.stateCache.TrieDB().Node(nodeKey)
			if err != nil {
				return nil, fmt.Errorf("error looking up intermediate state trie node %s\r\nerror: %v", nodeKey.Hex(), err)
			}
			aw := accountWrapper{
				Leaf:     false,
				RawKey:   nodeKey.Bytes(),
				RawValue: node,
			}
			log.Debug("intermediate state trie node lookup successful", "key", nodeKey.Hex(), "value", node)
			diffAccounts[nodeKey] = aw
		}
		cont := it.Next(true)
		if !cont {
			break
		}
	}

	return diffAccounts, nil
}

func (sdb *builder) buildDiffEventual(accounts AccountsMap) ([]AccountDiff, error) {
	accountDiffs := make([]AccountDiff, 0)
	var err error
	for _, val := range accounts {
		// If account is not nil, we need to process storage diffs
		var storageDiffs []StorageDiff
		if val.Account != nil {
			storageDiffs, err = sdb.buildStorageDiffsEventual(val.Account.Root)
			if err != nil {
				return nil, fmt.Errorf("failed building eventual storage diffs for %s\r\nerror: %v", common.BytesToHash(val.RawKey), err)
			}
		}
		accountDiffs = append(accountDiffs, AccountDiff{
			Leaf:    val.Leaf,
			Key:     val.RawKey,
			Value:   val.RawValue,
			Proof:   val.Proof,
			Path:    val.Path,
			Storage: storageDiffs,
		})
	}

	return accountDiffs, nil
}

func (sdb *builder) buildDiffIncremental(creations AccountsMap, deletions AccountsMap, updatedKeys []string) ([]AccountDiff, error) {
	updatedAccounts := make([]AccountDiff, 0)
	var err error
	for _, val := range updatedKeys {
		hashKey := common.HexToHash(val)
		createdAcc := creations[hashKey]
		deletedAcc := deletions[hashKey]
		var storageDiffs []StorageDiff
		if deletedAcc.Account != nil && createdAcc.Account != nil {
			oldSR := deletedAcc.Account.Root
			newSR := createdAcc.Account.Root
			storageDiffs, err = sdb.buildStorageDiffsIncremental(oldSR, newSR)
			if err != nil {
				return nil, fmt.Errorf("failed building incremental storage diffs for %s\r\nerror: %v", hashKey.Hex(), err)
			}
		}
		updatedAccounts = append(updatedAccounts, AccountDiff{
			Leaf:    createdAcc.Leaf,
			Key:     createdAcc.RawKey,
			Value:   createdAcc.RawValue,
			Proof:   createdAcc.Proof,
			Path:    createdAcc.Path,
			Storage: storageDiffs,
		})
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
	return sdb.buildStorageDiffsFromTrie(it)
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
	return sdb.buildStorageDiffsFromTrie(it)
}

func (sdb *builder) buildStorageDiffsFromTrie(it trie.NodeIterator) ([]StorageDiff, error) {
	storageDiffs := make([]StorageDiff, 0)
	for {
		log.Debug("Iterating over state at path ", "path", pathToStr(it))
		if it.Leaf() {
			log.Debug("Found leaf in storage", "path", pathToStr(it))
			leafKey := make([]byte, len(it.LeafKey()))
			copy(leafKey, it.LeafKey())
			leafValue := make([]byte, len(it.LeafBlob()))
			copy(leafValue, it.LeafBlob())
			sd := StorageDiff{
				Leaf:  true,
				Key:   leafKey,
				Value: leafValue,
			}
			if sdb.config.PathsAndProofs {
				leafProof := make([][]byte, len(it.LeafProof()))
				copy(leafProof, it.LeafProof())
				leafPath := make([]byte, len(it.Path()))
				copy(leafPath, it.Path())
				sd.Proof = leafProof
				sd.Path = leafPath
			}
			storageDiffs = append(storageDiffs, sd)
		} else if sdb.config.AllNodes && !bytes.Equal(nullNode, it.Hash().Bytes()) {
			nodeKey := it.Hash()
			node, err := sdb.stateCache.TrieDB().Node(nodeKey)
			if err != nil {
				return nil, fmt.Errorf("error looking up intermediate storage trie node %s\r\nerror: %v", nodeKey.Hex(), err)
			}
			storageDiffs = append(storageDiffs, StorageDiff{
				Leaf:  false,
				Key:   nodeKey.Bytes(),
				Value: node,
			})
			log.Debug("intermediate storage trie node lookup successful", "key", nodeKey.Hex(), "value", node)
		}
		cont := it.Next(true)
		if !cont {
			break
		}
	}

	return storageDiffs, nil
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
