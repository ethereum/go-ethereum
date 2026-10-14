// Copyright 2026-2027 The QuarkChain Authors
// This file is part of the QuarkChain library.
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

package types

import (
	"encoding/binary"
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ctypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

func newTestMinorHeader(shard uint32, number int64) *ctypes.Header {
	extra := &ctypes.QKCExtra{ShardID: shard, Cursor: ctypes.XShardCursor{RootBlockHeight: 1}}
	return &ctypes.Header{
		Number:     big.NewInt(number),
		Difficulty: common.Big0,
		Time:       uint64(number),
		Extra:      extra.Encode(),
		MixDigest:  common.BigToHash(big.NewInt(1000)),
		BaseFee:    big.NewInt(7),
	}
}

func newTestRootBlock() *RootBlock {
	header := &RootBlockHeader{
		Version:         1,
		Number:          10,
		ParentHash:      common.HexToHash("0x01"),
		Coinbase:        common.HexToAddress("0x02"),
		CoinbaseAmount:  *uint256.NewInt(100),
		Time:            1000,
		Difficulty:      *uint256.NewInt(5),
		TotalDifficulty: *uint256.NewInt(50),
		Nonce:           7,
		Extra:           []byte("extra"),
		MixDigest:       common.HexToHash("0x03"),
	}
	headers := MinorBlockHeaders{newTestMinorHeader(0, 1), newTestMinorHeader(1, 1)}
	return NewRootBlock(header, headers, []byte("tracking"))
}

func TestRootBlockEncoding(t *testing.T) {
	block := newTestRootBlock()
	enc, err := block.MarshalSSZ()
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}
	var dec RootBlock
	if err := dec.UnmarshalSSZ(enc); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if dec.Hash() != block.Hash() {
		t.Errorf("hash mismatch: have %x, want %x", dec.Hash(), block.Hash())
	}
	if !reflect.DeepEqual(dec.Header(), block.Header()) {
		t.Errorf("header mismatch:\nhave %+v\nwant %+v", dec.Header(), block.Header())
	}
	if len(dec.MinorBlockHeaders()) != 2 {
		t.Fatalf("minor block header count mismatch: have %d, want 2", len(dec.MinorBlockHeaders()))
	}
	for i, header := range dec.MinorBlockHeaders() {
		if header.Hash() != block.MinorBlockHeaders()[i].Hash() {
			t.Errorf("minor block header %d mismatch", i)
		}
	}
	if string(dec.TrackingData()) != "tracking" {
		t.Errorf("tracking data mismatch: have %q", dec.TrackingData())
	}
	if dec.Size() != uint64(len(enc)) || block.Size() != uint64(len(enc)) {
		t.Errorf("size mismatch: have %d and %d, want %d", dec.Size(), block.Size(), len(enc))
	}
}

func TestRootBlockDecodeInvalid(t *testing.T) {
	enc, err := newTestRootBlock().MarshalSSZ()
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}
	// The tracking data comes last, so only cutting into the header or the
	// shard block headers breaks the encoding.
	trackingOffset := len(enc) - len("tracking")
	for size := range trackingOffset {
		if err := new(RootBlock).UnmarshalSSZ(enc[:size]); err == nil {
			t.Errorf("size %d: expected error", size)
		}
	}
	// Corrupt the offset of the header's extra data.
	bad := common.CopyBytes(enc)
	bad[rootBlockFixedSize+204]++
	if err := new(RootBlock).UnmarshalSSZ(bad); err == nil {
		t.Error("expected error for bad extra data offset")
	}
}

func TestRootBlockMinorHeaderHash(t *testing.T) {
	block := newTestRootBlock()
	if have, want := block.MinorHeaderHash(), CalcMinorHeaderHash(block.MinorBlockHeaders()); have != want {
		t.Errorf("minor header hash mismatch: have %x, want %x", have, want)
	}
	if block.MinorBlockHeader(block.MinorBlockHeaders()[1].Hash()) == nil {
		t.Error("minor block header not found")
	}
	empty := NewRootBlock(block.Header(), nil, nil)
	if have := empty.MinorHeaderHash(); have != ctypes.EmptyRootHash {
		t.Errorf("empty minor header hash mismatch: have %x, want %x", have, ctypes.EmptyRootHash)
	}
}

func TestRootBlockSeal(t *testing.T) {
	block := newTestRootBlock()
	sealed := block.WithMiningResult(8, common.HexToHash("0x04"), nil)
	if sealed.Header().SealHash() != block.Header().SealHash() {
		t.Error("seal hash depends on the seal")
	}
	if sealed.Hash() == block.Hash() {
		t.Error("hash does not depend on the seal")
	}

	key, _ := crypto.GenerateKey()
	header := sealed.Header()
	if err := header.Sign(key); err != nil {
		t.Fatalf("failed to sign: %v", err)
	}
	if header.SealHash() != sealed.Header().SealHash() {
		t.Error("seal hash depends on the signature")
	}
	if !header.VerifySignature(&key.PublicKey) {
		t.Error("signature does not verify")
	}
	other, _ := crypto.GenerateKey()
	if header.VerifySignature(&other.PublicKey) {
		t.Error("signature verifies with another key")
	}
}

func TestRootBlockHashSSZ(t *testing.T) {
	header := newTestRootBlock().Header()
	header.MinorHeaderHash = common.HexToHash("0x06")
	unsealed := "01000000" + // version
		"0a000000" + // number
		strings.Repeat("00", 31) + "01" + // parentHash
		strings.Repeat("00", 31) + "06" + // minorHeaderHash
		strings.Repeat("00", 19) + "02" + // coinbase
		"64" + strings.Repeat("00", 31) + // coinbaseAmount
		"e803000000000000" + // time
		"05" + strings.Repeat("00", 31) + // difficulty
		"32" + strings.Repeat("00", 31) // totalDifficulty
	sealEnc := common.FromHex("0x" + unsealed +
		"c8000000" + // offset of extra
		"6578747261") // extra
	if have, want := header.SealHash(), crypto.Keccak256Hash(sealEnc); have != want {
		t.Fatalf("seal hash mismatch: have %x, want %x", have, want)
	}
	hashEnc := common.FromHex("0x" + unsealed +
		"0700000000000000" + // nonce
		"31010000" + // offset of extra
		strings.Repeat("00", 31) + "03" + // mixDigest
		strings.Repeat("00", 65) + // signature
		"6578747261") // extra
	if have, want := header.Hash(), crypto.Keccak256Hash(hashEnc); have != want {
		t.Fatalf("hash mismatch: have %x, want %x", have, want)
	}
}

func TestRootBlockCreateBlockToAppend(t *testing.T) {
	parent := newTestRootBlock().Header()
	block := parent.CreateBlockToAppend(nil, nil, nil, nil, nil)
	if block.Number() != parent.Number+1 {
		t.Errorf("number mismatch: have %d, want %d", block.Number(), parent.Number+1)
	}
	if block.ParentHash() != parent.Hash() {
		t.Errorf("parent hash mismatch: have %x, want %x", block.ParentHash(), parent.Hash())
	}
	if block.Time() != parent.Time+1 {
		t.Errorf("time mismatch: have %d, want %d", block.Time(), parent.Time+1)
	}
	if want := new(uint256.Int).Add(&parent.TotalDifficulty, &parent.Difficulty); !block.TotalDifficulty().Eq(want) {
		t.Errorf("total difficulty mismatch: have %v, want %v", block.TotalDifficulty(), want)
	}

	block.AddMinorBlockHeader(newTestMinorHeader(0, 2))
	block.ExtendMinorBlockHeaderList([]*ctypes.Header{newTestMinorHeader(1, 2), newTestMinorHeader(1, 5000)}, block.Time())
	coinbase := common.HexToAddress("0x05")
	block.Finalize(uint256.NewInt(42), &coinbase)
	if len(block.MinorBlockHeaders()) != 2 {
		t.Fatalf("minor block header count mismatch: have %d, want 2", len(block.MinorBlockHeaders()))
	}
	if have, want := block.MinorHeaderHash(), CalcMinorHeaderHash(block.MinorBlockHeaders()); have != want {
		t.Errorf("minor header hash mismatch: have %x, want %x", have, want)
	}
	if block.Coinbase() != coinbase || block.CoinbaseAmount().Uint64() != 42 {
		t.Errorf("coinbase mismatch: have %x %v", block.Coinbase(), block.CoinbaseAmount())
	}
	if block.Hash() != block.Header().Hash() {
		t.Error("cached hash not updated by Finalize")
	}
}

func mustMarshalSSZ(t *testing.T, b *RootBlock) []byte {
	t.Helper()
	enc, err := b.MarshalSSZ()
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}
	return enc
}

// TestRootBlockDecodeCacheReset verifies that re-decoding into a block whose
// hash cache is populated does not return the stale hash.
func TestRootBlockDecodeCacheReset(t *testing.T) {
	blockA := newTestRootBlock()
	blockB := NewRootBlock(blockA.Header(), nil, []byte("b"))
	if blockA.Hash() == blockB.Hash() {
		t.Fatal("test blocks have the same hash")
	}
	var dec RootBlock
	if err := dec.UnmarshalSSZ(mustMarshalSSZ(t, blockA)); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	dec.Hash() // populate the hash cache
	encB := mustMarshalSSZ(t, blockB)
	if err := dec.UnmarshalSSZ(encB); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if dec.Hash() != blockB.Hash() {
		t.Errorf("stale hash cache after re-decode: have %x, want %x", dec.Hash(), blockB.Hash())
	}
	if dec.Size() != uint64(len(encB)) {
		t.Errorf("stale size cache after re-decode: have %d, want %d", dec.Size(), len(encB))
	}
}

// TestRootBlockDecodeKeepsStateOnError verifies that a failed decode leaves
// the receiver, including its caches, untouched.
func TestRootBlockDecodeKeepsStateOnError(t *testing.T) {
	block := newTestRootBlock()
	var dec RootBlock
	if err := dec.UnmarshalSSZ(mustMarshalSSZ(t, block)); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	dec.Hash() // populate the hash cache
	wantHash, wantSize := dec.Hash(), dec.Size()
	wantHeaders, wantTracking := len(dec.MinorBlockHeaders()), string(dec.TrackingData())

	bad := common.CopyBytes(mustMarshalSSZ(t, block))
	bad[rootBlockFixedSize+204]++ // corrupt the header's extra data offset
	if err := dec.UnmarshalSSZ(bad); err == nil {
		t.Fatal("expected error")
	}
	if dec.Hash() != wantHash || dec.Size() != wantSize ||
		len(dec.MinorBlockHeaders()) != wantHeaders || string(dec.TrackingData()) != wantTracking {
		t.Errorf("decode failure mutated state: hash %x size %d headers %d tracking %q",
			dec.Hash(), dec.Size(), len(dec.MinorBlockHeaders()), string(dec.TrackingData()))
	}
	if dec.Hash() != block.Hash() {
		t.Error("decode failure corrupted the cached hash")
	}
}

// TestRootBlockHeaderDecodeInvalid verifies that the header decoder rejects
// truncated input and a bad extra data offset without modifying the receiver.
func TestRootBlockHeaderDecodeInvalid(t *testing.T) {
	header := newTestRootBlock().Header()
	enc, err := header.MarshalSSZ()
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}
	h := new(RootBlockHeader)
	if err := h.UnmarshalSSZ(enc); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if err := h.UnmarshalSSZ(enc[:sealedFixedSize-1]); err == nil {
		t.Error("expected error for truncated input")
	}
	bad := common.CopyBytes(enc)
	bad[204]++ // extra data offset no longer equals the sealed fixed size
	if err := h.UnmarshalSSZ(bad); err == nil {
		t.Error("expected error for bad extra data offset")
	}
	if h.Version != header.Version || h.Number != header.Number ||
		!h.CoinbaseAmount.Eq(&header.CoinbaseAmount) || !h.Difficulty.Eq(&header.Difficulty) ||
		!h.TotalDifficulty.Eq(&header.TotalDifficulty) || string(h.Extra) != string(header.Extra) {
		t.Error("decode failure mutated the header")
	}
}

// TestRootBlockDecodeBadHeaderList verifies that the decoder rejects malformed
// shard block header lists.
func TestRootBlockDecodeBadHeaderList(t *testing.T) {
	block := newTestRootBlock()
	enc := mustMarshalSSZ(t, block)
	headersOffset := rootBlockFixedSize + block.Header().SizeSSZ()
	trackingOffset := len(enc) - len(block.TrackingData())
	if have := binary.LittleEndian.Uint32(enc[4:8]); int(have) != headersOffset {
		t.Fatalf("header list offset mismatch: have %d, want %d", have, headersOffset)
	}

	badFirstOffset := func(v uint32) []byte {
		bad := common.CopyBytes(enc)
		binary.LittleEndian.PutUint32(bad[headersOffset:headersOffset+4], v)
		return bad
	}
	// The first offset must be divisible by four.
	if err := new(RootBlock).UnmarshalSSZ(badFirstOffset(5)); err == nil {
		t.Error("expected error for non-divisible first offset")
	}
	// The first offset must not imply more elements than the input can hold.
	// 0xfffffffc is divisible by four and truncates to a negative int on
	// 32-bit platforms.
	if err := new(RootBlock).UnmarshalSSZ(badFirstOffset(0xfffffffc)); err == nil {
		t.Error("expected error for too large first offset")
	}
	// Offsets must not decrease.
	bad := common.CopyBytes(enc)
	binary.LittleEndian.PutUint32(bad[headersOffset+4:headersOffset+8], 4)
	if err := new(RootBlock).UnmarshalSSZ(bad); err == nil {
		t.Error("expected error for decreasing offsets")
	}
	// Duplicate offsets imply a zero-length element, which is not a valid
	// RLP header.
	bad = common.CopyBytes(enc)
	binary.LittleEndian.PutUint32(bad[headersOffset+4:headersOffset+8], 8)
	if err := new(RootBlock).UnmarshalSSZ(bad); err == nil {
		t.Error("expected error for zero-length element")
	}
	// Trailing bytes merged into the last element make it invalid RLP.
	bad = make([]byte, 0, len(enc)+1)
	bad = append(bad, enc[:trackingOffset]...)
	bad = append(bad, 0x00)
	bad = append(bad, enc[trackingOffset:]...)
	binary.LittleEndian.PutUint32(bad[8:12], uint32(trackingOffset+1))
	if err := new(RootBlock).UnmarshalSSZ(bad); err == nil {
		t.Error("expected error for trailing garbage in the header list")
	}
}

// TestRootBlockRoundTripBoundaries verifies encoding and decoding at the
// boundaries of the field types: empty dynamic fields, maximum width integers
// and a large extra data blob.
func TestRootBlockRoundTripBoundaries(t *testing.T) {
	roundtrip := func(t *testing.T, block *RootBlock) {
		t.Helper()
		var dec RootBlock
		if err := dec.UnmarshalSSZ(mustMarshalSSZ(t, block)); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if dec.Hash() != block.Hash() {
			t.Errorf("hash mismatch: have %x, want %x", dec.Hash(), block.Hash())
		}
		if dec.Size() != block.Size() {
			t.Errorf("size mismatch: have %d, want %d", dec.Size(), block.Size())
		}
	}

	// Empty extra data, empty header list and empty tracking data.
	header := newTestRootBlock().Header()
	header.Extra = nil
	empty := NewRootBlock(header, nil, nil)
	roundtrip(t, empty)
	if len(empty.Extra()) != 0 || len(empty.MinorBlockHeaders()) != 0 || len(empty.TrackingData()) != 0 {
		t.Error("expected empty dynamic fields")
	}
	if empty.MinorHeaderHash() != ctypes.EmptyRootHash {
		t.Errorf("empty minor header hash mismatch: have %x, want %x", empty.MinorHeaderHash(), ctypes.EmptyRootHash)
	}

	// Maximum width integers.
	header = newTestRootBlock().Header()
	header.Number = math.MaxUint32
	header.Time = math.MaxUint64
	header.Nonce = math.MaxUint64
	header.CoinbaseAmount = *new(uint256.Int).SetAllOne()
	header.Difficulty = *new(uint256.Int).SetAllOne()
	header.TotalDifficulty = *new(uint256.Int).SetAllOne()
	maxBlock := NewRootBlock(header, nil, nil)
	roundtrip(t, maxBlock)
	decMax := new(RootBlock)
	if err := decMax.UnmarshalSSZ(mustMarshalSSZ(t, maxBlock)); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if decMax.Number() != math.MaxUint32 || decMax.Time() != math.MaxUint64 || decMax.Nonce() != math.MaxUint64 {
		t.Error("maximum width integers not preserved")
	}
	if !decMax.Difficulty().Eq(new(uint256.Int).SetAllOne()) ||
		!decMax.TotalDifficulty().Eq(new(uint256.Int).SetAllOne()) ||
		!decMax.CoinbaseAmount().Eq(new(uint256.Int).SetAllOne()) {
		t.Error("maximum width uint256 values not preserved")
	}

	// Large extra data.
	header = newTestRootBlock().Header()
	header.Extra = make([]byte, 65536)
	for i := range header.Extra {
		header.Extra[i] = byte(i)
	}
	large := NewRootBlock(header, nil, nil)
	roundtrip(t, large)
	decLarge := new(RootBlock)
	if err := decLarge.UnmarshalSSZ(mustMarshalSSZ(t, large)); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(decLarge.Extra()) != 65536 {
		t.Errorf("extra data length mismatch: have %d, want 65536", len(decLarge.Extra()))
	}
}

// TestRootBlockNilShardHeaders pins the nil policy: a nil shard block header
// is a programming error that never yields a successful encoding or
// commitment.
func TestRootBlockNilShardHeaders(t *testing.T) {
	assertPanics := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s: expected panic", name)
			}
		}()
		fn()
	}

	header := newTestRootBlock().Header()
	assertPanics("NewRootBlock", func() { NewRootBlock(header, MinorBlockHeaders{nil}, nil) })
	assertPanics("WithBody", func() { header.CreateBlockToAppend(nil, nil, nil, nil, nil).WithBody(MinorBlockHeaders{nil}, nil) })
	assertPanics("AddMinorBlockHeader", func() { header.CreateBlockToAppend(nil, nil, nil, nil, nil).AddMinorBlockHeader(nil) })
	assertPanics("ExtendMinorBlockHeaderList", func() {
		header.CreateBlockToAppend(nil, nil, nil, nil, nil).ExtendMinorBlockHeaderList([]*ctypes.Header{nil}, 0)
	})
	assertPanics("CalcMinorHeaderHash", func() { CalcMinorHeaderHash(MinorBlockHeaders{newTestMinorHeader(0, 1), nil}) })

	// The encoder rejects a nil element even if one is injected into the
	// private body slice, bypassing the constructors.
	block := NewRootBlockWithHeader(header)
	block.minorBlockHeaders = MinorBlockHeaders{newTestMinorHeader(0, 1), nil}
	if _, err := block.MarshalSSZ(); err == nil {
		t.Error("MarshalSSZ accepted a nil shard block header")
	}
	assertPanics("Size", func() { block.Size() })
}

// TestRootBlockOwnership pins that blocks own their data: later changes to
// the caller's headers never affect the block.
func TestRootBlockOwnership(t *testing.T) {
	minor := newTestMinorHeader(0, 1)
	block := newTestRootBlock()
	block.AddMinorBlockHeader(minor)
	block.Finalize(nil, nil)
	want := block.MinorHeaderHash()

	// Mutating the caller's header after it was added must not change the
	// block's body or its commitment.
	minor.Number.SetInt64(999)
	minor.Extra[0] ^= 0xff
	if have := CalcMinorHeaderHash(block.MinorBlockHeaders()); have != want {
		t.Errorf("block commitment changed with the caller's header: have %x, want %x", have, want)
	}

	// The header and tracking data passed to NewRootBlock are copied.
	header := newTestRootBlock().Header()
	tracking := []byte("tracking")
	other := NewRootBlock(header, nil, tracking)
	header.Time = 123456
	if other.Time() == 123456 {
		t.Error("NewRootBlock shares the caller's header")
	}
	tracking[0] = 'x'
	if string(other.TrackingData()) != "tracking" {
		t.Errorf("NewRootBlock shares the caller's tracking data: have %q", other.TrackingData())
	}

	// WithSeal copies the body, so the two blocks diverge independently.
	// block holds three shard headers at this point.
	sealed := block.WithSeal(block.Header())
	block.AddMinorBlockHeader(newTestMinorHeader(2, 3))
	if len(sealed.MinorBlockHeaders()) != 3 {
		t.Errorf("WithSeal shares the body: have %d headers, want 3", len(sealed.MinorBlockHeaders()))
	}
}

// TestRootBlockSizeUnencodable pins that Size does not silently return zero
// when the block cannot be encoded.
func TestRootBlockSizeUnencodable(t *testing.T) {
	block := newTestRootBlock()
	bad := newTestMinorHeader(0, 1)
	bad.Difficulty = big.NewInt(-1) // geth rlp refuses negative big.Int
	block.AddMinorBlockHeader(bad)
	defer func() {
		if recover() == nil {
			t.Error("expected panic")
		}
	}()
	block.Size()
}

// TestRootBlockSizeCacheInvalidation verifies that body mutations invalidate
// the cached size while the cached hash stays valid.
func TestRootBlockSizeCacheInvalidation(t *testing.T) {
	block := newTestRootBlock()
	sizeBefore := block.Size()
	block.AddMinorBlockHeader(newTestMinorHeader(2, 3))
	if block.Size() <= sizeBefore {
		t.Errorf("size cache not invalidated by AddMinorBlockHeader: %d -> %d", sizeBefore, block.Size())
	}
	if block.Hash() != block.Header().Hash() {
		t.Error("hash cache invalid after AddMinorBlockHeader")
	}

	block.Finalize(nil, nil)
	sizeBefore = block.Size()
	block.ExtendMinorBlockHeaderList([]*ctypes.Header{newTestMinorHeader(3, 4)}, block.Time())
	if block.Size() <= sizeBefore {
		t.Errorf("size cache not invalidated by ExtendMinorBlockHeaderList: %d -> %d", sizeBefore, block.Size())
	}
	if block.Hash() != block.Header().Hash() {
		t.Error("hash cache invalid after ExtendMinorBlockHeaderList")
	}
}
