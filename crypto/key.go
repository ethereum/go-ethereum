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
	"code.google.com/p/go-uuid/uuid"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/json"
	"io"
)

type Key struct {
	Id *uuid.UUID // Version 4 "random" for unique id not derived from key data
	// we only store privkey as pubkey/address can be derived from it
	// privkey in this struct is always in plaintext
	PrivateKey *ecdsa.PrivateKey
}

type plainKeyJSON struct {
	Id         []byte
	PrivateKey []byte
}

type cipherJSON struct {
	Salt       []byte
	IV         []byte
	CipherText []byte
}

type encryptedKeyJSON struct {
	Id     []byte
	Crypto cipherJSON
}

func (k *Key) Address() []byte {
	pubBytes := FromECDSAPub(&k.PrivateKey.PublicKey)
	return Sha3(pubBytes[1:])[12:]
}

func (k *Key) MarshalJSON() (j []byte, err error) {
	jStruct := plainKeyJSON{
		*k.Id,
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
	k.Id = u

	k.PrivateKey = ToECDSA(keyJSON.PrivateKey)

	return err
}

func NewKey(rand io.Reader) *Key {
	randBytes := make([]byte, 32)
	_, err := rand.Read(randBytes)
	if err != nil {
		panic("key generation: could not read from random source: " + err.Error())
	}
	reader := bytes.NewReader(randBytes)
	_, x, y, err := elliptic.GenerateKey(S256(), reader)
	if err != nil {
		panic("key generation: elliptic.GenerateKey failed: " + err.Error())
	}
	privateKeyMarshalled := elliptic.Marshal(S256(), x, y)
	privateKeyECDSA := ToECDSA(privateKeyMarshalled)

	id := uuid.NewRandom()
	key := &Key{
		Id:         &id,
		PrivateKey: privateKeyECDSA,
	}
	return key
}
