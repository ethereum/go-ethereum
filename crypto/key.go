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
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"io"

	"code.google.com/p/go-uuid/uuid"
	"github.com/ethereum/go-ethereum/common"
)

type Key struct {
	Id uuid.UUID // Version 4 "random" for unique id not derived from key data
	// to simplify lookups we also store the address
	Address common.Address
	// we only store privkey as pubkey/address can be derived from it
	// privkey in this struct is always in plaintext
	PrivateKey *ecdsa.PrivateKey
}

type plainKeyJSON struct {
	Id         []byte
	Address    []byte
	KeyHeader  keyHeaderJSON
	PrivateKey []byte
}

type encryptedKeyJSON struct {
	Id        []byte
	Address   []byte
	KeyHeader keyHeaderJSON
	Crypto    cipherJSON
}

type cipherJSON struct {
	MAC        []byte
	Salt       []byte
	IV         []byte
	CipherText []byte
}

type keyHeaderJSON struct {
	Version   string
	Kdf       string
	KdfParams *scryptParamsJSON // TODO: make more generic?
}

type scryptParamsJSON struct {
	N       int
	R       int
	P       int
	DkLen   int
	SaltLen int
}

func (k *Key) MarshalJSON() (j []byte, err error) {
	keyHeader := keyHeaderJSON{
		Version:   "1",
		Kdf:       "",
		KdfParams: nil,
	}
	jStruct := plainKeyJSON{
		k.Id,
		k.Address.Bytes(),
		keyHeader,
		FromECDSA(k.PrivateKey),
	}
	j, err = json.Marshal(jStruct)
	return j, err
}

func (k *Key) UnmarshalJSON(j []byte) (err error) {
	keyJSON := new(plainKeyJSON)
	err = json.Unmarshal(j, &keyJSON)
	if err != nil {
		return err
	}

	u := new(uuid.UUID)
	*u = keyJSON.Id
	k.Id = *u
	k.Address = common.BytesToAddress(keyJSON.Address)
	k.PrivateKey = ToECDSA(keyJSON.PrivateKey)

	return err
}

func NewKeyFromECDSA(privateKeyECDSA *ecdsa.PrivateKey) *Key {
	id := uuid.NewRandom()
	key := &Key{
		Id:         id,
		Address:    common.BytesToAddress(PubkeyToAddress(privateKeyECDSA.PublicKey)),
		PrivateKey: privateKeyECDSA,
	}
	return key
}

func NewKey(rand io.Reader) *Key {
	randBytes := make([]byte, 64)
	_, err := rand.Read(randBytes)
	if err != nil {
		panic("key generation: could not read from random source: " + err.Error())
	}
	reader := bytes.NewReader(randBytes)
	privateKeyECDSA, err := ecdsa.GenerateKey(S256(), reader)
	if err != nil {
		panic("key generation: ecdsa.GenerateKey failed: " + err.Error())
	}

	return NewKeyFromECDSA(privateKeyECDSA)
}
