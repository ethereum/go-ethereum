// Copyright 2017 The go-ethereum Authors
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
	"crypto/ecdsa"
	crand "crypto/rand"
	"fmt"
	"math/big"
	"reflect"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/pborman/uuid"
)

// DBKeyStoreType is the reflect type of a keystore backend.
var DBKeyStoreType = reflect.TypeOf(&keyStoreDB{})

type keyStoreDB struct {
	storage  *keyStorePassphraseDB        // storage backend, might be mysql or postgres
	unlocked map[common.Address]*unlocked // Currently unlocked account (decrypted private keys)

	mu sync.RWMutex
}

// Wallets implements accounts.Backend, returning all single-key wallets from the KeyStore.
func (ks *keyStoreDB) Wallets() []accounts.Wallet {
	accs := ks.storage.All()
	wallets := make([]accounts.Wallet, len(accs))
	for idx, acc := range accs {
		wallets[idx] = &keystoreWalletDB{account: acc, keystore: ks}
	}
	return wallets
}

// Subscribe implements accounts.Backend, creating an async subscription to
// receive notifications on the addition or removal of KeyStore wallets.
func (ks *keyStoreDB) Subscribe(sink chan<- accounts.WalletEvent) event.Subscription {
	// Since this is a database backend, we don't need a in-memory cache to hold all the wallets
	// so notifications on actions of wallets can be ignored
	return nil
}

// HasAddress reports whether a key with the given address is present.
func (ks *keyStoreDB) HasAddress(addr common.Address) bool {
	return ks.storage.Exists(addr)
}

// Accounts returns all key files present in the KeyStore.
func (ks *keyStoreDB) Accounts() []accounts.Account {
	return ks.storage.All()
}

// Delete deletes the key matched by account if the passphrase is correct.
// If the account contains no filename, the address must match a unique key.
func (ks *keyStoreDB) Delete(a accounts.Account, passphrase string) error {
	// Decrypting the key isn't really necessary, but we do
	// it anyway to check the password and zero out the key
	// immediately afterwards.
	a, key, err := ks.getDecryptedKey(a, passphrase)
	if key != nil {
		zeroKey(key.PrivateKey)
	}
	if err != nil {
		return err
	}
	ks.storage.db.Del(a.Address.Hex())
	return nil
}

// SignHash calculates an ECDSA signature for the given hash. The produced
// signature is in the [R || S || V] format where V is 0 or 1.
func (ks *keyStoreDB) SignHash(a accounts.Account, hash []byte) ([]byte, error) {
	// Look up the key to sign with and abort if it cannot be found
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	unlockedKey, found := ks.unlocked[a.Address]
	if !found {
		return nil, ErrLocked
	}
	// Sign the hash using plain ECDSA operations
	return crypto.Sign(hash, unlockedKey.PrivateKey)
}

// SignTx signs the given transaction with the requested account.
func (ks *keyStoreDB) SignTx(a accounts.Account, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	// Look up the key to sign with and abort if it cannot be found
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	unlockedKey, found := ks.unlocked[a.Address]
	if !found {
		return nil, ErrLocked
	}
	// Depending on the presence of the chain ID, sign with EIP155 or homestead
	if chainID != nil {
		return types.SignTx(tx, types.NewEIP155Signer(chainID), unlockedKey.PrivateKey)
	}
	return types.SignTx(tx, types.HomesteadSigner{}, unlockedKey.PrivateKey)
}

// SignHashWithPassphrase signs hash if the private key matching the given address
// can be decrypted with the given passphrase. The produced signature is in the
// [R || S || V] format where V is 0 or 1.
func (ks *keyStoreDB) SignHashWithPassphrase(a accounts.Account, passphrase string, hash []byte) (signature []byte, err error) {
	_, key, err := ks.getDecryptedKey(a, passphrase)
	if err != nil {
		return nil, err
	}
	defer zeroKey(key.PrivateKey)
	return crypto.Sign(hash, key.PrivateKey)
}

// SignTxWithPassphrase signs the transaction if the private key matching the
// given address can be decrypted with the given passphrase.
func (ks *keyStoreDB) SignTxWithPassphrase(a accounts.Account, passphrase string, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	_, key, err := ks.getDecryptedKey(a, passphrase)
	if err != nil {
		return nil, err
	}
	defer zeroKey(key.PrivateKey)

	// Depending on the presence of the chain ID, sign with EIP155 or homestead
	if chainID != nil {
		return types.SignTx(tx, types.NewEIP155Signer(chainID), key.PrivateKey)
	}
	return types.SignTx(tx, types.HomesteadSigner{}, key.PrivateKey)
}

// Unlock unlocks the given account indefinitely.
func (ks *keyStoreDB) Unlock(a accounts.Account, passphrase string) error {
	return ks.TimedUnlock(a, passphrase, 0)
}

// Lock removes the private key with the given address from memory.
func (ks *keyStoreDB) Lock(addr common.Address) error {
	ks.mu.Lock()
	if unl, found := ks.unlocked[addr]; found {
		ks.mu.Unlock()
		ks.expire(addr, unl, time.Duration(0)*time.Nanosecond)
	} else {
		ks.mu.Unlock()
	}
	return nil
}

// TimedUnlock unlocks the given account with the passphrase. The account
// stays unlocked for the duration of timeout. A timeout of 0 unlocks the account
// until the program exits. The account must match a unique key file.
//
// If the account address is already unlocked for a duration, TimedUnlock extends or
// shortens the active unlock timeout. If the address was previously unlocked
// indefinitely the timeout is not altered.
func (ks *keyStoreDB) TimedUnlock(a accounts.Account, passphrase string, timeout time.Duration) error {
	a, key, err := ks.getDecryptedKey(a, passphrase)
	if err != nil {
		return err
	}

	ks.mu.Lock()
	defer ks.mu.Unlock()

	u, found := ks.unlocked[a.Address]
	if found {
		if u.abort == nil {
			// The address was unlocked indefinitely, so unlocking
			// it with a timeout would be confusing.
			zeroKey(key.PrivateKey)
			return nil
		}
		// Terminate the expire goroutine and replace it below.
		close(u.abort)
	}
	if timeout > 0 {
		u = &unlocked{Key: key, abort: make(chan struct{})}
		go ks.expire(a.Address, u, timeout)
	} else {
		u = &unlocked{Key: key}
	}
	ks.unlocked[a.Address] = u
	return nil
}

func (ks *keyStoreDB) expire(addr common.Address, u *unlocked, timeout time.Duration) {
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-u.abort:
		// just quit
	case <-t.C:
		ks.mu.Lock()
		// only drop if it's still the same key instance that dropLater
		// was launched with. we can check that using pointer equality
		// because the map stores a new pointer every time the key is
		// unlocked.
		if ks.unlocked[addr] == u {
			zeroKey(u.PrivateKey)
			delete(ks.unlocked, addr)
		}
		ks.mu.Unlock()
	}
}

// Find resolves the given account into a unique entry in the KeyStore.
func (ks *keyStoreDB) Find(a accounts.Account) (accounts.Account, error) {
	return ks.storage.Find(a)
}

// NewAccount generates a new key and stores it into the KeyStore,
// encrypting it with the passphrase.
func (ks *keyStoreDB) NewAccount(passphrase string) (accounts.Account, error) {
	key, err := newKey(crand.Reader)
	if err != nil {
		return accounts.Account{}, err
	}
	a := accounts.Account{
		Address: key.Address,
		URL:     accounts.URL{Scheme: KeyStoreScheme, Path: ks.storage.JoinPath(key.Address.Hex())},
	}
	if err := ks.storage.StoreKey(key, passphrase); err != nil {
		zeroKey(key.PrivateKey)
		return accounts.Account{}, err
	}
	return a, err
}

// Export exports as a JSON key, encrypted with newPassphrase.
func (ks *keyStoreDB) Export(a accounts.Account, passphrase, newPassphrase string) (keyJSON []byte, err error) {
	_, key, err := ks.getDecryptedKey(a, passphrase)
	if err != nil {
		return nil, err
	}
	N, P := ks.storage.scryptN, ks.storage.scryptP
	return EncryptKey(key, newPassphrase, N, P)
}

// Import stores the given encrypted JSON key into the KeyStore.
func (ks *keyStoreDB) Import(keyJSON []byte, passphrase, newPassphrase string) (accounts.Account, error) {
	key, err := DecryptKey(keyJSON, passphrase)
	if key != nil && key.PrivateKey != nil {
		defer zeroKey(key.PrivateKey)
	}
	if err != nil {
		return accounts.Account{}, err
	}
	if ks.storage.Exists(key.Address) {
		return accounts.Account{}, fmt.Errorf("account already exists")
	}
	return ks.importKey(key, newPassphrase)
}

// ImportECDSA stores the given key into the KeyStore, encrypting it with the passphrase.
func (ks *keyStoreDB) ImportECDSA(priv *ecdsa.PrivateKey, passphrase string) (accounts.Account, error) {
	key := newKeyFromECDSA(priv)
	if ks.storage.Exists(key.Address) {
		return accounts.Account{}, fmt.Errorf("account already exists")
	}
	return ks.importKey(key, passphrase)
}

func (ks *keyStoreDB) importKey(key *Key, passphrase string) (accounts.Account, error) {
	a := accounts.Account{
		Address: key.Address,
		URL:     accounts.URL{Scheme: KeyStoreScheme, Path: ks.storage.JoinPath(key.Address.Hex())},
	}
	if err := ks.storage.StoreKey(key, passphrase); err != nil {
		return accounts.Account{}, err
	}
	return a, nil
}

// Update changes the passphrase of an existing account.
func (ks *keyStoreDB) Update(a accounts.Account, passphrase, newPassphrase string) error {
	_, key, err := ks.getDecryptedKey(a, passphrase)
	if err != nil {
		return err
	}
	return ks.storage.StoreKey(key, newPassphrase)
}

// ImportPreSaleKey decrypts the given Ethereum presale wallet and stores
// a key file in the KeyStore. The key file is encrypted with the same passphrase.
func (ks *keyStoreDB) ImportPreSaleKey(keyJSON []byte, passphrase string) (accounts.Account, error) {
	key, err := decryptPreSaleKey(keyJSON, passphrase)
	if err != nil {
		return accounts.Account{}, nil
	}
	key.Id = uuid.NewRandom()
	if err := ks.storage.StoreKey(key, passphrase); err != nil {
		return accounts.Account{}, err
	}
	a := accounts.Account{
		Address: key.Address,
		URL:     accounts.URL{Scheme: KeyStoreScheme, Path: ks.storage.JoinPath(key.Address.Hex())},
	}
	return a, nil
}

func (ks *keyStoreDB) getDecryptedKey(a accounts.Account, auth string) (accounts.Account, *Key, error) {
	a, err := ks.Find(a)
	if err != nil {
		return a, nil, err
	}
	key, err := ks.storage.GetKey(a.Address, auth)
	return a, key, err
}
