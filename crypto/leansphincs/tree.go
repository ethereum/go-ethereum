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

// fold climbs from leaf j along the authentication path to the root of a
// Merkle tree. The path holds one sibling per height, lowest first. The
// parent of the node at height z gets height z+1 and node index
// base>>(z+1) + j>>1, where base is the index of the tree's first leaf
// (k<<forsHeight for FORS tree k, 0 for a hypertree tree).
func (h *hasher) fold(layer uint32, tree uint64, typ, keyPair, base, j uint32, cur node, path []byte) node {
	for z := uint32(0); z < uint32(len(path)/n); z++ {
		sibling := node(path[z*n : (z+1)*n])
		h.setAdrs(layer, tree, typ, keyPair)
		h.setWord2(z + 1)
		h.setWord3(base>>(z+1) + j>>1)
		if j&1 == 0 {
			cur = h.thPair(cur, sibling)
		} else {
			cur = h.thPair(sibling, cur)
		}
		j >>= 1
	}
	return cur
}

// xmssNode computes node j at height z of the given hypertree tree from
// SK.seed. If path is not nil, it also writes the authentication path of
// leaf into it, one sibling per height below the root.
func (h *hasher) xmssNode(skSeed []byte, layer uint32, tree uint64, z, j, leaf uint32, path []byte) node {
	var nd node
	if z == 0 {
		nd = h.wotsPkGen(skSeed, layer, tree, j)
	} else {
		left := h.xmssNode(skSeed, layer, tree, z-1, 2*j, leaf, path)
		right := h.xmssNode(skSeed, layer, tree, z-1, 2*j+1, leaf, path)
		h.setAdrs(layer, tree, adrsTree, 0)
		h.setWord2(z)
		h.setWord3(j)
		nd = h.thPair(left, right)
	}
	if path != nil && z < uint32(len(path)/n) && j == (leaf>>z)^1 {
		copy(path[z*n:], nd[:])
	}
	return nd
}
