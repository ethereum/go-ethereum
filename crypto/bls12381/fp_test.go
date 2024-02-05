package bls12381

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"testing"
)

func TestFpSerialization(t *testing.T) {
	t.Run("zero", func(t *testing.T) {
		in := make([]byte, fpByteSize)
		fe, err := fromBytes(in)
		if err != nil {
			t.Fatal(err)
		}
		if !fe.isZero() {
			t.Fatal("serialization failed")
		}
		if !bytes.Equal(in, toBytes(fe)) {
			t.Fatal("serialization failed")
		}
	})
	t.Run("bytes", func(t *testing.T) {
		for i := 0; i < fuz; i++ {
			a, _ := new(fe).rand(rand.Reader)
			b, err := fromBytes(toBytes(a))
			if err != nil {
				t.Fatal(err)
			}
			if !a.equal(b) {
				t.Fatal("serialization failed")
			}
		}
	})
	t.Run("string", func(t *testing.T) {
		for i := 0; i < fuz; i++ {
			a, _ := new(fe).rand(rand.Reader)
			b, err := fromString(toString(a))
			if err != nil {
				t.Fatal(err)
			}
			if !a.equal(b) {
				t.Fatal("encoding or decoding failed")
			}
		}
	})
	t.Run("big", func(t *testing.T) {
		for i := 0; i < fuz; i++ {
			a, _ := new(fe).rand(rand.Reader)
			b, err := fromBig(toBig(a))
			if err != nil {
				t.Fatal(err)
			}
			if !a.equal(b) {
				t.Fatal("encoding or decoding failed")
			}
		}
	})
}

func TestFpAdditionCrossAgainstBigInt(t *testing.T) {
	for i := 0; i < fuz; i++ {
		a, _ := new(fe).rand(rand.Reader)
		b, _ := new(fe).rand(rand.Reader)
		c := new(fe)
		big_a := a.big()
		big_b := b.big()
		big_c := new(big.Int)
		add(c, a, b)
		out_1 := c.bytes()
		out_2 := padBytes(big_c.Add(big_a, big_b).Mod(big_c, modulus.big()).Bytes(), fpByteSize)
		if !bytes.Equal(out_1, out_2) {
			t.Fatal("cross test against big.Int is failed A")
		}
		double(c, a)
		out_1 = c.bytes()
		out_2 = padBytes(big_c.Add(big_a, big_a).Mod(big_c, modulus.big()).Bytes(), fpByteSize)
		if !bytes.Equal(out_1, out_2) {
			t.Fatal("cross test against big.Int is failed B")
		}
		sub(c, a, b)
		out_1 = c.bytes()
		out_2 = padBytes(big_c.Sub(big_a, big_b).Mod(big_c, modulus.big()).Bytes(), fpByteSize)
		if !bytes.Equal(out_1, out_2) {
			t.Fatal("cross test against big.Int is failed C")
		}
		neg(c, a)
		out_1 = c.bytes()
		out_2 = padBytes(big_c.Neg(big_a).Mod(big_c, modulus.big()).Bytes(), fpByteSize)
		if !bytes.Equal(out_1, out_2) {
			t.Fatal("cross test against big.Int is failed D")
		}
	}
}

func TestFpAdditionCrossAgainstBigIntAssigned(t *testing.T) {
	for i := 0; i < fuz; i++ {
		a, _ := new(fe).rand(rand.Reader)
		b, _ := new(fe).rand(rand.Reader)
		big_a, big_b := a.big(), b.big()
		addAssign(a, b)
		out_1 := a.bytes()
		out_2 := padBytes(big_a.Add(big_a, big_b).Mod(big_a, modulus.big()).Bytes(), fpByteSize)
		if !bytes.Equal(out_1, out_2) {
			t.Fatal("cross test against big.Int is failed A")
		}
		a, _ = new(fe).rand(rand.Reader)
		big_a = a.big()
		doubleAssign(a)
		out_1 = a.bytes()
		out_2 = padBytes(big_a.Add(big_a, big_a).Mod(big_a, modulus.big()).Bytes(), fpByteSize)
		if !bytes.Equal(out_1, out_2) {
			t.Fatal("cross test against big.Int is failed B")
		}
		a, _ = new(fe).rand(rand.Reader)
		b, _ = new(fe).rand(rand.Reader)
		big_a, big_b = a.big(), b.big()
		subAssign(a, b)
		out_1 = a.bytes()
		out_2 = padBytes(big_a.Sub(big_a, big_b).Mod(big_a, modulus.big()).Bytes(), fpByteSize)
		if !bytes.Equal(out_1, out_2) {
			t.Fatal("cross test against big.Int is failed A")
		}
	}
}

func TestFpAdditionProperties(t *testing.T) {
	for i := 0; i < fuz; i++ {

		zero := new(fe).zero()
		a, _ := new(fe).rand(rand.Reader)
		b, _ := new(fe).rand(rand.Reader)
		c1, c2 := new(fe), new(fe)
		add(c1, a, zero)
		if !c1.equal(a) {
			t.Fatal("a + 0 == a")
		}
		sub(c1, a, zero)
		if !c1.equal(a) {
			t.Fatal("a - 0 == a")
		}
		double(c1, zero)
		if !c1.equal(zero) {
			t.Fatal("2 * 0 == 0")
		}
		neg(c1, zero)
		if !c1.equal(zero) {
			t.Fatal("-0 == 0")
		}
		sub(c1, zero, a)
		neg(c2, a)
		if !c1.equal(c2) {
			t.Fatal("0-a == -a")
		}
		double(c1, a)
		add(c2, a, a)
		if !c1.equal(c2) {
			t.Fatal("2 * a == a + a")
		}
		add(c1, a, b)
		add(c2, b, a)
		if !c1.equal(c2) {
			t.Fatal("a + b = b + a")
		}
		sub(c1, a, b)
		sub(c2, b, a)
		neg(c2, c2)
		if !c1.equal(c2) {
			t.Fatal("a - b = - ( b - a )")
		}
		cx, _ := new(fe).rand(rand.Reader)
		add(c1, a, b)
		add(c1, c1, cx)
		add(c2, a, cx)
		add(c2, c2, b)
		if !c1.equal(c2) {
			t.Fatal("(a + b) + c == (a + c ) + b")
		}
		sub(c1, a, b)
		sub(c1, c1, cx)
		sub(c2, a, cx)
		sub(c2, c2, b)
		if !c1.equal(c2) {
			t.Fatal("(a - b) - c == (a - c ) -b")
		}
	}
}

func TestFpAdditionPropertiesAssigned(t *testing.T) {
	for i := 0; i < fuz; i++ {
		zero := new(fe).zero()
		a, b := new(fe), new(fe)
		_, _ = a.rand(rand.Reader)
		b.set(a)
		addAssign(a, zero)
		if !a.equal(b) {
			t.Fatal("a + 0 == a")
		}
		subAssign(a, zero)
		if !a.equal(b) {
			t.Fatal("a - 0 == a")
		}
		a.set(zero)
		doubleAssign(a)
		if !a.equal(zero) {
			t.Fatal("2 * 0 == 0")
		}
		a.set(zero)
		subAssign(a, b)
		neg(b, b)
		if !a.equal(b) {
			t.Fatal("0-a == -a")
		}
		_, _ = a.rand(rand.Reader)
		b.set(a)
		doubleAssign(a)
		addAssign(b, b)
		if !a.equal(b) {
			t.Fatal("2 * a == a + a")
		}
		_, _ = a.rand(rand.Reader)
		_, _ = b.rand(rand.Reader)
		c1, c2 := new(fe).set(a), new(fe).set(b)
		addAssign(c1, b)
		addAssign(c2, a)
		if !c1.equal(c2) {
			t.Fatal("a + b = b + a")
		}
		_, _ = a.rand(rand.Reader)
		_, _ = b.rand(rand.Reader)
		c1.set(a)
		c2.set(b)
		subAssign(c1, b)
		subAssign(c2, a)
		neg(c2, c2)
		if !c1.equal(c2) {
			t.Fatal("a - b = - ( b - a )")
		}
		_, _ = a.rand(rand.Reader)
		_, _ = b.rand(rand.Reader)
		c, _ := new(fe).rand(rand.Reader)
		a0 := new(fe).set(a)
		addAssign(a, b)
		addAssign(a, c)
		addAssign(b, c)
		addAssign(b, a0)
		if !a.equal(b) {
			t.Fatal("(a + b) + c == (b + c) + a")
		}
		_, _ = a.rand(rand.Reader)
		_, _ = b.rand(rand.Reader)
		_, _ = c.rand(rand.Reader)
		a0.set(a)
		subAssign(a, b)
		subAssign(a, c)
		subAssign(a0, c)
		subAssign(a0, b)
		if !a.equal(a0) {
			t.Fatal("(a - b) - c == (a - c) -b")
		}
	}
}

func TestFpLazyOperations(t *testing.T) {
	for i := 0; i < fuz; i++ {
		a, _ := new(fe).rand(rand.Reader)
		b, _ := new(fe).rand(rand.Reader)
		c, _ := new(fe).rand(rand.Reader)
		c0 := new(fe)
		c1 := new(fe)
		ladd(c0, a, b)
		add(c1, a, b)
		mul(c0, c0, c)
		mul(c1, c1, c)
		if !c0.equal(c1) {
			t.Fatal("(a + b) * c == (a l+ b) * c")
		}
		_, _ = a.rand(rand.Reader)
		b.set(a)
		ldouble(a, a)
		ladd(b, b, b)
		if !a.equal(b) {
			t.Fatal("2 l* a = a l+ a")
		}
		_, _ = a.rand(rand.Reader)
		_, _ = b.rand(rand.Reader)
		_, _ = c.rand(rand.Reader)
		a0 := new(fe).set(a)
		lsubAssign(a, b)
		laddAssign(a, &modulus)
		mul(a, a, c)
		subAssign(a0, b)
		mul(a0, a0, c)
		if !a.equal(a0) {
			t.Fatal("((a l- b) + p) * c = (a-b) * c")
		}
	}
}

func TestFpMultiplicationCrossAgainstBigInt(t *testing.T) {
	for i := 0; i < fuz; i++ {
		a, _ := new(fe).rand(rand.Reader)
		b, _ := new(fe).rand(rand.Reader)
		c := new(fe)
		big_a := toBig(a)
		big_b := toBig(b)
		big_c := new(big.Int)
		mul(c, a, b)
		out_1 := toBytes(c)
		out_2 := padBytes(big_c.Mul(big_a, big_b).Mod(big_c, modulus.big()).Bytes(), fpByteSize)
		if !bytes.Equal(out_1, out_2) {
			t.Fatal("cross test against big.Int is failed")
		}
	}
}

func TestFpMultiplicationProperties(t *testing.T) {
	for i := 0; i < fuz; i++ {
		a, _ := new(fe).rand(rand.Reader)
		b, _ := new(fe).rand(rand.Reader)
		zero, one := new(fe).zero(), new(fe).one()
		c1, c2 := new(fe), new(fe)
		mul(c1, a, zero)
		if !c1.equal(zero) {
			t.Fatal("a * 0 == 0")
		}
		mul(c1, a, one)
		if !c1.equal(a) {
			t.Fatal("a * 1 == a")
		}
		mul(c1, a, b)
		mul(c2, b, a)
		if !c1.equal(c2) {
			t.Fatal("a * b == b * a")
		}
		cx, _ := new(fe).rand(rand.Reader)
		mul(c1, a, b)
		mul(c1, c1, cx)
		mul(c2, cx, b)
		mul(c2, c2, a)
		if !c1.equal(c2) {
			t.Fatal("(a * b) * c == (a * c) * b")
		}
		square(a, zero)
		if !a.equal(zero) {
			t.Fatal("0^2 == 0")
		}
		square(a, one)
		if !a.equal(one) {
			t.Fatal("1^2 == 1")
		}
		_, _ = a.rand(rand.Reader)
		square(c1, a)
		mul(c2, a, a)
		if !c1.equal(c1) {
			t.Fatal("a^2 == a*a")
		}
	}
}

func TestFpExponentiation(t *testing.T) {
	for i := 0; i < fuz; i++ {
		a, _ := new(fe).rand(rand.Reader)
		u := new(fe)
		exp(u, a, big.NewInt(0))
		if !u.isOne() {
			t.Fatal("a^0 == 1")
		}
		exp(u, a, big.NewInt(1))
		if !u.equal(a) {
			t.Fatal("a^1 == a")
		}
		v := new(fe)
		mul(u, a, a)
		mul(u, u, u)
		mul(u, u, u)
		exp(v, a, big.NewInt(8))
		if !u.equal(v) {
			t.Fatal("((a^2)^2)^2 == a^8")
		}
		p := modulus.big()
		exp(u, a, p)
		if !u.equal(a) {
			t.Fatal("a^p == a")
		}
		exp(u, a, p.Sub(p, big.NewInt(1)))
		if !u.isOne() {
			t.Fatal("a^(p-1) == 1")
		}
	}
}

func TestFpInversion(t *testing.T) {
	for i := 0; i < fuz; i++ {
		u := new(fe)
		zero, one := new(fe).zero(), new(fe).one()
		inverse(u, zero)
		if !u.equal(zero) {
			t.Fatal("(0^-1) == 0)")
		}
		inverse(u, one)
		if !u.equal(one) {
			t.Fatal("(1^-1) == 1)")
		}
		a, _ := new(fe).rand(rand.Reader)
		inverse(u, a)
		mul(u, u, a)
		if !u.equal(one) {
			t.Fatal("(r*a) * r*(a^-1) == r)")
		}
		v := new(fe)
		p := modulus.big()
		exp(u, a, p.Sub(p, big.NewInt(2)))
		inverse(v, a)
		if !v.equal(u) {
			t.Fatal("a^(p-2) == a^-1")
		}
	}
}

func TestFpBatchInversion(t *testing.T) {
	n := 20
	for i := 0; i < n; i++ {
		e0 := make([]fe, n)
		e1 := make([]fe, n)
		for j := 0; j < n; j++ {
			if j != i {
				e, err := new(fe).rand(rand.Reader)
				if err != nil {
					t.Fatal(err)
				}
				e0[j].set(e)
			}
			inverse(&e1[j], &e0[j])
		}

		inverseBatch(e0)
		for j := 0; j < n; j++ {
			if !e0[j].equal(&e1[j]) {
				t.Fatal("batch inversion failed")
			}
		}
	}
}

func TestFpSquareRoot(t *testing.T) {
	if sqrt(new(fe), nonResidue1) {
		t.Fatal("non residue cannot have a sqrt")
	}
	for i := 0; i < fuz; i++ {
		a, _ := new(fe).rand(rand.Reader)
		r0, r1 := new(fe), new(fe)
		d0 := sqrt(r0, a)
		d1 := _sqrt(r1, a)
		if d0 != d1 {
			t.Fatal("sqrt decision failed")
		}
		if d0 {
			square(r0, r0)
			square(r1, r1)
			if !r0.equal(r1) {
				t.Fatal("sqrt failed")
			}
			if !r0.equal(a) {
				t.Fatal("sqrt failed")
			}
		}
	}
}

func TestFpNonResidue(t *testing.T) {
	if !isQuadraticNonResidue(nonResidue1) {
		t.Fatal("element is quadratic non residue, 1")
	}
	if isQuadraticNonResidue(new(fe).one()) {
		t.Fatal("one is not quadratic non residue")
	}
	if !isQuadraticNonResidue(new(fe).zero()) {
		t.Fatal("should accept zero as quadratic non residue")
	}
	for i := 0; i < fuz; i++ {
		a, _ := new(fe).rand(rand.Reader)
		square(a, a)
		if isQuadraticNonResidue(a) {
			t.Fatal("element is not quadratic non residue")
		}
	}
	for i := 0; i < fuz; i++ {
		a, _ := new(fe).rand(rand.Reader)
		if !sqrt(new(fe), a) {
			if !isQuadraticNonResidue(a) {
				t.Fatal("element is quadratic non residue, 2", i)
			}
		} else {
			i -= 1
		}
	}
}

func TestWFp(t *testing.T) {
	w := new(wfe)
	a := new(fe)
	fromWide(a, w)
	if !a.isZero() {
		t.Fatal("expect zero")
	}
	w[0] = r1[0]
	w[1] = r1[1]
	w[2] = r1[2]
	w[3] = r1[3]
	w[4] = r1[4]
	w[5] = r1[5]
	fromWide(a, w)
	if !(a[0] == 1 && a[1] == 0 && a[2] == 0 && a[3] == 0 && a[4] == 0 && a[5] == 0) {
		t.Fatal("expect one")
	}
}

func TestWFpAddition(t *testing.T) {
	for i := 0; i < fuz; i++ {
		a, _ := new(fe).rand(rand.Reader)
		b, _ := new(fe).rand(rand.Reader)
		w0, w1 := new(wfe), new(wfe)
		c0, c1 := new(fe), new(fe)

		wmul(w0, a, b)
		w1.set(w0)
		wadd(w0, w0, w0)
		wadd(w0, w0, w0)
		wadd(w0, w0, w0)
		lwadd(w1, w1, w1)
		lwadd(w1, w1, w1)
		lwadd(w1, w1, w1)
		fromWide(c0, w0)
		fromWide(c1, w1)

		if !c1.equal(c0) {
			t.Fatal("addition failed")
		}

		wmul(w0, a, b)
		w1.set(w0)
		wdouble(w0, w0)
		wdouble(w0, w0)
		wdouble(w0, w0)
		lwdouble(w1, w1)
		lwdouble(w1, w1)
		lwdouble(w1, w1)
		fromWide(c0, w0)
		fromWide(c1, w1)

		if !c1.equal(c0) {
			t.Fatal("doubling failed")
		}

		wmul(w0, a, &fe{10001})
		wmul(w1, a, &fe{10000})
		w2 := new(wfe)
		wsub(w2, w0, w1)
		lwsub(w0, w0, w1)
		fromWide(c0, w2)
		fromWide(c1, w0)

		fromMont(a, a)
		if !c1.equal(a) {
			t.Fatal("subtraction failed")
		}
		if !c0.equal(a) {
			t.Fatal("subtraction failed")
		}

		wmul(w0, a, &fe{10001})
		wmul(w1, a, &fe{10000})
		wsub(w0, w1, w0)
		fromWide(c0, w0)

		neg(a, a)
		fromMont(a, a)
		if !c0.equal(a) {
			t.Fatal("subtraction failed")
		}

	}
}

func TestWFpMultiplication(t *testing.T) {
	for i := 0; i < fuz; i++ {
		a0, _ := new(fe).rand(rand.Reader)
		b0, _ := new(fe).rand(rand.Reader)
		a1, _ := new(fe).rand(rand.Reader)
		b1, _ := new(fe).rand(rand.Reader)
		w0, w1, w2, w3 := new(wfe), new(wfe), new(wfe), new(wfe)
		c0, c1 := new(fe), new(fe)
		r0, r1 := new(fe), new(fe)

		wmul(w0, a0, b0)
		fromWide(r0, w0)
		mul(r1, a0, b0)

		if !r1.equal(r0) {
			t.Fatal("multiplication failed")
		}

		wmul(w0, a0, b0)
		wmul(w1, a1, b1)
		lwadd(w0, w0, w1)
		fromWide(r0, w0)

		mul(c0, a0, b0)
		mul(c1, a1, b1)
		add(r1, c0, c1)

		if !r1.equal(r0) {
			t.Fatal("multiplication failed")
		}

		wmul(w0, a0, b0)
		wmul(w1, a0, b1)
		wmul(w2, a1, b0)
		wmul(w3, a1, b1)
		lwadd(w0, w0, w1)
		lwadd(w0, w0, w2)
		lwadd(w0, w0, w3)
		fromWide(r0, w0)

		add(c0, a0, a1)
		add(c1, b0, b1)
		mul(r1, c0, c1)

		if !r1.equal(r0) {
			t.Fatal("multiplication failed")
		}
	}
}

func TestFp2Serialization(t *testing.T) {
	f := newFp2()
	for i := 0; i < fuz; i++ {
		a, _ := new(fe2).rand(rand.Reader)
		b, err := f.fromBytes(f.toBytes(a))
		if err != nil {
			t.Fatal(err)
		}
		if !a.equal(b) {
			t.Fatal("serialization failed")
		}
	}
}

func TestFp2AdditionProperties(t *testing.T) {
	f := newFp2()
	for i := 0; i < fuz; i++ {
		zero := f.zero()
		a, _ := new(fe2).rand(rand.Reader)
		b, _ := new(fe2).rand(rand.Reader)
		c1 := f.new()
		c2 := f.new()
		fp2Add(c1, a, zero)
		if !c1.equal(a) {
			t.Fatal("a + 0 == a")
		}
		fp2Sub(c1, a, zero)
		if !c1.equal(a) {
			t.Fatal("a - 0 == a")
		}
		fp2Double(c1, zero)
		if !c1.equal(zero) {
			t.Fatal("2 * 0 == 0")
		}
		fp2Neg(c1, zero)
		if !c1.equal(zero) {
			t.Fatal("-0 == 0")
		}
		fp2Sub(c1, zero, a)
		fp2Neg(c2, a)
		if !c1.equal(c2) {
			t.Fatal("0-a == -a")
		}
		fp2Double(c1, a)
		fp2Add(c2, a, a)
		if !c1.equal(c2) {
			t.Fatal("2 * a == a + a")
		}
		fp2Add(c1, a, b)
		fp2Add(c2, b, a)
		if !c1.equal(c2) {
			t.Fatal("a + b = b + a")
		}
		fp2Sub(c1, a, b)
		fp2Sub(c2, b, a)
		fp2Neg(c2, c2)
		if !c1.equal(c2) {
			t.Fatal("a - b = - ( b - a )")
		}
		cx, _ := new(fe2).rand(rand.Reader)
		fp2Add(c1, a, b)
		fp2Add(c1, c1, cx)
		fp2Add(c2, a, cx)
		fp2Add(c2, c2, b)
		if !c1.equal(c2) {
			t.Fatal("(a + b) + c == (a + c ) + b")
		}
		fp2Sub(c1, a, b)
		fp2Sub(c1, c1, cx)
		fp2Sub(c2, a, cx)
		fp2Sub(c2, c2, b)
		if !c1.equal(c2) {
			t.Fatal("(a - b) - c == (a - c ) -b")
		}
	}
}

func TestFp2AdditionPropertiesAssigned(t *testing.T) {
	for i := 0; i < fuz; i++ {
		zero := new(fe2).zero()
		a, b := new(fe2), new(fe2)
		_, _ = a.rand(rand.Reader)
		b.set(a)
		fp2AddAssign(a, zero)
		if !a.equal(b) {
			t.Fatal("a + 0 == a")
		}
		fp2SubAssign(a, zero)
		if !a.equal(b) {
			t.Fatal("a - 0 == a")
		}
		a.set(zero)
		fp2DoubleAssign(a)
		if !a.equal(zero) {
			t.Fatal("2 * 0 == 0")
		}
		a.set(zero)
		fp2SubAssign(a, b)
		fp2Neg(b, b)
		if !a.equal(b) {
			t.Fatal("0-a == -a")
		}
		_, _ = a.rand(rand.Reader)
		b.set(a)
		fp2DoubleAssign(a)
		fp2AddAssign(b, b)
		if !a.equal(b) {
			t.Fatal("2 * a == a + a")
		}
		_, _ = a.rand(rand.Reader)
		_, _ = b.rand(rand.Reader)
		c1, c2 := new(fe2).set(a), new(fe2).set(b)
		fp2AddAssign(c1, b)
		fp2AddAssign(c2, a)
		if !c1.equal(c2) {
			t.Fatal("a + b = b + a")
		}
		_, _ = a.rand(rand.Reader)
		_, _ = b.rand(rand.Reader)
		c1.set(a)
		c2.set(b)
		fp2SubAssign(c1, b)
		fp2SubAssign(c2, a)
		fp2Neg(c2, c2)
		if !c1.equal(c2) {
			t.Fatal("a - b = - ( b - a )")
		}
		_, _ = a.rand(rand.Reader)
		_, _ = b.rand(rand.Reader)
		c, _ := new(fe2).rand(rand.Reader)
		a0 := new(fe2).set(a)
		fp2AddAssign(a, b)
		fp2AddAssign(a, c)
		fp2AddAssign(b, c)
		fp2AddAssign(b, a0)
		if !a.equal(b) {
			t.Fatal("(a + b) + c == (b + c) + a")
		}
		_, _ = a.rand(rand.Reader)
		_, _ = b.rand(rand.Reader)
		_, _ = c.rand(rand.Reader)
		a0.set(a)
		fp2SubAssign(a, b)
		fp2SubAssign(a, c)
		fp2SubAssign(a0, c)
		fp2SubAssign(a0, b)
		if !a.equal(a0) {
			t.Fatal("(a - b) - c == (a - c) -b")
		}
	}
}

func TestFp2LazyOperations(t *testing.T) {
	f := newFp2()
	for i := 0; i < fuz; i++ {
		a, _ := new(fe2).rand(rand.Reader)
		b, _ := new(fe2).rand(rand.Reader)
		c, _ := new(fe2).rand(rand.Reader)
		c0 := new(fe2)
		c1 := new(fe2)
		fp2Ladd(c0, a, b)
		fp2Add(c1, a, b)
		fp2LaddAssign(a, b)
		f.mulAssign(c0, c)
		f.mulAssign(c1, c)
		f.mulAssign(a, c)
		if !c0.equal(c1) {
			t.Fatal("(a + b) * c == (a l+ b) * c")
		}
		if !c0.equal(c1) {
			t.Fatal("(a + b) * c == (a l+ b) * c")
		}
	}
}

func TestFp2MulByNonResidue(t *testing.T) {
	f := newFp2()
	for i := 0; i < fuz; i++ {
		a, _ := new(fe2).rand(rand.Reader)
		r0, r1, r2, r3 := new(fe2), new(fe2), new(fe2), new(fe2)
		f.mul(r1, a, nonResidue2)
		mulByNonResidue(r0, a)
		r2.set(a)
		mulByNonResidueAssign(r2)
		_fp2MulByNonResidue(r3, a)

		if !r0.equal(r1) {
			t.Fatal("mul by non residue failed")
		}
		if !r0.equal(r2) {
			t.Fatal("mul by non residue failed")
		}
		if !r0.equal(r3) {
			t.Fatal("mul by non residue failed")
		}
	}
}

func TestFp2MultiplicationProperties(t *testing.T) {
	f := newFp2()
	for i := 0; i < fuz; i++ {
		a, _ := new(fe2).rand(rand.Reader)
		b, _ := new(fe2).rand(rand.Reader)
		zero := f.zero()
		one := f.one()
		c1, c2 := f.new(), f.new()
		f.mul(c1, a, zero)
		if !c1.equal(zero) {
			t.Fatal("a * 0 == 0")
		}
		f.mul(c1, a, one)
		if !c1.equal(a) {
			t.Fatal("a * 1 == a")
		}
		f.mul(c1, a, b)
		f.mul(c2, b, a)
		if !c1.equal(c2) {
			t.Fatal("a * b == b * a")
		}
		cx, _ := new(fe2).rand(rand.Reader)
		f.mul(c1, a, b)
		f.mul(c1, c1, cx)
		f.mul(c2, cx, b)
		f.mul(c2, c2, a)
		if !c1.equal(c2) {
			t.Fatal("(a * b) * c == (a * c) * b")
		}
		f.square(a, zero)
		if !a.equal(zero) {
			t.Fatal("0^2 == 0")
		}
		f.square(a, one)
		if !a.equal(one) {
			t.Fatal("1^2 == 1")
		}
		_, _ = a.rand(rand.Reader)
		f.square(c1, a)
		f.mul(c2, a, a)
		if !c2.equal(c1) {
			t.Fatal("a^2 == a*a")
		}
	}
}

func TestFp2MultiplicationPropertiesAssigned(t *testing.T) {
	f := newFp2()
	for i := 0; i < fuz; i++ {
		a, _ := new(fe2).rand(rand.Reader)
		zero, one := new(fe2).zero(), new(fe2).one()
		f.mulAssign(a, zero)
		if !a.equal(zero) {
			t.Fatal("a * 0 == 0")
		}
		_, _ = a.rand(rand.Reader)
		a0 := new(fe2).set(a)
		f.mulAssign(a, one)
		if !a.equal(a0) {
			t.Fatal("a * 1 == a")
		}
		_, _ = a.rand(rand.Reader)
		b, _ := new(fe2).rand(rand.Reader)
		a0.set(a)
		f.mulAssign(a, b)
		f.mulAssign(b, a0)
		if !a.equal(b) {
			t.Fatal("a * b == b * a")
		}
		c, _ := new(fe2).rand(rand.Reader)
		a0.set(a)
		f.mulAssign(a, b)
		f.mulAssign(a, c)
		f.mulAssign(a0, c)
		f.mulAssign(a0, b)
		if !a.equal(a0) {
			t.Fatal("(a * b) * c == (a * c) * b")
		}
		a0.set(a)
		f.squareAssign(a)
		f.mulAssign(a0, a0)
		if !a.equal(a0) {
			t.Fatal("a^2 == a*a")
		}
	}
}

func TestFp2Exponentiation(t *testing.T) {
	f := newFp2()
	for i := 0; i < fuz; i++ {
		a, _ := new(fe2).rand(rand.Reader)
		u := f.new()
		f.exp(u, a, big.NewInt(0))
		if !u.equal(f.one()) {
			t.Fatal("a^0 == 1")
		}
		f.exp(u, a, big.NewInt(1))
		if !u.equal(a) {
			t.Fatal("a^1 == a")
		}
		v := f.new()
		f.mul(u, a, a)
		f.mul(u, u, u)
		f.mul(u, u, u)
		f.exp(v, a, big.NewInt(8))
		if !u.equal(v) {
			t.Fatal("((a^2)^2)^2 == a^8")
		}
	}
}

func TestFp2Inversion(t *testing.T) {
	f := newFp2()
	u := f.new()
	zero := f.zero()
	one := f.one()
	f.inverse(u, zero)
	if !u.equal(zero) {
		t.Fatal("(0 ^ -1) == 0)")
	}
	f.inverse(u, one)
	if !u.equal(one) {
		t.Fatal("(1 ^ -1) == 1)")
	}
	for i := 0; i < fuz; i++ {
		a, _ := new(fe2).rand(rand.Reader)
		f.inverse(u, a)
		f.mul(u, u, a)
		if !u.equal(one) {
			t.Fatal("(r * a) * r * (a ^ -1) == r)")
		}
	}
}

func TestFp2BatchInversion(t *testing.T) {
	f := newFp2()
	n := 20
	for i := 0; i < n; i++ {
		e0 := make([]fe2, n)
		e1 := make([]fe2, n)
		for j := 0; j < n; j++ {
			if j != i {
				e, err := new(fe2).rand(rand.Reader)
				if err != nil {
					t.Fatal(err)
				}
				e0[j].set(e)
			}
			f.inverse(&e1[j], &e0[j])
		}
		f.inverseBatch(e0)
		for j := 0; j < n; j++ {
			if !e0[j].equal(&e1[j]) {
				t.Fatal("batch inversion failed")
			}
		}
	}
}

func TestFp2SquareRoot(t *testing.T) {
	e := newFp2()
	if e.sqrtBLST(e.new(), nonResidue2) {
		t.Fatal("non residue cannot have a sqrt")
	}
	for i := 0; i < fuz; i++ {
		a, _ := new(fe2).rand(rand.Reader)
		r0, r1 := new(fe2), new(fe2)
		d0 := e.sqrt(r0, a)
		d1 := e.sqrtBLST(r1, a)
		if d0 != d1 {
			t.Fatal("sqrt decision failed")
		}
		if d0 {
			e.square(r0, r0)
			e.square(r1, r1)
			if !r0.equal(r1) {
				t.Fatal("sqrt failed")
			}
			if !r0.equal(a) {
				t.Fatal("sqrt failed")
			}
		}
	}
}

func TestFp2NonResidue(t *testing.T) {
	f := newFp2()
	if !f.isQuadraticNonResidue(nonResidue2) {
		t.Fatal("element is quadratic non residue, 1")
	}
	if f.isQuadraticNonResidue(new(fe2).one()) {
		t.Fatal("one is not quadratic non residue")
	}
	if !f.isQuadraticNonResidue(new(fe2).zero()) {
		t.Fatal("should accept zero as quadratic non residue")
	}
	for i := 0; i < fuz; i++ {
		a, _ := new(fe2).rand(rand.Reader)
		f.squareAssign(a)
		if f.isQuadraticNonResidue(a) {
			t.Fatal("element is not quadratic non residue")
		}
	}
	for i := 0; i < fuz; i++ {
		a, _ := new(fe2).rand(rand.Reader)
		if !f.sqrt(new(fe2), a) {
			if !f.isQuadraticNonResidue(a) {
				t.Fatal("element is quadratic non residue, 2", i)
			}
		} else {
			i -= 1
		}
	}
}

func TestWFp2Addition(t *testing.T) {
	for i := 0; i < fuz; i++ {
		r0, _ := new(fe2).rand(rand.Reader)
		r1, _ := new(fe2).rand(rand.Reader)
		r2, _ := new(fe2).rand(rand.Reader)
		rw0, rw1, w0, w1 := new(wfe2), new(wfe2), new(wfe2), new(wfe2)

		wfp2Mul(w0, r0, r1)
		wfp2Mul(w1, r0, r2)

		_wfp2Add(rw0, w0, w1)
		wfp2Add(rw1, w0, w1)
		if !rw0.equal(rw1) {
			t.Fatal("add failed")
		}
		rw1.set(w0)
		wfp2AddAssign(rw1, w1)
		if !rw0.equal(rw1) {
			t.Fatal("assigned add failed")
		}

		_wfp2AddMixed(rw0, w0, w1)
		wfp2AddMixed(rw1, w0, w1)
		if !rw0.equal(rw1) {
			t.Fatal("add mixed failed")
		}
		rw1.set(w0)
		wfp2AddMixedAssign(rw1, w1)
		if !rw0.equal(rw1) {
			t.Fatal("assigned mixed add failed")
		}

		_wfp2Ladd(rw0, w0, w1)
		wfp2Ladd(rw1, w0, w1)
		if !rw0.equal(rw1) {
			t.Fatal("lazy add failed")
		}
		rw1.set(w0)
		wfp2LaddAssign(rw1, w1)
		if !rw0.equal(rw1) {
			t.Fatal("assigned lazy add failed")
		}

		_wfp2Sub(rw0, w0, w1)
		wfp2Sub(rw1, w0, w1)
		if !rw0.equal(rw1) {
			t.Fatal("sub failed")
		}
		rw1.set(w0)
		wfp2SubAssign(rw1, w1)
		if !rw0.equal(rw1) {
			t.Fatal("assigned sub failed")
		}

		_wfp2SubMixed(rw0, w0, w1)
		wfp2SubMixed(rw1, w0, w1)
		if !rw0.equal(rw1) {
			t.Fatal("sub mixed failed")
		}
		rw1.set(w0)
		wfp2SubMixedAssign(rw1, w1)
		if !rw0.equal(rw1) {
			t.Fatal("assigned sub mixed failed")
		}

		_wfp2Double(rw0, w0)
		wfp2Double(rw1, w0)
		if !rw0.equal(rw1) {
			t.Fatal("doubling failed")
		}
		rw1.set(w0)
		wfp2DoubleAssign(rw1)
		if !rw0.equal(rw1) {
			t.Fatal("assigned doubling failed")
		}

	}
}

func TestFp2MultiplicationCross(t *testing.T) {
	f := newFp2()
	a, b, c0, c1 := new(fe2), new(fe2), new(fe2), new(fe2)
	w0, w1 := new(wfe2), new(wfe2)
	for i := 0; i < fuz; i++ {
		_, _ = a.rand(rand.Reader)
		_, _ = b.rand(rand.Reader)
		_wfp2Mul(w0, a, b)
		wfp2Mul(w1, a, b)
		if !w0.equal(w1) {
			t.Fatal("multiplication failed")
		}
		c0.fromWide(w0)
		f.mul(c1, a, b)
		if !c0.equal(c1) {
			t.Fatal("multiplication failed")
		}
	}
}

func TestFp2SquareCross(t *testing.T) {
	f := newFp2()
	a, c0, c1 := new(fe2), new(fe2), new(fe2)
	w0, w1 := new(wfe2), new(wfe2)
	for i := 0; i < fuz; i++ {
		_, _ = a.rand(rand.Reader)
		_wfp2Square(w0, a)
		wfp2Square(w1, a)
		if !w0.equal(w1) {
			t.Fatal("squaring failed")
		}
		c0.fromWide(w0)
		f.square(c1, a)
		if !c0.equal(c1) {
			t.Fatal("squaring failed")
		}
	}
}

func TestWFp2MulByNonResidue(t *testing.T) {
	f := newFp2()
	a, b, c0, c1 := new(fe2), new(fe2), new(fe2), new(fe2)
	w0, w1, w2, w3 := new(wfe2), new(wfe2), new(wfe2), new(wfe2)
	for i := 0; i < fuz; i++ {
		_, _ = a.rand(rand.Reader)
		_, _ = b.rand(rand.Reader)
		r := new(fe2)

		f.mul(r, a, b)
		wfp2Mul(w0, a, b)

		mulByNonResidue(c0, r)
		wfp2MulByNonResidue(w1, w0)
		w2.set(w0)
		wfp2MulByNonResidueAssign(w2)
		_wfp2MulByNonResidue(w3, w0)
		if !w1.equal(w2) {
			t.Fatal("mul by non residue failed")
		}
		if !w1.equal(w3) {
			t.Fatal("mul by non residue failed")
		}
		c1.fromWide(w1)
		if !c0.equal(c1) {
			t.Fatal("mul by non residue failed")
		}
	}
}

func TestFp6Serialization(t *testing.T) {
	f := newFp6(nil)
	for i := 0; i < fuz; i++ {
		a, _ := new(fe6).rand(rand.Reader)
		b, err := f.fromBytes(f.toBytes(a))
		if err != nil {
			t.Fatal(err)
		}
		if !a.equal(b) {
			t.Fatal("serialization")
		}
	}
}

func TestFp6AdditionProperties(t *testing.T) {
	f := newFp6(nil)
	for i := 0; i < fuz; i++ {
		zero := f.zero()
		a, _ := new(fe6).rand(rand.Reader)
		b, _ := new(fe6).rand(rand.Reader)
		c1 := f.new()
		c2 := f.new()
		fp6Add(c1, a, zero)
		if !c1.equal(a) {
			t.Fatal("a + 0 == a")
		}
		fp6Sub(c1, a, zero)
		if !c1.equal(a) {
			t.Fatal("a - 0 == a")
		}
		fp6Double(c1, zero)
		if !c1.equal(zero) {
			t.Fatal("2 * 0 == 0")
		}
		fp6Neg(c1, zero)
		if !c1.equal(zero) {
			t.Fatal("-0 == 0")
		}
		fp6Sub(c1, zero, a)
		fp6Neg(c2, a)
		if !c1.equal(c2) {
			t.Fatal("0-a == -a")
		}
		fp6Double(c1, a)
		fp6Add(c2, a, a)
		if !c1.equal(c2) {
			t.Fatal("2 * a == a + a")
		}
		fp6Add(c1, a, b)
		fp6Add(c2, b, a)
		if !c1.equal(c2) {
			t.Fatal("a + b = b + a")
		}
		fp6Sub(c1, a, b)
		fp6Sub(c2, b, a)
		fp6Neg(c2, c2)
		if !c1.equal(c2) {
			t.Fatal("a - b = - ( b - a )")
		}
		cx, _ := new(fe6).rand(rand.Reader)
		fp6Add(c1, a, b)
		fp6Add(c1, c1, cx)
		fp6Add(c2, a, cx)
		fp6Add(c2, c2, b)
		if !c1.equal(c2) {
			t.Fatal("(a + b) + c == (a + c ) + b")
		}
		fp6Sub(c1, a, b)
		fp6Sub(c1, c1, cx)
		fp6Sub(c2, a, cx)
		fp6Sub(c2, c2, b)
		if !c1.equal(c2) {
			t.Fatal("(a - b) - c == (a - c ) -b")
		}
	}
}

func TestFp6AdditionPropertiesAssigned(t *testing.T) {
	for i := 0; i < fuz; i++ {
		zero := new(fe6).zero()
		a, b := new(fe6), new(fe6)
		_, _ = a.rand(rand.Reader)
		b.set(a)
		fp6AddAssign(a, zero)
		if !a.equal(b) {
			t.Fatal("a + 0 == a")
		}
		fp6SubAssign(a, zero)
		if !a.equal(b) {
			t.Fatal("a - 0 == a")
		}
		a.set(zero)
		fp6DoubleAssign(a)
		if !a.equal(zero) {
			t.Fatal("2 * 0 == 0")
		}
		a.set(zero)
		fp6SubAssign(a, b)
		fp6Neg(b, b)
		if !a.equal(b) {
			t.Fatal("0-a == -a")
		}
		_, _ = a.rand(rand.Reader)
		b.set(a)
		fp6DoubleAssign(a)
		fp6AddAssign(b, b)
		if !a.equal(b) {
			t.Fatal("2 * a == a + a")
		}
		_, _ = a.rand(rand.Reader)
		_, _ = b.rand(rand.Reader)
		c1, c2 := new(fe6).set(a), new(fe6).set(b)
		fp6AddAssign(c1, b)
		fp6AddAssign(c2, a)
		if !c1.equal(c2) {
			t.Fatal("a + b = b + a")
		}
		_, _ = a.rand(rand.Reader)
		_, _ = b.rand(rand.Reader)
		c1.set(a)
		c2.set(b)
		fp6SubAssign(c1, b)
		fp6SubAssign(c2, a)
		fp6Neg(c2, c2)
		if !c1.equal(c2) {
			t.Fatal("a - b = - ( b - a )")
		}
		_, _ = a.rand(rand.Reader)
		_, _ = b.rand(rand.Reader)
		c, _ := new(fe6).rand(rand.Reader)
		a0 := new(fe6).set(a)
		fp6AddAssign(a, b)
		fp6AddAssign(a, c)
		fp6AddAssign(b, c)
		fp6AddAssign(b, a0)
		if !a.equal(b) {
			t.Fatal("(a + b) + c == (b + c) + a")
		}
		_, _ = a.rand(rand.Reader)
		_, _ = b.rand(rand.Reader)
		_, _ = c.rand(rand.Reader)
		a0.set(a)
		fp6SubAssign(a, b)
		fp6SubAssign(a, c)
		fp6SubAssign(a0, c)
		fp6SubAssign(a0, b)
		if !a.equal(a0) {
			t.Fatal("(a - b) - c == (a - c) -b")
		}
	}
}

func TestFp6LazyOperations(t *testing.T) {
	f := newFp6(nil)
	for i := 0; i < fuz; i++ {
		a, _ := new(fe6).rand(rand.Reader)
		b, _ := new(fe6).rand(rand.Reader)
		c, _ := new(fe6).rand(rand.Reader)
		c0 := new(fe6)
		c1 := new(fe6)
		fp6Ladd(c0, a, b)
		fp6Add(c1, a, b)
		f.mulAssign(c0, c)
		f.mulAssign(c1, c)
		if !c0.equal(c1) {
			t.Fatal("(a + b) * c == (a l+ b) * c")
		}
		if !c0.equal(c1) {
			t.Fatal("(a + b) * c == (a l+ b) * c")
		}
	}
}

func TestFp6SparseMultiplication(t *testing.T) {
	fp6 := newFp6(nil)
	var a, b, u *fe6
	for i := 0; i < fuz; i++ {
		a, _ = new(fe6).rand(rand.Reader)
		b, _ = new(fe6).rand(rand.Reader)
		u, _ = new(fe6).rand(rand.Reader)
		b[2].zero()
		fp6.mul(u, a, b)
		fp6._mul01(a, a, &b[0], &b[1])
		if !a.equal(u) {
			t.Fatal("mul by 01")
		}
	}
	for i := 0; i < fuz; i++ {
		a, _ = new(fe6).rand(rand.Reader)
		b, _ = new(fe6).rand(rand.Reader)
		u, _ = new(fe6).rand(rand.Reader)
		b[2].zero()
		b[0].zero()
		fp6.mul(u, a, b)
		fp6._mul1(a, a, &b[1])
		if !a.equal(u) {
			t.Fatal("mul by 1")
		}
	}
}

func TestFp6MultiplicationProperties(t *testing.T) {
	f := newFp6(nil)
	for i := 0; i < fuz; i++ {
		a, _ := new(fe6).rand(rand.Reader)
		b, _ := new(fe6).rand(rand.Reader)
		zero := f.zero()
		one := f.one()
		c1, c2 := f.new(), f.new()
		f.mul(c1, a, zero)
		if !c1.equal(zero) {
			t.Fatal("a * 0 == 0")
		}
		f.mul(c1, a, one)
		if !c1.equal(a) {
			t.Fatal("a * 1 == a")
		}
		f.mul(c1, a, b)
		f.mul(c2, b, a)
		if !c1.equal(c2) {
			t.Fatal("a * b == b * a")
		}
		cx, _ := new(fe6).rand(rand.Reader)
		f.mul(c1, a, b)
		f.mul(c1, c1, cx)
		f.mul(c2, cx, b)
		f.mul(c2, c2, a)
		if !c1.equal(c2) {
			t.Fatal("(a * b) * c == (a * c) * b")
		}
		f.square(a, zero)
		if !a.equal(zero) {
			t.Fatal("0^2 == 0")
		}
		f.square(a, one)
		if !a.equal(one) {
			t.Fatal("1^2 == 1")
		}
		_, _ = a.rand(rand.Reader)
		f.square(c1, a)
		f.mul(c2, a, a)
		if !c2.equal(c1) {
			t.Fatal("a^2 == a*a")
		}
	}
}

func TestFp6MultiplicationPropertiesAssigned(t *testing.T) {
	f := newFp6(nil)
	for i := 0; i < fuz; i++ {
		a, _ := new(fe6).rand(rand.Reader)
		zero, one := new(fe6).zero(), new(fe6).one()
		f.mulAssign(a, zero)
		if !a.equal(zero) {
			t.Fatal("a * 0 == 0")
		}
		_, _ = a.rand(rand.Reader)
		a0 := new(fe6).set(a)
		f.mulAssign(a, one)
		if !a.equal(a0) {
			t.Fatal("a * 1 == a")
		}
		_, _ = a.rand(rand.Reader)
		b, _ := new(fe6).rand(rand.Reader)
		a0.set(a)
		f.mulAssign(a, b)
		f.mulAssign(b, a0)
		if !a.equal(b) {
			t.Fatal("a * b == b * a")
		}
		c, _ := new(fe6).rand(rand.Reader)
		a0.set(a)
		f.mulAssign(a, b)
		f.mulAssign(a, c)
		f.mulAssign(a0, c)
		f.mulAssign(a0, b)
		if !a.equal(a0) {
			t.Fatal("(a * b) * c == (a * c) * b")
		}
	}
}

func TestFp6Exponentiation(t *testing.T) {
	f := newFp6(nil)
	for i := 0; i < fuz; i++ {
		a, _ := new(fe6).rand(rand.Reader)
		u := f.new()
		f.exp(u, a, big.NewInt(0))
		if !u.equal(f.one()) {
			t.Fatal("a^0 == 1")
		}
		f.exp(u, a, big.NewInt(1))
		if !u.equal(a) {
			t.Fatal("a^1 == a")
		}
		v := f.new()
		f.exp(v, a, big.NewInt(8))
		f.square(u, a)
		f.square(u, u)
		f.square(u, u)
		if !u.equal(v) {
			t.Fatal("((a^2)^2)^2 == a^8", i)
		}
	}
}

func TestFp6Inversion(t *testing.T) {
	f := newFp6(nil)
	for i := 0; i < fuz; i++ {
		u := f.new()
		zero := f.zero()
		one := f.one()
		f.inverse(u, zero)
		if !u.equal(zero) {
			t.Fatal("(0^-1) == 0)")
		}
		f.inverse(u, one)
		if !u.equal(one) {
			t.Fatal("(1^-1) == 1)")
		}
		a, _ := new(fe6).rand(rand.Reader)
		f.inverse(u, a)
		f.mul(u, u, a)
		if !u.equal(one) {
			t.Fatal("(r*a) * r*(a^-1) == r)")
		}
	}
}

func TestFp6MultiplicationCross(t *testing.T) {
	f := newFp6(nil)
	a, b, c0, c1, c2, c3 := new(fe6), new(fe6), new(fe6), new(fe6), new(fe6), new(fe6)
	w0 := new(wfe6)
	for i := 0; i < fuz; i++ {

		_, _ = a.rand(rand.Reader)
		_, _ = b.rand(rand.Reader)
		f.wmul(w0, a, b)
		c0.fromWide(w0)
		f.mul(c1, a, b)
		f._mul(c2, a, b)
		c3.set(a)
		f.mulAssign(c3, b)
		if !c0.equal(c1) {
			t.Fatal("multiplication failed")
		}
		if !c0.equal(c2) {
			t.Fatal("multiplication failed")
		}
		if !c0.equal(c3) {
			t.Fatal("multiplication failed")
		}

	}
}

func TestFp6SquareCross(t *testing.T) {
	f := newFp6(nil)
	a, c0, c1, c2 := new(fe6), new(fe6), new(fe6), new(fe6)
	w0 := new(wfe6)
	for i := 0; i < fuz; i++ {
		_, _ = a.rand(rand.Reader)
		f.wsquare(w0, a)
		c0.fromWide(w0)
		f.square(c1, a)
		f._square(c2, a)

		if !c0.equal(c2) {
			t.Fatal("squaring failed")
		}
		if !c0.equal(c1) {
			t.Fatal("squaring failed")
		}
	}
}

func TestFp6SparseMultiplicationCross(t *testing.T) {
	f := newFp6(nil)
	a, c0, c1, c2 := new(fe6), new(fe6), new(fe6), new(fe6)
	w0 := new(wfe6)
	for i := 0; i < fuz; i++ {
		// mul01
		{
			_, _ = a.rand(rand.Reader)
			b0, _ := new(fe2).rand(rand.Reader)
			b1, _ := new(fe2).rand(rand.Reader)
			b := new(fe6)
			b[0].set(b0)
			b[1].set(b1)

			f.wmul01(w0, a, b0, b1)
			c0.fromWide(w0)

			f._mul01(c1, a, b0, b1)
			f._mul(c2, a, b)

			if !c2.equal(c1) {
				t.Fatal("sparse multiplication 01 failed")
			}

			if !c0.equal(c1) {
				t.Fatal("sparse multiplication 01 failed")
			}

		}
		// mul0
		{
			_, _ = a.rand(rand.Reader)
			b1, _ := new(fe2).rand(rand.Reader)
			b := new(fe6)
			b[1].set(b1)

			f.wmul1(w0, a, b1)
			c0.fromWide(w0)
			f._mul1(c1, a, b1)
			f._mul(c2, a, b)

			if !c2.equal(c0) {
				t.Fatal("sparse multiplication 0 failed")
			}
			if !c2.equal(c1) {
				t.Fatal("sparse multiplication 0 failed")
			}
		}
	}
}

func TestFp12Serialization(t *testing.T) {
	f := newFp12(nil)
	for i := 0; i < fuz; i++ {
		a, _ := new(fe12).rand(rand.Reader)
		b, err := f.fromBytes(f.toBytes(a))
		if err != nil {
			t.Fatal(err)
		}
		if !a.equal(b) {
			t.Fatal("serialization")
		}
	}
}

func TestFp12AdditionProperties(t *testing.T) {
	f := newFp12(nil)
	for i := 0; i < fuz; i++ {
		zero := f.zero()
		a, _ := new(fe12).rand(rand.Reader)
		b, _ := new(fe12).rand(rand.Reader)
		c1 := f.new()
		c2 := f.new()
		fp12Add(c1, a, zero)
		if !c1.equal(a) {
			t.Fatal("a + 0 == a")
		}
		fp12Sub(c1, a, zero)
		if !c1.equal(a) {
			t.Fatal("a - 0 == a")
		}
		fp12Double(c1, zero)
		if !c1.equal(zero) {
			t.Fatal("2 * 0 == 0")
		}
		fp12Neg(c1, zero)
		if !c1.equal(zero) {
			t.Fatal("-0 == 0")
		}
		fp12Sub(c1, zero, a)
		fp12Neg(c2, a)
		if !c1.equal(c2) {
			t.Fatal("0-a == -a")
		}
		fp12Double(c1, a)
		fp12Add(c2, a, a)
		if !c1.equal(c2) {
			t.Fatal("2 * a == a + a")
		}
		fp12Add(c1, a, b)
		fp12Add(c2, b, a)
		if !c1.equal(c2) {
			t.Fatal("a + b = b + a")
		}
		fp12Sub(c1, a, b)
		fp12Sub(c2, b, a)
		fp12Neg(c2, c2)
		if !c1.equal(c2) {
			t.Fatal("a - b = - ( b - a )")
		}
		cx, _ := new(fe12).rand(rand.Reader)
		fp12Add(c1, a, b)
		fp12Add(c1, c1, cx)
		fp12Add(c2, a, cx)
		fp12Add(c2, c2, b)
		if !c1.equal(c2) {
			t.Fatal("(a + b) + c == (a + c ) + b")
		}
		fp12Sub(c1, a, b)
		fp12Sub(c1, c1, cx)
		fp12Sub(c2, a, cx)
		fp12Sub(c2, c2, b)
		if !c1.equal(c2) {
			t.Fatal("(a - b) - c == (a - c ) -b")
		}
	}
}

func TestFp12MultiplicationProperties(t *testing.T) {
	f := newFp12(nil)
	for i := 0; i < fuz; i++ {
		a, _ := new(fe12).rand(rand.Reader)
		b, _ := new(fe12).rand(rand.Reader)
		zero := f.zero()
		one := f.one()
		c1, c2 := f.new(), f.new()
		f.mul(c1, a, zero)
		if !c1.equal(zero) {
			t.Fatal("a * 0 == 0")
		}
		f.mul(c1, a, one)
		if !c1.equal(a) {
			t.Fatal("a * 1 == a")
		}
		f.mul(c1, a, b)
		f.mul(c2, b, a)
		if !c1.equal(c2) {
			t.Fatal("a * b == b * a")
		}
		cx, _ := new(fe12).rand(rand.Reader)
		f.mul(c1, a, b)
		f.mul(c1, c1, cx)
		f.mul(c2, cx, b)
		f.mul(c2, c2, a)
		if !c1.equal(c2) {
			t.Fatal("(a * b) * c == (a * c) * b")
		}
		f.square(a, zero)
		if !a.equal(zero) {
			t.Fatal("0^2 == 0")
		}
		f.square(a, one)
		if !a.equal(one) {
			t.Fatal("1^2 == 1")
		}
		_, _ = a.rand(rand.Reader)
		f.square(c1, a)
		f.mul(c2, a, a)
		if !c2.equal(c1) {
			t.Fatal("a^2 == a*a")
		}
	}
}

func TestFp12MultiplicationPropertiesAssigned(t *testing.T) {
	f := newFp12(nil)
	zero, one := new(fe12).zero(), new(fe12).one()
	for i := 0; i < fuz; i++ {
		a, _ := new(fe12).rand(rand.Reader)
		f.mulAssign(a, zero)
		if !a.equal(zero) {
			t.Fatal("a * 0 == 0")
		}
		_, _ = a.rand(rand.Reader)
		a0 := new(fe12).set(a)
		f.mulAssign(a, one)
		if !a.equal(a0) {
			t.Fatal("a * 1 == a")
		}
		_, _ = a.rand(rand.Reader)
		b, _ := new(fe12).rand(rand.Reader)
		a0.set(a)
		f.mulAssign(a, b)
		f.mulAssign(b, a0)
		if !a.equal(b) {
			t.Fatal("a * b == b * a")
		}
		c, _ := new(fe12).rand(rand.Reader)
		a0.set(a)
		f.mul(a, a, b)
		f.mul(a, a, c)
		f.mul(a0, a0, c)
		f.mul(a0, a0, b)
		if !a.equal(a0) {
			t.Fatal("(a * b) * c == (a * c) * b")
		}
	}
}

func TestFp12SparseMultiplication(t *testing.T) {
	fp12 := newFp12(nil)
	var a, b, u *fe12
	for j := 0; j < fuz; j++ {
		a, _ = new(fe12).rand(rand.Reader)
		b, _ = new(fe12).rand(rand.Reader)
		u, _ = new(fe12).rand(rand.Reader)
		b[0][2].zero()
		b[1][0].zero()
		b[1][2].zero()
		fp12.mul(u, a, b)
		fp12.mul014(a, &b[0][0], &b[0][1], &b[1][1])
		if !a.equal(u) {
			t.Fatal("mul by 01")
		}
	}
}

func TestFp12Exponentiation(t *testing.T) {
	f := newFp12(nil)
	for i := 0; i < fuz; i++ {
		a, _ := new(fe12).rand(rand.Reader)
		u := f.new()
		f.exp(u, a, big.NewInt(0))
		if !u.equal(f.one()) {
			t.Fatal("a^0 == 1")
		}
		f.exp(u, a, big.NewInt(1))
		if !u.equal(a) {
			t.Fatal("a^1 == a")
		}
		v := f.new()
		f.mul(u, a, a)
		f.mul(u, u, u)
		f.mul(u, u, u)
		f.exp(v, a, big.NewInt(8))
		if !u.equal(v) {
			t.Fatal("((a^2)^2)^2 == a^8")
		}
	}
}

func TestFp12Inversion(t *testing.T) {
	f := newFp12(nil)
	for i := 0; i < fuz; i++ {
		u := f.new()
		zero := f.zero()
		one := f.one()
		f.inverse(u, zero)
		if !u.equal(zero) {
			t.Fatal("(0^-1) == 0)")
		}
		f.inverse(u, one)
		if !u.equal(one) {
			t.Fatal("(1^-1) == 1)")
		}
		a, _ := new(fe12).rand(rand.Reader)
		f.inverse(u, a)
		f.mul(u, u, a)
		if !u.equal(one) {
			t.Fatal("(r*a) * r*(a^-1) == r)")
		}
	}
}

func TestFrobeniusMapping2(t *testing.T) {
	f := newFp2()
	a, _ := new(fe2).rand(rand.Reader)
	b0, b1, b2, b3 := new(fe2), new(fe2), new(fe2), new(fe2)
	f.exp(b0, a, modulus.big())
	fp2Conjugate(b1, a)
	b2.set(a)
	f.frobeniusMap1(b2)
	b3.set(a)
	f.frobeniusMap(b3, 1)
	if !b0.equal(b3) {
		t.Fatal("frobenius map failed")
	}
	if !b1.equal(b3) {
		t.Fatal("frobenius map failed")
	}
	if !b2.equal(b3) {
		t.Fatal("frobenius map failed")
	}
}

func TestFrobeniusMapping6(t *testing.T) {
	{
		f := newFp2()
		z := nonResidue2
		for i := 0; i < 6; i++ {
			p, r, e := modulus.big(), new(fe2), big.NewInt(0)
			// p ^ i
			p.Exp(p, big.NewInt(int64(i)), nil)
			// (p ^ i - 1) / 3
			e.Sub(p, big.NewInt(1)).Div(e, big.NewInt(3))
			// r = z ^ (p ^ i - 1) / 3
			f.exp(r, z, e)
			if !r.equal(&frobeniusCoeffs61[i]) {
				t.Fatalf("bad frobenius fp6 1q coefficient")
			}
		}
		for i := 0; i < 6; i++ {
			p, r, e := modulus.big(), new(fe2), big.NewInt(0)
			// p ^ i
			p.Exp(p, big.NewInt(int64(i)), nil).Mul(p, big.NewInt(2))
			// (2 * p ^ i - 2) / 3
			e.Sub(p, big.NewInt(2)).Div(e, big.NewInt(3))
			// r = z ^ (2 * p ^ i - 2) / 3
			f.exp(r, z, e)
			if !r.equal(&frobeniusCoeffs62[i]) {
				t.Fatalf("bad frobenius fp6 2q coefficient")
			}
		}
	}
	f := newFp6(nil)
	r0, r1 := f.new(), f.new()
	e, _ := new(fe6).rand(rand.Reader)
	r0.set(e)
	r1.set(e)
	f.frobeniusMap(r1, 1)
	f.frobeniusMap1(r0)
	if !r0.equal(r1) {
		t.Fatalf("frobenius mapping by 1 failed")
	}
	r0.set(e)
	r1.set(e)
	f.frobeniusMap(r1, 2)
	f.frobeniusMap2(r0)
	if !r0.equal(r1) {
		t.Fatalf("frobenius mapping by 2 failed")
	}
	r0.set(e)
	r1.set(e)
	f.frobeniusMap(r1, 3)
	f.frobeniusMap3(r0)
	if !r0.equal(r1) {
		t.Fatalf("frobenius mapping by 3 failed")
	}
}

func TestFrobeniusMapping12(t *testing.T) {
	{
		f := newFp2()
		z := nonResidue2
		for i := 0; i < 12; i++ {
			p, r, e := modulus.big(), new(fe2), big.NewInt(0)
			// p ^ i
			p.Exp(p, big.NewInt(int64(i)), nil)
			// (p ^ i - 1) / 6
			e.Sub(p, big.NewInt(1)).Div(e, big.NewInt(6))
			// r = z ^ (p ^ i - 1) / 6
			f.exp(r, z, e)
			if !r.equal(&frobeniusCoeffs12[i]) {
				t.Fatalf("bad frobenius fp12 coefficient")
			}
		}
	}
	f := newFp12(nil)
	r0, r1 := f.new(), f.new()
	e, _ := new(fe12).rand(rand.Reader)
	p := modulus.big()
	f.exp(r0, e, p)
	r1.set(e)
	f.frobeniusMap1(r1)
	if !r0.equal(r1) {
		t.Fatalf("frobenius mapping by 1 failed")
	}
	p.Mul(p, modulus.big())
	f.exp(r0, e, p)
	r1.set(e)
	f.frobeniusMap2(r1)
	if !r0.equal(r1) {
		t.Fatalf("frobenius mapping by 2 failed")
	}
	p.Mul(p, modulus.big())
	f.exp(r0, e, p)
	r1.set(e)
	f.frobeniusMap3(r1)
	if !r0.equal(r1) {
		t.Fatalf("frobenius mapping by 2 failed")
	}
}

func TestFp12MultiplicationCross(t *testing.T) {
	f := newFp12(nil)
	a, b, c0, c1, c2 := new(fe12), new(fe12), new(fe12), new(fe12), new(fe12)
	for i := 0; i < fuz; i++ {
		_, _ = a.rand(rand.Reader)
		_, _ = b.rand(rand.Reader)
		f.mul(c0, a, b)
		c1.set(a)
		f.mulAssign(c1, b)
		f._mul(c2, a, b)

		if !c0.equal(c1) {
			t.Fatal("multiplication failed")
		}
		if !c0.equal(c2) {
			t.Fatal("multiplication failed")
		}
	}
}

func TestFp12SparseMultiplicationCross(t *testing.T) {
	f := newFp12(nil)
	a, c0, c1 := new(fe12), new(fe12), new(fe12)

	for i := 0; i < fuz; i++ {
		_, _ = a.rand(rand.Reader)
		b0, _ := new(fe2).rand(rand.Reader)
		b1, _ := new(fe2).rand(rand.Reader)
		b4, _ := new(fe2).rand(rand.Reader)
		b := new(fe12)
		b[0][0].set(b0)
		b[0][1].set(b1)
		b[1][1].set(b4)

		c0.set(a)
		f.mul014(c0, b0, b1, b4)
		f._mul(c1, a, b)

		if !c0.equal(c1) {
			t.Fatal("sparse multiplication 014 failed")
		}
	}
}

func TestFp4MultiplicationCross(t *testing.T) {
	f := newFp12(nil)
	a0, a1, b0, b1 := new(fe2), new(fe2), new(fe2), new(fe2)
	c0, c1 := new(fe2), new(fe2)

	for i := 0; i < fuz; i++ {
		_, _ = a0.rand(rand.Reader)
		_, _ = a1.rand(rand.Reader)
		_, _ = b0.rand(rand.Reader)
		_, _ = b1.rand(rand.Reader)
		c0.set(a0)
		c1.set(a1)

		f._fp4Square(a0, a1, b0, b1)
		f.fp4Square(c0, c1, b0, b1)

		if !a0.equal(c0) {
			t.Fatal("fp4 multiplication failed")
		}
		if !a1.equal(c1) {
			t.Fatal("fp4 multiplication failed")
		}
	}
}

func BenchmarkFpMul(t *testing.B) {
	a, _ := new(fe).rand(rand.Reader)
	b, _ := new(fe).rand(rand.Reader)
	c := new(fe)
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		mul(c, a, b)
	}
}

func (fe *wfe) bytes() []byte {
	out := make([]byte, fpByteSize*2)
	var a int
	for i := 0; i < 2*fpNumberOfLimbs; i++ {
		a = fpByteSize*2 - i*8
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

func (fe *wfe) equal(fe2 *wfe) bool {
	return fe2[0] == fe[0] && fe2[1] == fe[1] && fe2[2] == fe[2] && fe2[3] == fe[3] && fe2[4] == fe[4] && fe2[5] == fe[5] && fe2[6] == fe[6] && fe2[7] == fe[7] && fe2[8] == fe[8] && fe2[9] == fe[9] && fe2[10] == fe[10] && fe2[11] == fe[11]
}

func (fe *wfe2) equal(fe2 *wfe2) bool {
	return fe[0].equal(&fe2[0]) && fe[1].equal(&fe2[1])
}

func _fp2MulByNonResidue(c, a *fe2) {
	t0 := &fe{}
	add(t0, &a[0], &a[1])
	sub(&c[0], &a[0], &a[1])
	c[1].set(t0)
}

func _wfp2Add(c, a, b *wfe2) {
	wadd(&c[0], &a[0], &b[0])
	wadd(&c[1], &a[1], &b[1])
}

func _wfp2Ladd(c, a, b *wfe2) {
	lwadd(&c[0], &a[0], &b[0])
	lwadd(&c[1], &a[1], &b[1])
}

func _wfp2AddMixed(c, a, b *wfe2) {
	wadd(&c[0], &a[0], &b[0])
	lwadd(&c[1], &a[1], &b[1])
}

func _wfp2Sub(c, a, b *wfe2) {
	wsub(&c[0], &a[0], &b[0])
	wsub(&c[1], &a[1], &b[1])
}

func _wfp2SubMixed(c, a, b *wfe2) {
	wsub(&c[0], &a[0], &b[0])
	lwsub(&c[1], &a[1], &b[1])
}

func _wfp2Double(c, a *wfe2) {
	wdouble(&c[0], &a[0])
	wdouble(&c[1], &a[1])
}

func _wfp2MulByNonResidue(c, a *wfe2) {
	wt0 := &wfe{}
	wadd(wt0, &a[0], &a[1])
	wsub(&c[0], &a[0], &a[1])
	c[1].set(wt0)
}

func _wfp2Mul(c *wfe2, a, b *fe2) {
	wt0, wt1 := new(wfe), new(wfe)
	t0, t1 := new(fe), new(fe)
	wmul(wt0, &a[0], &b[0]) // a0b0
	wmul(wt1, &a[1], &b[1]) // a1b1
	wsub(&c[0], wt0, wt1)   // c0 = a0b0 - a1b1
	lwaddAssign(wt0, wt1)   // a0b0 + a1b1
	ladd(t0, &a[0], &a[1])  // a0 + a1
	ladd(t1, &b[0], &b[1])  // b0 + b1
	wmul(wt1, t0, t1)       // (a0 + a1)(b0 + b1)
	lwsub(&c[1], wt1, wt0)  // c1 = (a0 + a1)(b0 + b1) - (a0b0 + a1b1)
}

func _wfp2Square(c *wfe2, a *fe2) {
	t0, t1, t2 := new(fe), new(fe), new(fe)
	ladd(t0, &a[0], &a[1]) // (a0 + a1)
	sub(t1, &a[0], &a[1])  // (a0 - a1)
	ldouble(t2, &a[0])     // 2a0
	wmul(&c[0], t1, t0)    // c0 = (a0 + a1)(a0 - a1)
	wmul(&c[1], t2, &a[1]) // c1 = 2a0a1
}

func (e *fp6) _mul(c, a, b *fe6) {
	t0, t1, t2, t3, t4, t5 := new(fe2), new(fe2), new(fe2), new(fe2), new(fe2), new(fe2)
	e.fp2.mul(t0, &a[0], &b[0]) // v0 = a0b0
	e.fp2.mul(t1, &a[1], &b[1]) // v1 = a1b1
	e.fp2.mul(t2, &a[2], &b[2]) // v2 = a2b2
	fp2Add(t3, &a[1], &a[2])    // a1 + a2
	fp2Add(t4, &b[1], &b[2])    // b1 + b2
	e.fp2.mulAssign(t3, t4)     // (a1 + a2)(b1 + b2)
	fp2Add(t4, t1, t2)          // v1 + v2
	fp2SubAssign(t3, t4)        // (a1 + a2)(b1 + b2) - v1 - v2
	mulByNonResidueAssign(t3)   // ((a1 + a2)(b1 + b2) - v1 - v2)β
	fp2AddAssign(t3, t0)        // c0 = ((a1 + a2)(b1 + b2) - v1 - v2)β + v0
	fp2Add(t5, &a[0], &a[1])    // a0 + a1
	fp2Add(t4, &b[0], &b[1])    // b0 + b1
	e.fp2.mulAssign(t5, t4)     // (a0 + a1)(b0 + b1)
	fp2Add(t4, t0, t1)          // v0 + v1
	fp2SubAssign(t5, t4)        // (a0 + a1)(b0 + b1) - v0 - v1
	mulByNonResidue(t4, t2)     // βv2
	fp2Add(&c[1], t5, t4)       // c1 = (a0 + a1)(b0 + b1) - v0 - v1 + βv2
	fp2Add(t5, &a[0], &a[2])    // a0 + a2
	fp2Add(t4, &b[0], &b[2])    // b0 + b2
	e.fp2.mulAssign(t5, t4)     // (a0 + a2)(b0 + b2)
	fp2Add(t4, t0, t2)          // v0 + v2
	fp2SubAssign(t5, t4)        // (a0 + a2)(b0 + b2) - v0 - v2
	fp2Add(&c[2], t1, t5)       // c2 = (a0 + a2)(b0 + b2) - v0 - v2 + v1
	c[0].set(t3)
}

func (e *fp6) _mul01(c, a *fe6, b0, b1 *fe2) {
	t0, t1, t2, t3, t4 := new(fe2), new(fe2), new(fe2), new(fe2), new(fe2)
	e.fp2.mul(t0, &a[0], b0)  // v0 = b0a0
	e.fp2.mul(t1, &a[1], b1)  // v1 = a1b1
	fp2Add(t2, &a[1], &a[2])  // a1 + a2
	e.fp2.mulAssign(t2, b1)   // b1(a1 + a2)
	fp2SubAssign(t2, t1)      // b1(a1 + a2) - v1
	mulByNonResidueAssign(t2) // (b1(a1 + a2) - v1)β
	fp2Add(t3, &a[0], &a[2])  // a0 + a2
	e.fp2.mulAssign(t3, b0)   // b0(a0 + a2)
	fp2SubAssign(t3, t0)      // b0(a0 + a2) - v0
	fp2Add(&c[2], t3, t1)     // b0(a0 + a2) - v0 + v1
	fp2Add(t4, b0, b1)        // (b0 + b1)
	fp2Add(t3, &a[0], &a[1])  // (a0 + a1)
	e.fp2.mulAssign(t4, t3)   // (a0 + a1)(b0 + b1)
	fp2SubAssign(t4, t0)      // (a0 + a1)(b0 + b1) - v0
	fp2Sub(&c[1], t4, t1)     // (a0 + a1)(b0 + b1) - v0 - v1
	fp2Add(&c[0], t2, t0)     //  (b1(a1 + a2) - v1)β + v0
}

func (e *fp6) _mul1(c, a *fe6, b1 *fe2) {
	t := new(fe2)
	e.fp2.mul(t, &a[2], b1)
	e.fp2.mul(&c[2], &a[1], b1)
	e.fp2.mul(&c[1], &a[0], b1)
	mulByNonResidue(&c[0], t)
}

func (e *fp6) _square(c, a *fe6) {
	t0, t1, t2, t3, t4, t5 := new(fe2), new(fe2), new(fe2), new(fe2), new(fe2), new(fe2)
	e.fp2.square(t0, &a[0])
	e.fp2.mul(t1, &a[0], &a[1])
	fp2DoubleAssign(t1)
	fp2Sub(t2, &a[0], &a[1])
	fp2AddAssign(t2, &a[2])
	e.fp2.squareAssign(t2)
	e.fp2.mul(t3, &a[1], &a[2])
	fp2DoubleAssign(t3)
	e.fp2.square(t4, &a[2])
	mulByNonResidue(t5, t3)
	fp2Add(&c[0], t0, t5)
	mulByNonResidue(t5, t4)
	fp2Add(&c[1], t1, t5)
	fp2AddAssign(t1, t2)
	fp2AddAssign(t1, t3)
	fp2AddAssign(t0, t4)
	fp2Sub(&c[2], t1, t0)
}

func (e *fp12) _mul(c, a, b *fe12) {
	t0, t1, t2, t3 := new(fe6), new(fe6), new(fe6), new(fe6)
	e.fp6.mul(t1, &a[0], &b[0])   // v0 = a0b0
	e.fp6.mul(t2, &a[1], &b[1])   // v1 = a1b1
	fp6Add(t0, &a[0], &a[1])      // a0 + a1
	fp6Add(t3, &b[0], &b[1])      // b0 + b1
	e.fp6.mulAssign(t0, t3)       // (a0 + a1)(b0 + b1)
	fp6SubAssign(t0, t1)          // (a0 + a1)(b0 + b1) - v0
	fp6Sub(&c[1], t0, t2)         // c1 = (a0 + a1)(b0 + b1) - v0 - v1
	e.fp6.mulByNonResidue(t2, t2) // βv1
	fp6Add(&c[0], t1, t2)         // c0 = v0 + βv1
}

func (e *fp12) _fp4Square(c0, c1, a0, a1 *fe2) {
	t, fp2 := e.t2, e.fp2()

	fp2.square(t[0], a0)        // a0^2
	fp2.square(t[1], a1)        // a1^2
	mulByNonResidue(t[2], t[1]) // βa1^2
	fp2Add(c0, t[2], t[0])      // c0 = βa1^2 + a0^2
	fp2Add(t[2], a0, a1)        // a0 + a1
	fp2.squareAssign(t[2])      // (a0 + a1)^2
	fp2SubAssign(t[2], t[0])    // (a0 + a1)^2 - a0^2
	fp2Sub(c1, t[2], t[1])      // (a0 + a1)^2 - a0^2 - a1^2
}
