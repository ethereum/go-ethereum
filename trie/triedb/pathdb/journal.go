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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package pathdb

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/trienode"
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
	Path []byte // Node path inside of the trie
	Blob []byte // RLP-encoded trie node blob, nil means the node is deleted
	Prev []byte // The previous value of trie node, rlp encoded. Nil means the node is non-existent
}

// journalNodes represents a list trie nodes belong to a single
// contract or the main account trie.
type journalNodes struct {
	Owner common.Hash
	Nodes []journalNode
}

// loadJournal tries to parse the snapshot journal from the disk.
func (db *Database) loadJournal(diskRoot common.Hash) (snapshot, error) {
	journal := rawdb.ReadTrieJournal(db.diskdb)
	if len(journal) == 0 {
		return nil, errMissJournal
	}
	r := rlp.NewStream(bytes.NewReader(journal), 0)

	// Firstly, resolve the first element as the journal version
	version, err := r.Uint64()
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
	base, err := db.loadDiskLayer(r)
	if err != nil {
		return nil, err
	}
	// Load all the snapshot diffs from the journal
	snapshot, err := db.loadDiffLayer(base, r)
	if err != nil {
		return nil, err
	}
	log.Debug("Loaded snapshot journal", "diskroot", diskRoot, "diffhead", snapshot.Root())
	return snapshot, nil
}

// loadSnapshot loads a pre-existing state snapshot backed by a key-value store.
func (db *Database) loadSnapshot() snapshot {
	// Retrieve the root node of in-disk state.
	_, root := rawdb.ReadAccountTrieNode(db.diskdb, nil)
	root = types.TrieRootHash(root)

	// Load the in-memory diff layers by resolving the journal
	snap, err := db.loadJournal(root)
	if err == nil {
		return snap
	}
	// Journal is not matched(or missing) with the in-disk state, discard it.
	// Display log for discarding journal, but try to avoid showing useless
	// information when the db is created from scratch.
	if !(root == types.EmptyRootHash && errors.Is(err, errMissJournal)) {
		log.Info("Failed to load journal, discard it", "err", err)
	}
	// Construct the entire layer tree with the single in-disk state.
	return newDiskLayer(root, rawdb.ReadPersistentStateID(db.diskdb), db, newDiskcache(db.dirtySize, nil, 0))
}

// loadDiskLayer reads the binary blob from the snapshot journal, reconstructing a new
// disk layer on it.
func (db *Database) loadDiskLayer(r *rlp.Stream) (snapshot, error) {
	// Resolve disk layer root
	var root common.Hash
	if err := r.Decode(&root); err != nil {
		return nil, fmt.Errorf("load disk root: %v", err)
	}
	// Resolve disk layer cached nodes
	var encoded []journalNodes
	if err := r.Decode(&encoded); err != nil {
		return nil, fmt.Errorf("load disk accounts: %v", err)
	}
	var nodes = make(map[common.Hash]map[string]*trienode.Node)
	for _, entry := range encoded {
		subset := make(map[string]*trienode.Node)
		for _, n := range entry.Nodes {
			if len(n.Blob) > 0 {
				subset[string(n.Path)] = trienode.New(crypto.Keccak256Hash(n.Blob), n.Blob)
			} else {
				subset[string(n.Path)] = trienode.New(common.Hash{}, nil)
			}
		}
		nodes[entry.Owner] = subset
	}
	// Resolve the state id of disk layer
	var id uint64
	if err := r.Decode(&id); err != nil {
		return nil, fmt.Errorf("load state id: %v", err)
	}
	stored := rawdb.ReadPersistentStateID(db.diskdb)
	if stored > id {
		return nil, fmt.Errorf("invalid state id, stored %d resolved %d", stored, id)
	}
	// Calculate the internal state transitions by id difference.
	base := newDiskLayer(root, id, db, newDiskcache(db.dirtySize, nodes, id-stored))
	return base, nil
}

// loadDiffLayer reads the next sections of a snapshot journal, reconstructing a new
// diff and verifying that it can be linked to the requested parent.
func (db *Database) loadDiffLayer(parent snapshot, r *rlp.Stream) (snapshot, error) {
	// Read the next diff journal entry
	var root common.Hash
	if err := r.Decode(&root); err != nil {
		// The first read may fail with EOF, marking the end of the journal
		if err == io.EOF {
			return parent, nil
		}
		return nil, fmt.Errorf("load diff root: %v", err)
	}
	var encoded []journalNodes
	if err := r.Decode(&encoded); err != nil {
		return nil, fmt.Errorf("load diff accounts: %v", err)
	}
	nodes := make(map[common.Hash]map[string]*trienode.WithPrev)
	for _, entry := range encoded {
		subset := make(map[string]*trienode.WithPrev)
		for _, n := range entry.Nodes {
			if len(n.Blob) > 0 {
				subset[string(n.Path)] = trienode.NewWithPrev(crypto.Keccak256Hash(n.Blob), n.Blob, n.Prev)
			} else {
				subset[string(n.Path)] = trienode.NewWithPrev(common.Hash{}, nil, n.Prev)
			}
		}
		nodes[entry.Owner] = subset
	}
	return db.loadDiffLayer(newDiffLayer(parent, root, parent.ID()+1, nodes), r)
}

// Journal terminates any in-progress snapshot generation, also implicitly pushing
// the progress into the database.
func (dl *diskLayer) Journal(buffer *bytes.Buffer) error {
	// Ensure the layer didn't get stale
	if dl.Stale() {
		return errSnapshotStale
	}
	// Step one, write the disk root into the journal.
	if err := rlp.Encode(buffer, dl.root); err != nil {
		return err
	}
	// Step two, write all accumulated dirty nodes into the journal
	nodes := make([]journalNodes, 0, len(dl.dirty.nodes))
	for owner, subset := range dl.dirty.nodes {
		entry := journalNodes{Owner: owner}
		for path, node := range subset {
			jnode := journalNode{Path: []byte(path)}
			if !node.IsDeleted() {
				jnode.Blob = node.Blob
			}
			entry.Nodes = append(entry.Nodes, jnode)
		}
		nodes = append(nodes, entry)
	}
	if err := rlp.Encode(buffer, nodes); err != nil {
		return err
	}
	// Step three, write the corresponding state id into the journal
	if err := rlp.Encode(buffer, dl.id); err != nil {
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
	nodes := make([]journalNodes, 0, len(dl.nodes))
	for owner, subset := range dl.nodes {
		entry := journalNodes{Owner: owner}
		for path, node := range subset {
			jnode := journalNode{Path: []byte(path), Prev: node.Prev}
			if !node.IsDeleted() {
				jnode.Blob = node.Blob
			}
			entry.Nodes = append(entry.Nodes, jnode)
		}
		nodes = append(nodes, entry)
	}
	if err := rlp.Encode(buffer, nodes); err != nil {
		return err
	}
	log.Debug("Journaled diff layer", "root", dl.root, "parent", dl.parent.Root(), "nodes", len(dl.nodes))
	return nil
}
