// Copyright 2016 The go-ethereum Authors
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

package api

import (
	"encoding/binary"
	"errors"

	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/swarm/storage/encryption"
)

type RefEncryption struct {
	spanEncryption encryption.Encryption
	dataEncryption encryption.Encryption
	span           []byte
}

func NewRefEncryption(refSize int) *RefEncryption {
	span := make([]byte, 8)
	binary.LittleEndian.PutUint64(span, uint64(refSize))
	return &RefEncryption{
		spanEncryption: encryption.New(0, uint32(refSize/32), sha3.NewKeccak256),
		dataEncryption: encryption.New(refSize, 0, sha3.NewKeccak256),
		span:           span,
	}
}

func (re *RefEncryption) Encrypt(ref []byte, key []byte) ([]byte, error) {
	encryptedSpan, err := re.spanEncryption.Encrypt(re.span, key)
	if err != nil {
		return nil, err
	}
	encryptedData, err := re.dataEncryption.Encrypt(ref, key)
	if err != nil {
		return nil, err
	}
	encryptedRef := make([]byte, len(ref)+8)
	copy(encryptedRef[:8], encryptedSpan)
	copy(encryptedRef[8:], encryptedData)

	return encryptedRef, nil
}

func (re *RefEncryption) Decrypt(ref []byte, key []byte) ([]byte, error) {
	decryptedSpan, err := re.spanEncryption.Decrypt(ref[:8], key)
	if err != nil {
		return nil, err
	}

	size := binary.LittleEndian.Uint64(decryptedSpan)
	if size != uint64(len(ref)-8) {
		return nil, errors.New("invalid span in encrypted reference")
	}

	decryptedRef, err := re.dataEncryption.Decrypt(ref[8:], key)
	if err != nil {
		return nil, err
	}

	return decryptedRef, nil
}
