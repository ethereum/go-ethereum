package storage

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// matches the SignFunc type
func NewGenericResourceSigner(privKey *ecdsa.PrivateKey) SignFunc {
	return func(data common.Hash) (signature Signature, err error) {
		signaturebytes, err := crypto.Sign(data.Bytes(), privKey)
		if err != nil {
			return
		}
		copy(signature[:], signaturebytes)
		return
	}
}
