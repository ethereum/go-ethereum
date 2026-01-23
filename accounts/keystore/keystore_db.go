// Copyright 2024 The go-ethereum Authors
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
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/pebble"
	"github.com/ethereum/go-ethereum/event"
)

// Database key prefixes for the keystore database.
var (
	keyPrefix   = []byte("k") // k<address> -> encrypted keystore JSON
	countKey    = []byte("m:count")
)

// dbKeyStorePassphrase implements the keyStore interface using a database backend.
type dbKeyStorePassphrase struct {
	db      ethdb.KeyValueStore
	scryptN int
	scryptP int
}

// makeKeyDBKey creates the database key for a given address.
func makeKeyDBKey(addr common.Address) []byte {
	return append(keyPrefix, addr.Bytes()...)
}

// GetKey retrieves and decrypts the key from the database.
func (ks *dbKeyStorePassphrase) GetKey(addr common.Address, filename string, auth string) (*Key, error) {
	// In DB mode, filename is ignored - we look up by address
	dbKey := makeKeyDBKey(addr)
	keyjson, err := ks.db.Get(dbKey)
	if err != nil {
		return nil, ErrNoMatch
	}
	key, err := DecryptKey(keyjson, auth)
	if err != nil {
		return nil, err
	}
	// Verify the decrypted key matches the requested address
	if key.Address != addr {
		return nil, fmt.Errorf("key content mismatch: have account %x, want %x", key.Address, addr)
	}
	return key, nil
}

// StoreKey encrypts and stores the key in the database.
func (ks *dbKeyStorePassphrase) StoreKey(filename string, key *Key, auth string) error {
	// In DB mode, filename is derived from address
	keyjson, err := EncryptKey(key, auth, ks.scryptN, ks.scryptP)
	if err != nil {
		return err
	}
	dbKey := makeKeyDBKey(key.Address)

	// Check if this is a new key or an update
	isNew := false
	if has, _ := ks.db.Has(dbKey); !has {
		isNew = true
	}

	// Store the encrypted key
	if err := ks.db.Put(dbKey, keyjson); err != nil {
		return err
	}

	// Update count if this is a new key
	if isNew {
		count := ks.getCount()
		ks.setCount(count + 1)
	}

	return nil
}

// JoinPath returns the "path" for the key (the address in hex).
// In DB mode, this is used as an identifier rather than a file path.
func (ks *dbKeyStorePassphrase) JoinPath(filename string) string {
	// For DB mode, we just return the filename as-is
	// The actual storage uses the address, not the filename
	return filename
}

// DeleteKey removes a key from the database.
func (ks *dbKeyStorePassphrase) DeleteKey(addr common.Address) error {
	dbKey := makeKeyDBKey(addr)

	// Check if the key exists
	if has, _ := ks.db.Has(dbKey); !has {
		return ErrNoMatch
	}

	// Delete the key
	if err := ks.db.Delete(dbKey); err != nil {
		return err
	}

	// Update count
	count := ks.getCount()
	if count > 0 {
		ks.setCount(count - 1)
	}

	return nil
}

// getCount returns the total number of keys in the database.
func (ks *dbKeyStorePassphrase) getCount() uint64 {
	data, err := ks.db.Get(countKey)
	if err != nil {
		return 0
	}
	if len(data) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(data)
}

// setCount updates the total number of keys in the database.
func (ks *dbKeyStorePassphrase) setCount(count uint64) {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, count)
	ks.db.Put(countKey, data)
}

// dbAccountCache implements account caching backed by a database.
type dbAccountCache struct {
	db     ethdb.KeyValueStore
	mu     sync.Mutex
	notify chan struct{}
}

// newDBAccountCache creates a new database-backed account cache.
func newDBAccountCache(db ethdb.KeyValueStore) (*dbAccountCache, chan struct{}) {
	ac := &dbAccountCache{
		db:     db,
		notify: make(chan struct{}, 1),
	}
	return ac, ac.notify
}

// accounts returns all accounts in the database.
func (ac *dbAccountCache) accounts() []accounts.Account {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	var accs []accounts.Account
	iter := ac.db.NewIterator(keyPrefix, nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		if len(key) != len(keyPrefix)+common.AddressLength {
			continue
		}
		addr := common.BytesToAddress(key[len(keyPrefix):])
		accs = append(accs, accounts.Account{
			Address: addr,
			URL:     accounts.URL{Scheme: KeyStoreScheme, Path: addr.Hex()},
		})
	}
	return accs
}

// hasAddress checks if an address exists in the database.
func (ac *dbAccountCache) hasAddress(addr common.Address) bool {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	dbKey := makeKeyDBKey(addr)
	has, _ := ac.db.Has(dbKey)
	return has
}

// add adds an account to the cache (notifies watchers).
func (ac *dbAccountCache) add(newAccount accounts.Account) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// Notify watchers of the change
	select {
	case ac.notify <- struct{}{}:
	default:
	}
}

// delete removes an account from the cache (notifies watchers).
func (ac *dbAccountCache) delete(removed accounts.Account) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// Notify watchers of the change
	select {
	case ac.notify <- struct{}{}:
	default:
	}
}

// find returns the account for a given address.
// Caller must hold ac.mu.
func (ac *dbAccountCache) find(a accounts.Account) (accounts.Account, error) {
	dbKey := makeKeyDBKey(a.Address)
	has, err := ac.db.Has(dbKey)
	if err != nil || !has {
		return accounts.Account{}, ErrNoMatch
	}
	return accounts.Account{
		Address: a.Address,
		URL:     accounts.URL{Scheme: KeyStoreScheme, Path: a.Address.Hex()},
	}, nil
}

// maybeReload is a no-op for DB cache since DB is always up-to-date.
func (ac *dbAccountCache) maybeReload() {}

// close closes the notify channel.
func (ac *dbAccountCache) close() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if ac.notify != nil {
		close(ac.notify)
		ac.notify = nil
	}
}

// DBKeyStore is a keystore backed by a database for scalability.
type DBKeyStore struct {
	db       ethdb.KeyValueStore
	storage  *dbKeyStorePassphrase
	cache    *dbAccountCache
	changes  chan struct{}
	unlocked map[common.Address]*unlocked

	wallets     []accounts.Wallet
	updateFeed  event.Feed
	updateScope event.SubscriptionScope
	updating    bool

	mu       sync.RWMutex
	importMu sync.Mutex
}

// NewDBKeyStore creates a keystore backed by a database for scalability.
// This is designed for use cases with millions of keys where the file-based
// keystore becomes impractical due to filesystem scanning overhead.
func NewDBKeyStore(dbPath string, scryptN, scryptP int) (*DBKeyStore, error) {
	db, err := pebble.New(dbPath, 16, 16, "keystore", false)
	if err != nil {
		return nil, fmt.Errorf("failed to open keystore database: %w", err)
	}

	storage := &dbKeyStorePassphrase{
		db:      db,
		scryptN: scryptN,
		scryptP: scryptP,
	}

	cache, changes := newDBAccountCache(db)

	ks := &DBKeyStore{
		db:       db,
		storage:  storage,
		cache:    cache,
		changes:  changes,
		unlocked: make(map[common.Address]*unlocked),
	}

	// Create the initial list of wallets from the cache
	accs := cache.accounts()
	ks.wallets = make([]accounts.Wallet, len(accs))
	for i := 0; i < len(accs); i++ {
		ks.wallets[i] = &dbKeystoreWallet{account: accs[i], keystore: ks}
	}

	// Set up a cleanup to close the cache (and notify channel) when the keystore is garbage collected
	runtime.AddCleanup(ks, func(c *dbAccountCache) {
		c.close()
	}, cache)

	return ks, nil
}

// Close closes the database.
func (ks *DBKeyStore) Close() error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.cache.close()
	if ks.db != nil {
		return ks.db.Close()
	}
	return nil
}

// Wallets returns all wallets managed by the keystore.
func (ks *DBKeyStore) Wallets() []accounts.Wallet {
	ks.refreshWallets()

	ks.mu.RLock()
	defer ks.mu.RUnlock()

	cpy := make([]accounts.Wallet, len(ks.wallets))
	copy(cpy, ks.wallets)
	return cpy
}

// refreshWallets updates the wallet list from the database.
func (ks *DBKeyStore) refreshWallets() {
	ks.mu.Lock()
	accs := ks.cache.accounts()

	var (
		wallets = make([]accounts.Wallet, 0, len(accs))
		events  []accounts.WalletEvent
	)

	// Build a map of existing wallets for comparison
	existing := make(map[common.Address]accounts.Wallet)
	for _, w := range ks.wallets {
		accts := w.Accounts()
		if len(accts) > 0 {
			existing[accts[0].Address] = w
		}
	}

	// Process accounts from DB
	seen := make(map[common.Address]bool)
	for _, account := range accs {
		seen[account.Address] = true
		if w, ok := existing[account.Address]; ok {
			wallets = append(wallets, w)
		} else {
			wallet := &dbKeystoreWallet{account: account, keystore: ks}
			events = append(events, accounts.WalletEvent{Wallet: wallet, Kind: accounts.WalletArrived})
			wallets = append(wallets, wallet)
		}
	}

	// Find removed wallets
	for addr, w := range existing {
		if !seen[addr] {
			events = append(events, accounts.WalletEvent{Wallet: w, Kind: accounts.WalletDropped})
		}
	}

	ks.wallets = wallets
	ks.mu.Unlock()

	// Fire wallet events
	for _, event := range events {
		ks.updateFeed.Send(event)
	}
}

// Subscribe implements accounts.Backend.
func (ks *DBKeyStore) Subscribe(sink chan<- accounts.WalletEvent) event.Subscription {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	sub := ks.updateScope.Track(ks.updateFeed.Subscribe(sink))

	if !ks.updating {
		ks.updating = true
		go ks.updater()
	}
	return sub
}

// updater listens for account changes and refreshes wallets.
func (ks *DBKeyStore) updater() {
	for {
		select {
		case <-ks.changes:
		}
		ks.refreshWallets()

		ks.mu.Lock()
		if ks.updateScope.Count() == 0 {
			ks.updating = false
			ks.mu.Unlock()
			return
		}
		ks.mu.Unlock()
	}
}

// HasAddress reports whether a key with the given address is present.
func (ks *DBKeyStore) HasAddress(addr common.Address) bool {
	return ks.cache.hasAddress(addr)
}

// Accounts returns all key files present in the keystore.
func (ks *DBKeyStore) Accounts() []accounts.Account {
	return ks.cache.accounts()
}

// NewAccount generates a new key and stores it in the database.
func (ks *DBKeyStore) NewAccount(passphrase string) (accounts.Account, error) {
	_, account, err := storeNewKey(ks.storage, crand.Reader, passphrase)
	if err != nil {
		return accounts.Account{}, err
	}
	ks.cache.add(account)
	ks.refreshWallets()
	return account, nil
}

// Delete deletes the key matched by account if the passphrase is correct.
func (ks *DBKeyStore) Delete(a accounts.Account, passphrase string) error {
	// Decrypt to verify password
	a, key, err := ks.getDecryptedKey(a, passphrase)
	if key != nil {
		zeroKey(key.PrivateKey)
	}
	if err != nil {
		return err
	}

	// Delete from database
	if err := ks.storage.DeleteKey(a.Address); err != nil {
		return err
	}

	ks.cache.delete(a)
	ks.refreshWallets()
	return nil
}

// Find resolves the given account into a unique entry in the keystore.
func (ks *DBKeyStore) Find(a accounts.Account) (accounts.Account, error) {
	ks.cache.mu.Lock()
	a, err := ks.cache.find(a)
	ks.cache.mu.Unlock()
	return a, err
}

// getDecryptedKey retrieves and decrypts the key for the given account.
func (ks *DBKeyStore) getDecryptedKey(a accounts.Account, auth string) (accounts.Account, *Key, error) {
	a, err := ks.Find(a)
	if err != nil {
		return a, nil, err
	}
	key, err := ks.storage.GetKey(a.Address, a.URL.Path, auth)
	return a, key, err
}

// Export exports as a JSON key, encrypted with newPassphrase.
func (ks *DBKeyStore) Export(a accounts.Account, passphrase, newPassphrase string) (keyJSON []byte, err error) {
	_, key, err := ks.getDecryptedKey(a, passphrase)
	if err != nil {
		return nil, err
	}
	defer zeroKey(key.PrivateKey)
	return EncryptKey(key, newPassphrase, ks.storage.scryptN, ks.storage.scryptP)
}

// Import stores the given encrypted JSON key into the database.
func (ks *DBKeyStore) Import(keyJSON []byte, passphrase, newPassphrase string) (accounts.Account, error) {
	key, err := DecryptKey(keyJSON, passphrase)
	if key != nil && key.PrivateKey != nil {
		defer zeroKey(key.PrivateKey)
	}
	if err != nil {
		return accounts.Account{}, err
	}
	ks.importMu.Lock()
	defer ks.importMu.Unlock()

	if ks.cache.hasAddress(key.Address) {
		return accounts.Account{Address: key.Address}, ErrAccountAlreadyExists
	}
	return ks.importKey(key, newPassphrase)
}

// importKey imports a key into the database.
func (ks *DBKeyStore) importKey(key *Key, passphrase string) (accounts.Account, error) {
	a := accounts.Account{
		Address: key.Address,
		URL:     accounts.URL{Scheme: KeyStoreScheme, Path: key.Address.Hex()},
	}
	if err := ks.storage.StoreKey(a.URL.Path, key, passphrase); err != nil {
		return accounts.Account{}, err
	}
	ks.cache.add(a)
	ks.refreshWallets()
	return a, nil
}

// Update changes the passphrase of an existing account.
func (ks *DBKeyStore) Update(a accounts.Account, passphrase, newPassphrase string) error {
	a, key, err := ks.getDecryptedKey(a, passphrase)
	if err != nil {
		return err
	}
	defer zeroKey(key.PrivateKey)
	return ks.storage.StoreKey(a.URL.Path, key, newPassphrase)
}

// Count returns the number of keys in the database.
func (ks *DBKeyStore) Count() uint64 {
	return ks.storage.getCount()
}

// dbKeystoreWallet implements accounts.Wallet for DB-backed keystore.
type dbKeystoreWallet struct {
	account  accounts.Account
	keystore *DBKeyStore
}

// URL implements accounts.Wallet, returning the URL of the account.
func (w *dbKeystoreWallet) URL() accounts.URL {
	return w.account.URL
}

// Status implements accounts.Wallet, returning whether the wallet is locked.
func (w *dbKeystoreWallet) Status() (string, error) {
	w.keystore.mu.RLock()
	defer w.keystore.mu.RUnlock()

	if _, ok := w.keystore.unlocked[w.account.Address]; ok {
		return "Unlocked", nil
	}
	return "Locked", nil
}

// Open implements accounts.Wallet, but is a no-op for DB keystores.
func (w *dbKeystoreWallet) Open(passphrase string) error {
	return nil
}

// Close implements accounts.Wallet, but is a no-op for DB keystores.
func (w *dbKeystoreWallet) Close() error {
	return nil
}

// Accounts implements accounts.Wallet, returning the account this wallet holds.
func (w *dbKeystoreWallet) Accounts() []accounts.Account {
	return []accounts.Account{w.account}
}

// Contains implements accounts.Wallet, returning whether a particular account is or is not part of this wallet.
func (w *dbKeystoreWallet) Contains(account accounts.Account) bool {
	return account.Address == w.account.Address
}

// Derive implements accounts.Wallet, but is a no-op for DB keystores.
func (w *dbKeystoreWallet) Derive(path accounts.DerivationPath, pin bool) (accounts.Account, error) {
	return accounts.Account{}, errors.New("not supported")
}

// SelfDerive implements accounts.Wallet, but is a no-op for DB keystores.
func (w *dbKeystoreWallet) SelfDerive(bases []accounts.DerivationPath, chain ethereum.ChainStateReader) {}

// SignData implements accounts.Wallet.
func (w *dbKeystoreWallet) SignData(account accounts.Account, mimeType string, data []byte) ([]byte, error) {
	return w.signHash(account, crypto.Keccak256(data))
}

// SignDataWithPassphrase implements accounts.Wallet.
func (w *dbKeystoreWallet) SignDataWithPassphrase(account accounts.Account, passphrase, mimeType string, data []byte) ([]byte, error) {
	return w.signHashWithPassphrase(account, passphrase, crypto.Keccak256(data))
}

// SignText implements accounts.Wallet.
func (w *dbKeystoreWallet) SignText(account accounts.Account, text []byte) ([]byte, error) {
	return w.signHash(account, accounts.TextHash(text))
}

// SignTextWithPassphrase implements accounts.Wallet.
func (w *dbKeystoreWallet) SignTextWithPassphrase(account accounts.Account, passphrase string, text []byte) ([]byte, error) {
	return w.signHashWithPassphrase(account, passphrase, accounts.TextHash(text))
}

// signHash signs the given hash with the account.
func (w *dbKeystoreWallet) signHash(account accounts.Account, hash []byte) ([]byte, error) {
	if !w.Contains(account) {
		return nil, accounts.ErrUnknownAccount
	}
	return w.keystore.SignHash(account, hash)
}

// signHashWithPassphrase signs the given hash with the account using the passphrase.
func (w *dbKeystoreWallet) signHashWithPassphrase(account accounts.Account, passphrase string, hash []byte) ([]byte, error) {
	if !w.Contains(account) {
		return nil, accounts.ErrUnknownAccount
	}
	return w.keystore.SignHashWithPassphrase(account, passphrase, hash)
}

// SignHash signs the given hash.
func (ks *DBKeyStore) SignHash(a accounts.Account, hash []byte) ([]byte, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	unlockedKey, found := ks.unlocked[a.Address]
	if !found {
		return nil, ErrLocked
	}
	return signHash(hash, unlockedKey.PrivateKey)
}

// SignHashWithPassphrase signs hash if the private key matching the given address
// can be decrypted with the given passphrase.
func (ks *DBKeyStore) SignHashWithPassphrase(a accounts.Account, passphrase string, hash []byte) (signature []byte, err error) {
	_, key, err := ks.getDecryptedKey(a, passphrase)
	if err != nil {
		return nil, err
	}
	defer zeroKey(key.PrivateKey)
	return crypto.Sign(hash, key.PrivateKey)
}

// signHash is a helper to sign a hash with a private key.
func signHash(hash []byte, priv *ecdsa.PrivateKey) ([]byte, error) {
	return crypto.Sign(hash, priv)
}

// SignTx signs the given transaction with the requested account.
func (ks *DBKeyStore) SignTx(a accounts.Account, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	unlockedKey, found := ks.unlocked[a.Address]
	if !found {
		return nil, ErrLocked
	}
	signer := types.LatestSignerForChainID(chainID)
	return types.SignTx(tx, signer, unlockedKey.PrivateKey)
}

// SignTxWithPassphrase signs the transaction if the private key matching the
// given address can be decrypted with the given passphrase.
func (ks *DBKeyStore) SignTxWithPassphrase(a accounts.Account, passphrase string, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	_, key, err := ks.getDecryptedKey(a, passphrase)
	if err != nil {
		return nil, err
	}
	defer zeroKey(key.PrivateKey)
	signer := types.LatestSignerForChainID(chainID)
	return types.SignTx(tx, signer, key.PrivateKey)
}

// Unlock unlocks the given account indefinitely.
func (ks *DBKeyStore) Unlock(a accounts.Account, passphrase string) error {
	return ks.TimedUnlock(a, passphrase, 0)
}

// Lock removes the private key with the given address from memory.
func (ks *DBKeyStore) Lock(addr common.Address) error {
	ks.mu.Lock()
	unl, found := ks.unlocked[addr]
	ks.mu.Unlock()
	if found {
		ks.expire(addr, unl, time.Duration(0)*time.Nanosecond)
	}
	return nil
}

// TimedUnlock unlocks the given account with the passphrase. The account
// stays unlocked for the duration of timeout. A timeout of 0 unlocks the account
// until the program exits.
func (ks *DBKeyStore) TimedUnlock(a accounts.Account, passphrase string, timeout time.Duration) error {
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

// expire removes an unlocked key after the timeout.
func (ks *DBKeyStore) expire(addr common.Address, u *unlocked, timeout time.Duration) {
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-u.abort:
		// just quit
	case <-t.C:
		ks.mu.Lock()
		if ks.unlocked[addr] == u {
			zeroKey(u.PrivateKey)
			delete(ks.unlocked, addr)
		}
		ks.mu.Unlock()
	}
}

// SignTx for dbKeystoreWallet implements accounts.Wallet.
func (w *dbKeystoreWallet) SignTx(account accounts.Account, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	if !w.Contains(account) {
		return nil, accounts.ErrUnknownAccount
	}
	return w.keystore.SignTx(account, tx, chainID)
}

// SignTxWithPassphrase for dbKeystoreWallet implements accounts.Wallet.
func (w *dbKeystoreWallet) SignTxWithPassphrase(account accounts.Account, passphrase string, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	if !w.Contains(account) {
		return nil, accounts.ErrUnknownAccount
	}
	return w.keystore.SignTxWithPassphrase(account, passphrase, tx, chainID)
}

// ImportECDSA stores the given key into the database, encrypting it with the passphrase.
func (ks *DBKeyStore) ImportECDSA(priv *ecdsa.PrivateKey, passphrase string) (accounts.Account, error) {
	ks.importMu.Lock()
	defer ks.importMu.Unlock()

	key := newKeyFromECDSA(priv)
	if ks.cache.hasAddress(key.Address) {
		return accounts.Account{Address: key.Address}, ErrAccountAlreadyExists
	}
	return ks.importKey(key, passphrase)
}
