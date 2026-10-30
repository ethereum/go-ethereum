// Copyright 2026 The go-ethereum Authors
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

package ssz

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Single-leaf Merkle proofs: a leaf, its generalized index, and the sibling
// of every node on its path to the root, bottom-up, the leaf's own sibling
// first. That is the branch of build_proof in ethereum/ssz-specs
// src/ssz/proofs.py and the input of beacon/merkle.VerifyProof, which walks
// the index bits from the leaf upward and insists on one sibling per level;
// every proof built here verifies there unchanged.
//
// Proofs are built the way roots are hashed, by hand per type from the
// primitives here, and composed across nesting with Wrap, MixLength and
// MixActiveFields. There is no tree: every sibling is the root of a run of
// leaves and is merkleized when asked for, so a proof costs one hashing pass,
// the same as the root it belongs to.

// ErrNoNode is returned when a proof is requested for a position the value's
// tree has no node at: a chunk beyond the padded width of a bounded tree, or
// a chunk in a level a progressive tree has not opened. Padding is not
// absence: a bounded tree pads to the power of two above its limit and an
// open progressive level to its subtree, and those zero leaves are provable,
// which is how a verifier shows a slot is empty. Whether a type declares the
// position at all is the caller's question, not the tree's.
var ErrNoNode = errors.New("no node at this position")

// Proof is a single-leaf Merkle proof. Gindex is measured from the root the
// proof verifies against; Branch holds GindexDepth(Gindex) siblings,
// bottom-up.
type Proof struct {
	Gindex uint64     // generalized index of the leaf
	Leaf   [32]byte   // the leaf value
	Branch [][32]byte // sibling hashes, the leaf's own sibling first
}

// Wrap composes p, a proof measured from the root of a nested value, with
// outer, the proof of that root inside the enclosing value, into the proof of
// p's leaf against the enclosing root: the indices concatenate and the
// branches follow, inner siblings first since they are nearer the leaf.
// outer.Leaf must be the root p rebuilds; that is not checked here, the
// round trip through the verifier is what catches a mismatch.
func (p Proof) Wrap(outer Proof) Proof {
	branch := make([][32]byte, 0, len(p.Branch)+len(outer.Branch))
	branch = append(branch, p.Branch...)
	branch = append(branch, outer.Branch...)
	return Proof{
		Gindex: ConcatGindex(outer.Gindex, p.Gindex),
		Leaf:   p.Leaf,
		Branch: branch,
	}
}

// MixLength lifts a proof measured from a list's data tree through the
// length mix-in to the list's hash tree root: the data tree is the left
// child, 2, and the length word its sibling.
func (p Proof) MixLength(length uint64) Proof {
	var word [32]byte
	binary.LittleEndian.PutUint64(word[:8], length)
	return p.Wrap(Proof{Gindex: 2, Branch: [][32]byte{word}})
}

// MixActiveFields lifts a proof measured from a progressive container's field
// tree through the active_fields mix-in to the container's hash tree root.
func (p Proof) MixActiveFields(af ActiveFields) Proof {
	if af.width == 0 {
		panic("ssz: mixing in an uninitialized ActiveFields")
	}
	return p.Wrap(Proof{Gindex: 2, Branch: [][32]byte{af.bits}})
}

// chunkAt returns chunk index, or the zero chunk past the end, which is what
// a padded tree holds there.
func chunkAt(chunks [][32]byte, index uint64) [32]byte {
	if index < uint64(len(chunks)) {
		return chunks[index]
	}
	return [32]byte{}
}

// rangeRoot returns the root of the subtree of span leaves, a power of two,
// whose first leaf is start: the chunks that fall in the run, merkleized to
// the span, which is the zero subtree when none do. Every node of a bounded
// tree is such a root, so a branch needs nothing else.
func rangeRoot(chunks [][32]byte, start, span uint64) [32]byte {
	if start >= uint64(len(chunks)) {
		return zeroHashes[depthOf(span)]
	}
	return Merkleize(chunks[start:min(start+span, uint64(len(chunks)))], span)
}

// boundedBranch returns the siblings of leaf index in a merkleize(chunks,
// limit) tree, bottom-up. At level d the sibling is the node beside the
// leaf's ancestor there, spanning the 2^d leaves from that node's first leaf.
func boundedBranch(chunks [][32]byte, limit, index uint64) [][32]byte {
	branch := make([][32]byte, depthOf(limit))
	for d := range branch {
		branch[d] = rangeRoot(chunks, (index>>d^1)<<d, 1<<d)
	}
	return branch
}

// ProveChunks builds the proof of chunk index in the merkleize(chunks, limit)
// tree, against that tree's root. Every leaf of the padded tree is a node, so
// a chunk at or past len(chunks), and even one at or past the limit, proves a
// zero leaf as long as it lies below next_pow_of_two(limit); the reference's
// node_root reads the same leaves. Beyond the padded width there is no node
// and the result is ErrNoNode. More chunks than the limit is a caller bug, as
// in Merkleize, and panics.
func ProveChunks(chunks [][32]byte, limit, index uint64) (Proof, error) {
	if uint64(len(chunks)) > limit {
		panic(fmt.Sprintf("ssz: proving among %d chunks over limit %d", len(chunks), limit))
	}
	if width := uint64(1) << depthOf(limit); index >= width {
		return Proof{}, fmt.Errorf("ssz: chunk %d outside the %d-leaf tree of limit %d: %w", index, width, limit, ErrNoNode)
	}
	return Proof{
		Gindex: ChunkGindex(limit, index),
		Leaf:   chunkAt(chunks, index),
		Branch: boundedBranch(chunks, limit, index),
	}, nil
}

// spineRoot returns the root of the spine node above level: the fold of every
// level from there down, ending in the zero terminator, which is all there is
// when the tree has no such level. Level 0 gives the progressive root itself;
// the prover asks for the spine below the level it enters.
func spineRoot(chunks [][32]byte, level int) [32]byte {
	n := 0
	for base, _ := progressiveBase(n); base < uint64(len(chunks)); base, _ = progressiveBase(n) {
		n++
	}
	var acc [32]byte
	for i := n - 1; i >= level; i-- {
		base, size := progressiveBase(i)
		acc = hashPair(rangeRoot(chunks, base, size), acc)
	}
	return acc
}

// ProveProgressive builds the proof of chunk index in the
// merkleize_progressive(chunks) tree, against that tree's root: the data tree
// of a progressive list, at 2 under the list root, or the field tree of a
// progressive container. MixLength and MixActiveFields lift the proof to the
// type root, where its Gindex becomes ProgressiveChunkGindex(index); until
// then it is the spine-relative index, 2 for chunk 0 and 24 for chunk 1.
//
// A level exists once its first chunk does, and its subtree is padded to its
// capacity, so a chunk past len(chunks) but inside the last open level proves
// a zero leaf. A chunk in a level the tree has not opened has no node, the
// spine ending in a zero terminator there, and returns ErrNoNode; so does
// every chunk of an empty tree. That is the specification's absence rule:
// provable in a bounded shape, not past the end of a progressive one.
func ProveProgressive(chunks [][32]byte, index uint64) (Proof, error) {
	if k, base, _ := progressiveLevel(index); base >= uint64(len(chunks)) {
		return Proof{}, fmt.Errorf("ssz: chunk %d lies in level %d, which a progressive tree of %d chunks has not opened: %w",
			index, k, len(chunks), ErrNoNode)
	}
	return progressiveProof(chunks, index), nil
}

// progressiveProof is ProveProgressive for a chunk inside an open level. The
// branch is the one inside the level's subtree, then the spine continuation
// beside that subtree, then the shallower subtrees, each the left sibling of
// a spine node on the way up.
func progressiveProof(chunks [][32]byte, index uint64) Proof {
	k, base, size := progressiveLevel(index)
	sub := chunks[base:min(base+size, uint64(len(chunks)))]
	branch := make([][32]byte, 0, 3*k+1)
	branch = append(branch, boundedBranch(sub, size, index-base)...)
	branch = append(branch, spineRoot(chunks, k+1))
	for i := k - 1; i >= 0; i-- {
		b, s := progressiveBase(i)
		branch = append(branch, rangeRoot(chunks, b, s))
	}
	return Proof{Gindex: progressiveLeafGindex(index), Leaf: chunkAt(sub, index-base), Branch: branch}
}

// ProveProgressiveContainer builds the proof of layout position pos of a
// ProgressiveContainer against the container's hash tree root, from the roots
// of its fields in declaration order, as HashTreeRootProgressiveContainer
// takes them. A position is a chunk of the field tree: a field sits at its
// set bit, a cleared bit is a zero leaf, and so is a position past the width
// but inside the last open level; all are nodes, so each proves its leaf,
// which together with the active_fields word at gindex 3 is how a verifier
// tells an absent field from a zero one. A position in a level the width
// does not open returns ErrNoNode, as ProveProgressive would. The Gindex is
// ProgressiveChunkGindex(pos).
func ProveProgressiveContainer(af ActiveFields, roots [][32]byte, pos int) (Proof, error) {
	chunks := layoutChunks(af, roots)
	if pos < 0 {
		return Proof{}, fmt.Errorf("ssz: negative position %d: %w", pos, ErrNoNode)
	}
	p, err := ProveProgressive(chunks, uint64(pos))
	if err != nil {
		return Proof{}, err
	}
	return p.MixActiveFields(af), nil
}
