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

//go:build zisk

package vm

import (
	"errors"

	"github.com/0xPolygonHermez/zisk/ziskasm/lang/go"
)

var errInvalidCurvePoint = errors.New("invalid curve point")

// -- BN254 (EIP-196, EIP-197) ------------------------------------------------

// runBn256Add implements the Bn256Add precompile, referenced by both
// Byzantium and Istanbul operations.
func runBn256Add(input []byte) ([]byte, error) {
	var p1, p2 [64]byte
	copy(p1[:], getData(input, 0, 64))
	copy(p2[:], getData(input, 64, 64))
	sum, err := zkvm.Bn254G1Add(&p1, &p2)
	if err != nil {
		return nil, errInvalidCurvePoint
	}
	return sum[:], nil
}

// runBn256ScalarMul implements the Bn256ScalarMul precompile, referenced by
// both Byzantium and Istanbul operations.
func runBn256ScalarMul(input []byte) ([]byte, error) {
	var (
		p [64]byte
		k [32]byte
	)
	copy(p[:], getData(input, 0, 64))
	copy(k[:], getData(input, 64, 32))
	product, err := zkvm.Bn254G1Mul(&p, &k)
	if err != nil {
		return nil, errInvalidCurvePoint
	}
	return product[:], nil
}

// runBn256Pairing implements the Bn256Pairing precompile, referenced by both
// Byzantium and Istanbul operations.
func runBn256Pairing(input []byte) ([]byte, error) {
	// Handle some corner cases cheaply
	if len(input)%192 > 0 {
		return nil, errBadPairingInput
	}
	pairs := make([]zkvm.Bn254PairingPair, len(input)/192)
	for i := range pairs {
		copy(pairs[i].G1[:], input[192*i:])
		copy(pairs[i].G2[:], input[192*i+64:])
	}
	ok, err := zkvm.Bn254Pairing(pairs)
	if err != nil {
		return nil, errInvalidCurvePoint
	}
	if ok {
		return true32Byte, nil
	}
	return false32Byte, nil
}

// -- BLS12-381 (EIP-2537) ----------------------------------------------------
//
// The EIP encodes a field element as 64 bytes, 16 zero bytes then the 48-byte
// big-endian value, and the routines take the 48-byte values: a G1 point is
// 128 bytes to the EIP and 96 to the routines, a G2 point 256 and 192.

// decodeBLS12381FieldElement checks the encoding of a field element and
// returns its 48-byte value.
func decodeBLS12381FieldElement(in []byte) ([48]byte, error) {
	var res [48]byte
	if len(in) != 64 {
		return res, errors.New("invalid field element length")
	}
	for i := 0; i < 16; i++ {
		if in[i] != byte(0x00) {
			return res, errBLS12381InvalidFieldElementTopBytes
		}
	}
	copy(res[:], in[16:])
	return res, nil
}

// decodeBLS12381Point strips the padding of a point of n field elements
// (2 for G1, 4 for G2) into out.
func decodeBLS12381Point(in []byte, out []byte) error {
	n := len(out) / 48
	if len(in) != 64*n {
		return errors.New("invalid point length")
	}
	for i := 0; i < n; i++ {
		fe, err := decodeBLS12381FieldElement(in[64*i : 64*i+64])
		if err != nil {
			return err
		}
		copy(out[48*i:], fe[:])
	}
	return nil
}

// encodeBLS12381Point pads the field elements of a point back to the EIP's
// encoding.
func encodeBLS12381Point(point []byte) []byte {
	n := len(point) / 48
	out := make([]byte, 64*n)
	for i := 0; i < n; i++ {
		copy(out[64*i+16:], point[48*i:48*i+48])
	}
	return out
}

// runBls12381G1Add implements the bls12381G1Add precompile.
func runBls12381G1Add(input []byte) ([]byte, error) {
	if len(input) != 256 {
		return nil, errBLS12381InvalidInputLength
	}
	var p0, p1 [96]byte
	if err := decodeBLS12381Point(input[:128], p0[:]); err != nil {
		return nil, err
	}
	if err := decodeBLS12381Point(input[128:], p1[:]); err != nil {
		return nil, err
	}
	sum, err := zkvm.Bls12G1Add(&p0, &p1)
	if err != nil {
		return nil, errInvalidCurvePoint
	}
	return encodeBLS12381Point(sum[:]), nil
}

// runBls12381G1MultiExp implements the bls12381G1MultiExp precompile.
func runBls12381G1MultiExp(input []byte) ([]byte, error) {
	k := len(input) / 160
	if len(input) == 0 || len(input)%160 != 0 {
		return nil, errBLS12381InvalidInputLength
	}
	pairs := make([]zkvm.Bls12G1MsmPair, k)
	for i := 0; i < k; i++ {
		off := 160 * i
		if err := decodeBLS12381Point(input[off:off+128], pairs[i].Point[:]); err != nil {
			return nil, err
		}
		copy(pairs[i].Scalar[:], input[off+128:off+160])
	}
	sum, err := zkvm.Bls12G1Msm(pairs)
	if err != nil {
		return nil, errBLS12381G1PointSubgroup
	}
	return encodeBLS12381Point(sum[:]), nil
}

// runBls12381G2Add implements the bls12381G2Add precompile.
func runBls12381G2Add(input []byte) ([]byte, error) {
	if len(input) != 512 {
		return nil, errBLS12381InvalidInputLength
	}
	var p0, p1 [192]byte
	if err := decodeBLS12381Point(input[:256], p0[:]); err != nil {
		return nil, err
	}
	if err := decodeBLS12381Point(input[256:], p1[:]); err != nil {
		return nil, err
	}
	sum, err := zkvm.Bls12G2Add(&p0, &p1)
	if err != nil {
		return nil, errInvalidCurvePoint
	}
	return encodeBLS12381Point(sum[:]), nil
}

// runBls12381G2MultiExp implements the bls12381G2MultiExp precompile.
func runBls12381G2MultiExp(input []byte) ([]byte, error) {
	k := len(input) / 288
	if len(input) == 0 || len(input)%288 != 0 {
		return nil, errBLS12381InvalidInputLength
	}
	pairs := make([]zkvm.Bls12G2MsmPair, k)
	for i := 0; i < k; i++ {
		off := 288 * i
		if err := decodeBLS12381Point(input[off:off+256], pairs[i].Point[:]); err != nil {
			return nil, err
		}
		copy(pairs[i].Scalar[:], input[off+256:off+288])
	}
	sum, err := zkvm.Bls12G2Msm(pairs)
	if err != nil {
		return nil, errBLS12381G2PointSubgroup
	}
	return encodeBLS12381Point(sum[:]), nil
}

// runBls12381Pairing implements the bls12381Pairing precompile.
func runBls12381Pairing(input []byte) ([]byte, error) {
	k := len(input) / 384
	if len(input) == 0 || len(input)%384 != 0 {
		return nil, errBLS12381InvalidInputLength
	}
	pairs := make([]zkvm.Bls12PairingPair, k)
	for i := 0; i < k; i++ {
		off := 384 * i
		if err := decodeBLS12381Point(input[off:off+128], pairs[i].G1[:]); err != nil {
			return nil, err
		}
		if err := decodeBLS12381Point(input[off+128:off+384], pairs[i].G2[:]); err != nil {
			return nil, err
		}
	}
	ok, err := zkvm.Bls12Pairing(pairs)
	if err != nil {
		return nil, errInvalidCurvePoint
	}
	out := make([]byte, 32)
	if ok {
		out[31] = 1
	}
	return out, nil
}

// runBls12381MapG1 implements the bls12381MapG1 precompile.
func runBls12381MapG1(input []byte) ([]byte, error) {
	if len(input) != 64 {
		return nil, errBLS12381InvalidInputLength
	}
	fe, err := decodeBLS12381FieldElement(input)
	if err != nil {
		return nil, err
	}
	point, err := zkvm.Bls12MapFpToG1(&fe)
	if err != nil {
		return nil, errInvalidCurvePoint
	}
	return encodeBLS12381Point(point[:]), nil
}

// runBls12381MapG2 implements the bls12381MapG2 precompile.
func runBls12381MapG2(input []byte) ([]byte, error) {
	if len(input) != 128 {
		return nil, errBLS12381InvalidInputLength
	}
	var fe [96]byte
	if err := decodeBLS12381Point(input, fe[:]); err != nil {
		return nil, err
	}
	point, err := zkvm.Bls12MapFp2ToG2(&fe)
	if err != nil {
		return nil, errInvalidCurvePoint
	}
	return encodeBLS12381Point(point[:]), nil
}
