// Copyright 2025 go-ethereum Authors
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
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

type (
	NodeFlushFn    func([]byte, BinaryNode)
	NodeResolverFn func([]byte, common.Hash) ([]byte, error)
)

// zero is the zero value for a 32-byte array.
var zero [32]byte

const (
	StemNodeWidth = 256 // Number of child per leaf node
	StemSize      = 31  // Number of bytes to travel before reaching a group of leaves
	NodeTypeBytes = 1   // Size of node type prefix in serialization
	HashSize      = 32  // Size of a hash in bytes
	BitmapSize    = 32  // Size of the bitmap in a stem node

	// GroupDepth is the number of levels in a grouped subtree serialization.
	// Groups are byte-aligned (depth % 8 == 0). This may become configurable later.
	// Serialization format for InternalNode groups:
	//   [1 byte type] [1 byte group depth (1-8)] [32 byte bitmap] [N × 32 byte hashes]
	// The bitmap has 2^groupDepth bits, indicating which bottom-layer children are present.
	// Only present children's hashes are stored, in order.
	GroupDepth = 8
)

const (
	nodeTypeStem = iota + 1 // Stem node, contains a stem and a bitmap of values
	nodeTypeInternal
)

// BinaryNode is an interface for a binary trie node.
type BinaryNode interface {
	Get([]byte, NodeResolverFn) ([]byte, error)
	Insert([]byte, []byte, NodeResolverFn, int) (BinaryNode, error)
	Copy() BinaryNode
	Hash() common.Hash
	GetValuesAtStem([]byte, NodeResolverFn) ([][]byte, error)
	InsertValuesAtStem([]byte, [][]byte, NodeResolverFn, int) (BinaryNode, error)
	CollectNodes([]byte, NodeFlushFn) error

	toDot(parent, path string) string
	GetHeight() int
}

// serializeSubtree recursively collects child hashes from a subtree of InternalNodes.
// It traverses up to `remainingDepth` levels, storing hashes of bottom-layer children.
// position tracks the current index (0 to 2^groupDepth - 1) for bitmap placement.
// hashes collects the hashes of present children, bitmap tracks which positions are present.
func serializeSubtree(node BinaryNode, remainingDepth int, position int, bitmap []byte, hashes *[]common.Hash) {
	if remainingDepth == 0 {
		// Bottom layer: store hash if not empty
		switch node.(type) {
		case Empty:
			// Leave bitmap bit unset, don't add hash
			return
		default:
			// StemNode, HashedNode, or InternalNode at boundary: store hash
			bitmap[position/8] |= 1 << (7 - (position % 8))
			*hashes = append(*hashes, node.Hash())
		}
		return
	}

	switch n := node.(type) {
	case *InternalNode:
		// Recurse into left (bit 0) and right (bit 1) children
		leftPos := position * 2
		rightPos := position*2 + 1
		serializeSubtree(n.left, remainingDepth-1, leftPos, bitmap, hashes)
		serializeSubtree(n.right, remainingDepth-1, rightPos, bitmap, hashes)
	case Empty:
		// Empty subtree: all positions in this subtree are empty (bits already 0)
		return
	default:
		// StemNode or HashedNode before reaching bottom: store hash at current position
		// This creates a variable-depth group where this branch terminates early.
		// We need to mark this single position and all its would-be descendants as "this hash".
		// For simplicity, we store the hash at the first leaf position of this subtree.
		firstLeafPos := position << remainingDepth
		bitmap[firstLeafPos/8] |= 1 << (7 - (firstLeafPos % 8))
		*hashes = append(*hashes, node.Hash())
	}
}

// SerializeNode serializes a binary trie node into a byte slice.
func SerializeNode(node BinaryNode) []byte {
	switch n := (node).(type) {
	case *InternalNode:
		// InternalNode group: 1 byte type + 1 byte group depth + 32 byte bitmap + N×32 byte hashes
		groupDepth := GroupDepth

		var bitmap [BitmapSize]byte
		var hashes []common.Hash

		serializeSubtree(n, groupDepth, 0, bitmap[:], &hashes)

		// Build serialized output
		serializedLen := NodeTypeBytes + 1 + BitmapSize + len(hashes)*HashSize
		serialized := make([]byte, serializedLen)
		serialized[0] = nodeTypeInternal
		serialized[1] = byte(groupDepth)
		copy(serialized[2:2+BitmapSize], bitmap[:])

		offset := NodeTypeBytes + 1 + BitmapSize
		for _, h := range hashes {
			copy(serialized[offset:offset+HashSize], h.Bytes())
			offset += HashSize
		}

		return serialized
	case *StemNode:
		// StemNode: 1 byte type + 31 bytes stem + 32 bytes bitmap + 256*32 bytes values
		var serialized [NodeTypeBytes + StemSize + BitmapSize + StemNodeWidth*HashSize]byte
		serialized[0] = nodeTypeStem
		copy(serialized[NodeTypeBytes:NodeTypeBytes+StemSize], n.Stem)
		bitmap := serialized[NodeTypeBytes+StemSize : NodeTypeBytes+StemSize+BitmapSize]
		offset := NodeTypeBytes + StemSize + BitmapSize
		for i, v := range n.Values {
			if v != nil {
				bitmap[i/8] |= 1 << (7 - (i % 8))
				copy(serialized[offset:offset+HashSize], v)
				offset += HashSize
			}
		}
		// Only return the actual data, not the entire array
		return serialized[:offset]
	default:
		panic("invalid node type")
	}
}

var invalidSerializedLength = errors.New("invalid serialized node length")

// deserializeSubtree reconstructs an InternalNode subtree from grouped serialization.
// remainingDepth is how many more levels to build, position is current index in the bitmap,
// nodeDepth is the actual trie depth for the node being created.
// hashIdx tracks the current position in the hash data (incremented as hashes are consumed).
func deserializeSubtree(remainingDepth int, position int, nodeDepth int, bitmap []byte, hashData []byte, hashIdx *int) (BinaryNode, error) {
	if remainingDepth == 0 {
		// Bottom layer: check bitmap and return HashedNode or Empty
		if bitmap[position/8]>>(7-(position%8))&1 == 1 {
			if len(hashData) < (*hashIdx+1)*HashSize {
				return nil, invalidSerializedLength
			}
			hash := common.BytesToHash(hashData[*hashIdx*HashSize : (*hashIdx+1)*HashSize])
			*hashIdx++
			return HashedNode(hash), nil
		}
		return Empty{}, nil
	}

	// Check if this entire subtree is empty by examining all relevant bitmap bits
	leftPos := position * 2
	rightPos := position*2 + 1

	left, err := deserializeSubtree(remainingDepth-1, leftPos, nodeDepth+1, bitmap, hashData, hashIdx)
	if err != nil {
		return nil, err
	}
	right, err := deserializeSubtree(remainingDepth-1, rightPos, nodeDepth+1, bitmap, hashData, hashIdx)
	if err != nil {
		return nil, err
	}

	// If both children are empty, return Empty
	_, leftEmpty := left.(Empty)
	_, rightEmpty := right.(Empty)
	if leftEmpty && rightEmpty {
		return Empty{}, nil
	}

	return &InternalNode{
		depth: nodeDepth,
		left:  left,
		right: right,
	}, nil
}

// DeserializeNode deserializes a binary trie node from a byte slice.
func DeserializeNode(serialized []byte, depth int) (BinaryNode, error) {
	if len(serialized) == 0 {
		return Empty{}, nil
	}

	switch serialized[0] {
	case nodeTypeInternal:
		// Grouped format: 1 byte type + 1 byte group depth + 32 byte bitmap + N×32 byte hashes
		if len(serialized) < NodeTypeBytes+1+BitmapSize {
			return nil, invalidSerializedLength
		}
		groupDepth := int(serialized[1])
		if groupDepth < 1 || groupDepth > GroupDepth {
			return nil, errors.New("invalid group depth")
		}
		bitmap := serialized[2 : 2+BitmapSize]
		hashData := serialized[2+BitmapSize:]

		// Count present children from bitmap
		hashIdx := 0
		return deserializeSubtree(groupDepth, 0, depth, bitmap, hashData, &hashIdx)
	case nodeTypeStem:
		if len(serialized) < 64 {
			return nil, invalidSerializedLength
		}
		var values [StemNodeWidth][]byte
		bitmap := serialized[NodeTypeBytes+StemSize : NodeTypeBytes+StemSize+BitmapSize]
		offset := NodeTypeBytes + StemSize + BitmapSize

		for i := range StemNodeWidth {
			if bitmap[i/8]>>(7-(i%8))&1 == 1 {
				if len(serialized) < offset+HashSize {
					return nil, invalidSerializedLength
				}
				values[i] = serialized[offset : offset+HashSize]
				offset += HashSize
			}
		}
		return &StemNode{
			Stem:   serialized[NodeTypeBytes : NodeTypeBytes+StemSize],
			Values: values[:],
			depth:  depth,
		}, nil
	default:
		return nil, errors.New("invalid node type")
	}
}

// ToDot converts the binary trie to a DOT language representation. Useful for debugging.
func ToDot(root BinaryNode) string {
	return root.toDot("", "")
}
