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
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	triedbCleanHitMeter   = metrics.NewRegisteredMeter("trie/triedb/clean/hit", nil)
	triedbCleanMissMeter  = metrics.NewRegisteredMeter("trie/triedb/clean/miss", nil)
	triedbCleanReadMeter  = metrics.NewRegisteredMeter("trie/triedb/clean/read", nil)
	triedbCleanWriteMeter = metrics.NewRegisteredMeter("trie/triedb/clean/write", nil)

	triedbFallbackHitMeter  = metrics.NewRegisteredMeter("trie/triedb/clean/fallback/hit", nil)
	triedbFallbackReadMeter = metrics.NewRegisteredMeter("trie/triedb/clean/fallback/read", nil)

	triedbDirtyHitMeter   = metrics.NewRegisteredMeter("trie/triedb/dirty/hit", nil)
	triedbDirtyMissMeter  = metrics.NewRegisteredMeter("trie/triedb/dirty/miss", nil)
	triedbDirtyReadMeter  = metrics.NewRegisteredMeter("trie/triedb/dirty/read", nil)
	triedbDirtyWriteMeter = metrics.NewRegisteredMeter("trie/triedb/dirty/write", nil)

	triedbDirtyNodeHitDepthHist = metrics.NewRegisteredHistogram("trie/triedb/dirty/depth", nil, metrics.NewExpDecaySample(1028, 0.015))

	triedbCommitTimeTimer  = metrics.NewRegisteredTimer("trie/triedb/commit/time", nil)
	triedbCommitNodesMeter = metrics.NewRegisteredMeter("trie/triedb/commit/nodes", nil)
	triedbCommitSizeMeter  = metrics.NewRegisteredMeter("trie/triedb/commit/size", nil)

	triedbDiffLayerSizeMeter  = metrics.NewRegisteredMeter("trie/triedb/diff/size", nil)
	triedbDiffLayerNodesMeter = metrics.NewRegisteredMeter("trie/triedb/diff/nodes", nil)

	triedbReverseDiffTimeTimer = metrics.NewRegisteredTimer("trie/triedb/reversediff/time", nil)
	triedbReverseDiffSizeMeter = metrics.NewRegisteredMeter("trie/triedb/reversediff/size", nil)

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

	// errUnmatchedReverseDiff is returned if an unmatched reverse-diff is applied
	// to the database for state rollback.
	errUnmatchedReverseDiff = errors.New("reverse diff is not matched")

	// errStateUnrecoverable is returned if state is required to be reverted to
	// a destination without associated reverse diff available.
	errStateUnrecoverable = errors.New("state is unrecoverable")

	// errImmatureState is returned if state is required to be reverted to an
	// immature destination.
	errImmatureState = errors.New("immature state")
)

// Snapshot represents the functionality supported by a snapshot storage layer.
type Snapshot interface {
	// Root returns the root hash for which this snapshot was made.
	Root() common.Hash

	// NodeBlob retrieves the RLP-encoded trie node blob associated with
	// a particular key and the corresponding node hash. The passed key
	// should be encoded in storage format. No error will be returned if
	// the node is not found.
	NodeBlob(storage []byte, hash common.Hash) ([]byte, error)
}

// snapshot is the internal version of the snapshot data layer that supports some
// additional methods compared to the public API.
type snapshot interface {
	Snapshot

	// Node retrieves the trie node associated with a particular key and the
	// corresponding node hash. The passed key should be encoded in storage
	// format. No error will be returned if the node is not found.
	Node(storage []byte, hash common.Hash) (node, error)

	// Parent returns the subsequent layer of a snapshot, or nil if the base was
	// reached.
	//
	// Note, the method is an internal helper to avoid type switching between the
	// disk and diff layers. There is no locking involved.
	Parent() snapshot

	// Update creates a new layer on top of the existing snapshot diff tree with
	// the given dirty trie node set. All dirty nodes are indexed with the storage
	// format key. The deleted trie nodes are also included with the nil as the
	// node object.
	//
	// Note, the maps are retained by the method to avoid copying everything.
	Update(blockRoot common.Hash, blockNumber uint64, nodes map[string]*cachedNode) *diffLayer

	// Journal commits an entire diff hierarchy to disk into a single journal entry.
	// This is meant to be used during shutdown to persist the snapshot without
	// flattening everything down (bad for reorgs).
	Journal(buffer *bytes.Buffer) error

	// Stale returns whether this layer has become stale (was flattened across) or
	// if it's still live.
	Stale() bool

	// ID returns the id of associated reverse diff.
	ID() uint64
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
	hash common.Hash // Node hash, derived by node value hashing, always non-empty
	node node        // Cached collapsed trie node, or raw rlp data, nil for deleted node
	size uint16      // Byte size of the useful cached data, 0 for deleted node
}

// cachedNodeSize is the raw size of a cachedNode data structure without any
// node data included. It's an approximate size, but should be a lot better
// than not counting them.
var cachedNodeSize = int(reflect.TypeOf(cachedNode{}).Size())

// rlp returns the raw rlp encoded blob of the cached trie node, either directly
// from the cache, or by regenerating it from the collapsed node.
func (n *cachedNode) rlp() []byte {
	if n.node == nil {
		return nil
	}
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

	// WriteLegacy indicates whether the flushed data will be stored with
	// an additional piece of data according to the legacy state scheme. It's
	// mainly used in the archive node mode which requires all historical state
	// and storing the preserved state like genesis.
	WriteLegacy bool

	// ReadOnly mode indicates whether the database is opened in read only mode.
	// All the mutations like journaling, updating disk layer will all be rejected.
	ReadOnly bool

	// Fallback is the function used to find the fallback base layer root. It's pretty
	// common that there is no singleton trie persisted in the disk(e.g. migrated from
	// the legacy database) so the function provided can find the alternative root as
	// the base.
	Fallback func() common.Hash
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
	// It will be set automatically when the database is journaled during
	// the shutdown to reject all following unexpected mutations.
	readOnly      bool
	config        *Config
	lock          sync.RWMutex
	diskdb        ethdb.Database           // Persistent database to store the snapshot
	cleans        *fastcache.Cache         // Megabytes permitted using for read caches
	layers        map[common.Hash]snapshot // Collection of all known layers
	preimages     map[common.Hash][]byte   // Preimages of nodes from the secure trie
	preimagesSize common.StorageSize       // Storage size of the preimages cache
}

// NewDatabase attempts to load an already existing snapshot from a persistent
// key-value store (with a number of memory layers from a journal). If the journal
// is not matched with the base persistent layer, all the recorded diff layers
// are discarded.
func NewDatabase(diskdb ethdb.Database, config *Config) *Database {
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
	head := loadSnapshot(diskdb, cleans, config)
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

// Snapshot retrieves a snapshot belonging to the given block root, or fallback
// to legacy/archive format node in disk if no snapshot is maintained for that
// block.
func (db *Database) Snapshot(blockRoot common.Hash) Snapshot {
	db.lock.Lock()
	defer db.lock.Unlock()

	layer := db.layers[convertEmpty(blockRoot)]
	if layer != nil {
		return layer
	}
	blob := rawdb.ReadArchiveTrieNode(db.diskdb, blockRoot)
	if len(blob) == 0 {
		return nil
	}
	// If the legacy/archive format root node indeed exists,
	// create a shadow diff layer with empty diffs for state
	// accessing.
	dl := db.disklayer()
	diff := newDiffLayer(dl, blockRoot, dl.rid+1, nil)
	db.layers[blockRoot] = diff
	return diff
}

// Update adds a new snapshot into the tree, if that can be linked to an existing
// old parent. It is disallowed to insert a disk layer (the origin of all).
// The passed keys must all be encoded in the **storage** format.
func (db *Database) Update(root common.Hash, parentRoot common.Hash, nodes map[string]*cachedNode) error {
	// Reject noop updates to avoid self-loops. This is a special case that can
	// only happen for Clique networks where empty blocks don't modify the state
	// (0 block subsidy).
	//
	// Although we could silently ignore this internally, it should be the caller's
	// responsibility to avoid even attempting to insert such a snapshot.
	root, parentRoot = convertEmpty(root), convertEmpty(parentRoot)
	if root == parentRoot {
		if root == emptyRoot {
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
	snap := parent.(snapshot).Update(root, parent.(snapshot).ID()+1, nodes)
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
		base := diff.persist(db.config).(*diskLayer)

		// Replace the entire snapshot tree with the flat base
		db.layers = map[common.Hash]snapshot{base.root: base}
		return nil
	}
	db.cap(diff, layers)

	// Remove any layer that is stale or links into a stale layer
	children := make(map[common.Hash][]common.Hash)
	for root, snap := range db.layers {
		if diff, ok := snap.(*diffLayer); ok {
			parent := diff.Parent().Root()
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
func (db *Database) cap(diff *diffLayer, layers int) {
	// Dive until we run out of layers or reach the persistent database
	for i := 0; i < layers-1; i++ {
		// If we still have diff layers below, continue down
		if parent, ok := diff.Parent().(*diffLayer); ok {
			diff = parent
		} else {
			// Diff stack too shallow, return without modifications
			return
		}
	}
	// We're out of layers, flatten anything below, stopping if it's the disk or if
	// the memory limit is not yet exceeded.
	switch parent := diff.Parent().(type) {
	case *diskLayer:
		return

	case *diffLayer:
		// Hold the lock to prevent any read operations until the new
		// parent is linked correctly.
		diff.lock.Lock()
		base := parent.persist(db.config)
		db.layers[base.Root()] = base
		diff.parent = base
		diff.lock.Unlock()
		return

	default:
		panic(fmt.Sprintf("unknown data layer in triedb: %T", parent))
	}
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
	size := journal.Len()
	rawdb.WriteTrieJournal(db.diskdb, journal.Bytes())

	// Set the db in read only mode to reject all following mutations
	db.readOnly = true
	log.Info("Stored snapshot journal in triedb", "disk", diskroot, "size", common.StorageSize(size))
	return nil
}

// Clean wipes all available journal from the persistent database and discard
// all caches and diff layers. Using the given root to create a new disk layer.
func (db *Database) Clean(root common.Hash) {
	db.lock.Lock()
	defer db.lock.Unlock()

	rawdb.DeleteTrieJournal(db.diskdb)

	// Iterate over all layers and mark them as stale
	for _, layer := range db.layers {
		switch layer := layer.(type) {
		case *diskLayer:
			// Layer should be inactive now, mark it as stale
			layer.MarkStale()

		case *diffLayer:
			// If the layer is a simple diff, simply mark as stale
			layer.lock.Lock()
			atomic.StoreUint32(&layer.stale, 1)
			layer.lock.Unlock()

		default:
			panic(fmt.Sprintf("unknown layer type: %T", layer))
		}
	}
	// Re-allocate the clean cache to prevent hit the unexpected value.
	var cleans *fastcache.Cache
	if db.config != nil && db.config.Cache > 0 {
		cleans = fastcache.New(db.config.Cache * 1024 * 1024)
	}
	head := truncateFromHead(db.diskdb, 0)
	db.layers = map[common.Hash]snapshot{
		root: newDiskLayer(root, head, cleans, db.diskdb),
	}
	log.Info("Rebuild triedb", "root", root, "rid", head)
}

// revert applies the reverse diffs to the database by reverting the disk layer
// content. The passed clean cache should be empty.
// This function assumes the lock in db is already held.
func (db *Database) revert(diff *reverseDiff, cleans *fastcache.Cache) error {
	var (
		dl    = db.disklayer()
		root  = dl.Root()
		batch = dl.diskdb.NewBatch()
	)
	if diff.Root != root {
		return errUnmatchedReverseDiff
	}
	if dl.rid == 0 {
		return fmt.Errorf("%w: zero reverse diff id", errStateUnrecoverable)
	}
	dl.MarkStale()

	for _, state := range diff.States {
		if len(state.Val) > 0 { // RLP loses nil-ness, but `[]byte{}` is not a valid item, so reinterpret that
			rawdb.WriteTrieNode(batch, state.Key, state.Val)
		} else {
			rawdb.DeleteTrieNode(batch, state.Key)
		}
	}
	// Flush all state changes in an atomic batch write
	if err := batch.Write(); err != nil {
		log.Crit("Failed to write reverse diff", "err", err)
	}
	batch.Reset()

	// Delete the lookup and update the diff head first. In case crash
	// happens after updating diff head, it's still possible to recover
	// in the next restart by truncating extra reverse diff.
	rawdb.DeleteReverseDiffLookup(batch, diff.Parent)
	rawdb.WriteReverseDiffHead(batch, dl.rid-1)
	if err := batch.Write(); err != nil {
		log.Crit("Failed to delete reverse diff", "err", err)
	}
	batch.Reset()

	// Truncate the reverse diff from the freezer in the last step
	rawdb.DeleteReverseDiff(db.diskdb, dl.rid)

	// Recreate the disk layer with newly created clean cache
	ndl := newDiskLayer(diff.Parent, dl.rid-1, cleans, dl.diskdb)
	db.layers = map[common.Hash]snapshot{
		ndl.root: ndl,
	}
	return nil
}

// Rollback rollbacks the database to a specified historical point.
// The state is supported as the rollback destination only if it's
// canonical state and the corresponding reverse diffs are existent.
//
// If the database is opened in Anonymous mode, then the reverted state
// won't be pushed into disk directly, instead a shadowy "disk layer"
// will be created maintaining all changed states on the top of the
// real disk layer. The shadowy disk layer can be deleted afterward
// by calling CleanJunks.
func (db *Database) Rollback(target common.Hash) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	// Ensure the destination is recoverable
	target = convertEmpty(target)
	id := rawdb.ReadReverseDiffLookup(db.diskdb, target)
	if id == nil {
		return errStateUnrecoverable
	}
	current := db.disklayer().ID()
	if *id > current {
		return fmt.Errorf("%w dest: %d head: %d", errImmatureState, *id, current)
	}
	// Apply the reverse diffs with the given order.
	var cleans *fastcache.Cache
	if db.config != nil && db.config.Cache > 0 {
		cleans = fastcache.New(db.config.Cache * 1024 * 1024)
	}
	for current >= *id {
		diff, err := loadReverseDiff(db.diskdb, current)
		if err != nil {
			return err
		}
		if err := db.revert(diff, cleans); err != nil {
			return err
		}
		current -= 1
	}
	return nil
}

// StateRecoverable returns the indicator if the specified state is enabled to be recovered.
func (db *Database) StateRecoverable(root common.Hash) bool {
	db.lock.Lock()
	defer db.lock.Unlock()

	root = convertEmpty(root)
	id := rawdb.ReadReverseDiffLookup(db.diskdb, root)
	if id == nil {
		return false
	}
	if db.disklayer().ID() < *id {
		return false
	}
	// In theory all the reverse diffs starts from the given id until the disk layer
	// should be checked for presence. In practice, the check is too expensive. So
	// optimistically believe that all the reverse diffs are present.
	return true
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
	for {
		if dl, ok := snap.(*diskLayer); ok {
			return dl
		}
		snap = snap.Parent()
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
		return emptyRoot
	}
	return hash
}
