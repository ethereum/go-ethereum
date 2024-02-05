package bls12381

import (
	"math/big"
)

func bigFromHex(hex string) *big.Int {
	if len(hex) > 1 && hex[:2] == "0x" {
		hex = hex[2:]
	}
	n, _ := new(big.Int).SetString(hex, 16)
	return n
}
