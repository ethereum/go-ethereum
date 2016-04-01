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

// Package implements a private key management facility.
//
// This abstracts part of a user's interaction with an account she controls.
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

type Account struct {
	Address common.Address
	File    string
}

func (acc *Account) MarshalJSON() ([]byte, error) {
	return []byte(`"` + acc.Address.Hex() + `"`), nil
}

func (acc *Account) UnmarshalJSON(raw []byte) error {
	return json.Unmarshal(raw, &acc.Address)
}

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

func NewManager(keydir string, scryptN, scryptP int) *Manager {
	keydir, _ = filepath.Abs(keydir)
	am := &Manager{keyStore: &keyStorePassphrase{keydir, scryptN, scryptP}}
	am.init(keydir)
	return am
}

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

func (am *Manager) HasAddress(addr common.Address) bool {
	return am.cache.hasAddress(addr)
}

func (am *Manager) Accounts() []Account {
	return am.cache.accounts()
}

func (am *Manager) DeleteAccount(a Account, auth string) error {
	// Decrypting the key isn't really necessary, but we do
	// it anyway to check the password and zero out the key
	// immediately afterwards.
	a, key, err := am.getDecryptedKey(a, auth)
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

func (am *Manager) Sign(a Account, toSign []byte) (signature []byte, err error) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	unlockedKey, found := am.unlocked[a.Address]
	if !found {
		return nil, ErrLocked
	}
	signature, err = crypto.Sign(toSign, unlockedKey.PrivateKey)
	return signature, err
}

// Unlock unlocks the given account indefinitely.
func (am *Manager) Unlock(a Account, keyAuth string) error {
	return am.TimedUnlock(a, keyAuth, 0)
}

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

// TimedUnlock unlocks the account with the given address. The account
// stays unlocked for the duration of timeout. A timeout of 0 unlocks the account
// until the program exits.
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

func (am *Manager) NewAccount(auth string) (Account, error) {
	_, account, err := storeNewKey(am.keyStore, crand.Reader, auth)
	if err != nil {
		return Account{}, err
	}
	// Add the account to the cache immediately rather
	// than waiting for file system notifications to pick it up.
	am.cache.add(account)
	return account, nil
}

func (am *Manager) AccountByIndex(index int) (Account, error) {
	accounts := am.Accounts()
	if index < 0 || index >= len(accounts) {
		return Account{}, fmt.Errorf("account index %d out of range [0, %d]", index, len(accounts)-1)
	}
	return accounts[index], nil
}

// USE WITH CAUTION = this will save an unencrypted private key on disk
// no cli or js interface
func (am *Manager) Export(path string, a Account, keyAuth string) error {
	_, key, err := am.getDecryptedKey(a, keyAuth)
	if err != nil {
		return err
	}
	return crypto.SaveECDSA(path, key.PrivateKey)
}

func (am *Manager) Import(path string, keyAuth string) (Account, error) {
	priv, err := crypto.LoadECDSA(path)
	if err != nil {
		return Account{}, err
	}
	return am.ImportECDSA(priv, keyAuth)
}

func (am *Manager) ImportECDSA(priv *ecdsa.PrivateKey, keyAuth string) (Account, error) {
	key := newKeyFromECDSA(priv)
	a := Account{Address: key.Address, File: am.keyStore.JoinPath(keyFileName(key.Address))}
	if err := am.keyStore.StoreKey(a.File, key, keyAuth); err != nil {
		return Account{}, err
	}
	am.cache.add(a)
	return a, nil
}

func (am *Manager) Update(a Account, authFrom, authTo string) error {
	a, key, err := am.getDecryptedKey(a, authFrom)
	if err != nil {
		return err
	}
	return am.keyStore.StoreKey(a.File, key, authTo)
}

func (am *Manager) ImportPreSaleKey(keyJSON []byte, password string) (Account, error) {
	a, _, err := importPreSaleKey(am.keyStore, keyJSON, password)
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
