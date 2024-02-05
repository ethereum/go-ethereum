package bls12381

import (
	"crypto/rand"
	"math/big"
	"testing"
)

var maxWindowSize uint = 9

func TestWNAFBig(t *testing.T) {
	var w uint
	for w = 1; w <= maxWindowSize; w++ {
		for i := 0; i < fuz; i++ {
			e0, err := rand.Int(rand.Reader, new(big.Int).SetUint64(100))
			if err != nil {
				t.Fatal(err)
			}
			n0 := bigToWNAF(e0, w)
			e1 := bigFromWNAF(n0)
			if e0.Cmp(e1) != 0 {
				t.Fatal("wnaf conversion failed")
			}
		}
	}
}

func TestFrWNAF(t *testing.T) {
	var w uint
	for w = 1; w <= maxWindowSize; w++ {
		for i := 0; i < fuz; i++ {
			a0, _ := new(Fr).Rand(rand.Reader)
			naf := a0.toWNAF(w)
			a1 := new(Fr).fromWNAF(naf, w)
			if !a0.Equal(a1) {
				t.Fatal("wnaf conversion failed")
			}
			naf.neg()
			a1.fromWNAF(naf, w)
			a0.Neg(a0)
			if !a0.Equal(a1) {
				t.Fatal("negated wnaf conversion failed")
			}
		}
	}
}

func TestFrWNAFCrossAgainstBig(t *testing.T) {
	var maxWindowSize uint = 20
	var w uint
	for w = 1; w <= maxWindowSize; w++ {
		for i := 0; i < fuz; i++ {
			a, _ := new(Fr).Rand(rand.Reader)
			aBig := a.ToBig()
			naf1 := a.toWNAF(w)
			naf2 := bigToWNAF(aBig, w)
			if len(naf1) != len(naf2) {
				t.Fatal("naf conversion failed", len(naf1), len(naf2))
			}
			for i := 0; i < len(naf1); i++ {
				if naf1[i] != naf2[i] {
					t.Fatal("naf conversion failed", i)
				}
			}
		}
	}
}
