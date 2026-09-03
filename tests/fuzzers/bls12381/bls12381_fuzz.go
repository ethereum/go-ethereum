// Copyright 2021 The go-ethereum Authors
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

//go:build cgo
// +build cgo

package bls

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc"
	gnark "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fp"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	"github.com/ethereum/go-ethereum/common"
	blst "github.com/supranational/blst/bindings/go"
)

func fuzzG1SubgroupChecks(data []byte) int {
	input := bytes.NewReader(data)
	cpG1, blG1, err := getG1Points(input)
	if err != nil {
		return 0
	}
	inSubGroupGnark := cpG1.IsInSubGroup()
	inSubGroupBLST := blG1.InG1()
	if inSubGroupGnark != inSubGroupBLST {
		panic(fmt.Sprintf("differing subgroup check, gnark %v, blst %v", inSubGroupGnark, inSubGroupBLST))
	}
	return 1
}

func fuzzG2SubgroupChecks(data []byte) int {
	input := bytes.NewReader(data)
	gpG2, blG2, err := getG2Points(input)
	if err != nil {
		return 0
	}
	inSubGroupGnark := gpG2.IsInSubGroup()
	inSubGroupBLST := blG2.InG2()
	if inSubGroupGnark != inSubGroupBLST {
		panic(fmt.Sprintf("differing subgroup check, gnark %v, blst %v", inSubGroupGnark, inSubGroupBLST))
	}
	return 1
}

func fuzzCrossPairing(data []byte) int {
	input := bytes.NewReader(data)

	// get random G1 points
	cpG1, blG1, err := getG1Points(input)
	if err != nil {
		return 0
	}

	// get random G2 points
	cpG2, blG2, err := getG2Points(input)
	if err != nil {
		return 0
	}

	// compute pairing using gnark
	cResult, err := gnark.Pair([]gnark.G1Affine{*cpG1}, []gnark.G2Affine{*cpG2})
	if err != nil {
		panic(fmt.Sprintf("gnark/bls12381 encountered error: %v", err))
	}

	// compute pairing using blst
	blstResult := blst.Fp12MillerLoop(blG2, blG1)
	blstResult.FinalExp()
	res := massageBLST(blstResult.ToBendian())
	if !(bytes.Equal(res, cResult.Marshal())) {
		panic("pairing mismatch blst / geth")
	}

	return 1
}

func massageBLST(in []byte) []byte {
	out := make([]byte, len(in))
	len := 12 * 48
	// 1
	copy(out[0:], in[len-1*48:len])
	copy(out[1*48:], in[len-2*48:len-1*48])
	// 2
	copy(out[6*48:], in[len-3*48:len-2*48])
	copy(out[7*48:], in[len-4*48:len-3*48])
	// 3
	copy(out[2*48:], in[len-5*48:len-4*48])
	copy(out[3*48:], in[len-6*48:len-5*48])
	// 4
	copy(out[8*48:], in[len-7*48:len-6*48])
	copy(out[9*48:], in[len-8*48:len-7*48])
	// 5
	copy(out[4*48:], in[len-9*48:len-8*48])
	copy(out[5*48:], in[len-10*48:len-9*48])
	// 6
	copy(out[10*48:], in[len-11*48:len-10*48])
	copy(out[11*48:], in[len-12*48:len-11*48])
	return out
}

func fuzzCrossG1Add(data []byte) int {
	input := bytes.NewReader(data)

	// get random G1 points
	cp1, bl1, err := getG1Points(input)
	if err != nil {
		return 0
	}

	// get random G1 points
	cp2, bl2, err := getG1Points(input)
	if err != nil {
		return 0
	}

	// compute cp = cp1 + cp2
	_cp1 := new(gnark.G1Jac).FromAffine(cp1)
	_cp2 := new(gnark.G1Jac).FromAffine(cp2)
	cp := new(gnark.G1Affine).FromJacobian(_cp1.AddAssign(_cp2))

	bl3 := blst.P1AffinesAdd([]*blst.P1Affine{bl1, bl2})
	if !(bytes.Equal(cp.Marshal(), bl3.Serialize())) {
		panic("G1 point addition mismatch blst / geth ")
	}

	return 1
}

func fuzzCrossG2Add(data []byte) int {
	input := bytes.NewReader(data)

	// get random G2 points
	gp1, bl1, err := getG2Points(input)
	if err != nil {
		return 0
	}

	// get random G2 points
	gp2, bl2, err := getG2Points(input)
	if err != nil {
		return 0
	}

	// compute cp = cp1 + cp2
	_gp1 := new(gnark.G2Jac).FromAffine(gp1)
	_gp2 := new(gnark.G2Jac).FromAffine(gp2)
	gp := new(gnark.G2Affine).FromJacobian(_gp1.AddAssign(_gp2))

	bl3 := blst.P2AffinesAdd([]*blst.P2Affine{bl1, bl2})
	if !(bytes.Equal(gp.Marshal(), bl3.Serialize())) {
		panic("G2 point addition mismatch blst / geth ")
	}

	return 1
}

func fuzzCrossG1MultiExp(data []byte) int {
	var (
		input        = bytes.NewReader(data)
		gnarkScalars []fr.Element
		gnarkPoints  []gnark.G1Affine
		blstScalars  []*blst.Scalar
		blstPoints   []*blst.P1Affine
	)
	// n random scalars (max 17)
	for i := 0; i < 17; i++ {
		// note that geth/crypto/bls12381 works only with scalars <= 32bytes
		s, err := randomScalar(input, fr.Modulus())
		if err != nil {
			break
		}
		// get a random G1 point as basis
		cp1, bl1, err := getG1Points(input)
		if err != nil {
			break
		}

		gnarkScalar := new(fr.Element).SetBigInt(s)
		gnarkScalars = append(gnarkScalars, *gnarkScalar)
		gnarkPoints = append(gnarkPoints, *cp1)

		blstScalar := new(blst.Scalar).FromBEndian(common.LeftPadBytes(s.Bytes(), 32))
		blstScalars = append(blstScalars, blstScalar)
		blstPoints = append(blstPoints, bl1)
	}

	if len(gnarkScalars) == 0 || len(gnarkScalars) != len(gnarkPoints) {
		return 0
	}

	// gnark multi exp
	cp := new(gnark.G1Affine)
	cp.MultiExp(gnarkPoints, gnarkScalars, ecc.MultiExpConfig{})

	expectedGnark := multiExpG1Gnark(gnarkPoints, gnarkScalars)
	if !bytes.Equal(cp.Marshal(), expectedGnark.Marshal()) {
		panic("g1 multi exponentiation mismatch")
	}

	// blst multi exp
	expectedBlst := blst.P1AffinesMult(blstPoints, blstScalars, 256).ToAffine()
	if !bytes.Equal(cp.Marshal(), expectedBlst.Serialize()) {
		panic("g1 multi exponentiation mismatch, gnark/blst")
	}
	return 1
}

func fuzzCrossG2MultiExp(data []byte) int {
	var (
		input        = bytes.NewReader(data)
		gnarkScalars []fr.Element
		gnarkPoints  []gnark.G2Affine
		blstScalars  []*blst.Scalar
		blstPoints   []*blst.P2Affine
	)
	// n random scalars (max 17)
	for i := 0; i < 17; i++ {
		// note that geth/crypto/bls12381 works only with scalars <= 32bytes
		s, err := randomScalar(input, fr.Modulus())
		if err != nil {
			break
		}
		// get a random G1 point as basis
		cp1, bl1, err := getG2Points(input)
		if err != nil {
			break
		}

		gnarkScalar := new(fr.Element).SetBigInt(s)
		gnarkScalars = append(gnarkScalars, *gnarkScalar)
		gnarkPoints = append(gnarkPoints, *cp1)

		blstScalar := new(blst.Scalar).FromBEndian(common.LeftPadBytes(s.Bytes(), 32))
		blstScalars = append(blstScalars, blstScalar)
		blstPoints = append(blstPoints, bl1)
	}

	if len(gnarkScalars) == 0 || len(gnarkScalars) != len(gnarkPoints) {
		return 0
	}

	// gnark multi exp
	cp := new(gnark.G2Affine)
	cp.MultiExp(gnarkPoints, gnarkScalars, ecc.MultiExpConfig{})

	expectedGnark := multiExpG2Gnark(gnarkPoints, gnarkScalars)
	if !bytes.Equal(cp.Marshal(), expectedGnark.Marshal()) {
		panic("g1 multi exponentiation mismatch")
	}

	// blst multi exp
	expectedBlst := blst.P2AffinesMult(blstPoints, blstScalars, 256).ToAffine()
	if !bytes.Equal(cp.Marshal(), expectedBlst.Serialize()) {
		panic("g1 multi exponentiation mismatch, gnark/blst")
	}
	return 1
}

func getG1Points(input io.Reader) (*gnark.G1Affine, *blst.P1Affine, error) {
	// sample a random scalar
	s, err := randomScalar(input, fp.Modulus())
	if err != nil {
		return nil, nil, err
	}

	// compute a random point
	cp := new(gnark.G1Affine)
	_, _, g1Gen, _ := gnark.Generators()
	cp.ScalarMultiplication(&g1Gen, s)
	cpBytes := cp.Marshal()

	// marshal gnark point -> blst point
	scalar := new(blst.Scalar).FromBEndian(common.LeftPadBytes(s.Bytes(), 32))
	p1 := new(blst.P1Affine).From(scalar)
	blstRes := p1.Serialize()
	if !bytes.Equal(blstRes, cpBytes) {
		panic(fmt.Sprintf("bytes(blst.G1) != bytes(geth.G1)\nblst.G1: %x\ngeth.G1: %x\n", blstRes, cpBytes))
	}

	return cp, p1, nil
}

func getG2Points(input io.Reader) (*gnark.G2Affine, *blst.P2Affine, error) {
	// sample a random scalar
	s, err := randomScalar(input, fp.Modulus())
	if err != nil {
		return nil, nil, err
	}

	// compute a random point
	gp := new(gnark.G2Affine)
	_, _, _, g2Gen := gnark.Generators()
	gp.ScalarMultiplication(&g2Gen, s)
	cpBytes := gp.Marshal()

	// marshal gnark point -> blst point
	// Left pad the scalar to 32 bytes
	scalar := new(blst.Scalar).FromBEndian(common.LeftPadBytes(s.Bytes(), 32))
	p2 := new(blst.P2Affine).From(scalar)
	if !bytes.Equal(p2.Serialize(), cpBytes) {
		panic("bytes(blst.G2) != bytes(bls12381.G2)")
	}

	return gp, p2, nil
}

func randomScalar(r io.Reader, max *big.Int) (k *big.Int, err error) {
	for {
		k, err = rand.Int(r, max)
		if err != nil || k.Sign() > 0 {
			return
		}
	}
}

// multiExpG1Gnark is a naive implementation of G1 multi-exponentiation
func multiExpG1Gnark(gs []gnark.G1Affine, scalars []fr.Element) gnark.G1Affine {
	res := gnark.G1Affine{}
	for i := 0; i < len(gs); i++ {
		tmp := new(gnark.G1Affine)
		sb := scalars[i].Bytes()
		scalarBytes := new(big.Int).SetBytes(sb[:])
		tmp.ScalarMultiplication(&gs[i], scalarBytes)
		res.Add(&res, tmp)
	}
	return res
}

// multiExpG2Gnark is a naive implementation of G2 multi-exponentiation
func multiExpG2Gnark(gs []gnark.G2Affine, scalars []fr.Element) gnark.G2Affine {
	res := gnark.G2Affine{}
	for i := 0; i < len(gs); i++ {
		tmp := new(gnark.G2Affine)
		sb := scalars[i].Bytes()
		scalarBytes := new(big.Int).SetBytes(sb[:])
		tmp.ScalarMultiplication(&gs[i], scalarBytes)
		res.Add(&res, tmp)
	}
	return res
}
