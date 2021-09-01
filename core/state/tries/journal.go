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

package tries

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

const journalVersion uint64 = 0

type journalNode struct {
	Key string
	Val []byte
}

// loadAndParseJournal tries to parse the snapshot journal in latest format.
func loadAndParseJournal(db ethdb.KeyValueStore, base *diskLayer) (snapshot, error) {
	journal := rawdb.ReadTriesJournal(db)
	if len(journal) == 0 {
		log.Warn("Loaded snapshot journal", "diskroot", base.root, "diffs", "missing")
		return base, nil
	}
	r := rlp.NewStream(bytes.NewReader(journal), 0)

	// Firstly, resolve the first element as the journal version
	version, err := r.Uint()
	if err != nil {
		log.Warn("Failed to resolve the journal version", "error", err)
		return base, nil
	}
	if version != journalVersion {
		log.Warn("Discarded the tries journal with wrong version", "required", journalVersion, "got", version)
		return base, nil
	}
	// Secondly, resolve the disk layer root, ensure it's continuous
	// with disk layer. Note now we can ensure it's the snapshot journal
	// correct version, so we expect everything can be resolved properly.
	var root common.Hash
	if err := r.Decode(&root); err != nil {
		return nil, errors.New("missing disk layer root")
	}
	// The diff journal is not matched with disk, discard them.
	// It can happen that Geth crashes without persisting the latest
	// diff journal.
	if !bytes.Equal(root.Bytes(), base.root.Bytes()) {
		log.Warn("Loaded snapshot journal", "diskroot", base.root, "diffs", "unmatched")
		return base, nil
	}
	// Load all the snapshot diffs from the journal
	snapshot, err := loadDiffLayer(base, r)
	if err != nil {
		return nil, err
	}
	log.Debug("Loaded snapshot journal", "diskroot", base.root, "diffhead", snapshot.Root())
	return snapshot, nil
}

// loadSnapshot loads a pre-existing state snapshot backed by a key-value store.
func loadSnapshot(diskdb ethdb.KeyValueStore, cache int, root common.Hash) (snapshot, error) {
	// Retrieve the block number and hash of the snapshot, failing if no snapshot
	// is present in the database (or crashed mid-update).
	baseRoot := rawdb.ReadPersistedTrieRoot(diskdb)
	if baseRoot == (common.Hash{}) {
		return nil, errors.New("missing or corrupted tries")
	}
	base := &diskLayer{
		diskdb: diskdb,
		cache:  fastcache.New(cache * 1024 * 1024),
		root:   baseRoot,
	}
	snapshot, err := loadAndParseJournal(diskdb, base)
	if err != nil {
		return nil, err
	}
	// Entire snapshot journal loaded, sanity check the head. If the loaded
	// snapshot is not matched with current state root, print a warning log
	// or discard the entire snapshot it's legacy snapshot.
	//
	// Possible scenario: Geth was crashed without persisting journal and then
	// restart, the head is rewound to the point with available state(trie)
	// which is below the snapshot. In this case the snapshot can be recovered
	// by re-executing blocks but right now it's unavailable.
	if head := snapshot.Root(); head != root {
		return nil, fmt.Errorf("head doesn't match snapshot: have %#x, want %#x", head, root)
	}
	return snapshot, nil
}

// loadDiffLayer reads the next sections of a snapshot journal, reconstructing a new
// diff and verifying that it can be linked to the requested parent.
func loadDiffLayer(parent snapshot, r *rlp.Stream) (snapshot, error) {
	// Read the next diff journal entry
	var root common.Hash
	if err := r.Decode(&root); err != nil {
		// The first read may fail with EOF, marking the end of the journal
		if err == io.EOF {
			return parent, nil
		}
		return nil, fmt.Errorf("load diff root: %v", err)
	}
	var nodes []journalNode
	if err := r.Decode(&nodes); err != nil {
		return nil, fmt.Errorf("load diff accounts: %v", err)
	}
	tireNodes := make(map[string][]byte)
	for _, entry := range nodes {
		if len(entry.Val) > 0 { // RLP loses nil-ness, but `[]byte{}` is not a valid item, so reinterpret that
			tireNodes[entry.Key] = entry.Val
		} else {
			tireNodes[entry.Key] = nil
		}
	}
	return loadDiffLayer(newDiffLayer(parent, root, tireNodes), r)
}

// Journal terminates any in-progress snapshot generation, also implicitly pushing
// the progress into the database.
func (dl *diskLayer) Journal(buffer *bytes.Buffer) (common.Hash, error) {
	// Ensure the layer didn't get stale
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if dl.stale {
		return common.Hash{}, ErrSnapshotStale
	}
	return dl.root, nil
}

// Journal writes the memory layer contents into a buffer to be stored in the
// database as the snapshot journal.
func (dl *diffLayer) Journal(buffer *bytes.Buffer) (common.Hash, error) {
	// Journal the parent first
	base, err := dl.parent.Journal(buffer)
	if err != nil {
		return common.Hash{}, err
	}
	// Ensure the layer didn't get stale
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if dl.Stale() {
		return common.Hash{}, ErrSnapshotStale
	}
	// Everything below was journalled, persist this layer too
	if err := rlp.Encode(buffer, dl.root); err != nil {
		return common.Hash{}, err
	}
	nodes := make([]journalNode, 0, len(dl.nodes))
	for key, blob := range dl.nodes {
		nodes = append(nodes, journalNode{Key: key, Val: blob})
	}
	if err := rlp.Encode(buffer, nodes); err != nil {
		return common.Hash{}, err
	}
	log.Debug("Journalled diff layer", "root", dl.root, "parent", dl.parent.Root())
	return base, nil
}
