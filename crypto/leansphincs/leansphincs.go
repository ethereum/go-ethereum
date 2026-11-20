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

// Package leansphincs implements leanSPHINCS, the stateless hash-based
// post-quantum signature scheme proposed for Ethereum transactions.
//
// leanSPHINCS is SLH-DSA (FIPS 205) over Keccak-256 truncated to 16 bytes,
// with a WOTS+C one-time signature, a FORS few-time signature of 24 trees of
// height 10, and a three-layer hypertree of heights 8, 8 and 10. Signing is
// deterministic. One key must not sign more than 2^30 messages; the package
// does not count signatures.
//
// The reference is the Lean specification in ethereum/EthLib, module
// EthLib.Cryptography.Sphincs, and the Rust implementation in
// leanEthereum/leanVM, crate signatures.
package leansphincs

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// Sizes of the scheme's byte strings.
const (
	SeedSize      = 32    // secret seed; everything else derives from it
	MessageSize   = 32    // the message is a 256-bit hash
	PublicKeySize = 2 * n // PK.seed || PK.root
	SignatureSize = 7740
)

const (
	seedTag = "leanSPHINCS"

	chains      = 64  // WOTS+ chains per one-time key (8n / w with w = 2)
	chainLength = 4   // positions per chain (2^w)
	targetSum   = 128 // digit sum every WOTS+C codeword must have
	counterLen  = 4   // WOTS+C counter, big-endian

	forsTrees  = 24 // FORS trees per instance
	forsHeight = 10 // height of each FORS tree

	layers = 3

	forsOpeningSize = n + forsHeight*n                          // secret and path of one FORS tree
	sigForsOffset   = n                                         // after the randomizer
	sigLayersOffset = sigForsOffset + forsTrees*forsOpeningSize // after the FORS openings
)

// layerHeights are the tree heights of the hypertree layers, bottom first.
var layerHeights = [layers]uint32{8, 8, 10}

// layerSigSize is the size of a layer signature: counter, chain values, path.
func layerSigSize(layer int) int {
	return counterLen + chains*n + int(layerHeights[layer])*n
}

var (
	errSeedSize    = errors.New("leansphincs: seed must be 32 bytes")
	errMessageSize = errors.New("leansphincs: message must be 32 bytes")
	errCounter     = errors.New("leansphincs: no WOTS+C counter encodes the message")
)

// Verify reports whether sig is a valid leanSPHINCS signature of msg under
// the public key pub. It returns false for any malformed input and never
// panics.
func Verify(pub, msg, sig []byte) bool {
	if len(pub) != PublicKeySize || len(msg) != MessageSize || len(sig) != SignatureSize {
		return false
	}
	return verify(newHasher(), pub, msg, sig)
}

func verify(h *hasher, pub, msg, sig []byte) bool {
	h.setPkSeed(pub[:n])
	leaves, idx := splitDigest(h.hMsg(sig[:n], pub, msg))

	m := h.forsPkFromSig(sig[sigForsOffset:sigLayersOffset], idx, &leaves)
	off := sigLayersOffset
	for l := range layers {
		size := layerSigSize(l)
		var ok bool
		if m, ok = h.layerRoot(l, idx, m, sig[off:off+size]); !ok {
			return false
		}
		off += size
	}
	return bytes.Equal(m[:], pub[n:])
}

// layerRoot recovers the root of the layer tree that holds the one-time key
// of idx, from the WOTS+C signature of m and the authentication path.
func (h *hasher) layerRoot(layer int, idx uint64, m node, lsig []byte) (node, bool) {
	tree, leaf := otsKey(layer, idx)
	x, ok := h.wotsLeafFromSig(uint32(layer), tree, leaf, m, lsig[:counterLen+chains*n])
	if !ok {
		return node{}, false
	}
	return h.fold(uint32(layer), tree, adrsTree, 0, 0, leaf, x, lsig[counterLen+chains*n:]), true
}

// KeyGen derives the public key from a 32-byte secret seed.
func KeyGen(seed []byte) ([]byte, error) {
	if len(seed) != SeedSize {
		return nil, errSeedSize
	}
	h := newHasher()
	pkSeed, _ := h.expandKey(seed)
	h.setPkSeed(pkSeed[:])
	root := h.xmssNode(seed, layers-1, 0, layerHeights[layers-1], 0, 0, nil)
	return append(pkSeed[:], root[:]...), nil
}

// Sign returns the deterministic leanSPHINCS signature of the 32-byte msg
// under the key derived from seed.
func Sign(seed, msg []byte) ([]byte, error) {
	if len(seed) != SeedSize {
		return nil, errSeedSize
	}
	if len(msg) != MessageSize {
		return nil, errMessageSize
	}
	h := newHasher()
	pkSeed, skPrf := h.expandKey(seed)
	h.setPkSeed(pkSeed[:])
	// shortcut: the top tree is hashed here for the root and again below for
	// its authentication path, cache it if signing ever becomes hot.
	root := h.xmssNode(seed, layers-1, 0, layerHeights[layers-1], 0, 0, nil)
	var pub [PublicKeySize]byte
	copy(pub[:n], pkSeed[:])
	copy(pub[n:], root[:])

	sig := make([]byte, SignatureSize)
	r := h.prfMsg(skPrf[:], pkSeed[:], msg)
	copy(sig, r[:])
	leaves, idx := splitDigest(h.hMsg(r[:], pub[:], msg))

	fs := sig[sigForsOffset:sigLayersOffset]
	tree, leaf := otsKey(0, idx)
	for k := range uint32(forsTrees) {
		h.forsNode(seed, tree, leaf, k, forsHeight, 0, leaves[k], fs[k*forsOpeningSize:(k+1)*forsOpeningSize])
	}
	m := h.forsPkFromSig(fs, idx, &leaves)

	off := sigLayersOffset
	for l := range layers {
		tree, leaf := otsKey(l, idx)
		lsig := sig[off : off+layerSigSize(l)]
		if !h.wotsSign(seed, uint32(l), tree, leaf, m, lsig[:counterLen+chains*n]) {
			return nil, errCounter
		}
		m = h.xmssNode(seed, uint32(l), tree, layerHeights[l], 0, leaf, lsig[counterLen+chains*n:])
		off += len(lsig)
	}
	return sig, nil
}

// expandKey derives PK.seed and SK.prf from the secret seed.
func (h *hasher) expandKey(seed []byte) (pkSeed node, skPrf [32]byte) {
	var out [32]byte
	h.sum(out[:], []byte(seedTag), seed, []byte{0})
	copy(pkSeed[:], out[:n])
	h.sum(skPrf[:], []byte(seedTag), seed, []byte{1})
	return pkSeed, skPrf
}

// prfMsg derives the randomizer. Signing is deterministic: opt_rand is
// PK.seed, and the message is framed as in FIPS 205 with the scheme name as
// context.
func (h *hasher) prfMsg(skPrf, pkSeed, msg []byte) (r node) {
	var out [32]byte
	h.sum(out[:], skPrf, pkSeed, []byte{0, byte(len(seedTag))}, []byte(seedTag), msg)
	copy(r[:], out[:n])
	return r
}

// hMsg computes the 34-byte message digest in counter mode:
// Keccak256(R || PK.seed || PK.root || M || BE32(i)) for i = 0, 1.
// The digest lives in h until the next call.
func (h *hasher) hMsg(r, pub, msg []byte) *[64]byte {
	for i := range 2 {
		clear(h.out[:4])
		h.out[3] = byte(i)
		h.sum(h.digest[32*i:32*i+32], r, pub, msg, h.out[:4])
	}
	return &h.digest
}

// splitDigest reads the 24 FORS leaf indices, each 10 bits, from the first
// 30 digest bytes, then the 18-bit tree index and the 8-bit leaf index that
// together select the bottom leaf idx.
func splitDigest(d *[64]byte) (leaves [forsTrees]uint32, idx uint64) {
	for k := range leaves {
		off := forsHeight * k
		v := binary.BigEndian.Uint32(d[off/8:])
		leaves[k] = (v >> (32 - forsHeight - off%8)) & (1<<forsHeight - 1)
	}
	tree := (uint64(d[30])<<16 | uint64(d[31])<<8 | uint64(d[32])) & (1<<18 - 1)
	return leaves, tree<<layerHeights[0] | uint64(d[33])
}

// otsKey returns the tree and leaf of the one-time key on the given layer
// that lies above bottom leaf idx.
func otsKey(layer int, idx uint64) (tree uint64, leaf uint32) {
	var below uint32
	for l := range layer {
		below += layerHeights[l]
	}
	height := layerHeights[layer]
	return idx >> (below + height), uint32(idx>>below) & (1<<height - 1)
}
