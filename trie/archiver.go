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

// processTrie finds and archives all height-3 subtrees in the trie.
func (a *Archiver) processTrie(owner common.Hash, t *Trie) error {
	if t.root == nil {
		return nil
	}

	subtrees := a.findHeight3Subtrees(t.root, nil, owner)
	log.Info("Found subtrees to archive", "owner", owner, "count", len(subtrees))

	lastLog := time.Now()
	for i, info := range subtrees {
		if time.Since(lastLog) > 30*time.Second {
			log.Info("Archiving subtrees", "owner", owner, "progress", fmt.Sprintf("%d/%d", i, len(subtrees)), "archived", a.subtreesArchived)
			lastLog = time.Now()
		}
		if err := a.archiveSubtree(info); err != nil {
			log.Warn("Failed to archive subtree", "path", common.Bytes2Hex(info.path), "err", err)
			continue
		}
		a.subtreesArchived++
		a.leavesArchived += uint64(len(info.leaves))

		if err := a.maybeCompact(); err != nil {
			log.Warn("Compaction failed", "err", err)
		}
	}
	return nil
}

// findHeight3Subtrees recursively finds all subtrees with height == 3.
// Height is measured from leaves: leaves=0, their parents=1, etc.
func (a *Archiver) findHeight3Subtrees(n node, path []byte, owner common.Hash) []*subtreeInfo {
	info, err := a.computeSubtreeInfo(n, path, owner)
	if err != nil {
		// computeSubtreeInfo failed (e.g. unresolvable hashNode within the
		// subtree). We cannot archive this node as-is, but deeper children
		// may still form valid height-3 subtrees. Recurse into them.
		log.Debug("computeSubtreeInfo failed, trying children", "path", common.Bytes2Hex(path), "err", err)
		return a.findSubtreesInChildren(n, path, owner)
	}
	if info == nil {
		return nil
	}

	// If this subtree has height 3, it's a candidate for archival
	if info.height == 3 {
		// Capture the original subtree root hash for verification.
		// The hash is available from the node that was passed in:
		// - hashNode: the hash IS the node
		// - fullNode/shortNode: loaded from DB, flags.hash is set
		switch nn := n.(type) {
		case hashNode:
			info.rootHash = common.BytesToHash(nn)
		case *fullNode:
			if nn.flags.hash != nil {
				info.rootHash = common.BytesToHash(nn.flags.hash)
			}
		case *shortNode:
			if nn.flags.hash != nil {
				info.rootHash = common.BytesToHash(nn.flags.hash)
			}
		}
		return []*subtreeInfo{info}
	}

	// If height > 3, recurse into children to find height-3 subtrees
	if info.height > 3 {
		return a.findSubtreesInChildren(n, path, owner)
	}

	// Height < 3: no archivable subtrees here
	return nil
}

// findSubtreesInChildren recurses into the children of a node to find
// height-3 subtrees. Used both by the normal height > 3 path and as a
// fallback when computeSubtreeInfo fails for a node.
func (a *Archiver) findSubtreesInChildren(n node, path []byte, owner common.Hash) []*subtreeInfo {
	var results []*subtreeInfo
	switch n := n.(type) {
	case *fullNode:
		for i, child := range n.Children[:16] {
			if child != nil {
				childPath := append(append([]byte{}, path...), byte(i))
				results = append(results, a.findHeight3Subtrees(child, childPath, owner)...)
			}
		}
	case *shortNode:
		childPath := append(append([]byte{}, path...), n.Key...)
		results = append(results, a.findHeight3Subtrees(n.Val, childPath, owner)...)
	case hashNode:
		// Resolve and recurse
		resolved, err := a.resolveNode(n, path, owner)
		if err == nil {
			results = append(results, a.findHeight3Subtrees(resolved, path, owner)...)
		}
	}
	return results
}

// computeSubtreeInfo computes height and collects leaves for a subtree.
// Returns (nil, nil) if the node is nil, already expired, or has no leaves.
// Returns (nil, error) if any constituent node could not be resolved — the
// caller MUST NOT archive a subtree when an error is returned, as the leaf
// set would be incomplete.
func (a *Archiver) computeSubtreeInfo(n node, path []byte, owner common.Hash) (*subtreeInfo, error) {
	switch n := n.(type) {
	case nil:
		return nil, nil

	case valueNode:
		// Leaf: height 0
		return &subtreeInfo{
			path:   copyBytes(path),
			owner:  owner,
			height: 0,
			leaves: []*archive.Record{{
				Path:  nil, // Empty relative path for leaf at root
				Value: []byte(n),
			}},
			nodePaths: [][]byte{copyBytes(path)},
		}, nil

	case *shortNode:
		childPath := append(append([]byte{}, path...), n.Key...)
		childInfo, err := a.computeSubtreeInfo(n.Val, childPath, owner)
		if err != nil {
			return nil, fmt.Errorf("shortNode key=%x: %w", n.Key, err)
		}
		if childInfo == nil {
			return nil, nil
		}

		// Adjust relative paths in leaves to include this node's key
		for _, leaf := range childInfo.leaves {
			leaf.Path = append(append([]byte{}, n.Key...), leaf.Path...)
		}

		return &subtreeInfo{
			path:      copyBytes(path),
			owner:     owner,
			height:    childInfo.height + 1,
			leaves:    childInfo.leaves,
			nodePaths: append([][]byte{copyBytes(path)}, childInfo.nodePaths...),
		}, nil

	case *fullNode:
		var (
			maxHeight = 0
			allLeaves []*archive.Record
			allPaths  = [][]byte{copyBytes(path)}
		)
		for i, child := range n.Children[:16] {
			if child != nil {
				childPath := append(append([]byte{}, path...), byte(i))
				childInfo, err := a.computeSubtreeInfo(child, childPath, owner)
				if err != nil {
					return nil, fmt.Errorf("fullNode child[%x]: %w", i, err)
				}
				if childInfo != nil {
					if childInfo.height+1 > maxHeight {
						maxHeight = childInfo.height + 1
					}
					// Adjust relative paths to include the branch index
					for _, leaf := range childInfo.leaves {
						leaf.Path = append([]byte{byte(i)}, leaf.Path...)
					}
					allLeaves = append(allLeaves, childInfo.leaves...)
					allPaths = append(allPaths, childInfo.nodePaths...)
				}
			}
		}

		if len(allLeaves) == 0 {
			return nil, nil
		}

		return &subtreeInfo{
			path:      copyBytes(path),
			owner:     owner,
			height:    maxHeight,
			leaves:    allLeaves,
			nodePaths: allPaths,
		}, nil

	case hashNode:
		resolved, err := a.resolveNode(n, path, owner)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve hashNode at path %s: %w", common.Bytes2Hex(path), err)
		}
		return a.computeSubtreeInfo(resolved, path, owner)

	case *expiredNode:
		// Already archived, skip
		return nil, nil
	}
	return nil, nil
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

// resolveNode resolves a hashNode to its actual node content.
func (a *Archiver) resolveNode(hash hashNode, path []byte, owner common.Hash) (node, error) {
	reader, err := a.triedb.NodeReader(a.stateRoot)
	if err != nil {
		return nil, err
	}
	blob, err := reader.Node(owner, path, common.BytesToHash(hash))
	if err != nil {
		return nil, err
	}
	return decodeNodeUnsafe(hash, blob)
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
