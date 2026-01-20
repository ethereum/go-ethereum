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
		return decodeNodeUnsafe(nil, records[0].Value)
	}

	var (
		newnode fullNode
		curnode *fullNode
	)
	for _, record := range records {
		curnode = &newnode
		resolved, err := decodeNodeUnsafe(nil, record.Value)
		if err != nil {
			return nil, err
		}
		// It's not needed to resurrect all nodes, nodes
		// not along the path of what has been asked can
		// be updated as expired. This is for v2.
		for i, b := range record.Path {
			if curnode.Children[b] == nil {
				if i < len(record.Path)-1 {
					curnode.Children[b] = &fullNode{}
				} else {
					curnode.Children[b] = resolved
				}
			}
		}
	}
	return &newnode, nil
}
