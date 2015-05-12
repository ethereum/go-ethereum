/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU Lesser General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU Lesser General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors
 * 	Gustav Simonsson <gustav.simonsson@gmail.com>
 * @date 2015
 *
 */

package crypto

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// TODO: rename to KeyStore when replacing existing KeyStore
type KeyStore2 interface {
	// create new key using io.Reader entropy source and optionally using auth string
	GenerateNewKey(io.Reader, string) (*Key, error)
	GetKey([]byte, string) (*Key, error) // key from addr and auth string
	GetKeyAddresses() ([][]byte, error)  // get all addresses
	StoreKey(*Key, string) error         // store key optionally using auth string
	DeleteKey([]byte, string) error      // delete key by addr and auth string
}

type keyStorePlain struct {
	keysDirPath string
}

func NewKeyStorePlain(path string) KeyStore2 {
	return &keyStorePlain{path}
}

func (ks keyStorePlain) GenerateNewKey(rand io.Reader, auth string) (key *Key, err error) {
	return GenerateNewKeyDefault(ks, rand, auth)
}

func GenerateNewKeyDefault(ks KeyStore2, rand io.Reader, auth string) (key *Key, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("GenerateNewKey error: %v", r)
		}
	}()
	key = NewKey(rand)
	err = ks.StoreKey(key, auth)
	return key, err
}

func (ks keyStorePlain) GetKey(keyAddr []byte, auth string) (key *Key, err error) {
	fileContent, err := GetKeyFile(ks.keysDirPath, keyAddr)
	if err != nil {
		return nil, err
	}

	key = new(Key)
	err = json.Unmarshal(fileContent, key)
	return key, err
}

func (ks keyStorePlain) GetKeyAddresses() (addresses [][]byte, err error) {
	return GetKeyAddresses(ks.keysDirPath)
}

func (ks keyStorePlain) StoreKey(key *Key, auth string) (err error) {
	keyJSON, err := json.Marshal(key)
	if err != nil {
		return err
	}
	err = WriteKeyFile(key.Address, ks.keysDirPath, keyJSON)
	return err
}

func (ks keyStorePlain) DeleteKey(keyAddr []byte, auth string) (err error) {
	keyDirPath := filepath.Join(ks.keysDirPath, hex.EncodeToString(keyAddr))
	err = os.RemoveAll(keyDirPath)
	return err
}

func GetKeyFile(keysDirPath string, keyAddr []byte) (fileContent []byte, err error) {
	fileName := hex.EncodeToString(keyAddr)
	return ioutil.ReadFile(filepath.Join(keysDirPath, fileName, fileName))
}

func WriteKeyFile(addr []byte, keysDirPath string, content []byte) (err error) {
	addrHex := hex.EncodeToString(addr)
	keyDirPath := filepath.Join(keysDirPath, addrHex)
	keyFilePath := filepath.Join(keyDirPath, addrHex)
	err = os.MkdirAll(keyDirPath, 0700) // read, write and dir search for user
	if err != nil {
		return err
	}
	return ioutil.WriteFile(keyFilePath, content, 0600) // read, write for user
}

func GetKeyAddresses(keysDirPath string) (addresses [][]byte, err error) {
	fileInfos, err := ioutil.ReadDir(keysDirPath)
	if err != nil {
		return nil, err
	}
	for _, fileInfo := range fileInfos {
		address, err := hex.DecodeString(fileInfo.Name())
		if err != nil {
			continue
		}
		addresses = append(addresses, address)
	}
	return addresses, err
}
