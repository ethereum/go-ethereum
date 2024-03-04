// Copyright 2018 The go-ethereum Authors
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

package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"

	"github.com/ethereum/go-ethereum/log"
)

type storedCredential struct {
	// The iv
	Iv []byte `json:"iv"`
	// The ciphertext
	CipherText []byte `json:"c"`
}

// AESEncryptedStorage is a storage type which is backed by a json-file. The json-file contains
// key-value mappings, where the keys are _not_ encrypted, only the values are.
type AESEncryptedStorage struct {
	// File to read/write credentials
	filename string
	// Key stored in base64
	key []byte
	gcm cipher.AEAD

	gcmNS int // NonceSize
}

// NewAESEncryptedStorage creates a new encrypted storage backed by the given file/key
func NewAESEncryptedStorage(filename string, key []byte) *AESEncryptedStorage {
	aesStore := &AESEncryptedStorage{
		filename: filename,
		key:      key,
	}

	blk, err := aes.NewCipher(key)
	if err != nil {
		log.Warn("reading AES key", "err", err)
		return nil
	}

	aesStore.gcm, err = cipher.NewGCM(blk)
	if err != nil {
		log.Warn("initializing AES AEAD", "err", err)
		return nil
	}

	aesStore.gcmNS = aesStore.gcm.NonceSize()
	return aesStore
}

// Put stores a value by key. 0-length keys results in noop.
func (s *AESEncryptedStorage) Put(key, value string) {
	if len(key) == 0 {
		return
	}
	data, err := s.readEncryptedStorage()
	if err != nil {
		log.Warn("Failed to read encrypted storage", "err", err, "file", s.filename)
		return
	}
	ciphertext, iv, err := s.encrypt([]byte(value), []byte(key))
	if err != nil {
		log.Warn("Failed to encrypt entry", "err", err)
		return
	}
	encrypted := storedCredential{Iv: iv, CipherText: ciphertext}
	data[key] = encrypted
	if err = s.writeEncryptedStorage(data); err != nil {
		log.Warn("Failed to write entry", "err", err)
	}
}

// Get returns the previously stored value, or an error if it does not exist or
// key is of 0-length.
func (s *AESEncryptedStorage) Get(key string) (string, error) {
	if len(key) == 0 {
		return "", ErrZeroKey
	}
	data, err := s.readEncryptedStorage()
	if err != nil {
		log.Warn("Failed to read encrypted storage", "err", err, "file", s.filename)
		return "", err
	}
	encrypted, exist := data[key]
	if !exist {
		log.Warn("Key does not exist", "key", key)
		return "", ErrNotFound
	}
	entry, err := s.decrypt(encrypted.CipherText, []byte(key))
	if err != nil {
		log.Warn("Failed to decrypt key", "key", key)
		return "", err
	}
	return string(entry), nil
}

// Del removes a key-value pair. If the key doesn't exist, the method is a noop.
func (s *AESEncryptedStorage) Del(key string) {
	data, err := s.readEncryptedStorage()
	if err != nil {
		log.Warn("Failed to read encrypted storage", "err", err, "file", s.filename)
		return
	}
	delete(data, key)
	if err = s.writeEncryptedStorage(data); err != nil {
		log.Warn("Failed to write entry", "err", err)
	}
}

// readEncryptedStorage reads the file with encrypted creds
func (s *AESEncryptedStorage) readEncryptedStorage() (map[string]storedCredential, error) {
	creds := make(map[string]storedCredential)
	raw, err := os.ReadFile(s.filename)

	if err != nil {
		if os.IsNotExist(err) {
			// Doesn't exist yet
			return creds, nil
		}
		log.Warn("Failed to read encrypted storage", "err", err, "file", s.filename)
	}
	if err = json.Unmarshal(raw, &creds); err != nil {
		log.Warn("Failed to unmarshal encrypted storage", "err", err, "file", s.filename)
		return nil, err
	}
	return creds, nil
}

// writeEncryptedStorage write the file with encrypted creds
func (s *AESEncryptedStorage) writeEncryptedStorage(creds map[string]storedCredential) error {
	raw, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	if err = os.WriteFile(s.filename, raw, 0600); err != nil {
		return err
	}
	return nil
}

func (s *AESEncryptedStorage) randBytes() (nonce []byte, err error) {
	nonce = make([]byte, s.gcmNS)
	_, err = io.ReadFull(rand.Reader, nonce)
	return
}

// encrypt encrypts plaintext with the given key, with additional data
// The 'additionalData' is used to place the (plaintext) KV-store key into the V,
// to prevent the possibility to alter a K, or swap two entries in the KV store with each other.
func (s *AESEncryptedStorage) encrypt(plaintext []byte, additionalData []byte) (_ []byte, _ []byte, err error) {
	// Never use more than 2^32 random nonce's with a given
	// key because of the risk of a repeat.
	var n []byte

	if n, err = s.randBytes(); err != nil {
		return
	}
	return append(n, s.gcm.Seal(nil, n, plaintext, additionalData)...), n, nil
}

func (s *AESEncryptedStorage) decrypt(ciphertext []byte, additionalData []byte) ([]byte, error) {
	if len(ciphertext) < s.gcmNS {
		return nil, errors.New("cipher data too short")
	}
	return s.gcm.Open(nil, ciphertext[0:s.gcmNS], ciphertext[s.gcmNS:], additionalData)
}
