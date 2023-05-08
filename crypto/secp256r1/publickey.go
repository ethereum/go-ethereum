package secp256r1

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"math/big"
)

// Generates approptiate public key format from given coordinates
func newPublicKey(x, y *big.Int) *ecdsa.PublicKey {
	// Check if the given coordinates are valid
	if x == nil || y == nil || !elliptic.P256().IsOnCurve(x, y) {
		return nil
	}

	// Check if the given coordinates are the reference point (infinity)
	if x.Sign() == 0 && y.Sign() == 0 {
		return nil
	}

	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}
}
