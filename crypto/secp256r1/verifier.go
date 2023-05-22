package secp256r1

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var (
	secp256k1halfN = new(big.Int).Div(elliptic.P256().Params().N, big.NewInt(2))
)

// Verifies the given signature (r, s) for the given hash and public key (x, y).
func Verify(hash []byte, r, s, x, y *big.Int) ([]byte, error) {
	// Create the public key format
	publicKey := newPublicKey(x, y)
	if publicKey == nil {
		return nil, errors.New("invalid public key coordinates")
	}

	if checkMalleability(s) {
		return nil, errors.New("malleability issue")
	}

	// Verify the signature with the public key and return 1 if it's valid, 0 otherwise
	if ok := ecdsa.Verify(publicKey, hash, r, s); ok {
		return common.LeftPadBytes(common.Big1.Bytes(), 32), nil
	}

	return common.LeftPadBytes(common.Big0.Bytes(), 32), nil
}

// Check the malleability issue
func checkMalleability(s *big.Int) bool {
	return s.Cmp(secp256k1halfN) > 0
}
