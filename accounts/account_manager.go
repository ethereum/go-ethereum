// Copyright 2015 The go-ethereum Authors
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

// Package accounts implements encrypted storage of secp256k1 private keys.
//
// Keys are stored as encrypted JSON files according to the Web3 Secret Storage specification.
// See https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition for more information.
package accounts

import (
	"crypto/ecdsa"
	crand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ErrLocked  = errors.New("account is locked")
	ErrNoMatch = errors.New("no key for given address or file")
	ErrDecrypt = errors.New("could not decrypt key with given passphrase")
)

// Account represents a stored key.
// When used as an argument, it selects a unique key file to act on.
type Account struct {
	Address common.Address // Ethereum account address derived from the key

	// File contains the key file name.
	// When Acccount is used as an argument to select a key, File can be left blank to
	// select just by address or set to the basename or absolute path of a file in the key
	// directory. Accounts returned by Manager will always contain an absolute path.
	File string
}

func (acc *Account) MarshalJSON() ([]byte, error) {
	return []byte(`"` + acc.Address.Hex() + `"`), nil
}

func (acc *Account) UnmarshalJSON(raw []byte) error {
	return json.Unmarshal(raw, &acc.Address)
}

// Manager manages a key storage directory on disk.
type Manager struct {
	cache    *addrCache
	keyStore keyStore
	mu       sync.RWMutex
	unlocked map[common.Address]*unlocked
}

type unlocked struct {
	*Key
	abort chan struct{}
}

// NewManager creates a manager for the given directory.
func NewManager(keydir string, scryptN, scryptP int) *Manager {
	keydir, _ = filepath.Abs(keydir)
	am := &Manager{keyStore: &keyStorePassphrase{keydir, scryptN, scryptP}}
	am.init(keydir)
	return am
}

// NewPlaintextManager creates a manager for the given directory.
// Deprecated: Use NewManager.
func NewPlaintextManager(keydir string) *Manager {
	keydir, _ = filepath.Abs(keydir)
	am := &Manager{keyStore: &keyStorePlain{keydir}}
	am.init(keydir)
	return am
}

func (am *Manager) init(keydir string) {
	am.unlocked = make(map[common.Address]*unlocked)
	am.cache = newAddrCache(keydir)
	// TODO: In order for this finalizer to work, there must be no references
	// to am. addrCache doesn't keep a reference but unlocked keys do,
	// so the finalizer will not trigger until all timed unlocks have expired.
	runtime.SetFinalizer(am, func(m *Manager) {
		m.cache.close()
	})
}

// HasAddress reports whether a key with the given address is present.
func (am *Manager) HasAddress(addr common.Address) bool {
	return am.cache.hasAddress(addr)
}

// Accounts returns all key files present in the directory.
func (am *Manager) Accounts() []Account {
	return am.cache.accounts()
}

// DeleteAccount deletes the key matched by account if the passphrase is correct.
// If a contains no filename, the address must match a unique key.
func (am *Manager) DeleteAccount(a Account, passphrase string) error {
	// Decrypting the key isn't really necessary, but we do
	// it anyway to check the password and zero out the key
	// immediately afterwards.
	a, key, err := am.getDecryptedKey(a, passphrase)
	if key != nil {
		zeroKey(key.PrivateKey)
	}
	if err != nil {
		return err
	}
	// The order is crucial here. The key is dropped from the
	// cache after the file is gone so that a reload happening in
	// between won't insert it into the cache again.
	err = os.Remove(a.File)
	if err == nil {
		am.cache.delete(a)
	}
	return err
}

// Sign signs hash with an unlocked private key matching the given address.
func (am *Manager) Sign(addr common.Address, hash []byte) (signature []byte, err error) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	unlockedKey, found := am.unlocked[addr]
	if !found {
		return nil, ErrLocked
	}
	return crypto.Sign(hash, unlockedKey.PrivateKey)
}

// Unlock unlocks the given account indefinitely.
func (am *Manager) Unlock(a Account, keyAuth string) error {
	return am.TimedUnlock(a, keyAuth, 0)
}

// Lock removes the private key with the given address from memory.
func (am *Manager) Lock(addr common.Address) error {
	am.mu.Lock()
	if unl, found := am.unlocked[addr]; found {
		am.mu.Unlock()
		am.expire(addr, unl, time.Duration(0)*time.Nanosecond)
	} else {
		am.mu.Unlock()
	}
	return nil
}

// TimedUnlock unlocks the given account with. The account
// stays unlocked for the duration of timeout. A timeout of 0 unlocks the account
// until the program exits. The account must match a unique key.
//
// If the accout is already unlocked, TimedUnlock extends or shortens
// the active unlock timeout.
func (am *Manager) TimedUnlock(a Account, keyAuth string, timeout time.Duration) error {
	_, key, err := am.getDecryptedKey(a, keyAuth)
	if err != nil {
		return err
	}

	am.mu.Lock()
	defer am.mu.Unlock()
	u, found := am.unlocked[a.Address]
	if found {
		// terminate dropLater for this key to avoid unexpected drops.
		if u.abort != nil {
			close(u.abort)
		}
	}
	if timeout > 0 {
		u = &unlocked{Key: key, abort: make(chan struct{})}
		go am.expire(a.Address, u, timeout)
	} else {
		u = &unlocked{Key: key}
	}
	am.unlocked[a.Address] = u
	return nil
}

func (am *Manager) getDecryptedKey(a Account, auth string) (Account, *Key, error) {
	am.cache.maybeReload()
	am.cache.mu.Lock()
	a, err := am.cache.find(a)
	am.cache.mu.Unlock()
	if err != nil {
		return a, nil, err
	}
	key, err := am.keyStore.GetKey(a.Address, a.File, auth)
	return a, key, err
}

func (am *Manager) expire(addr common.Address, u *unlocked, timeout time.Duration) {
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-u.abort:
		// just quit
	case <-t.C:
		am.mu.Lock()
		// only drop if it's still the same key instance that dropLater
		// was launched with. we can check that using pointer equality
		// because the map stores a new pointer every time the key is
		// unlocked.
		if am.unlocked[addr] == u {
			zeroKey(u.PrivateKey)
			delete(am.unlocked, addr)
		}
		am.mu.Unlock()
	}
}

// NewAccount generates a new key and stores it into the key directory,
// encrypting it with the passphrase.
func (am *Manager) NewAccount(passphrase string) (Account, error) {
	_, account, err := storeNewKey(am.keyStore, crand.Reader, passphrase)
	if err != nil {
		return Account{}, err
	}
	// Add the account to the cache immediately rather
	// than waiting for file system notifications to pick it up.
	am.cache.add(account)
	return account, nil
}

// AccountByIndex returns the ith account.
func (am *Manager) AccountByIndex(i int) (Account, error) {
	accounts := am.Accounts()
	if i < 0 || i >= len(accounts) {
		return Account{}, fmt.Errorf("account index %d out of range [0, %d]", i, len(accounts)-1)
	}
	return accounts[i], nil
}

// Export exports as a JSON key, encrypted with newPassphrase.
func (am *Manager) Export(a Account, passphrase, newPassphrase string) (keyJSON []byte, err error) {
	_, key, err := am.getDecryptedKey(a, passphrase)
	if err != nil {
		return nil, err
	}
	var N, P int
	if store, ok := am.keyStore.(*keyStorePassphrase); ok {
		N, P = store.scryptN, store.scryptP
	} else {
		N, P = StandardScryptN, StandardScryptP
	}
	return EncryptKey(key, newPassphrase, N, P)
}

// Import stores the given encrypted JSON key into the key directory.
func (am *Manager) Import(keyJSON []byte, passphrase, newPassphrase string) (Account, error) {
	key, err := DecryptKey(keyJSON, passphrase)
	if key != nil && key.PrivateKey != nil {
		defer zeroKey(key.PrivateKey)
	}
	if err != nil {
		return Account{}, err
	}
	return am.importKey(key, newPassphrase)
}

// ImportECDSA stores the given key into the key directory, encrypting it with the passphrase.
func (am *Manager) ImportECDSA(priv *ecdsa.PrivateKey, passphrase string) (Account, error) {
	return am.importKey(newKeyFromECDSA(priv), passphrase)
}

func (am *Manager) importKey(key *Key, passphrase string) (Account, error) {
	a := Account{Address: key.Address, File: am.keyStore.JoinPath(keyFileName(key.Address))}
	if err := am.keyStore.StoreKey(a.File, key, passphrase); err != nil {
		return Account{}, err
	}
	am.cache.add(a)
	return a, nil
}

// Update changes the passphrase of an existing account.
func (am *Manager) Update(a Account, passphrase, newPassphrase string) error {
	a, key, err := am.getDecryptedKey(a, passphrase)
	if err != nil {
		return err
	}
	return am.keyStore.StoreKey(a.File, key, newPassphrase)
}

// ImportPreSaleKey decrypts the given Ethereum presale wallet and stores
// a key file in the key directory. The key file is encrypted with the same passphrase.
func (am *Manager) ImportPreSaleKey(keyJSON []byte, passphrase string) (Account, error) {
	a, _, err := importPreSaleKey(am.keyStore, keyJSON, passphrase)
	if err != nil {
		return a, err
	}
	am.cache.add(a)
	return a, nil
}

// zeroKey zeroes a private key in memory.
func zeroKey(k *ecdsa.PrivateKey) {
	b := k.D.Bits()
	for i := range b {
		b[i] = 0
	}
}
