// Copyright 2014 The go-ethereum Authors
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

package accounts

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type keyStorePlain struct {
	keysDirPath string
}

func newKeyStorePlain(path string) keyStore {
	return &keyStorePlain{path}
}

func (ks keyStorePlain) GenerateNewKey(rand io.Reader, auth string) (key *Key, err error) {
	return generateNewKeyDefault(ks, rand, auth)
}

func generateNewKeyDefault(ks keyStore, rand io.Reader, auth string) (key *Key, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("GenerateNewKey error: %v", r)
		}
	}()
	key = NewKey(rand)
	err = ks.StoreKey(key, auth)
	return key, err
}

func (ks keyStorePlain) GetKey(keyAddr common.Address, auth string) (*Key, error) {
	keyjson, err := getKeyFile(ks.keysDirPath, keyAddr)
	if err != nil {
		return nil, err
	}
	key := new(Key)
	if err := json.Unmarshal(keyjson, key); err != nil {
		return nil, err
	}
	return key, nil
}

func (ks keyStorePlain) GetKeyAddresses() (addresses []common.Address, err error) {
	return getKeyAddresses(ks.keysDirPath)
}

func (ks keyStorePlain) Cleanup(keyAddr common.Address) (err error) {
	return cleanup(ks.keysDirPath, keyAddr)
}

func (ks keyStorePlain) StoreKey(key *Key, auth string) (err error) {
	keyJSON, err := json.Marshal(key)
	if err != nil {
		return
	}
	err = writeKeyFile(key.Address, ks.keysDirPath, keyJSON)
	return
}

func (ks keyStorePlain) DeleteKey(keyAddr common.Address, auth string) (err error) {
	return deleteKey(ks.keysDirPath, keyAddr)
}

func deleteKey(keysDirPath string, keyAddr common.Address) (err error) {
	var path string
	path, err = getKeyFilePath(keysDirPath, keyAddr)
	if err == nil {
		addrHex := hex.EncodeToString(keyAddr[:])
		if path == filepath.Join(keysDirPath, addrHex, addrHex) {
			path = filepath.Join(keysDirPath, addrHex)
		}
		err = os.RemoveAll(path)
	}
	return
}

func getKeyFilePath(keysDirPath string, keyAddr common.Address) (keyFilePath string, err error) {
	addrHex := hex.EncodeToString(keyAddr[:])
	matches, err := filepath.Glob(filepath.Join(keysDirPath, fmt.Sprintf("*--%s", addrHex)))
	if len(matches) > 0 {
		if err == nil {
			keyFilePath = matches[len(matches)-1]
		}
		return
	}
	keyFilePath = filepath.Join(keysDirPath, addrHex, addrHex)
	_, err = os.Stat(keyFilePath)
	return
}

func cleanup(keysDirPath string, keyAddr common.Address) (err error) {
	fileInfos, err := ioutil.ReadDir(keysDirPath)
	if err != nil {
		return
	}
	var paths []string
	account := hex.EncodeToString(keyAddr[:])
	for _, fileInfo := range fileInfos {
		path := filepath.Join(keysDirPath, fileInfo.Name())
		if len(path) >= 40 {
			addr := path[len(path)-40 : len(path)]
			if addr == account {
				if path == filepath.Join(keysDirPath, addr, addr) {
					path = filepath.Join(keysDirPath, addr)
				}
				paths = append(paths, path)
			}
		}
	}
	if len(paths) > 1 {
		for i := 0; err == nil && i < len(paths)-1; i++ {
			err = os.RemoveAll(paths[i])
			if err != nil {
				break
			}
		}
	}
	return
}

func getKeyFile(keysDirPath string, keyAddr common.Address) (fileContent []byte, err error) {
	var keyFilePath string
	keyFilePath, err = getKeyFilePath(keysDirPath, keyAddr)
	if err == nil {
		fileContent, err = ioutil.ReadFile(keyFilePath)
	}
	return
}

func writeKeyFile(addr common.Address, keysDirPath string, content []byte) (err error) {
	filename := keyFileName(addr)
	// read, write and dir search for user
	err = os.MkdirAll(keysDirPath, 0700)
	if err != nil {
		return err
	}
	// read, write for user
	return ioutil.WriteFile(filepath.Join(keysDirPath, filename), content, 0600)
}

// keyFilePath implements the naming convention for keyfiles:
// UTC--<created_at UTC ISO8601>-<address hex>
func keyFileName(keyAddr common.Address) string {
	ts := time.Now().UTC()
	return fmt.Sprintf("UTC--%s--%s", toISO8601(ts), hex.EncodeToString(keyAddr[:]))
}

func toISO8601(t time.Time) string {
	var tz string
	name, offset := t.Zone()
	if name == "UTC" {
		tz = "Z"
	} else {
		tz = fmt.Sprintf("%03d00", offset/3600)
	}
	return fmt.Sprintf("%04d-%02d-%02dT%02d-%02d-%02d.%09d%s", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tz)
}

func getKeyAddresses(keysDirPath string) (addresses []common.Address, err error) {
	fileInfos, err := ioutil.ReadDir(keysDirPath)
	if err != nil {
		return nil, err
	}
	for _, fileInfo := range fileInfos {
		filename := fileInfo.Name()
		if len(filename) >= 40 {
			addr := filename[len(filename)-40 : len(filename)]
			address, err := hex.DecodeString(addr)
			if err == nil {
				addresses = append(addresses, common.BytesToAddress(address))
			}
		}
	}
	return addresses, err
}
