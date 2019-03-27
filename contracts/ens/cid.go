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

package ens

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

const (
	cidv1 = 0x1

	nsIpfs  = 0xe3
	nsSwarm = 0xe4

	swarmTypecode = 0xfa // swarm manifest, see https://github.com/multiformats/multicodec/blob/master/table.csv
	swarmHashtype = 0x1b // keccak256, see https://github.com/multiformats/multicodec/blob/master/table.csv

	hashLength = 32
)

// deocodeEIP1577ContentHash decodes a chain-stored content hash from an ENS record according to EIP-1577
// a successful decode will result the different parts of the content hash in accordance to the CID spec
// Note: only CIDv1 is supported
func decodeEIP1577ContentHash(buf []byte) (storageNs, contentType, hashType, hashLength uint64, hash []byte, err error) {
	if len(buf) < 10 {
		return 0, 0, 0, 0, nil, errors.New("buffer too short")
	}

	storageNs, n := binary.Uvarint(buf)

	buf = buf[n:]
	vers, n := binary.Uvarint(buf)

	if vers != 1 {
		return 0, 0, 0, 0, nil, fmt.Errorf("expected cid v1, got: %d", vers)
	}
	buf = buf[n:]
	contentType, n = binary.Uvarint(buf)

	buf = buf[n:]
	hashType, n = binary.Uvarint(buf)

	buf = buf[n:]
	hashLength, n = binary.Uvarint(buf)

	hash = buf[n:]

	if len(hash) != int(hashLength) {
		return 0, 0, 0, 0, nil, errors.New("hash length mismatch")
	}
	return storageNs, contentType, hashType, hashLength, hash, nil
}

func extractContentHash(buf []byte) (common.Hash, error) {
	storageNs, _ /*contentType*/, _ /* hashType*/, decodedHashLength, hashBytes, err := decodeEIP1577ContentHash(buf)

	if err != nil {
		return common.Hash{}, err
	}

	if storageNs != nsSwarm {
		return common.Hash{}, errors.New("unknown storage system")
	}

	//todo: for the time being we implement loose enforcement for the EIP rules until ENS manager is updated
	/*if contentType != swarmTypecode {
		return common.Hash{}, errors.New("unknown content type")
	}

	if hashType != swarmHashtype {
		return common.Hash{}, errors.New("unknown multihash type")
	}*/

	if decodedHashLength != hashLength {
		return common.Hash{}, errors.New("odd hash length, swarm expects 32 bytes")
	}

	if len(hashBytes) != int(hashLength) {
		return common.Hash{}, errors.New("hash length mismatch")
	}

	return common.BytesToHash(buf), nil
}

func EncodeSwarmHash(hash common.Hash) ([]byte, error) {
	var cidBytes []byte
	var headerBytes = []byte{
		nsSwarm,       //swarm namespace
		cidv1,         // CIDv1
		swarmTypecode, // swarm hash
		swarmHashtype, // keccak256 hash
		hashLength,    //hash length. 32 bytes
	}

	varintbuf := make([]byte, binary.MaxVarintLen64)
	for _, v := range headerBytes {
		n := binary.PutUvarint(varintbuf, uint64(v))
		cidBytes = append(cidBytes, varintbuf[:n]...)
	}

	cidBytes = append(cidBytes, hash[:]...)
	return cidBytes, nil
}
