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
// left unchanged and the QuarkChain fields are packed into Header.Extra. Two
// header fields are repurposed, following the 2.0 design:
//
//   - Difficulty stays zero, as the Merge requires. A zero Difficulty is what
//     makes the EVM treat the block as post-Merge and read the block randomness
//     (PREVRANDAO) from MixDigest (see core/evm.go), so PoW cannot reuse
//     Difficulty; the PoW difficulty is kept in MixDigest instead and stands in
//     for Difficulty in the seal hash. Overloading MixDigest therefore also
//     changes the value the EVM sees as randomness, which the consensus engine
//     work has to settle.
//   - Nonce stays zero, as the Merge requires. The real PoW nonce lives in the
//     extra data below.
//
// The PoW seal (nonce and mix hash) occupies the tail of the extra data and is
// excluded from the seal hash, while everything before it is covered. See
// Header.QKCSealHash.
//
// The size is a fixed part of the format, but it does not fit stock geth yet:
// params.MaximumExtraDataSize caps Header.Extra at 32 bytes and is enforced on
// both the block import path and the Engine API path
// (beacon/engine/types.go, consensus/beacon/consensus.go). Raising that cap is
// a qkc-geth patch, not something this file can do.
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

// Offsets of the PoW seal inside the extra data, i.e. the tail region that
// Header.QKCSealHash zeroes. The seal is covered by the block hash but not by
// the seal hash.
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

// QKCDifficulty returns the PoW difficulty of a QuarkChain shard block, which
// is stored in MixDigest.
func (h *Header) QKCDifficulty() *big.Int {
	return new(big.Int).SetBytes(h.MixDigest[:])
}

// SetQKCDifficulty sets the PoW difficulty of a QuarkChain shard block, storing
// it in MixDigest. The header's own Difficulty is set to zero, as the Merge
// requires. A nil difficulty means no work, i.e. a zero difficulty.
//
// The difficulty must fit in 256 bits and be non-negative. The narrowing is
// silent: common.BigToHash drops the leading bytes of a wider value and takes
// the absolute value of a negative one, so neither can be caught here. Callers
// set it from the shard difficulty rule, which produces neither.
func (h *Header) SetQKCDifficulty(difficulty *big.Int) {
	h.Difficulty = new(big.Int)
	h.MixDigest = common.Hash{}
	if difficulty != nil {
		h.MixDigest = common.BigToHash(difficulty)
	}
}

// SetQKCSeal writes the PoW seal found by the miner (the nonce and the mix
// hash) into the extra data of a QuarkChain shard block. The header's own Nonce
// and Difficulty are left untouched, as they must stay zero after the Merge.
//
// It fails if the header does not already carry a QuarkChain extra.
func (h *Header) SetQKCSeal(nonce BlockNonce, mixHash common.Hash) error {
	extra, err := h.QKCExtra()
	if err != nil {
		return err
	}
	extra.Nonce = nonce
	extra.MixHash = mixHash
	h.Extra = extra.Encode()
	return nil
}

// QKCSealHash returns the hash a QuarkChain shard block is sealed against: the
// block hash with the PoW seal zeroed out.
//
// A shard block is a post-Merge block, so Header.Nonce is already zero and the
// seal found by the miner (nonce and mix hash) lives in the tail of the extra
// data; that region is zeroed here. Header.MixDigest is left untouched, so the
// PoW difficulty keeps taking part in the seal hash in place of Difficulty.
// This mirrors QuarkChain's legacy rule that the seal hash excludes exactly the
// seal fields and commits to the difficulty.
//
// The header is not modified. A header whose Extra is not exactly QKCExtraSize
// is not a QuarkChain shard block and is hashed unchanged; rejecting such a
// header is the caller's job (DecodeQKCExtra, and the consensus engine for
// headers arriving from the network).
func (h *Header) QKCSealHash() common.Hash {
	header := CopyHeader(h)
	if len(header.Extra) == QKCExtraSize {
		for i := qkcExtraNonceOffset; i < QKCExtraSize; i++ {
			header.Extra[i] = 0
		}
	}
	return header.Hash()
}
