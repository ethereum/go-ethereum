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

package leansphincs

// Every bottom leaf idx has its own FORS instance. Its addresses use the
// bottom one-time key: layer 0, that key's tree, and its leaf as key pair.
// Node j at height z of FORS tree k has node index k<<(forsHeight-z) + j.

// forsPkFromSig recovers the FORS public key of instance idx from the 24
// openings in fsig, one per tree: the leaf secret and the authentication
// path, for the leaves selected by the message digest.
func (h *hasher) forsPkFromSig(fsig []byte, idx uint64, leaves *[forsTrees]uint32) node {
	tree, leaf := otsKey(0, idx)
	var roots [forsTrees * n]byte
	for k := range uint32(forsTrees) {
		opening := fsig[k*forsOpeningSize : (k+1)*forsOpeningSize]
		s := leaves[k]
		h.setAdrs(0, tree, adrsForsTree, leaf)
		h.setWord3(k<<forsHeight + s)
		y := h.thNode(node(opening[:n]))
		root := h.fold(0, tree, adrsForsTree, leaf, k<<forsHeight, s, y, opening[n:])
		copy(roots[k*n:], root[:])
	}
	h.setAdrs(0, tree, adrsForsRoots, leaf)
	copy(h.buf[offMsg:], roots[:])
	return h.th(len(roots))
}

// forsNode computes node j at height z of FORS tree k from SK.seed. While
// it does so, it writes the opening of leaf s into opening: the leaf secret
// followed by the sibling at every height below the root.
func (h *hasher) forsNode(skSeed []byte, tree uint64, leaf, k, z, j, s uint32, opening []byte) node {
	var nd node
	if z == 0 {
		h.setAdrs(0, tree, adrsForsPrf, leaf)
		h.setWord3(k<<forsHeight + j)
		sk := h.prf(skSeed)
		if j == s {
			copy(opening, sk[:])
		}
		h.setAdrs(0, tree, adrsForsTree, leaf)
		h.setWord3(k<<forsHeight + j)
		nd = h.thNode(sk)
	} else {
		left := h.forsNode(skSeed, tree, leaf, k, z-1, 2*j, s, opening)
		right := h.forsNode(skSeed, tree, leaf, k, z-1, 2*j+1, s, opening)
		h.setAdrs(0, tree, adrsForsTree, leaf)
		h.setWord2(z)
		h.setWord3(k<<(forsHeight-z) + j)
		nd = h.thPair(left, right)
	}
	if z < forsHeight && j == (s>>z)^1 {
		copy(opening[n+z*n:], nd[:])
	}
	return nd
}
