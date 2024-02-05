package bls12381

import (
	"crypto/rand"
	"io"
	"math/big"
	"math/bits"
)

const frByteSize = 32
const frBitSize = 255
const frNumberOfLimbs = 4
const fourWordBitSize = 256

type Fr [4]uint64
type wideFr [8]uint64

func NewFr() *Fr {
	return &Fr{}
}

func (e *Fr) Rand(r io.Reader) (*Fr, error) {
	bi, err := rand.Int(r, qBig)
	if err != nil {
		return nil, err
	}
	_ = e.fromBig(bi)
	return e, nil
}

func (e *Fr) Set(e2 *Fr) *Fr {
	e[0] = e2[0]
	e[1] = e2[1]
	e[2] = e2[2]
	e[3] = e2[3]
	return e
}

func (e *Fr) Zero() *Fr {
	e[0] = 0
	e[1] = 0
	e[2] = 0
	e[3] = 0
	return e
}

func (e *Fr) One() *Fr {
	e.Set(&Fr{1})
	return e
}

func (e *Fr) RedOne() *Fr {
	e.Set(qr1)
	return e
}

func (e *Fr) FromBytes(in []byte) *Fr {
	e.fromBytes(in)
	return e
}

func (e *Fr) RedFromBytes(in []byte) *Fr {
	e.fromBytes(in)
	e.toMont()
	return e
}

func (e *Fr) fromBytes(in []byte) *Fr {
	u := new(big.Int).SetBytes(in)
	_ = e.fromBig(u)
	return e
}

func (e *Fr) fromBig(in *big.Int) *Fr {
	e.Zero()
	_in := new(big.Int).Set(in)
	zero := new(big.Int)
	c0 := _in.Cmp(zero)
	c1 := _in.Cmp(qBig)
	if c0 == -1 || c1 == 1 {
		_in.Mod(_in, qBig)
	}

	words := _in.Bits()      // a little-endian Word slice
	if bits.UintSize == 64 { // in the 64-bit architecture
		for i := 0; i < len(words); i++ {
			e[i] = uint64(words[i])
		}
	} else { // in the 32-bit architecture
		for i := 0; i < len(e); i++ {
			j := i * 2
			if j+1 < len(words) {
				e[i] = uint64(words[j+1])<<32 | uint64(words[j])
			} else if j < len(words) {
				e[i] = uint64(words[j])
			} else {
				e[i] = uint64(0)
			}
		}
	}

	return e
}

func (e *Fr) setUint64(n uint64) *Fr {
	e.Zero()
	e[0] = n
	return e
}

func (e *Fr) ToBytes() []byte {
	return NewFr().Set(e).bytes()
}

func (e *Fr) RedToBytes() []byte {
	out := NewFr().Set(e)
	out.fromMont()
	return out.bytes()
}

func (e *Fr) ToBig() *big.Int {
	return new(big.Int).SetBytes(e.ToBytes())
}

func (e *Fr) RedToBig() *big.Int {
	return new(big.Int).SetBytes(e.RedToBytes())
}

func (e *Fr) bytes() []byte {
	out := make([]byte, frByteSize)
	var a int
	for i := 0; i < frNumberOfLimbs; i++ {
		a = frByteSize - i*8
		out[a-1] = byte(e[i])
		out[a-2] = byte(e[i] >> 8)
		out[a-3] = byte(e[i] >> 16)
		out[a-4] = byte(e[i] >> 24)
		out[a-5] = byte(e[i] >> 32)
		out[a-6] = byte(e[i] >> 40)
		out[a-7] = byte(e[i] >> 48)
		out[a-8] = byte(e[i] >> 56)
	}
	return out
}

func (e *Fr) IsZero() bool {
	return (e[3] | e[2] | e[1] | e[0]) == 0
}

func (e *Fr) IsOne() bool {
	return e.Equal(&Fr{1})
}

func (e *Fr) IsRedOne() bool {
	return e.Equal(qr1)
}

func (e *Fr) Equal(e2 *Fr) bool {
	return e2[0] == e[0] && e2[1] == e[1] && e2[2] == e[2] && e2[3] == e[3]
}

func (e *Fr) Cmp(e1 *Fr) int {
	for i := frNumberOfLimbs - 1; i >= 0; i-- {
		if e[i] > e1[i] {
			return 1
		} else if e[i] < e1[i] {
			return -1
		}
	}
	return 0
}

func (e *Fr) sliceUint64(from int) uint64 {
	if from < 64 {
		return e[0]>>from | e[1]<<(64-from)
	} else if from < 128 {
		return e[1]>>(from-64) | e[2]<<(128-from)
	} else if from < 192 {
		return e[2]>>(from-128) | e[3]<<(192-from)
	}
	return e[3] >> (from - 192)
}

func (e *Fr) div2() {
	e[0] = e[0]>>1 | e[1]<<63
	e[1] = e[1]>>1 | e[2]<<63
	e[2] = e[2]>>1 | e[3]<<63
	e[3] = e[3] >> 1
}

func (e *Fr) mul2() uint64 {
	c := e[3] >> 63
	e[3] = e[3]<<1 | e[2]>>63
	e[2] = e[2]<<1 | e[1]>>63
	e[1] = e[1]<<1 | e[0]>>63
	e[0] = e[0] << 1
	return c
}

func (e *Fr) isEven() bool {
	var mask uint64 = 1
	return e[0]&mask == 0
}

func (e *Fr) Bit(at int) bool {
	if at < 64 {
		return (e[0]>>at)&1 == 1
	} else if at < 128 {
		return (e[1]>>(at-64))&1 == 1
	} else if at < 192 {
		return (e[2]>>(at-128))&1 == 1
	} else if at < 256 {
		return (e[3]>>(at-192))&1 == 1
	}
	return false
}

func (e *Fr) toMont() {
	e.RedMul(e, qr2)
}

func (e *Fr) fromMont() {
	e.RedMul(e, &Fr{1})
}

func (e *Fr) FromRed() {
	e.fromMont()
}

func (e *Fr) ToRed() {
	e.toMont()
}

func (e *Fr) Add(a, b *Fr) {
	addFR(e, a, b)
}

func (e *Fr) Double(a *Fr) {
	doubleFR(e, a)
}

func (e *Fr) Sub(a, b *Fr) {
	subFR(e, a, b)
}

func (e *Fr) Neg(a *Fr) {
	negFR(e, a)
}

func (e *Fr) Mul(a, b *Fr) {
	e.RedMul(a, b)
	e.toMont()
}

func (e *Fr) RedMul(a, b *Fr) {
	mulFR(e, a, b)
}

func (e *Fr) Square(a *Fr) {
	e.RedSquare(a)
	e.toMont()
}

func (e *Fr) RedSquare(a *Fr) {
	squareFR(e, a)
}

func (e *Fr) RedExp(a *Fr, ee *big.Int) {
	z := new(Fr).RedOne()
	for i := ee.BitLen(); i >= 0; i-- {
		z.RedSquare(z)
		if ee.Bit(i) == 1 {
			z.RedMul(z, a)
		}
	}
	e.Set(z)
}

func (e *Fr) Exp(a *Fr, ee *big.Int) {
	e.Set(a).toMont()
	e.RedExp(e, ee)
	e.fromMont()

}

func RedInverseBatchFr(in []Fr) {
	inverseBatchFr(in, func(a, b *Fr) { a.RedInverse(b) })
}

func InverseBatchFr(in []Fr) {
	inverseBatchFr(in, func(a, b *Fr) { a.Inverse(b) })
}

func inverseBatchFr(in []Fr, invFn func(out *Fr, in *Fr)) {
	n, N, setFirst := 0, len(in), false

	for i := 0; i < len(in); i++ {
		if !in[i].IsZero() {
			n++
		}
	}
	if n == 0 {
		return
	}

	tA := make([]Fr, n)
	tB := make([]Fr, n)

	for i, j := 0, 0; i < N; i++ {
		if !in[i].IsZero() {
			if !setFirst {
				setFirst = true
				tA[j].Set(&in[i])
			} else {
				tA[j].Mul(&in[i], &tA[j-1])
			}
			j = j + 1
		}
	}

	invFn(&tB[n-1], &tA[n-1])
	for i, j := N-1, n-1; j != 0; i-- {
		if !in[i].IsZero() {
			tB[j-1].Mul(&tB[j], &in[i])
			j = j - 1
		}
	}

	for i, j := 0, 0; i < N; i++ {
		if !in[i].IsZero() {
			if setFirst {
				setFirst = false
				in[i].Set(&tB[j])
			} else {
				in[i].Mul(&tA[j-1], &tB[j])
			}
			j = j + 1
		}
	}
}

func (e *Fr) Inverse(a *Fr) {
	e.Set(a).toMont()
	e.RedInverse(e)
	e.fromMont()
}

func (e *Fr) RedInverse(ei *Fr) {
	if ei.IsZero() {
		e.Zero()
		return
	}
	u := new(Fr).Set(&q)
	v := new(Fr).Set(ei)
	s := &Fr{1}
	r := &Fr{0}
	var k int
	var z uint64
	var found = false
	// Phase 1
	for i := 0; i < fourWordBitSize*2; i++ {
		if v.IsZero() {
			found = true
			break
		}
		if u.isEven() {
			u.div2()
			s.mul2()
		} else if v.isEven() {
			v.div2()
			z += r.mul2()
		} else if u.Cmp(v) == 1 {
			lsubAssignFR(u, v)
			u.div2()
			laddAssignFR(r, s)
			s.mul2()
		} else {
			lsubAssignFR(v, u)
			v.div2()
			laddAssignFR(s, r)
			z += r.mul2()
		}
		k += 1
	}

	if !found {
		e.Zero()
		return
	}

	if k < frBitSize || k > frBitSize+fourWordBitSize {
		e.Zero()
		return
	}

	if r.Cmp(&q) != -1 || z > 0 {
		lsubAssignFR(r, &q)
	}
	u.Set(&q)
	lsubAssignFR(u, r)

	// Phase 2
	for i := k; i < 2*fourWordBitSize; i++ {
		doubleFR(u, u)
	}
	e.Set(u)
}

func (ew *wideFr) mul(a, b *Fr) {
	wmulFR(ew, a, b)
}

func (ew *wideFr) add(a *wideFr) {
	waddFR(ew, a)
}

func (ew *wideFr) round() *Fr {
	ew.add(halfR)
	return ew.high()
}

func (ew *wideFr) high() *Fr {
	e := new(Fr)
	e[0] = ew[4]
	e[1] = ew[5]
	e[2] = ew[6]
	e[3] = ew[7]
	return e
}

func (ew *wideFr) low() *Fr {
	e := new(Fr)
	e[0] = ew[0]
	e[1] = ew[1]
	e[2] = ew[2]
	e[3] = ew[3]
	return e
}

func (e *wideFr) bytes() []byte {
	out := make([]byte, frByteSize*2)
	var a int
	for i := 0; i < frNumberOfLimbs*2; i++ {
		a = frByteSize*2 - i*8
		out[a-1] = byte(e[i])
		out[a-2] = byte(e[i] >> 8)
		out[a-3] = byte(e[i] >> 16)
		out[a-4] = byte(e[i] >> 24)
		out[a-5] = byte(e[i] >> 32)
		out[a-6] = byte(e[i] >> 40)
		out[a-7] = byte(e[i] >> 48)
		out[a-8] = byte(e[i] >> 56)
	}
	return out
}
