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
	archiveResolver archive.ResolverFn
}

func (n *expiredNode) cache() (hashNode, bool) {
	return nil, true
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

func archiveRecordsToNode(records []*archive.Record) (node, error) {
	if len(records) == 0 {
		return nil, archive.EmptyArchiveRecord
	}
	if len(records) == 1 {
		return buildLeafFromRecord(records[0])
	}

	var newnode fullNode
	for i, record := range records {
		if err := validateRecordPath(record.Path); err != nil {
			return nil, err
		}

		// we are not in the case of a single leaf node, so each
		// path should be at least 2 nibbles (terminator included)
		if len(record.Path) < 2 || !hasTerm(record.Path) {
			return nil, fmt.Errorf("invalid record path for non-leaf node #%d: %v", i, record.Path)
		}
		key, err := normalizeRecordKey(record.Path)
		if err != nil {
			return nil, err
		}
		child, err := insertTrieNode(newnode.Children[key[0]], key[1:], valueNode(record.Value))
		if err != nil {
			return nil, err
		}
		newnode.Children[key[0]] = child
	}
	return &newnode, nil
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

func buildLeafFromRecord(record *archive.Record) (node, error) {
	key, err := normalizeRecordKey(record.Path)
	if err != nil {
		return nil, err
	}
	return &shortNode{Key: key, Val: valueNode(record.Value)}, nil
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
