// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// go-ethereum is free software: you can redistribute it and/or modify
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

package crypto

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
)

type KeyPair struct {
	PrivateKey []byte
	PublicKey  []byte
	address    []byte
	mnemonic   string
	// The associated account
	// account *StateObject
}

func GenerateNewKeyPair() *KeyPair {
	_, prv := secp256k1.GenerateKeyPair()
	keyPair, _ := NewKeyPairFromSec(prv) // swallow error, this one cannot err
	return keyPair
}

func NewKeyPairFromSec(seckey []byte) (*KeyPair, error) {
	pubkey, err := secp256k1.GeneratePubKey(seckey)
	if err != nil {
		return nil, err
	}

	return &KeyPair{PrivateKey: seckey, PublicKey: pubkey}, nil
}

func (k *KeyPair) Address() []byte {
	if k.address == nil {
		k.address = Sha3(k.PublicKey[1:])[12:]
	}
	return k.address
}

func (k *KeyPair) Mnemonic() string {
	if k.mnemonic == "" {
		k.mnemonic = strings.Join(MnemonicEncode(common.Bytes2Hex(k.PrivateKey)), " ")
	}
	return k.mnemonic
}

func (k *KeyPair) AsStrings() (string, string, string, string) {
	return k.Mnemonic(), common.Bytes2Hex(k.Address()), common.Bytes2Hex(k.PrivateKey), common.Bytes2Hex(k.PublicKey)
}
