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
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/archive"
)

// expiredNodeMarker is a special marker byte to identify expired nodes.
// Using 0x00 as a marker since valid MPT nodes are always RLP lists (starting with 0xc0+).
const expiredNodeMarker = 0x00

// expiredNode represents a node whose data has been archived. It stores the
// file offset and size of the archived record group, plus the hex-nibble
// path of this node RELATIVE to the group root (empty for the group root
// itself). Sub-markers with a non-empty subpath are produced by partial
// resurrection: they re-expire the sibling branches that a key access did
// not need, so untouched data stays archived across commits.
type expiredNode struct {
	offset          uint64
	size            uint64
	subpath         []byte // hex nibbles below the group root, no terminator
	cachedHash      hashNode
	archiveResolver archive.ResolverFn
}

func (n *expiredNode) cache() (hashNode, bool) {
	return n.cachedHash, n.cachedHash == nil
}

func (n *expiredNode) encode(w rlp.EncoderBuffer) {
	w.Write(encodeExpiredNodeBlob(n.offset, n.size, n.subpath))
}

func (n *expiredNode) fstring(ind string) string {
	return fmt.Sprintf("<expired: offset=%d, size=%d, subpath=%x> ", n.offset, n.size, n.subpath)
}

// Offset returns the archive file offset for this expired node.
func (n *expiredNode) Offset() uint64 {
	return n.offset
}

// SetArchiveResolver sets the resolver function for this expired node.
func (n *expiredNode) SetArchiveResolver(resolver archive.ResolverFn) {
	n.archiveResolver = resolver
}

// templateKey identifies archive content: the file offset of the record
// group plus the sub-path within it (the archive file is append-only, so
// this pair uniquely identifies the reconstructed content).
type templateKey struct {
	offset  uint64
	subpath string
}

// resolvedTemplates caches fully rebuilt, fully HASHED (and, when the
// expected hash is known, verified) subtrees keyed by archive location.
// Templates are immutable: resolutions hand out copies (or pruned copies),
// never the template itself. Because every template node carries its
// stamped hash, pruned copies can always represent off-branch siblings as
// sub-markers, and the committer's embedded-node invariant (hash==nil means
// smaller than 32 bytes) holds on every copy.
var resolvedTemplates = lru.NewCache[templateKey, node](4096)

// resolveExpiredTemplate returns the immutable, fully hashed template for
// the archive content referenced by n, building and verifying it on first
// use. The returned node MUST NOT be mutated.
func resolveExpiredTemplate(n *expiredNode) (node, error) {
	key := templateKey{offset: n.offset, subpath: string(n.subpath)}
	if tpl, ok := resolvedTemplates.Get(key); ok {
		// The cached template must carry exactly the hash the parent
		// references; without an expected hash the template cannot be
		// authenticated (and the archive file may have been swapped, as
		// tests do), so fall through to a fresh rebuild in that case or
		// on any mismatch.
		if hash, _ := tpl.cache(); n.cachedHash != nil && bytes.Equal(hash, n.cachedHash) {
			return tpl, nil
		}
	}
	start := time.Now()
	records, err := archive.ArchivedNodeResolver(n.offset, n.size)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve expired node: %w", err)
	}
	resolved, err := archiveRecordsToNode(records, n.subpath)
	if err != nil {
		return nil, fmt.Errorf("failed to rebuild expired node from archive: %w", err)
	}
	// Hash the whole reconstruction. This serves three purposes: it
	// verifies the archive content against the parent-referenced hash, it
	// stamps flags.hash on every large interior node (the invariant the
	// committer relies on to distinguish embedded nodes), and it provides
	// the sibling hashes needed to emit sub-markers when pruning.
	h := newHasher(false)
	gotHash := h.hash(resolved, true)
	returnHasherToPool(h)
	if n.cachedHash != nil && !bytes.Equal(gotHash, n.cachedHash) {
		return nil, fmt.Errorf("expired node hash mismatch at offset=%d size=%d subpath=%x: archive data is corrupted (expected %x got %x, %d records)",
			n.offset, n.size, n.subpath, []byte(n.cachedHash), gotHash, len(records))
	}
	// Stamp the computed hash onto the root: safe, since the walk above
	// hashed every interior node.
	switch nn := resolved.(type) {
	case *fullNode:
		nn.flags.hash = hashNode(gotHash)
	case *shortNode:
		nn.flags.hash = hashNode(gotHash)
	}
	log.Debug("Resurrected expired node from archive",
		"offset", n.offset, "archiveBytes", n.size, "subpath", n.subpath,
		"records", len(records),
		"elapsed", time.Since(start))
	resolvedTemplates.Add(key, resolved)
	return resolved, nil
}

// resolveExpiredNodeData resolves an expired node from the archive into a
// fully materialized, mutable subtree, verifying the reconstruction against
// the parent-referenced hash. The entire subtree is marked dirty so the
// committer captures it into the diff layer: resolved-but-unmodified nodes
// exist nowhere else (the archiver deleted them from the database), so
// dropping them from the NodeSet would produce MissingNodeError on later
// accesses through higher layers. For read-only tries the marking is
// harmless: the nodes are discarded when the trie is GC'd.
func resolveExpiredNodeData(n *expiredNode) (node, error) {
	tpl, err := resolveExpiredTemplate(n)
	if err != nil {
		return nil, err
	}
	resolved := copyNode(tpl)
	markSubtreeDirty(resolved)
	return resolved, nil
}

// resolveExpiredNodeForKey resolves an expired node like
// resolveExpiredNodeData, but keeps ONLY the branch leading to relKey
// materialized: every large off-branch sibling is emitted as an expired
// sub-marker referencing the same archive record group, so it stays
// archived across the following commit instead of being fully resurrected.
// relKey is the remaining hex-nibble key below this node (terminator
// allowed). A nil relKey degrades to full resolution.
//
// The result is built as a pruned COPY of the immutable template: only the
// branch spine and embedded (sub-32-byte) siblings are copied; large
// siblings become markers carrying the hash stamped on the template, so no
// per-resolution hashing or archive I/O happens on template hits.
func resolveExpiredNodeForKey(n *expiredNode, relKey []byte) (node, error) {
	if relKey == nil {
		return resolveExpiredNodeData(n)
	}
	tpl, err := resolveExpiredTemplate(n)
	if err != nil {
		return nil, err
	}
	pruned := pruneCopy(n, tpl, n.subpath, relKey)
	markSubtreeDirty(pruned)
	return pruned, nil
}

// pruneCopy builds a mutable copy of the template restricted to the branch
// following key. rel is the path of nd relative to the archive group root.
// Off-branch children with a stamped hash become sub-markers; children
// without one are embedded (<32 bytes) and are deep-copied, since markers
// must be referenced by hash.
func pruneCopy(n *expiredNode, nd node, rel []byte, key []byte) node {
	switch v := nd.(type) {
	case *fullNode:
		cp := &fullNode{flags: v.flags.copy()}
		if v.Children[16] != nil {
			cp.Children[16] = copyNode(v.Children[16])
		}
		for i, child := range v.Children[:16] {
			if child == nil {
				continue
			}
			if len(key) > 0 && key[0] == byte(i) {
				cp.Children[i] = pruneCopy(n, child, append(append([]byte{}, rel...), byte(i)), key[1:])
				continue
			}
			if hash, _ := child.cache(); hash != nil {
				cp.Children[i] = &expiredNode{
					offset:     n.offset,
					size:       n.size,
					subpath:    append(append([]byte{}, rel...), byte(i)),
					cachedHash: hash,
				}
				continue
			}
			cp.Children[i] = copyNode(child)
		}
		return cp
	case *shortNode:
		cp := &shortNode{flags: v.flags.copy(), Key: common.CopyBytes(v.Key)}
		if len(key) >= len(v.Key) && prefixLen(key, v.Key) == len(v.Key) {
			cp.Val = pruneCopy(n, v.Val, append(append([]byte{}, rel...), v.Key...), key[len(v.Key):])
		} else if hash, _ := v.Val.cache(); hash != nil {
			// Key diverges inside the extension: nothing below is needed.
			cp.Val = &expiredNode{
				offset:     n.offset,
				size:       n.size,
				subpath:    append(append([]byte{}, rel...), v.Key...),
				cachedHash: hash,
			}
		} else {
			cp.Val = copyNode(v.Val)
		}
		return cp
	default:
		return copyNode(nd)
	}
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

// archiveRecordsToNode rebuilds the trie node at the given sub-path of an
// archived record group. An empty subpath rebuilds the whole group. Records
// outside the sub-path are skipped; their content is represented by expired
// sub-markers in the live trie.
func archiveRecordsToNode(records []*archive.Record, subpath []byte) (node, error) {
	if len(records) == 0 {
		return nil, archive.EmptyArchiveRecord
	}

	// Build the trie incrementally from nil to produce the canonical
	// MPT structure. Starting with a fullNode would be wrong when the
	// original subtree root was a shortNode (shared prefix).
	var (
		root node
		kept int
	)
	for i, record := range records {
		if err := validateRecordPath(record.Path); err != nil {
			return nil, err
		}
		if len(subpath) > 0 {
			if len(record.Path) < len(subpath) || !bytes.Equal(record.Path[:len(subpath)], subpath) {
				continue
			}
		}
		key, err := normalizeRecordKey(record.Path[len(subpath):])
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
		kept++
	}
	if kept == 0 {
		return nil, fmt.Errorf("no record matches subpath %x in archive group", subpath)
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
