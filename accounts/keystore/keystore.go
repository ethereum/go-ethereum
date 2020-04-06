package keystore

import (
	"crypto/ecdsa"
	"errors"
	"math/big"
	"path/filepath"
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/cmd/clef/dbutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

var (
	ErrLocked  = accounts.NewAuthNeededError("password or unlock")
	ErrNoMatch = errors.New("no key for given address or file")
	ErrDecrypt = errors.New("could not decrypt key with given password")
)

// KeyStoreScheme is the protocol scheme prefixing account and wallet URLs.
const KeyStoreScheme = "keystore"

const keystoreDBTableName = "keystore"

// DBKeyStoreType is the reflect type of a keystore backend.
var DBKeyStoreType = reflect.TypeOf(&keyStoreDB{})

// FSKeyStoreType is the reflect type of a keystore backend.
var FSKeyStoreType = reflect.TypeOf(&keyStoreFS{})

// KeyStore is the interface which abstracts all needed operations required
type KeyStore interface {
	// Wallets implements accounts.Backend, returning all single-key wallets from the KeyStore.
	Wallets() []accounts.Wallet

	// Subscribe implements accounts.Backend, creating an async subscription to
	// receive notifications on the addition or removal of KeyStore wallets.
	Subscribe(sink chan<- accounts.WalletEvent) event.Subscription

	// HasAddress reports whether a key with the given address is present.
	HasAddress(addr common.Address) bool

	// Accounts returns all key files present in the KeyStore.
	Accounts() []accounts.Account

	// Delete deletes the key matched by account if the passphrase is correct.
	// If the account contains no filename, the address must match a unique key.
	Delete(a accounts.Account, passphrase string) error

	// SignHash calculates an ECDSA signature for the given hash. The produced
	// signature is in the [R || S || V] format where V is 0 or 1.
	SignHash(a accounts.Account, hash []byte) ([]byte, error)

	// SignTx signs the given transaction with the requested account.
	SignTx(a accounts.Account, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)

	// SignHashWithPassphrase signs hash if the private key matching the given address
	// can be decrypted with the given passphrase. The produced signature is in the
	// [R || S || V] format where V is 0 or 1.
	SignHashWithPassphrase(a accounts.Account, passphrase string, hash []byte) (signature []byte, err error)

	// SignTxWithPassphrase signs the transaction if the private key matching the
	// given address can be decrypted with the given passphrase.
	SignTxWithPassphrase(a accounts.Account, passphrase string, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)

	// Unlock unlocks the given account indefinitely.
	Unlock(a accounts.Account, passphrase string) error

	// Lock removes the private key with the given address from memory.
	Lock(addr common.Address) error

	// TimedUnlock unlocks the given account with the passphrase. The account
	// stays unlocked for the duration of timeout. A timeout of 0 unlocks the account
	// until the program exits. The account must match a unique key file.
	//
	// If the account address is already unlocked for a duration, TimedUnlock extends or
	// shortens the active unlock timeout. If the address was previously unlocked
	// indefinitely the timeout is not altered.
	TimedUnlock(a accounts.Account, passphrase string, timeout time.Duration) error

	// Find resolves the given account into a unique entry in the KeyStore.
	Find(a accounts.Account) (accounts.Account, error)

	// NewAccount generates a new key and stores it into the KeyStore,
	// encrypting it with the passphrase.
	NewAccount(passphrase string) (accounts.Account, error)

	// Export exports as a JSON key, encrypted with newPassphrase.
	Export(a accounts.Account, passphrase, newPassphrase string) (keyJSON []byte, err error)

	// Import stores the given encrypted JSON key into the KeyStore.
	Import(keyJSON []byte, passphrase, newPassphrase string) (accounts.Account, error)

	// ImportECDSA stores the given key into the KeyStore, encrypting it with the passphrase.
	ImportECDSA(priv *ecdsa.PrivateKey, passphrase string) (accounts.Account, error)

	// Update changes the passphrase of an existing account.
	Update(a accounts.Account, passphrase, newPassphrase string) error

	// ImportPreSaleKey decrypts the given Ethereum presale wallet and stores
	// a key file in the KeyStore. The key file is encrypted with the same passphrase.
	ImportPreSaleKey(keyJSON []byte, passphrase string) (accounts.Account, error)
}

type unlocked struct {
	*Key
	abort chan struct{}
}

// NewKeyStore creates a keystore for the given directory.
func NewKeyStore(keydir string, scryptN, scryptP int) KeyStore {
	keydir, _ = filepath.Abs(keydir)
	ks := &keyStoreFS{storage: &keyStorePassphrase{keydir, scryptN, scryptP, false}}
	ks.init(keydir)
	return ks
}

// NewPlaintextKeyStore creates a keystore for the given directory.
// Deprecated: Use NewKeyStore.
func NewPlaintextKeyStore(keydir string) KeyStore {
	keydir, _ = filepath.Abs(keydir)
	ks := &keyStoreFS{storage: &keyStorePlain{keydir}}
	ks.init(keydir)
	return ks
}

// NewKeyStoreDB creates a keystore for the given database
func NewKeyStoreDB(path string, scryptN, scryptP int) (KeyStore, error) {
	kvstore, err := dbutil.NewKVStore(path, keystoreDBTableName)
	if err != nil {
		return nil, err
	}
	storage := &keyStorePassphraseDB{kvstore, scryptN, scryptP, false}
	ks := &keyStoreDB{storage: storage, unlocked: make(map[common.Address]*unlocked)}
	return ks, nil
}
