package bls12381

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
)

// fe is base field element representation
type fe /***			***/ [fpNumberOfLimbs]uint64

// fe2 is element representation of 'fp2' which is quadratic extention of base field 'fp'
// Representation follows c[0] + c[1] * u encoding order.
type fe2 /**			***/ [2]fe

// fe6 is element representation of 'fp6' field which is cubic extention of 'fp2'
// Representation follows c[0] + c[1] * v + c[2] * v^2 encoding order.
type fe6 /**			***/ [3]fe2

// fe12 is element representation of 'fp12' field which is quadratic extention of 'fp6'
// Representation follows c[0] + c[1] * w encoding order.
type fe12 /**			***/ [2]fe6

type wfe /***			***/ [fpNumberOfLimbs * 2]uint64
type wfe2 /**			***/ [2]wfe
type wfe6 /**			***/ [3]wfe2

func (fe *fe) setBytes(in []byte) *fe {
	l := len(in)
	if l >= fpByteSize {
		l = fpByteSize
	}
	padded := make([]byte, fpByteSize)
	copy(padded[fpByteSize-l:], in[:])
	var a int
	for i := 0; i < fpNumberOfLimbs; i++ {
		a = fpByteSize - i*8
		fe[i] = uint64(padded[a-1]) | uint64(padded[a-2])<<8 |
			uint64(padded[a-3])<<16 | uint64(padded[a-4])<<24 |
			uint64(padded[a-5])<<32 | uint64(padded[a-6])<<40 |
			uint64(padded[a-7])<<48 | uint64(padded[a-8])<<56
	}
	return fe
}

func (fe *fe) setBig(a *big.Int) *fe {
	return fe.setBytes(a.Bytes())
}

func (fe *fe) setString(s string) (*fe, error) {
	if s[:2] == "0x" {
		s = s[2:]
	}
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return fe.setBytes(bytes), nil
}

func (fe *fe) set(fe2 *fe) *fe {
	fe[0] = fe2[0]
	fe[1] = fe2[1]
	fe[2] = fe2[2]
	fe[3] = fe2[3]
	fe[4] = fe2[4]
	fe[5] = fe2[5]
	return fe
}

func (fe *fe) bytes() []byte {
	out := make([]byte, fpByteSize)
	var a int
	for i := 0; i < fpNumberOfLimbs; i++ {
		a = fpByteSize - i*8
		out[a-1] = byte(fe[i])
		out[a-2] = byte(fe[i] >> 8)
		out[a-3] = byte(fe[i] >> 16)
		out[a-4] = byte(fe[i] >> 24)
		out[a-5] = byte(fe[i] >> 32)
		out[a-6] = byte(fe[i] >> 40)
		out[a-7] = byte(fe[i] >> 48)
		out[a-8] = byte(fe[i] >> 56)
	}
	return out
}

func (fe *fe) big() *big.Int {
	return new(big.Int).SetBytes(fe.bytes())
}

func (fe *fe) string() (s string) {
	for i := fpNumberOfLimbs - 1; i >= 0; i-- {
		s = fmt.Sprintf("%s%16.16x", s, fe[i])
	}
	return "0x" + s
}

func (fe *fe) zero() *fe {
	fe[0] = 0
	fe[1] = 0
	fe[2] = 0
	fe[3] = 0
	fe[4] = 0
	fe[5] = 0
	return fe
}

func (fe *fe) one() *fe {
	return fe.set(r1)
}

func (fe *fe) rand(r io.Reader) (*fe, error) {
	bi, err := rand.Int(r, modulus.big())
	if err != nil {
		return nil, err
	}
	return fe.setBig(bi), nil
}

func (fe *fe) isValid() bool {
	return fe.cmp(&modulus) == -1
}

func (fe *fe) isOdd() bool {
	var mask uint64 = 1
	return fe[0]&mask != 0
}

func (fe *fe) isEven() bool {
	var mask uint64 = 1
	return fe[0]&mask == 0
}

func (fe *fe) isZero() bool {
	return (fe[5] | fe[4] | fe[3] | fe[2] | fe[1] | fe[0]) == 0
}

func (fe *fe) isOne() bool {
	return fe.equal(r1)
}

func (fe *fe) cmp(fe2 *fe) int {
	for i := fpNumberOfLimbs - 1; i >= 0; i-- {
		if fe[i] > fe2[i] {
			return 1
		} else if fe[i] < fe2[i] {
			return -1
		}
	}
	return 0
}

func (fe *fe) equal(fe2 *fe) bool {
	return fe2[0] == fe[0] && fe2[1] == fe[1] && fe2[2] == fe[2] && fe2[3] == fe[3] && fe2[4] == fe[4] && fe2[5] == fe[5]
}

func (e *fe) signBE() bool {
	negZ, z := new(fe), new(fe)
	fromMont(z, e)
	neg(negZ, z)
	return negZ.cmp(z) > -1
}

func (e *fe) sign() bool {
	r := new(fe)
	fromMont(r, e)
	return r[0]&1 == 0
}

func (e *fe) div2(u uint64) {
	e[0] = e[0]>>1 | e[1]<<63
	e[1] = e[1]>>1 | e[2]<<63
	e[2] = e[2]>>1 | e[3]<<63
	e[3] = e[3]>>1 | e[4]<<63
	e[4] = e[4]>>1 | e[5]<<63
	e[5] = e[5]>>1 | u<<63
}

func (e *fe) mul2() uint64 {
	u := e[5] >> 63
	e[5] = e[5]<<1 | e[4]>>63
	e[4] = e[4]<<1 | e[3]>>63
	e[3] = e[3]<<1 | e[2]>>63
	e[2] = e[2]<<1 | e[1]>>63
	e[1] = e[1]<<1 | e[0]>>63
	e[0] = e[0] << 1
	return u
}

func (e *fe2) zero() *fe2 {
	e[0].zero()
	e[1].zero()
	return e
}

func (e *fe2) one() *fe2 {
	e[0].one()
	e[1].zero()
	return e
}

func (e *fe2) set(e2 *fe2) *fe2 {
	e[0].set(&e2[0])
	e[1].set(&e2[1])
	return e
}

func (e *fe2) fromMont(a *fe2) {
	fromMont(&e[0], &a[0])
	fromMont(&e[1], &a[1])
}

func (e *fe2) fromWide(w *wfe2) {
	fromWide(&e[0], &w[0])
	fromWide(&e[1], &w[1])
}

func (e *fe2) rand(r io.Reader) (*fe2, error) {
	a0, err := new(fe).rand(r)
	if err != nil {
		return nil, err
	}
	e[0].set(a0)
	a1, err := new(fe).rand(r)
	if err != nil {
		return nil, err
	}
	e[1].set(a1)
	return e, nil
}

func (e *fe2) isOne() bool {
	return e[0].isOne() && e[1].isZero()
}

func (e *fe2) isZero() bool {
	return e[0].isZero() && e[1].isZero()
}

func (e *fe2) equal(e2 *fe2) bool {
	return e[0].equal(&e2[0]) && e[1].equal(&e2[1])
}

func (e *fe2) signBE() bool {
	if !e[1].isZero() {
		return e[1].signBE()
	}
	return e[0].signBE()
}

func (e *fe2) sign() bool {
	r := new(fe)
	if !e[0].isZero() {
		fromMont(r, &e[0])
		return r[0]&1 == 0
	}
	fromMont(r, &e[1])
	return r[0]&1 == 0
}

func (e *fe6) zero() *fe6 {
	e[0].zero()
	e[1].zero()
	e[2].zero()
	return e
}

func (e *fe6) one() *fe6 {
	e[0].one()
	e[1].zero()
	e[2].zero()
	return e
}

func (e *fe6) set(e2 *fe6) *fe6 {
	e[0].set(&e2[0])
	e[1].set(&e2[1])
	e[2].set(&e2[2])
	return e
}

func (e *fe6) fromMont(a *fe6) {
	e[0].fromMont(&a[0])
	e[1].fromMont(&a[1])
	e[2].fromMont(&a[2])
}

func (e *fe6) fromWide(w *wfe6) {
	e[0].fromWide(&w[0])
	e[1].fromWide(&w[1])
	e[2].fromWide(&w[2])
}

func (e *fe6) rand(r io.Reader) (*fe6, error) {
	a0, err := new(fe2).rand(r)
	if err != nil {
		return nil, err
	}
	e[0].set(a0)
	a1, err := new(fe2).rand(r)
	if err != nil {
		return nil, err
	}
	e[1].set(a1)
	a2, err := new(fe2).rand(r)
	if err != nil {
		return nil, err
	}
	e[2].set(a2)
	return e, nil
}

func (e *fe6) isOne() bool {
	return e[0].isOne() && e[1].isZero() && e[2].isZero()
}

func (e *fe6) isZero() bool {
	return e[0].isZero() && e[1].isZero() && e[2].isZero()
}

func (e *fe6) equal(e2 *fe6) bool {
	return e[0].equal(&e2[0]) && e[1].equal(&e2[1]) && e[2].equal(&e2[2])
}

func (e *fe12) zero() *fe12 {
	e[0].zero()
	e[1].zero()
	return e
}

func (e *fe12) one() *fe12 {
	e[0].one()
	e[1].zero()
	return e
}

func (e *fe12) set(e2 *fe12) *fe12 {
	e[0].set(&e2[0])
	e[1].set(&e2[1])
	return e
}

func (e *fe12) fromMont(a *fe12) {
	e[0].fromMont(&a[0])
	e[1].fromMont(&a[1])
}

func (e *fe12) rand(r io.Reader) (*fe12, error) {
	a0, err := new(fe6).rand(r)
	if err != nil {
		return nil, err
	}
	e[0].set(a0)
	a1, err := new(fe6).rand(r)
	if err != nil {
		return nil, err
	}
	e[1].set(a1)
	return e, nil
}

func (e *fe12) isOne() bool {
	return e[0].isOne() && e[1].isZero()
}

func (e *fe12) isZero() bool {
	return e[0].isZero() && e[1].isZero()
}

func (e *fe12) equal(e2 *fe12) bool {
	return e[0].equal(&e2[0]) && e[1].equal(&e2[1])
}

func (fe *wfe) set(fe2 *wfe) *wfe {
	fe[0] = fe2[0]
	fe[1] = fe2[1]
	fe[2] = fe2[2]
	fe[3] = fe2[3]
	fe[4] = fe2[4]
	fe[5] = fe2[5]
	fe[6] = fe2[6]
	fe[7] = fe2[7]
	fe[8] = fe2[8]
	fe[9] = fe2[9]
	fe[10] = fe2[10]
	fe[11] = fe2[11]
	return fe
}

func (fe *wfe2) set(fe2 *wfe2) *wfe2 {
	fe[0].set(&fe2[0])
	fe[1].set(&fe2[1])
	return fe
}

func (fe *wfe6) set(fe2 *wfe6) *wfe6 {
	fe[0].set(&fe2[0])
	fe[1].set(&fe2[1])
	fe[2].set(&fe2[2])
	return fe
}
