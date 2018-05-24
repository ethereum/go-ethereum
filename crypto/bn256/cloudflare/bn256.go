// Package bn256 implements a particular bilinear group at the 128-bit security
// level.
//
// Bilinear groups are the basis of many of the new cryptographic protocols that
// have been proposed over the past decade. They consist of a triplet of groups
// (G₁, G₂ and GT) such that there exists a function g(g₁ˣ,g₂ʸ)=gTˣʸ (where gₓ
// is a generator of the respective group). That function is called a pairing
// function.
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

func randomK(r io.Reader) (k *big.Int, err error) {
	for {
		k, err = rand.Int(r, Order)
		if k.Sign() > 0 || err != nil {
			return
		}
	}
}

// G1 is an abstract cyclic group. The zero value is suitable for use as the
// output of an operation, but cannot be used as an input.
type G1 struct {
	p *curvePoint
}

// RandomG1 returns x and g₁ˣ where x is a random, non-zero number read from r.
func RandomG1(r io.Reader) (*big.Int, *G1, error) {
	k, err := randomK(r)
	if err != nil {
		return nil, nil, err
	}

	return k, new(G1).ScalarBaseMult(k), nil
}

func (g *G1) String() string {
	return "bn256.G1" + g.p.String()
}

// ScalarBaseMult sets g to g*k where g is the generator of the group and then
// returns g.
func (g *G1) ScalarBaseMult(k *big.Int) *G1 {
	if g.p == nil {
		g.p = &curvePoint{}
	}
	g.p.Mul(curveGen, k)
	return g
}

// ScalarMult sets g to a*k and then returns g.
func (g *G1) ScalarMult(a *G1, k *big.Int) *G1 {
	if g.p == nil {
		g.p = &curvePoint{}
	}
	g.p.Mul(a.p, k)
	return g
}

// Add sets g to a+b and then returns g.
func (g *G1) Add(a, b *G1) *G1 {
	if g.p == nil {
		g.p = &curvePoint{}
	}
	g.p.Add(a.p, b.p)
	return g
}

// Neg sets g to -a and then returns g.
func (g *G1) Neg(a *G1) *G1 {
	if g.p == nil {
		g.p = &curvePoint{}
	}
	g.p.Neg(a.p)
	return g
}

// Set sets g to a and then returns g.
func (g *G1) Set(a *G1) *G1 {
	if g.p == nil {
		g.p = &curvePoint{}
	}
	g.p.Set(a.p)
	return g
}

// Marshal converts g to a byte slice.
func (g *G1) Marshal() []byte {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8

	g.p.MakeAffine()
	ret := make([]byte, numBytes*2)
	if g.p.IsInfinity() {
		return ret
	}
	temp := &gfP{}

	montDecode(temp, &g.p.x)
	temp.Marshal(ret)
	montDecode(temp, &g.p.y)
	temp.Marshal(ret[numBytes:])

	return ret
}

// Unmarshal sets g to the result of converting the output of Marshal back into
// a group element and then returns g.
func (g *G1) Unmarshal(m []byte) ([]byte, error) {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8
	if len(m) < 2*numBytes {
		return nil, errors.New("bn256: not enough data")
	}
	// Unmarshal the points and check their caps
	if g.p == nil {
		g.p = &curvePoint{}
	} else {
		g.p.x, g.p.y = gfP{0}, gfP{0}
	}
	var err error
	if err = g.p.x.Unmarshal(m); err != nil {
		return nil, err
	}
	if err = g.p.y.Unmarshal(m[numBytes:]); err != nil {
		return nil, err
	}
	// Encode into Montgomery form and ensure it's on the curve
	montEncode(&g.p.x, &g.p.x)
	montEncode(&g.p.y, &g.p.y)

	zero := gfP{0}
	if g.p.x == zero && g.p.y == zero {
		// This is the point at infinity.
		g.p.y = *newGFp(1)
		g.p.z = gfP{0}
		g.p.t = gfP{0}
	} else {
		g.p.z = *newGFp(1)
		g.p.t = *newGFp(1)

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
	k, err := randomK(r)
	if err != nil {
		return nil, nil, err
	}

	return k, new(G2).ScalarBaseMult(k), nil
}

func (g *G2) String() string {
	return "bn256.G2" + g.p.String()
}

// ScalarBaseMult sets g to g*k where g is the generator of the group and then
// returns out.
func (g *G2) ScalarBaseMult(k *big.Int) *G2 {
	if g.p == nil {
		g.p = &twistPoint{}
	}
	g.p.Mul(twistGen, k)
	return g
}

// ScalarMult sets g to a*k and then returns g.
func (g *G2) ScalarMult(a *G2, k *big.Int) *G2 {
	if g.p == nil {
		g.p = &twistPoint{}
	}
	g.p.Mul(a.p, k)
	return g
}

// Add sets g to a+b and then returns g.
func (g *G2) Add(a, b *G2) *G2 {
	if g.p == nil {
		g.p = &twistPoint{}
	}
	g.p.Add(a.p, b.p)
	return g
}

// Neg sets g to -a and then returns g.
func (g *G2) Neg(a *G2) *G2 {
	if g.p == nil {
		g.p = &twistPoint{}
	}
	g.p.Neg(a.p)
	return g
}

// Set sets g to a and then returns g.
func (g *G2) Set(a *G2) *G2 {
	if g.p == nil {
		g.p = &twistPoint{}
	}
	g.p.Set(a.p)
	return g
}

// Marshal converts g into a byte slice.
func (g *G2) Marshal() []byte {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8

	if g.p == nil {
		g.p = &twistPoint{}
	}

	g.p.MakeAffine()
	ret := make([]byte, numBytes*4)
	if g.p.IsInfinity() {
		return ret
	}
	temp := &gfP{}

	montDecode(temp, &g.p.x.x)
	temp.Marshal(ret)
	montDecode(temp, &g.p.x.y)
	temp.Marshal(ret[numBytes:])
	montDecode(temp, &g.p.y.x)
	temp.Marshal(ret[2*numBytes:])
	montDecode(temp, &g.p.y.y)
	temp.Marshal(ret[3*numBytes:])

	return ret
}

// Unmarshal sets g to the result of converting the output of Marshal back into
// a group element and then returns g.
func (g *G2) Unmarshal(m []byte) ([]byte, error) {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8
	if len(m) < 4*numBytes {
		return nil, errors.New("bn256: not enough data")
	}
	// Unmarshal the points and check their caps
	if g.p == nil {
		g.p = &twistPoint{}
	}
	var err error
	if err = g.p.x.x.Unmarshal(m); err != nil {
		return nil, err
	}
	if err = g.p.x.y.Unmarshal(m[numBytes:]); err != nil {
		return nil, err
	}
	if err = g.p.y.x.Unmarshal(m[2*numBytes:]); err != nil {
		return nil, err
	}
	if err = g.p.y.y.Unmarshal(m[3*numBytes:]); err != nil {
		return nil, err
	}
	// Encode into Montgomery form and ensure it's on the curve
	montEncode(&g.p.x.x, &g.p.x.x)
	montEncode(&g.p.x.y, &g.p.x.y)
	montEncode(&g.p.y.x, &g.p.y.x)
	montEncode(&g.p.y.y, &g.p.y.y)

	if g.p.x.IsZero() && g.p.y.IsZero() {
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

// Pair calculates an Optimal Ate pairing.
func Pair(g1 *G1, g2 *G2) *GT {
	return &GT{optimalAte(g2.p, g1.p)}
}

// PairingCheck calculates the Optimal Ate pairing for a set of points.
func PairingCheck(a []*G1, b []*G2) bool {
	acc := new(gfP12)
	acc.SetOne()

	for i := 0; i < len(a); i++ {
		if a[i].p.IsInfinity() || b[i].p.IsInfinity() {
			continue
		}
		acc.Mul(acc, miller(b[i].p, a[i].p))
	}
	return finalExponentiation(acc).IsOne()
}

// Miller applies Miller's algorithm, which is a bilinear function from the
// source groups to F_p^12. Miller(g1, g2).Finalize() is equivalent to Pair(g1,
// g2).
func Miller(g1 *G1, g2 *G2) *GT {
	return &GT{miller(g2.p, g1.p)}
}

func (g *GT) String() string {
	return "bn256.GT" + g.p.String()
}

// ScalarMult sets g to a*k and then returns g.
func (g *GT) ScalarMult(a *GT, k *big.Int) *GT {
	if g.p == nil {
		g.p = &gfP12{}
	}
	g.p.Exp(a.p, k)
	return g
}

// Add sets g to a+b and then returns g.
func (g *GT) Add(a, b *GT) *GT {
	if g.p == nil {
		g.p = &gfP12{}
	}
	g.p.Mul(a.p, b.p)
	return g
}

// Neg sets g to -a and then returns g.
func (g *GT) Neg(a *GT) *GT {
	if g.p == nil {
		g.p = &gfP12{}
	}
	g.p.Conjugate(a.p)
	return g
}

// Set sets g to a and then returns g.
func (g *GT) Set(a *GT) *GT {
	if g.p == nil {
		g.p = &gfP12{}
	}
	g.p.Set(a.p)
	return g
}

// Finalize is a linear function from F_p^12 to GT.
func (g *GT) Finalize() *GT {
	ret := finalExponentiation(g.p)
	g.p.Set(ret)
	return g
}

// Marshal converts g into a byte slice.
func (g *GT) Marshal() []byte {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8

	ret := make([]byte, numBytes*12)
	temp := &gfP{}

	montDecode(temp, &g.p.x.x.x)
	temp.Marshal(ret)
	montDecode(temp, &g.p.x.x.y)
	temp.Marshal(ret[numBytes:])
	montDecode(temp, &g.p.x.y.x)
	temp.Marshal(ret[2*numBytes:])
	montDecode(temp, &g.p.x.y.y)
	temp.Marshal(ret[3*numBytes:])
	montDecode(temp, &g.p.x.z.x)
	temp.Marshal(ret[4*numBytes:])
	montDecode(temp, &g.p.x.z.y)
	temp.Marshal(ret[5*numBytes:])
	montDecode(temp, &g.p.y.x.x)
	temp.Marshal(ret[6*numBytes:])
	montDecode(temp, &g.p.y.x.y)
	temp.Marshal(ret[7*numBytes:])
	montDecode(temp, &g.p.y.y.x)
	temp.Marshal(ret[8*numBytes:])
	montDecode(temp, &g.p.y.y.y)
	temp.Marshal(ret[9*numBytes:])
	montDecode(temp, &g.p.y.z.x)
	temp.Marshal(ret[10*numBytes:])
	montDecode(temp, &g.p.y.z.y)
	temp.Marshal(ret[11*numBytes:])

	return ret
}

// Unmarshal sets g to the result of converting the output of Marshal back into
// a group element and then returns g.
func (g *GT) Unmarshal(m []byte) ([]byte, error) {
	// Each value is a 256-bit number.
	const numBytes = 256 / 8

	if len(m) < 12*numBytes {
		return nil, errors.New("bn256: not enough data")
	}

	if g.p == nil {
		g.p = &gfP12{}
	}

	var err error
	if err = g.p.x.x.x.Unmarshal(m); err != nil {
		return nil, err
	}
	if err = g.p.x.x.y.Unmarshal(m[numBytes:]); err != nil {
		return nil, err
	}
	if err = g.p.x.y.x.Unmarshal(m[2*numBytes:]); err != nil {
		return nil, err
	}
	if err = g.p.x.y.y.Unmarshal(m[3*numBytes:]); err != nil {
		return nil, err
	}
	if err = g.p.x.z.x.Unmarshal(m[4*numBytes:]); err != nil {
		return nil, err
	}
	if err = g.p.x.z.y.Unmarshal(m[5*numBytes:]); err != nil {
		return nil, err
	}
	if err = g.p.y.x.x.Unmarshal(m[6*numBytes:]); err != nil {
		return nil, err
	}
	if err = g.p.y.x.y.Unmarshal(m[7*numBytes:]); err != nil {
		return nil, err
	}
	if err = g.p.y.y.x.Unmarshal(m[8*numBytes:]); err != nil {
		return nil, err
	}
	if err = g.p.y.y.y.Unmarshal(m[9*numBytes:]); err != nil {
		return nil, err
	}
	if err = g.p.y.z.x.Unmarshal(m[10*numBytes:]); err != nil {
		return nil, err
	}
	if err = g.p.y.z.y.Unmarshal(m[11*numBytes:]); err != nil {
		return nil, err
	}
	montEncode(&g.p.x.x.x, &g.p.x.x.x)
	montEncode(&g.p.x.x.y, &g.p.x.x.y)
	montEncode(&g.p.x.y.x, &g.p.x.y.x)
	montEncode(&g.p.x.y.y, &g.p.x.y.y)
	montEncode(&g.p.x.z.x, &g.p.x.z.x)
	montEncode(&g.p.x.z.y, &g.p.x.z.y)
	montEncode(&g.p.y.x.x, &g.p.y.x.x)
	montEncode(&g.p.y.x.y, &g.p.y.x.y)
	montEncode(&g.p.y.y.x, &g.p.y.y.x)
	montEncode(&g.p.y.y.y, &g.p.y.y.y)
	montEncode(&g.p.y.z.x, &g.p.y.z.x)
	montEncode(&g.p.y.z.y, &g.p.y.z.y)

	return m[12*numBytes:], nil
}
