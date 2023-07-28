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
	"github.com/ethereum/go-ethereum/trie/triestate"
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
	Path []byte // Path of the node in the trie
	Blob []byte // RLP-encoded trie node blob, nil means the node is deleted
}

// journalNodes represents a list trie nodes belong to a single
// account or the main trie.
type journalNodes struct {
	Owner common.Hash
	Nodes []journalNode
}

// journalAccounts represents a list accounts belonging to the layer
type journalAccounts struct {
	Addresses []common.Address
	Accounts  [][]byte // Nil means the account was not present
}

// journalStorage represents a list slot changes belong to an account.
type journalStorage struct {
	Incomplete bool
	Account    common.Address
	Hashes     []common.Hash
	Slots      [][]byte
}

// loadJournal tries to parse the layer journal from the disk.
func (db *Database) loadJournal(diskRoot common.Hash) (layer, error) {
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
	// with disk layer. Note now we can ensure it's the layer journal
	// correct version, so we expect everything can be resolved properly.
	var root common.Hash
	if err := r.Decode(&root); err != nil {
		return nil, errMissDiskRoot
	}
	// The journal is not matched with persistent state, discard them.
	// It can happen that geth crashes without persisting the journal.
	if !bytes.Equal(root.Bytes(), diskRoot.Bytes()) {
		return nil, fmt.Errorf("%w want %x got %x", errUnmatchedJournal, root, diskRoot)
	}
	// Load the disk layer from the journal
	base, err := db.loadDiskLayer(r)
	if err != nil {
		return nil, err
	}
	// Load all the diff layers from the journal
	head, err := db.loadDiffLayer(base, r)
	if err != nil {
		return nil, err
	}
	log.Debug("Loaded layer journal", "diskroot", diskRoot, "diffhead", head.rootHash())
	return head, nil
}

// loadLayers loads a pre-existing state layer backed by a key-value store.
func (db *Database) loadLayers() layer {
	// Retrieve the root node of persistent state.
	_, root := rawdb.ReadAccountTrieNode(db.diskdb, nil)
	root = types.TrieRootHash(root)

	// Load the layers by resolving the journal
	head, err := db.loadJournal(root)
	if err == nil {
		return head
	}
	// Journal is not matched(or missing) with the persistent state, discard
	// it. Display log for discarding journal, but try to avoid showing
	// useless information when the db is created from scratch.
	if !(root == types.EmptyRootHash && errors.Is(err, errMissJournal)) {
		log.Info("Failed to load journal, discard it", "err", err)
	}
	// Return single layer with persistent state.
	return newDiskLayer(root, rawdb.ReadPersistentStateID(db.diskdb), db, nil, newNodeBuffer(db.bufferSize, nil, 0))
}

// loadDiskLayer reads the binary blob from the layer journal, reconstructing
// a new disk layer on it.
func (db *Database) loadDiskLayer(r *rlp.Stream) (layer, error) {
	// Resolve disk layer root
	var root common.Hash
	if err := r.Decode(&root); err != nil {
		return nil, fmt.Errorf("load disk root: %v", err)
	}
	// Resolve the state id of disk layer, it can be different
	// with the persistent id tracked in disk, the id distance
	// is the number of transitions aggregated in disk layer.
	var id uint64
	if err := r.Decode(&id); err != nil {
		return nil, fmt.Errorf("load state id: %v", err)
	}
	stored := rawdb.ReadPersistentStateID(db.diskdb)
	if stored > id {
		return nil, fmt.Errorf("invalid state id, stored %d resolved %d", stored, id)
	}
	// Resolve nodes cached in node buffer
	var encoded []journalNodes
	if err := r.Decode(&encoded); err != nil {
		return nil, fmt.Errorf("load disk accounts: %v", err)
	}
	nodes := make(map[common.Hash]map[string]*trienode.Node)
	for _, entry := range encoded {
		subset := make(map[string]*trienode.Node)
		for _, n := range entry.Nodes {
			if len(n.Blob) > 0 {
				subset[string(n.Path)] = trienode.New(crypto.Keccak256Hash(n.Blob), n.Blob)
			} else {
				subset[string(n.Path)] = trienode.NewDeleted()
			}
		}
		nodes[entry.Owner] = subset
	}
	// Calculate the internal state transitions by id difference.
	base := newDiskLayer(root, id, db, nil, newNodeBuffer(db.bufferSize, nodes, id-stored))
	return base, nil
}

// loadDiffLayer reads the next sections of a layer journal, reconstructing a new
// diff and verifying that it can be linked to the requested parent.
func (db *Database) loadDiffLayer(parent layer, r *rlp.Stream) (layer, error) {
	// Read the next diff journal entry
	var root common.Hash
	if err := r.Decode(&root); err != nil {
		// The first read may fail with EOF, marking the end of the journal
		if err == io.EOF {
			return parent, nil
		}
		return nil, fmt.Errorf("load diff root: %v", err)
	}
	var block uint64
	if err := r.Decode(&block); err != nil {
		return nil, fmt.Errorf("load block number: %v", err)
	}
	// Read in-memory trie nodes from journal
	var encoded []journalNodes
	if err := r.Decode(&encoded); err != nil {
		return nil, fmt.Errorf("load diff nodes: %v", err)
	}
	nodes := make(map[common.Hash]map[string]*trienode.Node)
	for _, entry := range encoded {
		subset := make(map[string]*trienode.Node)
		for _, n := range entry.Nodes {
			if len(n.Blob) > 0 {
				subset[string(n.Path)] = trienode.New(crypto.Keccak256Hash(n.Blob), n.Blob)
			} else {
				subset[string(n.Path)] = trienode.NewDeleted()
			}
		}
		nodes[entry.Owner] = subset
	}
	// Read state changes from journal
	var (
		jaccounts  journalAccounts
		jstorages  []journalStorage
		accounts   = make(map[common.Address][]byte)
		storages   = make(map[common.Address]map[common.Hash][]byte)
		incomplete = make(map[common.Address]struct{})
	)
	if err := r.Decode(&jaccounts); err != nil {
		return nil, fmt.Errorf("load diff states: %v", err)
	}
	for i, addr := range jaccounts.Addresses {
		accounts[addr] = jaccounts.Accounts[i]
	}
	for _, jstorage := range jstorages {
		set := make(map[common.Hash][]byte)
		for i, h := range jstorage.Hashes {
			if len(jstorage.Slots[i]) > 0 {
				set[h] = jstorage.Slots[i]
			} else {
				set[h] = nil
			}
		}
		if jstorage.Incomplete {
			incomplete[jstorage.Account] = struct{}{}
		}
		storages[jstorage.Account] = set
	}
	return db.loadDiffLayer(newDiffLayer(parent, root, parent.stateID()+1, block, nodes, triestate.New(accounts, storages, incomplete)), r)
}

// journal implements the layer interface, marshaling the un-flushed trie nodes
// along with layer meta data into provided byte buffer.
func (dl *diskLayer) journal(w io.Writer) error {
	// Ensure the layer didn't get stale
	if dl.isStale() {
		return errSnapshotStale
	}
	// Step one, write the disk root into the journal.
	if err := rlp.Encode(w, dl.root); err != nil {
		return err
	}
	// Step two, write the corresponding state id into the journal
	if err := rlp.Encode(w, dl.id); err != nil {
		return err
	}
	// Step three, write all unwritten nodes into the journal
	nodes := make([]journalNodes, 0, len(dl.buffer.nodes))
	for owner, subset := range dl.buffer.nodes {
		entry := journalNodes{Owner: owner}
		for path, node := range subset {
			entry.Nodes = append(entry.Nodes, journalNode{Path: []byte(path), Blob: node.Blob})
		}
		nodes = append(nodes, entry)
	}
	if err := rlp.Encode(w, nodes); err != nil {
		return err
	}
	log.Debug("Journaled disk layer", "root", dl.root, "nodes", len(dl.buffer.nodes))
	return nil
}

// journal implements the layer interface, writing the memory layer contents
// into a buffer to be stored in the database as the layer journal.
func (dl *diffLayer) journal(w io.Writer) error {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	// Journal the parent first
	if err := dl.parent.journal(w); err != nil {
		return err
	}
	// Everything below was journaled, persist this layer too
	if err := rlp.Encode(w, dl.root); err != nil {
		return err
	}
	if err := rlp.Encode(w, dl.block); err != nil {
		return err
	}
	// Write the accumulated trie nodes into buffer
	nodes := make([]journalNodes, 0, len(dl.nodes))
	for owner, subset := range dl.nodes {
		entry := journalNodes{Owner: owner}
		for path, node := range subset {
			entry.Nodes = append(entry.Nodes, journalNode{Path: []byte(path), Blob: node.Blob})
		}
		nodes = append(nodes, entry)
	}
	if err := rlp.Encode(w, nodes); err != nil {
		return err
	}
	// Write the accumulated state changes into buffer
	var jaccounts journalAccounts
	for addr, account := range dl.states.Accounts {
		jaccounts.Addresses = append(jaccounts.Addresses, addr)
		jaccounts.Accounts = append(jaccounts.Accounts, account)
	}
	if err := rlp.Encode(w, jaccounts); err != nil {
		return err
	}
	storage := make([]journalStorage, 0, len(dl.states.Storages))
	for addr, slots := range dl.states.Storages {
		entry := journalStorage{Account: addr}
		if _, ok := dl.states.Incomplete[addr]; ok {
			entry.Incomplete = true
		}
		for slotHash, slot := range slots {
			entry.Hashes = append(entry.Hashes, slotHash)
			entry.Slots = append(entry.Slots, slot)
		}
		storage = append(storage, entry)
	}
	if err := rlp.Encode(w, storage); err != nil {
		return err
	}
	log.Debug("Journaled diff layer", "root", dl.root, "parentLayer", dl.parent.rootHash(), "id", dl.stateID(), "block", dl.block, "nodes", len(dl.nodes))
	return nil
}
