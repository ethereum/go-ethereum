// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bn256

import "testing"

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
