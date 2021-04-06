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

// build +gofuzz
package secp256k1

import (
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	fuzz "github.com/google/gofuzz"
)

func Fuzz(input []byte) int {
	var (
		fuzzer = fuzz.NewFromGoFuzz(input)
		curveA = S256()
		curveB = btcec.S256()
		dataP1 []byte
		dataP2 []byte
	)
	// first point
	fuzzer.Fuzz(&dataP1)
	x1, y1 := curveB.ScalarBaseMult(dataP1)
	// second point
	fuzzer.Fuzz(&dataP2)
	x2, y2 := curveB.ScalarBaseMult(dataP2)
	if curveA.IsOnCurve(x1, y1) {
		fmt.Println("point 1 on curve")
	}
	if curveA.IsOnCurve(x2, y2) {
		fmt.Println("point 2 on curve")
	}
	fmt.Printf("Input at 1: %v %v %v %v\n", x1, y1, x2, y2)
	resAX, resAY := curveA.Add(x1, y1, x2, y2)
	fmt.Printf("Input at 2: %v %v %v %v\n", x1, y1, x2, y2)
	resBX, resBY := curveB.Add(x1, y1, x2, y2)
	fmt.Printf("Input at 3: %v %v %v %v\n", x1, y1, x2, y2)
	if curveA.IsOnCurve(resAX, resAY) {
		fmt.Println("resA on curve")
	}
	if curveA.IsOnCurve(resBX, resBY) {
		fmt.Println("resB on curve")
	}
	resDx, resDy := curveA.Double(x1, y1)
	fmt.Printf("%s %s\n", resDx, resDy)
	if resAX.Cmp(resBX) != 0 || resAY.Cmp(resBY) != 0 {
		fmt.Printf("%s %s %s %s\n", x1, y1, x2, y2)
		panic(fmt.Sprintf("Addition failed: geth: %s %s btcd: %s %s", resAX, resAY, resBX, resBY))
	}
	return 0
}
