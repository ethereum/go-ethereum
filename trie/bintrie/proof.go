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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/blake3"
	"github.com/ethereum/go-ethereum/ethdb"
)

// Proofs are sets of node hash preimages keyed by their hash: branch
// preimages (tag ‖ encoded prefix ‖ child hashes) and leaf preimages (tag ‖
// key ‖ value). Group records never appear in proofs — their canonical
// leaf/branch structure is expanded instead, so a single-leaf proof does not
// haul a whole stem. Absence is proven by divergence: the proof ends at the
// branch whose prefix conflicts with the key, or at a leaf with a different
// key; canonical branches always have two children, so these witnesses are
// exhaustive.

// prove walks towards key, writing every canonical node preimage on the
// path into proofDb keyed by its hash.
func (t *BinaryTrie) prove(key []byte, proofDb ethdb.KeyValueWriter) error {
	if err := validateKey(key); err != nil {
		return err
	}
	n, err := t.proveWalk(t.root, key, bitstr{}, 0, proofDb)
	if err != nil {
		return err
	}
	t.root = n
	return nil
}

func (t *BinaryTrie) proveWalk(n binaryNode, key []byte, walk bitstr, pos int, proofDb ethdb.KeyValueWriter) (binaryNode, error) {
	switch nn := n.(type) {
	case empty:
		return n, nil
	case hashedNode:
		resolved, err := t.resolve(nn, walk)
		if err != nil {
			return n, err
		}
		return t.proveWalk(resolved, key, walk, pos, proofDb)
	case *groupNode:
		// Expand the group's canonical structure along (or diverging from)
		// the key.
		nn.hashAt(pos) // ensure cache; the expansion recomputes preimages
		return n, proveGroup(nn, key, pos, proofDb)
	case *branchNode:
		child := pos + nn.prefix.n + 1
		left, right := nn.left.hashAt(child), nn.right.hashAt(child)
		preimage := make([]byte, 0, 3+len(nn.prefix.b)+64)
		preimage = append(preimage, tagBranch)
		preimage = appendBitPrefix(preimage, nn.prefix)
		preimage = append(preimage, left[:]...)
		preimage = append(preimage, right[:]...)
		if err := proofDb.Put(hashOf(preimage).Bytes(), preimage); err != nil {
			return n, err
		}
		m := nn.prefix.matchKey(key, pos)
		if m < nn.prefix.n || pos+nn.prefix.n >= 8*len(key) {
			return n, nil // divergence witnessed by this branch
		}
		split := pos + nn.prefix.n
		bit := bitAt(key, split)
		childWalk := walk.append(nn.prefix).concat(bit, bitstr{})
		if bit == 0 {
			c, err := t.proveWalk(nn.left, key, childWalk, split+1, proofDb)
			nn.left = c
			return n, err
		}
		c, err := t.proveWalk(nn.right, key, childWalk, split+1, proofDb)
		nn.right = c
		return n, err
	default:
		return n, fmt.Errorf("bintrie: unknown node type %T", n)
	}
}

// proveGroup writes the canonical node preimages inside a group along the
// path towards key (or its divergence witness).
func proveGroup(g *groupNode, key []byte, pos int, proofDb ethdb.KeyValueWriter) error {
	if len(g.subs) == 1 {
		return putLeaf(g, 0, proofDb)
	}
	stemBits := 8 * len(g.stem)
	i, j, from := 0, len(g.subs), 0
	extraLo, extraHi := pos, stemBits
	sameStem := bytes.Equal(g.stem, key[:len(key)-1])
	for {
		if j-i == 1 {
			return putLeaf(g, i, proofDb)
		}
		b := 8 - len8(g.subs[i]^g.subs[j-1])
		m := i + 1
		for m < j && g.subs[m]>>(7-b)&1 == 0 {
			m++
		}
		// Emit this internal branch's preimage.
		var prefix bitstr
		if extraHi > extraLo {
			prefix = slice(g.stem, extraLo, extraHi-extraLo)
			for k := from; k < b; k++ {
				prefix = prefix.concat(g.subs[i]>>(7-k)&1, bitstr{})
			}
		} else {
			prefix = slice([]byte{g.subs[i]}, from, b-from)
		}
		left := g.foldRange(i, m, b+1, 0, 0)
		right := g.foldRange(m, j, b+1, 0, 0)
		preimage := make([]byte, 0, 3+len(prefix.b)+64)
		preimage = append(preimage, tagBranch)
		preimage = appendBitPrefix(preimage, prefix)
		preimage = append(preimage, left[:]...)
		preimage = append(preimage, right[:]...)
		if err := proofDb.Put(hashOf(preimage).Bytes(), preimage); err != nil {
			return err
		}
		if !sameStem {
			return nil // the top branch's stem-bit prefix witnesses divergence
		}
		sub := key[len(key)-1]
		if sub>>(7-b)&1 == 0 {
			j = m
		} else {
			i = m
		}
		from, extraLo, extraHi = b+1, 0, 0
	}
}

func putLeaf(g *groupNode, i int, proofDb ethdb.KeyValueWriter) error {
	preimage := make([]byte, 0, 1+len(g.stem)+1+32)
	preimage = append(preimage, tagLeaf)
	preimage = append(preimage, g.stem...)
	preimage = append(preimage, g.subs[i])
	preimage = append(preimage, g.vals[i]...)
	return proofDb.Put(hashOf(preimage).Bytes(), preimage)
}

func hashOf(preimage []byte) common.Hash {
	return common.Hash(blake3.Sum256(preimage))
}

func len8(x byte) int {
	n := 0
	for x != 0 {
		x >>= 1
		n++
	}
	return n
}

// VerifyProof checks a merkle proof for key against the given root hash. It
// returns the proven value, or nil if the proof shows the key absent. An
// error means the proof is malformed or insufficient.
func VerifyProof(root common.Hash, key []byte, proofDb ethdb.KeyValueReader) ([]byte, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	if root == (common.Hash{}) {
		return nil, nil // empty tree: everything absent
	}
	want, pos := root, 0
	for {
		preimage, err := proofDb.Get(want.Bytes())
		if err != nil || preimage == nil {
			return nil, errors.New("bintrie: proof node missing")
		}
		if hashOf(preimage) != want {
			return nil, errors.New("bintrie: proof node hash mismatch")
		}
		if len(preimage) == 0 {
			return nil, errInvalidNodeTag
		}
		switch preimage[0] {
		case tagLeaf:
			// Derive the key length before slicing by it. A short preimage
			// gives a negative bound, and the hash check above does not
			// constrain the shape of what it hashed.
			//
			// The three zone lengths are compared one at a time rather than
			// listed as switch cases because two of them are equal today: a
			// switch naming all three would not compile, and one naming only
			// two silently stops covering code keys the moment they diverge.
			keyLen := len(preimage) - 1 - 32
			if keyLen != AccountKeyLength && keyLen != CodeKeyLength && keyLen != StorageKeyLength {
				return nil, errInvalidSerializedLength
			}
			leafKey := preimage[1 : 1+keyLen]
			value := preimage[1+keyLen:]
			if err := validateKey(leafKey); err != nil {
				return nil, err
			}
			if bytes.Equal(leafKey, key) {
				return append([]byte{}, value...), nil
			}
			// The leaf's key must be consistent with the walked path, and
			// its divergence from the queried key proves absence.
			if commonPrefixLen(leafKey, key, 0) < pos {
				return nil, errors.New("bintrie: proof leaf off-path")
			}
			return nil, nil
		case tagBranch:
			prefix, consumed, err := decodeBitPrefix(preimage[1:])
			if err != nil {
				return nil, err
			}
			rest := preimage[1+consumed:]
			if len(rest) != 64 {
				return nil, errInvalidSerializedLength
			}
			m := prefix.matchKey(key, pos)
			if m < prefix.n || pos+prefix.n >= 8*len(key) {
				return nil, nil // prefix conflict: proven absent
			}
			pos += prefix.n
			if bitAt(key, pos) == 0 {
				copy(want[:], rest[:32])
			} else {
				copy(want[:], rest[32:])
			}
			pos++
			if want == (common.Hash{}) {
				return nil, errors.New("bintrie: empty child in proof branch")
			}
		default:
			return nil, errInvalidNodeTag
		}
	}
}

// proofList implements ethdb.KeyValueWriter, collecting proof preimages in
// insertion order.
type proofList struct {
	list [][]byte
}

func (p *proofList) Put(_ []byte, value []byte) error {
	p.list = append(p.list, append([]byte{}, value...))
	return nil
}

func (p *proofList) Delete([]byte) error { return errors.New("not supported") }
