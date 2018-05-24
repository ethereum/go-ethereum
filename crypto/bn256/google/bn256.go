// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bn256 implements a particular bilinear group at the 128-bit security level.
//
// Bilinear groups are the basis of many of the new cryptographic protocols
// that have been proposed over the past decade. They consist of a triplet of
// groups (G₁, G₂ and GT) such that there exists a function g(g₁ˣ,g₂ʸ)=gTˣʸ
// (where gₓ is a generator of the respective group). That function is called
// a pairing function.
//
// This package specifically implements the Optimal Ate pairing over a 256-bit
// Barreto-Naehrig curve as described in
// http://cryptojedi.org/papers/dclxvi-20100714.pdf. Its output is compatible
// with the implementation described in that paper.
package bn256

import (
	"crypto/rand"
	"errors"
	"io"
	"math/big"
)

// BUG(agl): this implementation is not constant time.
// TODO(agl): keep GF(p²) elements in Mongomery form.

// G1 is an abstract cyclic group. The zero value is suitable for use as the
// output of an operation, but cannot be used as an input.
type G1 struct {
	p *curvePoint
}

// RandomG1 returns x and g₁ˣ where x is a random, non-zero number read from r.
func RandomG1(r io.Reader) (*big.Int, *G1, error) {
	var k *big.Int
	var err error

	for {
		k, err = rand.Int(r, Order)
		if err != nil {
			return nil, nil, err
		}
		if k.Sign() > 0 {
			break
		}
	}

	return k, new(G1).ScalarBaseMult(k), nil
}

func (g *G1) String() string {
	return "bn256.G1" + g.p.String()
}

// CurvePoints returns p's curve points in big integer
func (g *G1) CurvePoints() (*big.Int, *big.Int, *big.Int, *big.Int) {
	return g.p.x, g.p.y, g.p.z, g.p.t
}

// ScalarBaseMult sets g to g*k where g is the generator of the group and
// then returns g.
func (g *G1) ScalarBaseMult(k *big.Int) *G1 {
	if g.p == nil {
		g.p = newCurvePoint(nil)
	}
	g.p.Mul(curveGen, k, new(bnPool))
	return g
}

// ScalarMult sets g to a*k and then returns g.
func (g *G1) ScalarMult(a *G1, k *big.Int) *G1 {
	if g.p == nil {
		g.p = newCurvePoint(nil)
	}
	g.p.Mul(a.p, k, new(bnPool))
	return g
}

// Add sets g to a+b and then returns g.
// BUG(agl): this function is not complete: a==b fails.
func (g *G1) Add(a, b *G1) *G1 {
	if g.p == nil {
		g.p = newCurvePoint(nil)
	}
	g.p.Add(a.p, b.p, new(bnPool))
	return g
}

// Neg sets g to -a and then returns g.
func (g *G1) Neg(a *G1) *G1 {
	if g.p == nil {
		g.p = newCurvePoint(nil)
	}
	g.p.Negative(a.p)
	return g
}

// Marshal converts g to a byte slice.
func (g *G1) Marshal() []byte {
	g.p.MakeAffine(nil)

	xBytes := new(big.Int).Mod(g.p.x, P).Bytes()
	yBytes := new(big.Int).Mod(g.p.y, P).Bytes()

	// Each value is a 256-bit number.
	const numBytes = 256 / 8

	ret := make([]byte, numBytes*2)
	copy(ret[1*numBytes-len(xBytes):], xBytes)
	copy(ret[2*numBytes-len(yBytes):], yBytes)

	return ret
}

// Unmarshal sets g to the result of converting the output of Marshal back into
// a group element and then returns g.
func (g *G1) Unmarshal(m []byte) ([]byte, error) {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8
	if len(m) != 2*numBytes {
		return nil, errors.New("bn256: not enough data")
	}
	// Unmarshal the points and check their caps
	if g.p == nil {
		g.p = newCurvePoint(nil)
	}
	g.p.x.SetBytes(m[0*numBytes : 1*numBytes])
	if g.p.x.Cmp(P) >= 0 {
		return nil, errors.New("bn256: coordinate exceeds modulus")
	}
	g.p.y.SetBytes(m[1*numBytes : 2*numBytes])
	if g.p.y.Cmp(P) >= 0 {
		return nil, errors.New("bn256: coordinate exceeds modulus")
	}
	// Ensure the point is on the curve
	if g.p.x.Sign() == 0 && g.p.y.Sign() == 0 {
		// This is the point at infinity.
		g.p.y.SetInt64(1)
		g.p.z.SetInt64(0)
		g.p.t.SetInt64(0)
	} else {
		g.p.z.SetInt64(1)
		g.p.t.SetInt64(1)

		if !g.p.IsOnCurve() {
			return nil, errors.New("bn256: malformed point")
		}
	}
	return m[2*numBytes:], nil
}

// G2 is an abstract cyclic group. The zero value is suitable for use as the
// output of an operation, but cannot be used as an input.
type G2 struct {
	p *twistPoint
}

// RandomG2 returns x and g₂ˣ where x is a random, non-zero number read from r.
func RandomG2(r io.Reader) (*big.Int, *G2, error) {
	var k *big.Int
	var err error

	for {
		k, err = rand.Int(r, Order)
		if err != nil {
			return nil, nil, err
		}
		if k.Sign() > 0 {
			break
		}
	}

	return k, new(G2).ScalarBaseMult(k), nil
}

func (g *G2) String() string {
	return "bn256.G2" + g.p.String()
}

// CurvePoints returns the curve points of p which includes the real
// and imaginary parts of the curve point.
func (g *G2) CurvePoints() (*gfP2, *gfP2, *gfP2, *gfP2) {
	return g.p.x, g.p.y, g.p.z, g.p.t
}

// ScalarBaseMult sets g to g*k where g is the generator of the group and
// then returns out.
func (g *G2) ScalarBaseMult(k *big.Int) *G2 {
	if g.p == nil {
		g.p = newTwistPoint(nil)
	}
	g.p.Mul(twistGen, k, new(bnPool))
	return g
}

// ScalarMult sets g to a*k and then returns g.
func (g *G2) ScalarMult(a *G2, k *big.Int) *G2 {
	if g.p == nil {
		g.p = newTwistPoint(nil)
	}
	g.p.Mul(a.p, k, new(bnPool))
	return g
}

// Add sets g to a+b and then returns g.
// BUG(agl): this function is not complete: a==b fails.
func (g *G2) Add(a, b *G2) *G2 {
	if g.p == nil {
		g.p = newTwistPoint(nil)
	}
	g.p.Add(a.p, b.p, new(bnPool))
	return g
}

// Marshal converts g into a byte slice.
func (g *G2) Marshal() []byte {
	g.p.MakeAffine(nil)

	xxBytes := new(big.Int).Mod(g.p.x.x, P).Bytes()
	xyBytes := new(big.Int).Mod(g.p.x.y, P).Bytes()
	yxBytes := new(big.Int).Mod(g.p.y.x, P).Bytes()
	yyBytes := new(big.Int).Mod(g.p.y.y, P).Bytes()

	// Each value is a 256-bit number.
	const numBytes = 256 / 8

	ret := make([]byte, numBytes*4)
	copy(ret[1*numBytes-len(xxBytes):], xxBytes)
	copy(ret[2*numBytes-len(xyBytes):], xyBytes)
	copy(ret[3*numBytes-len(yxBytes):], yxBytes)
	copy(ret[4*numBytes-len(yyBytes):], yyBytes)

	return ret
}

// Unmarshal sets g to the result of converting the output of Marshal back into
// a group element and then returns g.
func (g *G2) Unmarshal(m []byte) ([]byte, error) {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8
	if len(m) != 4*numBytes {
		return nil, errors.New("bn256: not enough data")
	}
	// Unmarshal the points and check their caps
	if g.p == nil {
		g.p = newTwistPoint(nil)
	}
	g.p.x.x.SetBytes(m[0*numBytes : 1*numBytes])
	if g.p.x.x.Cmp(P) >= 0 {
		return nil, errors.New("bn256: coordinate exceeds modulus")
	}
	g.p.x.y.SetBytes(m[1*numBytes : 2*numBytes])
	if g.p.x.y.Cmp(P) >= 0 {
		return nil, errors.New("bn256: coordinate exceeds modulus")
	}
	g.p.y.x.SetBytes(m[2*numBytes : 3*numBytes])
	if g.p.y.x.Cmp(P) >= 0 {
		return nil, errors.New("bn256: coordinate exceeds modulus")
	}
	g.p.y.y.SetBytes(m[3*numBytes : 4*numBytes])
	if g.p.y.y.Cmp(P) >= 0 {
		return nil, errors.New("bn256: coordinate exceeds modulus")
	}
	// Ensure the point is on the curve
	if g.p.x.x.Sign() == 0 &&
		g.p.x.y.Sign() == 0 &&
		g.p.y.x.Sign() == 0 &&
		g.p.y.y.Sign() == 0 {
		// This is the point at infinity.
		g.p.y.SetOne()
		g.p.z.SetZero()
		g.p.t.SetZero()
	} else {
		g.p.z.SetOne()
		g.p.t.SetOne()

		if !g.p.IsOnCurve() {
			return nil, errors.New("bn256: malformed point")
		}
	}
	return m[4*numBytes:], nil
}

// GT is an abstract cyclic group. The zero value is suitable for use as the
// output of an operation, but cannot be used as an input.
type GT struct {
	p *gfP12
}

func (g *GT) String() string {
	return "bn256.GT" + g.p.String()
}

// ScalarMult sets g to a*k and then returns g.
func (g *GT) ScalarMult(a *GT, k *big.Int) *GT {
	if g.p == nil {
		g.p = newGFp12(nil)
	}
	g.p.Exp(a.p, k, new(bnPool))
	return g
}

// Add sets g to a+b and then returns g.
func (g *GT) Add(a, b *GT) *GT {
	if g.p == nil {
		g.p = newGFp12(nil)
	}
	g.p.Mul(a.p, b.p, new(bnPool))
	return g
}

// Neg sets g to -a and then returns g.
func (g *GT) Neg(a *GT) *GT {
	if g.p == nil {
		g.p = newGFp12(nil)
	}
	g.p.Invert(a.p, new(bnPool))
	return g
}

// Marshal converts g into a byte slice.
func (g *GT) Marshal() []byte {
	g.p.Minimal()

	xxxBytes := g.p.x.x.x.Bytes()
	xxyBytes := g.p.x.x.y.Bytes()
	xyxBytes := g.p.x.y.x.Bytes()
	xyyBytes := g.p.x.y.y.Bytes()
	xzxBytes := g.p.x.z.x.Bytes()
	xzyBytes := g.p.x.z.y.Bytes()
	yxxBytes := g.p.y.x.x.Bytes()
	yxyBytes := g.p.y.x.y.Bytes()
	yyxBytes := g.p.y.y.x.Bytes()
	yyyBytes := g.p.y.y.y.Bytes()
	yzxBytes := g.p.y.z.x.Bytes()
	yzyBytes := g.p.y.z.y.Bytes()

	// Each value is a 256-bit number.
	const numBytes = 256 / 8

	ret := make([]byte, numBytes*12)
	copy(ret[1*numBytes-len(xxxBytes):], xxxBytes)
	copy(ret[2*numBytes-len(xxyBytes):], xxyBytes)
	copy(ret[3*numBytes-len(xyxBytes):], xyxBytes)
	copy(ret[4*numBytes-len(xyyBytes):], xyyBytes)
	copy(ret[5*numBytes-len(xzxBytes):], xzxBytes)
	copy(ret[6*numBytes-len(xzyBytes):], xzyBytes)
	copy(ret[7*numBytes-len(yxxBytes):], yxxBytes)
	copy(ret[8*numBytes-len(yxyBytes):], yxyBytes)
	copy(ret[9*numBytes-len(yyxBytes):], yyxBytes)
	copy(ret[10*numBytes-len(yyyBytes):], yyyBytes)
	copy(ret[11*numBytes-len(yzxBytes):], yzxBytes)
	copy(ret[12*numBytes-len(yzyBytes):], yzyBytes)

	return ret
}

// Unmarshal sets g to the result of converting the output of Marshal back into
// a group element and then returns g.
func (g *GT) Unmarshal(m []byte) (*GT, bool) {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8

	if len(m) != 12*numBytes {
		return nil, false
	}

	if g.p == nil {
		g.p = newGFp12(nil)
	}

	g.p.x.x.x.SetBytes(m[0*numBytes : 1*numBytes])
	g.p.x.x.y.SetBytes(m[1*numBytes : 2*numBytes])
	g.p.x.y.x.SetBytes(m[2*numBytes : 3*numBytes])
	g.p.x.y.y.SetBytes(m[3*numBytes : 4*numBytes])
	g.p.x.z.x.SetBytes(m[4*numBytes : 5*numBytes])
	g.p.x.z.y.SetBytes(m[5*numBytes : 6*numBytes])
	g.p.y.x.x.SetBytes(m[6*numBytes : 7*numBytes])
	g.p.y.x.y.SetBytes(m[7*numBytes : 8*numBytes])
	g.p.y.y.x.SetBytes(m[8*numBytes : 9*numBytes])
	g.p.y.y.y.SetBytes(m[9*numBytes : 10*numBytes])
	g.p.y.z.x.SetBytes(m[10*numBytes : 11*numBytes])
	g.p.y.z.y.SetBytes(m[11*numBytes : 12*numBytes])

	return g, true
}

// Pair calculates an Optimal Ate pairing.
func Pair(g1 *G1, g2 *G2) *GT {
	return &GT{optimalAte(g2.p, g1.p, new(bnPool))}
}

// PairingCheck calculates the Optimal Ate pairing for a set of points.
func PairingCheck(a []*G1, b []*G2) bool {
	pool := new(bnPool)

	acc := newGFp12(pool)
	acc.SetOne()

	for i := 0; i < len(a); i++ {
		if a[i].p.IsInfinity() || b[i].p.IsInfinity() {
			continue
		}
		acc.Mul(acc, miller(b[i].p, a[i].p, pool), pool)
	}
	ret := finalExponentiation(acc, pool)
	acc.Put(pool)

	return ret.IsOne()
}

// bnPool implements a tiny cache of *big.Int objects that's used to reduce the
// number of allocations made during processing.
type bnPool struct {
	bns   []*big.Int
	count int
}

func (pool *bnPool) Get() *big.Int {
	if pool == nil {
		return new(big.Int)
	}

	pool.count++
	l := len(pool.bns)
	if l == 0 {
		return new(big.Int)
	}

	bn := pool.bns[l-1]
	pool.bns = pool.bns[:l-1]
	return bn
}

func (pool *bnPool) Put(bn *big.Int) {
	if pool == nil {
		return
	}
	pool.bns = append(pool.bns, bn)
	pool.count--
}

func (pool *bnPool) Count() int {
	return pool.count
}
