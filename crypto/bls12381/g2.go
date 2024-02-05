package bls12381

import (
	"errors"
	"math"
	"math/big"
)

// PointG2 is type for point in G2 and used for both affine and Jacobian representation.
// A point is accounted as in affine form if z is equal to one.
type PointG2 [3]fe2

// Set copies values of one point to another.
func (p *PointG2) Set(p2 *PointG2) *PointG2 {
	p[0].set(&p2[0])
	p[1].set(&p2[1])
	p[2].set(&p2[2])
	return p
}

func (p *PointG2) Zero() *PointG2 {
	p[0].zero()
	p[1].one()
	p[2].zero()
	return p
}

// IsAffine checks a G1 point whether it is in affine form.
func (p *PointG2) IsAffine() bool {
	return p[2].isOne()
}

type tempG2 struct {
	t [9]*fe2
}

// G2 is struct for G2 group.
type G2 struct {
	f *fp2
	tempG2
}

// NewG2 constructs a new G2 instance.
func NewG2() *G2 {
	return newG2(nil)
}

func newG2(f *fp2) *G2 {
	if f == nil {
		f = newFp2()
	}
	t := newTempG2()
	return &G2{f, t}
}

func newTempG2() tempG2 {
	t := [9]*fe2{}
	for i := 0; i < 9; i++ {
		t[i] = &fe2{}
	}
	return tempG2{t}
}

// Q returns group order in big.Int.
func (g *G2) Q() *big.Int {
	return new(big.Int).Set(qBig)
}

// FromUncompressed expects byte slice at least 192 bytes and given bytes returns a new point in G2.
// Serialization rules are in line with zcash library. See below for details.
// https://github.com/zcash/librustzcash/blob/master/pairing/src/bls12_381/README.md#serialization
// https://docs.rs/bls12_381/0.1.1/bls12_381/notes/serialization/index.html
func (g *G2) FromUncompressed(uncompressed []byte) (*PointG2, error) {
	if len(uncompressed) != 4*fpByteSize {
		return nil, errors.New("input string length must be equal to 192 bytes")
	}
	var in [4 * fpByteSize]byte
	copy(in[:], uncompressed[:4*fpByteSize])
	if in[0]&(1<<7) != 0 {
		return nil, errors.New("compression flag must be zero")
	}
	if in[0]&(1<<5) != 0 {
		return nil, errors.New("sort flag must be zero")
	}
	if in[0]&(1<<6) != 0 {
		for i, v := range in {
			if (i == 0 && v != 0x40) || (i != 0 && v != 0x00) {
				return nil, errors.New("input string must be zero when infinity flag is set")
			}
		}
		return g.Zero(), nil
	}
	in[0] &= 0x1f
	x, err := g.f.fromBytes(in[:2*fpByteSize])
	if err != nil {
		return nil, err
	}
	y, err := g.f.fromBytes(in[2*fpByteSize:])
	if err != nil {
		return nil, err
	}
	z := new(fe2).one()
	p := &PointG2{*x, *y, *z}
	if !g.IsOnCurve(p) {
		return nil, errors.New("point is not on curve")
	}
	if !g.InCorrectSubgroup(p) {
		return nil, errors.New("point is not on correct subgroup")
	}
	return p, nil
}

// ToUncompressed given a G2 point returns bytes in uncompressed (x, y) form of the point.
// Serialization rules are in line with zcash library. See below for details.
// https://github.com/zcash/librustzcash/blob/master/pairing/src/bls12_381/README.md#serialization
// https://docs.rs/bls12_381/0.1.1/bls12_381/notes/serialization/index.html
func (g *G2) ToUncompressed(p *PointG2) []byte {
	out := make([]byte, 4*fpByteSize)
	g.Affine(p)
	if g.IsZero(p) {
		out[0] |= 1 << 6
		return out
	}
	copy(out[:2*fpByteSize], g.f.toBytes(&p[0]))
	copy(out[2*fpByteSize:], g.f.toBytes(&p[1]))
	return out
}

// FromCompressed expects byte slice at least 96 bytes and given bytes returns a new point in G2.
// Serialization rules are in line with zcash library. See below for details.
// https://github.com/zcash/librustzcash/blob/master/pairing/src/bls12_381/README.md#serialization
// https://docs.rs/bls12_381/0.1.1/bls12_381/notes/serialization/index.html
func (g *G2) FromCompressed(compressed []byte) (*PointG2, error) {
	if len(compressed) != 2*fpByteSize {
		return nil, errors.New("input string length must be equal to 96 bytes")
	}
	var in [2 * fpByteSize]byte
	copy(in[:], compressed[:])
	if in[0]&(1<<7) == 0 {
		return nil, errors.New("compression flag must be set")
	}
	if in[0]&(1<<6) != 0 {
		// in[0] == (1 << 6) + (1 << 7)
		for i, v := range in {
			if (i == 0 && v != 0xc0) || (i != 0 && v != 0x00) {
				return nil, errors.New("input string must be zero when infinity flag is set")
			}
		}
		return g.Zero(), nil
	}
	a := in[0]&(1<<5) != 0
	in[0] &= 0x1f
	x, err := g.f.fromBytes(in[:])
	if err != nil {
		return nil, err
	}
	// solve curve equation
	y := &fe2{}
	g.f.square(y, x)
	g.f.mul(y, y, x)
	fp2Add(y, y, b2)
	if ok := g.f.sqrt(y, y); !ok {
		return nil, errors.New("point is not on curve")
	}
	if y.signBE() == a {
		fp2Neg(y, y)
	}
	z := new(fe2).one()
	p := &PointG2{*x, *y, *z}
	if !g.InCorrectSubgroup(p) {
		return nil, errors.New("point is not on correct subgroup")
	}
	return p, nil
}

// ToCompressed given a G2 point returns bytes in compressed form of the point.
// Serialization rules are in line with zcash library. See below for details.
// https://github.com/zcash/librustzcash/blob/master/pairing/src/bls12_381/README.md#serialization
// https://docs.rs/bls12_381/0.1.1/bls12_381/notes/serialization/index.html
func (g *G2) ToCompressed(p *PointG2) []byte {
	out := make([]byte, 2*fpByteSize)
	g.Affine(p)
	if g.IsZero(p) {
		out[0] |= 1 << 6
	} else {
		copy(out[:], g.f.toBytes(&p[0]))
		if !p[1].signBE() {
			out[0] |= 1 << 5
		}
	}
	out[0] |= 1 << 7
	return out
}

func (g *G2) fromBytesUnchecked(in []byte) (*PointG2, error) {
	p0, err := g.f.fromBytes(in[:2*fpByteSize])
	if err != nil {
		return nil, err
	}
	p1, err := g.f.fromBytes(in[2*fpByteSize:])
	if err != nil {
		return nil, err
	}
	p2 := new(fe2).one()
	return &PointG2{*p0, *p1, *p2}, nil
}

// FromBytes constructs a new point given uncompressed byte input.
// Input string expected to be 192 bytes and concatenation of x and y values
// Point (0, 0) is considered as infinity.
func (g *G2) FromBytes(in []byte) (*PointG2, error) {
	if len(in) != 4*fpByteSize {
		return nil, errors.New("input string length must be equal to 192 bytes")
	}
	p0, err := g.f.fromBytes(in[:2*fpByteSize])
	if err != nil {
		return nil, err
	}
	p1, err := g.f.fromBytes(in[2*fpByteSize:])
	if err != nil {
		return nil, err
	}
	// check if given input points to infinity
	if p0.isZero() && p1.isZero() {
		return g.Zero(), nil
	}
	p2 := new(fe2).one()
	p := &PointG2{*p0, *p1, *p2}
	if !g.IsOnCurve(p) {
		return nil, errors.New("point is not on curve")
	}
	return p, nil
}

// ToBytes serializes a point into bytes in uncompressed form,
// returns (0, 0) if point is infinity.
func (g *G2) ToBytes(p *PointG2) []byte {
	out := make([]byte, 4*fpByteSize)
	if g.IsZero(p) {
		return out
	}
	g.Affine(p)
	copy(out[:2*fpByteSize], g.f.toBytes(&p[0]))
	copy(out[2*fpByteSize:], g.f.toBytes(&p[1]))
	return out
}

// New creates a new G2 Point which is equal to zero in other words point at infinity.
func (g *G2) New() *PointG2 {
	return new(PointG2).Zero()
}

// Zero returns a new G2 Point which is equal to point at infinity.
func (g *G2) Zero() *PointG2 {
	return new(PointG2).Zero()
}

// One returns a new G2 Point which is equal to generator point.
func (g *G2) One() *PointG2 {
	p := &PointG2{}
	return p.Set(&g2One)
}

// IsZero returns true if given point is equal to zero.
func (g *G2) IsZero(p *PointG2) bool {
	return p[2].isZero()
}

// Equal checks if given two G2 point is equal in their affine form.
func (g *G2) Equal(p1, p2 *PointG2) bool {
	if g.IsZero(p1) {
		return g.IsZero(p2)
	}
	if g.IsZero(p2) {
		return g.IsZero(p1)
	}
	t := g.t
	g.f.square(t[0], &p1[2])
	g.f.square(t[1], &p2[2])
	g.f.mul(t[2], t[0], &p2[0])
	g.f.mul(t[3], t[1], &p1[0])
	g.f.mulAssign(t[0], &p1[2])
	g.f.mulAssign(t[1], &p2[2])
	g.f.mulAssign(t[1], &p1[1])
	g.f.mulAssign(t[0], &p2[1])
	return t[0].equal(t[1]) && t[2].equal(t[3])
}

// IsOnCurve checks a G2 point is on curve.
func (g *G2) IsOnCurve(p *PointG2) bool {
	if g.IsZero(p) {
		return true
	}
	t := g.t
	g.f.square(t[0], &p[1])    // y^2
	g.f.square(t[1], &p[0])    // x^2
	g.f.mul(t[1], t[1], &p[0]) // x^3
	if p.IsAffine() {
		fp2Add(t[1], t[1], b2)  // x^2 + b
		return t[0].equal(t[1]) // y^2 ?= x^3 + b
	}
	g.f.square(t[2], &p[2])   // z^2
	g.f.square(t[3], t[2])    // z^4
	g.f.mulAssign(t[2], t[3]) // z^6
	g.f.mulAssign(t[2], b2)   // b*z^6
	fp2AddAssign(t[1], t[2])  // x^3 + b * z^6
	return t[0].equal(t[1])   // y^2 ?= x^3 + b * z^6
}

// IsAffine checks a G2 point whether it is in affine form.
func (g *G2) IsAffine(p *PointG2) bool {
	return p[2].isOne()
}

// Affine calculates affine form of given G2 point.
func (g *G2) Affine(p *PointG2) *PointG2 {
	return g.affine(p, p)
}

func (g *G2) affine(r, p *PointG2) *PointG2 {
	if g.IsZero(p) {
		return r.Zero()
	}
	if !g.IsAffine(p) {
		t := g.t
		g.f.inverse(t[0], &p[2])   // z^-1
		g.f.square(t[1], t[0])     // z^-2
		g.f.mulAssign(&r[0], t[1]) // x = x * z^-2
		g.f.mulAssign(t[0], t[1])  // z^-3
		g.f.mulAssign(&r[1], t[0]) // y = y * z^-3
		r[2].one()                 // z = 1
	} else {
		r.Set(p)
	}
	return r
}

// AffineBatch given multiple of points returns affine representations
func (g *G2) AffineBatch(p []*PointG2) {
	inverses := make([]fe2, len(p))
	for i := 0; i < len(p); i++ {
		inverses[i].set(&p[i][2])
	}
	g.f.inverseBatch(inverses)
	t := g.t
	for i := 0; i < len(p); i++ {
		if !g.IsAffine(p[i]) && !g.IsZero(p[i]) {
			g.f.square(t[1], &inverses[i])
			g.f.mulAssign(&p[i][0], t[1])
			g.f.mul(t[0], &inverses[i], t[1])
			g.f.mulAssign(&p[i][1], t[0])
			p[i][2].one()
		}
	}
}

// Add adds two G2 points p1, p2 and assigns the result to point at first argument.
func (g *G2) Add(r, p1, p2 *PointG2) *PointG2 {
	// http://www.hyperelliptic.org/EFD/gp/auto-shortw-jacobian-0.html#addition-add-2007-bl
	if g.IsZero(p1) {
		return r.Set(p2)
	}
	if g.IsZero(p2) {
		return r.Set(p1)
	}
	if g.IsAffine(p2) {
		return g.AddMixed(r, p1, p2)
	}
	t := g.t
	g.f.square(t[7], &p1[2])    // z1z1
	g.f.mul(t[1], &p2[0], t[7]) // u2 = x2 * z1z1
	g.f.mul(t[2], &p1[2], t[7]) // z1z1 * z1
	g.f.mul(t[0], &p2[1], t[2]) // s2 = y2 * z1z1 * z1
	g.f.square(t[8], &p2[2])    // z2z2
	g.f.mul(t[3], &p1[0], t[8]) // u1 = x1 * z2z2
	g.f.mul(t[4], &p2[2], t[8]) // z2z2 * z2
	g.f.mul(t[2], &p1[1], t[4]) // s1 = y1 * z2z2 * z2
	if t[1].equal(t[3]) {
		if t[0].equal(t[2]) {
			return g.Double(r, p1)
		} else {
			return r.Zero()
		}
	}
	fp2SubAssign(t[1], t[3])     // h = u2 - u1
	fp2Double(t[4], t[1])        // 2h
	g.f.squareAssign(t[4])       // i = 2h^2
	g.f.mul(t[5], t[1], t[4])    // j = h*i
	fp2SubAssign(t[0], t[2])     // s2 - s1
	fp2DoubleAssign(t[0])        // r = 2*(s2 - s1)
	g.f.square(t[6], t[0])       // r^2
	fp2SubAssign(t[6], t[5])     // r^2 - j
	g.f.mulAssign(t[3], t[4])    // v = u1 * i
	fp2Double(t[4], t[3])        // 2*v
	fp2Sub(&r[0], t[6], t[4])    // x3 = r^2 - j - 2*v
	fp2Sub(t[4], t[3], &r[0])    // v - x3
	g.f.mul(t[6], t[2], t[5])    // s1 * j
	fp2DoubleAssign(t[6])        // 2 * s1 * j
	g.f.mulAssign(t[0], t[4])    // r * (v - x3)
	fp2Sub(&r[1], t[0], t[6])    // y3 = r * (v - x3) - (2 * s1 * j)
	fp2Add(t[0], &p1[2], &p2[2]) // z1 + z2
	g.f.squareAssign(t[0])       // (z1 + z2)^2
	fp2SubAssign(t[0], t[7])     // (z1 + z2)^2 - z1z1
	fp2SubAssign(t[0], t[8])     // (z1 + z2)^2 - z1z1 - z2z2
	g.f.mul(&r[2], t[0], t[1])   // z3 = ((z1 + z2)^2 - z1z1 - z2z2) * h
	return r
}

// Add adds two G1 points p1, p2 and assigns the result to point at first argument.
// Expects the second point p2 in affine form.
func (g *G2) AddMixed(r, p1, p2 *PointG2) *PointG2 {
	// http://www.hyperelliptic.org/EFD/g1p/auto-shortw-jacobian-0.html#addition-madd-2007-bl
	if g.IsZero(p1) {
		return r.Set(p2)
	}
	if g.IsZero(p2) {
		return r.Set(p1)
	}
	t := g.t
	g.f.square(t[7], &p1[2])    // z1z1
	g.f.mul(t[1], &p2[0], t[7]) // u2 = x2 * z1z1
	g.f.mul(t[2], &p1[2], t[7]) // z1z1 * z1
	g.f.mul(t[0], &p2[1], t[2]) // s2 = y2 * z1z1 * z1

	if p1[0].equal(t[1]) && p1[1].equal(t[0]) {
		return g.Double(r, p1)
	}

	fp2SubAssign(t[1], &p1[0]) // h = u2 - x1
	g.f.square(t[2], t[1])     // hh
	fp2Double(t[4], t[2])
	fp2DoubleAssign(t[4])       // 4hh
	g.f.mul(t[5], t[1], t[4])   // j = h*i
	fp2SubAssign(t[0], &p1[1])  // s2 - y1
	fp2DoubleAssign(t[0])       // r = 2*(s2 - y1)
	g.f.square(t[6], t[0])      // r^2
	fp2SubAssign(t[6], t[5])    // r^2 - j
	g.f.mul(t[3], &p1[0], t[4]) // v = x1 * i
	fp2Double(t[4], t[3])       // 2*v
	fp2Sub(&r[0], t[6], t[4])   // x3 = r^2 - j - 2*v
	fp2Sub(t[4], t[3], &r[0])   // v - x3
	g.f.mul(t[6], &p1[1], t[5]) // y1 * j
	fp2DoubleAssign(t[6])       // 2 * y1 * j
	g.f.mulAssign(t[0], t[4])   // r * (v - x3)
	fp2Sub(&r[1], t[0], t[6])   // y3 = r * (v - x3) - (2 * y1 * j)
	fp2Add(t[0], &p1[2], t[1])  // z1 + h
	g.f.squareAssign(t[0])      // (z1 + h)^2
	fp2SubAssign(t[0], t[7])    // (z1 + h)^2 - z1z1
	fp2Sub(&r[2], t[0], t[2])   // z3 = (z1 + z2)^2 - z1z1 - hh
	return r
}

// Double doubles a G2 point p and assigns the result to the point at first argument.
func (g *G2) Double(r, p *PointG2) *PointG2 {
	// http://www.hyperelliptic.org/EFD/gp/auto-shortw-jacobian-0.html#doubling-dbl-2009-l
	if g.IsZero(p) {
		return r.Set(p)
	}
	t := g.t
	g.f.square(t[0], &p[0])     // a = x^2
	g.f.square(t[1], &p[1])     // b = y^2
	g.f.square(t[2], t[1])      // c = b^2
	fp2AddAssign(t[1], &p[0])   // b + x1
	g.f.squareAssign(t[1])      // (b + x1)^2
	fp2SubAssign(t[1], t[0])    // (b + x1)^2 - a
	fp2SubAssign(t[1], t[2])    // (b + x1)^2 - a - c
	fp2DoubleAssign(t[1])       // d = 2((b+x1)^2 - a - c)
	fp2Double(t[3], t[0])       // 2a
	fp2AddAssign(t[0], t[3])    // e = 3a
	g.f.square(t[4], t[0])      // f = e^2
	fp2Double(t[3], t[1])       // 2d
	fp2Sub(&r[0], t[4], t[3])   // x3 = f - 2d
	fp2SubAssign(t[1], &r[0])   // d-x3
	fp2DoubleAssign(t[2])       //
	fp2DoubleAssign(t[2])       //
	fp2DoubleAssign(t[2])       // 8c
	g.f.mulAssign(t[0], t[1])   // e * (d - x3)
	fp2Sub(t[1], t[0], t[2])    // x3 = e * (d - x3) - 8c
	g.f.mul(t[0], &p[1], &p[2]) // y1 * z1
	r[1].set(t[1])              //
	fp2Double(&r[2], t[0])      // z3 = 2(y1 * z1)
	return r
}

// Neg negates a G2 point p and assigns the result to the point at first argument.
func (g *G2) Neg(r, p *PointG2) *PointG2 {
	r[0].set(&p[0])
	fp2Neg(&r[1], &p[1])
	r[2].set(&p[2])
	return r
}

// Sub subtracts two G2 points p1, p2 and assigns the result to point at first argument.
func (g *G2) Sub(c, a, b *PointG2) *PointG2 {
	d := &PointG2{}
	g.Neg(d, b)
	g.Add(c, a, d)
	return c
}

// MulScalar multiplies a point by given scalar value and assigns the result to point at first argument.
func (g *G2) MulScalar(r, p *PointG2, e *Fr) *PointG2 {
	return g.glvMulFr(r, p, e)
}

// MulScalarBig multiplies a point by given scalar value in big.Int and assigns the result to point at first argument.
func (g *G2) MulScalarBig(r, p *PointG2, e *big.Int) *PointG2 {
	return g.glvMulBig(r, p, e)
}

func (g *G2) mulScalar(c, p *PointG2, e *Fr) *PointG2 {
	q, n := &PointG2{}, &PointG2{}
	n.Set(p)
	for i := 0; i < frBitSize; i++ {
		if e.Bit(i) {
			g.Add(q, q, n)
		}
		g.Double(n, n)
	}
	return c.Set(q)
}

func (g *G2) mulScalarBig(c, p *PointG2, e *big.Int) *PointG2 {
	q, n := &PointG2{}, &PointG2{}
	n.Set(p)
	l := e.BitLen()
	for i := 0; i < l; i++ {
		if e.Bit(i) == 1 {
			g.Add(q, q, n)
		}
		g.Double(n, n)
	}
	return c.Set(q)
}

func (g *G2) wnafMulFr(r, p *PointG2, e *Fr) *PointG2 {
	wnaf := e.toWNAF(wnafMulWindowG2)
	return g.wnafMul(r, p, wnaf)
}

func (g *G2) wnafMulBig(r, p *PointG2, e *big.Int) *PointG2 {
	wnaf := bigToWNAF(e, wnafMulWindowG2)
	return g.wnafMul(r, p, wnaf)
}

func (g *G2) wnafMul(c, p *PointG2, wnaf nafNumber) *PointG2 {

	l := (1 << (wnafMulWindowG2 - 1))

	twoP, acc := g.New(), new(PointG2).Set(p)
	g.Double(twoP, p)
	g.Affine(twoP)

	// table = {p, 3p, 5p, ..., -p, -3p, -5p}
	table := make([]*PointG2, l*2)
	table[0], table[l] = g.New(), g.New()
	table[0].Set(p)
	g.Neg(table[l], table[0])

	for i := 1; i < l; i++ {
		g.AddMixed(acc, acc, twoP)
		table[i], table[i+l] = g.New(), g.New()
		table[i].Set(acc)
		g.Neg(table[i+l], table[i])
	}

	q := g.Zero()
	for i := len(wnaf) - 1; i >= 0; i-- {
		if wnaf[i] > 0 {
			g.Add(q, q, table[wnaf[i]>>1])
		} else if wnaf[i] < 0 {
			g.Add(q, q, table[((-wnaf[i])>>1)+l])
		}
		if i != 0 {
			g.Double(q, q)
		}
	}
	return c.Set(q)
}

func (g *G2) glvMulFr(r, p *PointG2, e *Fr) *PointG2 {
	return g.glvMul(r, p, new(glvVectorFr).new(e))
}

func (g *G2) glvMulBig(r, p *PointG2, e *big.Int) *PointG2 {
	return g.glvMul(r, p, new(glvVectorBig).new(e))
}

func (g *G2) glvMul(r, p0 *PointG2, v glvVector) *PointG2 {

	w := glvMulWindowG2
	l := 1 << (w - 1)

	// prepare tables
	// tableK1 = {P, 3P, 5P, ...}
	// tableK2 = {λP, 3λP, 5λP, ...}
	tableK1, tableK2 := make([]*PointG2, l), make([]*PointG2, l)
	double := g.New()
	g.Double(double, p0)
	g.affine(double, double)
	tableK1[0] = new(PointG2)
	tableK1[0].Set(p0)
	for i := 1; i < l; i++ {
		tableK1[i] = new(PointG2)
		g.AddMixed(tableK1[i], tableK1[i-1], double)
	}
	g.AffineBatch(tableK1)
	for i := 0; i < l; i++ {
		tableK2[i] = new(PointG2)
		g.glvEndomorphism(tableK2[i], tableK1[i])
	}

	// recode small scalars
	naf1, naf2 := v.wnaf(w)
	lenNAF1, lenNAF2 := len(naf1), len(naf2)
	lenNAF := lenNAF1
	if lenNAF2 > lenNAF {
		lenNAF = lenNAF2
	}

	acc, p1 := g.New(), g.New()

	// function for naf addition
	add := func(table []*PointG2, naf int) {
		if naf != 0 {
			nafAbs := naf
			if nafAbs < 0 {
				nafAbs = -nafAbs
			}
			p1.Set(table[nafAbs>>1])
			if naf < 0 {
				g.Neg(p1, p1)
			}
			g.AddMixed(acc, acc, p1)
		}
	}

	// sliding
	for i := lenNAF - 1; i >= 0; i-- {
		if i < lenNAF1 {
			add(tableK1, naf1[i])
		}
		if i < lenNAF2 {
			add(tableK2, naf2[i])
		}
		if i != 0 {
			g.Double(acc, acc)
		}
	}
	return r.Set(acc)
}

// MultiExpBig calculates multi exponentiation. Scalar values are received as big.Int type.
// Given pairs of G2 point and scalar values `(P_0, e_0), (P_1, e_1), ... (P_n, e_n)`,
// calculates `r = e_0 * P_0 + e_1 * P_1 + ... + e_n * P_n`.
// Length of points and scalars are expected to be equal, otherwise an error is returned.
// Result is assigned to point at first argument.
func (g *G2) MultiExpBig(r *PointG2, points []*PointG2, scalars []*big.Int) (*PointG2, error) {
	if len(points) != len(scalars) {
		return nil, errors.New("point and scalar vectors should be in same length")
	}

	c := 3
	if len(scalars) >= 32 {
		c = int(math.Ceil(math.Log(float64(len(scalars)))))
	}

	bucketSize := (1 << c) - 1
	windows := make([]PointG2, 255/c+1)
	bucket := make([]PointG2, bucketSize)

	for j := 0; j < len(windows); j++ {

		for i := 0; i < bucketSize; i++ {
			bucket[i].Zero()
		}

		for i := 0; i < len(scalars); i++ {
			index := bucketSize & int(new(big.Int).Rsh(scalars[i], uint(c*j)).Int64())
			if index != 0 {
				g.Add(&bucket[index-1], &bucket[index-1], points[i])
			}
		}

		acc, sum := g.New(), g.New()
		for i := bucketSize - 1; i >= 0; i-- {
			g.Add(sum, sum, &bucket[i])
			g.Add(acc, acc, sum)
		}
		windows[j].Set(acc)
	}

	acc := g.New()
	for i := len(windows) - 1; i >= 0; i-- {
		for j := 0; j < c; j++ {
			g.Double(acc, acc)
		}
		g.Add(acc, acc, &windows[i])
	}
	return r.Set(acc), nil
}

// MultiExp calculates multi exponentiation. Given pairs of G2 point and scalar values `(P_0, e_0), (P_1, e_1), ... (P_n, e_n)`,
// calculates `r = e_0 * P_0 + e_1 * P_1 + ... + e_n * P_n`. Length of points and scalars are expected to be equal,
// otherwise an error is returned. Result is assigned to point at first argument.
func (g *G2) MultiExp(r *PointG2, points []*PointG2, scalars []*Fr) (*PointG2, error) {
	if len(points) != len(scalars) {
		return nil, errors.New("point and scalar vectors should be in same length")
	}

	g.AffineBatch(points)

	c := 3
	if len(scalars) >= 32 {
		c = int(math.Ceil(math.Log(float64(len(scalars)))))
	}

	bucketSize := (1 << c) - 1
	windows := make([]*PointG2, 255/c+1)
	bucket := make([]PointG2, bucketSize)

	for j := 0; j < len(windows); j++ {

		for i := 0; i < bucketSize; i++ {
			bucket[i].Zero()
		}

		for i := 0; i < len(scalars); i++ {
			index := bucketSize & int(scalars[i].sliceUint64(c*j))
			if index != 0 {
				g.AddMixed(&bucket[index-1], &bucket[index-1], points[i])
			}
		}

		acc, sum := g.New(), g.New()
		for i := bucketSize - 1; i >= 0; i-- {
			g.Add(sum, sum, &bucket[i])
			g.Add(acc, acc, sum)
		}
		windows[j] = g.New().Set(acc)
	}

	g.AffineBatch(windows)

	acc := g.New()
	for i := len(windows) - 1; i >= 0; i-- {
		for j := 0; j < c; j++ {
			g.Double(acc, acc)
		}
		g.AddMixed(acc, acc, windows[i])
	}
	return r.Set(acc), nil
}

// InCorrectSubgroup checks whether given point is in correct subgroup.
func (g *G2) InCorrectSubgroup(p *PointG2) bool {

	// Faster Subgroup Checks for BLS12-381
	// S. Bowe
	// https://eprint.iacr.org/2019/814.pdf

	// [z]ψ^3(P) − ψ^2(P) + P = O
	t0, t1 := g.New().Set(p), g.New()

	g.psi(t0)
	g.psi(t0)
	g.Neg(t1, t0) // - ψ^2(P)
	g.psi(t0)     // ψ^3(P)
	g.mulX(t0)    // - x ψ^3(P)
	g.Neg(t0, t0)

	g.Add(t0, t0, t1)
	g.Add(t0, t0, p)

	return g.IsZero(t0)
}

// ClearCofactor maps given a G2 point to correct subgroup
func (g *G2) ClearCofactor(p *PointG2) *PointG2 {

	// Efficient hash maps to G2 on BLS curves
	// A. Budroni, F. Pintore
	// https://eprint.iacr.org/2017/419.pdf

	// [h(ψ)]P = [x^2 − x − 1]P + [x − 1]ψ(P) + ψ^2(2P)
	t0, t1, t2, t3 := g.New().Set(p), g.New().Set(p), g.New().Set(p), g.New()

	g.Double(t0, t0)
	g.psi(t0)
	g.psi(t0)  // P2 = ψ^2(2P)
	g.psi(t2)  // P1 = ψ(P)
	g.mulX(t1) // -xP0

	g.Sub(t3, t1, t2) // -xP0 - P1
	g.mulX(t3)        // (x^2)P0 + xP1
	g.Sub(t1, t1, p)  // (-x-1)P0
	g.Add(t3, t3, t1) // (x^2-x-1)P0 + xP1
	g.Sub(t3, t3, t2) // (x^2-x-1)P0 + (x-1)P1
	g.Add(t3, t3, t0) // (x^2-x-1)P0 + (x-1)P1 + P2
	return p.Set(t3)
}

func (g *G2) psi(p *PointG2) {
	fp2Conjugate(&p[0], &p[0])
	fp2Conjugate(&p[1], &p[1])
	fp2Conjugate(&p[2], &p[2])
	g.f.mul(&p[0], &p[0], &psix)
	g.f.mul(&p[1], &p[1], &psiy)
}

func (g *G2) mulX(p *PointG2) {

	chain := func(p0 *PointG2, n int, p1 *PointG2) {
		g.Add(p0, p0, p1)
		for i := 0; i < n; i++ {
			g.Double(p0, p0)
		}
	}

	t := g.New().Set(p)
	g.Double(p, t)
	chain(p, 2, t)
	chain(p, 3, t)
	chain(p, 9, t)
	chain(p, 32, t)
	chain(p, 16, t)
}

// MapToCurve given a byte slice returns a valid G2 point.
// This mapping function implements the Simplified Shallue-van de Woestijne-Ulas method.
// https://tools.ietf.org/html/draft-irtf-cfrg-hash-to-curve-05#section-6.6.2
// Input byte slice should be a valid field element, otherwise an error is returned.
func (g *G2) MapToCurve(in []byte) (*PointG2, error) {
	fp2 := g.f
	u, err := fp2.fromBytes(in)
	if err != nil {
		return nil, err
	}
	x, y := swuMapG2(fp2, u)
	isogenyMapG2(fp2, x, y)
	z := new(fe2).one()
	q := &PointG2{*x, *y, *z}
	g.ClearCofactor(q)
	return g.Affine(q), nil
}

// EncodeToCurve given a message and domain seperator tag returns the hash result
// which is a valid curve point.
// Implementation follows BLS12381G1_XMD:SHA-256_SSWU_NU_ suite at
// https://tools.ietf.org/html/draft-irtf-cfrg-hash-to-curve-06
func (g *G2) EncodeToCurve(msg, domain []byte) (*PointG2, error) {
	hashRes, err := hashToFpXMDSHA256(msg, domain, 2)
	if err != nil {
		return nil, err
	}
	fp2 := g.f
	u := &fe2{*hashRes[0], *hashRes[1]}
	x, y := swuMapG2(fp2, u)
	isogenyMapG2(fp2, x, y)
	z := new(fe2).one()
	q := &PointG2{*x, *y, *z}
	g.ClearCofactor(q)
	return g.Affine(q), nil
}

// HashToCurve given a message and domain seperator tag returns the hash result
// which is a valid curve point.
// Implementation follows BLS12381G1_XMD:SHA-256_SSWU_RO_ suite at
// https://tools.ietf.org/html/draft-irtf-cfrg-hash-to-curve-06
func (g *G2) HashToCurve(msg, domain []byte) (*PointG2, error) {
	hashRes, err := hashToFpXMDSHA256(msg, domain, 4)
	if err != nil {
		return nil, err
	}
	fp2 := g.f
	u0, u1 := &fe2{*hashRes[0], *hashRes[1]}, &fe2{*hashRes[2], *hashRes[3]}
	x0, y0 := swuMapG2(fp2, u0)
	x1, y1 := swuMapG2(fp2, u1)
	z0 := new(fe2).one()
	z1 := new(fe2).one()
	p0, p1 := &PointG2{*x0, *y0, *z0}, &PointG2{*x1, *y1, *z1}
	g.Add(p0, p0, p1)
	g.Affine(p0)
	isogenyMapG2(fp2, &p0[0], &p0[1])
	g.ClearCofactor(p0)
	return g.Affine(p0), nil
}
