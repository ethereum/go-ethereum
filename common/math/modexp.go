// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math

import (
	"math/big"
	"math/bits"

	"github.com/ethereum/go-ethereum/common"
)

// FastExp is semantically equivalent to x.Exp(x,y, m), but is faster for even
// modulus.
func FastExp(x, y, m *big.Int) *big.Int {
	// Split m = m1 × m2 where m1 = 2ⁿ
	n := m.TrailingZeroBits()
	m1 := new(big.Int).Lsh(common.Big1, n)
	mask := new(big.Int).Sub(m1, common.Big1)
	m2 := new(big.Int).Rsh(m, n)

	// We want z = x**y mod m.
	// z1 = x**y mod m1 = (x**y mod m) mod m1 = z mod m1
	// z2 = x**y mod m2 = (x**y mod m) mod m2 = z mod m2
	z1 := fastExpPow2(x, y, mask)
	z2 := new(big.Int).Exp(x, y, m2)

	// Reconstruct z from z1, z2 using CRT, using algorithm from paper,
	// which uses only a single modInverse.
	//	p = (z1 - z2) * m2⁻¹ (mod m1)
	//	z = z2 + p * m2
	z := new(big.Int).Set(z2)

	// Compute (z1 - z2) mod m1 [m1 == 2**n] into z1.
	z1 = z1.And(z1, mask)
	z2 = z2.And(z2, mask)
	z1 = z1.Sub(z1, z2)
	if z1.Sign() < 0 {
		z1 = z1.Add(z1, m1)
	}

	// Reuse z2 for p = z1 * m2inv.
	m2inv := new(big.Int).ModInverse(m2, m1)
	z2 = z2.Mul(z1, m2inv)
	z2 = z2.And(z2, mask)

	// Reuse z1 for m2 * p.
	z = z.Add(z, z1.Mul(z2, m2))
	z = z.Rem(z, m)

	return z
}

func fastExpPow2(x, y *big.Int, mask *big.Int) *big.Int {
	z := big.NewInt(1)
	if y.Sign() == 0 {
		return z
	}
	p := new(big.Int).Set(x)
	p = p.And(p, mask)
	if p.Cmp(z) <= 0 { // p <= 1
		return p
	}
	if y.Cmp(mask) > 0 {
		y = new(big.Int).And(y, mask)
	}
	t := new(big.Int)

	for _, b := range y.Bits() {
		for i := 0; i < bits.UintSize; i++ {
			if b&1 != 0 {
				z, t = t.Mul(z, p), z
				z = z.And(z, mask)
			}
			p, t = t.Mul(p, p), p
			p = p.And(p, mask)
			b >>= 1
		}
	}
	return z
}
