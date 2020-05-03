package bls12381

import (
	bls "github.com/kilic/bls12-381"
)

type PointG2 = bls.PointG2
type G2 = bls.G2

func NewG2() *bls.G2 {
	return bls.NewG2()
}
