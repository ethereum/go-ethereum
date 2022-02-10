// Copyright 2021-2022 The go-xpayments Authors
// This file is part of go-xpayments.

package bls12381

import (
	"crypto/rand"
	"math/big"
)

var fuz = 10

func randScalar(max *big.Int) *big.Int {
	a, _ := rand.Int(rand.Reader, max)
	return a
}
