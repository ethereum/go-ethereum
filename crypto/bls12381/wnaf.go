package bls12381

import (
	"math/big"
)

type nafNumber []int

func (n nafNumber) neg() {
	for i := 0; i < len(n); i++ {
		n[i] = -n[i]
	}
}

var bigZero = big.NewInt(0)
var bigOne = big.NewInt(1)

func (e *Fr) toWNAF(w uint) nafNumber {
	naf := nafNumber{}
	if w == 0 {
		return naf
	}
	windowSize, halfSize, mask := 1<<(w+1), 1<<w, (1<<(w+1))-1
	ee := new(Fr).Set(e)
	z := new(Fr)
	for !ee.IsZero() {
		if !ee.isEven() {
			nafSign := int(ee[0]) & mask
			if nafSign >= halfSize {
				nafSign = nafSign - windowSize
			}
			naf = append(naf, int(nafSign))
			if nafSign < 0 {
				laddAssignFR(ee, z.setUint64(uint64(-nafSign)))
			} else {
				lsubAssignFR(ee, z.setUint64(uint64(nafSign)))
			}
		} else {
			naf = append(naf, 0)
		}
		ee.div2()
	}

	return naf
}

func (e *Fr) fromWNAF(naf nafNumber, w uint) *Fr {
	if w == 0 {
		return e
	}
	l := (1 << (w - 1))
	table := make([]*Fr, l)
	table[0] = new(Fr).One()
	two := new(Fr).setUint64(2)
	for i := 1; i < l; i++ {
		table[i] = new(Fr)
		table[i].Add(table[i-1], two)
	}
	acc := new(Fr).Zero()
	for i := len(naf) - 1; i >= 0; i-- {
		if naf[i] < 0 {
			acc.Sub(acc, table[-naf[i]>>1])
		} else if naf[i] > 0 {
			acc.Add(acc, table[naf[i]>>1])
		}
		if i != 0 {
			acc.Double(acc)
		}
	}
	return e.Set(acc)
}

// caution: does not cover negative case
func bigToWNAF(e *big.Int, w uint) nafNumber {
	naf := nafNumber{}
	if w == 0 {
		return naf
	}
	windowSize := new(big.Int).Lsh(bigOne, uint(w+1))
	halfSize := new(big.Int).Rsh(windowSize, 1)
	ee := new(big.Int).Abs(e)
	for ee.Cmp(bigZero) != 0 {
		if ee.Bit(0) == 1 {
			nafSign := new(big.Int)
			nafSign.Mod(ee, windowSize)
			if nafSign.Cmp(halfSize) >= 0 {
				nafSign.Sub(nafSign, windowSize)
			}
			naf = append(naf, int(nafSign.Int64()))
			ee.Sub(ee, nafSign)
		} else {
			naf = append(naf, 0)
		}
		ee.Rsh(ee, 1)
	}
	return naf
}

func bigFromWNAF(naf nafNumber) *big.Int {
	acc := new(big.Int)
	k := new(big.Int).Set(bigOne)
	for i := 0; i < len(naf); i++ {
		if naf[i] != 0 {
			z := new(big.Int).Mul(k, big.NewInt(int64(naf[i])))
			acc.Add(acc, z)
		}
		k.Lsh(k, 1)
	}
	return acc
}
