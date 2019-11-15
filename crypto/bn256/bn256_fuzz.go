// Copyright 2018 Péter Szilágyi. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

// +build gofuzz

package bn256

import (
	"bytes"
	"math/big"

	cloudflare "github.com/maticnetwork/bor/crypto/bn256/cloudflare"
	google "github.com/maticnetwork/bor/crypto/bn256/google"
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
