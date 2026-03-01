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
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/archive"
)

// expiredNodeMarker is a special marker byte to identify expired nodes.
// Using 0x00 as a marker since valid MPT nodes are always RLP lists (starting with 0xc0+).
const expiredNodeMarker = 0x00

// expiredNode represents a node whose data has been archived.
// It stores the file offset and size of the archived data.
type expiredNode struct {
	offset          uint64
	size            uint64
	cachedHash      hashNode
	archiveResolver archive.ResolverFn
}

func (n *expiredNode) cache() (hashNode, bool) {
	return n.cachedHash, n.cachedHash == nil
}

func (n *expiredNode) encode(w rlp.EncoderBuffer) {
	var buf [1 + 2*archive.OffsetSize]byte
	buf[0] = expiredNodeMarker
	binary.BigEndian.PutUint64(buf[1:], n.offset)
	binary.BigEndian.PutUint64(buf[1+archive.OffsetSize:], n.size)
	w.Write(buf[:])
}

func (n *expiredNode) fstring(ind string) string {
	return fmt.Sprintf("<expired: offset=%d, size=%d> ", n.offset, n.size)
}

// Offset returns the archive file offset for this expired node.
func (n *expiredNode) Offset() uint64 {
	return n.offset
}

// SetArchiveResolver sets the resolver function for this expired node.
func (n *expiredNode) SetArchiveResolver(resolver archive.ResolverFn) {
	n.archiveResolver = resolver
}

// resolveExpiredNodeData resolves an expired node from the archive, verifies
// the reconstructed subtree hash, and stamps the cached hash onto the root.
// Returns an error if the archive data is corrupted (hash mismatch).
func resolveExpiredNodeData(n *expiredNode) (node, error) {
	records, err := archive.ArchivedNodeResolver(n.offset, n.size)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve expired node: %w", err)
	}
	resolved, err := archiveRecordsToNode(records)
	if err != nil {
		return nil, fmt.Errorf("failed to rebuild expired node from archive: %w", err)
	}
	// Verify hash integrity: if the original hash is known, check that the
	// reconstructed subtree produces the same hash. A mismatch means the
	// archive is corrupted (e.g. missing leaves due to unresolvable hashNodes
	// during archival) and any data from it is unreliable.
	if n.cachedHash != nil {
		h := newHasher(false)
		gotHash := h.hash(resolved, true)
		returnHasherToPool(h)
		if !bytes.Equal(gotHash, n.cachedHash) {
			return nil, fmt.Errorf("expired node hash mismatch at offset=%d size=%d: archive data is corrupted (expected %x got %x, %d records)",
				n.offset, n.size, []byte(n.cachedHash), gotHash, len(records))
		}
		// Stamp the original hash onto the resolved subtree root so the
		// hasher returns it directly instead of re-computing.
		switch nn := resolved.(type) {
		case *fullNode:
			nn.flags.hash = n.cachedHash
		case *shortNode:
			nn.flags.hash = n.cachedHash
		}
	}
	// Mark the entire resolved subtree as dirty. This is critical for
	// correctness with pathdb's diff layer model: when a trie with expired
	// nodes is modified and committed, the committer only captures dirty
	// nodes into the NodeSet (which becomes the diff layer). Without this
	// marking, resolved-but-unmodified sibling nodes within the subtree
	// would exist nowhere — not in any diff layer (they're clean) and not
	// in the raw DB (the archiver deleted them). Subsequent trie accesses
	// from higher diff layers would fall through to the disk layer, find
	// nothing, and produce MissingNodeError.
	//
	// For read-only tries (only get operations, no commit), this dirty
	// marking is harmless — the nodes are discarded when the trie is GC'd.
	markSubtreeDirty(resolved)
	return resolved, nil
}

// markSubtreeDirty recursively marks all fullNode and shortNode in the
// subtree as dirty, preserving any cached hashes. This ensures the
// committer will capture them in the NodeSet during trie commit.
func markSubtreeDirty(n node) {
	switch n := n.(type) {
	case *fullNode:
		n.flags.dirty = true
		for _, child := range n.Children[:16] {
			if child != nil {
				markSubtreeDirty(child)
			}
		}
	case *shortNode:
		n.flags.dirty = true
		markSubtreeDirty(n.Val)
	}
	// valueNode, hashNode, nil: no flags to mark
}

func archiveRecordsToNode(records []*archive.Record) (node, error) {
	if len(records) == 0 {
		return nil, archive.EmptyArchiveRecord
	}

	// Build the trie incrementally from nil to produce the canonical
	// MPT structure. Starting with a fullNode would be wrong when the
	// original subtree root was a shortNode (shared prefix).
	var root node
	for i, record := range records {
		if err := validateRecordPath(record.Path); err != nil {
			return nil, err
		}

		key, err := normalizeRecordKey(record.Path)
		if err != nil {
			return nil, err
		}
		if len(key) < 1 {
			return nil, fmt.Errorf("empty key in record #%d", i)
		}
		root, err = insertTrieNode(root, key, valueNode(record.Value))
		if err != nil {
			return nil, err
		}
	}
	return root, nil
}

func validateRecordPath(path []byte) error {
	for i, b := range path {
		if b > 16 {
			return fmt.Errorf("invalid nibble in record path: %d", b)
		}
		if b == 16 && i != len(path)-1 {
			return fmt.Errorf("terminator nibble in middle of record path")
		}
	}
	return nil
}

// normalizeRecordKey ensures the record path is a hex-nibble key suitable for
// leaf insertion by guaranteeing a single terminator nibble and preserving any
// already-terminated path. Empty paths are normalized to a sole terminator.
func normalizeRecordKey(path []byte) ([]byte, error) {
	if len(path) == 0 {
		return []byte{16}, nil
	}
	if hasTerm(path) {
		return path, nil
	}
	key := append([]byte{}, path...)
	key = append(key, 16)
	return key, nil
}

func insertTrieNode(n node, key []byte, value node) (node, error) {
	if len(key) == 0 {
		return value, nil
	}
	switch n := n.(type) {
	case *shortNode:
		matchlen := prefixLen(key, n.Key)
		if matchlen == len(n.Key) {
			nn, err := insertTrieNode(n.Val, key[matchlen:], value)
			if err != nil {
				return nil, err
			}
			return &shortNode{Key: n.Key, Val: nn}, nil
		}
		branch := &fullNode{}
		var err error
		branch.Children[n.Key[matchlen]], err = insertTrieNode(nil, n.Key[matchlen+1:], n.Val)
		if err != nil {
			return nil, err
		}
		branch.Children[key[matchlen]], err = insertTrieNode(nil, key[matchlen+1:], value)
		if err != nil {
			return nil, err
		}
		if matchlen == 0 {
			return branch, nil
		}
		return &shortNode{Key: key[:matchlen], Val: branch}, nil

	case *fullNode:
		child, err := insertTrieNode(n.Children[key[0]], key[1:], value)
		if err != nil {
			return nil, err
		}
		n.Children[key[0]] = child
		return n, nil

	case nil:
		return &shortNode{Key: key, Val: value}, nil

	default:
		return nil, fmt.Errorf("invalid node type in trie insert: %T", n)
	}
}
