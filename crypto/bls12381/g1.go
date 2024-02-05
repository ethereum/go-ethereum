package bls12381

import (
	"errors"
	"math"
	"math/big"
)

// PointG1 is type for point in G1 and used for both Affine and Jacobian point representation.
// A point is accounted as in affine form if z is equal to one.
type PointG1 [3]fe

var wnafMulWindowG1 uint = 5

func (p *PointG1) Set(p2 *PointG1) *PointG1 {
	p[0].set(&p2[0])
	p[1].set(&p2[1])
	p[2].set(&p2[2])
	return p
}

func (p *PointG1) Zero() *PointG1 {
	p[0].zero()
	p[1].one()
	p[2].zero()
	return p
}

// IsAffine checks a G1 point whether it is in affine form.
func (p *PointG1) IsAffine() bool {
	return p[2].isOne()
}

type tempG1 struct {
	t [9]*fe
}

// G1 is struct for G1 group.
type G1 struct {
	tempG1
}

// NewG1 constructs a new G1 instance.
func NewG1() *G1 {
	t := newTempG1()
	return &G1{t}
}

func newTempG1() tempG1 {
	t := [9]*fe{}
	for i := 0; i < 9; i++ {
		t[i] = &fe{}
	}
	return tempG1{t}
}

// Q returns group order in big.Int.
func (g *G1) Q() *big.Int {
	return new(big.Int).Set(qBig)
}

// FromUncompressed expects byte slice at least 96 bytes and given bytes returns a new point in G1.
// Serialization rules are in line with zcash library. See below for details.
// https://github.com/zcash/librustzcash/blob/master/pairing/src/bls12_381/README.md#serialization
// https://docs.rs/bls12_381/0.1.1/bls12_381/notes/serialization/index.html
func (g *G1) FromUncompressed(uncompressed []byte) (*PointG1, error) {
	if len(uncompressed) != 2*fpByteSize {
		return nil, errors.New("input string length must be equal to 96 bytes")
	}
	var in [2 * fpByteSize]byte
	copy(in[:], uncompressed[:2*fpByteSize])
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
	x, err := fromBytes(in[:fpByteSize])
	if err != nil {
		return nil, err
	}
	y, err := fromBytes(in[fpByteSize:])
	if err != nil {
		return nil, err
	}
	z := new(fe).one()
	p := &PointG1{*x, *y, *z}
	if !g.IsOnCurve(p) {
		return nil, errors.New("point is not on curve")
	}
	if !g.InCorrectSubgroup(p) {
		return nil, errors.New("point is not on correct subgroup")
	}
	return p, nil
}

// ToUncompressed given a G1 point returns bytes in uncompressed (x, y) form of the point.
// Serialization rules are in line with zcash library. See below for details.
// https://github.com/zcash/librustzcash/blob/master/pairing/src/bls12_381/README.md#serialization
// https://docs.rs/bls12_381/0.1.1/bls12_381/notes/serialization/index.html
func (g *G1) ToUncompressed(p *PointG1) []byte {
	out := make([]byte, 2*fpByteSize)
	if g.IsZero(p) {
		out[0] |= 1 << 6
		return out
	}
	g.Affine(p)
	copy(out[:fpByteSize], toBytes(&p[0]))
	copy(out[fpByteSize:], toBytes(&p[1]))
	return out
}

// FromCompressed expects byte slice at least 48 bytes and given bytes returns a new point in G1.
// Serialization rules are in line with zcash library. See below for details.
// https://github.com/zcash/librustzcash/blob/master/pairing/src/bls12_381/README.md#serialization
// https://docs.rs/bls12_381/0.1.1/bls12_381/notes/serialization/index.html
func (g *G1) FromCompressed(compressed []byte) (*PointG1, error) {
	if len(compressed) != fpByteSize {
		return nil, errors.New("input string length must be equal to 48 bytes")
	}
	var in [fpByteSize]byte
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
	x, err := fromBytes(in[:])
	if err != nil {
		return nil, err
	}
	// solve curve equation
	y := &fe{}
	square(y, x)
	mul(y, y, x)
	add(y, y, b)
	if ok := sqrt(y, y); !ok {
		return nil, errors.New("point is not on curve")
	}
	if y.signBE() == a {
		neg(y, y)
	}
	z := new(fe).one()
	p := &PointG1{*x, *y, *z}
	if !g.InCorrectSubgroup(p) {
		return nil, errors.New("point is not on correct subgroup")
	}
	return p, nil
}

// ToCompressed given a G1 point returns bytes in compressed form of the point.
// Serialization rules are in line with zcash library. See below for details.
// https://github.com/zcash/librustzcash/blob/master/pairing/src/bls12_381/README.md#serialization
// https://docs.rs/bls12_381/0.1.1/bls12_381/notes/serialization/index.html
func (g *G1) ToCompressed(p *PointG1) []byte {
	out := make([]byte, fpByteSize)
	g.Affine(p)
	if g.IsZero(p) {
		out[0] |= 1 << 6
	} else {
		copy(out[:], toBytes(&p[0]))
		if !p[1].signBE() {
			out[0] |= 1 << 5
		}
	}
	out[0] |= 1 << 7
	return out
}

func (g *G1) fromBytesUnchecked(in []byte) (*PointG1, error) {
	p0, err := fromBytes(in[:fpByteSize])
	if err != nil {
		return nil, err
	}
	p1, err := fromBytes(in[fpByteSize:])
	if err != nil {
		return nil, err
	}
	p2 := new(fe).one()
	return &PointG1{*p0, *p1, *p2}, nil
}

// FromBytes constructs a new point given uncompressed byte input.
// Input string is expected to be equal to 96 bytes and concatenation of x and y cooridanates.
// (0, 0) is considered as infinity.
func (g *G1) FromBytes(in []byte) (*PointG1, error) {
	if len(in) != 2*fpByteSize {
		return nil, errors.New("input string length must be equal to 96 bytes")
	}
	p0, err := fromBytes(in[:fpByteSize])
	if err != nil {
		return nil, err
	}
	p1, err := fromBytes(in[fpByteSize:])
	if err != nil {
		return nil, err
	}
	// check if given input points to infinity
	if p0.isZero() && p1.isZero() {
		return g.Zero(), nil
	}
	p2 := new(fe).one()
	p := &PointG1{*p0, *p1, *p2}
	if !g.IsOnCurve(p) {
		return nil, errors.New("point is not on curve")
	}
	return p, nil
}

// ToBytes serializes a point into bytes in uncompressed form.
// ToBytes returns (0, 0) if point is infinity.
func (g *G1) ToBytes(p *PointG1) []byte {
	out := make([]byte, 2*fpByteSize)
	if g.IsZero(p) {
		return out
	}
	g.Affine(p)
	copy(out[:fpByteSize], toBytes(&p[0]))
	copy(out[fpByteSize:], toBytes(&p[1]))
	return out
}

// New creates a new G1 Point which is equal to zero in other words point at infinity.
func (g *G1) New() *PointG1 {
	return g.Zero()
}

// Zero returns a new G1 Point which is equal to point at infinity.
func (g *G1) Zero() *PointG1 {
	return new(PointG1).Zero()
}

// One returns a new G1 Point which is equal to generator point.
func (g *G1) One() *PointG1 {
	p := &PointG1{}
	return p.Set(&g1One)
}

// IsZero returns true if given point is equal to zero.
func (g *G1) IsZero(p *PointG1) bool {
	return p[2].isZero()
}

// Equal checks if given two G1 point is equal in their affine form.
func (g *G1) Equal(p1, p2 *PointG1) bool {
	if g.IsZero(p1) {
		return g.IsZero(p2)
	}
	if g.IsZero(p2) {
		return g.IsZero(p1)
	}
	t := g.t
	square(t[0], &p1[2])
	square(t[1], &p2[2])
	mul(t[2], t[0], &p2[0])
	mul(t[3], t[1], &p1[0])
	mul(t[0], t[0], &p1[2])
	mul(t[1], t[1], &p2[2])
	mul(t[1], t[1], &p1[1])
	mul(t[0], t[0], &p2[1])
	return t[0].equal(t[1]) && t[2].equal(t[3])
}

// InCorrectSubgroup checks whether given point is in correct subgroup.
func (g *G1) InCorrectSubgroup(p *PointG1) bool {

	// Faster Subgroup Checks for BLS12-381
	// S. Bowe
	// https://eprint.iacr.org/2019/814.pdf

	mulZ := func(p *PointG1) {
		// z = [(x^2 − 1)/3]
		z := &Fr{0x0000000055555555, 0x396c8c005555e156}
		e := z.toWNAF(wnafMulWindowG1)
		g.wnafMul(p, p, e)
	}

	// [(x^2 − 1)/3](2σ(P) − P − σ^2(P)) − σ^2(P) ?= O
	t0 := g.New().Set(p)
	g.glvEndomorphism(t0, t0)
	t1 := g.New().Set(t0)     // σ(P)
	g.glvEndomorphism(t0, t0) // σ^2(P)
	g.Double(t1, t1)          // 2σ(P)
	g.Sub(t1, t1, p)          // 2σ(P) − P
	g.Sub(t1, t1, t0)         // 2σ(P) − P − σ^2(P)
	mulZ(t1)                  // [(x^2 − 1)/3](2σ(P) − P − σ^2(P))
	g.Sub(t1, t1, t0)         // [(x^2 − 1)/3](2σ(P) − P − σ^2(P)) − σ^2(P)
	return g.IsZero(t1)
}

// IsOnCurve checks a G1 point is on curve.
func (g *G1) IsOnCurve(p *PointG1) bool {
	if g.IsZero(p) {
		return true
	}
	t := g.t
	square(t[0], &p[1])    // y^2
	square(t[1], &p[0])    // x^2
	mul(t[1], t[1], &p[0]) // x^3
	if p.IsAffine() {
		addAssign(t[1], b)      // x^2 + b
		return t[0].equal(t[1]) // y^2 ?= x^3 + b
	}
	square(t[2], &p[2])     // z^2
	square(t[3], t[2])      // z^4
	mul(t[2], t[2], t[3])   // z^6
	mul(t[2], b, t[2])      // b * z^6
	add(t[1], t[1], t[2])   // x^3 + b * z^6
	return t[0].equal(t[1]) // y^2 ?= x^3 + b * z^6
}

// IsAffine checks a G1 point whether it is in affine form.
func (g *G1) IsAffine(p *PointG1) bool {
	return p[2].isOne()
}

// Affine returns the affine representation of the given point
func (g *G1) Affine(p *PointG1) *PointG1 {
	return g.affine(p, p)
}

func (g *G1) affine(r, p *PointG1) *PointG1 {
	if g.IsZero(p) {
		return r.Zero()
	}
	if !g.IsAffine(p) {
		t := g.t
		inverse(t[0], &p[2])    // z^-1
		square(t[1], t[0])      // z^-2
		mul(&r[0], &p[0], t[1]) // x = x * z^-2
		mul(t[0], t[0], t[1])   // z^-3
		mul(&r[1], &p[1], t[0]) // y = y * z^-3
		r[2].one()              // z = 1
	} else {
		r.Set(p)
	}
	return r
}

// AffineBatch given multiple of points returns affine representations
func (g *G1) AffineBatch(p []*PointG1) {
	inverses := make([]fe, len(p))
	for i := 0; i < len(p); i++ {
		inverses[i].set(&p[i][2])
	}
	inverseBatch(inverses)
	t := g.t
	for i := 0; i < len(p); i++ {
		if !g.IsAffine(p[i]) && !g.IsZero(p[i]) {
			square(t[1], &inverses[i])
			mul(&p[i][0], &p[i][0], t[1])
			mul(t[0], &inverses[i], t[1])
			mul(&p[i][1], &p[i][1], t[0])
			p[i][2].one()
		}
	}
}

// Add adds two G1 points p1, p2 and assigns the result to point at first argument.
func (g *G1) Add(r, p1, p2 *PointG1) *PointG1 {

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
	square(t[7], &p1[2])    // z1z1
	mul(t[1], &p2[0], t[7]) // u2 = x2 * z1z1
	mul(t[2], &p1[2], t[7]) // z1z1 * z1
	mul(t[0], &p2[1], t[2]) // s2 = y2 * z1z1 * z1
	square(t[8], &p2[2])    // z2z2
	mul(t[3], &p1[0], t[8]) // u1 = x1 * z2z2
	mul(t[4], &p2[2], t[8]) // z2z2 * z2
	mul(t[2], &p1[1], t[4]) // s1 = y1 * z2z2 * z2
	if t[1].equal(t[3]) {
		if t[0].equal(t[2]) {
			return g.Double(r, p1)
		} else {
			return r.Zero()
		}
	}
	subAssign(t[1], t[3])     // h = u2 - u1
	double(t[4], t[1])        // 2h
	square(t[4], t[4])        // i = 2h^2
	mul(t[5], t[1], t[4])     // j = h*i
	subAssign(t[0], t[2])     // s2 - s1
	doubleAssign(t[0])        // r = 2*(s2 - s1)
	square(t[6], t[0])        // r^2
	subAssign(t[6], t[5])     // r^2 - j
	mul(t[3], t[3], t[4])     // v = u1 * i
	double(t[4], t[3])        // 2*v
	sub(&r[0], t[6], t[4])    // x3 = r^2 - j - 2*v
	sub(t[4], t[3], &r[0])    // v - x3
	mul(t[6], t[2], t[5])     // s1 * j
	doubleAssign(t[6])        // 2 * s1 * j
	mul(t[0], t[0], t[4])     // r * (v - x3)
	sub(&r[1], t[0], t[6])    // y3 = r * (v - x3) - (2 * s1 * j)
	add(t[0], &p1[2], &p2[2]) // z1 + z2
	square(t[0], t[0])        // (z1 + z2)^2
	subAssign(t[0], t[7])     // (z1 + z2)^2 - z1z1
	subAssign(t[0], t[8])     // (z1 + z2)^2 - z1z1 - z2z2
	mul(&r[2], t[0], t[1])    // z3 = ((z1 + z2)^2 - z1z1 - z2z2) * h
	return r
}

// Add adds two G1 points p1, p2 and assigns the result to point at first argument.
// Expects the second point p2 in affine form.
func (g *G1) AddMixed(r, p1, p2 *PointG1) *PointG1 {
	// http://www.hyperelliptic.org/EFD/g1p/auto-shortw-jacobian-0.html#addition-madd-2007-bl
	if g.IsZero(p1) {
		return r.Set(p2)
	}
	if g.IsZero(p2) {
		return r.Set(p1)
	}
	t := g.t
	square(t[7], &p1[2])    // z1z1
	mul(t[1], &p2[0], t[7]) // u2 = x2 * z1z1
	mul(t[2], &p1[2], t[7]) // z1z1 * z1
	mul(t[0], &p2[1], t[2]) // s2 = y2 * z1z1 * z1

	if p1[0].equal(t[1]) && p1[1].equal(t[0]) {
		return g.Double(r, p1)
	}

	sub(t[1], t[1], &p1[0]) // h = u2 - x1
	square(t[2], t[1])      // hh
	double(t[4], t[2])
	doubleAssign(t[4])      // 4hh
	mul(t[5], t[1], t[4])   // j = h*i
	subAssign(t[0], &p1[1]) // s2 - y1
	doubleAssign(t[0])      // r = 2*(s2 - y1)
	square(t[6], t[0])      // r^2
	subAssign(t[6], t[5])   // r^2 - j
	mul(t[3], &p1[0], t[4]) // v = x1 * i
	double(t[4], t[3])      // 2*v
	sub(&r[0], t[6], t[4])  // x3 = r^2 - j - 2*v
	sub(t[4], t[3], &r[0])  // v - x3
	mul(t[6], &p1[1], t[5]) // y1 * j
	doubleAssign(t[6])      // 2 * y1 * j
	mul(t[0], t[0], t[4])   // r * (v - x3)
	sub(&r[1], t[0], t[6])  // y3 = r * (v - x3) - (2 * y1 * j)
	add(t[0], &p1[2], t[1]) // z1 + h
	square(t[0], t[0])      // (z1 + h)^2
	subAssign(t[0], t[7])   // (z1 + h)^2 - z1z1
	sub(&r[2], t[0], t[2])  // z3 = (z1 + z2)^2 - z1z1 - hh
	return r
}

// Double doubles a G1 point p and assigns the result to the point at first argument.
func (g *G1) Double(r, p *PointG1) *PointG1 {
	// http://www.hyperelliptic.org/EFD/gp/auto-shortw-jacobian-0.html#doubling-dbl-2009-l
	if g.IsZero(p) {
		return r.Zero()
	}
	t := g.t
	square(t[0], &p[0])     // a = x^2
	square(t[1], &p[1])     // b = y^2
	square(t[2], t[1])      // c = b^2
	add(t[1], &p[0], t[1])  // b + x1
	square(t[1], t[1])      // (b + x1)^2
	subAssign(t[1], t[0])   // (b + x1)^2 - a
	subAssign(t[1], t[2])   // (b + x1)^2 - a - c
	doubleAssign(t[1])      // d = 2((b+x1)^2 - a - c)
	double(t[3], t[0])      // 2a
	addAssign(t[0], t[3])   // e = 3a
	square(t[4], t[0])      // f = e^2
	double(t[3], t[1])      // 2d
	sub(&r[0], t[4], t[3])  // x3 = f - 2d
	subAssign(t[1], &r[0])  // d-x3
	doubleAssign(t[2])      //
	doubleAssign(t[2])      //
	doubleAssign(t[2])      // 8c
	mul(t[0], t[0], t[1])   // e * (d - x3)
	sub(t[1], t[0], t[2])   // x3 = e * (d - x3) - 8c
	mul(t[0], &p[1], &p[2]) // y1 * z1
	r[1].set(t[1])          //
	double(&r[2], t[0])     // z3 = 2(y1 * z1)
	return r
}

// Neg negates a G1 point p and assigns the result to the point at first argument.
func (g *G1) Neg(r, p *PointG1) *PointG1 {
	r[0].set(&p[0])
	r[2].set(&p[2])
	neg(&r[1], &p[1])
	return r
}

// Sub subtracts two G1 points p1, p2 and assigns the result to point at first argument.
func (g *G1) Sub(c, a, b *PointG1) *PointG1 {
	d := &PointG1{}
	g.Neg(d, b)
	g.Add(c, a, d)
	return c
}

// MulScalar multiplies a point by given scalar value and assigns the result to point at first argument.
func (g *G1) MulScalar(r, p *PointG1, e *Fr) *PointG1 {
	return g.glvMulFr(r, p, e)
}

// MulScalar multiplies a point by given scalar value in big.Int and assigns the result to point at first argument.
func (g *G1) MulScalarBig(r, p *PointG1, e *big.Int) *PointG1 {
	return g.glvMulBig(r, p, e)
}

func (g *G1) mulScalar(c, p *PointG1, e *Fr) *PointG1 {
	q, n := &PointG1{}, &PointG1{}
	n.Set(p)
	for i := 0; i < frBitSize; i++ {
		if e.Bit(i) {
			g.Add(q, q, n)
		}
		g.Double(n, n)
	}
	return c.Set(q)
}

func (g *G1) mulScalarBig(c, p *PointG1, e *big.Int) *PointG1 {
	q, n := &PointG1{}, &PointG1{}
	n.Set(p)
	for i := 0; i < frBitSize; i++ {
		if e.Bit(i) == 1 {
			g.Add(q, q, n)
		}
		g.Double(n, n)
	}
	return c.Set(q)
}

func (g *G1) wnafMulFr(r, p *PointG1, e *Fr) *PointG1 {
	wnaf := e.toWNAF(wnafMulWindowG1)
	return g.wnafMul(r, p, wnaf)
}

func (g *G1) wnafMulBig(r, p *PointG1, e *big.Int) *PointG1 {
	wnaf := bigToWNAF(e, wnafMulWindowG1)
	return g.wnafMul(r, p, wnaf)
}

func (g *G1) wnafMul(c, p *PointG1, wnaf nafNumber) *PointG1 {

	l := (1 << (wnafMulWindowG1 - 1))

	twoP, acc := g.New(), new(PointG1).Set(p)
	g.Double(twoP, p)
	g.Affine(twoP)

	// table = {p, 3p, 5p, ..., -p, -3p, -5p}
	table := make([]*PointG1, l*2)
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

func (g *G1) glvMulFr(r, p *PointG1, e *Fr) *PointG1 {
	return g.glvMul(r, p, new(glvVectorFr).new(e))
}

func (g *G1) glvMulBig(r, p *PointG1, e *big.Int) *PointG1 {
	return g.glvMul(r, p, new(glvVectorBig).new(e))
}

func (g *G1) glvMul(r, p0 *PointG1, v glvVector) *PointG1 {

	w := glvMulWindowG1
	l := 1 << (w - 1)

	// prepare tables
	// tableK1 = {P, 3P, 5P, ...}
	// tableK2 = {λP, 3λP, 5λP, ...}
	tableK1, tableK2 := make([]*PointG1, l), make([]*PointG1, l)
	double := g.New()
	g.Double(double, p0)
	g.affine(double, double)
	tableK1[0] = new(PointG1)
	tableK1[0].Set(p0)
	for i := 1; i < l; i++ {
		tableK1[i] = new(PointG1)
		g.AddMixed(tableK1[i], tableK1[i-1], double)
	}
	g.AffineBatch(tableK1)
	for i := 0; i < l; i++ {
		tableK2[i] = new(PointG1)
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
	add := func(table []*PointG1, naf int) {
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
// Given pairs of G1 point and scalar values `(P_0, e_0), (P_1, e_1), ... (P_n, e_n)`,
// calculates `r = e_0 * P_0 + e_1 * P_1 + ... + e_n * P_n`.
// Length of points and scalars are expected to be equal, otherwise an error is returned.
// Result is assigned to point at first argument.
func (g *G1) MultiExpBig(r *PointG1, points []*PointG1, scalars []*big.Int) (*PointG1, error) {
	if len(points) != len(scalars) {
		return nil, errors.New("point and scalar vectors should be in same length")
	}

	c := 3
	if len(scalars) >= 32 {
		c = int(math.Ceil(math.Log(float64(len(scalars)))))
	}

	bucketSize := (1 << c) - 1
	windows := make([]PointG1, 255/c+1)
	bucket := make([]PointG1, bucketSize)

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

// MultiExp calculates multi exponentiation. Given pairs of G1 point and scalar values `(P_0, e_0), (P_1, e_1), ... (P_n, e_n)`,
// calculates `r = e_0 * P_0 + e_1 * P_1 + ... + e_n * P_n`. Length of points and scalars are expected to be equal,
// otherwise an error is returned. Result is assigned to point at first argument.
func (g *G1) MultiExp(r *PointG1, points []*PointG1, scalars []*Fr) (*PointG1, error) {
	if len(points) != len(scalars) {
		return nil, errors.New("point and scalar vectors should be in same length")
	}

	g.AffineBatch(points)

	c := 3
	if len(scalars) >= 32 {
		c = int(math.Ceil(math.Log(float64(len(scalars)))))
	}

	bucketSize := (1 << c) - 1
	windows := make([]*PointG1, 255/c+1)
	bucket := make([]PointG1, bucketSize)

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

func (g *G1) ClearCofactor(p *PointG1) *PointG1 {
	chain := func(p0 *PointG1, n int, p1 *PointG1) {
		for i := 0; i < n; i++ {
			g.Double(p0, p0)
		}
		g.Add(p0, p0, p1)
	}
	t := g.New().Set(p)
	chain(p, 1, t)
	chain(p, 2, t)
	chain(p, 3, t)
	chain(p, 9, t)
	chain(p, 32, t)
	chain(p, 16, t)
	return p
}

// MapToCurve given a byte slice returns a valid G1 point.
// This mapping function implements the Simplified Shallue-van de Woestijne-Ulas method.
// https://tools.ietf.org/html/draft-irtf-cfrg-hash-to-curve-06
// Input byte slice should be a valid field element, otherwise an error is returned.
func (g *G1) MapToCurve(in []byte) (*PointG1, error) {
	u, err := fromBytes(in)
	if err != nil {
		return nil, err
	}
	x, y := swuMapG1(u)
	isogenyMapG1(x, y)
	one := new(fe).one()
	p := &PointG1{*x, *y, *one}
	g.ClearCofactor(p)
	return g.Affine(p), nil
}

// EncodeToCurve given a message and domain seperator tag returns the hash result
// which is a valid curve point.
// Implementation follows BLS12381G1_XMD:SHA-256_SSWU_NU_ suite at
// https://tools.ietf.org/html/draft-irtf-cfrg-hash-to-curve-06
func (g *G1) EncodeToCurve(msg, domain []byte) (*PointG1, error) {
	hashRes, err := hashToFpXMDSHA256(msg, domain, 1)
	if err != nil {
		return nil, err
	}
	u := hashRes[0]
	x, y := swuMapG1(u)
	isogenyMapG1(x, y)
	one := new(fe).one()
	p := &PointG1{*x, *y, *one}
	g.ClearCofactor(p)
	return g.Affine(p), nil
}

// HashToCurve given a message and domain seperator tag returns the hash result
// which is a valid curve point.
// Implementation follows BLS12381G1_XMD:SHA-256_SSWU_RO_ suite at
// https://tools.ietf.org/html/draft-irtf-cfrg-hash-to-curve-06
func (g *G1) HashToCurve(msg, domain []byte) (*PointG1, error) {
	hashRes, err := hashToFpXMDSHA256(msg, domain, 2)
	if err != nil {
		return nil, err
	}
	u0, u1 := hashRes[0], hashRes[1]
	x0, y0 := swuMapG1(u0)
	x1, y1 := swuMapG1(u1)
	one := new(fe).one()
	p0, p1 := &PointG1{*x0, *y0, *one}, &PointG1{*x1, *y1, *one}
	g.Add(p0, p0, p1)
	g.Affine(p0)
	isogenyMapG1(&p0[0], &p0[1])
	g.ClearCofactor(p0)
	return g.Affine(p0), nil
}
