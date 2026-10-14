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

package ssz_test

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/ssz"
)

// verifyProof routes a proof through beacon/merkle.VerifyProof, the consumer
// every proof of this package is built for, after checking the Proof
// invariant that the verifier would otherwise report as a length error.
func verifyProof(root [32]byte, p ssz.Proof) error {
	if len(p.Branch) != ssz.GindexDepth(p.Gindex) {
		return fmt.Errorf("branch has %d siblings, gindex %b needs %d", len(p.Branch), p.Gindex, ssz.GindexDepth(p.Gindex))
	}
	branch := make(merkle.Values, len(p.Branch))
	for i, b := range p.Branch {
		branch[i] = merkle.Value(b)
	}
	return merkle.VerifyProof(common.Hash(root), p.Gindex, branch, merkle.Value(p.Leaf))
}

// verify is verifyProof as a test assertion.
func verify(t *testing.T, root [32]byte, p ssz.Proof) {
	t.Helper()
	if err := verifyProof(root, p); err != nil {
		t.Fatalf("gindex %d: %v", p.Gindex, err)
	}
}

// randomChunks returns n pseudo-random chunks for a seed.
func randomChunks(seed int64, n int) [][32]byte {
	rng := rand.New(rand.NewSource(seed))
	chunks := make([][32]byte, n)
	for i := range chunks {
		rng.Read(chunks[i][:])
	}
	return chunks
}

// u64Chunk is hash_tree_root of a uint64: the value little-endian in a chunk.
func u64Chunk(v uint64) [32]byte {
	var c [32]byte
	binary.LittleEndian.PutUint64(c[:8], v)
	return c
}

// TestPublishedBranch reproduces, sibling by sibling, the one progressive
// branch the reference publishes (tests/test_proofs.py
// test_a_field_on_a_progressive_spine): field f1 of
// ProgressiveContainer(active_fields=[1, 1, 1]) with f0=11, f1=22, f2=33, at
// gindex 40, whose branch is [root(f2), H(0, 0), 0, root(f0), word(0b111)].
func TestPublishedBranch(t *testing.T) {
	af := ssz.MustActiveFields(1, 1, 1)
	roots := [][32]byte{u64Chunk(11), u64Chunk(22), u64Chunk(33)}
	p, err := ssz.ProveProgressiveContainer(af, roots, 1)
	if err != nil {
		t.Fatal(err)
	}
	if p.Gindex != 40 {
		t.Fatalf("gindex %d, want 40", p.Gindex)
	}
	var zero [32]byte
	want := [][32]byte{roots[2], sha256.Sum256(make([]byte, 64)), zero, roots[0], {0b111}}
	if len(p.Branch) != len(want) {
		t.Fatalf("branch has %d siblings, want %d", len(p.Branch), len(want))
	}
	for i := range want {
		if p.Branch[i] != want[i] {
			t.Errorf("sibling %d: %x, want %x", i, p.Branch[i], want[i])
		}
	}
	if p.Leaf != roots[1] {
		t.Errorf("leaf %x, want root(f1)", p.Leaf)
	}
	verify(t, ssz.HashTreeRootProgressiveContainer(af, roots), p)
}

// paddedWidth is next_pow_of_two(limit), the leaf count of a bounded tree.
func paddedWidth(limit uint64) uint64 {
	width := uint64(1)
	for width < limit {
		width *= 2
	}
	return width
}

// TestProveChunks round-trips bounded-tree proofs for every leaf, real and
// padding, at several limits, and checks the boundary of the padded tree.
func TestProveChunks(t *testing.T) {
	for _, n := range []int{0, 1, 2, 3, 5, 8, 21, 64} {
		chunks := randomChunks(int64(n), n)
		for _, limit := range []uint64{uint64(n), uint64(n) + 1, 64, 1000, 1024} {
			if uint64(n) > limit {
				continue
			}
			root := ssz.Merkleize(chunks, limit)
			width := paddedWidth(limit)
			// Every real leaf, the first padding leaf, the first leaf past the
			// limit and the last leaf of the padded tree.
			indices := []uint64{uint64(n), limit, width - 1}
			for i := range n {
				indices = append(indices, uint64(i))
			}
			for _, idx := range indices {
				if idx >= width {
					continue
				}
				p, err := ssz.ProveChunks(chunks, limit, idx)
				if err != nil {
					t.Fatalf("n=%d limit=%d idx=%d: %v", n, limit, idx, err)
				}
				if p.Gindex != ssz.ChunkGindex(limit, idx) {
					t.Fatalf("n=%d limit=%d idx=%d: gindex %d", n, limit, idx, p.Gindex)
				}
				var want [32]byte
				if idx < uint64(n) {
					want = chunks[idx]
				}
				if p.Leaf != want {
					t.Fatalf("n=%d limit=%d idx=%d: wrong leaf", n, limit, idx)
				}
				verify(t, root, p)
			}
			// The first leaf beyond the padded tree is no node.
			if _, err := ssz.ProveChunks(chunks, limit, width); !errors.Is(err, ssz.ErrNoNode) {
				t.Errorf("n=%d limit=%d idx=%d: err %v, want ErrNoNode", n, limit, width, err)
			}
		}
	}
	mustPanic(t, "chunks over limit", func() { ssz.ProveChunks(randomChunks(9, 3), 2, 0) })
}

// TestProveProgressive round-trips progressive-tree proofs for every chunk at
// every level boundary length and one past, for the padding inside the last
// open level, and checks that the first chunk of an unopened level and every
// chunk of an empty tree are refused.
func TestProveProgressive(t *testing.T) {
	if _, err := ssz.ProveProgressive(nil, 0); !errors.Is(err, ssz.ErrNoNode) {
		t.Errorf("empty tree: err %v, want ErrNoNode", err)
	}
	for _, n := range []int{1, 2, 5, 6, 21, 22, 85, 86, 341, 342, 1365, 1366} {
		chunks := randomChunks(int64(n), n)
		root := ssz.MerkleizeProgressive(chunks)
		// The last open level ends where the next level would begin.
		open := uint64(1)
		for base, size := uint64(0), uint64(1); base+size < uint64(n); {
			base, size = base+size, size*4
			open = base + size
		}
		// Every position of the small trees; a stride through the big ones,
		// always touching the last real chunk, the first padding slot and the
		// last slot of the open level.
		step := uint64(1)
		if n > 400 {
			step = 97
		}
		for idx := uint64(0); idx < open; idx += step {
			if idx+step >= open {
				idx = open - 1
			} else if idx < uint64(n) && idx+step > uint64(n) {
				idx = uint64(n) - 1
			} else if idx > uint64(n) && idx-step < uint64(n) {
				idx = uint64(n)
			}
			p, err := ssz.ProveProgressive(chunks, idx)
			if err != nil {
				t.Fatalf("n=%d idx=%d: %v", n, idx, err)
			}
			var want [32]byte
			if idx < uint64(n) {
				want = chunks[idx]
			}
			if p.Leaf != want {
				t.Fatalf("n=%d idx=%d: wrong leaf", n, idx)
			}
			if lifted := p.MixLength(uint64(n)).Gindex; lifted != ssz.ProgressiveChunkGindex(idx) {
				t.Fatalf("n=%d idx=%d: lifted gindex %d, want %d", n, idx, lifted, ssz.ProgressiveChunkGindex(idx))
			}
			if want, ok := nimbusLeafGindex[idx]; ok && p.Gindex != want {
				t.Fatalf("n=%d idx=%d: gindex %b, nimbus %b", n, idx, p.Gindex, want)
			}
			verify(t, root, p)
		}
		if _, err := ssz.ProveProgressive(chunks, open); !errors.Is(err, ssz.ErrNoNode) {
			t.Errorf("n=%d idx=%d: err %v, want ErrNoNode", n, open, err)
		}
	}
}

// TestProveProgressiveContainer proves every position of a gapped layout,
// fields and gaps alike, against the container root, pins the field indices
// to the remerkleable Square/Circle examples, and checks the width boundary.
func TestProveProgressiveContainer(t *testing.T) {
	af := ssz.MustActiveFields(1, 0, 1, 0, 1)
	roots := randomChunks(5, 3)
	root := ssz.HashTreeRootProgressiveContainer(af, roots)
	field := 0
	for pos := range af.Width() {
		p, err := ssz.ProveProgressiveContainer(af, roots, pos)
		if err != nil {
			t.Fatalf("position %d: %v", pos, err)
		}
		if p.Gindex != ssz.ProgressiveChunkGindex(uint64(pos)) {
			t.Fatalf("position %d: gindex %d", pos, p.Gindex)
		}
		var want [32]byte
		if af.Active(pos) {
			want = roots[field]
			field++
		}
		if p.Leaf != want {
			t.Fatalf("position %d: wrong leaf", pos)
		}
		verify(t, root, p)
	}
	for _, pos := range []int{-1, 5, 6} {
		if _, err := ssz.ProveProgressiveContainer(af, roots, pos); !errors.Is(err, ssz.ErrNoNode) {
			t.Errorf("position %d: err %v, want ErrNoNode", pos, err)
		}
	}
	// A three-position layout opens the second level, so positions 3 and 4
	// are zero leaves past the width ("past the layout, leaving both leaves
	// zero", as the reference puts it); position 5 is a level it never opens.
	three := ssz.MustActiveFields(1, 1, 1)
	for pos := 3; pos <= 4; pos++ {
		p, err := ssz.ProveProgressiveContainer(three, roots, pos)
		if err != nil {
			t.Fatalf("position %d: %v", pos, err)
		}
		if p.Leaf != [32]byte{} {
			t.Fatalf("position %d: leaf %x, want zero", pos, p.Leaf)
		}
		verify(t, ssz.HashTreeRootProgressiveContainer(three, roots), p)
	}
	if _, err := ssz.ProveProgressiveContainer(three, roots, 5); !errors.Is(err, ssz.ErrNoNode) {
		t.Errorf("position 5: err %v, want ErrNoNode", err)
	}
	mustPanic(t, "root count", func() { ssz.ProveProgressiveContainer(af, roots[:2], 0) })
	mustPanic(t, "uninitialized", func() { ssz.ProveProgressiveContainer(ssz.ActiveFields{}, nil, 0) })

	// remerkleable test_typing.py: Square(active_fields=[1, 0, 1]) has side at
	// 0b100 and color at 0b101001; Circle(active_fields=[0, 1, 1]) has radius
	// at 0b101000 and color at 0b101001.
	square, circle := ssz.MustActiveFields(1, 0, 1), ssz.MustActiveFields(0, 1, 1)
	two := randomChunks(6, 2)
	for _, c := range []struct {
		name string
		af   ssz.ActiveFields
		pos  int
		want uint64
	}{
		{"Square.side", square, 0, 0b100}, {"Square.color", square, 2, 0b101001},
		{"Circle.radius", circle, 1, 0b101000}, {"Circle.color", circle, 2, 0b101001},
	} {
		p, err := ssz.ProveProgressiveContainer(c.af, two, c.pos)
		if err != nil {
			t.Fatal(err)
		}
		if p.Gindex != c.want {
			t.Errorf("%s: gindex %b, want %b", c.name, p.Gindex, c.want)
		}
		verify(t, ssz.HashTreeRootProgressiveContainer(c.af, two), p)
	}
}

// TestProofComposition proves leaves through nesting boundaries: a chunk of a
// progressive byte list held by a classic container, and an element of a
// classic list held by a progressive container, each against the outer root
// and with the gindex checked against the hand-computed concatenation.
func TestProofComposition(t *testing.T) {
	// A ProgressiveByteList of 300 bytes (10 chunks) as field 1 of a classic
	// three-field container: the proof of chunk 3 under the container root.
	data := make([]byte, 300)
	rand.New(rand.NewSource(7)).Read(data)
	listRoot := ssz.HashTreeRootProgressiveBytes(data)
	fields := [][32]byte{randomChunks(8, 1)[0], listRoot, randomChunks(9, 1)[0]}
	containerRoot := ssz.Merkleize(fields, 3)

	inner, err := ssz.ProveProgressive(ssz.Pack(data), 3)
	if err != nil {
		t.Fatal(err)
	}
	inList := inner.MixLength(uint64(len(data)))
	verify(t, listRoot, inList)
	outer, err := ssz.ProveChunks(fields, 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	full := inList.Wrap(outer)
	if want := ssz.ConcatGindex(ssz.ChunkGindex(3, 1), ssz.ProgressiveChunkGindex(3)); full.Gindex != want {
		t.Errorf("gindex %d, want %d", full.Gindex, want)
	}
	verify(t, containerRoot, full)

	// A List[uint64, 8] of five elements (two chunks, limit two) at position 2
	// of ProgressiveContainer(active_fields=[1, 0, 1]): the proof of chunk 1,
	// elements 4 to 7, under the container root.
	packed := ssz.Pack(binary.LittleEndian.AppendUint64(make([]byte, 32), 5))
	classicRoot := ssz.MixInLength(ssz.Merkleize(packed, 2), 5)
	af := ssz.MustActiveFields(1, 0, 1)
	roots := [][32]byte{randomChunks(10, 1)[0], classicRoot}
	inner, err = ssz.ProveChunks(packed, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	outer, err = ssz.ProveProgressiveContainer(af, roots, 2)
	if err != nil {
		t.Fatal(err)
	}
	full = inner.MixLength(5).Wrap(outer)
	if want := ssz.ConcatGindex(ssz.ProgressiveChunkGindex(2), ssz.ConcatGindex(2, ssz.ChunkGindex(2, 1))); full.Gindex != want {
		t.Errorf("gindex %d, want %d", full.Gindex, want)
	}
	verify(t, ssz.HashTreeRootProgressiveContainer(af, roots), full)
}

// TestAbsence is the reference's absence table (ssz-specs tests/test_proofs.py
// TestZeroPaddingIsAProvableNode and test_an_index_naming_no_node_is_refused),
// one row per shape: padding inside a tree is a provable zero leaf, a
// position the tree does not reach is ErrNoNode.
func TestAbsence(t *testing.T) {
	two := randomChunks(11, 2)
	gapped := ssz.MustActiveFields(1, 0, 1, 0, 1)
	for _, c := range []struct {
		name  string
		prove func() (ssz.Proof, error)
		root  [32]byte
		node  bool
	}{
		{"capacity chunk past the length", func() (ssz.Proof, error) { return ssz.ProveChunks(two, 8, 5) }, ssz.Merkleize(two, 8), true},
		{"leaf past the field count", func() (ssz.Proof, error) { return ssz.ProveChunks(two[:1], 3, 3) }, ssz.Merkleize(two[:1], 3), true},
		{"leaf past the padded tree", func() (ssz.Proof, error) { return ssz.ProveChunks(two[:1], 3, 4) }, [32]byte{}, false},
		{"chunk outside the capacity", func() (ssz.Proof, error) { return ssz.ProveChunks(two, 8, 8) }, [32]byte{}, false},
		{"slot inside an open level", func() (ssz.Proof, error) { return ssz.ProveProgressive(two, 4) }, ssz.MerkleizeProgressive(two), true},
		{"slot in an unopened level", func() (ssz.Proof, error) { return ssz.ProveProgressive(two, 5) }, [32]byte{}, false},
		{"one chunk, second level unopened", func() (ssz.Proof, error) { return ssz.ProveProgressive(two[:1], 1) }, [32]byte{}, false},
		{"empty progressive tree", func() (ssz.Proof, error) { return ssz.ProveProgressive(nil, 0) }, [32]byte{}, false},
		{"cleared progressive position", func() (ssz.Proof, error) { return ssz.ProveProgressiveContainer(gapped, randomChunks(12, 3), 3) }, ssz.HashTreeRootProgressiveContainer(gapped, randomChunks(12, 3)), true},
		{"position in a level the layout never opens", func() (ssz.Proof, error) { return ssz.ProveProgressiveContainer(gapped, randomChunks(12, 3), 5) }, [32]byte{}, false},
	} {
		p, err := c.prove()
		switch {
		case !c.node:
			if !errors.Is(err, ssz.ErrNoNode) {
				t.Errorf("%s: err %v, want ErrNoNode", c.name, err)
			}
		case err != nil:
			t.Errorf("%s: %v", c.name, err)
		case p.Leaf != [32]byte{}:
			t.Errorf("%s: leaf %x, want zero", c.name, p.Leaf)
		default:
			if err := verifyProof(c.root, p); err != nil {
				t.Errorf("%s: %v", c.name, err)
			}
		}
	}
}

// TestProofRejections gives the verifier the other tests lean on something
// to reject: a tampered leaf, a reversed branch, a short branch, the sibling's
// index and a mis-composed Wrap all fail, so a passing round trip means the
// proof was right and not that the check was vacuous. The reference pins the
// first three the same way (test_a_tampered_leaf_fails,
// test_a_reversed_branch_fails, test_a_branch_of_the_wrong_length_is_refused).
func TestProofRejections(t *testing.T) {
	chunks := randomChunks(13, 6)
	root := ssz.MerkleizeProgressive(chunks)
	p, err := ssz.ProveProgressive(chunks, 3) // second level: an asymmetric four-sibling branch
	if err != nil {
		t.Fatal(err)
	}
	verify(t, root, p)

	tampered := p
	tampered.Leaf[0] ^= 1
	if verifyProof(root, tampered) == nil {
		t.Error("tampered leaf verified")
	}
	reversed := p
	reversed.Branch = slices.Clone(p.Branch)
	slices.Reverse(reversed.Branch)
	if verifyProof(root, reversed) == nil {
		t.Error("reversed branch verified")
	}
	short := p
	short.Branch = p.Branch[:len(p.Branch)-1]
	if verifyProof(root, short) == nil {
		t.Error("short branch verified")
	}
	sibling := p
	sibling.Gindex ^= 1
	if verifyProof(root, sibling) == nil {
		t.Error("proof verified at the sibling's index")
	}
	// Wrap trusts outer.Leaf to be the inner root; here it is not.
	fields := randomChunks(14, 3)
	outer, err := ssz.ProveChunks(fields, 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	if verifyProof(ssz.Merkleize(fields, 3), p.MixLength(6).Wrap(outer)) == nil {
		t.Error("mis-composed proof verified")
	}
	mustPanic(t, "MixActiveFields uninitialized", func() { ssz.Proof{Gindex: 2}.MixActiveFields(ssz.ActiveFields{}) })
}

// The cases below were produced by the SSZ executable reference, eth-ssz-specs
// 0.1.0 (ethereum/ssz-specs v0.1.0): a value, a path into it, and what
// get_generalized_index, node_root and build_proof return for that path, or
// the refusal they raise. They are ten of the 140 a generator run produced,
// one per shape; the full set and the generator stay out of the tree. Values
// are rebuilt here exactly as the generator built them: uint64 fields and
// elements are their little-endian chunk, Root32 fields are the given bytes,
// and a seeded run has chunk i = sha256(seed LE64 || i LE64).

// hex32 decodes a 0x-prefixed 32-byte hex string.
func hex32(t *testing.T, s string) [32]byte {
	t.Helper()
	b, err := hexutil.Decode(s)
	if err != nil || len(b) != 32 {
		t.Fatalf("bad 32-byte hex %q: %v", s, err)
	}
	return [32]byte(b)
}

// seeded returns n chunks derived from a seed.
func seeded(seed uint64, n int) [][32]byte {
	chunks := make([][32]byte, n)
	var buf [16]byte
	binary.LittleEndian.PutUint64(buf[:8], seed)
	for i := range chunks {
		binary.LittleEndian.PutUint64(buf[8:], uint64(i))
		chunks[i] = sha256.Sum256(buf[:])
	}
	return chunks
}

// fill returns a chunk of one repeated byte.
func fill(b byte) [32]byte {
	var c [32]byte
	for i := range c {
		c[i] = b
	}
	return c
}

// allActive returns an active_fields configuration of n set bits.
func allActive(n int) ssz.ActiveFields {
	mask := make([]int, n)
	for i := range mask {
		mask[i] = 1
	}
	return ssz.MustActiveFields(mask...)
}

// proveList proves a chunk of a progressive list against the list root.
func proveList(chunks [][32]byte, length, index uint64) (ssz.Proof, error) {
	p, err := ssz.ProveProgressive(chunks, index)
	if err != nil {
		return p, err
	}
	return p.MixLength(length), nil
}

// squares100 packs the ProgressiveList[uint64] of i*i for i below 100.
func squares100() [][32]byte {
	var data []byte
	for i := uint64(0); i < 100; i++ {
		data = binary.LittleEndian.AppendUint64(data, i*i)
	}
	return ssz.Pack(data)
}

// bits300 packs a ProgressiveBitlist of 300 bits, set where i%3 == 0.
func bits300() []byte {
	bits := make([]byte, 38)
	for i := 0; i < 300; i += 3 {
		bits[i/8] |= 1 << (i % 8)
	}
	return bits
}

var gappedAF = ssz.MustActiveFields(1, 0, 1, 0, 1)

// gappedRoots are the fields a=1, b=0x07.., c=2 of the gapped container.
func gappedRoots() [][32]byte { return [][32]byte{u64Chunk(1), fill(0x07), u64Chunk(2)} }

// tripleRoots are the fields x=1, b=0x42.., y=2 of a three-field container.
func tripleRoots() [][32]byte { return [][32]byte{u64Chunk(1), fill(0x42), u64Chunk(2)} }

// pairField proves field `field` of element `elem` of a ProgressiveList of
// seven containers {a: i, b: 100+i} against the list root.
func pairField(elem, field uint64) (ssz.Proof, error) {
	pairs := make([][32]byte, 7)
	for i := range pairs {
		pairs[i] = ssz.Merkleize([][32]byte{u64Chunk(uint64(i)), u64Chunk(100 + uint64(i))}, 2)
	}
	inner, err := ssz.ProveChunks([][32]byte{u64Chunk(elem), u64Chunk(100 + elem)}, 2, field)
	if err != nil {
		return inner, err
	}
	outer, err := proveList(pairs, 7, elem)
	if err != nil {
		return outer, err
	}
	return inner.Wrap(outer), nil
}

// gloasBlockHash proves parent_block_hash through a Gloas-shaped body: a
// 13-position progressive body whose position 10 is a classic {message,
// signature} container whose message is a 12-position progressive bid, with
// parent_block_hash at bid position 0. Bid fields are 0xB0+i, body fields
// 0xC0+i, the signature 0x51.
func gloasBlockHash() (ssz.Proof, error) {
	bidAF, bodyAF := allActive(12), allActive(13)
	bidRoots := make([][32]byte, 12)
	for i := range bidRoots {
		bidRoots[i] = fill(0xB0 + byte(i))
	}
	bid, err := ssz.ProveProgressiveContainer(bidAF, bidRoots, 0)
	if err != nil {
		return bid, err
	}
	signed := [][32]byte{ssz.HashTreeRootProgressiveContainer(bidAF, bidRoots), fill(0x51)}
	message, err := ssz.ProveChunks(signed, 2, 0)
	if err != nil {
		return message, err
	}
	bodyRoots := make([][32]byte, 13)
	for i := range bodyRoots {
		bodyRoots[i] = fill(0xC0 + byte(i))
	}
	bodyRoots[10] = ssz.Merkleize(signed, 2)
	body, err := ssz.ProveProgressiveContainer(bodyAF, bodyRoots, 10)
	if err != nil {
		return body, err
	}
	return bid.Wrap(message).Wrap(body), nil
}

var referenceProofs = []struct {
	name    string
	build   func() (ssz.Proof, error)
	root    string
	gindex  uint64
	leaf    string
	branch  []string
	refused bool
}{
	// ProgressiveList of six roots, element 5: the first chunk of the third level.
	{
		name:   "progressive_roots/len6/elem5",
		build:  func() (ssz.Proof, error) { return proveList(seeded(6, 6), 6, 5) },
		root:   "0x892ff0b45185b4c73f89a50680a4fcdbc5e12e7705aa5f55481bd81503374b22",
		gindex: 352,
		leaf:   "0xd9c0a65b5aa2d86f56f284314944176e8025caf72bc38af9b7cedd3c282a0a3c",
		branch: []string{
			"0x0000000000000000000000000000000000000000000000000000000000000000",
			"0xf5a5fd42d16a20302798ef6ed309979b43003d2320d9f0e8ea9831a92759fb4b",
			"0xdb56114e00fdd4c1f85c892bf35ac9a89289aaecb1ebd0a96cde606a748b5d71",
			"0xc78009fdf07fc56a11f122370658a353aaa542ed63e44c4bc15ff4cd105ab33c",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
			"0x4378d8b1ec6878ca380401c0e5eb19663ab3cd39b814705c5e9f079ab0a363bd",
			"0x5e9a5de4a59a7c3c9ba8846060ba77ab402e3088658aa705e1a64e796f6cd5cf",
			"0x0600000000000000000000000000000000000000000000000000000000000000",
		},
	},
	// ProgressiveList[uint64] of 100 elements (25 chunks), element 20 in chunk 5.
	{
		name:   "progressive_u64/len100/elem20",
		build:  func() (ssz.Proof, error) { return proveList(squares100(), 100, 5) },
		root:   "0xfaf24d8cdd6176bdbe47d6e09d679d4e9e411b7dccaa347cca228671400c0516",
		gindex: 352,
		leaf:   "0x9001000000000000b901000000000000e4010000000000001102000000000000",
		branch: []string{
			"0x40020000000000007102000000000000a402000000000000d902000000000000",
			"0x42058172b481a775c0896ecb461663a764948dcc4c8507c4697ea3a6c2059f06",
			"0xbc686a1103be292f2861cc70152e050bf156874448fbddf0dfc740d61220ef33",
			"0xcf3d78bd67dd63de40e60d1fc975e44a3701145f27dce17e420557e57518c939",
			"0x6f847d0b6e9cce917ed221e184521fe00ba0edb8f490c1418ef224c342eefce3",
			"0xc4791f6820dd21f0725de03e0a3115bad64a3cb0d120182b1978c70934d6a629",
			"0x0000000000000000010000000000000004000000000000000900000000000000",
			"0x6400000000000000000000000000000000000000000000000000000000000000",
		},
	},
	// The same list, chunk 85: the fifth level, which 25 chunks never open.
	{
		name:    "progressive_u64/len100/unopened85",
		build:   func() (ssz.Proof, error) { return ssz.ProveProgressive(squares100(), 85) },
		refused: true,
	},
	// ProgressiveBitlist of 300 bits, bit 299 in chunk 1.
	{
		name:   "progressive_bits/len300/bit299",
		build:  func() (ssz.Proof, error) { return proveList(ssz.PackBits(bits300()), 300, 1) },
		root:   "0xbafba44db4a64aee2c89b8a6998ab445340cfe61e0ee51882ac5948cb60542af",
		gindex: 40,
		leaf:   "0x2449922449020000000000000000000000000000000000000000000000000000",
		branch: []string{
			"0x0000000000000000000000000000000000000000000000000000000000000000",
			"0xf5a5fd42d16a20302798ef6ed309979b43003d2320d9f0e8ea9831a92759fb4b",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
			"0x4992244992244992244992244992244992244992244992244992244992244992",
			"0x2c01000000000000000000000000000000000000000000000000000000000000",
		},
	},
	// ProgressiveContainer(active_fields=[1,0,1,0,1]), the cleared position 1.
	{
		name:   "progressive_container/gapped/gap1",
		build:  func() (ssz.Proof, error) { return ssz.ProveProgressiveContainer(gappedAF, gappedRoots(), 1) },
		root:   "0x9ac93a1ee809e60248a9dd87c52ad7e7df8dec544507e0455fd3594b0424b0dd",
		gindex: 40,
		leaf:   "0x0000000000000000000000000000000000000000000000000000000000000000",
		branch: []string{
			"0x0707070707070707070707070707070707070707070707070707070707070707",
			"0x1205f4789155711e2542dba1a64d226626fe3eb43baa854752d0b59077e010fc",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
			"0x0100000000000000000000000000000000000000000000000000000000000000",
			"0x1500000000000000000000000000000000000000000000000000000000000000",
		},
	},
	// A 46-position layout like the Gloas BeaconState, position 20.
	{
		name:   "progressive_container/wide46/f20",
		build:  func() (ssz.Proof, error) { return ssz.ProveProgressiveContainer(allActive(46), seeded(46, 46), 20) },
		root:   "0xd01989789f1fb1458f6dc25e0a986779969d341b67e55d3eacf4820798981bd9",
		gindex: 367,
		leaf:   "0xbab1998f070b8f8488a2b959aed34dc8b2eb353f8076489c5247bdb3de0d4736",
		branch: []string{
			"0x5a176710dc0c6cff1cb85a94010bf1ea0c5cc1503109ec3b9ae00a162ed98e39",
			"0xf333241d37b2789b3929b99ba80ca379421b734ac46b15a6f9b05e5ef4242596",
			"0x815ffa2e349c95cdd36747cd4382ea31fdf3aab554ee230ed89d938b98d333ea",
			"0x6a523a8ab37c21624b881a450c4d152b283f1679a077f2147d892da774d95f0a",
			"0x398f4dc8a77c0c18f3d1d7896f3da9207fc7d1728a05ea84e9bc0a1a8ca543be",
			"0x5dfe23cf31cf826f631aa1c6add405ec2d1e574dc4462c31e97d3dd8a69d6afd",
			"0x52390ff845440f8181c7b4ad1f64c4b01420c0503c959c8a082c73af93fbe577",
			"0xffffffffff3f0000000000000000000000000000000000000000000000000000",
		},
	},
	// A three-field container, the padding leaf past the field count.
	{
		name:   "bounded_container/triple/padding3",
		build:  func() (ssz.Proof, error) { return ssz.ProveChunks(tripleRoots(), 3, 3) },
		root:   "0x6febda1be81a67f5e9f117fc42e34e08418db1437812699ae19e20d4bd011e2a",
		gindex: 7,
		leaf:   "0x0000000000000000000000000000000000000000000000000000000000000000",
		branch: []string{
			"0x0200000000000000000000000000000000000000000000000000000000000000",
			"0xa1f8349579c6543908b3da647eac2b7e032b1472134d60334f402d00cafc90cb",
		},
	},
	// ProgressiveList of seven two-field containers, field b of element 6.
	{
		name:   "nested/progressive_list_of_containers/6b",
		build:  func() (ssz.Proof, error) { return pairField(6, 1) },
		root:   "0x7b5be04317dc79a0b7bcfb2d35d7f47fcd9263d12b492309b74951a65b9f994f",
		gindex: 707,
		leaf:   "0x6a00000000000000000000000000000000000000000000000000000000000000",
		branch: []string{
			"0x0600000000000000000000000000000000000000000000000000000000000000",
			"0x7e57e1f7c17ae396534910a888f6014e39bda886c52765b734e06ee1fa4f0e8d",
			"0xf5a5fd42d16a20302798ef6ed309979b43003d2320d9f0e8ea9831a92759fb4b",
			"0xdb56114e00fdd4c1f85c892bf35ac9a89289aaecb1ebd0a96cde606a748b5d71",
			"0xc78009fdf07fc56a11f122370658a353aaa542ed63e44c4bc15ff4cd105ab33c",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
			"0x3fa32ed2ad0fe0ee84d4b7c406db593eb2de72a411d7eb5ebc02bb029e206c05",
			"0x2b15345da96dd0c2ac1e9a4749717bdc1845714420e0178fff2366b17b14dcdd",
			"0x0700000000000000000000000000000000000000000000000000000000000000",
		},
	},
	// A Gloas-shaped body: progressive body, classic signed bid, progressive bid, parent_block_hash.
	{
		name:   "nested/gloas_body/parent_block_hash_2856",
		build:  func() (ssz.Proof, error) { return gloasBlockHash() },
		root:   "0xccce48fdb868c3bd47f16e36b5eea0925c6bafc130aa62210927cf30a0ccb7ad",
		gindex: 2856,
		leaf:   "0xb0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0",
		branch: []string{
			"0xf76151119a90cfe691797a2a2d448f004ed8b2ed90f743aac8e87a28ce86901e",
			"0xff0f000000000000000000000000000000000000000000000000000000000000",
			"0x5151515151515151515151515151515151515151515151515151515151515151",
			"0xc9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9c9",
			"0x171e345d32e8f67762c4fe7731b075f9e676474aa3766471d08c8bc8f12d1bc6",
			"0x9387b8336ffa7bf4bf60f4d839358438d5b39badba7d9484af7fc454c8e734c8",
			"0xc78009fdf07fc56a11f122370658a353aaa542ed63e44c4bc15ff4cd105ab33c",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
			"0xa9874d0bb7b1e2b8853f626cff9498f941e729f2a08575a8cbde947faeaaba02",
			"0xc0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0",
			"0xff1f000000000000000000000000000000000000000000000000000000000000",
		},
	},
	// A progressive list of three roots inside a container, chunk 5 of the list: unopened.
	{
		name:    "nested/container_progressive_list/lst_unopened5",
		build:   func() (ssz.Proof, error) { return ssz.ProveProgressive(seeded(3, 3), 5) },
		refused: true,
	},
}

// TestReferenceProofs rebuilds the proofs the reference built and requires
// its generalized index, leaf and branch exactly, verified against its root,
// or ErrNoNode where it refused.
func TestReferenceProofs(t *testing.T) {
	for _, c := range referenceProofs {
		t.Run(c.name, func(t *testing.T) {
			p, err := c.build()
			if c.refused {
				if !errors.Is(err, ssz.ErrNoNode) {
					t.Fatalf("err %v, want ErrNoNode", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if p.Gindex != c.gindex {
				t.Fatalf("gindex %d, reference %d", p.Gindex, c.gindex)
			}
			if leaf := hex32(t, c.leaf); p.Leaf != leaf {
				t.Fatalf("leaf %x, reference %x", p.Leaf, leaf)
			}
			if len(p.Branch) != len(c.branch) {
				t.Fatalf("branch has %d siblings, reference %d", len(p.Branch), len(c.branch))
			}
			for i, h := range c.branch {
				if want := hex32(t, h); p.Branch[i] != want {
					t.Fatalf("sibling %d: %x, reference %x", i, p.Branch[i], want)
				}
			}
			verify(t, hex32(t, c.root), p)
		})
	}
}

// FuzzProve builds bounded and progressive proofs for arbitrary positions and
// checks that each either verifies against the matching root, with the leaf
// the tree holds there, or is refused with ErrNoNode exactly where the
// absence rule says there is no node.
func FuzzProve(f *testing.F) {
	f.Add(uint16(0), uint16(0), uint32(0), int64(1))
	f.Add(uint16(5), uint16(3), uint32(7), int64(2))
	f.Add(uint16(21), uint16(0), uint32(84), int64(3))
	f.Add(uint16(86), uint16(1000), uint32(340), int64(4))
	f.Fuzz(func(t *testing.T, n, extra uint16, index uint32, seed int64) {
		n %= 400
		chunks := randomChunks(seed, int(n))
		limit := uint64(n) + uint64(extra%1024)
		idx := uint64(index)

		p, err := ssz.ProveChunks(chunks, limit, idx)
		if idx < paddedWidth(limit) {
			if err != nil {
				t.Fatalf("bounded n=%d limit=%d idx=%d: %v", n, limit, idx, err)
			}
			if want := chunkOrZero(chunks, idx); p.Leaf != want {
				t.Fatalf("bounded n=%d limit=%d idx=%d: wrong leaf", n, limit, idx)
			}
			if err := verifyProof(ssz.Merkleize(chunks, limit), p); err != nil {
				t.Fatalf("bounded n=%d limit=%d idx=%d: %v", n, limit, idx, err)
			}
		} else if !errors.Is(err, ssz.ErrNoNode) {
			t.Fatalf("bounded n=%d limit=%d idx=%d: err %v, want ErrNoNode", n, limit, idx, err)
		}

		open := uint64(0)
		if n > 0 {
			open = 1
			for base, size := uint64(0), uint64(1); base+size < uint64(n); {
				base, size = base+size, size*4
				open = base + size
			}
		}
		p, err = ssz.ProveProgressive(chunks, idx)
		if idx < open {
			if err != nil {
				t.Fatalf("progressive n=%d idx=%d: %v", n, idx, err)
			}
			if want := chunkOrZero(chunks, idx); p.Leaf != want {
				t.Fatalf("progressive n=%d idx=%d: wrong leaf", n, idx)
			}
			if err := verifyProof(ssz.MerkleizeProgressive(chunks), p); err != nil {
				t.Fatalf("progressive n=%d idx=%d: %v", n, idx, err)
			}
		} else if !errors.Is(err, ssz.ErrNoNode) {
			t.Fatalf("progressive n=%d idx=%d: err %v, want ErrNoNode", n, idx, err)
		}
	})
}

// chunkOrZero is the leaf a padded tree holds at index.
func chunkOrZero(chunks [][32]byte, index uint64) [32]byte {
	if index < uint64(len(chunks)) {
		return chunks[index]
	}
	return [32]byte{}
}
