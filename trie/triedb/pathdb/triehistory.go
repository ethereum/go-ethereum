// Copyright 2022 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package pathdb

import (
	"bytes"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// Trie history records the state changes involved in executing a block. The state can be
// reverted to the previous state by applying the associated trie history object (reverse
// diff). Trie history objects are kept to guarantee that the system can perform state
// rollbacks in case of a deep reorg.
//
// Each state transition will generate a trie history object. Note that not every block
// has a corresponding state/history object. If a block performs no state changes
// whatsoever, no state is created for it. Each state/history will have a sequentially
// increasing number acting as its unique identifier.
//
// The trie history are written to disk (ancient store) when the corresponding diff layer
// is merged into the disk layer. At the same time, system can prune the oldest histories
// according to config.
//
//                                                        Disk State
//                                                            ^
//                                                            |
//   +------------+     +---------+     +---------+     +---------+
//   | Init State |---->| State 1 |---->|   ...   |---->| State n |
//   +------------+     +---------+     +---------+     +---------+
//
//                     +-----------+      +------+     +-----------+
//                     | History 1 |----> | ...  |---->| History n |
//                     +-----------+      +------+     +-----------+
//
// # Rollback
//
// If the system wants to roll back to a previous state n, it needs to ensure all history
// objects from n+1 up to the current disk layer are existent. The history objects are
// applied to the state in reverse order, starting from the current disk layer.

// trieHistoryVersion is the initial version of trie history structure.
const trieHistoryVersion = uint8(0)

// nodeDiff represents a change record of a trie node. The prev refers to the
// content before the change is applied.
type nodeDiff struct {
	Path []byte // Path of node inside of the trie
	Prev []byte // RLP-encoded node blob, nil means the node is previously non-existent
}

// trieDiff represents a list of trie node changes belong to a single contract
// trie or the main account trie.
type trieDiff struct {
	Owner common.Hash // Identifier of contract or empty for main account trie
	Nodes []nodeDiff  // The list of trie node diffs
}

// trieHistory represents a set of trie node changes belong to the same block.
// All the trie history in disk are linked with each other by a unique id
// (8byte integer), the tail(oldest) trie history will be pruned in order to
// control the storage size.
type trieHistory struct {
	Parent common.Hash // The corresponding state root of parent block
	Root   common.Hash // The corresponding state root which these diffs belong to
	Tries  []trieDiff  // The list of trie changes
}

func (h *trieHistory) encode() ([]byte, error) {
	var buf = new(bytes.Buffer)
	buf.WriteByte(trieHistoryVersion)
	if err := rlp.Encode(buf, h); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (h *trieHistory) decode(blob []byte) error {
	if len(blob) < 1 {
		return fmt.Errorf("no version tag")
	}
	switch blob[0] {
	case trieHistoryVersion:
		var dec trieHistory
		if err := rlp.DecodeBytes(blob[1:], &dec); err != nil {
			return err
		}
		h.Parent, h.Root, h.Tries = dec.Parent, dec.Root, dec.Tries
		return nil
	default:
		return fmt.Errorf("unknown trie history version %d", blob[0])
	}
}

// apply writes the reverse diffs into the provided database batch.
func (h *trieHistory) apply(batch ethdb.Batch) error {
	for _, entry := range h.Tries {
		accTrie := entry.Owner == (common.Hash{})
		for _, state := range entry.Nodes {
			if len(state.Prev) > 0 {
				if accTrie {
					rawdb.WriteAccountTrieNode(batch, state.Path, state.Prev)
				} else {
					rawdb.WriteStorageTrieNode(batch, entry.Owner, state.Path, state.Prev)
				}
			} else {
				if accTrie {
					rawdb.DeleteAccountTrieNode(batch, state.Path)
				} else {
					rawdb.DeleteStorageTrieNode(batch, entry.Owner, state.Path)
				}
			}
			// Ensure the reverted state matches with the history itself.
			if accTrie && len(state.Path) == 0 {
				root := crypto.Keccak256Hash(state.Prev)
				if len(state.Prev) == 0 {
					root = types.EmptyRootHash
				}
				if h.Parent == root {
					continue
				}
				return fmt.Errorf("corrupted history: expect %x got %x", h.Parent, root)
			}
		}
	}
	return nil
}

// loadTrieHistory reads and decodes the trie history by the given id.
func loadTrieHistory(freezer *rawdb.ResettableFreezer, id uint64) (*trieHistory, error) {
	blob := rawdb.ReadTrieHistory(freezer, id)
	if len(blob) == 0 {
		return nil, fmt.Errorf("trie history not found %d", id)
	}
	var dec trieHistory
	if err := dec.decode(blob); err != nil {
		return nil, err
	}
	return &dec, nil
}

// storeTrieHistory constructs the trie history for the passed bottom-most
// diff layer. After storing the corresponding trie history, it will also
// prune the stale histories from the disk with the given threshold.
// This function will panic if it's called for non-bottom-most diff layer.
func storeTrieHistory(freezer *rawdb.ResettableFreezer, dl *diffLayer, limit uint64) error {
	var (
		start = time.Now()
		enc   = &trieHistory{
			Parent: dl.Parent().Root(),
			Root:   dl.Root(),
		}
	)
	for owner, subset := range dl.nodes {
		entry := trieDiff{Owner: owner}
		for path, n := range subset {
			entry.Nodes = append(entry.Nodes, nodeDiff{
				Path: []byte(path),
				Prev: n.Prev,
			})
		}
		enc.Tries = append(enc.Tries, entry)
	}
	blob, err := enc.encode()
	if err != nil {
		return err
	}
	rawdb.WriteTrieHistory(freezer, dl.id, blob)
	trieHistorySizeMeter.Mark(int64(len(blob)))

	logs := []interface{}{
		"id", dl.id,
		"nodes", len(dl.nodes),
		"size", common.StorageSize(len(blob)),
	}
	// Prune stale trie histories if necessary
	if limit != 0 && dl.id > limit {
		pruned, err := truncateFromTail(freezer, dl.id-limit)
		if err != nil {
			return err
		}
		logs = append(logs, "pruned", pruned, "limit", limit)
	}
	duration := time.Since(start)
	trieHistoryTimeMeter.Update(duration)
	logs = append(logs, "elapsed", common.PrettyDuration(duration))
	log.Debug("Stored the trie history", logs...)
	return nil
}

// truncateFromHead removes the extra trie histories from the head with
// the given parameters. If the passed database is a non-freezer database,
// nothing to do here.
func truncateFromHead(freezer *rawdb.ResettableFreezer, nhead uint64) (int, error) {
	ohead, err := freezer.TruncateHead(nhead)
	if err != nil {
		return 0, err
	}
	return int(ohead - nhead), nil
}

// truncateFromTail removes the extra trie histories from the tail with
// the given parameters. If the passed database is a non-freezer database,
// nothing to do here.
// It returns the number of items removed from the tail.
func truncateFromTail(freezer *rawdb.ResettableFreezer, ntail uint64) (int, error) {
	otail, err := freezer.TruncateTail(ntail)
	if err != nil {
		return 0, err
	}
	return int(ntail - otail), nil
}
