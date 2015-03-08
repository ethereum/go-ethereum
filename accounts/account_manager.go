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
	"crypto/ecdsa"
	crand "crypto/rand"

	"errors"
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

type AccountManager struct {
	keyStore   crypto.KeyStore2
	unlocked   map[string]*unlocked
	unlockTime time.Duration
	mutex      sync.RWMutex
}

type unlocked struct {
	addr  []byte
	abort chan struct{}

	*crypto.Key
}

func NewAccountManager(keyStore crypto.KeyStore2, unlockTime time.Duration) *AccountManager {
	return &AccountManager{
		keyStore:   keyStore,
		unlocked:   make(map[string]*unlocked),
		unlockTime: unlockTime,
	}
}

// Coinbase returns the account address that mining rewards are sent to.
func (am *AccountManager) Coinbase() (addr []byte, err error) {
	// TODO: persist coinbase address on disk
	return am.firstAddr()
}

// MainAccount returns the primary account used for transactions.
func (am *AccountManager) Default() (Account, error) {
	// TODO: persist main account address on disk
	addr, err := am.firstAddr()
	return Account{Address: addr}, err
}

func (am *AccountManager) firstAddr() ([]byte, error) {
	addrs, err := am.keyStore.GetKeyAddresses()
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, ErrNoKeys
	}
	return addrs[0], nil
}

func (am *AccountManager) DeleteAccount(address []byte, auth string) error {
	return am.keyStore.DeleteKey(address, auth)
}

func (am *AccountManager) Sign(a Account, toSign []byte) (signature []byte, err error) {
	am.mutex.RLock()
	unlockedKey, found := am.unlocked[string(a.Address)]
	am.mutex.RUnlock()
	if !found {
		return nil, ErrLocked
	}
	signature, err = crypto.Sign(toSign, unlockedKey.PrivateKey)
	return signature, err
}

func (am *AccountManager) SignLocked(a Account, keyAuth string, toSign []byte) (signature []byte, err error) {
	key, err := am.keyStore.GetKey(a.Address, keyAuth)
	if err != nil {
		return nil, err
	}
	u := am.addUnlocked(a.Address, key)
	go am.dropLater(u)
	signature, err = crypto.Sign(toSign, key.PrivateKey)
	return signature, err
}

func (am *AccountManager) NewAccount(auth string) (Account, error) {
	key, err := am.keyStore.GenerateNewKey(crand.Reader, auth)
	if err != nil {
		return Account{}, err
	}
	return Account{Address: key.Address}, nil
}

func (am *AccountManager) Accounts() ([]Account, error) {
	addresses, err := am.keyStore.GetKeyAddresses()
	if err != nil {
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

func (am *AccountManager) addUnlocked(addr []byte, key *crypto.Key) *unlocked {
	u := &unlocked{addr: addr, abort: make(chan struct{}), Key: key}
	am.mutex.Lock()
	prev, found := am.unlocked[string(addr)]
	if found {
		// terminate dropLater for this key to avoid unexpected drops.
		close(prev.abort)
		zeroKey(prev.PrivateKey)
	}
	am.unlocked[string(addr)] = u
	am.mutex.Unlock()
	return u
}

func (am *AccountManager) dropLater(u *unlocked) {
	t := time.NewTimer(am.unlockTime)
	defer t.Stop()
	select {
	case <-u.abort:
		// just quit
	case <-t.C:
		am.mutex.Lock()
		if am.unlocked[string(u.addr)] == u {
			zeroKey(u.PrivateKey)
			delete(am.unlocked, string(u.addr))
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
