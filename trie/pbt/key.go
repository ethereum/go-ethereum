// Copyright 2026 The go-ethereum Authors
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

package pbt

import (
	"encoding/binary"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/zeebo/blake3"
)

// EIP-8297 tree embedding. A key is zone ‖ tree position ‖ sub-index; its stem
// is the key without the sub-index.
const (
	accountZone = 0x00
	codeZone    = 0x01
	storageZone = 0xff

	basicDataLeafKey    = 0
	codeHashLeafKey     = 1
	delegationLeafKey   = 2
	headerStorageOffset = 64
	headerStorageSlots  = 64
	stemSubtreeWidth    = 256

	accountKeyLength = 34
	codeKeyLength    = 34
	storageKeyLength = 66

	// maxPathBits is the longest key in bits, bounding every branch prefix.
	maxPathBits = 8 * storageKeyLength
)

var errBalanceOverflow = errors.New("pbt: balance exceeds 16 bytes")

func keyHash(data []byte) [32]byte {
	return blake3.Sum256(data)
}

func address32(addr common.Address) (a [32]byte) {
	copy(a[12:], addr[:])
	return a
}

// zoneStem returns zone ‖ key_hash(address32), the account's header stem in
// the account zone and its storage bucket in the storage zone.
func zoneStem(zone byte, addr common.Address) []byte {
	a := address32(addr)
	h := keyHash(a[:])
	return append(append(make([]byte, 0, storageKeyLength), zone), h[:]...)
}

func headerStem(addr common.Address) []byte {
	return zoneStem(accountZone, addr)
}

// storageKey returns the tree key of a raw storage slot.
func storageKey(addr common.Address, slot []byte) []byte {
	var key [32]byte
	copy(key[32-len(slot):], slot)

	var index uint256.Int
	index.SetBytes32(key[:])
	if index.LtUint64(headerStorageSlots) {
		return append(headerStem(addr), headerStorageOffset+key[31])
	}
	var buf [64]byte
	a := address32(addr)
	copy(buf[:32], a[:])
	index.Rsh(&index, 8).PutUint256(buf[32:])
	h := keyHash(buf[:])
	return append(append(zoneStem(storageZone, addr), h[:]...), key[31])
}

// codeStem returns the stem of the content-addressed code group at index.
func codeStem(codeHash common.Hash, index uint64) []byte {
	var buf [64]byte
	copy(buf[:32], codeHash[:])
	binary.BigEndian.PutUint64(buf[56:], index)
	h := keyHash(buf[:])
	return append(append(make([]byte, 0, codeKeyLength), codeZone), h[:]...)
}

// validateStem checks that a stem has its zone's length.
func validateStem(stem []byte) error {
	if len(stem) > 0 {
		switch {
		case stem[0] == accountZone && len(stem) == accountKeyLength-1,
			stem[0] == codeZone && len(stem) == codeKeyLength-1,
			stem[0] == storageZone && len(stem) == storageKeyLength-1:
			return nil
		}
	}
	return errInvalidKey
}

// encodeBasicData packs version(1) ‖ reserved(3) ‖ code_size(4) ‖ nonce(8) ‖
// balance(16), with version and reserved bytes zero.
func encodeBasicData(codeSize uint32, nonce uint64, balance *uint256.Int) ([]byte, error) {
	if balance.BitLen() > 128 {
		return nil, errBalanceOverflow
	}
	var out [32]byte
	binary.BigEndian.PutUint32(out[4:], codeSize)
	binary.BigEndian.PutUint64(out[8:], nonce)
	b := balance.Bytes32()
	copy(out[16:], b[16:])
	return out[:], nil
}

func decodeBasicData(v []byte) (codeSize uint32, nonce uint64, balance *uint256.Int) {
	return binary.BigEndian.Uint32(v[4:]), binary.BigEndian.Uint64(v[8:]), new(uint256.Int).SetBytes(v[16:32])
}
