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

package bintrie

import (
	"bytes"
	"errors"
	"math/bits"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"
)

// A multiproof answers a set of keys at once, shipping only what a verifier
// cannot work out for itself.
//
// The proof this package already had expands every node on every path into a
// standalone hash preimage, so proving a whole contract's code pays three
// times over: for internal branches the verifier could recompute from the
// leaves it was given, for the two child hashes of every branch when only the
// sibling is unknown, and for a copy of each key it supplied in the request.
// Measured on a 24 KiB contract those are 50.2%, part of the same 50.2%, and
// 26.0% of a 106,795-byte proof, against a 25,376-byte payload.
//
// The encoding below is a preorder walk of the union of paths to the queried
// keys. Branches carry their prefix but neither child; a subtree holding no
// queried key collapses to its hash; and a stem ships its values against a
// bitmap rather than repeating the stem per value.
//
// Verification is a full bottom-up recomputation ending in a root comparison,
// which is what makes the omissions safe: anything the prover misstates about
// a value, a prefix or a shape changes a preimage and the root stops matching.
// Two things that recomputation does not cover are handled explicitly:
//
//   - Absence is answered by reading the rebuilt tree, never by consulting a
//     bitmap. Bits describing leaves under a collapsed subtree enter no
//     preimage, so a prover can lie about them freely; a stem holding
//     {0,1,5} proved for {5} folds identically to one holding {0,5}, and the
//     genuine hash of the {0,1} subtree satisfies both. The walk never
//     collapses a subtree a queried key descends into, so a rebuilt tree
//     answers absence from structure the root check covers.
//   - Decoding never adopts a hash from the proof. Every node is rebuilt from
//     its children so that Hash() recomputes rather than replays; taking the
//     claimed hashes would make verification tautological.
type Multiproof struct {
	tokens []mpToken
}

// Token kinds in the preorder stream.
const (
	mpBranch = 0x01 // prefix, then the left subtree, then the right
	mpStub   = 0x02 // hash of a subtree holding no queried key
	mpGroup  = 0x03 // one stem: its shape, the values proved, and inner stubs
)

type mpToken struct {
	kind byte

	prefix bitstr      // mpBranch
	hash   common.Hash // mpStub

	stem    []byte   // mpGroup
	present [32]byte // sub-indices the stem holds; fixes the fold shape
	covered [32]byte // those whose value ships here (subset of present)
	values  [][]byte // in ascending sub-index order
	stubs   []common.Hash
}

// ErrProofMalformed is returned when a proof is structurally invalid, does not
// hash to the root it claims, or is a second encoding of a tree that already
// has one. A proof that is merely incomplete is not an error here: it rebuilds
// into a tree whose uncovered parts are unresolved, and reading one of those
// reports a missing node.
var ErrProofMalformed = errors.New("bintrie: malformed multiproof")

//
// Bitmap helpers. Sub-indices are one byte, so a stem's occupancy is 256 bits.
//

func bitmapSet(m *[32]byte, sub byte)      { m[sub>>3] |= 1 << (7 - sub&7) }
func bitmapHas(m *[32]byte, sub byte) bool { return m[sub>>3]&(1<<(7-sub&7)) != 0 }

func bitmapSubs(m *[32]byte) []byte {
	out := make([]byte, 0, 32)
	for i := 0; i < 256; i++ {
		if bitmapHas(m, byte(i)) {
			out = append(out, byte(i))
		}
	}
	return out
}

func bitmapCount(m *[32]byte) int {
	n := 0
	for _, b := range m {
		n += bits.OnesCount8(b)
	}
	return n
}

//
// Proving
//

// ProveMulti proves every key in keys against the trie's current root.
//
// Keys may be present or absent; an absent key is answered by the structure
// that shows it absent, the same divergence a single-key proof witnesses.
func (t *BinaryTrie) ProveMulti(keys [][]byte) (*Multiproof, error) {
	if t.committed {
		return nil, trie.ErrCommitted
	}
	sorted := make([][]byte, 0, len(keys))
	for _, k := range keys {
		if err := validateKey(k); err != nil {
			return nil, err
		}
		sorted = append(sorted, k)
	}
	slices.SortFunc(sorted, bytes.Compare)
	sorted = slices.CompactFunc(sorted, bytes.Equal)

	mp := new(Multiproof)
	n, err := t.proveMultiWalk(t.root, sorted, 0, mp)
	if err != nil {
		return nil, err
	}
	t.root = n
	return mp, nil
}

// proveMultiWalk emits the preorder covering keys, which are sorted and all
// share the bits consumed above this node.
func (t *BinaryTrie) proveMultiWalk(n binaryNode, keys [][]byte, pos int, mp *Multiproof) (binaryNode, error) {
	// A subtree no queried key descends into is shipped as one hash. This is
	// also what keeps absence answerable: a key always routes into a subtree
	// that gets walked, never into one that collapses.
	if len(keys) == 0 {
		if _, isEmpty := n.(empty); !isEmpty {
			mp.tokens = append(mp.tokens, mpToken{kind: mpStub, hash: n.hashAt(pos)})
		}
		return n, nil
	}
	switch nn := n.(type) {
	case empty:
		// Every queried key here is absent, and an empty tree proves it.
		return n, nil

	case hashedNode:
		resolved, err := t.resolve(nn, keyWalk(keys[0], pos))
		if err != nil {
			return n, err
		}
		return t.proveMultiWalk(resolved, keys, pos, mp)

	case *groupNode:
		mp.tokens = append(mp.tokens, groupToken(nn, keys, pos))
		return n, nil

	case *branchNode:
		mp.tokens = append(mp.tokens, mpToken{kind: mpBranch, prefix: nn.prefix})
		split := pos + nn.prefix.n

		// Keys that diverge from the prefix are absent, and this branch is the
		// witness; they take no further part in the walk.
		var live [][]byte
		for _, k := range keys {
			if nn.prefix.matchKey(k, pos) == nn.prefix.n && split < 8*len(k) {
				live = append(live, k)
			}
		}
		var left, right [][]byte
		for _, k := range live {
			if bitAt(k, split) == 0 {
				left = append(left, k)
			} else {
				right = append(right, k)
			}
		}
		l, err := t.proveMultiWalk(nn.left, left, split+1, mp)
		if err != nil {
			return n, err
		}
		nn.left = l
		r, err := t.proveMultiWalk(nn.right, right, split+1, mp)
		if err != nil {
			return n, err
		}
		nn.right = r
		return n, nil

	default:
		return n, ErrProofMalformed
	}
}

// groupToken describes one stem: which sub-indices it holds, which of them
// the caller asked for, their values, and one hash per maximal subtree of the
// stem that holds none of them.
func groupToken(g *groupNode, keys [][]byte, pos int) mpToken {
	tok := mpToken{kind: mpGroup, stem: slices.Clone(g.stem)}
	for _, sub := range g.subs {
		bitmapSet(&tok.present, sub)
	}
	// Only keys naming this stem contribute; the rest are absent here and the
	// stem itself is their witness.
	for _, k := range keys {
		if !bytes.Equal(k[:len(k)-1], g.stem) {
			continue
		}
		if sub := k[len(k)-1]; g.lookup(sub) != nil {
			bitmapSet(&tok.covered, sub)
		}
	}
	for i, sub := range g.subs {
		if bitmapHas(&tok.covered, sub) {
			tok.values = append(tok.values, slices.Clone(g.vals[i]))
		}
	}
	if len(g.subs) == 1 {
		if bitmapCount(&tok.covered) == 0 {
			tok.stubs = append(tok.stubs, g.fold(pos))
		}
		return tok
	}
	collectGroupStubs(g, &tok.covered, 0, len(g.subs), 0, pos, 8*len(g.stem), &tok.stubs)
	return tok
}

// collectGroupStubs mirrors foldRange, emitting the hash of every maximal
// range that holds no covered sub-index. The verifier walks the same shape
// from the present bitmap and consumes these in the same order.
func collectGroupStubs(g *groupNode, covered *[32]byte, i, j, from, extraLo, extraHi int, out *[]common.Hash) {
	anyCovered := false
	for k := i; k < j; k++ {
		if bitmapHas(covered, g.subs[k]) {
			anyCovered = true
			break
		}
	}
	if !anyCovered {
		*out = append(*out, g.foldRange(i, j, from, extraLo, extraHi))
		return
	}
	if j-i == 1 {
		return // a covered leaf; its value ships instead
	}
	b := 8 - bits.Len8(g.subs[i]^g.subs[j-1])
	m := i + 1
	for m < j && g.subs[m]>>(7-b)&1 == 0 {
		m++
	}
	collectGroupStubs(g, covered, i, m, b+1, 0, 0, out)
	collectGroupStubs(g, covered, m, j, b+1, 0, 0, out)
}

//
// Verifying
//

// VerifyMultiproof rebuilds the proved fragment of the tree, checks that it
// hashes to root, and returns it as a trie.
//
// The returned trie answers reads for everything the proof covers and returns
// a missing-node error for everything it does not, so an under-covered proof
// cannot be mistaken for a tree in which those keys are absent.
func VerifyMultiproof(root common.Hash, mp *Multiproof) (*BinaryTrie, error) {
	if mp == nil {
		return nil, ErrProofMalformed
	}
	pos := 0
	n, rest, err := rebuild(mp.tokens, &pos)
	if err != nil {
		return nil, err
	}
	if len(rest) != 0 {
		return nil, ErrProofMalformed // trailing tokens: a second encoding
	}
	var got common.Hash
	if n != nil {
		got = n.hashAt(0)
	}
	if got != root {
		return nil, ErrProofMalformed
	}
	reader, err := trie.NewReader(common.Hash{}, common.Hash{}, nil)
	if err != nil {
		return nil, err
	}
	if n == nil {
		n = empty{}
	}
	return &BinaryTrie{
		root:   n,
		reader: reader,
		tracer: trie.NewPrevalueTracer(),
		ops:    newOpTracer(),
	}, nil
}

// rebuild consumes one preorder subtree, returning it and the tokens left.
// Every hash is recomputed from children; none is adopted from the proof.
func rebuild(toks []mpToken, pos *int) (binaryNode, []mpToken, error) {
	if len(toks) == 0 {
		return nil, nil, ErrProofMalformed
	}
	tok := toks[0]
	switch tok.kind {
	case mpStub:
		if tok.hash == (common.Hash{}) {
			return nil, nil, ErrProofMalformed // canonical branches have no empty child
		}
		return hashedNode(tok.hash), toks[1:], nil

	case mpBranch:
		at := *pos
		*pos = at + tok.prefix.n + 1
		left, rest, err := rebuild(toks[1:], pos)
		if err != nil {
			return nil, nil, err
		}
		*pos = at + tok.prefix.n + 1
		right, rest, err := rebuild(rest, pos)
		if err != nil {
			return nil, nil, err
		}
		if left == nil || right == nil {
			return nil, nil, ErrProofMalformed
		}
		return &branchNode{prefix: tok.prefix, left: left, right: right}, rest, nil

	case mpGroup:
		n, err := rebuildGroup(&tok, *pos)
		if err != nil {
			return nil, nil, err
		}
		return n, toks[1:], nil

	default:
		return nil, nil, ErrProofMalformed
	}
}

// rebuildGroup turns one stem token back into nodes: a plain group when every
// value is present, otherwise the branch-and-leaf structure the group hashes
// as, with the uncovered ranges standing in as their hashes.
func rebuildGroup(tok *mpToken, pos int) (binaryNode, error) {
	if err := validateStem(tok.stem); err != nil {
		return nil, err
	}
	subs := bitmapSubs(&tok.present)
	if len(subs) == 0 {
		return nil, ErrProofMalformed
	}
	covered := bitmapSubs(&tok.covered)
	if len(covered) != len(tok.values) {
		return nil, ErrProofMalformed
	}
	vals := make(map[byte][]byte, len(covered))
	for i, sub := range covered {
		if !bitmapHas(&tok.present, sub) {
			return nil, ErrProofMalformed // covered must be a subset of present
		}
		if len(tok.values[i]) != 32 {
			return nil, ErrProofMalformed
		}
		vals[sub] = tok.values[i]
	}
	// Fully covered: the group is representable as itself, and stays writable.
	if len(covered) == len(subs) {
		if len(tok.stubs) != 0 {
			return nil, ErrProofMalformed
		}
		out := make([][]byte, len(subs))
		for i, sub := range subs {
			out[i] = vals[sub]
		}
		return &groupNode{stem: tok.stem, subs: subs, vals: out}, nil
	}
	stubs := tok.stubs
	n, err := rebuildRange(tok.stem, subs, vals, 0, len(subs), 0, pos, 8*len(tok.stem), &stubs)
	if err != nil {
		return nil, err
	}
	if len(stubs) != 0 {
		return nil, ErrProofMalformed // unconsumed stubs: not a canonical encoding
	}
	return n, nil
}

// rebuildRange mirrors foldRange over the present sub-indices, materialising
// covered leaves and substituting a supplied hash for each uncovered range.
func rebuildRange(stem []byte, subs []byte, vals map[byte][]byte, i, j, from, extraLo, extraHi int, stubs *[]common.Hash) (binaryNode, error) {
	anyCovered := false
	for k := i; k < j; k++ {
		if _, ok := vals[subs[k]]; ok {
			anyCovered = true
			break
		}
	}
	if !anyCovered {
		if len(*stubs) == 0 {
			return nil, ErrProofMalformed
		}
		h := (*stubs)[0]
		*stubs = (*stubs)[1:]
		if h == (common.Hash{}) {
			return nil, ErrProofMalformed
		}
		return hashedNode(h), nil
	}
	if j-i == 1 {
		return &groupNode{stem: stem, subs: []byte{subs[i]}, vals: [][]byte{vals[subs[i]]}}, nil
	}
	b := 8 - bits.Len8(subs[i]^subs[j-1])
	m := i + 1
	for m < j && subs[m]>>(7-b)&1 == 0 {
		m++
	}
	var prefix bitstr
	if extraHi > extraLo {
		prefix = slice(stem, extraLo, extraHi-extraLo)
		for k := from; k < b; k++ {
			prefix = prefix.concat(subs[i]>>(7-k)&1, bitstr{})
		}
	} else {
		prefix = slice([]byte{subs[i]}, from, b-from)
	}
	left, err := rebuildRange(stem, subs, vals, i, m, b+1, 0, 0, stubs)
	if err != nil {
		return nil, err
	}
	right, err := rebuildRange(stem, subs, vals, m, j, b+1, 0, 0, stubs)
	if err != nil {
		return nil, err
	}
	return &branchNode{prefix: prefix, left: left, right: right}, nil
}

//
// Encoding
//

// Encode serialises the proof. Lengths are fixed by the structure, so the
// encoding is canonical: one tree and one key set have exactly one form.
func (mp *Multiproof) Encode() []byte {
	var out []byte
	for _, tok := range mp.tokens {
		switch tok.kind {
		case mpBranch:
			out = append(out, mpBranch)
			out = appendBitPrefix(out, tok.prefix)
		case mpStub:
			out = append(out, mpStub)
			out = append(out, tok.hash[:]...)
		case mpGroup:
			out = append(out, mpGroup, byte(len(tok.stem)))
			out = append(out, tok.stem...)
			out = append(out, tok.present[:]...)
			out = append(out, tok.covered[:]...)
			for _, v := range tok.values {
				out = append(out, v...)
			}
			for _, h := range tok.stubs {
				out = append(out, h[:]...)
			}
		}
	}
	return out
}

// DecodeMultiproof parses an encoded proof. Every length it reads is checked
// against what remains, and every field against the shape the structure
// implies, so a truncated or padded proof is rejected rather than walked.
func DecodeMultiproof(blob []byte) (*Multiproof, error) {
	mp := new(Multiproof)
	for len(blob) > 0 {
		kind := blob[0]
		blob = blob[1:]
		switch kind {
		case mpBranch:
			prefix, used, err := decodeBitPrefix(blob)
			if err != nil {
				return nil, err
			}
			blob = blob[used:]
			mp.tokens = append(mp.tokens, mpToken{kind: mpBranch, prefix: prefix})

		case mpStub:
			if len(blob) < 32 {
				return nil, errInvalidSerializedLength
			}
			var h common.Hash
			copy(h[:], blob[:32])
			blob = blob[32:]
			mp.tokens = append(mp.tokens, mpToken{kind: mpStub, hash: h})

		case mpGroup:
			if len(blob) < 1 {
				return nil, errInvalidSerializedLength
			}
			stemLen := int(blob[0])
			blob = blob[1:]
			if stemLen != AccountKeyLength-1 && stemLen != StorageKeyLength-1 {
				return nil, ErrNonConformantKey
			}
			if len(blob) < stemLen+64 {
				return nil, errInvalidSerializedLength
			}
			tok := mpToken{kind: mpGroup, stem: slices.Clone(blob[:stemLen])}
			blob = blob[stemLen:]
			copy(tok.present[:], blob[:32])
			copy(tok.covered[:], blob[32:64])
			blob = blob[64:]

			nvals := bitmapCount(&tok.covered)
			if len(blob) < 32*nvals {
				return nil, errInvalidSerializedLength
			}
			for i := 0; i < nvals; i++ {
				tok.values = append(tok.values, slices.Clone(blob[:32]))
				blob = blob[32:]
			}
			// The stub count is implied by the shape, so it is derived rather
			// than trusted: a proof cannot declare a different one.
			nstubs, err := stubCount(&tok)
			if err != nil {
				return nil, err
			}
			if len(blob) < 32*nstubs {
				return nil, errInvalidSerializedLength
			}
			for i := 0; i < nstubs; i++ {
				var h common.Hash
				copy(h[:], blob[:32])
				tok.stubs = append(tok.stubs, h)
				blob = blob[32:]
			}
			mp.tokens = append(mp.tokens, tok)

		default:
			return nil, errInvalidNodeTag
		}
	}
	return mp, nil
}

// stubCount derives how many hashes a stem token carries from its bitmaps.
func stubCount(tok *mpToken) (int, error) {
	subs := bitmapSubs(&tok.present)
	if len(subs) == 0 {
		return 0, ErrProofMalformed
	}
	covered := bitmapSubs(&tok.covered)
	for _, sub := range covered {
		if !bitmapHas(&tok.present, sub) {
			return 0, ErrProofMalformed
		}
	}
	if len(covered) == len(subs) {
		return 0, nil
	}
	if len(subs) == 1 {
		return 1, nil // uncovered single-value group: its own hash
	}
	n := 0
	countRange(subs, &tok.covered, 0, len(subs), &n)
	return n, nil
}

func countRange(subs []byte, covered *[32]byte, i, j int, out *int) {
	anyCovered := false
	for k := i; k < j; k++ {
		if bitmapHas(covered, subs[k]) {
			anyCovered = true
			break
		}
	}
	if !anyCovered {
		*out++
		return
	}
	if j-i == 1 {
		return
	}
	b := 8 - bits.Len8(subs[i]^subs[j-1])
	m := i + 1
	for m < j && subs[m]>>(7-b)&1 == 0 {
		m++
	}
	countRange(subs, covered, i, m, out)
	countRange(subs, covered, m, j, out)
}

// Size reports the encoded length.
func (mp *Multiproof) Size() int { return len(mp.Encode()) }
