package ethcrypto

import (
	"github.com/ethereum/eth-go/ethutil"
	"github.com/obscuren/secp256k1-go"
)

type KeyPair struct {
	PrivateKey []byte
	PublicKey  []byte

	// The associated account
	// account *StateObject
}

func GenerateNewKeyPair() *KeyPair {
	_, prv := secp256k1.GenerateKeyPair()
	keyPair, _ := NewKeyPairFromSec(prv) // swallow error, this one cannot err
	return keyPair
}

func NewKeyPairFromSec(seckey []byte) (*KeyPair, error) {
	pubkey, err := secp256k1.GeneratePubKey(seckey)
	if err != nil {
		return nil, err
	}

	return &KeyPair{PrivateKey: seckey, PublicKey: pubkey}, nil
}

func (k *KeyPair) Address() []byte {
	return Sha3Bin(k.PublicKey[1:])[12:]
}

func (k *KeyPair) RlpEncode() []byte {
	return k.RlpValue().Encode()
}

func (k *KeyPair) RlpValue() *ethutil.Value {
	return ethutil.NewValue(k.PrivateKey)
}
