package bls12381

import (
	"errors"
	"math/big"
)

type fp12 struct {
	fp12temp
	fp6 *fp6
}

type fp12temp struct {
	t2  [7]*fe2
	t6  [4]*fe6
	wt2 [3]*wfe2
	wt6 [3]*wfe6
}

func newFp12Temp() fp12temp {
	t2 := [7]*fe2{}
	t6 := [4]*fe6{}
	for i := 0; i < len(t2); i++ {
		t2[i] = &fe2{}
	}
	for i := 0; i < len(t6); i++ {
		t6[i] = &fe6{}
	}
	wt2 := [3]*wfe2{}
	for i := 0; i < len(wt2); i++ {
		wt2[i] = &wfe2{}
	}
	wt6 := [3]*wfe6{}
	for i := 0; i < len(wt6); i++ {
		wt6[i] = &wfe6{}
	}
	return fp12temp{t2, t6, wt2, wt6}
}

func newFp12(fp6 *fp6) *fp12 {
	t := newFp12Temp()
	if fp6 == nil {
		return &fp12{t, newFp6(nil)}
	}
	return &fp12{t, fp6}
}

func (e *fp12) fp2() *fp2 {
	return e.fp6.fp2
}

func (e *fp12) fromBytes(in []byte) (*fe12, error) {
	if len(in) != 576 {
		return nil, errors.New("input string length must be equal to 576 bytes")
	}
	fp6 := e.fp6
	c1, err := fp6.fromBytes(in[:6*fpByteSize])
	if err != nil {
		return nil, err
	}
	c0, err := fp6.fromBytes(in[6*fpByteSize:])
	if err != nil {
		return nil, err
	}
	return &fe12{*c0, *c1}, nil
}

func (e *fp12) toBytes(a *fe12) []byte {
	fp6 := e.fp6
	out := make([]byte, 12*fpByteSize)
	copy(out[:6*fpByteSize], fp6.toBytes(&a[1]))
	copy(out[6*fpByteSize:], fp6.toBytes(&a[0]))
	return out
}

func (e *fp12) new() *fe12 {
	return new(fe12)
}

func (e *fp12) zero() *fe12 {
	return new(fe12)
}

func (e *fp12) one() *fe12 {
	return new(fe12).one()
}

func fp12Add(c, a, b *fe12) {
	fp6Add(&c[0], &a[0], &b[0])
	fp6Add(&c[1], &a[1], &b[1])
}

func fp12Double(c, a *fe12) {
	fp6Double(&c[0], &a[0])
	fp6Double(&c[1], &a[1])
}

func fp12Sub(c, a, b *fe12) {
	fp6Sub(&c[0], &a[0], &b[0])
	fp6Sub(&c[1], &a[1], &b[1])

}

func fp12Neg(c, a *fe12) {
	fp6Neg(&c[0], &a[0])
	fp6Neg(&c[1], &a[1])
}

func fp12Conjugate(c, a *fe12) {
	c[0].set(&a[0])
	fp6Neg(&c[1], &a[1])
}

func (e *fp12) mul(c, a, b *fe12) {
	wt, t := e.wt6, e.t6
	e.fp6.wmul(wt[1], &a[0], &b[0])
	e.fp6.wmul(wt[2], &a[1], &b[1])
	fp6Add(t[0], &a[0], &a[1])
	fp6Add(t[3], &b[0], &b[1])
	e.fp6.wmul(wt[0], t[0], t[3])
	wfp6SubAssign(wt[0], wt[1])
	wfp6SubAssign(wt[0], wt[2])
	c[1].fromWide(wt[0])
	e.fp6.wmulByNonResidueAssign(wt[2])
	wfp6AddAssign(wt[1], wt[2])
	c[0].fromWide(wt[1])

}

func (e *fp12) mulAssign(a, b *fe12) {
	wt, t := e.wt6, e.t6
	e.fp6.wmul(wt[1], &a[0], &b[0])
	e.fp6.wmul(wt[2], &a[1], &b[1])
	fp6Add(t[0], &a[0], &a[1])
	fp6Add(t[3], &b[0], &b[1])
	e.fp6.wmul(wt[0], t[0], t[3])
	wfp6SubAssign(wt[0], wt[1])
	wfp6SubAssign(wt[0], wt[2])
	a[1].fromWide(wt[0])
	e.fp6.wmulByNonResidueAssign(wt[2])
	wfp6AddAssign(wt[1], wt[2])
	a[0].fromWide(wt[1])
}

func (e *fp12) mul014(a *fe12, b0, b1, b4 *fe2) {
	wt, t := e.wt6, e.t6
	e.fp6.wmul01(wt[0], &a[0], b0, b1)
	e.fp6.wmul1(wt[1], &a[1], b4)
	fp2LaddAssign(b1, b4)
	fp6Ladd(t[2], &a[1], &a[0])
	e.fp6.wmul01(wt[2], t[2], b0, b1)
	wfp6SubAssign(wt[2], wt[0])
	wfp6SubAssign(wt[2], wt[1])
	a[1].fromWide(wt[2])
	e.fp6.wmulByNonResidueAssign(wt[1])
	wfp6AddAssign(wt[0], wt[1])
	a[0].fromWide(wt[0])
}

func (e *fp12) square(c, a *fe12) {
	t := e.t6
	// Multiplication and Squaring on Pairing-Friendly Fields
	// Complex squaring algorithm
	// https://eprint.iacr.org/2006/471

	fp6Add(t[0], &a[0], &a[1])
	e.fp6.mul(t[2], &a[0], &a[1])
	e.fp6.mulByNonResidue(t[1], &a[1])
	fp6AddAssign(t[1], &a[0])
	e.fp6.mulByNonResidue(t[3], t[2])
	e.fp6.mul(t[0], t[0], t[1])
	fp6SubAssign(t[0], t[2])
	fp6Sub(&c[0], t[0], t[3])
	fp6Double(&c[1], t[2])
}

func (e *fp12) squareAssign(a *fe12) {
	t := e.t6
	// Multiplication and Squaring on Pairing-Friendly Fields
	// Complex squaring algorithm
	// https://eprint.iacr.org/2006/471

	fp6Add(t[0], &a[0], &a[1])
	e.fp6.mul(t[2], &a[0], &a[1])
	e.fp6.mulByNonResidue(t[1], &a[1])
	fp6AddAssign(t[1], &a[0])
	e.fp6.mulByNonResidue(t[3], t[2])
	e.fp6.mul(t[0], t[0], t[1])
	fp6SubAssign(t[0], t[2])
	fp6Sub(&a[0], t[0], t[3])
	fp6Double(&a[1], t[2])
}

func (e *fp12) inverse(c, a *fe12) {
	// Guide to Pairing Based Cryptography
	// Algorithm 5.16

	t := e.t6
	e.fp6.square(t[0], &a[0])         // a0^2
	e.fp6.square(t[1], &a[1])         // a1^2
	e.fp6.mulByNonResidue(t[1], t[1]) // βa1^2
	fp6SubAssign(t[0], t[1])          // v = (a0^2 - a1^2)
	e.fp6.inverse(t[1], t[0])         // v = v^-1
	e.fp6.mul(&c[0], &a[0], t[1])     // c0 = a0v
	e.fp6.mulAssign(t[1], &a[1])      //
	fp6Neg(&c[1], t[1])               // c1 = -a1v
}

func (e *fp12) exp(c, a *fe12, s *big.Int) {
	z := e.one()
	for i := s.BitLen() - 1; i >= 0; i-- {
		e.square(z, z)
		if s.Bit(i) == 1 {
			e.mul(z, z, a)
		}
	}
	c.set(z)
}

func (e *fp12) cyclotomicExp(c, a *fe12, s *big.Int) {
	z := e.one()
	for i := s.BitLen() - 1; i >= 0; i-- {
		e.cyclotomicSquare(z)
		if s.Bit(i) == 1 {
			e.mul(z, z, a)
		}
	}
	c.set(z)
}

func (e *fp12) cyclotomicSquare(a *fe12) {
	t := e.t2
	// Guide to Pairing Based Cryptography
	// 5.5.4 Airthmetic in Cyclotomic Groups

	e.fp4Square(t[3], t[4], &a[0][0], &a[1][1])
	fp2Sub(t[2], t[3], &a[0][0])
	fp2DoubleAssign(t[2])
	fp2Add(&a[0][0], t[2], t[3])
	fp2Add(t[2], t[4], &a[1][1])
	fp2DoubleAssign(t[2])
	fp2Add(&a[1][1], t[2], t[4])
	e.fp4Square(t[3], t[4], &a[1][0], &a[0][2])
	e.fp4Square(t[5], t[6], &a[0][1], &a[1][2])
	fp2Sub(t[2], t[3], &a[0][1])
	fp2DoubleAssign(t[2])
	fp2Add(&a[0][1], t[2], t[3])
	fp2Add(t[2], t[4], &a[1][2])
	fp2DoubleAssign(t[2])
	fp2Add(&a[1][2], t[2], t[4])
	mulByNonResidue(t[3], t[6])
	fp2Add(t[2], t[3], &a[1][0])
	fp2DoubleAssign(t[2])
	fp2Add(&a[1][0], t[2], t[3])
	fp2Sub(t[2], t[5], &a[0][2])
	fp2DoubleAssign(t[2])
	fp2Add(&a[0][2], t[2], t[5])
}

func (e *fp12) fp4Square(c0, c1, a0, a1 *fe2) {
	wt, t := e.wt2, e.t2
	// Multiplication and Squaring on Pairing-Friendly Fields
	// Karatsuba squaring algorithm
	// https://eprint.iacr.org/2006/471

	wfp2Square(wt[0], a0)
	wfp2Square(wt[1], a1)
	wfp2MulByNonResidue(wt[2], wt[1])
	wfp2AddAssign(wt[2], wt[0])
	c0.fromWide(wt[2])
	fp2Add(t[0], a0, a1)
	wfp2Square(wt[2], t[0])
	wfp2SubAssign(wt[2], wt[0])
	wfp2SubAssign(wt[2], wt[1])
	c1.fromWide(wt[2])
}

func (e *fp12) frobeniusMap1(a *fe12) {
	fp6, fp2 := e.fp6, e.fp6.fp2
	fp6.frobeniusMap1(&a[0])
	fp6.frobeniusMap1(&a[1])
	fp2.mulAssign(&a[1][0], &frobeniusCoeffs12[1])
	fp2.mulAssign(&a[1][1], &frobeniusCoeffs12[1])
	fp2.mulAssign(&a[1][2], &frobeniusCoeffs12[1])
}

func (e *fp12) frobeniusMap2(a *fe12) {
	fp6, fp2 := e.fp6, e.fp6.fp2
	fp6.frobeniusMap2(&a[0])
	fp6.frobeniusMap2(&a[1])
	fp2.mulAssign(&a[1][0], &frobeniusCoeffs12[2])
	fp2.mulAssign(&a[1][1], &frobeniusCoeffs12[2])
	fp2.mulAssign(&a[1][2], &frobeniusCoeffs12[2])
}

func (e *fp12) frobeniusMap3(a *fe12) {
	fp6, fp2 := e.fp6, e.fp6.fp2
	fp6.frobeniusMap3(&a[0])
	fp6.frobeniusMap3(&a[1])
	fp2.mulAssign(&a[1][0], &frobeniusCoeffs12[3])
	fp2.mulAssign(&a[1][1], &frobeniusCoeffs12[3])
	fp2.mulAssign(&a[1][2], &frobeniusCoeffs12[3])
}
