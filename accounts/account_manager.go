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
	crand "crypto/rand"
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
	"sync"
	"time"
)

var ErrLocked = errors.New("account is locked; please request passphrase")

// TODO: better name for this struct?
type Account struct {
	Address []byte
}

type AccountManager struct {
	keyStore           crypto.KeyStore2
	unlockedKeys       map[string]crypto.Key
	unlockMilliseconds time.Duration
	mutex              sync.RWMutex
}

func NewAccountManager(keyStore crypto.KeyStore2, unlockMilliseconds time.Duration) AccountManager {
	keysMap := make(map[string]crypto.Key)
	am := &AccountManager{
		keyStore:           keyStore,
		unlockedKeys:       keysMap,
		unlockMilliseconds: unlockMilliseconds,
		mutex:              sync.RWMutex{}, // for accessing unlockedKeys map
	}
	return *am
}

func (am AccountManager) DeleteAccount(address []byte, auth string) error {
	return am.keyStore.DeleteKey(address, auth)
}

func (am *AccountManager) Sign(fromAccount *Account, toSign []byte) (signature []byte, err error) {
	am.mutex.RLock()
	unlockedKey := am.unlockedKeys[string(fromAccount.Address)]
	am.mutex.RUnlock()
	if unlockedKey.Address == nil {
		return nil, ErrLocked
	}
	signature, err = crypto.Sign(toSign, unlockedKey.PrivateKey)
	return signature, err
}

func (am *AccountManager) SignLocked(fromAccount *Account, keyAuth string, toSign []byte) (signature []byte, err error) {
	key, err := am.keyStore.GetKey(fromAccount.Address, keyAuth)
	if err != nil {
		return nil, err
	}
	am.mutex.RLock()
	am.unlockedKeys[string(fromAccount.Address)] = *key
	am.mutex.RUnlock()
	go unlockLater(am, fromAccount.Address)
	signature, err = crypto.Sign(toSign, key.PrivateKey)
	return signature, err
}

func (am AccountManager) NewAccount(auth string) (*Account, error) {
	key, err := am.keyStore.GenerateNewKey(crand.Reader, auth)
	if err != nil {
		return nil, err
	}
	ua := &Account{
		Address: key.Address,
	}
	return ua, err
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

func unlockLater(am *AccountManager, addr []byte) {
	select {
	case <-time.After(time.Millisecond * am.unlockMilliseconds):
	}
	am.mutex.RLock()
	// TODO: how do we know the key is actually gone from memory?
	delete(am.unlockedKeys, string(addr))
	am.mutex.RUnlock()
}
