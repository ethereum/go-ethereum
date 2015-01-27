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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
)

// TODO: better name for this struct?
type UserAccount struct {
	Addr []byte
}

type AccountManager struct {
	keyStore crypto.KeyStore2
}

// TODO: get key by addr - modify KeyStore2 GetKey to work with addr

// TODO: pass through passphrase for APIs which require access to private key?
func NewAccountManager(keyStore crypto.KeyStore2) AccountManager {
	am := &AccountManager{
		keyStore: keyStore,
	}
	return *am
}

func (am *AccountManager) Sign(fromAddr []byte, keyAuth string, toSign []byte) (signature []byte, err error) {
	key, err := am.keyStore.GetKey(fromAddr, keyAuth)
	if err != nil {
		return nil, err
	}
	privKey := crypto.FromECDSA(key.PrivateKey)
	// TODO: what is second value?
	signature, err = secp256k1.Sign(toSign, privKey)
	return signature, err
}

func (am AccountManager) NewAccount(auth string) (*UserAccount, error) {
	key, err := am.keyStore.GenerateNewKey(crand.Reader, auth)
	if err != nil {
		return nil, err
	}
	ua := &UserAccount{
		Addr: key.Address,
	}
	return ua, err
}

// set of accounts == set of keys in given key store
// TODO: do we need persistence of accounts as well?
func (am *AccountManager) Accounts() ([]UserAccount, error) {
	addresses, err := am.keyStore.GetKeyAddresses()
	if err != nil {
		return nil, err
	}

	accounts := make([]UserAccount, len(addresses))

	for i, addr := range addresses {
		ua := &UserAccount{
			Addr: addr,
		}
		accounts[i] = *ua
	}
	return accounts, err
}
