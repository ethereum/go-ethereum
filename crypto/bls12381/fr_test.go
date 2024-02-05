package bls12381

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"testing"
)

func TestScalarField(t *testing.T) {
	r := new(Fr).Set(qr1)
	r.fromMont()
	if r[0] != 1 && r[1] != 0 && r[2] != 0 && r[3] != 0 {
		t.Fatal("bad r value")
	}
	r.Set(qr2)
	r.fromMont()
	r.fromMont()
	if r[0] != 1 && r[1] != 0 && r[2] != 0 && r[3] != 0 {
		t.Fatal("bad r2 value")
	}
	r = &Fr{1}
	r.toMont()
	if !r.Equal(qr1) {
		t.Fatal("mont transformaition failed")
	}
}

func TestFrSerialization(t *testing.T) {
	in := make([]byte, frByteSize)

	e := new(Fr).FromBytes(in)
	if !e.IsZero() {
		t.Fatal("serialization failed, from bytes zero")
	}
	if !bytes.Equal(in, e.ToBytes()) {
		t.Fatal("serialization failed, to bytes zero")
	}

	e = new(Fr).RedFromBytes(in)
	if !e.IsZero() {
		t.Fatal("serialization failed, from bytes zero, reduced")
	}
	if !bytes.Equal(in, e.RedToBytes()) {
		t.Fatal("serialization failed, to bytes zero, reduced")
	}

	a, err := new(Fr).Rand(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	b := new(Fr)
	b.fromBytes(a.bytes())
	if !a.Equal(b) {
		t.Fatal("serialization failed, set bytes")
	}

	b = new(Fr).FromBytes(a.ToBytes())
	if !a.Equal(b) {
		t.Fatal("serialization failed, from/to bytes")
	}

	b = new(Fr).RedFromBytes(a.RedToBytes())
	if !a.Equal(b) {
		t.Fatal("serialization failed, from/to bytes, reduced")
	}
}

func TestFrSliceUint(t *testing.T) {
	s, err := new(Fr).Rand(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sBig := s.ToBig()
	for offset := 0; offset < 260; offset++ {
		a0 := new(big.Int).Rsh(sBig, uint(offset)).Uint64()
		a1 := s.sliceUint64(offset)
		if a0 != a1 {
			t.Fatal("uint slice failed", offset)
		}
	}
}

func TestFrBitTest(t *testing.T) {
	s, err := new(Fr).Rand(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sBig := s.ToBig()
	for i := 0; i < 260; i++ {
		a0 := sBig.Bit(i) == 1
		a1 := s.Bit(i)
		if a0 != a1 {
			t.Fatal("bit test failed", i)
		}
	}
}

func TestFrBitShift(t *testing.T) {
	a, _ := new(Fr).Rand(rand.Reader)
	b := new(Fr).Set(a)
	b.mul2()
	b.div2()
	if !b.Equal(a) {
		t.Fatal("mul2 div2 failed")
	}
	a, _ = new(Fr).Rand(rand.Reader)
	a[0] = a[0] & 0xfffffffffffffffe
	b.Set(a)
	b.div2()
	b.mul2()
	if !b.Equal(a) {
		t.Fatal("mul2 div2 failed")
	}
}

func TestFrAdditionCrossAgainstBigInt(t *testing.T) {
	for i := 0; i < fuz; i++ {
		a, _ := new(Fr).Rand(rand.Reader)
		b, _ := new(Fr).Rand(rand.Reader)
		c := new(Fr)
		bigA := a.ToBig()
		bigB := b.ToBig()
		bigC := new(big.Int)
		c.Add(a, b)
		out1 := c.ToBytes()
		out2 := padBytes(bigC.Add(bigA, bigB).Mod(bigC, qBig).Bytes(), frByteSize)
		if !bytes.Equal(out1, out2) {
			t.Fatal("cross test against big.Int is failed, add")
		}
		c.Double(a)
		out1 = c.ToBytes()
		out2 = padBytes(bigC.Add(bigA, bigA).Mod(bigC, qBig).Bytes(), frByteSize)
		if !bytes.Equal(out1, out2) {
			t.Fatal("cross test against big.Int is failed, double")
		}
		c.Sub(a, b)
		out1 = c.ToBytes()
		out2 = padBytes(bigC.Sub(bigA, bigB).Mod(bigC, qBig).Bytes(), frByteSize)
		if !bytes.Equal(out1, out2) {
			t.Fatal("cross test against big.Int is failed, sub")
		}
		c.Neg(a)
		out1 = c.ToBytes()
		out2 = padBytes(bigC.Neg(bigA).Mod(bigC, qBig).Bytes(), frByteSize)
		if !bytes.Equal(out1, out2) {
			t.Fatal("cross test against big.Int is failed, neg")
		}
	}
}

func TestFrAdditionProperties(t *testing.T) {
	for i := 0; i < fuz; i++ {
		zero := new(Fr)
		a, _ := new(Fr).Rand(rand.Reader)
		b, _ := new(Fr).Rand(rand.Reader)
		c1, c2 := new(Fr), new(Fr)
		c1.Add(a, zero)
		if !c1.Equal(a) {
			t.Fatal("a + 0 == a")
		}
		c1.Sub(a, zero)
		if !c1.Equal(a) {
			t.Fatal("a - 0 == a")
		}
		c1.Double(zero)
		if !c1.Equal(zero) {
			t.Fatal("2 * 0 == 0")
		}
		c1.Neg(zero)
		if !c1.Equal(zero) {
			t.Fatal("-0 == 0")
		}
		c1.Sub(zero, a)
		c2.Neg(a)
		if !c1.Equal(c2) {
			t.Fatal("0-a == -a")
		}
		c1.Double(a)
		c2.Add(a, a)
		if !c1.Equal(c2) {
			t.Fatal("2 * a == a + a")
		}
		c1.Add(a, b)
		c2.Add(b, a)
		if !c1.Equal(c2) {
			t.Fatal("a + b = b + a")
		}
		c1.Sub(a, b)
		c2.Sub(b, a)
		c2.Neg(c2)
		if !c1.Equal(c2) {
			t.Fatal("a - b = - ( b - a )")
		}
		c0, _ := new(Fr).Rand(rand.Reader)
		c1.Add(a, b)
		c1.Add(c1, c0)
		c2.Add(a, c0)
		c2.Add(c2, b)
		if !c1.Equal(c2) {
			t.Fatal("(a + b) + c == (a + c ) + b")
		}
		c1.Sub(a, b)
		c1.Sub(c1, c0)
		c2.Sub(a, c0)
		c2.Sub(c2, b)
		if !c1.Equal(c2) {
			t.Fatal("(a - b) - c == (a - c ) -b")
		}
	}
}

func TestFrMultiplicationCrossAgainstBigInt(t *testing.T) {
	for i := 0; i < fuz; i++ {
		a, _ := new(Fr).Rand(rand.Reader)
		b, _ := new(Fr).Rand(rand.Reader)
		c := new(Fr)
		bigA := a.ToBig()
		bigB := b.ToBig()
		bigC := new(big.Int)
		c.Mul(a, b)
		out1 := c.ToBytes()
		out2 := padBytes(bigC.Mul(bigA, bigB).Mod(bigC, qBig).Bytes(), frByteSize)
		if !bytes.Equal(out1, out2) {
			t.Fatal("cross test against big.Int is failed")
		}
	}
}

func TestFrMultiplicationCrossAgainstBigIntReduced(t *testing.T) {
	for i := 0; i < fuz; i++ {
		a, _ := new(Fr).Rand(rand.Reader)
		b, _ := new(Fr).Rand(rand.Reader)
		c := new(Fr)
		bigA := a.RedToBig()
		bigB := b.RedToBig()
		bigC := new(big.Int)
		c.RedMul(a, b)
		out1 := c.RedToBytes()
		out2 := padBytes(bigC.Mul(bigA, bigB).Mod(bigC, qBig).Bytes(), frByteSize)
		if !bytes.Equal(out1, out2) {
			t.Fatal("cross test against big.Int is failed, reduced")
		}
	}
}

func TestFrMultiplicationProperties(t *testing.T) {
	for i := 0; i < fuz; i++ {
		a, _ := new(Fr).Rand(rand.Reader)
		b, _ := new(Fr).Rand(rand.Reader)
		zero, one := new(Fr).Zero(), new(Fr).One()
		c1, c2 := new(Fr), new(Fr)
		c1.Mul(a, zero)
		if !c1.Equal(zero) {
			t.Fatal("a * 0 == 0")
		}
		c1.Mul(a, one)
		if !c1.Equal(a) {
			t.Fatal("a * 1 == a")
		}
		c1.Mul(a, b)
		c2.Mul(b, a)
		if !c1.Equal(c2) {
			t.Fatal("a * b == b * a")
		}
		c0, _ := new(Fr).Rand(rand.Reader)
		c1.Mul(a, b)
		c1.Mul(c1, c0)
		c2.Mul(c0, b)
		c2.Mul(c2, a)
		if !c1.Equal(c2) {
			t.Fatal("(a * b) * c == (a * c) * b")
		}
		a.Square(zero)
		if !a.Equal(zero) {
			t.Fatal("0^2 == 0")
		}
		a.Square(one)
		if !a.Equal(one) {
			t.Fatal("1^2 == 1")
		}
		_, _ = a.Rand(rand.Reader)
		c1.Square(a)
		c2.Mul(a, a)
		if !c1.Equal(c1) {
			t.Fatal("a^2 == a*a")
		}
	}
}

func TestFrMultiplicationPropertiesReduced(t *testing.T) {
	for i := 0; i < fuz; i++ {
		a, _ := new(Fr).Rand(rand.Reader)
		b, _ := new(Fr).Rand(rand.Reader)
		zero, one := new(Fr).Zero(), new(Fr).RedOne()
		c1, c2 := new(Fr), new(Fr)
		c1.RedMul(a, zero)
		if !c1.Equal(zero) {
			t.Fatal("a * 0 == 0")
		}
		c1.RedMul(a, one)
		if !c1.Equal(a) {
			t.Fatal("a * 1 == a")
		}
		c1.RedMul(a, b)
		c2.RedMul(b, a)
		if !c1.Equal(c2) {
			t.Fatal("a * b == b * a")
		}
		c0, _ := new(Fr).Rand(rand.Reader)
		c1.RedMul(a, b)
		c1.RedMul(c1, c0)
		c2.RedMul(c0, b)
		c2.RedMul(c2, a)
		if !c1.Equal(c2) {
			t.Fatal("(a * b) * c == (a * c) * b")
		}
		a.RedSquare(zero)
		if !a.Equal(zero) {
			t.Fatal("0^2 == 0")
		}
		a.RedSquare(one)
		if !a.Equal(one) {
			t.Fatal("1^2 == 1")
		}
		_, _ = a.Rand(rand.Reader)
		c1.RedSquare(a)
		c2.RedMul(a, a)
		if !c1.Equal(c1) {
			t.Fatal("a^2 == a*a")
		}
	}
}

func TestFrExponentiation(t *testing.T) {
	for i := 0; i < fuz; i++ {
		a, _ := new(Fr).Rand(rand.Reader)
		u := new(Fr)
		u.Exp(a, big.NewInt(0))
		if !u.IsOne() {
			t.Fatal("a^0 == 1")
		}
		u.Exp(a, big.NewInt(1))
		if !u.Equal(a) {
			t.Fatal("a^1 == a")
		}
		v := new(Fr)
		u.Mul(a, a)
		u.Mul(u, u)
		u.Mul(u, u)
		v.Exp(a, big.NewInt(8))
		if !u.Equal(v) {
			t.Fatal("((a^2)^2)^2 == a^8")
		}
		u.Exp(a, qBig)
		if !u.Equal(a) {
			t.Fatal("a^p == a")
		}
		qMinus1 := new(big.Int).Sub(qBig, big.NewInt(1))
		u.Exp(a, qMinus1)
		if !u.IsOne() {
			t.Fatal("a^(p-1) == 1")
		}
	}
}

func TestFrInversion(t *testing.T) {
	for i := 0; i < fuz; i++ {
		u := new(Fr)
		zero, one := new(Fr).Zero(), new(Fr).One()
		u.Inverse(zero)
		if !u.Equal(zero) {
			t.Fatal("(0^-1) == 0)")
		}
		u.Inverse(one)
		if !u.IsOne() {
			t.Fatal("(1^-1) == 1)")
		}
		a, _ := new(Fr).Rand(rand.Reader)
		u.Inverse(a)
		u.Mul(u, a)
		if !u.IsOne() {
			t.Fatal("a * a^-1 == 1")
		}
		v := new(Fr)
		z := new(big.Int)
		u.Exp(a, z.Sub(qBig, big.NewInt(2)))
		v.Inverse(a)
		if !v.Equal(u) {
			t.Fatal("a^(p-2) == a^-1")
		}
	}
}

func TestFnBatchInversion(t *testing.T) {
	for i := 0; i < fuz; i++ {
		zero, one := new(Fr).Zero(), new(Fr).One()

		a, _ := new(Fr).Rand(rand.Reader)
		u := new(Fr)
		z := new(big.Int)
		u.Exp(a, z.Sub(qBig, big.NewInt(2)))

		var arr []Fr
		arr = append(arr, *zero, *one, *a)
		InverseBatchFr(arr)
		if !arr[0].Equal(zero) {
			t.Fatal("(0^-1) == 0)")
		}
		if !arr[1].IsOne() {
			t.Fatal("(1^-1) == 1)")
		}
		if !arr[2].Equal(u) {
			t.Fatal("a^(p-2) == a^-1")
		}
	}
}
