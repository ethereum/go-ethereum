// Copyright 2018 Péter Szilágyi. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package bn256

import (
	"bytes"
	"fmt"
	"io"
	"math/big"

	cloudflare "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
	google "github.com/ethereum/go-ethereum/crypto/bn256/google"
)

func getG1Points(input io.Reader) (*cloudflare.G1, *google.G1) {
	_, xc, err := cloudflare.RandomG1(input)
	if err != nil {
		// insufficient input
		return nil, nil
	}
	xg := new(google.G1)
	if _, err := xg.Unmarshal(xc.Marshal()); err != nil {
		panic(fmt.Sprintf("Could not marshal cloudflare -> google: %v", err))
	}
	return xc, xg
}

func getG2Points(input io.Reader) (*cloudflare.G2, *google.G2) {
	_, xc, err := cloudflare.RandomG2(input)
	if err != nil {
		// insufficient input
		return nil, nil
	}
	xg := new(google.G2)
	if _, err := xg.Unmarshal(xc.Marshal()); err != nil {
		panic(fmt.Sprintf("Could not marshal cloudflare -> google: %v", err))
	}
	return xc, xg
}

// FuzzAdd fuzzez bn256 addition between the Google and Cloudflare libraries.
func FuzzAdd(data []byte) int {
	input := bytes.NewReader(data)
	xc, xg := getG1Points(input)
	if xc == nil {
		return 0
	}
	yc, yg := getG1Points(input)
	if yc == nil {
		return 0
	}
	// Ensure both libs can parse the second curve point
	// Add the two points and ensure they result in the same output
	rc := new(cloudflare.G1)
	rc.Add(xc, yc)

	rg := new(google.G1)
	rg.Add(xg, yg)

	if !bytes.Equal(rc.Marshal(), rg.Marshal()) {
		panic("add mismatch")
	}
	return 1
}

// FuzzMul fuzzez bn256 scalar multiplication between the Google and Cloudflare
// libraries.
func FuzzMul(data []byte) int {
	input := bytes.NewReader(data)
	pc, pg := getG1Points(input)
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

	if !bytes.Equal(rc.Marshal(), rg.Marshal()) {
		panic("scalar mul mismatch")
	}
	return 1
}

func FuzzPair(data []byte) int {
	input := bytes.NewReader(data)
	pc, pg := getG1Points(input)
	if pc == nil {
		return 0
	}
	tc, tg := getG2Points(input)
	if tc == nil {
		return 0
	}
	// Pair the two points and ensure thet result in the same output
	if cloudflare.PairingCheck([]*cloudflare.G1{pc}, []*cloudflare.G2{tc}) != google.PairingCheck([]*google.G1{pg}, []*google.G2{tg}) {
		panic("pair mismatch")
	}
	return 1
}
