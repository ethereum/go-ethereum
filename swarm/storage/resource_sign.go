package storage

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Signs resource updates
type ResourceSigner interface {
	Sign(common.Hash) (Signature, error)
}

type GenericResourceSigner struct {
	PrivKey *ecdsa.PrivateKey
}

func (self *GenericResourceSigner) Sign(data common.Hash) (signature Signature, err error) {
	signaturebytes, err := crypto.Sign(data.Bytes(), self.PrivKey)
	if err != nil {
		return
	}
	copy(signature[:], signaturebytes)
	return
}
