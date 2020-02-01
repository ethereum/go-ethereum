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
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/log"
)

// FileStorageAPI is a storage type which is backed by a json-file.
type FileStorageAPI struct {
	// File to read/write credentials
	filename string
}

// Put stores a value by key. 0-length keys results in noop.
func (s *FileStorageAPI) Put(key, value string) {
	if len(key) == 0 {
		return
	}
	data, err := s.readStorage()
	if err != nil {
		log.Warn("Failed to read encrypted storage", "err", err, "file", s.filename)
		return
	}
	data[key] = value
	if err = s.writeStorage(data); err != nil {
		log.Warn("Failed to write entry", "err", err)
	}
}

// Get returns the previously stored value, or an error if it does not exist or
// key is of 0-length.
func (s *FileStorageAPI) Get(key string) (string, error) {
	if len(key) == 0 {
		return "", ErrZeroKey
	}
	data, err := s.readStorage()
	if err != nil {
		log.Warn("Failed to read encrypted storage", "err", err, "file", s.filename)
		return "", err
	}
	value, exist := data[key]
	if !exist {
		log.Warn("Key does not exist", "key", key)
		return "", ErrNotFound
	}
	return value, nil
}

// Del removes a key-value pair. If the key doesn't exist, the method is a noop.
func (s *FileStorageAPI) Del(key string) {
	data, err := s.readStorage()
	if err != nil {
		log.Warn("Failed to read encrypted storage", "err", err, "file", s.filename)
		return
	}
	delete(data, key)
	if err = s.writeStorage(data); err != nil {
		log.Warn("Failed to write entry", "err", err)
	}
}

// readEncryptedStorage reads the file with encrypted creds
// func (s *FileStorageAPI) readEncryptedStorage() (map[string]StoredCredential, error) {
// 	creds := make(map[string]StoredCredential)
// 	raw, err := ioutil.ReadFile(s.filename)

// 	if err != nil {
// 		if os.IsNotExist(err) {
// 			// Doesn't exist yet
// 			return creds, nil
// 		}
// 		log.Warn("Failed to read encrypted storage", "err", err, "file", s.filename)
// 	}
// 	if err = json.Unmarshal(raw, &creds); err != nil {
// 		log.Warn("Failed to unmarshal encrypted storage", "err", err, "file", s.filename)
// 		return nil, err
// 	}
// 	return creds, nil
// }

// readStorage reads the file with encrypted creds and return them as is
func (s *FileStorageAPI) readStorage() (map[string]string, error) {
	creds := make(map[string]string)
	raw, err := ioutil.ReadFile(s.filename)

	if err != nil {
		if os.IsNotExist(err) {
			// Doesn't exist yet
			return creds, nil
		}
		log.Warn("Failed to read file storage", "err", err, "file", s.filename)
	}

	if err = json.Unmarshal(raw, &creds); err != nil {
		log.Warn("Failed to unmarshal file storage", "err", err, "file", s.filename)
		return nil, err
	}
	return creds, nil
}

// writeEncryptedStorage write the file with encrypted creds
// func (s *FileStorageAPI) writeEncryptedStorage(creds map[string]StoredCredential) error {
// 	raw, err := json.Marshal(creds)
// 	if err != nil {
// 		return err
// 	}
// 	if err = ioutil.WriteFile(s.filename, raw, 0600); err != nil {
// 		return err
// 	}
// 	return nil
// }

func (s *FileStorageAPI) writeStorage(creds map[string]string) error {
	raw, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(s.filename, raw, 0600); err != nil {
		return err
	}
	return nil
}
