package ethclient

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// EncryptedData is encrypted blob
type EncryptedData struct {
	Version        string `json:"version,omitempty"`
	Nonce          string `json:"nonce,omitempty"`
	EphemPublicKey string `json:"ephemPublicKey,omitempty"`
	Ciphertext     string `json:"ciphertext,omitempty"`
}

// GetEncryptionPublicKey returns user's public Encryption key derived from privateKey Ethereum key
func (ec *Client) GetEncryptionPublicKey(ctx context.Context, ethPrivKey string) (string, error) {
	privateKey0, err := hexutil.Decode("0x" + ethPrivKey)
	if err != nil {
		return "", err
	}
	privateKey := [32]byte{}
	copy(privateKey[:], privateKey0[:32])

	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)
	return base64.StdEncoding.EncodeToString(publicKey[:]), nil
}

// Encrypt plain data
func (ec *Client) Encrypt(ctx context.Context, receiverPublicKey string, data []byte, version string) (*EncryptedData, error) {
	switch version {
	case "x25519-xsalsa20-poly1305":
		ephemeralPublic, ephemeralPrivate, err := box.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}

		publicKey0, err := base64.StdEncoding.DecodeString(receiverPublicKey)
		if err != nil {
			return nil, err
		}

		publicKey := [32]byte{}
		copy(publicKey[:], publicKey0[:32])

		var nonce [24]byte
		if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
			return nil, err
		}

		out := box.Seal(nil, data, &nonce, &publicKey, ephemeralPrivate)

		return &EncryptedData{
			Version:        version,
			Nonce:          base64.StdEncoding.EncodeToString(nonce[:]),
			EphemPublicKey: base64.StdEncoding.EncodeToString(ephemeralPublic[:]),
			Ciphertext:     base64.StdEncoding.EncodeToString(out),
		}, nil
	default:
		return nil, errors.New("Encryption type/version not supported")
	}
}

// Decrypt some encrypted data.
func (ec *Client) Decrypt(ctx context.Context, receiverPrivatekey string, encryptedData *EncryptedData) ([]byte, error) {
	switch encryptedData.Version {
	case "x25519-xsalsa20-poly1305":
		privateKey0, err := hexutil.Decode("0x" + receiverPrivatekey)
		if err != nil {
			return nil, err
		}

		privateKey := [32]byte{}
		copy(privateKey[:], privateKey0[:32])

		// assemble decryption parameters
		nonce, err := base64.StdEncoding.DecodeString(encryptedData.Nonce)
		if err != nil {
			return nil, err
		}
		ciphertext, err := base64.StdEncoding.DecodeString(encryptedData.Ciphertext)
		if err != nil {
			return nil, err
		}
		ephemPublicKey, err := base64.StdEncoding.DecodeString(encryptedData.EphemPublicKey)
		if err != nil {
			return nil, err
		}

		publicKey := [32]byte{}
		copy(publicKey[:], ephemPublicKey[:32])

		nonce24 := [24]byte{}
		copy(nonce24[:], nonce[:24])

		decryptedMessage, ok := box.Open(nil, ciphertext, &nonce24, &publicKey, &privateKey)
		if !ok {
			return nil, errors.New("Decryption fail")
		}
		return decryptedMessage, nil
	default:
		return nil, errors.New("Decryption type/version not supported")
	}
}
