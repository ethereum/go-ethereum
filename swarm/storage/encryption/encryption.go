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

package encryption

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"hash"
	"sync"
)

const KeyLength = 32

type Key []byte

type Encryption interface {
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
}

type encryption struct {
	key      Key              // the encryption key (hashSize bytes long)
	keyLen   int              // length of the key = length of blockcipher block
	padding  int              // encryption will pad the data upto this if > 0
	initCtr  uint32           // initial counter used for counter mode blockcipher
	hashFunc func() hash.Hash // hasher constructor function
}

// New constructs a new encryptor/decryptor
func New(key Key, padding int, initCtr uint32, hashFunc func() hash.Hash) *encryption {
	return &encryption{
		key:      key,
		keyLen:   len(key),
		padding:  padding,
		initCtr:  initCtr,
		hashFunc: hashFunc,
	}
}

// Encrypt encrypts the data and does padding if specified
func (e *encryption) Encrypt(data []byte) ([]byte, error) {
	length := len(data)
	outLength := length
	isFixedPadding := e.padding > 0
	if isFixedPadding {
		if length > e.padding {
			return nil, fmt.Errorf("Data length longer than padding, data length %v padding %v", length, e.padding)
		}
		outLength = e.padding
	}
	out := make([]byte, outLength)
	e.transform(data, out)
	return out, nil
}

// Decrypt decrypts the data, if padding was used caller must know original length and truncate
func (e *encryption) Decrypt(data []byte) ([]byte, error) {
	length := len(data)
	if e.padding > 0 && length != e.padding {
		return nil, fmt.Errorf("Data length different than padding, data length %v padding %v", length, e.padding)
	}
	out := make([]byte, length)
	e.transform(data, out)
	return out, nil
}

//
func (e *encryption) transform(in, out []byte) {
	inLength := len(in)
	wg := sync.WaitGroup{}
	wg.Add((inLength-1)/e.keyLen + 1)
	for i := 0; i < inLength; i += e.keyLen {
		l := min(e.keyLen, inLength-i)
		// call transformations per segment (asyncronously)
		go func(i int, x, y []byte) {
			defer wg.Done()
			e.Transcrypt(i, x, y)
		}(i/e.keyLen, in[i:i+l], out[i:i+l])
	}
	// pad the rest if out is longer
	pad(out[inLength:])
	wg.Wait()
}

// used for segmentwise transformation
// if in is shorter than out, padding is used
func (e *encryption) Transcrypt(i int, in []byte, out []byte) {
	// first hash key with counter (initial counter + i)
	hasher := e.hashFunc()
	hasher.Write(e.key)

	ctrBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(ctrBytes, uint32(i)+e.initCtr)
	hasher.Write(ctrBytes)

	ctrHash := hasher.Sum(nil)
	hasher.Reset()

	// second round of hashing for selective disclosure
	hasher.Write(ctrHash)
	segmentKey := hasher.Sum(nil)
	hasher.Reset()

	// XOR bytes uptil length of in (out must be at least as long)
	inLength := len(in)
	for j := 0; j < inLength; j++ {
		out[j] = in[j] ^ segmentKey[j]
	}
	// insert padding if out is longer
	pad(out[inLength:])
}

func pad(b []byte) {
	l := len(b)
	for total := 0; total < l; {
		read, _ := rand.Read(b[total:])
		total += read
	}
}

// GenerateRandomKey generates a random key of length l
func GenerateRandomKey(l int) Key {
	key := make([]byte, l)
	var total int
	for total < l {
		read, _ := rand.Read(key[total:])
		total += read
	}
	return key
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
