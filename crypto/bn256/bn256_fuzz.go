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

// +build gofuzz

package bn256

import (
	"bytes"
	"math/big"

	cloudflare "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
	google "github.com/ethereum/go-ethereum/crypto/bn256/google"
)

// FuzzAdd fuzzez bn256 addition between the Google and Cloudflare libraries.
func FuzzAdd(data []byte) int {
	// Ensure we have enough data in the first place
	if len(data) != 128 {
		return 0
	}
	// Ensure both libs can parse the first curve point
	xc := new(cloudflare.G1)
	_, errc := xc.Unmarshal(data[:64])

	xg := new(google.G1)
	_, errg := xg.Unmarshal(data[:64])

	if (errc == nil) != (errg == nil) {
		panic("parse mismatch")
	} else if errc != nil {
		return 0
	}
	// Ensure both libs can parse the second curve point
	yc := new(cloudflare.G1)
	_, errc = yc.Unmarshal(data[64:])

	yg := new(google.G1)
	_, errg = yg.Unmarshal(data[64:])

	if (errc == nil) != (errg == nil) {
		panic("parse mismatch")
	} else if errc != nil {
		return 0
	}
	// Add the two points and ensure they result in the same output
	rc := new(cloudflare.G1)
	rc.Add(xc, yc)

	rg := new(google.G1)
	rg.Add(xg, yg)

	if !bytes.Equal(rc.Marshal(), rg.Marshal()) {
		panic("add mismatch")
	}
	return 0
}

// FuzzMul fuzzez bn256 scalar multiplication between the Google and Cloudflare
// libraries.
func FuzzMul(data []byte) int {
	// Ensure we have enough data in the first place
	if len(data) != 96 {
		return 0
	}
	// Ensure both libs can parse the curve point
	pc := new(cloudflare.G1)
	_, errc := pc.Unmarshal(data[:64])

	pg := new(google.G1)
	_, errg := pg.Unmarshal(data[:64])

	if (errc == nil) != (errg == nil) {
		panic("parse mismatch")
	} else if errc != nil {
		return 0
	}
	// Add the two points and ensure they result in the same output
	rc := new(cloudflare.G1)
	rc.ScalarMult(pc, new(big.Int).SetBytes(data[64:]))

	rg := new(google.G1)
	rg.ScalarMult(pg, new(big.Int).SetBytes(data[64:]))

	if !bytes.Equal(rc.Marshal(), rg.Marshal()) {
		panic("scalar mul mismatch")
	}
	return 0
}

func FuzzPair(data []byte) int {
	// Ensure we have enough data in the first place
	if len(data) != 192 {
		return 0
	}
	// Ensure both libs can parse the curve point
	pc := new(cloudflare.G1)
	_, errc := pc.Unmarshal(data[:64])

	pg := new(google.G1)
	_, errg := pg.Unmarshal(data[:64])

	if (errc == nil) != (errg == nil) {
		panic("parse mismatch")
	} else if errc != nil {
		return 0
	}
	// Ensure both libs can parse the twist point
	tc := new(cloudflare.G2)
	_, errc = tc.Unmarshal(data[64:])

	tg := new(google.G2)
	_, errg = tg.Unmarshal(data[64:])

	if (errc == nil) != (errg == nil) {
		panic("parse mismatch")
	} else if errc != nil {
		return 0
	}
	// Pair the two points and ensure thet result in the same output
	if cloudflare.PairingCheck([]*cloudflare.G1{pc}, []*cloudflare.G2{tc}) != google.PairingCheck([]*google.G1{pg}, []*google.G2{tg}) {
		panic("pair mismatch")
	}
	return 0
}
