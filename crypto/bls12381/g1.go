package bls12381

import (
	bls "github.com/kilic/bls12-381"
)

type PointG1 = bls.PointG1
type G1 = bls.G1

func NewG1() *G1 {
	return bls.NewG1()
}
