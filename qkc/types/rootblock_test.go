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
