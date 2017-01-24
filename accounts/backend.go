// Copyright 2017 The go-ethereum Authors
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

package accounts

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Backend is an "account provider" that can specify a batch of accounts it can
// sign transactions with and upon request, do so.
type Backend interface {
	// Accounts retrieves the list of signing accounts the backend is currently aware of.
	Accounts() []Account

	// HasAddress reports whether an account with the given address is present.
	HasAddress(addr common.Address) bool

	// SignHash requests the backend to sign the given hash.
	//
	// It looks up the account specified either solely via its address contained within,
	// or optionally with the aid of any location metadata from the embedded URL field.
	//
	// If the backend requires additional authentication to sign the request (e.g.
	// a password to decrypt the account, or a PIN code o verify the transaction),
	// an AuthNeededError instance will be returned, containing infos for the user
	// about which fields or actions are needed. The user may retry by providing
	// the needed details via SignHashWithPassphrase, or by other means (e.g. unlock
	// the account in a keystore).
	SignHash(acc Account, hash []byte) ([]byte, error)

	// SignTx requests the backend to sign the given transaction.
	//
	// It looks up the account specified either solely via its address contained within,
	// or optionally with the aid of any location metadata from the embedded URL field.
	//
	// If the backend requires additional authentication to sign the request (e.g.
	// a password to decrypt the account, or a PIN code o verify the transaction),
	// an AuthNeededError instance will be returned, containing infos for the user
	// about which fields or actions are needed. The user may retry by providing
	// the needed details via SignTxWithPassphrase, or by other means (e.g. unlock
	// the account in a keystore).
	SignTx(acc Account, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)

	// SignHashWithPassphrase requests the backend to sign the given transaction with
	// the given passphrase as extra authentication information.
	//
	// It looks up the account specified either solely via its address contained within,
	// or optionally with the aid of any location metadata from the embedded URL field.
	SignHashWithPassphrase(acc Account, passphrase string, hash []byte) ([]byte, error)

	// SignTxWithPassphrase requests the backend to sign the given transaction, with the
	// given passphrase as extra authentication information.
	//
	// It looks up the account specified either solely via its address contained within,
	// or optionally with the aid of any location metadata from the embedded URL field.
	SignTxWithPassphrase(acc Account, passphrase string, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)

	// TODO(karalabe,fjl): watching and caching needs the Go subscription system
	// Watch requests the backend to send a notification to the specified channel whenever
	// an new account appears or an existing one disappears.
	//Watch(chan AccountEvent) error

	// Unwatch requests the backend stop sending notifications to the given channel.
	//Unwatch(chan AccountEvent) error
}

// TODO(karalabe,fjl): watching and caching needs the Go subscription system
// type AccountEvent struct {
// 	Account Account
// 	Added   bool
// }
