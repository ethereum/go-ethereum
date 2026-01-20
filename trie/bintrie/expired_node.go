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

package bintrie

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie/archive"
)

// expiredNode represents a node whose data has been archived.
// It stores the file offset and size of the archived subtree data.
type expiredNode struct {
	Offset          uint64
	Size            uint64
	depth           int
	archiveResolver archive.ResolverFn
}

func archiveRecordsToNode(records []*archive.Record, depth int) (BinaryNode, error) {
	if len(records) == 0 {
		return nil, archive.EmptyArchiveRecord
	}
	if len(records) == 1 {
		return DeserializeNode(records[0].Value, depth)
	}

	var (
		newnode InternalNode
		curnode *InternalNode
	)
	for _, record := range records {
		curnode = &newnode
		resolved, err := DeserializeNode(record.Value, depth)
		if err != nil {
			return nil, err
		}
		// It's not needed to resurrect all nodes, nodes
		// not along the path of what has been asked can
		// be updated as expired. This is for v2.
		for i, b := range record.Path {
			var child BinaryNode
			if b == 0 {
				child = curnode.left
			} else {
				child = curnode.right
			}
			if child == nil {
				if i < len(record.Path)-1 {
					child = &InternalNode{depth: depth}
				} else {
					// Not good, I need to update the pointer
					child = resolved
				}
			}
			depth++
		}
	}
	return &newnode, nil
}

func (n *expiredNode) Get(key []byte, resolver NodeResolverFn) ([]byte, error) {
	if n.archiveResolver == nil {
		return nil, archive.ErrNoResolver
	}
	records, err := n.archiveResolver(n.Offset, n.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve expired node: %w", err)
	}

	resolved, err := archiveRecordsToNode(records, n.depth)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize expired node: %w", err)
	}
	return resolved.Get(key, resolver)
}

func (n *expiredNode) Insert(key, value []byte, resolver NodeResolverFn, depth int) (BinaryNode, error) {
	if n.archiveResolver == nil {
		return nil, archive.ErrNoResolver
	}
	blob, err := n.archiveResolver(n.Offset, n.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve expired node: %w", err)
	}
	resolved, err := archiveRecordsToNode(blob, n.depth)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize expired node: %w", err)
	}
	return resolved.Insert(key, value, resolver, depth)
}

func (n *expiredNode) Copy() BinaryNode {
	return &expiredNode{
		Offset:          n.Offset,
		Size:            n.Size,
		depth:           n.depth,
		archiveResolver: n.archiveResolver,
	}
}

func (n *expiredNode) Hash() common.Hash {
	return common.Hash{}
}

func (n *expiredNode) GetValuesAtStem(stem []byte, resolver NodeResolverFn) ([][]byte, error) {
	if n.archiveResolver == nil {
		return nil, archive.ErrNoResolver
	}
	blob, err := n.archiveResolver(n.Offset, n.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve expired node: %w", err)
	}
	resolved, err := archiveRecordsToNode(blob, n.depth)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize expired node: %w", err)
	}
	return resolved.GetValuesAtStem(stem, resolver)
}

func (n *expiredNode) InsertValuesAtStem(stem []byte, values [][]byte, resolver NodeResolverFn, depth int) (BinaryNode, error) {
	if n.archiveResolver == nil {
		return nil, archive.ErrNoResolver
	}
	blob, err := n.archiveResolver(n.Offset, n.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve expired node: %w", err)
	}
	resolved, err := archiveRecordsToNode(blob, n.depth)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize expired node: %w", err)
	}
	return resolved.InsertValuesAtStem(stem, values, resolver, depth)
}

func (n *expiredNode) CollectNodes(path []byte, flushfn NodeFlushFn) error {
	return nil
}

func (n *expiredNode) toDot(parent, path string) string {
	me := fmt.Sprintf("expired%s", path)
	ret := fmt.Sprintf("%s [label=\"EXPIRED: offset=%d\"]\n", me, n.Offset)
	if len(parent) > 0 {
		ret = fmt.Sprintf("%s %s -> %s\n", ret, parent, me)
	}
	return ret
}

func (n *expiredNode) GetHeight() int {
	return 0
}

// SetArchiveResolver sets the resolver function for this expired node.
func (n *expiredNode) SetArchiveResolver(resolver archive.ResolverFn) {
	n.archiveResolver = resolver
}

// Depth returns the depth of this node in the trie.
func (n *expiredNode) Depth() int {
	return n.depth
}
