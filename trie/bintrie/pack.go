// Copyright 2026 go-ethereum Authors
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
	"encoding/binary"

	"github.com/holiman/uint256"
)

// PackBasicData encodes an account's basic metadata (code size, nonce,
// balance) into the 32-byte BasicData leaf value defined by EIP-7864.
//
// The canonical spec layout is:
//
//	byte 0        version (currently always 0, left as the implicit zero)
//	bytes 1..4    reserved
//	bytes 5..7    code_size (big-endian, 3 bytes, max 2^24-1)
//	bytes 8..15   nonce (big-endian, 8 bytes)
//	bytes 16..31  balance (big-endian, right-justified, 16 bytes)
//
// For historical reasons the existing BinaryTrie implementation writes
// code_size as a 4-byte big-endian uint32 starting at byte 4 rather than a
// 3-byte big-endian field starting at byte 5. Byte 4 is reserved per the
// EIP, so for any realistic code size (below 2^24 ≈ 16 MB, well under the
// EIP-170 24 KB contract limit) the high byte is always 0 and the two
// encodings are bit-equivalent. This function preserves that existing
// behavior byte-for-byte so callers can substitute it for the inlined
// encoding in BinaryTrie.UpdateAccount without changing any state root.
//
// Any future correction of the byte offset is a consensus-level change
// and must be coordinated across clients.
func PackBasicData(nonce uint64, balance *uint256.Int, codeSize int) [HashSize]byte {
	var data [HashSize]byte
	binary.BigEndian.PutUint32(data[BasicDataCodeSizeOffset-1:], uint32(codeSize))
	binary.BigEndian.PutUint64(data[BasicDataNonceOffset:], nonce)

	// Balance is a 256-bit uint stored right-justified in the lower 16
	// bytes of BasicData. For dev-mode accounts whose balance exceeds
	// 2^128 - 1 (e.g. 0xff × HashSize), truncate to the upper 16 bytes to
	// match the existing BinaryTrie behavior rather than panicking.
	balanceBytes := balance.Bytes()
	if len(balanceBytes) > 16 {
		balanceBytes = balanceBytes[16:]
	}
	copy(data[HashSize-len(balanceBytes):], balanceBytes[:])
	return data
}

// UnpackBasicData is the inverse of PackBasicData. It decodes the code
// size, nonce, and balance fields from a BasicData leaf value.
//
// Note: the returned balance is always 128-bit or smaller because the
// encoding reserves 16 bytes for it; dev-mode accounts whose pre-encoded
// balance exceeded 2^128 - 1 are not recoverable losslessly.
func UnpackBasicData(data [HashSize]byte) (nonce uint64, balance *uint256.Int, codeSize int) {
	codeSize = int(binary.BigEndian.Uint32(data[BasicDataCodeSizeOffset-1:]))
	nonce = binary.BigEndian.Uint64(data[BasicDataNonceOffset:])

	var b [16]byte
	copy(b[:], data[BasicDataBalanceOffset:])
	balance = new(uint256.Int).SetBytes(b[:])
	return
}
