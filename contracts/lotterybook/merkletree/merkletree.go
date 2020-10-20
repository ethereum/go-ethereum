// Copyright 2019 The go-ethereum Authors
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

// merkletree package implements a merkle tree as the probability tree.
// The basic idea is different entries referenced by this tree has different
// position. The position of the tree node can be used as the probability range
// of the node.
//
// All entries will have an initial weight, which represents the probability that
// this node will be picked. Because the merkletree implemented in this package is
// a binary tree, so the final weight of each entry will be adjusted to 1/2^N format.
//
// To simplify the verification process of merkle proof, the hash value calculation
// process of the parent node, the left subtree hash value is smaller than the right
// subtree hash value. So that we can get rid of building instructions when we try
// to rebuild the tree based on the proof.
package merkletree

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

var (
	// maxLevel indicates the deepest Level the node can be. It means
	// the minimal weight supported is 1/1024. If the assigned initial
	// weight is too small, it will be assigned to the minimal weight
	// or zero. If the weight is zero, it means the entry is not included.
	maxLevel = 10

	// maxWeight indicates the denominator used to calculate weight.
	maxWeight = uint64(1) << 63
)

var (
	// ErrUnknownEntry is returned if caller wants to prove an non-existent entry.
	ErrUnknownEntry = errors.New("the entry is non-existent requested for proof")

	// ErrInvalidEntry is returned if caller wants to prove an invalid entry.
	ErrInvalidEntry = errors.New("the entry is invalid requested for proof")

	// ErrInvalidProof is returned if the provided merkle proof to verify is invalid.
	ErrInvalidProof = errors.New("invalid merkle proof")
)

// Entry represents the data entry referenced by the merkle tree.
type Entry struct {
	Value  []byte // The corresponding value of this entry
	Weight uint64 // The initial weight specified by caller

	// Internal fields
	level uint64  // The level of node which references this entry in the tree
	bias  float64 // The bias between initial weight and the assigned weight
	salt  uint64  // A random value used as the input for hash calculation
}

// Hash return the hash of the entry
func (s *Entry) Hash() common.Hash {
	var buff [8]byte
	binary.BigEndian.PutUint64(buff[:], s.salt)
	return crypto.Keccak256Hash(append(s.Value, buff[:]...))
}

// Salt returns the random number used for hash calculation.
func (s *Entry) Salt() uint64 {
	return s.salt
}

// Level returns the level of entry.
func (s *Entry) Level() uint64 {
	return s.level
}

// EntryByBias implements the sort interface to allow sorting a list of entries
// by their weight bias.
type EntryByBias []*Entry

func (s EntryByBias) Len() int           { return len(s) }
func (s EntryByBias) Less(i, j int) bool { return s[i].bias < s[j].bias }
func (s EntryByBias) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// EntryByLevel implements the sort interface to allow sorting a list of entries
// by their position in the tree in descending order.
type EntryByLevel []*Entry

func (s EntryByLevel) Len() int           { return len(s) }
func (s EntryByLevel) Less(i, j int) bool { return s[i].level > s[j].level }
func (s EntryByLevel) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// Node represents a node in merkle tree.
type Node struct {
	Nodehash common.Hash // The hash of node.
	Parent   *Node       // The parent of this node, nil if it's root node.
	Left     *Node       // The left child of this node
	Right    *Node       // The right child of this node
	Level    uint64      // The level of node in this tree
	Value    *Entry      // The referenced entry by this node, nil if it's not leaf.
}

// Hash returns the hash of this tree node.
func (node *Node) Hash() common.Hash {
	// Short circuit if nodehash is already cached.
	if node.Nodehash != (common.Hash{}) {
		return node.Nodehash
	}
	// If it's a leaf node, derive the hash by the entry content.
	if node.Value != nil {
		node.Nodehash = node.Value.Hash()
		return node.Nodehash
	}
	// It's a branch node, derive the hash via two children.
	left, right := node.Left.Hash(), node.Right.Hash() // Both children should never be nil.
	if bytes.Compare(left.Bytes(), right.Bytes()) < 0 {
		node.Nodehash = crypto.Keccak256Hash(append(left.Bytes(), right.Bytes()...))
	} else {
		node.Nodehash = crypto.Keccak256Hash(append(right.Bytes(), left.Bytes()...))
	}
	return node.Nodehash
}

// String returns the string format of node.
func (node *Node) String() string {
	if node.Value != nil {
		return fmt.Sprintf("E(%x:%d)", node.Value.Value, node.Value.level)
	}
	return fmt.Sprintf("N(%x) => L.(%s) R.(%s)", node.Hash(), node.Left.String(), node.Right.String())
}

type MerkleTree struct {
	Roothash common.Hash // The hash of root node, maybe null if we never calculate it.
	Root     *Node       // The root node of merkle tree.
	Leaves   []*Node     // Batch of leaves node included in the tree.
}

// NewMerkleTree constructs a merkle tree with given entries.
// If there is no entry given, an empty tree is returned.
func NewMerkleTree(entries []*Entry) (*MerkleTree, map[string]struct{}) {
	if len(entries) == 0 {
		return nil, nil
	}
	// Assign an unique salt for each entry. The hash
	// of entry is calculated by keccak256(value, salt)
	// so we can preserve the privacy of given value.
	for _, entry := range entries {
		entry.salt = rand.Uint64()
	}
	var sum, totalWeight uint64
	for _, entry := range entries {
		sum += entry.Weight
	}
	for _, entry := range entries {
		// If the initial weight is 0, set it maxLevel+1 temporarily.
		// Will try to allocate some weight if there is some free space.
		if entry.Weight == 0 {
			entry.bias = 0
			entry.level = uint64(maxLevel + 1)
			continue
		}
		// Calculate the node level in the tree based on the proportion.
		l := math.Log2(float64(sum) / float64(entry.Weight))
		c := math.Ceil(l)
		if int(c) > maxLevel {
			entry.bias = 0
			entry.level = uint64(maxLevel + 1)
			continue
		}
		entry.bias = l - c + 1
		entry.level = uint64(c)
		totalWeight += maxWeight >> entry.level
	}
	sort.Sort(EntryByBias(entries))

	// Bump the weight of entry if we can't reach 100%
	shift := entries
	for totalWeight < maxWeight && len(shift) > 0 {
		var limit int
		for index, entry := range shift {
			var addWeight uint64
			if entry.level <= uint64(maxLevel) {
				addWeight = maxWeight >> entry.level
			} else {
				addWeight = maxWeight >> uint64(maxLevel)
			}
			if totalWeight+addWeight <= maxWeight {
				totalWeight += addWeight
				entry.level -= 1
				if index != limit {
					shift[limit], shift[index] = shift[index], shift[limit]
				}
				limit += 1
				if totalWeight == maxWeight {
					break
				}
			}
		}
		shift = shift[:limit]
	}
	sort.Sort(EntryByLevel(entries))

	dropped := make(map[string]struct{})
	for len(entries) > 0 && entries[0].level > uint64(maxLevel) {
		dropped[string(entries[0].Value)] = struct{}{}
		entries = entries[1:]
	}
	// Start to build the merkle tree, short circuit if there is only 1 entry.
	root, leaves := newTree(entries)
	return &MerkleTree{Root: root, Leaves: leaves}, dropped
}

func newTree(entries []*Entry) (*Node, []*Node) {
	// Short circuit if we only have 1 entry, return it as the root node
	// of sub tree.
	if len(entries) == 1 {
		n := &Node{Value: entries[0], Level: 0}
		return n, []*Node{n}
	}
	var current *Node
	var leaves []*Node
	for i := 0; i < len(entries); {
		// Because all nodes are sorted in descending order of level,
		// So the level of first two nodes must be same and can be
		// grouped as a sub tree.
		if i == 0 {
			if entries[0].level != entries[1].level {
				panic("invalid level in same group") // Should never happen
			}
			n1, n2 := &Node{Value: entries[0], Level: entries[0].level}, &Node{Value: entries[1], Level: entries[1].level}
			current = &Node{Left: n1, Right: n2, Level: entries[0].level - 1}
			n1.Parent, n2.Parent = current, current
			i += 2
			leaves = append(leaves, n1, n2)
			continue
		}
		switch {
		case current.Level > entries[i].level:
			panic("invalid levels") // Should never happen
		case current.Level == entries[i].level:
			n := &Node{Value: entries[i], Level: entries[i].level}
			tmp := &Node{Left: current, Right: n, Level: current.Level - 1}
			current.Parent, n.Parent = tmp, tmp
			current = tmp
			leaves = append(leaves, n)
			i += 1
		default:
			var j int
			var weight uint64
			for j = i; j < len(entries); j++ {
				weight += maxWeight >> entries[j].level
				if weight == maxWeight>>current.Level {
					break
				}
			}
			right, subLeaves := newTree(entries[i : j+1])

			tmp := &Node{Left: current, Right: right, Level: current.Level - 1}
			current.Parent, right.Parent = tmp, tmp
			current = tmp
			leaves = append(leaves, subLeaves...)
			i += len(subLeaves)
		}
	}
	return current, leaves
}

// Hash calculates the root hash of merkle tree.
func (t *MerkleTree) Hash() common.Hash {
	return t.Root.Hash()
}

// Prove constructs a merkle proof for the specified entry.
func (t *MerkleTree) Prove(e *Entry) ([]common.Hash, error) {
	var n *Node
	for _, leaf := range t.Leaves {
		if bytes.Equal(leaf.Value.Value, e.Value) {
			// Ensure the salt is match.
			if leaf.Value.salt != e.salt {
				return nil, ErrInvalidEntry
			}
			n = leaf
			break
		}
	}
	if n == nil {
		return nil, ErrUnknownEntry
	}
	var hashes []common.Hash
	hashes = append(hashes, n.Hash())
	for {
		if n.Parent == nil {
			break
		}
		if n.Parent.Left == n {
			hashes = append(hashes, n.Parent.Right.Hash())
		} else {
			hashes = append(hashes, n.Parent.Left.Hash())
		}
		n = n.Parent
	}
	return hashes, nil
}

// VerifyProof verifies the provided merkle proof is valid or not.
//
// Except returning the error indicates whether the proof is valid,
// this function will also return the "position" of entry which is
// proven.
//
// The merkle tree looks like:
//
//            e2     e3
//             \     /
//              \   /
//               \ /
//        e1     h2
//         \     /
//          \   /
//           \ /
//           h1     e4
//            \     /
//             \   /
//              \ /
//           root hash
//
// The position of the nodes is essentially is the path from root
// node to target node. Like the position of e2 is 010 => 2, while
// for e3 the position is 011 => 3. Combine with the level node is
// in, we can calculate the probability range represented by this entry.
func VerifyProof(root common.Hash, proof []common.Hash) (uint64, error) {
	if len(proof) == 0 {
		return 0, ErrInvalidProof
	}
	if len(proof) == 1 {
		if root == proof[0] {
			return 0, nil
		}
		return 0, ErrInvalidProof
	}
	var (
		current = proof[0]
		pos     uint64
	)
	for i := 1; i < len(proof); i += 1 {
		if bytes.Compare(current.Bytes(), proof[i].Bytes()) < 0 {
			current = crypto.Keccak256Hash(append(current.Bytes(), proof[i].Bytes()...))
		} else {
			pos = pos + 1<<(i-1)
			current = crypto.Keccak256Hash(append(proof[i].Bytes(), current.Bytes()...))
		}
	}
	if root != current {
		return 0, ErrInvalidProof
	}
	return pos, nil
}

// String returns the string format of tree which helps to debug.
func (t *MerkleTree) String() string {
	return t.Root.String()
}
