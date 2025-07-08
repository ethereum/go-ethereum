package bigint

import (
	"math/big"
)

// ModExp performs modular exponentiation on byte arrays using math/big
// result = base^exp mod mod
// This function matches the behavior of the EVM modexp precompile
func ModExp(base, exp, mod []byte) ([]byte, error) {
	// Create big.Int values
	baseBig := new(big.Int).SetBytes(base)
	expBig := new(big.Int).SetBytes(exp)
	modBig := new(big.Int).SetBytes(mod)

	var v []byte
	switch {
	case modBig.BitLen() == 0:
		// Modulo 0 is undefined, return zero (matching EVM behavior)
		return []byte{}, nil
	case baseBig.BitLen() == 1: // a bit length of 1 means it's 1 (or -1).
		// If base == 1, then we can just return base % mod (if mod >= 1, which it is)
		v = baseBig.Mod(baseBig, modBig).Bytes()
	default:
		v = baseBig.Exp(baseBig, expBig, modBig).Bytes()
	}

	// Return the result bytes without padding
	// The caller is responsible for padding if needed
	return v, nil
}