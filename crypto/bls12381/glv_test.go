package bls12381

import (
	"crypto/rand"
	"math/big"
	"testing"
)

func TestGLVConstruction(t *testing.T) {
	t.Run("Parameters", func(t *testing.T) {
		t0, t1 := new(Fr), new(Fr)
		one := new(Fr).setUint64(1)
		t0.Square(glvLambda)
		t0.Add(t0, glvLambda)
		t1.Sub(&q, one)
		if !t0.Equal(t1) {
			t.Fatal("lambda1^2 + lambda1 + 1 = 0")
		}
		c0 := new(fe)
		square(c0, glvPhi1)
		mul(c0, c0, glvPhi1)
		if !c0.isOne() {
			t.Fatal("phi1^3 = 1")
		}
		square(c0, glvPhi2)
		mul(c0, c0, glvPhi2)
		if !c0.isOne() {
			t.Fatal("phi2^3 = 1")
		}
	})
	t.Run("Endomorphism G1", func(t *testing.T) {
		g := NewG1()
		{
			p0, p1 := g.randAffine(), g.New()
			g.MulScalar(p1, p0, glvLambda)
			g.Affine(p1)
			r := g.New()
			g.glvEndomorphism(r, p0)
			if !g.Equal(r, p1) {
				t.Fatal("f(x, y) = (phi * x, y)")
			}
		}
	})
	t.Run("Endomorphism G2", func(t *testing.T) {
		g := NewG2()
		{
			p0, p1 := g.randAffine(), g.New()
			g.MulScalar(p1, p0, glvLambda)
			g.Affine(p1)
			r := g.New()
			g.glvEndomorphism(r, p0)
			if !g.Equal(r, p1) {
				t.Fatal("f(x, y) = (phi * x, y)")
			}
		}
	})
	t.Run("Scalar Decomposition", func(t *testing.T) {
		for i := 0; i < fuz; i++ {
			m, err := new(Fr).Rand(rand.Reader)
			if err != nil {
				t.Fatal(err)
			}
			mBig := m.ToBig()
			var vFr *glvVectorFr
			var vBig *glvVectorBig
			{
				vFr = new(glvVectorFr).new(m)
				v := vFr

				if v.k1.Cmp(r128) >= 0 {
					t.Fatal("bad scalar component, k1")
				}
				if v.k2.Cmp(r128) >= 0 {
					t.Fatal("bad scalar component, k2")
				}

				k := new(Fr)
				if v.neg1 && v.neg2 {
					k.Mul(glvLambda, v.k2)
					k.Sub(k, v.k1)
				} else if v.neg1 {
					k.Mul(glvLambda, v.k2)
					k.Add(k, v.k1)
					k.Neg(k)
				} else if v.neg2 {
					k.Mul(glvLambda, v.k2)
					k.Add(v.k1, k)
				} else {
					k.Mul(glvLambda, v.k2)
					k.Sub(v.k1, k)
				}

				if !k.Equal(m) {
					t.Fatal("scalar decomposing failed")
				}
			}

			r128Big := r128.ToBig()
			{
				vBig = new(glvVectorBig).new(mBig)

				if new(big.Int).Abs(vBig.k1).Cmp(r128Big) >= 0 {
					t.Fatal("bad scalar component, big k1")
				}
				if new(big.Int).Abs(vBig.k2).Cmp(r128Big) >= 0 {
					t.Fatal("bad scalar component, big k2")
				}

				k := new(big.Int)
				k.Mul(glvLambdaBig, vBig.k2)
				k.Sub(vBig.k1, k).Mod(k, qBig)
				if k.Cmp(mBig) != 0 {
					t.Fatal("scalar decomposing with big.Int failed", i)
				}
			}

			zeroBig := new(big.Int)
			k1Abs, k2Abs := new(big.Int).Abs(vBig.k1), new(big.Int).Abs(vBig.k2)

			if vFr.neg1 != (vBig.k1.Cmp(zeroBig) == -1) {
				t.Fatal("cross: scalar decomposing with failed neg1")
			}
			if vFr.neg2 != (vBig.k2.Cmp(zeroBig) == -1) {
				t.Fatal("cross: scalar decomposing with failed neg2")
			}
			if k1Abs.Cmp(vFr.k1.ToBig()) != 0 {
				t.Fatal("cross: scalar decomposing with failed k1", i)
			}
			if k2Abs.Cmp(vFr.k2.ToBig()) != 0 {
				t.Fatal("cross: scalar decomposing with failed k2", i)
			}
		}
	})
}
