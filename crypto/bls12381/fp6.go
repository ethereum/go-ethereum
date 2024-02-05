package bls12381

import (
	"errors"
	"math/big"
)

type fp6Temp struct {
	t  [5]*fe2
	wt [6]*wfe2
}

type fp6 struct {
	fp2 *fp2
	fp6Temp
}

func newFp6Temp() fp6Temp {
	t := [5]*fe2{}
	for i := 0; i < len(t); i++ {
		t[i] = &fe2{}
	}
	wt := [6]*wfe2{}
	for i := 0; i < len(wt); i++ {
		wt[i] = &wfe2{}
	}
	return fp6Temp{t, wt}
}

func newFp6(f *fp2) *fp6 {
	t := newFp6Temp()
	if f == nil {
		return &fp6{newFp2(), t}
	}
	return &fp6{f, t}
}

func (e *fp6) fromBytes(b []byte) (*fe6, error) {
	if len(b) != 288 {
		return nil, errors.New("input string length must be equal to 288 bytes")
	}
	fp2 := e.fp2
	u2, err := fp2.fromBytes(b[:2*fpByteSize])
	if err != nil {
		return nil, err
	}
	u1, err := fp2.fromBytes(b[2*fpByteSize : 4*fpByteSize])
	if err != nil {
		return nil, err
	}
	u0, err := fp2.fromBytes(b[4*fpByteSize:])
	if err != nil {
		return nil, err
	}
	return &fe6{*u0, *u1, *u2}, nil
}

func (e *fp6) toBytes(a *fe6) []byte {
	fp2 := e.fp2
	out := make([]byte, 6*fpByteSize)
	copy(out[:2*fpByteSize], fp2.toBytes(&a[2]))
	copy(out[2*fpByteSize:4*fpByteSize], fp2.toBytes(&a[1]))
	copy(out[4*fpByteSize:], fp2.toBytes(&a[0]))
	return out
}

func (e *fp6) new() *fe6 {
	return new(fe6)
}

func (e *fp6) zero() *fe6 {
	return new(fe6)
}

func (e *fp6) one() *fe6 {
	return new(fe6).one()
}

func fp6Ladd(c, a, b *fe6) {
	fp2Ladd(&c[0], &a[0], &b[0])
	fp2Ladd(&c[1], &a[1], &b[1])
	fp2Ladd(&c[2], &a[2], &b[2])
}

func wfp6SubAssign(a, b *wfe6) {
	wfp2SubAssign(&a[0], &b[0])
	wfp2SubAssign(&a[1], &b[1])
	wfp2SubAssign(&a[2], &b[2])
}

func wfp6AddAssign(a, b *wfe6) {
	wfp2AddAssign(&a[0], &b[0])
	wfp2AddAssign(&a[1], &b[1])
	wfp2AddAssign(&a[2], &b[2])
}

func fp6Add(c, a, b *fe6) {
	fp2Add(&c[0], &a[0], &b[0])
	fp2Add(&c[1], &a[1], &b[1])
	fp2Add(&c[2], &a[2], &b[2])
}

func fp6AddAssign(a, b *fe6) {
	fp2AddAssign(&a[0], &b[0])
	fp2AddAssign(&a[1], &b[1])
	fp2AddAssign(&a[2], &b[2])
}

func fp6Double(c, a *fe6) {
	fp2Double(&c[0], &a[0])
	fp2Double(&c[1], &a[1])
	fp2Double(&c[2], &a[2])
}

func fp6DoubleAssign(a *fe6) {
	fp2DoubleAssign(&a[0])
	fp2DoubleAssign(&a[1])
	fp2DoubleAssign(&a[2])
}

func fp6Sub(c, a, b *fe6) {
	fp2Sub(&c[0], &a[0], &b[0])
	fp2Sub(&c[1], &a[1], &b[1])
	fp2Sub(&c[2], &a[2], &b[2])
}

func fp6SubAssign(a, b *fe6) {
	fp2SubAssign(&a[0], &b[0])
	fp2SubAssign(&a[1], &b[1])
	fp2SubAssign(&a[2], &b[2])
}

func fp6Neg(c, a *fe6) {
	fp2Neg(&c[0], &a[0])
	fp2Neg(&c[1], &a[1])
	fp2Neg(&c[2], &a[2])
}

func (e *fp6) wmul01(c *wfe6, a *fe6, b0, b1 *fe2) {
	wt, t := e.wt, e.t
	wfp2Mul(wt[0], &a[0], b0)   // v0 = b0a0
	wfp2Mul(wt[1], &a[1], b1)   // v1 = a1b1
	fp2Ladd(t[2], &a[1], &a[2]) // a1 + a2
	wfp2Mul(wt[2], t[2], b1)    // b1(a1 + a2)
	wfp2SubAssign(wt[2], wt[1]) // b1(a1 + a2) - v1
	wfp2MulByNonResidueAssign(wt[2])
	fp2Ladd(t[3], &a[0], &a[2]) // a0 + a2
	wfp2Mul(wt[3], t[3], b0)    // b0(a0 + a2)
	wfp2SubAssign(wt[3], wt[0])
	wfp2Add(&c[2], wt[3], wt[1])
	fp2Ladd(t[0], b0, b1)       // (b0 + b1)
	fp2Ladd(t[1], &a[0], &a[1]) // (a0 + a1)
	wfp2Mul(wt[4], t[0], t[1])  // (a0 + a1)(b0 + b1)
	wfp2SubAssign(wt[4], wt[0])
	wfp2Sub(&c[1], wt[4], wt[1])
	wfp2Add(&c[0], wt[2], wt[0])
}

func (e *fp6) wmul1(c *wfe6, a *fe6, b1 *fe2) {
	wt := e.wt
	wfp2Mul(wt[0], &a[2], b1)
	wfp2Mul(&c[2], &a[1], b1)
	wfp2Mul(&c[1], &a[0], b1)
	wfp2MulByNonResidue(&c[0], wt[0])
}

func (e *fp6) wmul(c *wfe6, a, b *fe6) {

	wt, t := e.wt, e.t

	// Faster Explicit Formulas for Computing Pairings over Ordinary Curves
	// AKLGL
	// https://eprint.iacr.org/2010/526.pdf
	// Algorithm 3

	// 1. T0 = a0b0,T1 = a1b1, T2 = a2b2
	wfp2Mul(wt[0], &a[0], &b[0])
	wfp2Mul(wt[1], &a[1], &b[1])
	wfp2Mul(wt[2], &a[2], &b[2])
	// 2. t0 = a1 + a2, t1 = b1 + b2
	fp2Ladd(t[0], &a[1], &a[2])
	fp2Ladd(t[1], &b[1], &b[2])
	// 3. T3 = t0 * t1
	wfp2Mul(wt[3], t[0], t[1])
	// 4. T4 = T1 + T2
	wfp2Add(wt[4], wt[1], wt[2])

	// 5,6. T3 = T3 - T4
	wfp2SubMixedAssign(wt[3], wt[4])

	// 7. T4 = β * T3
	wfp2MulByNonResidue(wt[4], wt[3])

	// 8. T5 = T4 + T0
	wfp2Add(wt[5], wt[4], wt[0])

	// 9. t0 = a0 + a1, t1 = b0 + b1
	fp2Ladd(t[0], &a[0], &a[1])
	fp2Ladd(t[1], &b[0], &b[1])

	// 10. T3 = t0 * t1
	wfp2Mul(wt[3], t[0], t[1])

	// 11. T4 = T0 + T1
	wfp2Add(wt[4], wt[0], wt[1])

	// 12,13. T3 = T3 - T4
	wfp2SubMixedAssign(wt[3], wt[4])

	// 14,15. T4 = β * T2
	wfp2MulByNonResidue(wt[4], wt[2])

	// 17. t0 = a0 + a2, t1 = b0 + b2
	fp2Ladd(t[0], &a[0], &a[2])
	fp2Ladd(t[1], &b[0], &b[2])

	// 16. T6 = T3 + T4
	wfp2Add(&c[1], wt[3], wt[4])

	// 18. T3 = t0 * t1
	wfp2Mul(wt[3], t[0], t[1])

	// 19. T4 = T0 + T2
	wfp2Add(wt[4], wt[0], wt[2])

	// 20,21. T3 = T3 - T4
	wfp2SubMixedAssign(wt[3], wt[4])

	// 22,23. T7 = T3 + T1
	wfp2AddMixed(&c[2], wt[3], wt[1])

	// c = T5, T6, T7
	c[0].set(wt[5])
}

func (e *fp6) mul(c *fe6, a, b *fe6) {
	wt, t := e.wt, e.t

	// 1. T0 = a0b0,T1 = a1b1, T2 = a2b2
	wfp2Mul(wt[0], &a[0], &b[0])
	wfp2Mul(wt[1], &a[1], &b[1])
	wfp2Mul(wt[2], &a[2], &b[2])
	// 2. t0 = a1 + a2, t1 = b1 + b2
	fp2Ladd(t[0], &a[1], &a[2])
	fp2Ladd(t[1], &b[1], &b[2])
	// 3. T3 = t0 * t1
	wfp2Mul(wt[3], t[0], t[1])
	// 4. T4 = T1 + T2
	wfp2Add(wt[4], wt[1], wt[2])

	// 5,6. T3 = T3 - T4
	wfp2SubMixedAssign(wt[3], wt[4])

	// 7. T4 = β * T3
	wfp2MulByNonResidue(wt[4], wt[3])

	// 8. T5 = T4 + T0
	wfp2Add(wt[5], wt[4], wt[0])

	// 9. t0 = a0 + a1, t1 = b0 + b1
	fp2Ladd(t[0], &a[0], &a[1])
	fp2Ladd(t[1], &b[0], &b[1])

	// 10. T3 = t0 * t1
	wfp2Mul(wt[3], t[0], t[1])

	// 11. T4 = T0 + T1
	wfp2Add(wt[4], wt[0], wt[1])

	// 12,13. T3 = T3 - T4
	wfp2SubMixed(wt[3], wt[3], wt[4])

	// 14,15. T4 = β * T2
	wfp2MulByNonResidue(wt[4], wt[2])

	// 17. t0 = a0 + a2, t1 = b0 + b2
	fp2Ladd(t[0], &a[0], &a[2])
	fp2Ladd(t[1], &b[0], &b[2])

	// 16. T6 = T3 + T4
	wfp2Add(wt[3], wt[3], wt[4])
	c[1].fromWide(wt[3])

	// 18. T3 = t0 * t1
	wfp2Mul(wt[3], t[0], t[1])

	// 19. T4 = T0 + T2
	wfp2Add(wt[4], wt[0], wt[2])

	// 20,21. T3 = T3 - T4
	wfp2SubMixed(wt[3], wt[3], wt[4])

	// 22,23. T7 = T3 + T1
	wfp2AddMixed(wt[3], wt[3], wt[1])
	c[2].fromWide(wt[3])

	// c = T5, T6, T7
	c[0].fromWide(wt[5])
}

func (e *fp6) mulAssign(a, b *fe6) {
	wt, t := e.wt, e.t

	// Faster Explicit Formulas for Computing Pairings over Ordinary Curves
	// AKLGL
	// https://eprint.iacr.org/2010/526.pdf
	// Algorithm 3

	// 1. T0 = a0b0,T1 = a1b1, T2 = a2b2
	wfp2Mul(wt[0], &a[0], &b[0])
	wfp2Mul(wt[1], &a[1], &b[1])
	wfp2Mul(wt[2], &a[2], &b[2])
	// 2. t0 = a1 + a2, t1 = b1 + b2
	fp2Ladd(t[0], &a[1], &a[2])
	fp2Ladd(t[1], &b[1], &b[2])
	// 3. T3 = t0 * t1
	wfp2Mul(wt[3], t[0], t[1])
	// 4. T4 = T1 + T2
	wfp2Add(wt[4], wt[1], wt[2])

	// 5,6. T3 = T3 - T4
	wfp2SubMixed(wt[3], wt[3], wt[4])

	// 7. T4 = β * T3
	wfp2MulByNonResidue(wt[4], wt[3])

	// 8. T5 = T4 + T0
	wfp2Add(wt[5], wt[4], wt[0])

	// 9. t0 = a0 + a1, t1 = b0 + b1
	fp2Ladd(t[0], &a[0], &a[1])
	fp2Ladd(t[1], &b[0], &b[1])

	// 10. T3 = t0 * t1
	wfp2Mul(wt[3], t[0], t[1])

	// 11. T4 = T0 + T1
	wfp2Add(wt[4], wt[0], wt[1])

	// 12,13. T3 = T3 - T4
	wfp2SubMixed(wt[3], wt[3], wt[4])

	// 14,15. T4 = β * T2
	wfp2MulByNonResidue(wt[4], wt[2])

	// 17. t0 = a0 + a2, t1 = b0 + b2
	fp2Ladd(t[0], &a[0], &a[2])
	fp2Ladd(t[1], &b[0], &b[2])

	// 16. T6 = T3 + T4
	wfp2Add(wt[3], wt[3], wt[4])
	a[1].fromWide(wt[3])

	// 18. T3 = t0 * t1
	wfp2Mul(wt[3], t[0], t[1])

	// 19. T4 = T0 + T2
	wfp2Add(wt[4], wt[0], wt[2])

	// 20,21. T3 = T3 - T4
	wfp2SubMixed(wt[3], wt[3], wt[4])

	// 22,23. T7 = T3 + T1
	wfp2AddMixed(wt[3], wt[3], wt[1])
	a[2].fromWide(wt[3])

	// a = T5, T6, T7
	a[0].fromWide(wt[5])
}

func (e *fp6) square(c, a *fe6) {
	wt, t := e.wt, e.t
	wfp2Square(wt[0], &a[0])
	wfp2Mul(wt[1], &a[0], &a[1])
	wfp2DoubleAssign(wt[1])
	fp2Sub(t[2], &a[0], &a[1])
	fp2AddAssign(t[2], &a[2])
	wfp2Square(wt[2], t[2])
	wfp2Mul(wt[3], &a[1], &a[2])
	wfp2DoubleAssign(wt[3])
	wfp2Square(wt[4], &a[2])
	wfp2MulByNonResidue(wt[5], wt[3])
	wfp2AddAssign(wt[5], wt[0])
	c[0].fromWide(wt[5])
	wfp2MulByNonResidue(wt[5], wt[4])
	wfp2AddAssign(wt[5], wt[1])
	c[1].fromWide(wt[5])
	wfp2AddAssign(wt[1], wt[2])
	wfp2AddAssign(wt[1], wt[3])
	wfp2AddAssign(wt[0], wt[4])
	wfp2SubAssign(wt[1], wt[0])
	c[2].fromWide(wt[1])

}

func (e *fp6) wsquare(c *wfe6, a *fe6) {
	wt, t := e.wt, e.t
	wfp2Square(wt[0], &a[0])
	wfp2Mul(wt[1], &a[0], &a[1])
	wfp2DoubleAssign(wt[1])
	fp2Sub(t[2], &a[0], &a[1])
	fp2AddAssign(t[2], &a[2])
	wfp2Square(wt[2], t[2])
	wfp2Mul(wt[3], &a[1], &a[2])
	wfp2DoubleAssign(wt[3])
	wfp2Square(wt[4], &a[2])
	wfp2MulByNonResidue(wt[5], wt[3])
	wfp2Add(&c[0], wt[5], wt[0])
	wfp2MulByNonResidue(wt[5], wt[4])
	wfp2Add(&c[1], wt[1], wt[5])
	wfp2AddAssign(wt[1], wt[2])
	wfp2AddAssign(wt[1], wt[3])
	wfp2AddAssign(wt[0], wt[4])
	wfp2Sub(&c[2], wt[1], wt[0])
}

func (e *fp6) mulByNonResidue(c, a *fe6) {
	t := e.t
	t[0].set(&a[0])
	mulByNonResidue(&c[0], &a[2])
	c[2].set(&a[1])
	c[1].set(t[0])
}

func (e *fp6) wmulByNonResidue(c, a *wfe6) {
	t := e.wt
	t[0].set(&a[0])
	wfp2MulByNonResidue(&c[0], &a[2])
	c[2].set(&a[1])
	c[1].set(t[0])
}

func (e *fp6) wmulByNonResidueAssign(a *wfe6) {
	t := e.wt
	t[0].set(&a[0])
	wfp2MulByNonResidue(&a[0], &a[2])
	a[2].set(&a[1])
	a[1].set(t[0])
}

func (e *fp6) mulByBaseField(c, a *fe6, b *fe2) {
	fp2 := e.fp2
	fp2.mul(&c[0], &a[0], b)
	fp2.mul(&c[1], &a[1], b)
	fp2.mul(&c[2], &a[2], b)
}

func (e *fp6) exp(c, a *fe6, s *big.Int) {
	z := e.one()
	for i := s.BitLen() - 1; i >= 0; i-- {
		e.square(z, z)
		if s.Bit(i) == 1 {
			e.mul(z, z, a)
		}
	}
	c.set(z)
}

func (e *fp6) inverse(c, a *fe6) {
	fp2, t := e.fp2, e.t
	fp2.square(t[0], &a[0])
	fp2.mul(t[1], &a[1], &a[2])
	mulByNonResidueAssign(t[1])
	fp2SubAssign(t[0], t[1])    // A = v0 - βv5
	fp2.square(t[1], &a[1])     // v1 = a1^2
	fp2.mul(t[2], &a[0], &a[2]) // v4 = a0a2
	fp2SubAssign(t[1], t[2])    // C = v1 - v4
	fp2.square(t[2], &a[2])     // v2 = a2^2
	mulByNonResidueAssign(t[2]) // βv2
	fp2.mul(t[3], &a[0], &a[1]) // v3 = a0a1
	fp2SubAssign(t[2], t[3])    // B = βv2 - v3
	fp2.mul(t[3], &a[2], t[2])  // B * a2
	fp2.mul(t[4], &a[1], t[1])  // C * a1
	fp2AddAssign(t[3], t[4])    // Ca1 + Ba2
	mulByNonResidueAssign(t[3]) // β(Ca1 + Ba2)
	fp2.mul(t[4], &a[0], t[0])  // Aa0
	fp2AddAssign(t[3], t[4])    // v6 = Aa0 + β(Ca1 + Ba2)
	fp2.inverse(t[3], t[3])     // F = v6^-1
	fp2.mul(&c[0], t[0], t[3])  // c0 = AF
	fp2.mul(&c[1], t[2], t[3])  // c1 = BF
	fp2.mul(&c[2], t[1], t[3])  // c2 = CF
}

func (e *fp6) frobeniusMap(a *fe6, power int) {
	fp2 := e.fp2
	fp2.frobeniusMap(&a[0], power)
	fp2.frobeniusMap(&a[1], power)
	fp2.frobeniusMap(&a[2], power)
	fp2.mulAssign(&a[1], &frobeniusCoeffs61[power%6])
	fp2.mulAssign(&a[2], &frobeniusCoeffs62[power%6])
}

func (e *fp6) frobeniusMap1(a *fe6) {
	fp2 := e.fp2
	fp2.frobeniusMap1(&a[0])
	fp2.frobeniusMap1(&a[1])
	fp2.frobeniusMap1(&a[2])
	fp2.mulAssign(&a[1], &frobeniusCoeffs61[1])
	fp2.mulAssign(&a[2], &frobeniusCoeffs62[1])
}

func (e *fp6) frobeniusMap2(a *fe6) {
	e.fp2.mulAssign(&a[1], &frobeniusCoeffs61[2])
	e.fp2.mulAssign(&a[2], &frobeniusCoeffs62[2])
}

func (e *fp6) frobeniusMap3(a *fe6) {
	t := e.t
	e.fp2.frobeniusMap1(&a[0])
	e.fp2.frobeniusMap1(&a[1])
	e.fp2.frobeniusMap1(&a[2])
	neg(&t[0][0], &a[1][1])
	a[1][1].set(&a[1][0])
	a[1][0].set(&t[0][0])
	fp2Neg(&a[2], &a[2])
}
