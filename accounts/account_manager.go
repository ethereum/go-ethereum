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
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/syndtr/goleveldb/leveldb"
)

/*
type Account struct {
	Address           []byte
	AssociatedAddress []byte // e.g. a wallet contract "owned" by the account
}
*/

type AccountManager struct {
	keyStore   crypto.KeyStore2
	accountsDb *ethdb.LDBDatabase
}

// TODO: get key by addr - modify KeyStore2 GetKey to work with addr

// TODO: pass through passphrase for APIs which require access to private key?
func NewAccountManager(keyStore crypto.KeyStore2) (AccountManager, error) {
	db, err := ethdb.NewLDBDatabase("accounts")
	if err != nil {
		panic(err)
	}
	am := &AccountManager{
		keyStore:   keyStore,
		accountsDb: db,
	}
	return *am, nil
}

func (am AccountManager) NewAccount(auth string) ([]byte, error) {
	newKey, err := am.keyStore.GenerateNewKey(crand.Reader, auth)
	if err != nil {
		return nil, err
	}
	return newKey.Address, err
}

func (am AccountManager) AssociateContract(ownerAddr []byte, associateAddr []byte) error {
	am.accountsDb.Put(ownerAddr, associateAddr)
	am.accountsDb.Put(associateAddr, ownerAddr)
	return nil
}

func (am *AccountManager) GetAssociatedAddr(addr []byte) ([]byte, error) {
	res, err := am.accountsDb.Get(addr)
	if err == leveldb.ErrNotFound {
		return []byte{}, nil
	} else {
		return res, nil
	}
}

func (am *AccountManager) Sign(fromAddr []byte, keyAuth string, toSign []byte) ([]byte, error) {
	key, err := am.keyStore.GetKey(fromAddr, keyAuth)
	if err != nil {
		return nil, err
	}
	signature, err := crypto.Sign(toSign, key.PrivateKey)
	return signature, err
}
