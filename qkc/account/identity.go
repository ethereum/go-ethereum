// Ported verbatim from github.com/QuarkChain/goquarkchain/account (byte-compatible).

package account

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
)

// Identity include recipient and key
type Identity struct {
	recipient Recipient
	key       Key
}

// NewIdentity new identity include recipient and key
func NewIdentity(recipient Recipient, key Key) Identity {
	return Identity{
		recipient: recipient,
		key:       key,
	}
}

// CreatRandomIdentity create a random identity
func CreatRandomIdentity() (Identity, error) {
	sk, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return Identity{}, ErrGenIdentityKey
	}

	key := crypto.FromECDSA(sk)
	if len(key) != KeyLength {
		return Identity{}, fmt.Errorf("privateKey To Bytes falied: unexceptd %d ,excepted 32", len(key))
	}
	if len(crypto.FromECDSAPub(&sk.PublicKey)) != 2*KeyLength+1 {
		return Identity{}, fmt.Errorf("fromECDSAPub len is not match :unexcepted %d,excepted 65", len(crypto.FromECDSAPub(&sk.PublicKey)))
	}

	recipient := crypto.Keccak256(crypto.FromECDSAPub(&sk.PublicKey)[1:])
	if len(recipient) != KeyLength {
		return Identity{}, fmt.Errorf("recipient len is not match:unexceptd %d,exceptd 32", len(recipient))
	}
	return newIdentity(recipient, key)

}

// CreatIdentityFromKey creat identity from key
func CreatIdentityFromKey(key Key) (Identity, error) {
	keyValue := big.NewInt(0)
	keyValue.SetBytes(key.Bytes())
	sk := new(ecdsa.PrivateKey)
	sk.PublicKey.Curve = crypto.S256()
	sk.D = keyValue
	sk.PublicKey.X, sk.PublicKey.Y = crypto.S256().ScalarBaseMult(keyValue.Bytes())
	if len(crypto.FromECDSAPub(&sk.PublicKey)) != 2*KeyLength+1 {
		return Identity{}, fmt.Errorf("fromECDSAPub len is not match :unexcepted %d,excepted %d", len(crypto.FromECDSAPub(&sk.PublicKey)), 2*KeyLength+1)
	}

	recipient := crypto.Keccak256(crypto.FromECDSAPub(&sk.PublicKey)[1:]) //"0x04"+64
	if len(recipient) != KeyLength {
		return Identity{}, fmt.Errorf("recipient len is not match:unexceptd %d,exceptd 32", len(recipient))
	}

	return newIdentity(recipient, key.Bytes())
}

func newIdentity(recipient []byte, key []byte) (Identity, error) {
	recipientType := BytesToIdentityRecipient(recipient[(len(recipient) - RecipientLength):])
	keyType := BytesToIdentityKey(key)
	return NewIdentity(recipientType, keyType), nil
}

// GetDefaultFullShardKey get identity's default fullShardKey
func (Self *Identity) GetDefaultFullShardKey() (uint32, error) {
	var fullShardKey uint32
	r := Self.recipient
	realShardKey := []byte{0x00, 0x00}
	realShardKey = append(realShardKey, r[0:1]...)
	realShardKey = append(realShardKey, r[10:11]...)
	buffer := bytes.NewBuffer(realShardKey)
	err := binary.Read(buffer, binary.BigEndian, &fullShardKey)
	if err != nil {
		return fullShardKey, err
	}
	return fullShardKey, nil
}

// GetRecipient Get it's recipient
func (Self *Identity) GetRecipient() Recipient {
	return Self.recipient
}

// GetKey get it's key
func (Self *Identity) GetKey() Key {
	return Self.key
}
