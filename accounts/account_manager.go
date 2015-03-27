/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU Lesser General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU Lesser General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors
 * 	Gustav Simonsson <gustav.simonsson@gmail.com>
 * @date 2015
 *
 */
/*

This abstracts part of a user's interaction with an account she controls.
It's not an abstraction of core Ethereum accounts data type / logic -
for that see the core processing code of blocks / txs.

Currently this is pretty much a passthrough to the KeyStore2 interface,
and accounts persistence is derived from stored keys' addresses

*/
package accounts

import (
	"bytes"
	"crypto/ecdsa"
	crand "crypto/rand"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ErrLocked = errors.New("account is locked")
	ErrNoKeys = errors.New("no keys in store")
)

type Account struct {
	Address []byte
}

type Manager struct {
	keyStore crypto.KeyStore2
	unlocked map[string]*unlocked
	mutex    sync.RWMutex
}

type unlocked struct {
	*crypto.Key
	abort chan struct{}
}

func NewManager(keyStore crypto.KeyStore2) *Manager {
	return &Manager{
		keyStore: keyStore,
		unlocked: make(map[string]*unlocked),
	}
}

func (am *Manager) HasAccount(addr []byte) bool {
	accounts, _ := am.Accounts()
	for _, acct := range accounts {
		if bytes.Compare(acct.Address, addr) == 0 {
			return true
		}
	}
	return false
}

func (am *Manager) Primary() (addr []byte, err error) {
	addrs, err := am.keyStore.GetKeyAddresses()
	if os.IsNotExist(err) {
		return nil, ErrNoKeys
	} else if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, ErrNoKeys
	}
	return addrs[0], nil
}

func (am *Manager) DeleteAccount(address []byte, auth string) error {
	return am.keyStore.DeleteKey(address, auth)
}

func (am *Manager) Sign(a Account, toSign []byte) (signature []byte, err error) {
	am.mutex.RLock()
	unlockedKey, found := am.unlocked[string(a.Address)]
	am.mutex.RUnlock()
	if !found {
		return nil, ErrLocked
	}
	signature, err = crypto.Sign(toSign, unlockedKey.PrivateKey)
	return signature, err
}

// TimedUnlock unlocks the account with the given address.
// When timeout has passed, the account will be locked again.
func (am *Manager) TimedUnlock(addr []byte, keyAuth string, timeout time.Duration) error {
	key, err := am.keyStore.GetKey(addr, keyAuth)
	if err != nil {
		return err
	}
	u := am.addUnlocked(addr, key)
	go am.dropLater(addr, u, timeout)
	return nil
}

// Unlock unlocks the account with the given address. The account
// stays unlocked until the program exits or until a TimedUnlock
// timeout (started after the call to Unlock) expires.
func (am *Manager) Unlock(addr []byte, keyAuth string) error {
	key, err := am.keyStore.GetKey(addr, keyAuth)
	if err != nil {
		return err
	}
	am.addUnlocked(addr, key)
	return nil
}

func (am *Manager) NewAccount(auth string) (Account, error) {
	key, err := am.keyStore.GenerateNewKey(crand.Reader, auth)
	if err != nil {
		return Account{}, err
	}
	return Account{Address: key.Address}, nil
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

func (am *Manager) addUnlocked(addr []byte, key *crypto.Key) *unlocked {
	u := &unlocked{Key: key, abort: make(chan struct{})}
	am.mutex.Lock()
	prev, found := am.unlocked[string(addr)]
	if found {
		// terminate dropLater for this key to avoid unexpected drops.
		close(prev.abort)
		// the key is zeroed here instead of in dropLater because
		// there might not actually be a dropLater running for this
		// key, i.e. when Unlock was used.
		zeroKey(prev.PrivateKey)
	}
	am.unlocked[string(addr)] = u
	am.mutex.Unlock()
	return u
}

func (am *Manager) dropLater(addr []byte, u *unlocked, timeout time.Duration) {
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
		if am.unlocked[string(addr)] == u {
			zeroKey(u.PrivateKey)
			delete(am.unlocked, string(addr))
		}
		am.mutex.Unlock()
	}
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
func (am *Manager) Export(path string, addr []byte, keyAuth string) error {
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

func (am *Manager) ImportPreSaleKey(keyJSON []byte, password string) (acc Account, err error) {
	var key *crypto.Key
	key, err = crypto.ImportPreSaleKey(am.keyStore, keyJSON, password)
	if err != nil {
		return
	}
	if err = am.keyStore.StoreKey(key, password); err != nil {
		return
	}
	return Account{Address: key.Address}, nil
}
