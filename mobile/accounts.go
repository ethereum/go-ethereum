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
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

const (
	// StandardScryptN is the N parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptN = int(keystore.StandardScryptN)

	// StandardScryptP is the P parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptP = int(keystore.StandardScryptP)

	// LightScryptN is the N parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptN = int(keystore.LightScryptN)

	// LightScryptP is the P parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptP = int(keystore.LightScryptP)
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
	return a.account.URL
}

// KeyStore manages a key storage directory on disk.
type KeyStore struct{ keystore *keystore.KeyStore }

// NewKeyStore creates a keystore for the given directory.
func NewKeyStore(keydir string, scryptN, scryptP int) *KeyStore {
	return &KeyStore{keystore: keystore.NewKeyStore(keydir, scryptN, scryptP)}
}

// HasAddress reports whether a key with the given address is present.
func (ks *KeyStore) HasAddress(address *Address) bool {
	return ks.keystore.HasAddress(address.address)
}

// GetAccounts returns all key files present in the directory.
func (ks *KeyStore) GetAccounts() *Accounts {
	return &Accounts{ks.keystore.Accounts()}
}

// DeleteAccount deletes the key matched by account if the passphrase is correct.
// If a contains no filename, the address must match a unique key.
func (ks *KeyStore) DeleteAccount(account *Account, passphrase string) error {
	return ks.keystore.Delete(account.account, passphrase)
}

// SignHash calculates a ECDSA signature for the given hash. The produced signature
// is in the [R || S || V] format where V is 0 or 1.
func (ks *KeyStore) SignHash(address *Address, hash []byte) (signature []byte, _ error) {
	return ks.keystore.SignHash(accounts.Account{Address: address.address}, hash)
}

// SignTx signs the given transaction with the requested account.
func (ks *KeyStore) SignTx(account *Account, tx *Transaction, chainID *BigInt) (*Transaction, error) {
	signed, err := ks.keystore.SignTx(account.account, tx.tx, chainID.bigint)
	if err != nil {
		return nil, err
	}
	return &Transaction{signed}, nil
}

// SignHashPassphrase signs hash if the private key matching the given address can
// be decrypted with the given passphrase. The produced signature is in the
// [R || S || V] format where V is 0 or 1.
func (ks *KeyStore) SignHashPassphrase(account *Account, passphrase string, hash []byte) (signature []byte, _ error) {
	return ks.keystore.SignHashWithPassphrase(account.account, passphrase, hash)
}

// SignTxPassphrase signs the transaction if the private key matching the
// given address can be decrypted with the given passphrase.
func (ks *KeyStore) SignTxPassphrase(account *Account, passphrase string, tx *Transaction, chainID *BigInt) (*Transaction, error) {
	signed, err := ks.keystore.SignTxWithPassphrase(account.account, passphrase, tx.tx, chainID.bigint)
	if err != nil {
		return nil, err
	}
	return &Transaction{signed}, nil
}

// Unlock unlocks the given account indefinitely.
func (ks *KeyStore) Unlock(account *Account, passphrase string) error {
	return ks.keystore.TimedUnlock(account.account, passphrase, 0)
}

// Lock removes the private key with the given address from memory.
func (ks *KeyStore) Lock(address *Address) error {
	return ks.keystore.Lock(address.address)
}

// TimedUnlock unlocks the given account with the passphrase. The account stays
// unlocked for the duration of timeout (nanoseconds). A timeout of 0 unlocks the
// account until the program exits. The account must match a unique key file.
//
// If the account address is already unlocked for a duration, TimedUnlock extends or
// shortens the active unlock timeout. If the address was previously unlocked
// indefinitely the timeout is not altered.
func (ks *KeyStore) TimedUnlock(account *Account, passphrase string, timeout int64) error {
	return ks.keystore.TimedUnlock(account.account, passphrase, time.Duration(timeout))
}

// NewAccount generates a new key and stores it into the key directory,
// encrypting it with the passphrase.
func (ks *KeyStore) NewAccount(passphrase string) (*Account, error) {
	account, err := ks.keystore.NewAccount(passphrase)
	if err != nil {
		return nil, err
	}
	return &Account{account}, nil
}

// ExportKey exports as a JSON key, encrypted with newPassphrase.
func (ks *KeyStore) ExportKey(account *Account, passphrase, newPassphrase string) (key []byte, _ error) {
	return ks.keystore.Export(account.account, passphrase, newPassphrase)
}

// ImportKey stores the given encrypted JSON key into the key directory.
func (ks *KeyStore) ImportKey(keyJSON []byte, passphrase, newPassphrase string) (account *Account, _ error) {
	acc, err := ks.keystore.Import(keyJSON, passphrase, newPassphrase)
	if err != nil {
		return nil, err
	}
	return &Account{acc}, nil
}

// UpdateAccount changes the passphrase of an existing account.
func (ks *KeyStore) UpdateAccount(account *Account, passphrase, newPassphrase string) error {
	return ks.keystore.Update(account.account, passphrase, newPassphrase)
}

// ImportPreSaleKey decrypts the given Ethereum presale wallet and stores
// a key file in the key directory. The key file is encrypted with the same passphrase.
func (ks *KeyStore) ImportPreSaleKey(keyJSON []byte, passphrase string) (ccount *Account, _ error) {
	account, err := ks.keystore.ImportPreSaleKey(keyJSON, passphrase)
	if err != nil {
		return nil, err
	}
	return &Account{account}, nil
}
