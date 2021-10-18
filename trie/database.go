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
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	// ErrSnapshotStale is returned from data accessors if the underlying snapshot
	// layer had been invalidated due to the chain progressing forward far enough
	// to not maintain the layer's original state.
	ErrSnapshotStale = errors.New("snapshot stale")

	// ErrSnapshotReadOnly is returned if the database is opened in read only mode
	// and mutation is requested.
	ErrSnapshotReadOnly = errors.New("read only")

	// errSnapshotCycle is returned if a snapshot is attempted to be inserted
	// that forms a cycle in the snapshot tree.
	errSnapshotCycle = errors.New("snapshot cycle")

	// emptyRoot is the known root hash of an empty trie. In this package this
	// special hash is used to represent empty layer.
	emptyHash = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
)

// Snapshot represents the functionality supported by a snapshot storage layer.
type Snapshot interface {
	// Root returns the root hash for which this snapshot was made.
	Root() common.Hash

	// NodeBlob retrieves the RLP-encoded trie node blob associated with
	// a particular key. The passed key should be encoded in internal format
	// with hash encoded. No error will be returned if the node is not found.
	NodeBlob(internalKey []byte) ([]byte, error)
}

// snapshot is the internal version of the snapshot data layer that supports some
// additional methods compared to the public API.
type snapshot interface {
	Snapshot

	// Node retrieves the trie node associated with a particular key. The
	// passed key should be encoded in internal format with hash encoded.
	// No error will be returned if the node is not found.
	Node(internalKey []byte) (node, error)

	// Parent returns the subsequent layer of a snapshot, or nil if the base was
	// reached.
	//
	// Note, the method is an internal helper to avoid type switching between the
	// disk and diff layers. There is no locking involved.
	Parent() snapshot

	// Update creates a new layer on top of the existing snapshot diff tree with
	// the given dirty trie node set. The deleted trie nodes are also included
	// with the nil as the value.
	//
	// Note, the maps are retained by the method to avoid copying everything.
	Update(blockRoot common.Hash, nodes map[string]*cachedNode) *diffLayer

	// Journal commits an entire diff hierarchy to disk into a single journal entry.
	// This is meant to be used during shutdown to persist the snapshot without
	// flattening everything down (bad for reorgs).
	Journal(buffer *bytes.Buffer) error

	// Stale return whether this layer has become stale (was flattened across) or
	// if it's still live.
	Stale() bool
}

// rawNode is a simple binary blob used to differentiate between collapsed trie
// nodes and already encoded RLP binary blobs (while at the same time store them
// in the same cache fields).
type rawNode []byte

func (n rawNode) cache() (hashNode, bool)   { panic("this should never end up in a live trie") }
func (n rawNode) fstring(ind string) string { panic("this should never end up in a live trie") }

func (n rawNode) EncodeRLP(w io.Writer) error {
	_, err := w.Write(n)
	return err
}

// rawFullNode represents only the useful data content of a full node, with the
// caches and flags stripped out to minimize its data storage. This type honors
// the same RLP encoding as the original parent.
type rawFullNode [17]node

func (n rawFullNode) cache() (hashNode, bool)   { panic("this should never end up in a live trie") }
func (n rawFullNode) fstring(ind string) string { panic("this should never end up in a live trie") }

func (n rawFullNode) EncodeRLP(w io.Writer) error {
	var nodes [17]node

	for i, child := range n {
		if child != nil {
			nodes[i] = child
		} else {
			nodes[i] = nilValueNode
		}
	}
	return rlp.Encode(w, nodes)
}

// rawShortNode represents only the useful data content of a short node, with the
// caches and flags stripped out to minimize its data storage. This type honors
// the same RLP encoding as the original parent.
type rawShortNode struct {
	Key []byte
	Val node
}

func (n rawShortNode) cache() (hashNode, bool)   { panic("this should never end up in a live trie") }
func (n rawShortNode) fstring(ind string) string { panic("this should never end up in a live trie") }

// cachedNode is all the information we know about a single cached trie node
// in the memory database write layer.
type cachedNode struct {
	node node   // Cached collapsed trie node, or raw rlp data
	size uint16 // Byte size of the useful cached data
}

// cachedNodeSize is the raw size of a cachedNode data structure without any
// node data included. It's an approximate size, but should be a lot better
// than not counting them.
var cachedNodeSize = int(reflect.TypeOf(cachedNode{}).Size())

// rlp returns the raw rlp encoded blob of the cached trie node, either directly
// from the cache, or by regenerating it from the collapsed node.
func (n *cachedNode) rlp() []byte {
	if node, ok := n.node.(rawNode); ok {
		return node
	}
	blob, err := rlp.EncodeToBytes(n.node)
	if err != nil {
		panic(err)
	}
	return blob
}

// obj returns the decoded and expanded trie node, either directly from the cache,
// or by regenerating it from the rlp encoded blob.
func (n *cachedNode) obj(hash common.Hash) node {
	if node, ok := n.node.(rawNode); ok {
		return mustDecodeNode(hash[:], node)
	}
	return expandNode(hash[:], n.node)
}

// simplifyNode traverses the hierarchy of an expanded memory node and discards
// all the internal caches, returning a node that only contains the raw data.
func simplifyNode(n node) node {
	switch n := n.(type) {
	case *shortNode:
		// Short nodes discard the flags and cascade
		return &rawShortNode{Key: n.Key, Val: simplifyNode(n.Val)}

	case *fullNode:
		// Full nodes discard the flags and cascade
		node := rawFullNode(n.Children)
		for i := 0; i < len(node); i++ {
			if node[i] != nil {
				node[i] = simplifyNode(node[i])
			}
		}
		return node

	case valueNode, hashNode, rawNode:
		return n

	default:
		panic(fmt.Sprintf("unknown node type: %T", n))
	}
}

// expandNode traverses the node hierarchy of a collapsed storage node and converts
// all fields and keys into expanded memory form.
func expandNode(hash hashNode, n node) node {
	switch n := n.(type) {
	case *rawShortNode:
		// Short nodes need key and child expansion
		return &shortNode{
			Key: compactToHex(n.Key),
			Val: expandNode(nil, n.Val),
			flags: nodeFlag{
				hash: hash,
			},
		}

	case rawFullNode:
		// Full nodes need child expansion
		node := &fullNode{
			flags: nodeFlag{
				hash: hash,
			},
		}
		for i := 0; i < len(node.Children); i++ {
			if n[i] != nil {
				node.Children[i] = expandNode(nil, n[i])
			}
		}
		return node

	case valueNode, hashNode:
		return n

	default:
		panic(fmt.Sprintf("unknown node type: %T", n))
	}
}

// Config defines all necessary options for database.
type Config struct {
	Cache     int    // Memory allowance (MB) to use for caching trie nodes in memory
	Journal   string // Journal of clean cache to survive node restarts
	Preimages bool   // Flag whether the preimage of trie key is recorded

	// Archive mode indicates whether the flushed data will be stored with
	// an additional piece of data according to the legacy state scheme. It's
	// mainly used in the archive node mode which requires all historical state
	// and storing the preserved state like genesis.
	Archive bool

	// ReadOnly mode indicates whether the database is opened in read only mode.
	// All the mutations like journalling, updating disk layer will all be rejected.
	ReadOnly bool

	// Fallback is the function used to find the fallback base layer root. It's pretty
	// common that there is no singleton trie persisted in the disk(e.g. migrated from
	// the legacy database) so the function provided can find the alternative root as
	// the base.
	Fallback func() common.Hash

	// OnCommit is called when the in-memory trie nodes are flushed into the disk.
	// The passed key is in **internal** key format and the val can be nil which
	// indicates the node is deleted from the disk.
	OnCommit func(key, val []byte)
}

// Database is a multiple-layered structure for maintaining in-memory trie nodes.
// It consists of one persistent base layer backed by a key-value store, on top
// of which arbitrarily many in-memory diff layers are topped. The memory diffs
// can form a tree with branching, but the disk layer is singleton and common to
// all. If a reorg goes deeper than the disk layer, a batch of reverse diffs should
// be applied. The deepest reorg can be handled depends on the amount of reverse
// diffs tracked in the disk.
type Database struct {
	// readOnly is the flag whether the mutation is allowed to be applied.
	// It will be set automatically when the database is journalled during
	// the shutdown to reject all following unexpected mutations.
	readOnly      bool
	config        *Config
	lock          sync.RWMutex
	diskdb        ethdb.KeyValueStore      // Persistent database to store the snapshot
	cleans        *fastcache.Cache         // Megabytes permitted using for read caches
	layers        map[common.Hash]snapshot // Collection of all known layers
	preimages     map[common.Hash][]byte   // Preimages of nodes from the secure trie
	preimagesSize common.StorageSize       // Storage size of the preimages cache
}

// NewDatabase attempts to load an already existing snapshot from a persistent
// key-value store (with a number of memory layers from a journal). If the journal
// is not matched with the base persistent layer, all the recorded diff layers
// are discarded.
func NewDatabase(diskdb ethdb.KeyValueStore, config *Config) *Database {
	var cleans *fastcache.Cache
	if config != nil && config.Cache > 0 {
		if config.Journal == "" {
			cleans = fastcache.New(config.Cache * 1024 * 1024)
		} else {
			cleans = fastcache.LoadFromFileOrNew(config.Journal, config.Cache*1024*1024)
		}
	}
	var readOnly bool
	if config != nil {
		readOnly = config.ReadOnly
	}
	db := &Database{
		readOnly: readOnly,
		config:   config,
		diskdb:   diskdb,
		cleans:   cleans,
		layers:   make(map[common.Hash]snapshot),
	}
	head := loadSnapshot(diskdb, cleans, config, db)
	for head != nil {
		db.layers[head.Root()] = head
		head = head.Parent()
	}
	if config == nil || config.Preimages {
		db.preimages = make(map[common.Hash][]byte)
	}
	return db
}

// InsertPreimage writes a new trie node pre-image to the memory database if it's
// yet unknown. The method will NOT make a copy of the slice, only use if the
// preimage will NOT be changed later on.
func (db *Database) InsertPreimage(preimages map[common.Hash][]byte) {
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if preimage collection is disabled
	if db.preimages == nil {
		return
	}
	for hash, preimage := range preimages {
		if _, ok := db.preimages[hash]; ok {
			continue
		}
		db.preimages[hash] = preimage
		db.preimagesSize += common.StorageSize(common.HashLength + len(preimage))
	}
}

// Preimage retrieves a cached trie node pre-image from memory. If it cannot be
// found cached, the method queries the persistent database for the content.
func (db *Database) Preimage(hash common.Hash) []byte {
	// Short circuit if preimage collection is disabled
	if db.preimages == nil {
		return nil
	}
	db.lock.RLock()
	preimage := db.preimages[hash]
	db.lock.RUnlock()

	if preimage != nil {
		return preimage
	}
	return rawdb.ReadPreimage(db.diskdb, hash)
}

// Snapshot retrieves a snapshot belonging to the given block root, or nil if no
// snapshot is maintained for that block.
func (db *Database) Snapshot(blockRoot common.Hash) Snapshot {
	db.lock.RLock()
	defer db.lock.RUnlock()

	return db.layers[convertEmpty(blockRoot)]
}

// Update adds a new snapshot into the tree, if that can be linked to an existing
// old parent. It is disallowed to insert a disk layer (the origin of all).
// The passed keys must all be encoded in the internal format.
func (db *Database) Update(root common.Hash, parentRoot common.Hash, nodes map[string]*cachedNode) error {
	// Reject noop updates to avoid self-loops. This is a special case that can
	// only happen for Clique networks where empty blocks don't modify the state
	// (0 block subsidy).
	//
	// Although we could silently ignore this internally, it should be the caller's
	// responsibility to avoid even attempting to insert such a snapshot.
	root, parentRoot = convertEmpty(root), convertEmpty(parentRoot)
	if root == parentRoot {
		if root == emptyHash {
			return nil
		}
		return errSnapshotCycle
	}
	// Generate a new snapshot on top of the parent
	parent := db.Snapshot(parentRoot)
	if parent == nil {
		return fmt.Errorf("triedb parent [%#x] snapshot missing", parentRoot)
	}
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if the database is in read only mode.
	if db.readOnly {
		return ErrSnapshotReadOnly
	}
	snap := parent.(snapshot).Update(root, nodes)
	db.layers[snap.root] = snap
	return nil
}

// Cap traverses downwards the snapshot tree from a head block hash until the
// number of allowed layers are crossed. All layers beyond the permitted number
// are flattened downwards.
//
// Note, the final diff layer count in general will be one more than the amount
// requested. This happens because the bottom-most diff layer is the accumulator
// which may or may not overflow and cascade to disk. Since this last layer's
// survival is only known *after* capping, we need to omit it from the count if
// we want to ensure that *at least* the requested number of diff layers remain.
func (db *Database) Cap(root common.Hash, layers int) error {
	// Retrieve the head snapshot to cap from
	root = convertEmpty(root)
	snap := db.Snapshot(root)
	if snap == nil {
		return fmt.Errorf("triedb snapshot [%#x] missing", root)
	}
	diff, ok := snap.(*diffLayer)
	if !ok {
		return fmt.Errorf("triedb snapshot [%#x] is disk layer", root)
	}
	// Run the internal capping and discard all stale layers
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if the database is in read only mode.
	if db.readOnly {
		return ErrSnapshotReadOnly
	}
	// Move all of the accumulated preimages into a write batch
	if db.preimages != nil && db.preimagesSize > 4*1024*1024 {
		batch := db.diskdb.NewBatch()
		rawdb.WritePreimages(batch, db.preimages)
		if err := batch.Write(); err != nil {
			return err
		}
		db.preimages, db.preimagesSize = make(map[common.Hash][]byte), 0
	}
	// Flattening the bottom-most diff layer requires special casing since there's
	// no child to rewire to the grandparent. In that case we can fake a temporary
	// child for the capping and then remove it.
	if layers == 0 {
		// If full commit was requested, flatten the diffs and merge onto disk
		diff.lock.RLock()
		base := diffToDisk(diff.flatten().(*diffLayer), db.config)
		diff.lock.RUnlock()

		// Replace the entire snapshot tree with the flat base
		db.layers = map[common.Hash]snapshot{base.root: base}
		return nil
	}
	persisted := db.cap(diff, layers)

	// Remove any layer that is stale or links into a stale layer
	children := make(map[common.Hash][]common.Hash)
	for root, snap := range db.layers {
		if diff, ok := snap.(*diffLayer); ok {
			parent := diff.parent.Root()
			children[parent] = append(children[parent], root)
		}
	}
	var remove func(root common.Hash)
	remove = func(root common.Hash) {
		delete(db.layers, root)
		for _, child := range children[root] {
			remove(child)
		}
		delete(children, root)
	}
	for root, snap := range db.layers {
		if snap.Stale() {
			remove(root)
		}
	}
	// If the disk layer was modified, regenerate all the cumulative blooms
	if persisted != nil {
		var rebloom func(root common.Hash)
		rebloom = func(root common.Hash) {
			if diff, ok := db.layers[root].(*diffLayer); ok {
				diff.origin = persisted
			}
			for _, child := range children[root] {
				rebloom(child)
			}
		}
		rebloom(persisted.root)
	}
	return nil
}

// cap traverses downwards the diff tree until the number of allowed layers are
// crossed. All diffs beyond the permitted number are flattened downwards. If the
// layer limit is reached, memory cap is also enforced (but not before).
//
// The method returns the new disk layer if diffs were persisted into it.
//
// Note, the final diff layer count in general will be one more than the amount
// requested. This happens because the bottom-most diff layer is the accumulator
// which may or may not overflow and cascade to disk. Since this last layer's
// survival is only known *after* capping, we need to omit it from the count if
// we want to ensure that *at least* the requested number of diff layers remain.
func (db *Database) cap(diff *diffLayer, layers int) *diskLayer {
	// Dive until we run out of layers or reach the persistent database
	for i := 0; i < layers-1; i++ {
		// If we still have diff layers below, continue down
		if parent, ok := diff.parent.(*diffLayer); ok {
			diff = parent
		} else {
			// Diff stack too shallow, return without modifications
			return nil
		}
	}
	// We're out of layers, flatten anything below, stopping if it's the disk or if
	// the memory limit is not yet exceeded.
	switch parent := diff.parent.(type) {
	case *diskLayer:
		return nil

	case *diffLayer:
		// Hold the write lock until the flattened parent is linked correctly.
		// Otherwise, data race can happen which may lead the read operations to
		// a stale parent layer.
		diff.lock.Lock()
		defer diff.lock.Unlock()

		// Flatten the parent into the grandparent. The flattening internally obtains
		// a write lock on grandparent.
		flattened := parent.flatten().(*diffLayer)
		db.layers[flattened.root] = flattened

		diff.parent = flattened
		if flattened.memory < aggregatorMemoryLimit {
			return nil
		}
	default:
		panic(fmt.Sprintf("unknown data layer in triedb: %T", parent))
	}
	// If the bottom-most layer is larger than our memory cap, persist to disk
	bottom := diff.parent.(*diffLayer)

	bottom.lock.RLock()
	base := diffToDisk(bottom, db.config)
	bottom.lock.RUnlock()

	db.layers[base.root] = base
	diff.parent = base
	return base
}

// diffToDisk merges a bottom-most diff into the persistent disk layer underneath
// it. The method will panic if called onto a non-bottom-most diff layer. The disk
// layer persistence should be operated in an atomic way. All updates should be
// discarded if the whole transition if not finished.
func diffToDisk(bottom *diffLayer, config *Config) *diskLayer {
	var (
		base  = bottom.parent.(*diskLayer)
		batch = base.diskdb.NewBatch()
	)
	// Mark the original base as stale as we're going to create a new wrapper
	base.lock.Lock()
	if base.stale {
		panic("triedb parent disk layer is stale") // we've committed into the same base from two children, boo
	}
	base.stale = true
	base.lock.Unlock()

	// Push all updated accounts into the database.
	// TODO all the nodes belong to the same layer should be written
	// in atomic way. However a huge disk write should be avoid in the
	// first place. A balance needs to be found to ensure that the bottom
	// most layer is large enough to combine duplicated writes, and also
	// the big write can be avoided.
	for key, node := range bottom.nodes {
		path, hash := DecodeInternalKey([]byte(key))
		if node == nil {
			rawdb.DeleteTrieNode(batch, path)
		} else {
			rawdb.WriteTrieNode(batch, path, node.rlp())
			if config != nil && config.Archive {
				rawdb.WriteArchiveTrieNode(batch, hash, node.rlp())
			}
		}
		if config != nil && config.OnCommit != nil {
			config.OnCommit([]byte(key), node.rlp())
		}
	}
	// Flush all the updates in the single db operation. Ensure the
	// disk layer transition is atomic.
	if err := batch.Write(); err != nil {
		log.Crit("Failed to write bottom dirty trie nodes", "err", err)
	}
	log.Debug("Journalled triedb disk layer", "root", bottom.root)
	res := &diskLayer{
		root:   bottom.root,
		cache:  base.cache,
		diskdb: base.diskdb,
	}
	return res
}

// Journal commits an entire diff hierarchy to disk into a single journal entry.
// This is meant to be used during shutdown to persist the snapshot without
// flattening everything down (bad for reorgs). And this function will mark the
// database as read-only to prevent all following mutation to disk.
func (db *Database) Journal(root common.Hash) error {
	// Retrieve the head snapshot to journal from var snap snapshot
	root = convertEmpty(root)
	snap := db.Snapshot(root)
	if snap == nil {
		return fmt.Errorf("triedb snapshot [%#x] missing", root)
	}
	// Run the journaling
	db.lock.Lock()
	defer db.lock.Unlock()

	// Short circuit if the database is in read only mode.
	if db.readOnly {
		return ErrSnapshotReadOnly
	}
	// Firstly write out the metadata of journal
	journal := new(bytes.Buffer)
	if err := rlp.Encode(journal, journalVersion); err != nil {
		return err
	}
	diskroot := db.diskRoot()
	if diskroot == (common.Hash{}) {
		return errors.New("invalid disk root in triedb")
	}
	// Secondly write out the disk layer root, ensure the
	// diff journal is continuous with disk.
	if err := rlp.Encode(journal, diskroot); err != nil {
		return err
	}
	// Finally write out the journal of each layer in reverse order.
	if err := snap.(snapshot).Journal(journal); err != nil {
		return err
	}
	// Store the journal into the database and return
	rawdb.WriteTriesJournal(db.diskdb, journal.Bytes())

	// Set the db in read only mode to reject all following mutations
	db.readOnly = true
	log.Info("Stored snapshot journal in triedb", "disk", diskroot)
	return nil
}

// Rebuild wipes all available journal from the persistent database and discard
// all caches and diff layers. Using the given root to create a new disk layer.
func (db *Database) Rebuild(root common.Hash) {
	db.lock.Lock()
	defer db.lock.Unlock()

	rawdb.DeleteTriesJournal(db.diskdb)

	// Iterate over and mark all layers stale
	for _, layer := range db.layers {
		switch layer := layer.(type) {
		case *diskLayer:
			// Layer should be inactive now, mark it as stale
			layer.lock.Lock()
			layer.stale = true
			layer.lock.Unlock()

		case *diffLayer:
			// If the layer is a simple diff, simply mark as stale
			layer.lock.Lock()
			atomic.StoreUint32(&layer.stale, 1)
			layer.lock.Unlock()

		default:
			panic(fmt.Sprintf("unknown layer type: %T", layer))
		}
	}
	db.layers = map[common.Hash]snapshot{
		root: &diskLayer{
			db:     db,
			diskdb: db.diskdb,
			cache:  db.cleans,
			root:   root,
		},
	}
	log.Info("Rebuild triedb", "root", root)
}

// DiskDB retrieves the persistent storage backing the trie database.
func (db *Database) DiskDB() ethdb.KeyValueStore {
	return db.diskdb
}

// disklayer is an internal helper function to return the disk layer.
// The lock of trieDB is assumed to be held already.
func (db *Database) disklayer() *diskLayer {
	var snap snapshot
	for _, s := range db.layers {
		snap = s
		break
	}
	if snap == nil {
		return nil
	}
	switch layer := snap.(type) {
	case *diskLayer:
		return layer
	case *diffLayer:
		return layer.origin
	default:
		panic(fmt.Sprintf("%T: undefined layer", snap))
	}
}

// diskRoot is a internal helper function to return the disk layer root.
// The lock of snapTree is assumed to be held already.
func (db *Database) diskRoot() common.Hash {
	disklayer := db.disklayer()
	if disklayer == nil {
		return common.Hash{}
	}
	return disklayer.Root()
}

// DiskLayer returns the disk layer for state accessing. It's usually used
// as the fallback to access state in disk directly.
func (db *Database) DiskLayer() Snapshot {
	db.lock.RLock()
	defer db.lock.RUnlock()

	return db.disklayer()
}

// Size returns the current storage size of the memory cache in front of the
// persistent database layer.
func (db *Database) Size() (common.StorageSize, common.StorageSize) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	var nodes common.StorageSize
	for _, layer := range db.layers {
		if diff, ok := layer.(*diffLayer); ok {
			nodes += common.StorageSize(diff.memory)
		}
	}
	return nodes, db.preimagesSize
}

// saveCache saves clean state cache to given directory path
// using specified CPU cores.
func (db *Database) saveCache(dir string, threads int) error {
	if db.cleans == nil {
		return nil
	}
	log.Info("Writing clean trie cache to disk", "path", dir, "threads", threads)

	start := time.Now()
	err := db.cleans.SaveToFileConcurrent(dir, threads)
	if err != nil {
		log.Error("Failed to persist clean trie cache", "error", err)
		return err
	}
	log.Info("Persisted the clean trie cache", "path", dir, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

// SaveCache atomically saves fast cache data to the given dir using all
// available CPU cores.
func (db *Database) SaveCache(dir string) error {
	return db.saveCache(dir, runtime.GOMAXPROCS(0))
}

// SaveCachePeriodically atomically saves fast cache data to the given dir with
// the specified interval. All dump operation will only use a single CPU core.
func (db *Database) SaveCachePeriodically(dir string, interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			db.saveCache(dir, 1)
		case <-stopCh:
			return
		}
	}
}

// Config returns the configures used by db.
func (db *Database) Config() *Config {
	return db.config
}

// convertEmpty converts the given hash to predefined emptyHash if it's empty.
func convertEmpty(hash common.Hash) common.Hash {
	if hash == (common.Hash{}) {
		return emptyHash
	}
	return hash
}
