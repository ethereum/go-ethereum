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
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/cmd/clef/dbutil"
	"github.com/ethereum/go-ethereum/log"
)

var (
	// ErrZeroKey is returned if an attempt was made to inset a 0-length key.
	ErrZeroKey = errors.New("0-length key")

	// ErrNotFound is returned if an unknown key is attempted to be retrieved.
	ErrNotFound = errors.New("not found")
)

// storageAPI is the interface that defines interactions with backend storage client
type storageAPI interface {
	// Put stores a value by key. 0-length keys results in noop.
	Put(key, value string) error

	// Get returns the previously stored value, or an error if the key is 0-length
	// or unknown.
	Get(key string) (string, error)

	// Del removes a key-value pair. If the key doesn't exist, the method is a noop.
	Del(key string)
}

// Storage is the storage client which is used by client to store key/value mappings.
// The keys are _not_ encrypted, only the values are.
type Storage struct {
	api storageAPI
	key []byte
}

// Get calls the underlying storageApi's Get function and then decrypts the value field
func (s *Storage) Get(key string) (string, error) {
	if len(key) == 0 {
		return "", ErrZeroKey
	}

	data, err := s.api.Get(key)
	if err != nil {
		return "", err
	}

	if len(s.key) == 0 {
		// if there is no key for decryption, just store the value
		return data, nil
	}

	cred := StoredCredential{}
	if err = json.Unmarshal([]byte(data), &cred); err != nil {
		log.Warn("Failed to unmarshal encrypted credential", "err", err)
		return "", err
	}

	entry, err := Decrypt(s.key, cred.Iv, cred.CipherText, []byte(key))
	if err != nil {
		log.Warn("Failed to decrypt key", "key", key)
		return "", err
	}

	return string(entry), nil
}

// Put encrypts the value field with key as additionalData to prevent value swap attack.
// Then calls the underlying storageApi's Put function to persist the key/value pair
func (s *Storage) Put(key, value string) error {
	if len(key) == 0 {
		return ErrZeroKey
	}

	if len(s.key) == 0 {
		// if there is no key for encryption, just store the value
		return s.api.Put(key, value)
	}

	ciphertext, iv, err := Encrypt(s.key, []byte(value), []byte(key))
	if err != nil {
		log.Warn("Failed to encrypt entry", "err", err)
		return err
	}

	encrypted := StoredCredential{Iv: iv, CipherText: ciphertext}
	raw, err := json.Marshal(encrypted)
	if err != nil {
		log.Warn("Failed to marshal credential", "err", err)
		return err
	}
	return s.api.Put(key, string(raw))
}

// Del calls the underlying storageApi's Del function to delete the key/value pair
func (s *Storage) Del(key string) {
	s.api.Del(key)
}

// NewEphemeralStorage creates an in-memory storage that does
// not persist values to disk. Mainly used for testing
func NewEphemeralStorage() Storage {
	api := &EphemeralStorageAPI{
		data: make(map[string]string),
	}
	return Storage{
		api: api,
		key: []byte(""),
	}
}

// NewNoStorage creates an dummy storage which didn't remember anything you tell it
func NewNoStorage() Storage {
	api := &NoStorageAPI{}
	return Storage{
		api: api,
		key: []byte(""),
	}
}

// NewDBStorage creates a database storage
func NewDBStorage(path, table string, key []byte) (Storage, error) {
	kvstore, err := dbutil.NewKVStore(path, table)
	if err != nil {
		return NewNoStorage(), err
	}
	api := &DBStorageAPI{
		kvstore: kvstore,
	}
	return Storage{
		api: api,
		key: key,
	}, nil
}

// NewFileStorage creates a storage type which is backed by a json-file.
func NewFileStorage(filename string, key []byte) Storage {
	api := &FileStorageAPI{
		filename: filename,
	}
	return Storage{
		api: api,
		key: key,
	}
}
