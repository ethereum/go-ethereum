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

// A QuarkChain shard block is a post-Merge block. Its header carries the
// QuarkChain fields without changing the header format:
//
//   - Difficulty and Nonce stay zero, as the Merge requires.
//   - MixDigest holds the PoW difficulty. It replaces Difficulty in the seal hash.
//   - Extra holds the other fields, packed with the fixed layout below. The PoW
//     seal (nonce and mix hash) is written into it after mining.
//
//	offset  size  field
//	0       4     version
//	4       4     shardId
//	8       32    prevRootBlock
//	40      24    cursor (rootBlockHeight, minorBlockIndex, xShardDepositIndex)
//	64      8     xShardGasUsed
//	72      8     xShardGasLimit
//	80      8     nonce
//	88      32    mixHash
//
// Integers are big-endian.

// QKCExtraSize is the size of the extra data of a QuarkChain shard block.
const QKCExtraSize = 120

// XShardCursor is a position in the stream of cross-shard messages confirmed by
// the root chain, as in QuarkChain's XshardTxCursorInfo. The cursor of a shard
// block is where the next block resumes applying messages.
type XShardCursor struct {
	RootBlockHeight    uint64 // height of the root block that confirms the source block
	MinorBlockIndex    uint64 // position of the source block in that root block
	XShardDepositIndex uint64 // position of the message in the source block
}

// QKCExtra holds the QuarkChain fields packed into the extra data of a shard
// block header.
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
// block header.
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
	copy(e.Nonce[:], extra[80:88])
	copy(e.MixHash[:], extra[88:120])
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

// QKCDifficulty returns the PoW difficulty of a QuarkChain shard block, which
// is stored in MixDigest.
func (h *Header) QKCDifficulty() *big.Int {
	return new(big.Int).SetBytes(h.MixDigest[:])
}
