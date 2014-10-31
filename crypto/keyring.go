package crypto

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/ethereum/go-ethereum/ethutil"
)

type KeyRing struct {
	keys []*KeyPair
}

func NewKeyRing() *KeyRing {
	return &KeyRing{}
}

func (k *KeyRing) AddKeyPair(keyPair *KeyPair) {
	k.keys = append(k.keys, keyPair)
}

func (k *KeyRing) GetKeyPair(i int) *KeyPair {
	if len(k.keys) > i {
		return k.keys[i]
	}

	return nil
}

func (k *KeyRing) Empty() bool {
	return k.Len() == 0
}

func (k *KeyRing) Len() int {
	return len(k.keys)
}

func (k *KeyRing) Each(f func(*KeyPair)) {
	for _, keyPair := range k.keys {
		f(keyPair)
	}
}

func NewGeneratedKeyRing(len int) *KeyRing {
	keyRing := NewKeyRing()
	for i := 0; i < len; i++ {
		keyRing.AddKeyPair(GenerateNewKeyPair())
	}
	return keyRing
}

func NewKeyRingFromFile(secfile string) (*KeyRing, error) {
	var content []byte
	var err error
	content, err = ioutil.ReadFile(secfile)
	if err != nil {
		return nil, err
	}
	keyRing, err := NewKeyRingFromString(string(content))
	if err != nil {
		return nil, err
	}
	return keyRing, nil
}

func NewKeyRingFromString(content string) (*KeyRing, error) {
	secretStrings := strings.Split(content, "\n")
	var secrets [][]byte
	for _, secretString := range secretStrings {
		secret := secretString
		words := strings.Split(secretString, " ")
		if len(words) == 24 {
			secret = MnemonicDecode(words)
		} else if len(words) != 1 {
			return nil, fmt.Errorf("Unrecognised key format")
		}

		if len(secret) != 0 {
			secrets = append(secrets, ethutil.Hex2Bytes(secret))
		}
	}

	return NewKeyRingFromSecrets(secrets)
}

func NewKeyRingFromSecrets(secs [][]byte) (*KeyRing, error) {
	keyRing := NewKeyRing()
	for _, sec := range secs {
		keyPair, err := NewKeyPairFromSec(sec)
		if err != nil {
			return nil, err
		}
		keyRing.AddKeyPair(keyPair)
	}
	return keyRing, nil
}

func NewKeyRingFromBytes(data []byte) (*KeyRing, error) {
	var secrets [][]byte
	it := ethutil.NewValueFromBytes(data).NewIterator()
	for it.Next() {
		secret := it.Value().Bytes()
		secrets = append(secrets, secret)
	}
	keyRing, err := NewKeyRingFromSecrets(secrets)
	if err != nil {
		return nil, err
	}
	return keyRing, nil
}

func (k *KeyRing) RlpEncode() []byte {
	return k.RlpValue().Encode()
}

func (k *KeyRing) RlpValue() *ethutil.Value {
	v := ethutil.EmptyValue()
	k.Each(func(keyPair *KeyPair) {
		v.Append(keyPair.RlpValue())
	})
	return v
}
