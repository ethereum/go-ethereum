package secp256r1

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"math/big"
)

var (
	// Half of the order of the subgroup in the elliptic curve
	secp256k1halfN = new(big.Int).Div(elliptic.P256().Params().N, big.NewInt(2))
)

// Verifies the given signature (r, s) for the given hash and public key (x, y).
func Verify(hash []byte, r, s, x, y *big.Int) bool {
	// Create the public key format
	publicKey := newPublicKey(x, y)

	// Check if they are invalid public key coordinates
	if publicKey == nil {
		return false
	}

	// Check the malleability issue
	if checkMalleability(s) {
		return false
	}

	// Verify the signature with the public key,
	// then return true if it's valid, false otherwise
	if ok := ecdsa.Verify(publicKey, hash, r, s); ok {
		return true
	}

	return false
}

// Check the malleability issue
func checkMalleability(s *big.Int) bool {
	return s.Cmp(secp256k1halfN) > 0
}
