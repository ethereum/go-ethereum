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

// Currently this is pretty much a passthrough to the KeyStore interface,
// and accounts persistence is derived from stored keys' addresses

import (
	"crypto/ecdsa"
	crand "crypto/rand"
	"errors"
	"fmt"
	"os"
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

type Manager struct {
	keyStore crypto.KeyStore
	unlocked map[common.Address]*unlocked
	mutex    sync.RWMutex
}

type unlocked struct {
	*crypto.Key
	abort chan struct{}
}

func NewManager(keyStore crypto.KeyStore) *Manager {
	return &Manager{
		keyStore: keyStore,
		unlocked: make(map[common.Address]*unlocked),
	}
}

func (am *Manager) HasAccount(addr common.Address) bool {
	accounts, _ := am.Accounts()
	for _, acct := range accounts {
		if acct.Address == addr {
			return true
		}
	}
	return false
}

func (am *Manager) DeleteAccount(address common.Address, auth string) error {
	return am.keyStore.DeleteKey(address, auth)
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
func (am *Manager) Unlock(addr common.Address, keyAuth string) error {
	return am.TimedUnlock(addr, keyAuth, 0)
}

// TimedUnlock unlocks the account with the given address. The account
// stays unlocked for the duration of timeout. A timeout of 0 unlocks the account
// until the program exits.
//
// If the accout is already unlocked, TimedUnlock extends or shortens
// the active unlock timeout.
func (am *Manager) TimedUnlock(addr common.Address, keyAuth string, timeout time.Duration) error {
	key, err := am.keyStore.GetKey(addr, keyAuth)
	if err != nil {
		return err
	}
	var u *unlocked
	am.mutex.Lock()
	defer am.mutex.Unlock()
	var found bool
	u, found = am.unlocked[addr]
	if found {
		// terminate dropLater for this key to avoid unexpected drops.
		if u.abort != nil {
			close(u.abort)
		}
	}
	if timeout > 0 {
		u = &unlocked{Key: key, abort: make(chan struct{})}
		go am.expire(addr, u, timeout)
	} else {
		u = &unlocked{Key: key}
	}
	am.unlocked[addr] = u
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

func (am *Manager) AddressByIndex(index int) (addr string, err error) {
	var addrs []common.Address
	addrs, err = am.keyStore.GetKeyAddresses()
	if err != nil {
		return
	}
	if index < 0 || index >= len(addrs) {
		err = fmt.Errorf("index out of range: %d (should be 0-%d)", index, len(addrs)-1)
	} else {
		addr = addrs[index].Hex()
	}
	return
}

func (am *Manager) Accounts() ([]Account, error) {
	addresses, err := am.keyStore.GetKeyAddresses()
	if os.IsNotExist(err) {
		return nil, ErrNoKeys
	} else if err != nil {
		return nil, err
	}
	accounts := make([]Account, len(addresses))
	for i, addr := range addresses {
		accounts[i] = Account{
			Address: addr,
		}
	}
	return accounts, err
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
func (am *Manager) Export(path string, addr common.Address, keyAuth string) error {
	key, err := am.keyStore.GetKey(addr, keyAuth)
	if err != nil {
		return err
	}
	return crypto.SaveECDSA(path, key.PrivateKey)
}

func (am *Manager) Import(path string, keyAuth string) (Account, error) {
	privateKeyECDSA, err := crypto.LoadECDSA(path)
	if err != nil {
		return Account{}, err
	}
	key := crypto.NewKeyFromECDSA(privateKeyECDSA)
	if err = am.keyStore.StoreKey(key, keyAuth); err != nil {
		return Account{}, err
	}
	return Account{Address: key.Address}, nil
}

func (am *Manager) Update(addr common.Address, authFrom, authTo string) (err error) {
	var key *crypto.Key
	key, err = am.keyStore.GetKey(addr, authFrom)

	if err == nil {
		err = am.keyStore.StoreKey(key, authTo)
		if err == nil {
			am.keyStore.Cleanup(addr)
		}
	}
	return
}

func (am *Manager) ImportPreSaleKey(keyJSON []byte, password string) (acc Account, err error) {
	var key *crypto.Key
	key, err = crypto.ImportPreSaleKey(am.keyStore, keyJSON, password)
	if err != nil {
		return
	}
	return Account{Address: key.Address}, nil
}
