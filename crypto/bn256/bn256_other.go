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

// +build !amd64 appengine gccgo

// Package bn256 implements the Optimal Ate pairing over a 256-bit Barreto-Naehrig curve.
package bn256

import (
	"math/big"

	"github.com/ethereum/go-ethereum/crypto/bn256/google"
)

// G1 is an abstract cyclic group. The zero value is suitable for use as the
// output of an operation, but cannot be used as an input.
type G1 struct {
	bn256.G1
}

// Add sets e to a+b and then returns e.
func (e *G1) Add(a, b *G1) *G1 {
	e.G1.Add(&a.G1, &b.G1)
	return e
}

// ScalarMult sets e to a*k and then returns e.
func (e *G1) ScalarMult(a *G1, k *big.Int) *G1 {
	e.G1.ScalarMult(&a.G1, k)
	return e
}

// G2 is an abstract cyclic group. The zero value is suitable for use as the
// output of an operation, but cannot be used as an input.
type G2 struct {
	bn256.G2
}

// PairingCheck calculates the Optimal Ate pairing for a set of points.
func PairingCheck(a []*G1, b []*G2) bool {
	as := make([]*bn256.G1, len(a))
	for i, p := range a {
		as[i] = &p.G1
	}
	bs := make([]*bn256.G2, len(b))
	for i, p := range b {
		bs[i] = &p.G2
	}
	return bn256.PairingCheck(as, bs)
}
