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

// Contains all the wrappers from the accounts package to support client side key
// management on mobile platforms.

package geth

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
)

const (
	// StandardScryptN is the N parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptN = int(accounts.StandardScryptN)

	// StandardScryptP is the P parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptP = int(accounts.StandardScryptP)

	// LightScryptN is the N parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptN = int(accounts.LightScryptN)

	// LightScryptP is the P parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptP = int(accounts.LightScryptP)
)

// Account represents a stored key.
type Account struct{ account accounts.Account }

// Accounts represents a slice of accounts.
type Accounts struct{ accounts []accounts.Account }

// Size returns the number of accounts in the slice.
func (a *Accounts) Size() int {
	return len(a.accounts)
}

// Get returns the account at the given index from the slice.
func (a *Accounts) Get(index int) (account *Account, _ error) {
	if index < 0 || index >= len(a.accounts) {
		return nil, errors.New("index out of bounds")
	}
	return &Account{a.accounts[index]}, nil
}

// Set sets the account at the given index in the slice.
func (a *Accounts) Set(index int, account *Account) error {
	if index < 0 || index >= len(a.accounts) {
		return errors.New("index out of bounds")
	}
	a.accounts[index] = account.account
	return nil
}

// GetAddress retrieves the address associated with the account.
func (a *Account) GetAddress() *Address {
	return &Address{a.account.Address}
}

// GetFile retrieves the path of the file containing the account key.
func (a *Account) GetFile() string {
	return a.account.File
}

// AccountManager manages a key storage directory on disk.
type AccountManager struct{ manager *accounts.Manager }

// NewAccountManager creates a manager for the given directory.
func NewAccountManager(keydir string, scryptN, scryptP int) *AccountManager {
	return &AccountManager{manager: accounts.NewManager(keydir, scryptN, scryptP)}
}

// HasAddress reports whether a key with the given address is present.
func (am *AccountManager) HasAddress(address *Address) bool {
	return am.manager.HasAddress(address.address)
}

// GetAccounts returns all key files present in the directory.
func (am *AccountManager) GetAccounts() *Accounts {
	return &Accounts{am.manager.Accounts()}
}

// DeleteAccount deletes the key matched by account if the passphrase is correct.
// If a contains no filename, the address must match a unique key.
func (am *AccountManager) DeleteAccount(account *Account, passphrase string) error {
	return am.manager.DeleteAccount(accounts.Account{
		Address: account.account.Address,
		File:    account.account.File,
	}, passphrase)
}

// Sign calculates a ECDSA signature for the given hash. The produced signature
// is in the [R || S || V] format where V is 0 or 1.
func (am *AccountManager) Sign(address *Address, hash []byte) (signature []byte, _ error) {
	return am.manager.Sign(address.address, hash)
}

// SignPassphrase signs hash if the private key matching the given address can
// be decrypted with the given passphrase. The produced signature is in the
// [R || S || V] format where V is 0 or 1.
func (am *AccountManager) SignPassphrase(account *Account, passphrase string, hash []byte) (signature []byte, _ error) {
	return am.manager.SignWithPassphrase(account.account, passphrase, hash)
}

// Unlock unlocks the given account indefinitely.
func (am *AccountManager) Unlock(account *Account, passphrase string) error {
	return am.manager.TimedUnlock(account.account, passphrase, 0)
}

// Lock removes the private key with the given address from memory.
func (am *AccountManager) Lock(address *Address) error {
	return am.manager.Lock(address.address)
}

// TimedUnlock unlocks the given account with the passphrase. The account stays
// unlocked for the duration of timeout (nanoseconds). A timeout of 0 unlocks the
// account until the program exits. The account must match a unique key file.
//
// If the account address is already unlocked for a duration, TimedUnlock extends or
// shortens the active unlock timeout. If the address was previously unlocked
// indefinitely the timeout is not altered.
func (am *AccountManager) TimedUnlock(account *Account, passphrase string, timeout int64) error {
	return am.manager.TimedUnlock(account.account, passphrase, time.Duration(timeout))
}

// NewAccount generates a new key and stores it into the key directory,
// encrypting it with the passphrase.
func (am *AccountManager) NewAccount(passphrase string) (*Account, error) {
	account, err := am.manager.NewAccount(passphrase)
	if err != nil {
		return nil, err
	}
	return &Account{account}, nil
}

// ExportKey exports as a JSON key, encrypted with newPassphrase.
func (am *AccountManager) ExportKey(account *Account, passphrase, newPassphrase string) (key []byte, _ error) {
	return am.manager.Export(account.account, passphrase, newPassphrase)
}

// ImportKey stores the given encrypted JSON key into the key directory.
func (am *AccountManager) ImportKey(keyJSON []byte, passphrase, newPassphrase string) (account *Account, _ error) {
	acc, err := am.manager.Import(keyJSON, passphrase, newPassphrase)
	if err != nil {
		return nil, err
	}
	return &Account{acc}, nil
}

// UpdateAccount changes the passphrase of an existing account.
func (am *AccountManager) UpdateAccount(account *Account, passphrase, newPassphrase string) error {
	return am.manager.Update(account.account, passphrase, newPassphrase)
}

// ImportPreSaleKey decrypts the given Ethereum presale wallet and stores
// a key file in the key directory. The key file is encrypted with the same passphrase.
func (am *AccountManager) ImportPreSaleKey(keyJSON []byte, passphrase string) (ccount *Account, _ error) {
	account, err := am.manager.ImportPreSaleKey(keyJSON, passphrase)
	if err != nil {
		return nil, err
	}
	return &Account{account}, nil
}
