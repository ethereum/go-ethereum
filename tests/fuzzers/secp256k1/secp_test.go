// Copyright 2021-2022 The go-xpayments Authors
// This file is part of go-xpayments.

package secp256k1

import "testing"

func TestFuzzer(t *testing.T) {
	test := "00000000N0000000/R00000000000000000U0000S0000000mkhP000000000000000U"
	Fuzz([]byte(test))
}
