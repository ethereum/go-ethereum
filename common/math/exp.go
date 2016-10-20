package math

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const (
	// Perverted constants straight out of big.Int
	// https://golang.org/src/math/big/arith.go
	_m    = ^big.Word(0)
	_logS = _m>>8&1 + _m>>16&1 + _m>>32&1
	_S    = 1 << _logS
	_W    = _S << 3 // word size in bits
)

// Exp implement exponentiation by squaring algorithm.
//
// Courtesy @karalabe and @chfast
func Exp(base, exponent *big.Int) *big.Int {
	result := big.NewInt(1)

	for _, word := range exponent.Bits() {
		for i := 0; i < _W; i++ {
			if word&1 == 1 {
				common.U256(result.Mul(result, base))
			}
			common.U256(base.Mul(base, base))
			word >>= 1
		}
	}
	return result
}
