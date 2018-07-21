// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package mru

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const signatureLength = 65

// Signature is an alias for a static byte array with the size of a signature
type Signature [signatureLength]byte

// Signer signs Mutable Resource update payloads
type Signer interface {
	Sign(common.Hash) (Signature, error)
	Address() common.Address
}

// GenericSigner implements the Signer interface
// It is the vanilla signer that probably should be used in most cases
type GenericSigner struct {
	PrivKey *ecdsa.PrivateKey
	address common.Address
}

// NewGenericSigner builds a signer that will sign everything with the provided private key
func NewGenericSigner(privKey *ecdsa.PrivateKey) *GenericSigner {
	return &GenericSigner{
		PrivKey: privKey,
		address: crypto.PubkeyToAddress(privKey.PublicKey),
	}
}

// Sign signs the supplied data
// It wraps the ethereum crypto.Sign() method
func (s *GenericSigner) Sign(data common.Hash) (signature Signature, err error) {
	signaturebytes, err := crypto.Sign(data.Bytes(), s.PrivKey)
	if err != nil {
		return
	}
	copy(signature[:], signaturebytes)
	return
}

// PublicKey returns the public key of the signer's private key
func (s *GenericSigner) Address() common.Address {
	return s.address
}
