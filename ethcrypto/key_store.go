package ethcrypto

import (
	"fmt"
	"github.com/ethereum/eth-go/ethutil"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

type KeyStore interface {
	Load(string) (*KeyRing, error)
	Save(string, *KeyRing) error
}

type DBKeyStore struct {
	db ethutil.Database
}

const dbKeyPrefix = "KeyRing"

func (k *DBKeyStore) dbKey(session string) []byte {
	return []byte(fmt.Sprintf("%s%s", dbKeyPrefix, session))
}

func (k *DBKeyStore) Save(session string, keyRing *KeyRing) error {
	k.db.Put(k.dbKey(session), keyRing.RlpEncode())
	return nil
}

func (k *DBKeyStore) Load(session string) (*KeyRing, error) {
	data, err := k.db.Get(k.dbKey(session))
	if err != nil {
		return nil, nil
	}
	var keyRing *KeyRing
	keyRing, err = NewKeyRingFromBytes(data)
	if err != nil {
		return nil, err
	}
	// if empty keyRing is found we return nil, no error
	if keyRing.Len() == 0 {
		return nil, nil
	}
	return keyRing, nil
}

type FileKeyStore struct {
	basedir string
}

func (k *FileKeyStore) Save(session string, keyRing *KeyRing) error {
	var content []byte
	var err error
	var privateKeys []string
	var publicKeys []string
	var mnemonics []string
	var addresses []string
	keyRing.Each(func(keyPair *KeyPair) {
		privateKeys = append(privateKeys, ethutil.Bytes2Hex(keyPair.PrivateKey))
		publicKeys = append(publicKeys, ethutil.Bytes2Hex(keyPair.PublicKey))
		addresses = append(addresses, ethutil.Bytes2Hex(keyPair.Address()))
		mnemonics = append(mnemonics, keyPair.Mnemonic())
	})

	basename := session
	if session == "" {
		basename = "default"
	}

	path := path.Join(k.basedir, basename)
	content = []byte(strings.Join(privateKeys, "\n"))
	err = ioutil.WriteFile(path+".prv", content, 0600)
	if err != nil {
		return err
	}

	content = []byte(strings.Join(publicKeys, "\n"))
	err = ioutil.WriteFile(path+".pub", content, 0644)
	if err != nil {
		return err
	}

	content = []byte(strings.Join(addresses, "\n"))
	err = ioutil.WriteFile(path+".addr", content, 0644)
	if err != nil {
		return err
	}

	content = []byte(strings.Join(mnemonics, "\n"))
	err = ioutil.WriteFile(path+".mne", content, 0600)
	if err != nil {
		return err
	}

	return nil
}

func (k *FileKeyStore) Load(session string) (*KeyRing, error) {
	basename := session
	if session == "" {
		basename = "default"
	}
	secfile := path.Join(k.basedir, basename+".prv")
	_, err := os.Stat(secfile)
	// if file is not found then we return nil, no error
	if err != nil {
		return nil, nil
	}
	return NewKeyRingFromFile(secfile)
}
