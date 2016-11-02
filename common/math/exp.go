package math

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// wordSize is the size number of bits in a big.Int Word.
const wordSize = 32 << (uint64(^big.Word(0))>>63)

// Exp implement exponentiation by squaring algorithm.
//
// Courtesy @karalabe and @chfast
func Exp(base, exponent *big.Int) *big.Int {
	result := big.NewInt(1)

	for _, word := range exponent.Bits() {
		for i := 0; i < wordSize; i++ {
			if word&1 == 1 {
				common.U256(result.Mul(result, base))
			}
			common.U256(base.Mul(base, base))
			word >>= 1
		}
	}
	return result
}
