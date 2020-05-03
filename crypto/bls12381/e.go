package bls12381

import (
	bls "github.com/kilic/bls12-381"
)

type PairingEngine = bls.Engine

func NewPairingEngine() *PairingEngine {
	return bls.NewEngine()
}
