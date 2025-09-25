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
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

const tempJournalSuffix = ".tmp"

var (
	errMissJournal       = errors.New("journal not found")
	errMissVersion       = errors.New("version not found")
	errUnexpectedVersion = errors.New("unexpected journal version")
	errMissDiskRoot      = errors.New("disk layer root not found")
	errUnmatchedJournal  = errors.New("unmatched journal")
)

// journalVersion ensures that an incompatible journal is detected and discarded.
//
// Changelog:
//
// - Version 0: initial version
// - Version 1: storage.Incomplete field is removed
// - Version 2: add post-modification state values
// - Version 3: a flag has been added to indicate whether the storage slot key is the raw key or a hash
const journalVersion uint64 = 3

// loadJournal tries to parse the layer journal from the disk.
func (db *Database) loadJournal(diskRoot common.Hash) (layer, error) {
	var reader io.Reader
	if path := db.journalPath(); path != "" && common.FileExist(path) {
		// If a journal file is specified, read it from there
		log.Info("Load database journal from file", "path", path)
		f, err := os.OpenFile(path, os.O_RDONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to read journal file %s: %w", path, err)
		}
		defer f.Close()
		reader = f
	} else {
		log.Info("Load database journal from disk")
		journal := rawdb.ReadTrieJournal(db.diskdb)
		if len(journal) == 0 {
			return nil, errMissJournal
		}
		reader = bytes.NewReader(journal)
	}
	r := rlp.NewStream(reader, 0)

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

// journalGenerator is a disk layer entry containing the generator progress marker.
type journalGenerator struct {
	// Indicator that whether the database was in progress of being wiped.
	// It's deprecated but keep it here for backward compatibility.
	Wiping bool

	Done     bool // Whether the generator finished creating the snapshot
	Marker   []byte
	Accounts uint64
	Slots    uint64
	Storage  uint64
}

// loadGenerator loads the state generation progress marker from the database.
func loadGenerator(db ethdb.KeyValueReader, hash nodeHasher) (*journalGenerator, common.Hash, error) {
	trieRoot, err := hash(rawdb.ReadAccountTrieNode(db, nil))
	if err != nil {
		return nil, common.Hash{}, err
	}
	// State generation progress marker is lost, rebuild it
	blob := rawdb.ReadSnapshotGenerator(db)
	if len(blob) == 0 {
		log.Info("State snapshot generator is not found")
		return nil, trieRoot, nil
	}
	// State generation progress marker is not compatible, rebuild it
	var generator journalGenerator
	if err := rlp.DecodeBytes(blob, &generator); err != nil {
		log.Info("State snapshot generator is not compatible")
		return nil, trieRoot, nil
	}
	// The state snapshot is inconsistent with the trie data and must
	// be rebuilt.
	//
	// Note: The SnapshotRoot and SnapshotGenerator are always consistent
	// with each other, both in the legacy state snapshot and the path database.
	// Therefore, if the SnapshotRoot does not match the trie root,
	// the entire generator is considered stale and must be discarded.
	stateRoot := rawdb.ReadSnapshotRoot(db)
	if trieRoot != stateRoot {
		log.Info("State snapshot is not consistent", "trie", trieRoot, "state", stateRoot)
		return nil, trieRoot, nil
	}
	// Slice null-ness is lost after rlp decoding, reset it back to empty
	if !generator.Done && generator.Marker == nil {
		generator.Marker = []byte{}
	}
	return &generator, trieRoot, nil
}

// loadLayers loads a pre-existing state layer backed by a key-value store.
func (db *Database) loadLayers() layer {
	// Retrieve the root node of persistent state.
	root, err := db.hasher(rawdb.ReadAccountTrieNode(db.diskdb, nil))
	if err != nil {
		log.Crit("Failed to compute node hash", "err", err)
	}
	// Load the layers by resolving the journal
	head, err := db.loadJournal(root)
	if err == nil {
		return head
	}
	// journal is not matched(or missing) with the persistent state, discard
	// it. Display log for discarding journal, but try to avoid showing
	// useless information when the db is created from scratch.
	if !(root == types.EmptyRootHash && errors.Is(err, errMissJournal)) {
		log.Info("Failed to load journal, discard it", "err", err)
	}
	// Return single layer with persistent state.
	return newDiskLayer(root, rawdb.ReadPersistentStateID(db.diskdb), db, nil, nil, newBuffer(db.config.WriteBufferSize, nil, nil, 0), nil)
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
		return nil, fmt.Errorf("invalid state id: stored %d resolved %d", stored, id)
	}
	// Resolve nodes cached in aggregated buffer
	var nodes nodeSet
	if err := nodes.decode(r); err != nil {
		return nil, err
	}
	// Resolve flat state sets in aggregated buffer
	var states stateSet
	if err := states.decode(r); err != nil {
		return nil, err
	}
	return newDiskLayer(root, id, db, nil, nil, newBuffer(db.config.WriteBufferSize, &nodes, &states, id-stored), nil), nil
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
	var nodes nodeSetWithOrigin
	if err := nodes.decode(r); err != nil {
		return nil, err
	}
	// Read flat states set (with original value attached) from journal
	var stateSet StateSetWithOrigin
	if err := stateSet.decode(r); err != nil {
		return nil, err
	}
	return db.loadDiffLayer(newDiffLayer(parent, root, parent.stateID()+1, block, &nodes, &stateSet), r)
}

// journal implements the layer interface, marshaling the un-flushed trie nodes
// along with layer meta data into provided byte buffer.
func (dl *diskLayer) journal(w io.Writer) error {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	// Ensure the layer didn't get stale
	if dl.stale {
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
	// Step three, write the accumulated trie nodes into the journal
	if err := dl.buffer.nodes.encode(w); err != nil {
		return err
	}
	// Step four, write the accumulated flat states into the journal
	if err := dl.buffer.states.encode(w); err != nil {
		return err
	}
	log.Debug("Journaled pathdb disk layer", "root", dl.root, "id", dl.id)
	return nil
}

// journal implements the layer interface, writing the memory layer contents
// into a buffer to be stored in the database as the layer journal.
func (dl *diffLayer) journal(w io.Writer) error {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	// journal the parent first
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
	if err := dl.nodes.encode(w); err != nil {
		return err
	}
	// Write the associated flat state set into buffer
	if err := dl.states.encode(w); err != nil {
		return err
	}
	log.Debug("Journaled pathdb diff layer", "root", dl.root, "parent", dl.parent.rootHash(), "id", dl.stateID(), "block", dl.block)
	return nil
}

// Journal commits an entire diff hierarchy to disk into a single journal entry.
// This is meant to be used during shutdown to persist the layer without
// flattening everything down (bad for reorgs). And this function will mark the
// database as read-only to prevent all following mutation to disk.
//
// The supplied root must be a valid trie hash value.
func (db *Database) Journal(root common.Hash) error {
	// Retrieve the head layer to journal from.
	l := db.tree.get(root)
	if l == nil {
		return fmt.Errorf("triedb layer [%#x] missing", root)
	}
	disk := db.tree.bottom()
	if l, ok := l.(*diffLayer); ok {
		log.Info("Persisting dirty state", "head", l.block, "root", root, "layers", l.id-disk.id+disk.buffer.layers)
	} else { // disk layer only on noop runs (likely) or deep reorgs (unlikely)
		log.Info("Persisting dirty state", "root", root, "layers", disk.buffer.layers)
	}
	// Block until the background flushing is finished and terminate
	// the potential active state generator.
	if err := disk.terminate(); err != nil {
		return err
	}
	start := time.Now()

	// Run the journaling
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if the database is in read only mode.
	if db.readOnly {
		return errDatabaseReadOnly
	}
	// Forcibly sync the ancient store before persisting the in-memory layers.
	// This prevents an edge case where the in-memory layers are persisted
	// but the ancient store is not properly closed, resulting in recent writes
	// being lost. After a restart, the ancient store would then be misaligned
	// with the disk layer, causing data corruption.
	if db.stateFreezer != nil {
		if err := db.stateFreezer.SyncAncient(); err != nil {
			return err
		}
	}
	// Store the journal into the database and return
	var (
		file        *os.File
		journal     io.Writer
		journalPath = db.journalPath()
	)
	if journalPath != "" {
		// Write into a temp file first
		err := os.MkdirAll(db.config.JournalDirectory, 0755)
		if err != nil {
			return err
		}
		tmp := journalPath + tempJournalSuffix
		file, err = os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("failed to open journal file %s: %w", tmp, err)
		}
		defer func() {
			if file != nil {
				file.Close()
				os.Remove(tmp) // Clean up temp file if we didn't successfully rename it
				log.Warn("Removed leftover temporary journal file", "path", tmp)
			}
		}()
		journal = file
	} else {
		journal = new(bytes.Buffer)
	}

	// Firstly write out the metadata of journal
	if err := rlp.Encode(journal, journalVersion); err != nil {
		return err
	}
	// Secondly write out the state root in disk, ensure all layers
	// on top are continuous with disk.
	diskRoot, err := db.hasher(rawdb.ReadAccountTrieNode(db.diskdb, nil))
	if err != nil {
		return err
	}
	if err := rlp.Encode(journal, diskRoot); err != nil {
		return err
	}
	// Finally write out the journal of each layer in reverse order.
	if err := l.journal(journal); err != nil {
		return err
	}

	// Store the journal into the database and return
	if file == nil {
		data := journal.(*bytes.Buffer)
		size := data.Len()
		rawdb.WriteTrieJournal(db.diskdb, data.Bytes())
		log.Info("Persisted dirty state to disk", "size", common.StorageSize(size), "elapsed", common.PrettyDuration(time.Since(start)))
	} else {
		stat, err := file.Stat()
		if err != nil {
			return err
		}
		size := int(stat.Size())

		// Close the temporary file and atomically rename it
		if err := file.Sync(); err != nil {
			return fmt.Errorf("failed to fsync the journal, %v", err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("failed to close the journal: %v", err)
		}
		// Replace the live journal with the newly generated one
		if err := os.Rename(journalPath+tempJournalSuffix, journalPath); err != nil {
			return fmt.Errorf("failed to rename the journal: %v", err)
		}
		if err := syncDir(db.config.JournalDirectory); err != nil {
			return fmt.Errorf("failed to fsync the dir: %v", err)
		}
		file = nil
		log.Info("Persisted dirty state to file", "path", journalPath, "size", common.StorageSize(size), "elapsed", common.PrettyDuration(time.Since(start)))
	}
	// Set the db in read only mode to reject all following mutations
	db.readOnly = true
	return nil
}
