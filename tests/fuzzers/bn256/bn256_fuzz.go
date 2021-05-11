// Copyright 2018 Péter Szilágyi. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

// +build gofuzz

package bn256

import (
	"bytes"
	"fmt"
	"io"
	"math/big"

	gurvy "github.com/consensys/gurvy/bn256"
	cloudflare "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
	google "github.com/ethereum/go-ethereum/crypto/bn256/google"
)

func getG1Points(input io.Reader) (*cloudflare.G1, *google.G1, *gurvy.G1Affine) {
	_, xc, err := cloudflare.RandomG1(input)
	if err != nil {
		// insufficient input
		return nil, nil, nil
	}
	xg := new(google.G1)
	if _, err := xg.Unmarshal(xc.Marshal()); err != nil {
		panic(fmt.Sprintf("Could not marshal cloudflare -> google:", err))
	}
	xs := new(gurvy.G1Affine)
	if err := xs.Unmarshal(xc.Marshal()); err != nil {
		panic(fmt.Sprintf("Could not marshal cloudflare -> consensys:", err))
	}
	return xc, xg, xs
}

func getG2Points(input io.Reader) (*cloudflare.G2, *google.G2, *gurvy.G2Affine) {
	_, xc, err := cloudflare.RandomG2(input)
	if err != nil {
		// insufficient input
		return nil, nil, nil
	}
	xg := new(google.G2)
	if _, err := xg.Unmarshal(xc.Marshal()); err != nil {
		panic(fmt.Sprintf("Could not marshal cloudflare -> google:", err))
	}
	xs := new(gurvy.G2Affine)
	if err := xs.Unmarshal(xc.Marshal()); err != nil {
		panic(fmt.Sprintf("Could not marshal cloudflare -> consensys:", err))
	}
	return xc, xg, xs
}

// FuzzAdd fuzzez bn256 addition between the Google and Cloudflare libraries.
func FuzzAdd(data []byte) int {
	input := bytes.NewReader(data)
	xc, xg, xs := getG1Points(input)
	if xc == nil {
		return 0
	}
	yc, yg, ys := getG1Points(input)
	if yc == nil {
		return 0
	}
	// Ensure both libs can parse the second curve point
	// Add the two points and ensure they result in the same output
	rc := new(cloudflare.G1)
	rc.Add(xc, yc)

	rg := new(google.G1)
	rg.Add(xg, yg)

	tmpX := new(gurvy.G1Jac).FromAffine(xs)
	tmpY := new(gurvy.G1Jac).FromAffine(ys)
	rs := new(gurvy.G1Affine).FromJacobian(tmpX.AddAssign(tmpY))

	if !bytes.Equal(rc.Marshal(), rg.Marshal()) {
		panic("add mismatch: cloudflare/google")
	}

	if !bytes.Equal(rc.Marshal(), rs.Marshal()) {
		panic("add mismatch: cloudflare/consensys")
	}
	return 1
}

// FuzzMul fuzzez bn256 scalar multiplication between the Google and Cloudflare
// libraries.
func FuzzMul(data []byte) int {
	input := bytes.NewReader(data)
	pc, pg, ps := getG1Points(input)
	if pc == nil {
		return 0
	}
	// Add the two points and ensure they result in the same output
	remaining := input.Len()
	if remaining == 0 {
		return 0
	}
	if remaining > 128 {
		// The evm only ever uses 32 byte integers, we need to cap this otherwise
		// we run into slow exec. A 236Kb byte integer cause oss-fuzz to report it as slow.
		// 128 bytes should be fine though
		return 0
	}
	buf := make([]byte, remaining)
	input.Read(buf)

	rc := new(cloudflare.G1)
	rc.ScalarMult(pc, new(big.Int).SetBytes(buf))

	rg := new(google.G1)
	rg.ScalarMult(pg, new(big.Int).SetBytes(buf))

	rs := new(gurvy.G1Jac)
	psJac := new(gurvy.G1Jac).FromAffine(ps)
	rs.ScalarMultiplication(psJac, new(big.Int).SetBytes(buf))
	rsAffine := new(gurvy.G1Affine).FromJacobian(rs)

	if !bytes.Equal(rc.Marshal(), rg.Marshal()) {
		panic("scalar mul mismatch: cloudflare/google")
	}
	if !bytes.Equal(rc.Marshal(), rsAffine.Marshal()) {
		panic("scalar mul mismatch: cloudflare/consensys")
	}
	return 1
}

func FuzzPair(data []byte) int {
	input := bytes.NewReader(data)
	pc, pg, ps := getG1Points(input)
	if pc == nil {
		return 0
	}
	tc, tg, ts := getG2Points(input)
	if tc == nil {
		return 0
	}
	// Pair the two points and ensure they result in the same output
	clPair := cloudflare.PairingCheck([]*cloudflare.G1{pc}, []*cloudflare.G2{tc})
	if clPair != google.PairingCheck([]*google.G1{pg}, []*google.G2{tg}) {
		panic("pairing mismatch: cloudflare/google")
	}

	coPair, err := gurvy.PairingCheck([]gurvy.G1Affine{*ps}, []gurvy.G2Affine{*ts})
	if err != nil {
		panic(fmt.Sprintf("gurvy encountered error: %v", err))
	}
	if clPair != coPair {
		panic("pairing mismatch: cloudflare/consensys")
	}

	return 1
}
