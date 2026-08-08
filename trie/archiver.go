// Copyright 2026 go-ethereum Authors
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
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/archive"
	"github.com/ethereum/go-ethereum/triedb/database"
)

// subtreeInfo holds information about a subtree to be archived.
// It contains all the data needed to write the subtree to an archive
// and replace it with an expiredNode in the database.
type subtreeInfo struct {
	path      []byte            // Hex-encoded path to subtree root
	owner     common.Hash       // Zero for account trie, account hash for storage
	height    int               // Height of subtree (from leaves)
	leaves    []*archive.Record // All leaf records (relative path + encoded node)
	nodePaths [][]byte          // Paths of all nodes to delete
	rootHash  common.Hash       // Hash of the original subtree root (for verification)
}

// Archiver handles the archival process of trie nodes.
// It walks the state trie, identifies subtrees at height 3,
// archives their leaf data, and replaces them with expiredNode markers.
//
// The archiver uses a streaming approach: it walks the trie using a
// NodeIterator, probes each node's height via bounded raw DB reads,
// and archives subtrees immediately when found. This keeps memory
// usage proportional to the iterator stack depth + the current subtree
// being processed, rather than loading the entire trie into memory.
type Archiver struct {
	db                 ethdb.Database
	triedb             database.NodeDatabase
	writer             *archive.ArchiveWriter
	compactionInterval uint64
	dryRun             bool
	stateRoot          common.Hash

	// Progress tracking
	subtreesArchived uint64
	bytesDeleted     uint64
	leavesArchived   uint64
	lastCompaction   uint64
}

// NewArchiver creates a new archiver instance.
//
// Parameters:
//   - db: The underlying key-value database
//   - triedb: The trie database for reading nodes
//   - writer: Archive file writer (can be nil for dry run)
//   - compactionInterval: Run compaction after this many subtrees (0 = disable)
//   - dryRun: If true, don't modify the database
func NewArchiver(db ethdb.Database, triedb database.NodeDatabase,
	writer *archive.ArchiveWriter, compactionInterval uint64, dryRun bool) *Archiver {
	return &Archiver{
		db:                 db,
		triedb:             triedb,
		writer:             writer,
		compactionInterval: compactionInterval,
		dryRun:             dryRun,
	}
}

// ProcessState archives subtrees from the given state root.
// It processes storage tries first, then the account trie.
func (a *Archiver) ProcessState(root common.Hash) error {
	a.stateRoot = root

	accountTrie, err := New(StateTrieID(root), a.triedb)
	if err != nil {
		return fmt.Errorf("failed to open account trie: %w", err)
	}

	log.Info("Processing storage tries")
	iter, err := accountTrie.NodeIterator(nil)
	if err != nil {
		return fmt.Errorf("failed to create account iterator: %w", err)
	}

	kvIter := NewIterator(iter)
	for kvIter.Next() {
		// Decode the account to check for storage
		var acc types.StateAccount
		if err := rlp.DecodeBytes(kvIter.Value, &acc); err != nil {
			log.Warn("Failed to decode account", "err", err)
			continue
		}
		if acc.Root == types.EmptyRootHash {
			continue
		}

		// Process this account's storage trie
		accountHash := common.BytesToHash(kvIter.Key)
		storageID := StorageTrieID(root, accountHash, acc.Root)
		storageTrie, err := New(storageID, a.triedb)
		if err != nil {
			log.Warn("Failed to open storage trie", "account", accountHash, "err", err)
			continue
		}

		if err := a.processTrie(accountHash, storageTrie); err != nil {
			log.Warn("Failed to process storage trie", "account", accountHash, "err", err)
		}
	}

	if kvIter.Err != nil {
		return fmt.Errorf("account iteration error: %w", kvIter.Err)
	}

	log.Info("Processing account trie", "root", root)
	if err := a.processTrie(common.Hash{}, accountTrie); err != nil {
		return fmt.Errorf("failed to process account trie: %w", err)
	}

	return nil
}

// processTrie finds and archives all height-3 subtrees in the trie using
// a streaming approach. It walks the trie with a NodeIterator, probes each
// node's height via bounded raw DB reads, and archives subtrees immediately.
//
// Memory usage is O(iterator_stack_depth + current_subtree_size) instead of
// O(entire_trie) as with the previous recursive approach.
func (a *Archiver) processTrie(owner common.Hash, t *Trie) error {
	if t.root == nil {
		return nil
	}

	iter, err := t.NodeIterator(nil)
	if err != nil {
		return fmt.Errorf("failed to create node iterator: %w", err)
	}

	var (
		lastLog = time.Now()
		found   uint64
	)

	for iter.Next(true) {
		if iter.Leaf() {
			continue
		}

		// Progress logging
		if time.Since(lastLog) > 30*time.Second {
			log.Info("Scanning trie for subtrees",
				"owner", owner,
				"path", common.Bytes2Hex(iter.Path()),
				"found", found,
				"archived", a.subtreesArchived)
			lastLog = time.Now()
		}

		path := copyBytes(iter.Path())
		hash := iter.Hash()
		if hash == (common.Hash{}) {
			// Embedded node (no hash), skip — it will be part of a
			// parent subtree.
			continue
		}

		// Probe subtree height via bounded raw DB reads.
		// This does NOT load the trie into memory — it reads blobs from
		// the DB, decodes them, computes height, and discards them.
		height := a.probeHeight(owner, path, hash, 3)
		if height != 3 {
			// Too small to archive; the iterator will visit children.
			// Too tall — descend into children to find height-3 subtrees.
			continue
		}

		// height == 3: collect and archive this subtree immediately.
		info := a.collectSubtree(owner, path, hash)
		if info == nil {
			continue
		}
		found++

		if err := a.archiveSubtree(info); err != nil {
			log.Warn("Failed to archive subtree", "path", common.Bytes2Hex(path), "err", err)
			continue
		}
		a.subtreesArchived++
		a.leavesArchived += uint64(len(info.leaves))

		if err := a.maybeCompact(); err != nil {
			log.Warn("Compaction failed", "err", err)
		}

		// Skip children — they're now archived.
		// We call Next(false) to move past the subtree without descending.
		iter.Next(false)
	}

	if iter.Error() != nil {
		return fmt.Errorf("iterator error: %w", iter.Error())
	}

	log.Info("Found subtrees to archive", "owner", owner, "count", found)
	return nil
}

// probeHeight computes the height of a node by reading from the raw DB.
// It stops early once height exceeds maxHeight (returns maxHeight+1).
// The decoded nodes are not retained — they are discarded after inspection.
//
// Height is measured from leaves: leaves=0, their parents=1, etc.
func (a *Archiver) probeHeight(owner common.Hash, path []byte, hash common.Hash, maxHeight int) int {
	blob := a.readNodeBlob(owner, path)
	if len(blob) == 0 {
		return 0
	}

	// Already expired — skip.
	if blob[0] == expiredNodeMarker {
		return -1
	}

	n, err := decodeNodeUnsafe(hash[:], blob)
	if err != nil {
		return 0
	}

	return a.nodeHeight(n, path, owner, maxHeight)
}

// nodeHeight computes the height of a decoded node, bounded by maxHeight.
// Returns maxHeight+1 early if the subtree is taller than maxHeight.
func (a *Archiver) nodeHeight(n node, path []byte, owner common.Hash, maxHeight int) int {
	switch n := n.(type) {
	case nil:
		return 0

	case valueNode:
		return 0

	case *shortNode:
		childPath := append(append([]byte{}, path...), n.Key...)
		switch child := n.Val.(type) {
		case valueNode:
			return 1 // shortNode → leaf
		case hashNode:
			if maxHeight <= 1 {
				return maxHeight + 1
			}
			childHeight := a.probeHeight(owner, childPath, common.BytesToHash(child), maxHeight-1)
			if childHeight < 0 {
				return -1 // expired child
			}
			return childHeight + 1
		default:
			// Inline node
			childHeight := a.nodeHeight(child, childPath, owner, maxHeight-1)
			if childHeight < 0 {
				return -1
			}
			return childHeight + 1
		}

	case *fullNode:
		maxH := 0
		for i, child := range n.Children[:16] {
			if child == nil {
				continue
			}
			childPath := append(append([]byte{}, path...), byte(i))
			var childHeight int
			switch c := child.(type) {
			case valueNode:
				childHeight = 0
			case hashNode:
				if maxH+1 > maxHeight {
					return maxHeight + 1
				}
				childHeight = a.probeHeight(owner, childPath, common.BytesToHash(c), maxHeight-1)
			default:
				childHeight = a.nodeHeight(c, childPath, owner, maxHeight-1)
			}
			if childHeight < 0 {
				continue // expired child, skip
			}
			h := childHeight + 1
			if h > maxH {
				maxH = h
			}
			if maxH > maxHeight {
				return maxHeight + 1
			}
		}
		return maxH

	case hashNode:
		return a.probeHeight(owner, path, common.BytesToHash(n), maxHeight)

	case *expiredNode:
		return -1
	}
	return 0
}

// collectSubtree reads a height-3 subtree from the raw DB and collects its
// leaves and node paths for archival. The subtree is bounded (height ≤ 3),
// so memory usage is limited.
func (a *Archiver) collectSubtree(owner common.Hash, path []byte, hash common.Hash) *subtreeInfo {
	blob := a.readNodeBlob(owner, path)
	if len(blob) == 0 {
		return nil
	}
	if blob[0] == expiredNodeMarker {
		return nil
	}

	n, err := decodeNodeUnsafe(hash[:], blob)
	if err != nil {
		log.Warn("Failed to decode node for collection", "path", common.Bytes2Hex(path), "err", err)
		return nil
	}

	info := &subtreeInfo{
		path:     copyBytes(path),
		owner:    owner,
		rootHash: hash,
	}

	leaves, nodePaths, height, err := a.collectNodeLeaves(n, path, nil, owner)
	if err != nil {
		log.Warn("Failed to collect subtree leaves", "path", common.Bytes2Hex(path), "err", err)
		return nil
	}

	info.height = height
	info.leaves = leaves
	info.nodePaths = append([][]byte{copyBytes(path)}, nodePaths...)
	return info
}

// collectNodeLeaves recursively collects all leaves and node paths in a
// bounded subtree. relPath is the path relative to the subtree root.
// Returns (leaves, nodePaths, height, error).
func (a *Archiver) collectNodeLeaves(n node, absPath, relPath []byte, owner common.Hash) ([]*archive.Record, [][]byte, int, error) {
	switch n := n.(type) {
	case nil:
		return nil, nil, 0, nil

	case valueNode:
		return []*archive.Record{{
			Path:  copyBytes(relPath),
			Value: []byte(n),
		}}, nil, 0, nil

	case *shortNode:
		childAbsPath := append(append([]byte{}, absPath...), n.Key...)
		var childNode node
		switch c := n.Val.(type) {
		case hashNode:
			resolved, err := a.resolveRawNode(owner, childAbsPath, common.BytesToHash(c))
			if err != nil {
				return nil, nil, 0, fmt.Errorf("resolve shortNode child at %s: %w", common.Bytes2Hex(childAbsPath), err)
			}
			childNode = resolved
		default:
			childNode = c
		}

		// Pass nil relPath to child — we prepend the key ourselves
		leaves, nodePaths, height, err := a.collectNodeLeaves(childNode, childAbsPath, nil, owner)
		if err != nil {
			return nil, nil, 0, err
		}

		// Prepend [relPath + extension key] to leaf relative paths
		prefix := append(append([]byte{}, relPath...), n.Key...)
		for _, leaf := range leaves {
			leaf.Path = append(append([]byte{}, prefix...), leaf.Path...)
		}

		return leaves, append([][]byte{copyBytes(absPath)}, nodePaths...), height + 1, nil

	case *fullNode:
		var (
			allLeaves []*archive.Record
			allPaths  [][]byte
			maxHeight int
		)
		for i, child := range n.Children[:16] {
			if child == nil {
				continue
			}
			childAbsPath := append(append([]byte{}, absPath...), byte(i))

			var childNode node
			switch c := child.(type) {
			case hashNode:
				resolved, err := a.resolveRawNode(owner, childAbsPath, common.BytesToHash(c))
				if err != nil {
					return nil, nil, 0, fmt.Errorf("resolve fullNode child[%x] at %s: %w", i, common.Bytes2Hex(childAbsPath), err)
				}
				childNode = resolved
			default:
				childNode = c
			}

			// Pass nil relPath to child — we prepend the index ourselves
			leaves, nodePaths, height, err := a.collectNodeLeaves(childNode, childAbsPath, nil, owner)
			if err != nil {
				return nil, nil, 0, err
			}

			// Prepend [relPath + branch index] to leaf relative paths
			prefix := append(append([]byte{}, relPath...), byte(i))
			for _, leaf := range leaves {
				leaf.Path = append(append([]byte{}, prefix...), leaf.Path...)
			}

			allLeaves = append(allLeaves, leaves...)
			allPaths = append(allPaths, nodePaths...)
			h := height + 1
			if h > maxHeight {
				maxHeight = h
			}
		}
		return allLeaves, allPaths, maxHeight, nil

	case hashNode:
		resolved, err := a.resolveRawNode(owner, absPath, common.BytesToHash(n))
		if err != nil {
			return nil, nil, 0, err
		}
		return a.collectNodeLeaves(resolved, absPath, relPath, owner)

	case *expiredNode:
		return nil, nil, 0, nil
	}
	return nil, nil, 0, nil
}

// readNodeBlob reads a trie node blob directly from the raw key-value
// database, bypassing pathdb layers.
func (a *Archiver) readNodeBlob(owner common.Hash, path []byte) []byte {
	if owner == (common.Hash{}) {
		return rawdb.ReadAccountTrieNode(a.db, path)
	}
	return rawdb.ReadStorageTrieNode(a.db, owner, path)
}

// resolveRawNode reads and decodes a trie node directly from the raw DB.
// Unlike resolveNode, this does NOT use the trie database (no caching,
// no diff layers). The decoded node is ephemeral and will be GC'd after use.
func (a *Archiver) resolveRawNode(owner common.Hash, path []byte, hash common.Hash) (node, error) {
	blob := a.readNodeBlob(owner, path)
	if len(blob) == 0 {
		return nil, fmt.Errorf("node not found: owner=%s path=%s", owner, common.Bytes2Hex(path))
	}
	if blob[0] == expiredNodeMarker {
		return &expiredNode{}, nil
	}
	return decodeNodeUnsafe(hash[:], blob)
}

// archiveSubtree writes leaves to archive and replaces subtree with expiredNode.
func (a *Archiver) archiveSubtree(info *subtreeInfo) error {
	if a.dryRun {
		log.Info("Would archive subtree",
			"path", common.Bytes2Hex(info.path),
			"owner", info.owner,
			"height", info.height,
			"leaves", len(info.leaves),
			"nodes", len(info.nodePaths))
		return nil
	}

	// 1. Write to archive file
	offset, size, err := a.writer.WriteSubtree(info.leaves)
	if err != nil {
		return fmt.Errorf("failed to write subtree to archive: %w", err)
	}

	// 2. Sync to ensure durability before modifying DB
	if err := a.writer.Sync(); err != nil {
		return fmt.Errorf("failed to sync archive: %w", err)
	}

	// 3. Verify archive round-trip: reconstruct trie from records and
	// check that the hash matches the original subtree root. This
	// catches any data corruption before we delete the original nodes.
	if info.rootHash != (common.Hash{}) {
		reconstructed, err := archiveRecordsToNode(info.leaves)
		if err != nil {
			return fmt.Errorf("archive verification failed: cannot reconstruct trie from records: %w", err)
		}
		h := newHasher(false)
		gotHash := common.BytesToHash(h.hash(reconstructed, true))
		returnHasherToPool(h)
		if gotHash != info.rootHash {
			return fmt.Errorf("archive verification failed: hash mismatch at path %s owner %s: got %s want %s (leaves=%d offset=%d size=%d)",
				common.Bytes2Hex(info.path), info.owner, gotHash, info.rootHash,
				len(info.leaves), offset, size)
		}
	}

	// 4. Batch database operations
	batch := a.db.NewBatch()

	// Delete all nodes in subtree (except the root which we'll overwrite)
	for _, nodePath := range info.nodePaths[1:] { // Skip first (root)
		if info.owner == (common.Hash{}) {
			rawdb.DeleteAccountTrieNode(batch, nodePath)
		} else {
			rawdb.DeleteStorageTrieNode(batch, info.owner, nodePath)
		}
		a.bytesDeleted += uint64(len(nodePath))
	}

	// Write expiredNode at subtree root
	expiredBlob := encodeExpiredNodeBlob(offset, size)
	if info.owner == (common.Hash{}) {
		rawdb.WriteAccountTrieNode(batch, info.path, expiredBlob)
	} else {
		rawdb.WriteStorageTrieNode(batch, info.owner, info.path, expiredBlob)
	}

	if err := batch.Write(); err != nil {
		return fmt.Errorf("failed to write batch: %w", err)
	}

	log.Debug("Archived subtree",
		"path", common.Bytes2Hex(info.path),
		"owner", info.owner,
		"leaves", len(info.leaves),
		"offset", offset,
		"size", size)

	return nil
}

// maybeCompact runs database compaction if the threshold is reached.
func (a *Archiver) maybeCompact() error {
	if a.compactionInterval == 0 {
		return nil
	}
	if a.subtreesArchived-a.lastCompaction >= a.compactionInterval {
		log.Info("Running database compaction", "subtrees", a.subtreesArchived)
		if err := a.db.Compact(nil, nil); err != nil {
			return err
		}
		a.lastCompaction = a.subtreesArchived
	}
	return nil
}

// encodeExpiredNodeBlob creates the raw bytes for an expiredNode.
// Format: 1-byte marker (0x00) + 8-byte offset + 8-byte size = 17 bytes
func encodeExpiredNodeBlob(offset, size uint64) []byte {
	buf := make([]byte, 1+2*archive.OffsetSize) // 17 bytes
	buf[0] = expiredNodeMarker                  // 0x00
	binary.BigEndian.PutUint64(buf[1:], offset)
	binary.BigEndian.PutUint64(buf[1+archive.OffsetSize:], size)
	return buf
}

// Stats returns archival statistics.
func (a *Archiver) Stats() (subtrees, leaves, bytesDeleted uint64) {
	return a.subtreesArchived, a.leavesArchived, a.bytesDeleted
}

// copyBytes returns a copy of the given byte slice.
func copyBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
