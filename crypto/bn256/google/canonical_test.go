// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bn256

import (
	"math/big"
	"testing"
)

// reduced reports whether n is in [0, P).
func reduced(n *big.Int) bool {
	return n.Sign() >= 0 && n.Cmp(P) < 0
}

// TestMakeAffineZeroModP checks that a z which is a non-zero multiple of P is
// recognised as the point at infinity. Such a z has no inverse mod P, so the
// curve point used to hand ModInverse's nil return straight to big.Int.Mul,
// and the twist point used to invert a zero norm and end up claiming to be the
// affine point 0 : 0 : 1.
func TestMakeAffineZeroModP(t *testing.T) {
	pool := new(bnPool)

	cp := newCurvePoint(pool)
	cp.x.SetInt64(1)
	cp.y.SetInt64(2)
	cp.z.Set(P)
	cp.t.SetInt64(0)
	cp.MakeAffine(pool)
	if !cp.IsInfinity() {
		t.Errorf("curvePoint with z = P is (%s, %s, %s), want ∞", cp.x, cp.y, cp.z)
	}

	tp := newTwistPoint(pool)
	tp.Set(twistGen)
	tp.z.x.SetInt64(0)
	tp.z.y.Set(P)
	tp.MakeAffine(pool)
	if !tp.IsInfinity() {
		t.Errorf("twistPoint with z = (0, P) is %s, want ∞", tp)
	}
}

// TestMakeAffineReduces checks that MakeAffine canonicalises the coordinates it
// returns even when the point is already affine and so skips the inversion.
// Add and Double leave their outputs unreduced and Negative leaves y negative,
// so without this a caller cannot compare two points by their coordinates.
func TestMakeAffineReduces(t *testing.T) {
	pool := new(bnPool)

	want := newCurvePoint(pool).Mul(curveGen, big.NewInt(7), pool).MakeAffine(pool)

	// The same point with its coordinates shifted out of [0, P) the way the
	// group law leaves them, once with z = 1 and once with z merely ≡ 1.
	for _, z := range []*big.Int{big.NewInt(1), new(big.Int).Add(P, big.NewInt(1))} {
		got := newCurvePoint(pool)
		got.x.Add(want.x, P)
		got.y.Sub(want.y, P)
		got.z.Set(z)
		got.t.SetInt64(0)
		got.MakeAffine(pool)

		if got.x.Cmp(want.x) != 0 || got.y.Cmp(want.y) != 0 {
			t.Errorf("z = %s: MakeAffine = (%s, %s), want (%s, %s)", z, got.x, got.y, want.x, want.y)
		}
		if !reduced(got.x) || !reduced(got.y) || !reduced(got.z) {
			t.Errorf("z = %s: MakeAffine left unreduced coordinates: (%s, %s, %s)", z, got.x, got.y, got.z)
		}
		if got.t.Cmp(big.NewInt(1)) != 0 {
			t.Errorf("z = %s: MakeAffine left t = %s, want 1 = z²", z, got.t)
		}
	}
}

// TestNegativeIsCanonical checks that negating an affine point and calling
// MakeAffine yields the reduced representation. Negative sets y to -y, and
// MakeAffine used to return early on z = 1 without touching it.
func TestNegativeIsCanonical(t *testing.T) {
	pool := new(bnPool)

	a := newCurvePoint(pool).Mul(curveGen, big.NewInt(7), pool).MakeAffine(pool)

	neg := newCurvePoint(pool)
	neg.Negative(a)
	neg.MakeAffine(pool)
	if !reduced(neg.x) || !reduced(neg.y) {
		t.Errorf("Negative left unreduced coordinates: (%s, %s)", neg.x, neg.y)
	}
	if wantY := new(big.Int).Sub(P, a.y); neg.x.Cmp(a.x) != 0 || neg.y.Cmp(wantY) != 0 {
		t.Errorf("Negative = (%s, %s), want (%s, %s)", neg.x, neg.y, a.x, wantY)
	}

	tp := newTwistPoint(pool).Mul(twistGen, big.NewInt(7), pool).MakeAffine(pool)
	negT := newTwistPoint(pool)
	negT.Negative(tp, pool)
	negT.MakeAffine(pool)
	for _, n := range []*big.Int{negT.x.x, negT.x.y, negT.y.x, negT.y.y} {
		if !reduced(n) {
			t.Errorf("twistPoint.Negative left unreduced coordinates: %s", negT)
			break
		}
	}
}

// TestGFp2IsOneCanonical checks that IsOne only accepts the reduced one. It
// reads y through Bits, which reports the magnitude, so -1 used to pass.
func TestGFp2IsOneCanonical(t *testing.T) {
	for _, e := range []*gfP2{
		{new(big.Int), big.NewInt(-1)},
		{new(big.Int), new(big.Int).Add(P, big.NewInt(1))},
		{new(big.Int).Set(P), big.NewInt(1)},
	} {
		if e.IsOne() {
			t.Errorf("gfP2%s.IsOne() = true, want false", e)
		}
	}
	one := newGFp2(nil).SetOne()
	if !one.IsOne() {
		t.Errorf("gfP2%s.IsOne() = false, want true", one)
	}
}
