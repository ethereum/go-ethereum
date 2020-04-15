// Copyright 2014 The go-ethereum Authors
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

package keystore

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/cmd/clef/dbutil"
	"github.com/ethereum/go-ethereum/common"
)

type keyStorePassphraseDB struct {
	db      *dbutil.KVStore
	scryptN int
	scryptP int

	// skipKeyFileVerification disables the security-feature which does
	// reads and decrypts any newly created keyfiles. This should be 'false' in all
	// cases except tests -- setting this to 'true' is not recommended.
	skipKeyFileVerification bool
}

func (ks keyStorePassphraseDB) GetKey(addr common.Address, auth string) (*Key, error) {
	keyjson, err := ks.db.Get(addr.Hex())
	if err != nil {
		return nil, err
	}

	key, err := DecryptKey([]byte(keyjson), auth)
	if err != nil {
		return nil, err
	}
	// Make sure we're really operating on the requested key (no swap attacks)
	if key.Address != addr {
		return nil, fmt.Errorf("key content mismatch: have account %x, want %x", key.Address, addr)
	}
	return key, nil
}

func (ks keyStorePassphraseDB) StoreKey(key *Key, auth string) error {
	keyjson, err := EncryptKey(key, auth, ks.scryptN, ks.scryptP)
	if err != nil {
		return err
	}
	// Write into database
	err = ks.db.Put(key.Address.Hex(), string(keyjson))
	if err != nil {
		return err
	}
	if !ks.skipKeyFileVerification {
		// Verify that we can decrypt the file with the given password.
		_, err = ks.GetKey(key.Address, auth)
		if err != nil {
			msg := "An error was encountered when saving and verifying the keystore file. \n" +
				"This indicates that the keystore is corrupted. \n" +
				"The corrupted key is stored at \n%v\n" +
				"Please file a ticket at:\n\n" +
				"https://github.com/ethereum/go-ethereum/issues." +
				"The error was : %s"
			//lint:ignore ST1005 This is a message for the user
			return fmt.Errorf(msg, ks.db.Conf.DataSourceName(), err)
		}
	}
	return nil
}

// JoinPath returns Path (custom database related) for creating accounts.Account
func (ks keyStorePassphraseDB) JoinPath(key string) string {
	return ks.db.Conf.Adapter + "/" + ks.db.Table + "/" + key
}

func (ks keyStorePassphraseDB) Exists(addr common.Address) bool {
	return ks.db.Exists(addr.Hex())
}

func (ks keyStorePassphraseDB) All() []accounts.Account {
	keys := ks.db.All()
	accs := make([]accounts.Account, len(keys))
	for idx, key := range keys {
		accs[idx] = accounts.Account{
			Address: common.HexToAddress(key),
			URL:     accounts.URL{Scheme: KeyStoreScheme, Path: ks.JoinPath(key)},
		}
	}
	return accs
}

// Find returns the account with the correct URL for database backed account
func (ks keyStorePassphraseDB) Find(a accounts.Account) (accounts.Account, error) {
	found := ks.Exists(a.Address)
	if found {
		return accounts.Account{
			Address: a.Address,
			URL:     accounts.URL{Scheme: KeyStoreScheme, Path: ks.JoinPath(a.Address.Hex())},
		}, nil
	}
	return accounts.Account{}, ErrNoMatch
}

func (ks keyStorePassphraseDB) Size() int {
	return ks.db.Size()
}
