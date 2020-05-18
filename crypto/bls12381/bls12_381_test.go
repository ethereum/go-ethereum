package bls12381

import (
	"crypto/rand"
	"math/big"
)

var fuz int = 1000

func randScalar(max *big.Int) *big.Int {
	a, _ := rand.Int(rand.Reader, max)
	return a
}
