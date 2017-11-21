// Copyright 2015 The go-ethereum Authors
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

package discv5

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
)

type symmEncryption interface {
	encode(packet []byte) []byte
	decode(encPacket []byte) []byte
	maxDecodedLength() int
}

type Aes256Encryption struct {
	blockCipher cipher.Block
	randGen     *rand.Rand
	randLock    sync.Mutex
	maxLength   int
}

func newEcdhAes256Encryption(privKey *ecdsa.PrivateKey, pubKey *ecdsa.PublicKey, maxEncodedLength int) (*Aes256Encryption, error) {
	x, _ := crypto.S256().ScalarMult(pubKey.X, pubKey.Y, privKey.D.Bytes())
	key := x.Bytes()
	return newAes256Encryption(key, maxEncodedLength)
}

func newAes256Encryption(key []byte, maxEncodedLength int) (*Aes256Encryption, error) {
	if len(key) != 32 {
		panic(nil)
	}
	if cipher, err := aes.NewCipher(key); err == nil {
		var seedArr [8]byte
		crand.Read(seedArr[:])
		seed := int64(binary.BigEndian.Uint64(seedArr[:]))
		randGen := rand.New(rand.NewSource(seed))
		return &Aes256Encryption{blockCipher: cipher, randGen: randGen, maxLength: maxEncodedLength}, nil
	} else {
		return nil, err
	}
}

const (
	cipherIvLength = 16
	maxPadding     = 32
	tailSize       = 8
)

func (e *Aes256Encryption) encode(packet []byte) []byte {
	length := len(packet)
	maxpad := e.maxLength - length - cipherIvLength - tailSize
	if maxpad < 1 {
		// packet is too large, should be checked by caller
		panic(nil)
	}
	if maxpad > maxPadding {
		maxpad = maxPadding
	}
	e.randLock.Lock()
	padding := e.randGen.Intn(maxpad) + 1
	encLength := cipherIvLength + padding + length + tailSize
	dest := make([]byte, encLength)
	e.randGen.Read(dest[:cipherIvLength])
	dest[cipherIvLength] = byte(padding - 1)
	if padding > 1 {
		e.randGen.Read(dest[cipherIvLength+1 : cipherIvLength+padding])
	}
	e.randLock.Unlock()
	copy(dest[cipherIvLength+padding:encLength-tailSize], packet)
	integrityHash := crypto.Keccak256(dest[cipherIvLength : encLength-tailSize])
	copy(dest[encLength-tailSize:], integrityHash[:tailSize])
	cipher.NewCFBEncrypter(e.blockCipher, dest[:cipherIvLength]).XORKeyStream(dest[cipherIvLength:], dest[cipherIvLength:])
	return dest
}

func (e *Aes256Encryption) decode(encPacket []byte) []byte {
	paddedLength := len(encPacket) - cipherIvLength
	dest := make([]byte, paddedLength)
	cipher.NewCFBDecrypter(e.blockCipher, encPacket[:cipherIvLength]).XORKeyStream(dest, encPacket[cipherIvLength:])
	padding := int(dest[0]) + 1
	if padding > paddedLength-tailSize {
		return nil
	}
	// check packet integrity and reject if tail does not match hash
	if !bytes.Equal(dest[paddedLength-tailSize:], crypto.Keccak256(dest[:paddedLength-tailSize])[:tailSize]) {
		return nil
	}
	return dest[padding : paddedLength-tailSize]
}

func (e *Aes256Encryption) maxDecodedLength() int {
	return e.maxLength - cipherIvLength - tailSize - 1
}

const (
	rpMinLength = 100
	rpMaxLength = 1000
)

func newReconnectSeedAndHash() (int64, common.Hash) {
	var seedArr [8]byte
	crand.Read(seedArr[:])
	seed := int64(binary.BigEndian.Uint64(seedArr[:]))
	hash := crypto.Keccak256Hash(reconnectPacket(seed))
	return seed, hash
}

func reconnectPacket(seed int64) []byte {
	r := rand.New(rand.NewSource(seed))
	length := rpMinLength + r.Intn(rpMaxLength-rpMinLength+1)
	rp := make([]byte, length)
	r.Read(rp)
	return rp
}

type asymmEncryption interface {
	encode(packet []byte, pubKey *ecdsa.PublicKey) ([]byte, error) // will receive an ENR record instead of an ECDSA pubkey
	decode(encPacket []byte) []byte
	maxDecodedLength() int
}

type EciesEncryption struct {
	privKey                    *ecies.PrivateKey
	randGen                    *rand.Rand
	randLock                   sync.Mutex
	maxEncLength, maxDecLength int
}

func newEciesEncryption(privKey *ecdsa.PrivateKey, maxEncodedLength int) *EciesEncryption {
	var seedArr [8]byte
	crand.Read(seedArr[:])
	seed := int64(binary.BigEndian.Uint64(seedArr[:]))
	randGen := rand.New(rand.NewSource(seed))
	privateKey := ecies.ImportECDSA(privKey)
	testEnc, err := ecies.Encrypt(randGen, &privateKey.PublicKey, []byte{42}, nil, nil)
	if err != nil {
		panic(err)
	}
	maxDecodedLength := maxEncodedLength - len(testEnc) + 1

	return &EciesEncryption{
		privKey:      privateKey,
		randGen:      randGen,
		maxEncLength: maxEncodedLength,
		maxDecLength: maxDecodedLength,
	}
}

func (e *EciesEncryption) encode(packet []byte, pubKey *ecdsa.PublicKey) ([]byte, error) {
	if len(packet) > e.maxDecLength {
		panic(nil)
	}
	//TODO add random padding
	enc, err := ecies.Encrypt(e.randGen, ecies.ImportECDSAPublic(pubKey), packet, nil, nil)
	if len(enc) > e.maxEncLength {
		panic(nil)
	}
	return enc, err
}

func (e *EciesEncryption) decode(encPacket []byte) []byte {
	dec, err := e.privKey.Decrypt(e.randGen, encPacket, nil, nil)
	if err != nil {
		return nil
	}
	return dec
}

func (e *EciesEncryption) maxDecodedLength() int {
	return e.maxDecLength
}
