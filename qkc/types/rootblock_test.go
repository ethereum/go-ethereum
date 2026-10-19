// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"encoding/binary"
	"math"
	"math/big"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ctypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	ssz "github.com/ferranbt/fastssz"
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
	bad[rootBlockFixedSize+196]++
	if err := new(RootBlock).UnmarshalSSZ(bad); err == nil {
		t.Error("expected error for bad extra data offset")
	}
}

// emptyMinorHeaderHash is CalcMinorHeaderHash over the empty list: a single
// zero leaf hashed together with the 8-byte big-endian zero length, i.e. the
// keccak256 of 40 zero bytes, matching pyquarkchain's calculate_merkle_root.
var emptyMinorHeaderHash = common.HexToHash("0xdaa77426c30c02a43d9fba4e841a6556c524d47030762eb14dc4af897e605d9b")

// TestCalcMinorHeaderHashTree pins the merkle tree of CalcMinorHeaderHash
// against values independently computed with pyquarkchain's
// calculate_merkle_root, covering the empty list, a single leaf, even, odd and
// non-power-of-two counts.
func TestCalcMinorHeaderHashTree(t *testing.T) {
	var a, b, c, d, e common.Hash
	a[0], b[0], c[0], d[0], e[0] = 0x01, 0x02, 0x03, 0x01, 0x02

	tests := []struct {
		name   string
		leaves []common.Hash
		want   common.Hash
	}{
		{"empty", nil, common.HexToHash("0xdaa77426c30c02a43d9fba4e841a6556c524d47030762eb14dc4af897e605d9b")},
		{"one", []common.Hash{a}, common.HexToHash("0xc46a48c98dc0e58bb11f278c1b8b491227df0b86824fe48780eb696d05905af5")},
		{"two", []common.Hash{a, b}, common.HexToHash("0x0e4383494e7bca2054ff377aa37ac85a554a4fe3d633da838b18e9acb5ddf8ff")},
		{"three", []common.Hash{a, b, c}, common.HexToHash("0x8eddc91dd27054aa531e7dfebc3810740a80fa42eb3892e7f6d4256b9ab809e6")},
		{"five", []common.Hash{a, b, c, d, e}, common.HexToHash("0x6d9c3af7b1136f88d798a7707cca3303b537f2aeb67f079e8e8ffc84e4699416")},
	}
	for _, tc := range tests {
		if have := sszMerkleRoot(tc.leaves); have != tc.want {
			t.Errorf("%s: sszMerkleRoot mismatch: have %s, want %s", tc.name, have.Hex(), tc.want.Hex())
		}
	}
}

// TestSszMerkleRootZeroPadding pins the zero padding value at every level for
// leaf counts whose tree is even before it turns odd (6, 10, 11, 12, 14). In
// pyquarkchain's calculate_merkle_root the padding hash doubles once per level,
// while a count of that shape makes the padding lag by one level if it is only
// doubled when a level turns out odd. TestCalcMinorHeaderHashTree cannot catch
// such a regression: every odd level in its cases comes before any even one.
// The values are from pyquarkchain's calculate_merkle_root.
func TestSszMerkleRootZeroPadding(t *testing.T) {
	distinctLeaves := func(n int) []common.Hash {
		leaves := make([]common.Hash, n)
		for i := range leaves {
			leaves[i][0] = byte(i + 1)
		}
		return leaves
	}

	tests := []struct {
		n    int
		want common.Hash
	}{
		{6, common.HexToHash("0x92be719d6e9a3231883ba0dcef0e34b2efdda5cf3a28d38aca40debe7c218dfb")},
		{7, common.HexToHash("0xe84f75438923b600a2c8301cbe4f71bce7c88dbbfe0eee11bd903b5030ab37ac")},
		{10, common.HexToHash("0xac142a568af9208828b16d3e460340d4650171b97135efae7983bb6743d02d51")},
		{11, common.HexToHash("0x16c7174d552b969841a00ac054a406c04d2aeaf52b7ea6a178590ae021b5abd3")},
		{12, common.HexToHash("0x5994662588ff256d81e68a07f1acf77472fa71672052f8e4e9475a170d2ec610")},
		{13, common.HexToHash("0x61244d521b09d3370888b9c2942dca8ea1e3240d0d25d559d583d36b3d8da913")},
		{14, common.HexToHash("0x29b6156e3a1e23b250376567cb45bb5896719dbccafe10bbf8e7844550e94c7c")},
		{16, common.HexToHash("0x77ed9f4eb93a2b1d06a7000f5f82618f56f61145aae120edbba2d70abd01d411")},
	}
	for _, tc := range tests {
		if have := sszMerkleRoot(distinctLeaves(tc.n)); have != tc.want {
			t.Errorf("%d leaves: sszMerkleRoot mismatch: have %s, want %s", tc.n, have.Hex(), tc.want.Hex())
		}
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
	if have := empty.MinorHeaderHash(); have != emptyMinorHeaderHash {
		t.Errorf("empty minor header hash mismatch: have %x, want %x", have, emptyMinorHeaderHash)
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
		"31010000" + // offset of extra
		"0700000000000000" + // nonce
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
	bad[rootBlockFixedSize+196]++ // corrupt the header's extra data offset
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
	bad[196]++ // extra data offset no longer equals the sealed fixed size
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

// TestRootBlockRejectsNonCanonicalEmptyMinorHeaderList verifies that a
// four-byte zero offset table standing for an empty shard block header list is
// rejected instead of being silently absorbed.
func TestRootBlockRejectsNonCanonicalEmptyMinorHeaderList(t *testing.T) {
	block := NewRootBlock(&RootBlockHeader{}, nil, []byte("T"))
	encoded, err := block.MarshalSSZ()
	if err != nil {
		t.Fatalf("MarshalSSZ: %v", err)
	}

	headersOffset := int(binary.LittleEndian.Uint32(encoded[4:8]))
	trackingOffset := int(binary.LittleEndian.Uint32(encoded[8:12]))
	if headersOffset != trackingOffset {
		t.Fatalf("expected empty header-list region, got headers=%d tracking=%d", headersOffset, trackingOffset)
	}

	// Insert a four-byte zero offset table for the empty dynamic list.
	malformed := make([]byte, len(encoded)+4)
	copy(malformed[:trackingOffset], encoded[:trackingOffset])
	copy(malformed[trackingOffset+4:], encoded[trackingOffset:])
	binary.LittleEndian.PutUint32(malformed[8:12], uint32(trackingOffset+4))

	var decoded RootBlock
	if err := decoded.UnmarshalSSZ(malformed); err != ssz.ErrOffset {
		t.Fatalf("UnmarshalSSZ error = %v, want %v", err, ssz.ErrOffset)
	}
}

// TestRootBlockDecodedSizeMatchesCanonicalEncoding verifies that the size
// cached by UnmarshalSSZ is the canonical encoding size, which requires a
// decoded block to re-encode to the bytes it was decoded from.
func TestRootBlockDecodedSizeMatchesCanonicalEncoding(t *testing.T) {
	block := NewRootBlock(&RootBlockHeader{}, nil, []byte("tracking"))
	encoded, err := block.MarshalSSZ()
	if err != nil {
		t.Fatalf("MarshalSSZ: %v", err)
	}

	var decoded RootBlock
	if err := decoded.UnmarshalSSZ(encoded); err != nil {
		t.Fatalf("UnmarshalSSZ: %v", err)
	}
	reencoded, err := decoded.MarshalSSZ()
	if err != nil {
		t.Fatalf("MarshalSSZ after decode: %v", err)
	}
	if !reflect.DeepEqual(reencoded, encoded) {
		t.Errorf("re-encoded bytes differ from the input")
	}
	if got, want := decoded.Size(), uint64(len(reencoded)); got != want {
		t.Fatalf("Size() = %d, want canonical encoding size %d", got, want)
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
	if empty.MinorHeaderHash() != emptyMinorHeaderHash {
		t.Errorf("empty minor header hash mismatch: have %x, want %x", empty.MinorHeaderHash(), emptyMinorHeaderHash)
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

// TestRootBlockNilInputs pins the nil policy: a nil block header or shard block
// header is a programming error that never yields a successful encoding or
// commitment.
func TestRootBlockNilInputs(t *testing.T) {
	assertPanics := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s: expected panic", name)
			}
		}()
		fn()
	}

	// A nil block header fails fast in the constructors and in every method
	// that has to touch it.
	assertPanics("NewRootBlock", func() { NewRootBlock(nil, nil, nil) })
	assertPanics("NewRootBlockWithHeader", func() { NewRootBlockWithHeader(nil) })
	assertPanics("CopyRootBlockHeader", func() { CopyRootBlockHeader(nil) })
	var zero RootBlock
	assertPanics("Header", func() { zero.Header() })
	assertPanics("Hash", func() { zero.Hash() })
	assertPanics("MarshalSSZ", func() { zero.MarshalSSZ() })
	assertPanics("Size", func() { zero.Size() })

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

// TestRootBlockWithBodyContract pins that WithBody keeps the previous header
// commitment while replacing the body, so the result is only a complete block
// once Finalize has recomputed MinorHeaderHash from the new body.
func TestRootBlockWithBodyContract(t *testing.T) {
	block := newTestRootBlock()
	commitment := block.MinorHeaderHash()

	body := MinorBlockHeaders{newTestMinorHeader(2, 2)}
	if CalcMinorHeaderHash(body) == commitment {
		t.Fatal("test body has the same commitment as the original body")
	}
	replaced := block.WithBody(body, []byte("new tracking"))
	if have := replaced.MinorHeaderHash(); have != commitment {
		t.Errorf("WithBody recomputed the commitment: have %x, want %x", have, commitment)
	}

	replaced.Finalize(nil, nil)
	if have, want := replaced.MinorHeaderHash(), CalcMinorHeaderHash(replaced.MinorBlockHeaders()); have != want {
		t.Errorf("Finalize did not recompute the commitment: have %x, want %x", have, want)
	}
	if replaced.Hash() != replaced.Header().Hash() {
		t.Error("hash cache not refreshed by Finalize")
	}

	// The original block still commits to its own body.
	if have, want := block.MinorHeaderHash(), commitment; have != want {
		t.Errorf("WithBody changed the original block: have %x, want %x", have, want)
	}
}

// TestRootBlockConcurrentHash pins that concurrent Hash calls on a published
// block agree with each other. The atomic cache makes the cache itself race
// free; it does not make the block safe to mutate concurrently, which is why
// this test never mutates it.
func TestRootBlockConcurrentHash(t *testing.T) {
	block := newTestRootBlock()
	want := block.Hash()

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 100 {
				if have := block.Hash(); have != want {
					t.Errorf("hash mismatch: have %x, want %x", have, want)
					return
				}
			}
		})
	}
	wg.Wait()
}
