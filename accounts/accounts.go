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

// Package accounts implements high level Ethereum account management.
package accounts

import (
	"encoding/json"
	"errors"
	"math/big"
	"reflect"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ErrUnknownAccount is returned for any requested operation for which no backend
// provides the specified account.
var ErrUnknownAccount = errors.New("unknown account")

// ErrNotSupported is returned when an operation is requested from an account
// backend that it does not support.
var ErrNotSupported = errors.New("not supported")

// Account represents a stored key.
// When used as an argument, it selects a unique key to act on.
type Account struct {
	Address common.Address // Ethereum account address derived from the key
	URL     string         // Optional resource locator within a backend
	backend Backend        // Backend where this account originates from
}

func (acc *Account) MarshalJSON() ([]byte, error) {
	return []byte(`"` + acc.Address.Hex() + `"`), nil
}

func (acc *Account) UnmarshalJSON(raw []byte) error {
	return json.Unmarshal(raw, &acc.Address)
}

// Manager is an overarching account manager that can communicate with various
// backends for signing transactions.
type Manager struct {
	backends []Backend                // List of currently registered backends (ordered by registration)
	index    map[reflect.Type]Backend // Set of currently registered backends
	lock     sync.RWMutex
}

// NewManager creates a generic account manager to sign transaction via various
// supported backends.
func NewManager(backends ...Backend) *Manager {
	am := &Manager{
		backends: backends,
		index:    make(map[reflect.Type]Backend),
	}
	for _, backend := range backends {
		am.index[reflect.TypeOf(backend)] = backend
	}
	return am
}

// Backend retrieves the backend with the given type from the account manager.
func (am *Manager) Backend(backend reflect.Type) Backend {
	return am.index[backend]
}

// Accounts returns all signer accounts registered under this account manager.
func (am *Manager) Accounts() []Account {
	am.lock.RLock()
	defer am.lock.RUnlock()

	var all []Account
	for _, backend := range am.backends { // TODO(karalabe): cache these after subscriptions are in
		accounts := backend.Accounts()
		for i := 0; i < len(accounts); i++ {
			accounts[i].backend = backend
		}
		all = append(all, accounts...)
	}
	return all
}

// HasAddress reports whether a key with the given address is present.
func (am *Manager) HasAddress(addr common.Address) bool {
	am.lock.RLock()
	defer am.lock.RUnlock()

	for _, backend := range am.backends {
		if backend.HasAddress(addr) {
			return true
		}
	}
	return false
}

// SignHash requests the account manager to get the hash signed with an arbitrary
// signing backend holding the authorization for the specified account.
func (am *Manager) SignHash(acc Account, hash []byte) ([]byte, error) {
	am.lock.RLock()
	defer am.lock.RUnlock()

	if err := am.ensureBackend(&acc); err != nil {
		return nil, err
	}
	return acc.backend.SignHash(acc, hash)
}

// SignTx requests the account manager to get the transaction signed with an
// arbitrary signing backend holding the authorization for the specified account.
func (am *Manager) SignTx(acc Account, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	am.lock.RLock()
	defer am.lock.RUnlock()

	if err := am.ensureBackend(&acc); err != nil {
		return nil, err
	}
	return acc.backend.SignTx(acc, tx, chainID)
}

// SignHashWithPassphrase requests the account manager to get the hash signed with
// an arbitrary signing backend holding the authorization for the specified account.
func (am *Manager) SignHashWithPassphrase(acc Account, passphrase string, hash []byte) ([]byte, error) {
	am.lock.RLock()
	defer am.lock.RUnlock()

	if err := am.ensureBackend(&acc); err != nil {
		return nil, err
	}
	return acc.backend.SignHashWithPassphrase(acc, passphrase, hash)
}

// SignTxWithPassphrase requests the account manager to get the transaction signed
// with an arbitrary signing backend holding the authorization for the specified
// account.
func (am *Manager) SignTxWithPassphrase(acc Account, passphrase string, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	am.lock.RLock()
	defer am.lock.RUnlock()

	if err := am.ensureBackend(&acc); err != nil {
		return nil, err
	}
	return acc.backend.SignTxWithPassphrase(acc, passphrase, tx, chainID)
}

// ensureBackend ensures that the account has a correctly set backend and that
// it is still alive.
//
// Please note, this method assumes the manager lock is held!
func (am *Manager) ensureBackend(acc *Account) error {
	// If we have a backend, make sure it's still live
	if acc.backend != nil {
		if _, exists := am.index[reflect.TypeOf(acc.backend)]; !exists {
			return ErrUnknownAccount
		}
		return nil
	}
	// If we don't have a known backend, look up one that can service it
	for _, backend := range am.backends {
		if backend.HasAddress(acc.Address) { // TODO(karalabe): this assumes unique addresses per backend
			acc.backend = backend
			return nil
		}
	}
	return ErrUnknownAccount
}
