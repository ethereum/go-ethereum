// Copyright 2016 The go-ethereum Authors
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

package bind

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// NewTransactor is a utility method to easily create a transaction signer from
// an encrypted json key file and the associated passphrase.
func NewTransactor(keyjson string, passphrase string) (*TransactOpts, error) {
	key, err := crypto.DecryptKey([]byte(keyjson), passphrase)
	if err != nil {
		return nil, err
	}
	return NewKeyedTransactor(key), nil
}

// NewKeyedTransactor is a utility method to easily create a transaction signer
// from a plain go-ethereum crypto key.
func NewKeyedTransactor(key *crypto.Key) *TransactOpts {
	return &TransactOpts{
		Account: key.Address,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != key.Address {
				return nil, errors.New("not authorized to sign this account")
			}
			signature, err := crypto.Sign(tx.SigHash().Bytes(), key.PrivateKey)
			if err != nil {
				return nil, err
			}
			return tx.WithSignature(signature)
		},
	}
}
