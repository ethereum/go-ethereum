// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bn256

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"testing"
)

// TestG1AddAliasing checks that G1.Add tolerates a receiver that aliases one of
// its operands. The case that used to break is a+b with a and b the same point:
// Add dispatches to curvePoint.Double, which computed z3 = 2*y1*z1 after it had
// already overwritten c.y, so an aliased receiver fed it the wrong y1.
func TestG1AddAliasing(t *testing.T) {
	a := new(G1).ScalarBaseMult(big.NewInt(7))
	b := new(G1).ScalarBaseMult(big.NewInt(11))

	doubled := new(G1).ScalarMult(a, big.NewInt(2)).Marshal()
	sum := new(G1).Add(a, b).Marshal()

	got := new(G1).ScalarBaseMult(big.NewInt(7))
	if got.Add(got, got); !bytes.Equal(got.Marshal(), doubled) {
		t.Errorf("g.Add(g, g) = %x, want %x", got.Marshal(), doubled)
	}
	if got = new(G1).Add(a, a); !bytes.Equal(got.Marshal(), doubled) {
		t.Errorf("e.Add(g, g) = %x, want %x", got.Marshal(), doubled)
	}
	got = new(G1).ScalarBaseMult(big.NewInt(7))
	if got.Add(got, b); !bytes.Equal(got.Marshal(), sum) {
		t.Errorf("e.Add(e, b) = %x, want %x", got.Marshal(), sum)
	}
	got = new(G1).ScalarBaseMult(big.NewInt(11))
	if got.Add(a, got); !bytes.Equal(got.Marshal(), sum) {
		t.Errorf("e.Add(a, e) = %x, want %x", got.Marshal(), sum)
	}
}

// TestG2AddAliasing is TestG1AddAliasing for the twist.
func TestG2AddAliasing(t *testing.T) {
	a := new(G2).ScalarBaseMult(big.NewInt(7))
	b := new(G2).ScalarBaseMult(big.NewInt(11))

	doubled := new(G2).ScalarMult(a, big.NewInt(2)).Marshal()
	sum := new(G2).Add(a, b).Marshal()

	got := new(G2).ScalarBaseMult(big.NewInt(7))
	if got.Add(got, got); !bytes.Equal(got.Marshal(), doubled) {
		t.Errorf("g.Add(g, g) = %x, want %x", got.Marshal(), doubled)
	}
	if got = new(G2).Add(a, a); !bytes.Equal(got.Marshal(), doubled) {
		t.Errorf("e.Add(g, g) = %x, want %x", got.Marshal(), doubled)
	}
	got = new(G2).ScalarBaseMult(big.NewInt(7))
	if got.Add(got, b); !bytes.Equal(got.Marshal(), sum) {
		t.Errorf("e.Add(e, b) = %x, want %x", got.Marshal(), sum)
	}
	got = new(G2).ScalarBaseMult(big.NewInt(11))
	if got.Add(a, got); !bytes.Equal(got.Marshal(), sum) {
		t.Errorf("e.Add(a, e) = %x, want %x", got.Marshal(), sum)
	}
}

// TestPointDoubleAliasing exercises the underlying Double and Negative directly,
// on random points, with and without an aliased receiver.
func TestPointDoubleAliasing(t *testing.T) {
	pool := new(bnPool)
	for i := 0; i < 10; i++ {
		k, err := rand.Int(rand.Reader, Order)
		if err != nil {
			t.Fatal(err)
		}
		cp := newCurvePoint(pool).Mul(curveGen, k, pool)
		want := newCurvePoint(pool)
		want.Double(cp, pool)
		want.MakeAffine(pool)
		got := newCurvePoint(pool)
		got.Set(cp)
		got.Double(got, pool)
		got.MakeAffine(pool)
		if got.x.Cmp(want.x) != 0 || got.y.Cmp(want.y) != 0 {
			t.Fatalf("%d: aliased curvePoint.Double = (%s, %s), want (%s, %s)", i, got.x, got.y, want.x, want.y)
		}

		want.Negative(cp)
		want.MakeAffine(pool)
		got.Set(cp)
		got.Negative(got)
		got.MakeAffine(pool)
		if got.x.Cmp(want.x) != 0 || got.y.Cmp(want.y) != 0 {
			t.Fatalf("%d: aliased curvePoint.Negative = (%s, %s), want (%s, %s)", i, got.x, got.y, want.x, want.y)
		}

		tp := newTwistPoint(pool).Mul(twistGen, k, pool)
		wantT := newTwistPoint(pool)
		wantT.Double(tp, pool)
		wantT.MakeAffine(pool)
		gotT := newTwistPoint(pool)
		gotT.Set(tp)
		gotT.Double(gotT, pool)
		gotT.MakeAffine(pool)
		if !twistEqual(gotT, wantT) {
			t.Fatalf("%d: aliased twistPoint.Double disagrees with the unaliased result", i)
		}

		wantT.Negative(tp, pool)
		wantT.MakeAffine(pool)
		gotT.Set(tp)
		gotT.Negative(gotT, pool)
		gotT.MakeAffine(pool)
		if !twistEqual(gotT, wantT) {
			t.Fatalf("%d: aliased twistPoint.Negative disagrees with the unaliased result", i)
		}
	}
}

// twistEqual compares two affine twist points.
func twistEqual(a, b *twistPoint) bool {
	if a.IsInfinity() || b.IsInfinity() {
		return a.IsInfinity() == b.IsInfinity()
	}
	a.x.Minimal()
	a.y.Minimal()
	b.x.Minimal()
	b.y.Minimal()
	return a.x.x.Cmp(b.x.x) == 0 && a.x.y.Cmp(b.x.y) == 0 &&
		a.y.x.Cmp(b.y.x) == 0 && a.y.y.Cmp(b.y.y) == 0
}
