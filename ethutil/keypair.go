package ethutil

import (
	"github.com/obscuren/secp256k1-go"
)

type KeyPair struct {
	PrivateKey []byte
	PublicKey  []byte

	// The associated account
	account *StateObject
}

func GenerateNewKeyPair() (*KeyPair, error) {
	_, prv := secp256k1.GenerateKeyPair()

	return NewKeyPairFromSec(prv)
}

func NewKeyPairFromSec(seckey []byte) (*KeyPair, error) {
	pubkey, err := secp256k1.GeneratePubKey(seckey)
	if err != nil {
		return nil, err
	}

	return &KeyPair{PrivateKey: seckey, PublicKey: pubkey}, nil
}

func NewKeyPairFromValue(val *Value) *KeyPair {
	v, _ := NewKeyPairFromSec(val.Bytes())

	return v
}

func (k *KeyPair) Address() []byte {
	return Sha3Bin(k.PublicKey[1:])[12:]
}

func (k *KeyPair) RlpEncode() []byte {
	return k.RlpValue().Encode()
}

func (k *KeyPair) RlpValue() *Value {
	return NewValue(k.PrivateKey)
}

type KeyRing struct {
	keys []*KeyPair
}

func (k *KeyRing) Add(pair *KeyPair) {
	k.keys = append(k.keys, pair)
}

func (k *KeyRing) Get(i int) *KeyPair {
	if len(k.keys) > i {
		return k.keys[i]
	}

	return nil
}

func (k *KeyRing) Len() int {
	return len(k.keys)
}

func (k *KeyRing) NewKeyPair(sec []byte) (*KeyPair, error) {
	keyPair, err := NewKeyPairFromSec(sec)
	if err != nil {
		return nil, err
	}

	k.Add(keyPair)
	Config.Db.Put([]byte("KeyRing"), k.RlpValue().Encode())

	return keyPair, nil
}

func (k *KeyRing) Reset() {
	Config.Db.Put([]byte("KeyRing"), nil)
	k.keys = nil
}

func (k *KeyRing) RlpValue() *Value {
	v := EmptyValue()
	for _, keyPair := range k.keys {
		v.Append(keyPair.RlpValue())
	}

	return v
}

// The public "singleton" keyring
var keyRing *KeyRing

func GetKeyRing() *KeyRing {
	if keyRing == nil {
		keyRing = &KeyRing{}

		data, _ := Config.Db.Get([]byte("KeyRing"))
		it := NewValueFromBytes(data).NewIterator()
		for it.Next() {
			v := it.Value()

			key, err := NewKeyPairFromSec(v.Bytes())
			if err != nil {
				panic(err)
			}
			keyRing.Add(key)
		}
	}

	return keyRing
}
