package math

import (
	"math/big"

	"github.com/holiman/uint256"
)

var (
	U0   = uint256.NewInt(0)
	U1   = uint256.NewInt(1)
	U100 = uint256.NewInt(100)
)

func U256LTE(a, b *uint256.Int) bool {
	return a.Lt(b) || a.Eq(b)
}

func FromBig(v *big.Int) *uint256.Int {
	u, _ := uint256.FromBig(v)

	return u
}
