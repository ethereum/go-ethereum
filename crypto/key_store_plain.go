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
	"code.google.com/p/go-uuid/uuid"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path"
)

// TODO: rename to KeyStore when replacing existing KeyStore
type KeyStore2 interface {
	// create new key using io.Reader entropy source and optionally using auth string
	GenerateNewKey(io.Reader, string) (*Key, error)
	GetKey(*uuid.UUID, string) (*Key, error) // key from id and auth string
	StoreKey(*Key, string) error             // store key optionally using auth string
	DeleteKey(*uuid.UUID, string) error      // delete key by id and auth string
}

type keyStorePlain struct {
	keysDirPath string
}

// TODO: copied from cmd/ethereum/flags.go
func DefaultDataDir() string {
	usr, _ := user.Current()
	return path.Join(usr.HomeDir, ".ethereum")
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

func (ks keyStorePlain) GetKey(keyId *uuid.UUID, auth string) (key *Key, err error) {
	fileContent, err := GetKeyFile(ks.keysDirPath, keyId)
	if err != nil {
		return nil, err
	}

	key = new(Key)
	err = json.Unmarshal(fileContent, key)
	return key, err
}

func (ks keyStorePlain) StoreKey(key *Key, auth string) (err error) {
	keyJSON, err := json.Marshal(key)
	if err != nil {
		return err
	}
	err = WriteKeyFile(key.Id.String(), ks.keysDirPath, keyJSON)
	return err
}

func (ks keyStorePlain) DeleteKey(keyId *uuid.UUID, auth string) (err error) {
	keyDirPath := path.Join(ks.keysDirPath, keyId.String())
	err = os.RemoveAll(keyDirPath)
	return err
}

func GetKeyFile(keysDirPath string, keyId *uuid.UUID) (fileContent []byte, err error) {
	id := keyId.String()
	return ioutil.ReadFile(path.Join(keysDirPath, id, id))
}

func WriteKeyFile(id string, keysDirPath string, content []byte) (err error) {
	keyDirPath := path.Join(keysDirPath, id)
	keyFilePath := path.Join(keyDirPath, id)
	err = os.MkdirAll(keyDirPath, 0700) // read, write and dir search for user
	if err != nil {
		return err
	}
	return ioutil.WriteFile(keyFilePath, content, 0600) // read, write for user
}
