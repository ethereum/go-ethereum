package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

// StoredCredential stores the json structure of the stored credential
type StoredCredential struct {
	// The iv
	Iv []byte `json:"iv"`
	// The ciphertext
	CipherText []byte `json:"c"`
}

// Encrypt encrypts plaintext with the given key, with additional data
// The 'additionalData' is used to place the (plaintext) KV-store key into the V,
// to prevent the possibility to alter a K, or swap two entries in the KV store with eachother.
func Encrypt(key []byte, plaintext []byte, additionalData []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	if err != nil {
		return nil, nil, err
	}
	ciphertext := aesgcm.Seal(nil, nonce, plaintext, additionalData)
	return ciphertext, nonce, nil
}

// Decrypt decrypts plaintext from given key, nounce, ciphertext and additionalData
func Decrypt(key []byte, nonce []byte, ciphertext []byte, additionalData []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, additionalData)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
