package ethcrypto

import (
	"fmt"
	"github.com/ethereum/eth-go/ethutil"
	"sync"
)

type KeyManager struct {
	keyRing  *KeyRing
	session  string
	keyStore KeyStore            // interface
	keyRings map[string]*KeyRing // cache
	keyPair  *KeyPair
}

func NewDBKeyManager(db ethutil.Database) *KeyManager {
	return &KeyManager{keyStore: &DBKeyStore{db: db}, keyRings: make(map[string]*KeyRing)}
}

func NewFileKeyManager(basedir string) *KeyManager {
	return &KeyManager{keyStore: &FileKeyStore{basedir: basedir}, keyRings: make(map[string]*KeyRing)}
}

func (k *KeyManager) KeyPair() *KeyPair {
	return k.keyPair
}

func (k *KeyManager) KeyRing() *KeyPair {
	return k.keyPair
}

func (k *KeyManager) PrivateKey() []byte {
	return k.keyPair.PrivateKey
}

func (k *KeyManager) PublicKey() []byte {
	return k.keyPair.PublicKey
}

func (k *KeyManager) Address() []byte {
	return k.keyPair.Address()
}

func (k *KeyManager) save(session string, keyRing *KeyRing) error {
	err := k.keyStore.Save(session, keyRing)
	if err != nil {
		return err
	}
	k.keyRings[session] = keyRing
	return nil
}

func (k *KeyManager) load(session string) (*KeyRing, error) {
	keyRing, found := k.keyRings[session]
	if !found {
		var err error
		keyRing, err = k.keyStore.Load(session)
		if err != nil {
			return nil, err
		}
	}
	return keyRing, nil
}

func cursorError(cursor int, len int) error {
	return fmt.Errorf("cursor %d out of range (0..%d)", cursor, len)
}

func (k *KeyManager) reset(session string, cursor int, keyRing *KeyRing) error {
	if cursor >= keyRing.Len() {
		return cursorError(cursor, keyRing.Len())
	}
	lock := &sync.Mutex{}
	lock.Lock()
	defer lock.Unlock()
	err := k.save(session, keyRing)
	if err != nil {
		return err
	}
	k.session = session
	k.keyRing = keyRing
	k.keyPair = keyRing.GetKeyPair(cursor)
	return nil
}

func (k *KeyManager) SetCursor(cursor int) error {
	if cursor >= k.keyRing.Len() {
		return cursorError(cursor, k.keyRing.Len())
	}
	k.keyPair = k.keyRing.GetKeyPair(cursor)
	return nil
}

func (k *KeyManager) Init(session string, cursor int, force bool) error {
	var keyRing *KeyRing
	if !force {
		var err error
		keyRing, err = k.load(session)
		if err != nil {
			return err
		}
	}
	if keyRing == nil {
		keyRing = NewGeneratedKeyRing(1)
	}
	return k.reset(session, cursor, keyRing)
}

func (k *KeyManager) InitFromSecretsFile(session string, cursor int, secretsfile string) error {
	keyRing, err := NewKeyRingFromFile(secretsfile)
	if err != nil {
		return err
	}
	return k.reset(session, cursor, keyRing)
}

func (k *KeyManager) InitFromString(session string, cursor int, secrets string) error {
	keyRing, err := NewKeyRingFromString(secrets)
	if err != nil {
		return err
	}
	return k.reset(session, cursor, keyRing)
}

func (k *KeyManager) Export(dir string) error {
	fileKeyStore := FileKeyStore{dir}
	return fileKeyStore.Save(k.session, k.keyRing)
}
