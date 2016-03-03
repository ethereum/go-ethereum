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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ErrLocked = errors.New("account is locked")
	ErrNoKeys = errors.New("no keys in store")
)

type Account struct {
	Address common.Address
}

func (acc *Account) MarshalJSON() ([]byte, error) {
	return []byte(`"` + acc.Address.Hex() + `"`), nil
}

type Manager struct {
	keyStore keyStore
	unlocked map[common.Address]*unlocked
	mutex    sync.RWMutex
}

type unlocked struct {
	*Key
	abort chan struct{}
}

func NewManager(keydir string, scryptN, scryptP int) *Manager {
	return &Manager{
		keyStore: newKeyStorePassphrase(keydir, scryptN, scryptP),
		unlocked: make(map[common.Address]*unlocked),
	}
}

func NewPlaintextManager(keydir string) *Manager {
	return &Manager{
		keyStore: newKeyStorePlain(keydir),
		unlocked: make(map[common.Address]*unlocked),
	}
}

func (am *Manager) HasAddress(addr common.Address) bool {
	accounts := am.Accounts()
	for _, acct := range accounts {
		if acct.Address == addr {
			return true
		}
	}
	return false
}

func (am *Manager) DeleteAccount(a Account, auth string) error {
	return am.keyStore.DeleteKey(a.Address, auth)
}

func (am *Manager) Sign(a Account, toSign []byte) (signature []byte, err error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
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
	am.mutex.Lock()
	if unl, found := am.unlocked[addr]; found {
		am.mutex.Unlock()
		am.expire(addr, unl, time.Duration(0)*time.Nanosecond)
	} else {
		am.mutex.Unlock()
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
	key, err := am.keyStore.GetKey(a.Address, keyAuth)
	if err != nil {
		return err
	}
	var u *unlocked
	am.mutex.Lock()
	defer am.mutex.Unlock()
	var found bool
	u, found = am.unlocked[a.Address]
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

func (am *Manager) expire(addr common.Address, u *unlocked, timeout time.Duration) {
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-u.abort:
		// just quit
	case <-t.C:
		am.mutex.Lock()
		// only drop if it's still the same key instance that dropLater
		// was launched with. we can check that using pointer equality
		// because the map stores a new pointer every time the key is
		// unlocked.
		if am.unlocked[addr] == u {
			zeroKey(u.PrivateKey)
			delete(am.unlocked, addr)
		}
		am.mutex.Unlock()
	}
}

func (am *Manager) NewAccount(auth string) (Account, error) {
	key, err := am.keyStore.GenerateNewKey(crand.Reader, auth)
	if err != nil {
		return Account{}, err
	}
	return Account{Address: key.Address}, nil
}

func (am *Manager) AccountByIndex(index int) (Account, error) {
	addrs, err := am.keyStore.GetKeyAddresses()
	if err != nil {
		return Account{}, err
	}
	if index < 0 || index >= len(addrs) {
		return Account{}, fmt.Errorf("account index %d not in range [0, %d]", index, len(addrs)-1)
	}
	return Account{Address: addrs[index]}, nil
}

func (am *Manager) Accounts() []Account {
	addresses, _ := am.keyStore.GetKeyAddresses()
	accounts := make([]Account, len(addresses))
	for i, addr := range addresses {
		accounts[i] = Account{
			Address: addr,
		}
	}
	return accounts
}

// zeroKey zeroes a private key in memory.
func zeroKey(k *ecdsa.PrivateKey) {
	b := k.D.Bits()
	for i := range b {
		b[i] = 0
	}
}

// USE WITH CAUTION = this will save an unencrypted private key on disk
// no cli or js interface
func (am *Manager) Export(path string, a Account, keyAuth string) error {
	key, err := am.keyStore.GetKey(a.Address, keyAuth)
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
	if err := am.keyStore.StoreKey(key, keyAuth); err != nil {
		return Account{}, err
	}
	return Account{Address: key.Address}, nil
}

func (am *Manager) Update(a Account, authFrom, authTo string) (err error) {
	var key *Key
	key, err = am.keyStore.GetKey(a.Address, authFrom)

	if err == nil {
		err = am.keyStore.StoreKey(key, authTo)
		if err == nil {
			am.keyStore.Cleanup(a.Address)
		}
	}
	return
}

func (am *Manager) ImportPreSaleKey(keyJSON []byte, password string) (acc Account, err error) {
	var key *Key
	key, err = importPreSaleKey(am.keyStore, keyJSON, password)
	if err != nil {
		return
	}
	return Account{Address: key.Address}, nil
}
