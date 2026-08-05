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

package bintrie

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/blake3"
	"github.com/holiman/uint256"
)

// The EIP-8297 state embedding: how accounts, storage slots and code chunks
// map to tree keys and 32-byte values. Mirrors the EELS reference
// (ethereum/binary_trie/embedding.py) byte for byte.
//
// Every key is zone byte ‖ hash-derived tree position ‖ sub-index byte, with
// a fixed length per zone; a key's stem is everything except the final
// sub-index byte. Fixed per-zone lengths keep the key set prefix-free.

const (
	// Zone identifiers, the first byte of every key.
	AccountZone byte = 0x00 // account header stems
	CodeZone    byte = 0x01 // content-addressed code
	StorageZone byte = 0xFF // overflow storage

	// Sub-indices within an account's header stem.
	BasicDataLeafKey = 0 // version ‖ reserved ‖ code_size ‖ nonce ‖ balance
	CodeHashLeafKey  = 1 // keccak256 of the account's code

	// HeaderStorageOffset is the header sub-index of storage slot 0, and
	// HeaderStorageSlots how many slots live there: slots 0..63 at sub-indices
	// 64..127. Code used to occupy 128..255 and no longer does.
	HeaderStorageOffset = 64
	HeaderStorageSlots  = 64
	// StemSubtreeWidth is the number of values grouped under one stem.
	StemSubtreeWidth = 256

	// Per-zone key lengths in bytes.
	AccountKeyLength = 34
	CodeKeyLength    = 34
	StorageKeyLength = 66

	// maxPathBits is the deepest a key can be walked: the longest zone key,
	// in bits. Every branch prefix is a fragment of one such path, so no
	// prefix can be longer than this.
	//
	// EIP-8297's "Maximum key length" permits far more - the two-byte prefix
	// count admits 65535 bits - but this engine only ever stores keys of the
	// three zone lengths, so a record claiming more than this cannot have been
	// produced by it and is refused on the way in.
	maxPathBits = 8 * StorageKeyLength
)

func init() {
	// Required invariant of the embedding (EIP-8297 "Tree embedding").
	if !(HeaderStorageOffset > CodeHashLeafKey && HeaderStorageOffset+HeaderStorageSlots <= StemSubtreeWidth) {
		panic("bintrie: invalid header layout constants")
	}
}

// ErrBalanceOverflow is returned when an account balance does not fit the
// 16-byte BASIC_DATA field.
var ErrBalanceOverflow = errors.New("bintrie: balance exceeds 16 bytes")

// Address32 converts a 20-byte address to the 32-byte form used in key
// derivation, prepending 12 zero bytes.
func Address32(addr common.Address) (a32 [32]byte) {
	copy(a32[12:], addr[:])
	return a32
}

// KeyHash is the tree's key-derivation hash (the same function as node
// merkelization, per the spec).
func KeyHash(data []byte) [32]byte {
	return blake3.Sum256(data)
}

// HeaderStem returns the stem of addr's account header: AccountZone ‖
// KeyHash(addr32), 33 bytes. Every account has exactly one header stem.
func HeaderStem(addr common.Address) []byte {
	a32 := Address32(addr)
	h := KeyHash(a32[:])
	stem := make([]byte, 0, AccountKeyLength-1)
	stem = append(stem, AccountZone)
	return append(stem, h[:]...)
}

// HeaderKey returns the account header key at the given sub-index.
func HeaderKey(addr common.Address, sub byte) []byte {
	return append(HeaderStem(addr), sub)
}

// BasicDataKey returns the key of addr's BASIC_DATA leaf.
func BasicDataKey(addr common.Address) []byte {
	return HeaderKey(addr, BasicDataLeafKey)
}

// CodeHashKey returns the key of addr's code-hash leaf.
func CodeHashKey(addr common.Address) []byte {
	return HeaderKey(addr, CodeHashLeafKey)
}

// StorageBucketPrefix returns the 33-byte prefix under which all of addr's
// overflow storage lives: StorageZone ‖ KeyHash(addr32). The subtree below
// it is the account's storage bucket.
func StorageBucketPrefix(addr common.Address) []byte {
	a32 := Address32(addr)
	h := KeyHash(a32[:])
	p := make([]byte, 0, 33)
	p = append(p, StorageZone)
	return append(p, h[:]...)
}

// StorageStem returns the stem of addr's overflow storage group at
// treeIndex: StorageZone ‖ KeyHash(addr32) ‖ KeyHash(addr32 ‖ treeIndex32),
// 65 bytes.
func StorageStem(addr common.Address, treeIndex *uint256.Int) []byte {
	a32 := Address32(addr)
	prefix := KeyHash(a32[:])

	var buf [64]byte
	copy(buf[:32], a32[:])
	treeIndex.PutUint256(buf[32:])
	suffix := KeyHash(buf[:])

	stem := make([]byte, 0, StorageKeyLength-1)
	stem = append(stem, StorageZone)
	stem = append(stem, prefix[:]...)
	return append(stem, suffix[:]...)
}

// StorageIndex resolves a raw storage slot key (at most 32 bytes, big
// endian) to its tree coordinates: header placement for slots below 64,
// otherwise the overflow group index and sub-index.
func StorageIndex(slot []byte) (inHeader bool, treeIndex uint256.Int, sub byte) {
	var padded [32]byte
	copy(padded[32-len(slot):], slot)

	headerCandidate := padded[31] < HeaderStorageOffset
	if headerCandidate {
		for _, b := range padded[:31] {
			if b != 0 {
				headerCandidate = false
				break
			}
		}
	}
	if headerCandidate {
		return true, treeIndex, HeaderStorageOffset + padded[31]
	}
	treeIndex.SetBytes(padded[:])
	treeIndex.Rsh(&treeIndex, 8) // slot / StemSubtreeWidth
	return false, treeIndex, padded[31]
}

// StorageSlotKey returns the tree key of a raw storage slot. Slots below 64
// live in the account header stem; the rest in the storage zone.
func StorageSlotKey(addr common.Address, slot []byte) []byte {
	inHeader, treeIndex, sub := StorageIndex(slot)
	if inHeader {
		return HeaderKey(addr, sub)
	}
	return append(StorageStem(addr, &treeIndex), sub)
}

// CodeChunkIndex resolves a code chunk number to its tree coordinates: the
// content-addressed group index and the sub-index within it.
func CodeChunkIndex(chunk uint64) (treeIndex uint64, sub byte) {
	return chunk / StemSubtreeWidth, byte(chunk % StemSubtreeWidth)
}

// CodeChunkStem returns the stem of a content-addressed code group:
// CodeZone ‖ KeyHash(codeHash ‖ treeIndex32), 33 bytes. Contracts with
// identical bytecode share these stems.
func CodeChunkStem(codeHash common.Hash, treeIndex uint64) []byte {
	var buf [64]byte
	copy(buf[:32], codeHash[:])
	new(uint256.Int).SetUint64(treeIndex).PutUint256(buf[32:])
	h := KeyHash(buf[:])
	stem := make([]byte, 0, CodeKeyLength-1)
	stem = append(stem, CodeZone)
	return append(stem, h[:]...)
}

// CodeChunkKey returns the tree key of code chunk number chunk of the code
// with hash codeHash. No address takes part: chunks are content-addressed, so
// contracts with identical bytecode share them.
func CodeChunkKey(codeHash common.Hash, chunk uint64) []byte {
	treeIndex, sub := CodeChunkIndex(chunk)
	return append(CodeChunkStem(codeHash, treeIndex), sub)
}

// validateStem checks a stem (a key without its sub-index byte) against the
// engine's key-conformance restriction, without materializing the key.
func validateStem(stem []byte) error {
	if len(stem) == 0 {
		return ErrNonConformantKey
	}
	switch stem[0] {
	case AccountZone:
		if len(stem) != AccountKeyLength-1 {
			return ErrNonConformantKey
		}
	case CodeZone:
		if len(stem) != CodeKeyLength-1 {
			return ErrNonConformantKey
		}
	case StorageZone:
		if len(stem) != StorageKeyLength-1 {
			return ErrNonConformantKey
		}
	default:
		return ErrNonConformantKey
	}
	return nil
}

// validateKey checks the engine's key-conformance restriction: known zone
// byte and the zone's exact length. This is what keeps stems non-nested
// (equal-length stems cannot prefix each other; zones differ in byte 0).
func validateKey(key []byte) error {
	if len(key) == 0 {
		return ErrNonConformantKey
	}
	switch key[0] {
	case AccountZone:
		if len(key) != AccountKeyLength {
			return ErrNonConformantKey
		}
	case CodeZone:
		if len(key) != CodeKeyLength {
			return ErrNonConformantKey
		}
	case StorageZone:
		if len(key) != StorageKeyLength {
			return ErrNonConformantKey
		}
	default:
		return ErrNonConformantKey
	}
	return nil
}

// ChunkedCode represents a sequence of 32-byte chunks of code (31 bytes of
// which are code data and 1 byte is the pushdata offset).
type ChunkedCode []byte

// Copy of the values that determine the stack item count of an instruction.
const (
	PUSH_OFFSET = 95
	PUSH1       = PUSH_OFFSET + 1
	PUSH32      = PUSH_OFFSET + 32
)

// ChunkifyCode generates the chunked version of an array representing EVM
// bytecode. Chunk i holds the i'th 31-byte slice of the code in bytes 1..31,
// preceded by one byte counting how many of the slice's leading bytes are
// data of a push instruction that began in an earlier chunk (capped at 31).
func ChunkifyCode(code []byte) ChunkedCode {
	var (
		chunkOffset = 0 // offset in the chunk
		chunkCount  = len(code) / 31
		codeOffset  = 0 // offset in the code
	)
	if len(code)%31 != 0 {
		chunkCount++
	}
	chunks := make([]byte, chunkCount*32)
	for i := 0; i < chunkCount; i++ {
		// number of bytes to copy, 31 unless the end of the code has been
		// reached.
		end := 31 * (i + 1)
		if len(code) < end {
			end = len(code)
		}
		copy(chunks[i*32+1:], code[31*i:end]) // copy the code itself

		// chunk offset = taken from the last chunk.
		if chunkOffset > 31 {
			// skip offset calculation if push data covers the whole chunk
			chunks[i*32] = 31
			chunkOffset = 1
			continue
		}
		chunks[32*i] = byte(chunkOffset)
		chunkOffset = 0

		// Check each instruction and update the offset it should be 0 unless
		// a PUSH-N overflows.
		for ; codeOffset < end; codeOffset++ {
			if code[codeOffset] >= PUSH1 && code[codeOffset] <= PUSH32 {
				codeOffset += int(code[codeOffset] - PUSH_OFFSET)
				if codeOffset+1 >= 31*(i+1) {
					codeOffset++
					chunkOffset = codeOffset - 31*(i+1)
					break
				}
			}
		}
	}
	return chunks
}

// EncodeBasicData packs an account's basic data into the 32-byte BASIC_DATA
// value: version(1) ‖ reserved(3) ‖ code_size(4, BE) ‖ nonce(8, BE) ‖
// balance(16, BE). The version and reserved bytes are zero; any rewrite of
// the leaf re-encodes all 32 bytes.
func EncodeBasicData(codeSize uint32, nonce uint64, balance *uint256.Int) ([32]byte, error) {
	var out [32]byte
	if balance != nil {
		if balance.BitLen() > 128 {
			return out, ErrBalanceOverflow
		}
		var full [32]byte
		balance.PutUint256(full[:])
		copy(out[16:], full[16:])
	}
	out[4] = byte(codeSize >> 24)
	out[5] = byte(codeSize >> 16)
	out[6] = byte(codeSize >> 8)
	out[7] = byte(codeSize)
	out[8] = byte(nonce >> 56)
	out[9] = byte(nonce >> 48)
	out[10] = byte(nonce >> 40)
	out[11] = byte(nonce >> 32)
	out[12] = byte(nonce >> 24)
	out[13] = byte(nonce >> 16)
	out[14] = byte(nonce >> 8)
	out[15] = byte(nonce)
	return out, nil
}

// DecodeBasicData unpacks a BASIC_DATA value.
func DecodeBasicData(v []byte) (version byte, codeSize uint32, nonce uint64, balance *uint256.Int) {
	var padded [32]byte
	copy(padded[:], v)
	version = padded[0]
	codeSize = uint32(padded[4])<<24 | uint32(padded[5])<<16 | uint32(padded[6])<<8 | uint32(padded[7])
	for i := 8; i < 16; i++ {
		nonce = nonce<<8 | uint64(padded[i])
	}
	balance = new(uint256.Int).SetBytes(padded[16:32])
	return version, codeSize, nonce, balance
}
