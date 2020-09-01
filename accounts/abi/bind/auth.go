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
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/external"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// NewTransactor is a utility method to easily create a transaction signer from
// an encrypted json key stream and the associated passphrase.
//
// WARNING: NewTransactor has been deprecated in favour of NewTransactorWithChainID.
func NewTransactor(keyin io.Reader, passphrase string) (*TransactOpts, error) {
	fmt.Println("WARNING: NewTransactor has been deprecated in favour of NewTransactorWithChainID")
	json, err := ioutil.ReadAll(keyin)
	if err != nil {
		return nil, err
	}
	key, err := keystore.DecryptKey(json, passphrase)
	if err != nil {
		return nil, err
	}
	return NewKeyedTransactor(key.PrivateKey), nil
}

// NewKeyStoreTransactor is a utility method to easily create a transaction signer from
// an decrypted key from a keystore.
//
// WARNING: NewKeyStoreTransactor has been deprecated in favour of NewTransactorWithChainID.
func NewKeyStoreTransactor(keystore *keystore.KeyStore, account accounts.Account) (*TransactOpts, error) {
	fmt.Println("WARNING: NewKeyStoreTransactor has been deprecated in favour of NewTransactorWithChainID")
	signer := types.HomesteadSigner{}
	return &TransactOpts{
		From: account.Address,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != account.Address {
				return nil, errors.New("not authorized to sign this account")
			}
			signature, err := keystore.SignHash(account, signer.Hash(tx).Bytes())
			if err != nil {
				return nil, err
			}
			return tx.WithSignature(signer, signature)
		},
	}, nil
}

// NewKeyedTransactor is a utility method to easily create a transaction signer
// from a single private key.
//
// WARNING: NewKeyedTransactor has been deprecated in favour of NewKeyedTransactorWithChainID.
func NewKeyedTransactor(key *ecdsa.PrivateKey) *TransactOpts {
	fmt.Println("WARNING: NewKeyedTransactor has been deprecated in favour of NewKeyedTransactorWithChainID")
	keyAddr := crypto.PubkeyToAddress(key.PublicKey)
	signer := types.HomesteadSigner{}
	return &TransactOpts{
		From: keyAddr,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != keyAddr {
				return nil, errors.New("not authorized to sign this account")
			}
			signature, err := crypto.Sign(signer.Hash(tx).Bytes(), key)
			if err != nil {
				return nil, err
			}
			return tx.WithSignature(signer, signature)
		},
	}
}

// NewTransactorWithChainID is a utility method to easily create a transaction signer from
// an encrypted json key stream and the associated passphrase.
func NewTransactorWithChainID(keyin io.Reader, passphrase string, chainID *big.Int) (*TransactOpts, error) {
	json, err := ioutil.ReadAll(keyin)
	if err != nil {
		return nil, err
	}
	key, err := keystore.DecryptKey(json, passphrase)
	if err != nil {
		return nil, err
	}
	return NewKeyedTransactorWithChainID(key.PrivateKey, chainID), nil
}

// NewKeyStoreTransactorWithChainID is a utility method to easily create a transaction signer from
// an decrypted key from a keystore
func NewKeyStoreTransactorWithChainID(keystore *keystore.KeyStore, account accounts.Account, chainID *big.Int) (*TransactOpts, error) {
	if chainID == nil {
		// If no chainID is explicitly passed, sign for mainnet
		chainID = common.Big1
	}
	signer := types.NewEIP155Signer(chainID)
	return &TransactOpts{
		From: account.Address,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != account.Address {
				return nil, errors.New("not authorized to sign this account")
			}
			signature, err := keystore.SignHash(account, signer.Hash(tx).Bytes())
			if err != nil {
				return nil, err
			}
			return tx.WithSignature(signer, signature)
		},
	}, nil
}

// NewKeyedTransactorWithChainID is a utility method to easily create a transaction signer
// from a single private key.
func NewKeyedTransactorWithChainID(key *ecdsa.PrivateKey, chainID *big.Int) *TransactOpts {
	keyAddr := crypto.PubkeyToAddress(key.PublicKey)
	if chainID == nil {
		// If no chainID is explicitly passed, sign for mainnet
		chainID = common.Big1
	}
	signer := types.NewEIP155Signer(chainID)
	return &TransactOpts{
		From: keyAddr,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != keyAddr {
				return nil, errors.New("not authorized to sign this account")
			}
			signature, err := crypto.Sign(signer.Hash(tx).Bytes(), key)
			if err != nil {
				return nil, err
			}
			return tx.WithSignature(signer, signature)
		},
	}
}

// NewClefTransactor is a utility method to easily create a transaction signer
// with a clef backend.
func NewClefTransactor(clef *external.ExternalSigner, account accounts.Account) *TransactOpts {
	return &TransactOpts{
		From: account.Address,
		Signer: func(address common.Address, transaction *types.Transaction) (*types.Transaction, error) {
			if address != account.Address {
				return nil, errors.New("not authorized to sign this account")
			}
			return clef.SignTx(account, transaction, nil) // Clef enforces its own chain id
		},
	}
}
