// Copyright 2021 The go-ethereum Authors
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

package trie

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	errMissJournal       = errors.New("journal not found")
	errMissVersion       = errors.New("version not found")
	errUnexpectedVersion = errors.New("unexpected journal version")
	errMissDiskRoot      = errors.New("disk layer root not found")
	errUnmatchedJournal  = errors.New("unmatched journal")
)

const journalVersion uint64 = 0

// journalNode represents a trie node persisted in the journal.
type journalNode struct {
	Key string // Storage format trie node key
	Val []byte // RLP-encoded trie node blob, nil means the node is deleted
	Pre []byte // The previous value of trie node, rlp encoded. Nil means the node is non-existent
}

// loadJournal tries to parse the snapshot journal from the disk.
func loadJournal(disk ethdb.Database, diskRoot common.Hash, cleans *fastcache.Cache) (snapshot, error) {
	journal := rawdb.ReadTrieJournal(disk)
	if len(journal) == 0 {
		return nil, errMissJournal
	}
	r := rlp.NewStream(bytes.NewReader(journal), 0)

	// Firstly, resolve the first element as the journal version
	version, err := r.Uint()
	if err != nil {
		return nil, errMissVersion
	}
	if version != journalVersion {
		return nil, fmt.Errorf("%w want %d got %d", errUnexpectedVersion, journalVersion, version)
	}
	// Secondly, resolve the disk layer root, ensure it's continuous
	// with disk layer. Note now we can ensure it's the snapshot journal
	// correct version, so we expect everything can be resolved properly.
	var root common.Hash
	if err := r.Decode(&root); err != nil {
		return nil, errMissDiskRoot
	}
	// The journal is not matched with disk state, discard them. It can
	// happen that Geth crashes without persisting the journal properly.
	if !bytes.Equal(root.Bytes(), diskRoot.Bytes()) {
		return nil, fmt.Errorf("%w want %x got %x", errUnmatchedJournal, root, diskRoot)
	}
	// Load the disk layer from the journal
	base, err := loadDiskLayer(r, cleans, disk)
	if err != nil {
		return nil, err
	}
	// Load all the snapshot diffs from the journal
	snapshot, err := loadDiffLayer(base, r)
	if err != nil {
		return nil, err
	}
	log.Debug("Loaded snapshot journal", "diskroot", diskRoot, "diffhead", snapshot.Root())
	return snapshot, nil
}

// loadSnapshot loads a pre-existing state snapshot backed by a key-value store.
func (db *snapDatabase) loadSnapshot(diskdb ethdb.Database, cleans *fastcache.Cache) snapshot {
	// Retrieve the root node of single persisted in-disk state.
	_, root := rawdb.ReadTrieNode(diskdb, EncodeStorageKey(common.Hash{}, nil))
	if root == (common.Hash{}) {
		root = emptyRoot
	}
	// Load the in-memory diff layers by resolving the journal
	snap, err := loadJournal(diskdb, root, cleans)
	if err == nil {
		return snap
	}
	// Journal is not matched(or missing) with the in-disk state, discard
	// it and reconstruct the snap tree with disk state. Display log for
	// discarding journal, but try to avoid showing useless information
	// when the db is created from scratch.
	if !(root == emptyRoot && errors.Is(err, errMissJournal)) {
		log.Info("Failed to load journal, discard it", "err", err)
	}
	// Try to find an alternative state root if the db is empty. It can
	// happen when upgrade from a legacy node database.
	if root == emptyRoot {
		base := rawdb.ReadLegacyStateRoot(diskdb)
		if base != (common.Hash{}) {
			root = base
		}
		// Purge the reverse diff freezer in case the disk layer is built
		// from the scratch. The extra diffs will be truncated later.
		var diffid uint64
		if db.freezer != nil && !db.readOnly {
			diffid, _ = db.freezer.Tail()
			rawdb.WriteReverseDiffHead(diskdb, diffid)
		}
		return newDiskLayer(root, diffid, cleans, newDiskcache(nil, 0), diskdb)
	}
	// Construct the snap tree with the single in-disk state.
	return newDiskLayer(root, rawdb.ReadReverseDiffHead(diskdb), cleans, newDiskcache(nil, 0), diskdb)
}

// loadDiskLayer reads the binary blob from the snapshot journal, reconstructing a new
// disk layer on it.
func loadDiskLayer(r *rlp.Stream, clean *fastcache.Cache, disk ethdb.Database) (snapshot, error) {
	// Resolve disk layer root
	var root common.Hash
	if err := r.Decode(&root); err != nil {
		return nil, fmt.Errorf("load disk root: %v", err)
	}
	// Resolve disk layer cached nodes
	var encoded []journalNode
	if err := r.Decode(&encoded); err != nil {
		return nil, fmt.Errorf("load disk accounts: %v", err)
	}
	var nodes = make(map[string]*cachedNode)
	for _, entry := range encoded {
		if len(entry.Val) > 0 {
			nodes[entry.Key] = &cachedNode{
				hash: crypto.Keccak256Hash(entry.Val),
				node: rawNode(entry.Val),
				size: uint16(len(entry.Val)),
			}
		} else {
			nodes[entry.Key] = &cachedNode{
				hash: common.Hash{},
				node: nil,
				size: 0,
			}
		}
	}
	// Resolve corresponding reverse diff id
	var diffid uint64
	if err := r.Decode(&diffid); err != nil {
		return nil, fmt.Errorf("load reverse diff id: %v", err)
	}
	diskid := rawdb.ReadReverseDiffHead(disk)
	if diskid > diffid {
		return nil, fmt.Errorf("invalid reverse diff id, disk %d resolved %d", diskid, diffid)
	}
	// Calculate the internal state transitions by id difference.
	seq := diffid - diskid
	base := newDiskLayer(root, diffid, clean, newDiskcache(nodes, seq), disk)
	return base, nil
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
	var encoded []journalNode
	if err := r.Decode(&encoded); err != nil {
		return nil, fmt.Errorf("load diff accounts: %v", err)
	}
	nodes := make(map[string]*nodeWithPreValue)
	for _, entry := range encoded {
		if len(entry.Val) > 0 {
			nodes[entry.Key] = &nodeWithPreValue{
				cachedNode: &cachedNode{
					hash: crypto.Keccak256Hash(entry.Val),
					node: rawNode(entry.Val),
					size: uint16(len(entry.Val)),
				},
				pre: entry.Pre,
			}
		} else {
			nodes[entry.Key] = &nodeWithPreValue{
				cachedNode: &cachedNode{
					hash: common.Hash{},
					node: nil,
					size: 0,
				},
				pre: entry.Pre,
			}
		}
	}
	return loadDiffLayer(newDiffLayer(parent, root, parent.ID()+1, nodes), r)
}

// Journal terminates any in-progress snapshot generation, also implicitly pushing
// the progress into the database.
func (dl *diskLayer) Journal(buffer *bytes.Buffer) error {
	// Ensure the layer didn't get stale
	if dl.Stale() {
		return errSnapshotStale
	}
	// Step one, write the disk root into the journal.
	diskroot := dl.root
	if diskroot == (common.Hash{}) {
		return errors.New("invalid disk root in triedb")
	}
	if err := rlp.Encode(buffer, diskroot); err != nil {
		return err
	}
	// Step two, write all accumulated dirty nodes into the journal
	nodes := make([]journalNode, 0, len(dl.dirty.nodes))
	for key, node := range dl.dirty.nodes {
		jnode := journalNode{Key: key}
		if node.node != nil {
			jnode.Val = node.rlp()
		}
		nodes = append(nodes, jnode)
	}
	if err := rlp.Encode(buffer, nodes); err != nil {
		return err
	}
	// Step three, write the corresponding reverse diff id into the journal
	if err := rlp.Encode(buffer, dl.diffid); err != nil {
		return err
	}
	log.Debug("Journaled disk layer", "root", dl.root, "nodes", len(dl.dirty.nodes))
	return nil
}

// Journal writes the memory layer contents into a buffer to be stored in the
// database as the snapshot journal.
func (dl *diffLayer) Journal(buffer *bytes.Buffer) error {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	// Journal the parent first
	if err := dl.parent.Journal(buffer); err != nil {
		return err
	}
	if dl.Stale() {
		return errSnapshotStale
	}
	// Everything below was journaled, persist this layer too
	if err := rlp.Encode(buffer, dl.root); err != nil {
		return err
	}
	nodes := make([]journalNode, 0, len(dl.nodes))
	for key, node := range dl.nodes {
		jnode := journalNode{Key: key, Pre: node.pre}
		if node.node != nil {
			jnode.Val = node.rlp()
		}
		nodes = append(nodes, jnode)
	}
	if err := rlp.Encode(buffer, nodes); err != nil {
		return err
	}
	log.Debug("Journaled diff layer", "root", dl.root, "parent", dl.parent.Root(), "nodes", len(dl.nodes))
	return nil
}
