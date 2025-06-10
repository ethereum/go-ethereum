// Copyright 2025 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

// Package p256verify implements an EVM precompile to verify P256 ECDSA
// signatures, as described in RIP-7212.
package p256verify

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"math/big"

	"github.com/ava-labs/libevm/params"
)

// Precompile implements ECDSA verification on the P256 curve, as defined by
// [RIP-7212].
//
// [RIP-7212]: https://github.com/ethereum/RIPs/blob/1f55794f65caa4c4bb2b8d9bda7d713b8c734157/RIPS/rip-7212.md
type Precompile struct{}

// RequiredGas returns [params.P256VerifyGas].
func (Precompile) RequiredGas([]byte) uint64 {
	return params.P256VerifyGas
}

const (
	wordLen  = 32
	inputLen = 5 * wordLen
)

type input [inputLen]byte

type index int

const (
	hashPos index = iota * wordLen
	rPos
	sPos
	xPos
	yPos
)

// Run parses and verifies the signature. On success it returns a 32-byte
// big-endian representation of the number 1, otherwise it returns an empty
// slice. The returned error is always nil.
func (Precompile) Run(sig []byte) ([]byte, error) {
	if len(sig) != inputLen || !(*input)(sig).verify() {
		return nil, nil
	}
	return bigEndianOne(), nil
}

func bigEndianOne() []byte {
	return []byte{wordLen - 1: 1}
}

func (in *input) verify() bool {
	key, ok := in.pubkey()
	if !ok {
		return false
	}
	return ecdsa.Verify(key, in.word(hashPos), in.bigWord(rPos), in.bigWord(sPos))
}

func (in *input) pubkey() (*ecdsa.PublicKey, bool) {
	x := in.bigWord(xPos)
	y := in.bigWord(yPos)

	// There is no need to explicitly check for the point at infinity because
	// [elliptic.Curve] documentation states that it's not on the curve and the
	// check would therefore be performed twice.
	// See https://cs.opensource.google/go/go/+/refs/tags/go1.24.3:src/crypto/elliptic/nistec.go;l=132
	curve := elliptic.P256()
	if !curve.IsOnCurve(x, y) {
		return nil, false
	}
	return &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}, true
}

func (in *input) word(i index) []byte {
	return in[i : i+wordLen]
}

func (in *input) bigWord(i index) *big.Int {
	return new(big.Int).SetBytes(in.word(i))
}

// Pack packs the arguments into a byte slice compatible with [Precompile.Run].
// It does NOT perform any validation on its inputs and therefore may panic if,
// for example, a [big.Int] with >256 bits is received. Keys and signatures
// generated with [elliptic.GenerateKey] and [ecdsa.Sign] are valid inputs.
func Pack(hash [32]byte, r, s *big.Int, key *ecdsa.PublicKey) []byte {
	var in input

	copy(in.word(hashPos), hash[:])

	r.FillBytes(in.word(rPos))
	s.FillBytes(in.word(sPos))

	key.X.FillBytes(in.word(xPos))
	key.Y.FillBytes(in.word(yPos))

	return in[:]
}
