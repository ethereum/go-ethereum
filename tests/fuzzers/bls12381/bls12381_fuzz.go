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

// +build gofuzz

package bls

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"

	consensys "github.com/consensys/gnark-crypto/ecc/bls12-381"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fp"
	"github.com/ethereum/go-ethereum/crypto/bls12381"
)

func FuzzInteropPairing(data []byte) int {
	input := bytes.NewReader(data)
	kpG1, cpG1 := getG1Points(input)
	if kpG1 == nil {
		return 0
	}

	kpG2, cpG2 := getG2Points(input)
	if kpG2 == nil {
		return 0
	}

	// pairings
	engine := bls12381.NewPairingEngine()
	engine.AddPair(kpG1, kpG2)
	kResult := engine.Result()
	cResult, _ := consensys.Pair([]consensys.G1Affine{*cpG1}, []consensys.G2Affine{*cpG2})

	if !(bytes.Equal(cResult.Marshal(), bls12381.NewGT().ToBytes(kResult))) {
		panic("pairing mismatch consensys / geth ")
	}

	return 1
}

func getG1Points(input io.Reader) (*bls12381.PointG1, *consensys.G1Affine) {
	// sample a random scalar
	s, err := randomScalar(input)
	if err != nil {
		return nil, nil
	}

	// compute a random point
	cp := new(consensys.G1Affine)
	_, _, g1Gen, _ := consensys.Generators()
	cp.ScalarMultiplication(&g1Gen, s)
	cpBytes := cp.Marshal()

	// marshal consensys point -> geth point
	kp, err := bls12381.NewG1().FromBytes(cpBytes)
	if err != nil {
		panic(fmt.Sprintf("Could not marshal consensys.G1 -> geth.G1:", err))
	}
	if !bytes.Equal(bls12381.NewG1().ToBytes(kp), cpBytes) {
		panic("bytes(consensys.G1) != bytes(geth.G1)")
	}

	return kp, cp
}

func getG2Points(input io.Reader) (*bls12381.PointG2, *consensys.G2Affine) {
	// sample a random scalar
	s, err := randomScalar(input)
	if err != nil {
		return nil, nil
	}

	// compute a random point
	cp := new(consensys.G2Affine)
	_, _, _, g2Gen := consensys.Generators()
	cp.ScalarMultiplication(&g2Gen, s)
	cpBytes := cp.Marshal()

	// marshal consensys point -> geth point
	kp, err := bls12381.NewG2().FromBytes(cpBytes)
	if err != nil {
		panic(fmt.Sprintf("Could not marshal consensys.G2 -> geth.G2:", err))
	}
	if !bytes.Equal(bls12381.NewG2().ToBytes(kp), cpBytes) {
		panic("bytes(consensys.G2) != bytes(geth.G2)")
	}

	return kp, cp
}

func randomScalar(r io.Reader) (k *big.Int, err error) {
	for {
		k, err = rand.Int(r, fp.Modulus())
		if err != nil || k.Sign() > 0 {
			return
		}
	}
}
