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

	ns_ipfs  = 0xe3
	ns_swarm = 0xe4

	swarm_typecode = 0x99 //todo change
	swarm_hashtype = 0x1b //todo change

	ipfs_hashtype = 0x12

	hash_length = 32
)

// deocodeEIP1577ContentHash decodes a chain-stored content hash from an ENS record according to EIP-1577
// a successful decode will result the different parts of the content hash in accordance to the CID spec
// Note: only CIDv1 is supported
func decodeEIP1577ContentHash(buf []byte) (storageNs, contentType, hashType, hashLength uint64, hash []byte, err error) {
	if len(buf) < 10 {
		return 0, 0, 0, 0, nil, fmt.Errorf("buffer too short")
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
	storageNs, contentType, hashType, hashLength, hashBytes, err := decodeEIP1577ContentHash(buf)

	if err != nil {
		return common.Hash{}, err
	}

	if storageNs != ns_swarm {
		return common.Hash{}, errors.New("unknown storage system")
	}

	if contentType != swarm_typecode { //todo pending pr
		return common.Hash{}, errors.New("unknown content type")
	}

	if hashType != swarm_hashtype { //todo: should be bmt
		return common.Hash{}, errors.New("unknown multihash type")
	}

	if hashLength != hash_length {
		return common.Hash{}, errors.New("odd hash length, swarm expects 32 bytes")
	}

	if len(hashBytes) != int(hashLength) {
		return common.Hash{}, errors.New("hash length mismatch")
	}

	return common.BytesToHash(buf), nil
}

func encodeSwarmHash(hash common.Hash) ([]byte, error) {
	var cidBytes []byte
	var headerBytes = []byte{
		ns_swarm,       //swarm namespace
		cidv1,          // CIDv1
		swarm_typecode, // the swarm type-code
		swarm_hashtype, // swarm hash type. todo BMT
		hash_length,    //hash length. 32 bytes
	}

	varintbuf := make([]byte, binary.MaxVarintLen64)
	for _, v := range headerBytes {
		n := binary.PutUvarint(varintbuf, uint64(v))
		cidBytes = append(cidBytes, varintbuf[:n]...)
	}

	cidBytes = append(cidBytes, hash[:]...)
	return cidBytes, nil
}

// encodeCid encodes a swarm hash into an IPLD CID
/*func encodeCid(h common.Hash) (cid.Cid, error) {
	b := []byte{0x1b, 0x20}     //0x1b = keccak256 (should be changed to bmt), 0x20 = 32 bytes hash length
	b = append(b, h.Bytes()...) // append actual hash bytes
	multi, err := mh.Cast(b)
	if err != nil {
		return cid.Cid{}, err
	}

	c := cid.NewCidV1(cid.Raw, multi) //todo: cid.Raw should be swarm manifest

	return c, nil
}*/
