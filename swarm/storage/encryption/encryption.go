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
)

const KeyLength = 32

type Key []byte

type Encryption interface {
	Encrypt(data []byte, key Key) ([]byte, error)
	Decrypt(data []byte, key Key) ([]byte, error)
}

type encryption struct {
	padding  int
	initCtr  uint32
	hashFunc func() hash.Hash
}

func New(padding int, initCtr uint32, hashFunc func() hash.Hash) *encryption {
	return &encryption{
		padding:  padding,
		initCtr:  initCtr,
		hashFunc: hashFunc,
	}
}

func (e *encryption) Encrypt(data []byte, key Key) ([]byte, error) {
	length := len(data)
	isFixedPadding := e.padding > 0
	if isFixedPadding && length > e.padding {
		return nil, fmt.Errorf("Data length longer than padding, data length %v padding %v", length, e.padding)
	}

	paddedData := data
	if isFixedPadding && length < e.padding {
		paddedData = make([]byte, e.padding)
		copy(paddedData[:length], data)
		rand.Read(paddedData[length:])
	}
	return e.transform(paddedData, key), nil
}

func (e *encryption) Decrypt(data []byte, key Key) ([]byte, error) {
	length := len(data)
	if e.padding > 0 && length != e.padding {
		return nil, fmt.Errorf("Data length different than padding, data length %v padding %v", length, e.padding)
	}

	return e.transform(data, key), nil
}

func (e *encryption) transform(data []byte, key Key) []byte {
	dataLength := len(data)
	transformedData := make([]byte, dataLength)
	hasher := e.hashFunc()
	ctr := e.initCtr
	hashSize := hasher.Size()
	for i := 0; i < dataLength; i += hashSize {
		hasher.Write(key)

		ctrBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(ctrBytes, ctr)

		hasher.Write(ctrBytes)

		ctrHash := hasher.Sum(nil)
		hasher.Reset()
		hasher.Write(ctrHash)

		segmentKey := hasher.Sum(nil)

		hasher.Reset()

		segmentSize := min(hashSize, dataLength-i)
		for j := 0; j < segmentSize; j++ {
			transformedData[i+j] = data[i+j] ^ segmentKey[j]
		}
		ctr++
	}
	return transformedData
}

func GenerateRandomKey() (Key, error) {
	key := make([]byte, KeyLength)
	_, err := rand.Read(key)
	return key, err
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
