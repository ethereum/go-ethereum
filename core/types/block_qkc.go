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

package types

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// QKCExtraSize is the size of the QuarkChain extra data carried in the
// Header.Extra of a shard block. The layout is a fixed protocol format.
//
// A QuarkChain shard block is a post-Merge geth block: the header format is
// left unchanged and the QuarkChain fields are packed into Header.Extra. Three
// geth header fields then take on QuarkChain semantics:
//
//   - Difficulty stays zero, as the Merge requires. A zero Difficulty is what
//     makes the EVM treat the block as post-Merge and read the block randomness
//     (PREVRANDAO) from MixDigest (see core/evm.go), so PoW cannot reuse
//     Difficulty.
//   - Nonce stays zero, as the Merge requires. The real PoW nonce lives in the
//     extra data below.
//   - MixDigest carries the PoW difficulty, which stands in for Difficulty in
//     the seal hash. MixDigest is not zero, so it is also the value the EVM sees
//     as the block randomness.
//
// The PoW seal (nonce and mix hash) occupies the tail of the extra data and is
// excluded from the seal hash, while everything before it is covered. See
// Header.QKCSealHash.
//
// Note that 120 bytes exceeds the 32-byte cap stock geth puts on Header.Extra
// (params.MaximumExtraDataSize, enforced on the block import path and on the
// Engine API path), so this layout cannot reach a chain until that cap is
// raised.
//
// The extra data is laid out as:
//
//	offset  size  field
//	0       4     version
//	4       4     shardId
//	8       32    prevRootBlock
//	40      24    cursor (rootBlockHeight, minorBlockIndex, xShardDepositIndex)
//	64      8     xShardGasUsed
//	72      8     xShardGasLimit
//	80      8     nonce       <- start of the PoW seal
//	88      32    mixHash
//
// Integers are big-endian.
const QKCExtraSize = 120

// Offsets of the PoW seal inside the extra data: the nonce followed by the mix
// hash, together the tail region that Header.QKCSealHash zeroes. The seal is
// covered by the block hash but not by the seal hash.
const (
	qkcExtraNonceOffset   = 80 // nonce, 8 bytes
	qkcExtraMixHashOffset = 88 // mix hash, 32 bytes
)

// XShardCursor is a position in the stream of cross-shard messages confirmed by
// the root chain, as in QuarkChain's XshardTxCursorInfo. The cursor of a shard
// block is where the next block resumes applying messages.
type XShardCursor struct {
	RootBlockHeight    uint64 // height of the root block that confirms the source block
	MinorBlockIndex    uint64 // position of the source block in that root block
	XShardDepositIndex uint64 // position of the message in the source block
}

// QKCExtra holds the QuarkChain fields packed into the extra data of a shard
// block header. See QKCExtraSize for the byte layout.
//
// Nonce and MixHash are the real PoW seal of the shard block. They are not the
// geth Header fields of the same name: Header.Nonce is always zero, and
// Header.MixDigest carries the QuarkChain difficulty rather than a mix hash.
type QKCExtra struct {
	Version        uint32
	ShardID        uint32
	PrevRootBlock  common.Hash // hash of the latest root block seen by the shard block
	Cursor         XShardCursor
	XShardGasUsed  uint64      // gas used by the cross-shard messages applied in the block
	XShardGasLimit uint64      // gas budget for the cross-shard messages applied in the block
	Nonce          BlockNonce  // PoW nonce
	MixHash        common.Hash // PoW mix hash
}

// DecodeQKCExtra decodes the QuarkChain fields from the extra data of a shard
// block header. It fails unless the extra data is exactly QKCExtraSize bytes.
func DecodeQKCExtra(extra []byte) (*QKCExtra, error) {
	if len(extra) != QKCExtraSize {
		return nil, fmt.Errorf("invalid qkc extra data length: have %d, want %d", len(extra), QKCExtraSize)
	}
	e := &QKCExtra{
		Version: binary.BigEndian.Uint32(extra[0:4]),
		ShardID: binary.BigEndian.Uint32(extra[4:8]),
		Cursor: XShardCursor{
			RootBlockHeight:    binary.BigEndian.Uint64(extra[40:48]),
			MinorBlockIndex:    binary.BigEndian.Uint64(extra[48:56]),
			XShardDepositIndex: binary.BigEndian.Uint64(extra[56:64]),
		},
		XShardGasUsed:  binary.BigEndian.Uint64(extra[64:72]),
		XShardGasLimit: binary.BigEndian.Uint64(extra[72:80]),
	}
	copy(e.PrevRootBlock[:], extra[8:40])
	copy(e.Nonce[:], extra[qkcExtraNonceOffset:qkcExtraMixHashOffset])
	copy(e.MixHash[:], extra[qkcExtraMixHashOffset:QKCExtraSize])
	return e, nil
}

// Encode packs the QuarkChain fields into the extra data of a shard block
// header.
func (e *QKCExtra) Encode() []byte {
	extra := make([]byte, 0, QKCExtraSize)
	extra = binary.BigEndian.AppendUint32(extra, e.Version)
	extra = binary.BigEndian.AppendUint32(extra, e.ShardID)
	extra = append(extra, e.PrevRootBlock[:]...)
	extra = binary.BigEndian.AppendUint64(extra, e.Cursor.RootBlockHeight)
	extra = binary.BigEndian.AppendUint64(extra, e.Cursor.MinorBlockIndex)
	extra = binary.BigEndian.AppendUint64(extra, e.Cursor.XShardDepositIndex)
	extra = binary.BigEndian.AppendUint64(extra, e.XShardGasUsed)
	extra = binary.BigEndian.AppendUint64(extra, e.XShardGasLimit)
	extra = append(extra, e.Nonce[:]...)
	extra = append(extra, e.MixHash[:]...)
	return extra
}

// QKCExtra decodes the QuarkChain fields from the header's extra data.
func (h *Header) QKCExtra() (*QKCExtra, error) {
	return DecodeQKCExtra(h.Extra)
}

// QKCDifficulty returns the PoW difficulty of a QuarkChain shard block. It is
// stored in MixDigest, not in the header's own Difficulty, which stays zero.
func (h *Header) QKCDifficulty() *big.Int {
	return new(big.Int).SetBytes(h.MixDigest[:])
}

// QKCSealHash returns the hash a QuarkChain shard block is sealed against: the
// block hash with the PoW seal zeroed out.
//
// The seal the miner searches for is the nonce and the mix hash stored inside
// the extra data, so the seal region documented on QKCExtraSize is zeroed here
// and the seal is thereby excluded from the hash. Nothing else is zeroed:
// Header.MixDigest keeps carrying the PoW difficulty, which therefore still
// takes part in the seal hash in place of Difficulty, while Header.Nonce and
// Header.Difficulty are hashed as they stand. A shard block must already carry
// those two at zero, as the Merge requires; this function neither checks nor
// forces that.
//
// The header is not modified: the zeroing happens on a copy. A header whose
// Extra is not exactly QKCExtraSize is not a QuarkChain shard block and is
// hashed as it stands. The length is the only discriminator, so an Extra of
// exactly QKCExtraSize bytes is taken to be a QuarkChain extra; rejecting a
// header that is not one at all is the caller's job (DecodeQKCExtra, and the
// consensus engine for headers arriving from the network).
func (h *Header) QKCSealHash() common.Hash {
	header := CopyHeader(h)
	if len(header.Extra) == QKCExtraSize {
		for i := qkcExtraNonceOffset; i < QKCExtraSize; i++ {
			header.Extra[i] = 0
		}
	}
	return header.Hash()
}
