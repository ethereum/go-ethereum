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

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/crypto"
)

// n is the byte length of a hash value, a Merkle node and PK.seed.
const n = 16

// node is a 16-byte hash value.
type node = [n]byte

// Address types. All but adrsWotsEnc are the FIPS 205 address types.
const (
	adrsWotsHash  uint32 = 0 // one step along a WOTS+ chain
	adrsWotsPk    uint32 = 1 // Merkle leaf of a one-time key, from its chain ends
	adrsTree      uint32 = 2 // Merkle node of the hypertree
	adrsForsTree  uint32 = 3 // FORS leaf or node
	adrsForsRoots uint32 = 4 // FORS public key, from the roots of its trees
	adrsWotsPrf   uint32 = 5 // start of one WOTS+ chain, derived from SK.seed
	adrsForsPrf   uint32 = 6 // secret of one FORS leaf, derived from SK.seed
	adrsWotsEnc   uint32 = 7 // WOTS+C digest a one-time key signs
)

// hasher owns one Keccak state and the input buffer of the tweakable hash.
// The buffer holds PK.seed (16 bytes), the 32-byte address and then the
// message, so every tweakable hash is a single Reset/Write/Read with no
// allocation. The largest message is the 64 chain ends of a WOTS+ key.
type hasher struct {
	kh     crypto.KeccakState
	buf    [48 + chains*n]byte
	out    [32]byte
	digest [64]byte // two Keccak outputs of H_msg
}

func newHasher() *hasher {
	return &hasher{kh: crypto.NewKeccakState()}
}

// Layout of buf[16:48], the address, as byte offsets into buf.
const (
	offLayer   = 16 // 4 bytes
	offTree    = 20 // 12 bytes, a big-endian 64-bit value in the last 8
	offType    = 32 // 4 bytes
	offKeyPair = 36 // 4 bytes
	offWord2   = 40 // 4 bytes: chain index or tree height
	offWord3   = 44 // 4 bytes: chain position or node index
	offMsg     = 48
)

func (h *hasher) setPkSeed(pkSeed []byte) {
	copy(h.buf[:n], pkSeed)
}

// setAdrs sets the layer, tree, type and key pair words and zeroes the rest.
func (h *hasher) setAdrs(layer uint32, tree uint64, typ, keyPair uint32) {
	binary.BigEndian.PutUint32(h.buf[offLayer:], layer)
	clear(h.buf[offTree : offTree+4])
	binary.BigEndian.PutUint64(h.buf[offTree+4:], tree)
	binary.BigEndian.PutUint32(h.buf[offType:], typ)
	binary.BigEndian.PutUint32(h.buf[offKeyPair:], keyPair)
	clear(h.buf[offWord2:offMsg])
}

func (h *hasher) setWord2(v uint32) {
	binary.BigEndian.PutUint32(h.buf[offWord2:], v)
}

func (h *hasher) setWord3(v uint32) {
	binary.BigEndian.PutUint32(h.buf[offWord3:], v)
}

// th is the tweakable hash Trunc_n(Keccak256(PK.seed || ADRS || M)) over the
// msgLen message bytes already placed at buf[offMsg:].
func (h *hasher) th(msgLen int) (out node) {
	h.kh.Reset()
	h.kh.Write(h.buf[:offMsg+msgLen])
	h.kh.Read(h.out[:])
	copy(out[:], h.out[:n])
	return out
}

// thNode hashes a single node under the current address.
func (h *hasher) thNode(v node) node {
	copy(h.buf[offMsg:], v[:])
	return h.th(n)
}

// thPair hashes the concatenation of two nodes under the current address.
func (h *hasher) thPair(left, right node) node {
	copy(h.buf[offMsg:], left[:])
	copy(h.buf[offMsg+n:], right[:])
	return h.th(2 * n)
}

// prf derives a secret value from SK.seed under the current address.
func (h *hasher) prf(skSeed []byte) node {
	copy(h.buf[offMsg:], skSeed)
	return h.th(SeedSize)
}

// sum is a plain Keccak-256 over the concatenation of the inputs. It is used
// for the key expansion and message hashes, which are not tweakable hashes.
func (h *hasher) sum(out []byte, data ...[]byte) {
	h.kh.Reset()
	for _, d := range data {
		h.kh.Write(d)
	}
	h.kh.Read(out)
}
