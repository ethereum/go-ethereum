// Copyright 2026-2027 The QuarkChain Authors
// This file is part of the quarkchain library.
//
// The QuarkChain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The QuarkChain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the QuarkChain library. If not, see <http://www.gnu.org/licenses/>.

// Package types contains the data types of the QuarkChain root chain.
package types

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"reflect"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	ctypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	ssz "github.com/ferranbt/fastssz"
	"github.com/holiman/uint256"
)

// RootBlockHeader represents a root block header in the QuarkChain.
type RootBlockHeader struct {
	Version         uint32         `json:"version"          gencodec:"required"`
	Number          uint32         `json:"number"           gencodec:"required"`
	ParentHash      common.Hash    `json:"parentHash"       gencodec:"required"`
	MinorHeaderHash common.Hash    `json:"minorHeaderHash"  gencodec:"required"`
	Coinbase        common.Address `json:"miner"            gencodec:"required"`
	CoinbaseAmount  uint256.Int    `json:"coinbaseAmount"   gencodec:"required"`
	Time            uint64         `json:"timestamp"        gencodec:"required"`
	Difficulty      uint256.Int    `json:"difficulty"       gencodec:"required"`
	TotalDifficulty uint256.Int    `json:"total_difficulty" gencodec:"required"`
	Extra           []byte         `json:"extraData"        gencodec:"required"`
	Nonce           uint64         `json:"nonce"`
	MixDigest       common.Hash    `json:"mixHash"`
	Signature       [65]byte       `json:"signature"        gencodec:"required"`
}

const (
	// unsealedFixedSize is the size of the fixed part of the SSZ encoding of
	// the header without the seal, the offset of Extra included.
	unsealedFixedSize = 4 + 4 + 32 + 32 + 20 + 32 + 8 + 32 + 32 + 4

	// sealedFixedSize is the size of the fixed part of the SSZ encoding of the
	// header with the seal.
	sealedFixedSize = unsealedFixedSize + 8 + 32 + 65
)

// Hash returns the block hash of the header, which is the keccak256 hash of its
// SSZ encoding.
func (h *RootBlockHeader) Hash() common.Hash {
	enc, _ := h.MarshalSSZ()
	return crypto.Keccak256Hash(enc)
}

// SealHash returns the hash of the header without the seal (nonce and mix
// digest) and the signature. It is the hash mined and signed.
func (h *RootBlockHeader) SealHash() common.Hash {
	enc := make([]byte, 0, unsealedFixedSize+len(h.Extra))
	return crypto.Keccak256Hash(h.appendSSZ(enc, false))
}

// appendSSZ appends the SSZ encoding of the header as the container below.
// Without the seal, Nonce, MixDigest and Signature are left out.
// Keep this encoder infallible: Hash and SealHash have no error return.
//
//	Version         uint32      0
//	Number          uint32      4
//	ParentHash      Bytes32     8
//	MinorHeaderHash Bytes32     40
//	Coinbase        Bytes20     72
//	CoinbaseAmount  uint256     92
//	Time            uint64      124
//	Difficulty      uint256     132
//	TotalDifficulty uint256     164
//	ExtraOffset     uint32      196  (offset of Extra, i.e. the fixed size)
//	Nonce           uint64      200  (seal)
//	MixDigest       Bytes32     208  (seal)
//	Signature       Bytes65     240  (seal)
//	Extra           List[byte]  305
func (h *RootBlockHeader) appendSSZ(enc []byte, withSeal bool) []byte {
	fixedSize := unsealedFixedSize
	if withSeal {
		fixedSize = sealedFixedSize
	}
	enc = ssz.MarshalUint32(enc, h.Version)
	enc = ssz.MarshalUint32(enc, h.Number)
	enc = append(enc, h.ParentHash[:]...)
	enc = append(enc, h.MinorHeaderHash[:]...)
	enc = append(enc, h.Coinbase[:]...)
	enc, _ = h.CoinbaseAmount.MarshalSSZAppend(enc)
	enc = ssz.MarshalUint64(enc, h.Time)
	enc, _ = h.Difficulty.MarshalSSZAppend(enc)
	enc, _ = h.TotalDifficulty.MarshalSSZAppend(enc)
	enc = ssz.WriteOffset(enc, fixedSize)
	if withSeal {
		enc = ssz.MarshalUint64(enc, h.Nonce)
		enc = append(enc, h.MixDigest[:]...)
		enc = append(enc, h.Signature[:]...)
	}
	enc = append(enc, h.Extra...)
	return enc
}

// SizeSSZ returns the size of the SSZ encoding of the header.
func (h *RootBlockHeader) SizeSSZ() int {
	return sealedFixedSize + len(h.Extra)
}

// MarshalSSZTo appends the SSZ encoding of the header to dst.
func (h *RootBlockHeader) MarshalSSZTo(dst []byte) ([]byte, error) {
	return h.appendSSZ(dst, true), nil
}

// MarshalSSZ returns the SSZ encoding of the header.
func (h *RootBlockHeader) MarshalSSZ() ([]byte, error) {
	return h.MarshalSSZTo(make([]byte, 0, h.SizeSSZ()))
}

// UnmarshalSSZ decodes the header from its SSZ encoding.
func (h *RootBlockHeader) UnmarshalSSZ(buf []byte) error {
	if len(buf) < sealedFixedSize {
		return ssz.ErrSize
	}
	if ssz.ReadOffset(buf[196:200]) != sealedFixedSize {
		return ssz.ErrOffset
	}
	// Decode into a temporary value so a future field decoder error cannot
	// leave the receiver partially updated.
	var decoded RootBlockHeader
	decoded.Version = ssz.UnmarshallUint32(buf[0:4])
	decoded.Number = ssz.UnmarshallUint32(buf[4:8])
	copy(decoded.ParentHash[:], buf[8:40])
	copy(decoded.MinorHeaderHash[:], buf[40:72])
	copy(decoded.Coinbase[:], buf[72:92])
	if err := decoded.CoinbaseAmount.UnmarshalSSZ(buf[92:124]); err != nil {
		return err
	}
	decoded.Time = ssz.UnmarshallUint64(buf[124:132])
	if err := decoded.Difficulty.UnmarshalSSZ(buf[132:164]); err != nil {
		return err
	}
	if err := decoded.TotalDifficulty.UnmarshalSSZ(buf[164:196]); err != nil {
		return err
	}
	decoded.Nonce = ssz.UnmarshallUint64(buf[200:208])
	copy(decoded.MixDigest[:], buf[208:240])
	copy(decoded.Signature[:], buf[240:305])
	decoded.Extra = common.CopyBytes(buf[sealedFixedSize:])
	*h = decoded
	return nil
}

var rootBlockHeaderSize = common.StorageSize(reflect.TypeFor[RootBlockHeader]().Size())

// Size returns the approximate memory used by all internal contents. It is used
// to approximate and limit the memory consumption of various caches.
func (h *RootBlockHeader) Size() common.StorageSize {
	return rootBlockHeaderSize + common.StorageSize(len(h.Extra))
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
func (h *RootBlockHeader) CreateBlockToAppend(createTime *uint64, difficulty *uint256.Int, address *common.Address, nonce *uint64, extraData []byte) *RootBlock {
	if createTime == nil {
		preTime := h.Time + 1
		createTime = &preTime
	}
	if difficulty == nil {
		difficulty = &h.Difficulty
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
		Version:    h.Version,
		Number:     h.Number + 1,
		ParentHash: h.Hash(),
		Coinbase:   *address,
		Time:       *createTime,
		Difficulty: *difficulty,
		Nonce:      *nonce,
		Extra:      common.CopyBytes(extraData),
	}
	// The total difficulty cannot overflow in practice. Block validation must
	// still reject an overflow, as Add wraps around.
	header.TotalDifficulty.Add(&h.TotalDifficulty, difficulty)
	return &RootBlock{
		header:            header,
		minorBlockHeaders: MinorBlockHeaders{},
		trackingData:      []byte{},
	}
}

// CopyRootBlockHeader creates a deep copy of a root block header.
func CopyRootBlockHeader(h *RootBlockHeader) *RootBlockHeader {
	cpy := *h
	if len(h.Extra) > 0 {
		cpy.Extra = common.CopyBytes(h.Extra)
	}
	return &cpy
}

// MinorBlockHeaders is a list of shard block headers confirmed by a root block.
type MinorBlockHeaders []*ctypes.Header

// CalcMinorHeaderHash returns the merkle root of the given shard block headers.
// The leaves are the keccak256 hashes of the headers' RLP encodings (i.e. their
// block hashes); a list of odd length is padded with a zero hash that doubles at
// every level, and the final root is hashed together with the list length
// appended as an 8-byte big-endian integer, matching pyquarkchain's
// calculate_merkle_root. It panics if the list contains a nil header.
func CalcMinorHeaderHash(headers MinorBlockHeaders) common.Hash {
	leaves := make([]common.Hash, len(headers))
	for i, header := range headers {
		if header == nil {
			panic(fmt.Sprintf("nil shard block header at index %d", i))
		}
		leaves[i] = header.Hash()
	}
	return sszMerkleRoot(leaves)
}

// sszMerkleRoot merkleizes the leaves the way pyquarkchain's
// calculate_merkle_root does: a list of odd length is padded with a zero hash
// that doubles at every level, and the root is hashed together with the leaf
// count appended as an 8-byte big-endian integer.
//
// The length encoding deliberately differs from the SSZ mix-in rule, which uses
// a 32-byte little-endian uint256: the 8-byte big-endian length is kept for
// compatibility with pyquarkchain, so do not "fix" it to match SSZ.
func sszMerkleRoot(leaves []common.Hash) common.Hash {
	tree := make([]common.Hash, len(leaves))
	copy(tree, leaves)
	if len(tree) == 0 {
		tree = append(tree, common.Hash{})
	}
	var zero common.Hash
	for len(tree) > 1 {
		if len(tree)%2 != 0 {
			tree = append(tree, zero)
		}
		for i := range tree[:len(tree)/2] {
			tree[i] = crypto.Keccak256Hash(tree[2*i][:], tree[2*i+1][:])
		}
		tree = tree[:len(tree)/2]
		zero = crypto.Keccak256Hash(zero[:], zero[:])
	}
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(leaves)))
	return crypto.Keccak256Hash(tree[0][:], length[:])
}

// RootBlock represents an entire root block in the QuarkChain.
//
// The block owns its data: constructors and mutators copy their inputs, so
// later changes to the arguments never affect the block. The block header and
// the shard block headers must be non-nil; a nil header is a programming error
// and fails fast.
// MinorBlockHeaders and TrackingData expose aliases to internal storage; callers
// must treat the returned data and pointed-to headers as read-only. Mutators
// may only be called while the block is exclusively owned. Once published to
// other goroutines, the block must be treated as immutable. A block returned by
// WithBody has not had MinorHeaderHash recomputed; call Finalize before treating
// it as a complete block.
type RootBlock struct {
	header            *RootBlockHeader
	minorBlockHeaders MinorBlockHeaders
	trackingData      []byte

	// caches
	hash atomic.Pointer[common.Hash]
	size atomic.Uint64
}

// rootBlockFixedSize is the size of the fixed part of the SSZ encoding of a
// root block, i.e. the offsets of its three fields.
const rootBlockFixedSize = 3 * 4

func rootBlockSSZOffsets(buf []byte) (headerOffset, headersOffset, trackingOffset uint64, err error) {
	if len(buf) < rootBlockFixedSize {
		return 0, 0, 0, ssz.ErrSize
	}
	headerOffset = ssz.ReadOffset(buf[0:4])
	headersOffset = ssz.ReadOffset(buf[4:8])
	trackingOffset = ssz.ReadOffset(buf[8:12])
	if headerOffset != rootBlockFixedSize || headerOffset > headersOffset || headersOffset > trackingOffset || trackingOffset > uint64(len(buf)) {
		return 0, 0, 0, ssz.ErrOffset
	}
	return headerOffset, headersOffset, trackingOffset, nil
}

// NewRootBlock creates a new root block. The given header, shard block headers
// and tracking data are copied, so later changes to them do not affect the
// block.
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
// header is copied, so later changes to it do not affect the block. The header
// must be non-nil.
func NewRootBlockWithHeader(header *RootBlockHeader) *RootBlock {
	return &RootBlock{header: CopyRootBlockHeader(header)}
}

// MarshalSSZ returns the SSZ encoding of the block as the container below. The
// shard block headers keep their RLP encoding, which their hashes are computed
// over.
//
//	Header            RootBlockHeader
//	MinorBlockHeaders List[List[byte]]
//	TrackingData      List[byte]
func (b *RootBlock) MarshalSSZ() ([]byte, error) {
	headers := make([][]byte, len(b.minorBlockHeaders))
	headersSize := 0
	for i, header := range b.minorBlockHeaders {
		if header == nil {
			return nil, fmt.Errorf("nil shard block header at index %d", i)
		}
		enc, err := rlp.EncodeToBytes(header)
		if err != nil {
			return nil, err
		}
		headers[i] = enc
		headersSize += 4 + len(enc)
	}
	var (
		headerOffset   = rootBlockFixedSize
		headersOffset  = headerOffset + b.header.SizeSSZ()
		trackingOffset = headersOffset + headersSize
	)
	enc := make([]byte, 0, trackingOffset+len(b.trackingData))
	enc = ssz.WriteOffset(enc, headerOffset)
	enc = ssz.WriteOffset(enc, headersOffset)
	enc = ssz.WriteOffset(enc, trackingOffset)
	enc, _ = b.header.MarshalSSZTo(enc)
	offset := 4 * len(headers)
	for _, header := range headers {
		enc = ssz.WriteOffset(enc, offset)
		offset += len(header)
	}
	for _, header := range headers {
		enc = append(enc, header...)
	}
	enc = append(enc, b.trackingData...)
	return enc, nil
}

// UnmarshalSSZ decodes the block from its SSZ encoding.
func (b *RootBlock) UnmarshalSSZ(buf []byte) error {
	headerOffset, headersOffset, trackingOffset, err := rootBlockSSZOffsets(buf)
	if err != nil {
		return err
	}
	header := new(RootBlockHeader)
	if err := header.UnmarshalSSZ(buf[headerOffset:headersOffset]); err != nil {
		return err
	}
	headersBuf := buf[headersOffset:trackingOffset]
	n, err := ssz.DecodeDynamicLength(headersBuf, len(headersBuf)/4)
	if err != nil {
		return err
	}
	// fastssz truncates the uint32 offset to a negative int on 32-bit
	// platforms and returns it without error.
	if n < 0 {
		return ssz.ErrSize
	}
	// The canonical SSZ encoding of an empty dynamic list is empty. Some
	// fastssz versions accept a four-byte zero offset table for an empty list;
	// reject it so decoding followed by encoding preserves the byte length.
	if n == 0 && len(headersBuf) != 0 {
		return ssz.ErrOffset
	}
	headers := make(MinorBlockHeaders, n)
	err = ssz.UnmarshalDynamic(headersBuf, n, func(i int, enc []byte) error {
		headers[i] = new(ctypes.Header)
		return rlp.DecodeBytes(enc, headers[i])
	})
	if err != nil {
		return err
	}
	b.header, b.minorBlockHeaders, b.trackingData = header, headers, common.CopyBytes(buf[trackingOffset:])
	b.hash.Store(nil)
	// Accepted input is canonical, so its length is the canonical SSZ size.
	b.size.Store(uint64(len(buf)))
	return nil
}

// DecodeRootBlockHeaderFromBlockSSZ decodes the header from a root block SSZ
// encoding without decoding its shard block headers or tracking data.
func DecodeRootBlockHeaderFromBlockSSZ(buf []byte) (*RootBlockHeader, error) {
	headerOffset, headersOffset, _, err := rootBlockSSZOffsets(buf)
	if err != nil {
		return nil, err
	}
	header := new(RootBlockHeader)
	if err := header.UnmarshalSSZ(buf[headerOffset:headersOffset]); err != nil {
		return nil, err
	}
	return header, nil
}

// MinorBlockHeaders returns the shard block headers confirmed by the block.
// The returned slice and its header pointers alias internal data and must be
// treated as read-only; modifying them can invalidate block invariants.
func (b *RootBlock) MinorBlockHeaders() MinorBlockHeaders { return b.minorBlockHeaders }

// MinorBlockHeader returns the shard block header with the given hash, or nil
// if the block does not confirm it. The returned header aliases internal data
// and must be treated as read-only.
func (b *RootBlock) MinorBlockHeader(hash common.Hash) *ctypes.Header {
	for _, header := range b.minorBlockHeaders {
		if header.Hash() == hash {
			return header
		}
	}
	return nil
}

// TrackingData returns the tracking data of the block. The returned slice
// aliases internal data and must be treated as read-only.
func (b *RootBlock) TrackingData() []byte { return b.trackingData }

func (b *RootBlock) Version() uint32               { return b.header.Version }
func (b *RootBlock) Number() uint32                { return b.header.Number }
func (b *RootBlock) NumberU64() uint64             { return uint64(b.header.Number) }
func (b *RootBlock) ParentHash() common.Hash       { return b.header.ParentHash }
func (b *RootBlock) MinorHeaderHash() common.Hash  { return b.header.MinorHeaderHash }
func (b *RootBlock) Coinbase() common.Address      { return b.header.Coinbase }
func (b *RootBlock) CoinbaseAmount() *uint256.Int  { return b.header.CoinbaseAmount.Clone() }
func (b *RootBlock) Time() uint64                  { return b.header.Time }
func (b *RootBlock) Difficulty() *uint256.Int      { return b.header.Difficulty.Clone() }
func (b *RootBlock) TotalDifficulty() *uint256.Int { return b.header.TotalDifficulty.Clone() }
func (b *RootBlock) Nonce() uint64                 { return b.header.Nonce }
func (b *RootBlock) Extra() []byte                 { return common.CopyBytes(b.header.Extra) }
func (b *RootBlock) MixDigest() common.Hash        { return b.header.MixDigest }
func (b *RootBlock) Signature() [65]byte           { return b.header.Signature }

// Header returns a deep-copy of the entire root block header.
func (b *RootBlock) Header() *RootBlockHeader { return CopyRootBlockHeader(b.header) }

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

// Size returns the true SSZ encoded storage size of the block, either by
// encoding and returning it, or returning a previously cached value.
// It panics if the block cannot be encoded.
func (b *RootBlock) Size() uint64 {
	if size := b.size.Load(); size > 0 {
		return size
	}
	enc, err := b.MarshalSSZ()
	if err != nil {
		panic(fmt.Sprintf("cannot encode root block %s: %v", b.Hash(), err))
	}
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
// the sealed one. The body and tracking data are copied.
func (b *RootBlock) WithSeal(header *RootBlockHeader) *RootBlock {
	block := &RootBlock{
		header:            CopyRootBlockHeader(header),
		minorBlockHeaders: make(MinorBlockHeaders, len(b.minorBlockHeaders)),
		trackingData:      common.CopyBytes(b.trackingData),
	}
	for i := range b.minorBlockHeaders {
		block.minorBlockHeaders[i] = ctypes.CopyHeader(b.minorBlockHeaders[i])
	}
	return block
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

// Finalize sets the minor header hash, the coinbase address and the coinbase
// amount of the block in place, and refreshes the cached hash; the cached size
// is invalidated as well. Nil arguments take default values: zero amount and the
// zero address.
func (b *RootBlock) Finalize(coinbaseAmount *uint256.Int, coinbaseAddress *common.Address) *RootBlock {
	if coinbaseAmount == nil {
		coinbaseAmount = new(uint256.Int)
	}
	if coinbaseAddress == nil {
		coinbaseAddress = new(common.Address)
	}
	b.header.MinorHeaderHash = CalcMinorHeaderHash(b.minorBlockHeaders)
	b.header.CoinbaseAmount = *coinbaseAmount
	b.header.Coinbase = *coinbaseAddress
	h := b.header.Hash()
	b.hash.Store(&h)
	b.size.Store(0)
	return b
}

// AddMinorBlockHeader appends a shard block header to the block in place. The
// header is copied, and the minor header hash is updated by Finalize.
func (b *RootBlock) AddMinorBlockHeader(header *ctypes.Header) {
	b.minorBlockHeaders = append(b.minorBlockHeaders, ctypes.CopyHeader(header))
	b.size.Store(0)
}

// ExtendMinorBlockHeaderList appends the shard block headers created no later
// than createTime to the block in place. The matching headers are copied, and
// the minor header hash is updated by Finalize.
func (b *RootBlock) ExtendMinorBlockHeaderList(headers []*ctypes.Header, createTime uint64) {
	for _, header := range headers {
		if header.Time <= createTime {
			b.minorBlockHeaders = append(b.minorBlockHeaders, ctypes.CopyHeader(header))
		}
	}
	b.size.Store(0)
}
