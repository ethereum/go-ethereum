package secp256r1

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"math/big"
)

// Generates appropriate public key format from given coordinates
func newPublicKey(x, y *big.Int) *ecdsa.PublicKey {
	// Check if the given coordinates are valid
	if x == nil || y == nil || !elliptic.P256().IsOnCurve(x, y) {
		return nil
	}

	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}
}
