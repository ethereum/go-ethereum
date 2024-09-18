// Copyright 2018 The go-ethereum Authors
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

package bn256

import (
	"bytes"
	"fmt"
	"io"
	"math/big"

	cloudflare "github.com/XinFinOrg/XDPoSChain/crypto/bn256/cloudflare"
	google "github.com/XinFinOrg/XDPoSChain/crypto/bn256/google"
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
