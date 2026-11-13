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

import "encoding/binary"

// encode hashes the message and counter under the WOTS+C encoding address
// and reads the result as 64 base-4 digits, most significant first. The
// digits form a codeword only if they sum to targetSum.
func (h *hasher) encode(layer uint32, tree uint64, leaf uint32, m node, counter []byte) (x [chains]uint8, ok bool) {
	h.setAdrs(layer, tree, adrsWotsEnc, leaf)
	copy(h.buf[offMsg:], m[:])
	copy(h.buf[offMsg+n:], counter[:counterLen])
	d := h.th(n + counterLen)

	sum := 0
	for i := range x {
		x[i] = d[i/4] >> (6 - 2*(i%4)) & 3
		sum += int(x[i])
	}
	return x, sum == targetSum
}

// chain walks chain i for steps steps from position start.
func (h *hasher) chain(layer uint32, tree uint64, leaf, i uint32, start, steps uint8, v node) node {
	for j := start; j < start+steps; j++ {
		h.setAdrs(layer, tree, adrsWotsHash, leaf)
		h.setWord2(i)
		h.setWord3(uint32(j))
		v = h.thNode(v)
	}
	return v
}

// wotsLeafFromSig recovers the Merkle leaf of the one-time key from the
// WOTS+C signature of m: the counter and 64 chain values.
func (h *hasher) wotsLeafFromSig(layer uint32, tree uint64, leaf uint32, m node, wsig []byte) (node, bool) {
	x, ok := h.encode(layer, tree, leaf, m, wsig[:counterLen])
	if !ok {
		return node{}, false
	}
	var ends [chains * n]byte
	for i := range uint32(chains) {
		v := node(wsig[counterLen+i*n : counterLen+(i+1)*n])
		e := h.chain(layer, tree, leaf, i, x[i], chainLength-1-x[i], v)
		copy(ends[i*n:], e[:])
	}
	return h.leafHash(layer, tree, leaf, &ends), true
}

// leafHash hashes the 64 chain ends into the Merkle leaf of the one-time key.
func (h *hasher) leafHash(layer uint32, tree uint64, leaf uint32, ends *[chains * n]byte) node {
	h.setAdrs(layer, tree, adrsWotsPk, leaf)
	copy(h.buf[offMsg:], ends[:])
	return h.th(len(ends))
}

// chainStart derives the secret start of chain i from SK.seed.
func (h *hasher) chainStart(skSeed []byte, layer uint32, tree uint64, leaf, i uint32) node {
	h.setAdrs(layer, tree, adrsWotsPrf, leaf)
	h.setWord2(i)
	return h.prf(skSeed)
}

// wotsPkGen computes the Merkle leaf of the one-time key from SK.seed.
func (h *hasher) wotsPkGen(skSeed []byte, layer uint32, tree uint64, leaf uint32) node {
	var ends [chains * n]byte
	for i := range uint32(chains) {
		e := h.chain(layer, tree, leaf, i, 0, chainLength-1, h.chainStart(skSeed, layer, tree, leaf, i))
		copy(ends[i*n:], e[:])
	}
	return h.leafHash(layer, tree, leaf, &ends)
}

// wotsSign writes the counter and chain values signing m into wsig. It
// returns false if no 32-bit counter encodes m as a codeword.
func (h *hasher) wotsSign(skSeed []byte, layer uint32, tree uint64, leaf uint32, m node, wsig []byte) bool {
	for c := range uint64(1 << 32) {
		binary.BigEndian.PutUint32(wsig, uint32(c))
		x, ok := h.encode(layer, tree, leaf, m, wsig[:counterLen])
		if !ok {
			continue
		}
		for i := range uint32(chains) {
			y := h.chain(layer, tree, leaf, i, 0, x[i], h.chainStart(skSeed, layer, tree, leaf, i))
			copy(wsig[counterLen+i*n:], y[:])
		}
		return true
	}
	return false
}
