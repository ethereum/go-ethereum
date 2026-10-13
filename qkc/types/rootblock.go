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

// Package types contains the data types of the QuarkChain root chain.
package types

import (
	"bytes"
	"crypto/ecdsa"
	"io"
	"math/big"
	"reflect"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	ctypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// RootBlockHeader represents a root block header in the QuarkChain.
type RootBlockHeader struct {
	Version         uint32         `json:"version"          gencodec:"required"`
	Number          uint32         `json:"number"           gencodec:"required"`
	ParentHash      common.Hash    `json:"parentHash"       gencodec:"required"`
	MinorHeaderHash common.Hash    `json:"headerHash"       gencodec:"required"`
	Coinbase        common.Address `json:"miner"            gencodec:"required"`
	CoinbaseAmount  *big.Int       `json:"coinbaseAmount"   gencodec:"required"`
	Time            uint64         `json:"timestamp"        gencodec:"required"`
	Difficulty      *big.Int       `json:"difficulty"       gencodec:"required"`
	TotalDifficulty *big.Int       `json:"total_difficulty" gencodec:"required"`
	Nonce           uint64         `json:"nonce"`
	Extra           []byte         `json:"extraData"        gencodec:"required"`
	MixDigest       common.Hash    `json:"mixHash"`
	Signature       [65]byte       `json:"signature"        gencodec:"required"`
}

// Hash returns the block hash of the header, which is the keccak256 hash of its
// RLP encoding.
func (h *RootBlockHeader) Hash() common.Hash {
	return rlpHash(h)
}

// SealHash returns the hash of the header without the seal (nonce and mix
// digest) and the signature. It is the hash mined and signed.
func (h *RootBlockHeader) SealHash() common.Hash {
	return rlpHash([]any{
		h.Version,
		h.Number,
		h.ParentHash,
		h.MinorHeaderHash,
		h.Coinbase,
		h.CoinbaseAmount,
		h.Time,
		h.Difficulty,
		h.TotalDifficulty,
		h.Extra,
	})
}

var rootBlockHeaderSize = common.StorageSize(reflect.TypeFor[RootBlockHeader]().Size())

// Size returns the approximate memory used by all internal contents. It is used
// to approximate and limit the memory consumption of various caches.
func (h *RootBlockHeader) Size() common.StorageSize {
	return rootBlockHeaderSize + common.StorageSize(len(h.Extra)+h.Difficulty.BitLen()/8)
}

// NumberU64 returns the block number as a uint64.
func (h *RootBlockHeader) NumberU64() uint64 { return uint64(h.Number) }

// Sign signs the seal hash of the header with the given key, and stores the
// signature in the header.
func (h *RootBlockHeader) Sign(prv *ecdsa.PrivateKey) error {
	sig, err := crypto.Sign(h.SealHash().Bytes(), prv)
	if err != nil {
		return err
	}
	copy(h.Signature[:], sig)
	return nil
}

// VerifySignature reports whether the header is signed by the given key.
func (h *RootBlockHeader) VerifySignature(key *ecdsa.PublicKey) bool {
	return crypto.VerifySignature(crypto.CompressPubkey(key), h.SealHash().Bytes(), h.Signature[:64])
}

// CreateBlockToAppend creates an empty block on top of the header. Nil
// arguments take default values: the parent time plus one, the parent
// difficulty, the zero address, a zero nonce and empty extra data.
func (h *RootBlockHeader) CreateBlockToAppend(createTime *uint64, difficulty *big.Int, address *common.Address, nonce *uint64, extraData []byte) *RootBlock {
	if createTime == nil {
		preTime := h.Time + 1
		createTime = &preTime
	}
	if difficulty == nil {
		difficulty = h.Difficulty
	}
	if address == nil {
		address = new(common.Address)
	}
	if nonce == nil {
		nonce = new(uint64)
	}
	if extraData == nil {
		extraData = []byte{}
	}
	header := &RootBlockHeader{
		Version:         h.Version,
		Number:          h.Number + 1,
		ParentHash:      h.Hash(),
		Coinbase:        *address,
		CoinbaseAmount:  new(big.Int),
		Time:            *createTime,
		Difficulty:      new(big.Int).Set(difficulty),
		TotalDifficulty: new(big.Int).Add(h.TotalDifficulty, difficulty),
		Nonce:           *nonce,
		Extra:           common.CopyBytes(extraData),
	}
	return &RootBlock{
		header:            header,
		minorBlockHeaders: MinorBlockHeaders{},
		trackingData:      []byte{},
	}
}

// CopyRootBlockHeader creates a deep copy of a root block header.
func CopyRootBlockHeader(h *RootBlockHeader) *RootBlockHeader {
	cpy := *h
	if cpy.CoinbaseAmount = new(big.Int); h.CoinbaseAmount != nil {
		cpy.CoinbaseAmount.Set(h.CoinbaseAmount)
	}
	if cpy.Difficulty = new(big.Int); h.Difficulty != nil {
		cpy.Difficulty.Set(h.Difficulty)
	}
	if cpy.TotalDifficulty = new(big.Int); h.TotalDifficulty != nil {
		cpy.TotalDifficulty.Set(h.TotalDifficulty)
	}
	if len(h.Extra) > 0 {
		cpy.Extra = common.CopyBytes(h.Extra)
	}
	return &cpy
}

// MinorBlockHeaders is a list of shard block headers confirmed by a root block.
type MinorBlockHeaders []*ctypes.Header

// Len returns the length of s.
func (s MinorBlockHeaders) Len() int { return len(s) }

// EncodeIndex encodes the i'th header to w.
func (s MinorBlockHeaders) EncodeIndex(i int, w *bytes.Buffer) {
	rlp.Encode(w, s[i])
}

// CalcMinorHeaderHash returns the trie root of the given shard block headers.
func CalcMinorHeaderHash(headers MinorBlockHeaders) common.Hash {
	return ctypes.DeriveSha(headers, trie.NewStackTrie(nil))
}

// RootBlock represents an entire root block in the QuarkChain.
type RootBlock struct {
	header            *RootBlockHeader
	minorBlockHeaders MinorBlockHeaders
	trackingData      []byte

	// caches
	hash atomic.Pointer[common.Hash]
	size atomic.Uint64
}

// "external" root block encoding. used for p2p, etc.
type extRootBlock struct {
	Header            *RootBlockHeader
	MinorBlockHeaders MinorBlockHeaders
	TrackingData      []byte
}

// NewRootBlock creates a new root block. The input data is copied, changes to
// header and to the field values will not affect the block.
//
// The MinorHeaderHash in header is ignored and set to the value derived from
// the given shard block headers.
func NewRootBlock(header *RootBlockHeader, minorBlockHeaders MinorBlockHeaders, trackingData []byte) *RootBlock {
	b := NewRootBlockWithHeader(header)
	b.header.MinorHeaderHash = CalcMinorHeaderHash(minorBlockHeaders)
	b.minorBlockHeaders = make(MinorBlockHeaders, len(minorBlockHeaders))
	for i := range minorBlockHeaders {
		b.minorBlockHeaders[i] = ctypes.CopyHeader(minorBlockHeaders[i])
	}
	b.trackingData = common.CopyBytes(trackingData)
	return b
}

// NewRootBlockWithHeader creates a root block with the given header data. The
// header data is copied, changes to header and to the field values will not
// affect the block.
func NewRootBlockWithHeader(header *RootBlockHeader) *RootBlock {
	return &RootBlock{header: CopyRootBlockHeader(header)}
}

// DecodeRLP decodes a root block from RLP.
func (b *RootBlock) DecodeRLP(s *rlp.Stream) error {
	var eb extRootBlock
	_, size, _ := s.Kind()
	if err := s.Decode(&eb); err != nil {
		return err
	}
	b.header, b.minorBlockHeaders, b.trackingData = eb.Header, eb.MinorBlockHeaders, eb.TrackingData
	b.size.Store(rlp.ListSize(size))
	return nil
}

// EncodeRLP serializes a root block as RLP.
func (b *RootBlock) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &extRootBlock{
		Header:            b.header,
		MinorBlockHeaders: b.minorBlockHeaders,
		TrackingData:      b.trackingData,
	})
}

// MinorBlockHeaders returns the shard block headers confirmed by the block.
// Note the returned data is not an independent copy.
func (b *RootBlock) MinorBlockHeaders() MinorBlockHeaders { return b.minorBlockHeaders }

// MinorBlockHeader returns the shard block header with the given hash, or nil
// if the block does not confirm it.
func (b *RootBlock) MinorBlockHeader(hash common.Hash) *ctypes.Header {
	for _, header := range b.minorBlockHeaders {
		if header.Hash() == hash {
			return header
		}
	}
	return nil
}

func (b *RootBlock) TrackingData() []byte { return b.trackingData }

func (b *RootBlock) Version() uint32              { return b.header.Version }
func (b *RootBlock) Number() uint32               { return b.header.Number }
func (b *RootBlock) NumberU64() uint64            { return uint64(b.header.Number) }
func (b *RootBlock) ParentHash() common.Hash      { return b.header.ParentHash }
func (b *RootBlock) MinorHeaderHash() common.Hash { return b.header.MinorHeaderHash }
func (b *RootBlock) Coinbase() common.Address     { return b.header.Coinbase }
func (b *RootBlock) CoinbaseAmount() *big.Int     { return new(big.Int).Set(b.header.CoinbaseAmount) }
func (b *RootBlock) Time() uint64                 { return b.header.Time }
func (b *RootBlock) Difficulty() *big.Int         { return new(big.Int).Set(b.header.Difficulty) }
func (b *RootBlock) TotalDifficulty() *big.Int    { return new(big.Int).Set(b.header.TotalDifficulty) }
func (b *RootBlock) Nonce() uint64                { return b.header.Nonce }
func (b *RootBlock) Extra() []byte                { return common.CopyBytes(b.header.Extra) }
func (b *RootBlock) MixDigest() common.Hash       { return b.header.MixDigest }
func (b *RootBlock) Signature() [65]byte          { return b.header.Signature }

// Header returns a deep-copy of the entire root block header.
func (b *RootBlock) Header() *RootBlockHeader { return CopyRootBlockHeader(b.header) }

// Size returns the true RLP encoded storage size of the block, either by
// encoding and returning it, or returning a previously cached value.
func (b *RootBlock) Size() uint64 {
	if size := b.size.Load(); size > 0 {
		return size
	}
	enc, _ := rlp.EncodeToBytes(b)
	b.size.Store(uint64(len(enc)))
	return uint64(len(enc))
}

// WithMiningResult returns a new block with the data from b, and the header
// sealed with the given nonce and mix digest. The signature is kept if nil.
func (b *RootBlock) WithMiningResult(nonce uint64, mixDigest common.Hash, signature *[65]byte) *RootBlock {
	cpy := CopyRootBlockHeader(b.header)
	cpy.Nonce = nonce
	cpy.MixDigest = mixDigest
	if signature != nil {
		cpy.Signature = *signature
	}
	return b.WithSeal(cpy)
}

// WithSeal returns a new block with the data from b but the header replaced with
// the sealed one.
func (b *RootBlock) WithSeal(header *RootBlockHeader) *RootBlock {
	return &RootBlock{
		header:            CopyRootBlockHeader(header),
		minorBlockHeaders: b.minorBlockHeaders,
		trackingData:      b.trackingData,
	}
}

// WithBody returns a new block with the original header and a deep copy of the
// given shard block headers and tracking data.
func (b *RootBlock) WithBody(minorBlockHeaders MinorBlockHeaders, trackingData []byte) *RootBlock {
	block := &RootBlock{
		header:            CopyRootBlockHeader(b.header),
		minorBlockHeaders: make(MinorBlockHeaders, len(minorBlockHeaders)),
		trackingData:      common.CopyBytes(trackingData),
	}
	for i := range minorBlockHeaders {
		block.minorBlockHeaders[i] = ctypes.CopyHeader(minorBlockHeaders[i])
	}
	return block
}

// Hash returns the keccak256 hash of b's header.
// The hash is computed on the first call and cached thereafter.
func (b *RootBlock) Hash() common.Hash {
	if hash := b.hash.Load(); hash != nil {
		return *hash
	}
	h := b.header.Hash()
	b.hash.Store(&h)
	return h
}

// Finalize sets the minor header hash and the coinbase of the block in place.
// Nil arguments take default values: zero amount and the zero address.
func (b *RootBlock) Finalize(coinbaseAmount *big.Int, coinbaseAddress *common.Address) *RootBlock {
	if coinbaseAmount == nil {
		coinbaseAmount = new(big.Int)
	}
	if coinbaseAddress == nil {
		coinbaseAddress = new(common.Address)
	}
	b.header.MinorHeaderHash = CalcMinorHeaderHash(b.minorBlockHeaders)
	b.header.CoinbaseAmount = new(big.Int).Set(coinbaseAmount)
	b.header.Coinbase = *coinbaseAddress
	h := b.header.Hash()
	b.hash.Store(&h)
	b.size.Store(0)
	return b
}

// AddMinorBlockHeader appends a shard block header to the block in place. The
// minor header hash is updated by Finalize.
func (b *RootBlock) AddMinorBlockHeader(header *ctypes.Header) {
	b.minorBlockHeaders = append(b.minorBlockHeaders, header)
	b.size.Store(0)
}

// ExtendMinorBlockHeaderList appends the shard block headers created no later
// than createTime to the block in place. The minor header hash is updated by
// Finalize.
func (b *RootBlock) ExtendMinorBlockHeaderList(headers []*ctypes.Header, createTime uint64) {
	for _, header := range headers {
		if header.Time <= createTime {
			b.minorBlockHeaders = append(b.minorBlockHeaders, header)
		}
	}
	b.size.Store(0)
}

// rlpHash encodes x and hashes the encoded bytes.
func rlpHash(x any) (h common.Hash) {
	sha := crypto.NewKeccakState()
	rlp.Encode(sha, x)
	sha.Read(h[:])
	return h
}
