package modexp

import (
	"github.com/ethereum/go-ethereum/crypto/modexp/gmp"
)

// ModExp performs modular exponentiation on byte arrays
// result = base^exp mod mod
// This uses the bigint implementation by default.
// To use GMP implementation, import crypto/modexp/gmp directly.
func ModExp(base, exp, mod []byte) ([]byte, error) {
	return gmp.ModExp(base, exp, mod)
}