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
	"bytes"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// The tests below check two different hashes on purpose, and never substitute
// one for the other:
//
//   - Hash() is the final block hash: keccak256(RLP(header)) over every field,
//     including the extra data (and therefore the PoW seal) and MixDigest.
//   - QKCSealHash() is the hash a block is mined against: the same, but with the
//     PoW seal region of the extra data zeroed, so that the seal is excluded and
//     the difficulty (in MixDigest) is still included.
//
// They cannot be cross-validated against the legacy QKC codec because the 2.0
// header uses RLP: the same field set is encoded differently, so the legacy
// bytes are irrelevant by construction (regenesis changes block identity). The
// reference used instead is the explicit RLP field list in
// qkcSealHashReference, which does not go through Header.EncodeRLP/Hash.

func TestQKCExtraEncoding(t *testing.T) {
	want := &QKCExtra{
		Version:        1,
		ShardID:        0x00010003,
		PrevRootBlock:  common.BytesToHash(bytes.Repeat([]byte{0xaa}, 32)),
		Cursor:         XShardCursor{RootBlockHeight: 1, MinorBlockIndex: 2, XShardDepositIndex: 3},
		XShardGasUsed:  21000,
		XShardGasLimit: 1_000_000,
		Nonce:          EncodeNonce(0x0102030405060708),
		MixHash:        common.BytesToHash(bytes.Repeat([]byte{0xbb}, 32)),
	}
	wantExtra := common.FromHex("0x" +
		"00000001" + // version
		"00010003" + // shardId
		strings.Repeat("aa", 32) + // prevRootBlock
		"0000000000000001" + "0000000000000002" + "0000000000000003" + // cursor
		"0000000000005208" + // xShardGasUsed
		"00000000000f4240" + // xShardGasLimit
		"0102030405060708" + // nonce
		strings.Repeat("bb", 32)) // mixHash

	extra := want.Encode()
	if !bytes.Equal(extra, wantExtra) {
		t.Fatalf("encoding mismatch:\nhave %x\nwant %x", extra, wantExtra)
	}
	header := &Header{Extra: extra}
	have, err := header.QKCExtra()
	if err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !reflect.DeepEqual(have, want) {
		t.Fatalf("decoding mismatch:\nhave %+v\nwant %+v", have, want)
	}
}

func TestQKCExtraInvalidLength(t *testing.T) {
	for _, size := range []int{0, 32, QKCExtraSize - 1, QKCExtraSize + 1} {
		if _, err := DecodeQKCExtra(make([]byte, size)); err == nil {
			t.Errorf("size %d: expected error", size)
		}
	}
}

func TestQKCExtraEncodeSize(t *testing.T) {
	max := &QKCExtra{
		Version:        ^uint32(0),
		ShardID:        ^uint32(0),
		PrevRootBlock:  common.BytesToHash(bytes.Repeat([]byte{0xff}, 32)),
		Cursor:         XShardCursor{RootBlockHeight: ^uint64(0), MinorBlockIndex: ^uint64(0), XShardDepositIndex: ^uint64(0)},
		XShardGasUsed:  ^uint64(0),
		XShardGasLimit: ^uint64(0),
		Nonce:          EncodeNonce(^uint64(0)),
		MixHash:        common.BytesToHash(bytes.Repeat([]byte{0xff}, 32)),
	}
	for _, extra := range []*QKCExtra{{}, max} {
		if n := len(extra.Encode()); n != QKCExtraSize {
			t.Fatalf("encoded to %d bytes, want %d", n, QKCExtraSize)
		}
	}
}

func TestQKCExtraRoundTrip(t *testing.T) {
	cases := []*QKCExtra{
		{},
		{
			Version:        1,
			ShardID:        0x00010003,
			PrevRootBlock:  common.BytesToHash(bytes.Repeat([]byte{0xaa}, 32)),
			Cursor:         XShardCursor{RootBlockHeight: 1, MinorBlockIndex: 2, XShardDepositIndex: 3},
			XShardGasUsed:  21000,
			XShardGasLimit: 1_000_000,
			Nonce:          EncodeNonce(0x0102030405060708),
			MixHash:        common.BytesToHash(bytes.Repeat([]byte{0xbb}, 32)),
		},
		{
			Version:        ^uint32(0),
			ShardID:        ^uint32(0),
			PrevRootBlock:  common.BytesToHash(bytes.Repeat([]byte{0xff}, 32)),
			Cursor:         XShardCursor{RootBlockHeight: ^uint64(0), MinorBlockIndex: ^uint64(0), XShardDepositIndex: ^uint64(0)},
			XShardGasUsed:  ^uint64(0),
			XShardGasLimit: ^uint64(0),
			Nonce:          EncodeNonce(^uint64(0)),
			MixHash:        common.BytesToHash(bytes.Repeat([]byte{0xff}, 32)),
		},
	}
	for i, want := range cases {
		have, err := DecodeQKCExtra(want.Encode())
		if err != nil {
			t.Fatalf("case %d: failed to decode: %v", i, err)
		}
		if !reflect.DeepEqual(have, want) {
			t.Fatalf("case %d: decoding mismatch:\nhave %+v\nwant %+v", i, have, want)
		}
	}
}

func TestQKCExtraDecodeCopiesInput(t *testing.T) {
	prevRootBlock := common.BytesToHash(bytes.Repeat([]byte{0xaa}, 32))
	extra := (&QKCExtra{PrevRootBlock: prevRootBlock, MixHash: prevRootBlock}).Encode()

	have, err := DecodeQKCExtra(extra)
	if err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	for i := range extra {
		extra[i] = 0
	}
	if have.PrevRootBlock != prevRootBlock || have.MixHash != prevRootBlock {
		t.Fatalf("decoded value aliases the input: %x/%x", have.PrevRootBlock, have.MixHash)
	}
}

// TestQKCDifficultyConsistency checks that the PoW difficulty round-trips
// through MixDigest, that the header's own Difficulty stays zero, and that a
// difficulty change is visible in both the seal hash and the block hash.
func TestQKCDifficultyConsistency(t *testing.T) {
	diff := big.NewInt(1_000_000)
	header := &Header{}
	header.SetQKCDifficulty(diff)
	if have := header.QKCDifficulty(); have.Cmp(diff) != 0 {
		t.Fatalf("difficulty mismatch: have %v, want %v", have, diff)
	}
	// The header's own Difficulty must stay zero: a non-zero value would flip
	// the EVM into the pre-Merge path and contradict the design.
	if header.Difficulty == nil || header.Difficulty.Sign() != 0 {
		t.Fatalf("header difficulty must stay zero, have %v", header.Difficulty)
	}
	if want := common.BigToHash(diff); header.MixDigest != want {
		t.Fatalf("difficulty not stored in MixDigest: have %x, want %x", header.MixDigest, want)
	}
	// A nil difficulty means a zero PoW difficulty.
	header.SetQKCDifficulty(nil)
	if have := header.QKCDifficulty(); have.Sign() != 0 {
		t.Fatalf("nil difficulty: have %v, want 0", have)
	}
	if header.Difficulty == nil || header.Difficulty.Sign() != 0 {
		t.Fatalf("header difficulty must stay zero, have %v", header.Difficulty)
	}
	// Setting a difficulty after a previous one must not leave the old value.
	header.SetQKCDifficulty(big.NewInt(7))
	header.SetQKCDifficulty(big.NewInt(9))
	if have := header.QKCDifficulty(); have.Cmp(big.NewInt(9)) != 0 {
		t.Fatalf("difficulty was not replaced: have %v, want 9", have)
	}
}

func TestQKCSeal(t *testing.T) {
	header := &Header{Extra: (&QKCExtra{Version: 1, ShardID: 3, Cursor: XShardCursor{RootBlockHeight: 7}}).Encode()}
	header.SetQKCDifficulty(big.NewInt(11))

	nonce := EncodeNonce(0x0102030405060708)
	mixHash := common.BytesToHash(bytes.Repeat([]byte{0xcc}, 32))
	if err := header.SetQKCSeal(nonce, mixHash); err != nil {
		t.Fatalf("failed to set seal: %v", err)
	}
	if len(header.Extra) != QKCExtraSize {
		t.Fatalf("seal changed the extra size: %d", len(header.Extra))
	}
	have, err := header.QKCExtra()
	if err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if have.Nonce != nonce || have.MixHash != mixHash {
		t.Fatalf("seal mismatch: have %x/%x, want %x/%x", have.Nonce, have.MixHash, nonce, mixHash)
	}
	if have.Version != 1 || have.ShardID != 3 || have.Cursor.RootBlockHeight != 7 {
		t.Fatalf("seal overwrote other fields: %+v", have)
	}
	// Setting the seal must not touch the geth header fields that have to stay
	// zero after the Merge.
	if header.Nonce != EncodeNonce(0) {
		t.Fatalf("seal set the header nonce: %x", header.Nonce)
	}
	if header.Difficulty == nil || header.Difficulty.Sign() != 0 {
		t.Fatalf("seal changed the header difficulty: %v", header.Difficulty)
	}
	if header.QKCDifficulty().Cmp(big.NewInt(11)) != 0 {
		t.Fatalf("seal changed the PoW difficulty: %v", header.QKCDifficulty())
	}
	// Writing a seal into a header that carries no QuarkChain extra must fail.
	if err := (&Header{Extra: []byte{1, 2, 3}}).SetQKCSeal(nonce, mixHash); err == nil {
		t.Fatal("expected error for a header without a QuarkChain extra")
	}
}

// sealHeader builds a minimal QuarkChain shard header with the given seal and
// difficulty.
func sealHeader(nonce BlockNonce, mixHash common.Hash, difficulty int64) *Header {
	header := &Header{
		Number: big.NewInt(1),
		Extra:  (&QKCExtra{ShardID: 3, Nonce: nonce, MixHash: mixHash}).Encode(),
	}
	header.SetQKCDifficulty(big.NewInt(difficulty))
	return header
}

// TestQKCSealHashSealMatrix checks the seal hash is invariant to the seal while
// the block hash is not, for every zero / non-zero combination of the seal.
func TestQKCSealHashSealMatrix(t *testing.T) {
	zeroHash := common.Hash{}
	mix := common.BytesToHash(bytes.Repeat([]byte{0xcc}, 32))

	cases := []struct {
		name  string
		nonce BlockNonce
		mix   common.Hash
	}{
		{"zero seal", BlockNonce{}, zeroHash},
		{"nonce only", EncodeNonce(1), zeroHash},
		{"mixhash only", BlockNonce{}, mix},
		{"both set", EncodeNonce(1), mix},
	}
	var sealHashes []common.Hash
	var blockHashes []common.Hash
	for _, c := range cases {
		h := sealHeader(c.nonce, c.mix, 1000)
		sealHashes = append(sealHashes, h.QKCSealHash())
		blockHashes = append(blockHashes, h.Hash())
	}
	for i := 1; i < len(cases); i++ {
		if sealHashes[i] != sealHashes[0] {
			t.Errorf("%s: seal hash depends on the seal: have %x, want %x", cases[i].name, sealHashes[i], sealHashes[0])
		}
	}
	for i := range blockHashes {
		for j := i + 1; j < len(blockHashes); j++ {
			if blockHashes[i] == blockHashes[j] {
				t.Errorf("%s and %s: block hash ignores the seal", cases[i].name, cases[j].name)
			}
		}
	}
}

// TestQKCSealHashSensitivity checks the seal hash commits to the difficulty and
// to every other consensus field.
func TestQKCSealHashSensitivity(t *testing.T) {
	base := sealHeader(EncodeNonce(1), common.BytesToHash(bytes.Repeat([]byte{0xcc}, 32)), 1000)
	baseSeal := base.QKCSealHash()
	baseBlock := base.Hash()

	// The PoW difficulty is committed by the seal hash, in place of Difficulty.
	harder := sealHeader(EncodeNonce(1), common.BytesToHash(bytes.Repeat([]byte{0xcc}, 32)), 2000)
	if harder.QKCSealHash() == baseSeal {
		t.Error("seal hash ignores the difficulty")
	}
	if harder.Hash() == baseBlock {
		t.Error("block hash ignores the difficulty")
	}
	// The non-seal part of the QuarkChain extra is committed too.
	otherShard := &Header{
		Number: big.NewInt(1),
		Extra:  (&QKCExtra{ShardID: 4, Nonce: EncodeNonce(1), MixHash: common.BytesToHash(bytes.Repeat([]byte{0xcc}, 32))}).Encode(),
	}
	otherShard.SetQKCDifficulty(big.NewInt(1000))
	if otherShard.QKCSealHash() == baseSeal {
		t.Error("seal hash ignores the QuarkChain extra")
	}
	// And so are the plain geth consensus fields.
	mutations := map[string]func(*Header){
		"number":     func(h *Header) { h.Number = big.NewInt(2) },
		"time":       func(h *Header) { h.Time = 1 },
		"gasLimit":   func(h *Header) { h.GasLimit = 1 },
		"gasUsed":    func(h *Header) { h.GasUsed = 1 },
		"coinbase":   func(h *Header) { h.Coinbase = common.BytesToAddress([]byte{1}) },
		"root":       func(h *Header) { h.Root = common.BytesToHash([]byte{1}) },
		"txhash":     func(h *Header) { h.TxHash = common.BytesToHash([]byte{1}) },
		"parenthash": func(h *Header) { h.ParentHash = common.BytesToHash([]byte{1}) },
		"unclehash":  func(h *Header) { h.UncleHash = common.BytesToHash([]byte{1}) },
		"bloom":      func(h *Header) { h.Bloom[0] = 1 },
	}
	for name, mutate := range mutations {
		h := sealHeader(EncodeNonce(1), common.BytesToHash(bytes.Repeat([]byte{0xcc}, 32)), 1000)
		mutate(h)
		if h.QKCSealHash() == baseSeal {
			t.Errorf("seal hash ignores the %s field", name)
		}
		if h.Hash() == baseBlock {
			t.Errorf("block hash ignores the %s field", name)
		}
	}
}

// TestQKCSealHashDoesNotMutate checks computing the seal hash never modifies the
// header it is called on, including on a header that already carries a seal.
func TestQKCSealHashDoesNotMutate(t *testing.T) {
	for _, h := range []*Header{
		qkcReferenceHeader(),
		sealHeader(EncodeNonce(9), common.BytesToHash(bytes.Repeat([]byte{0xee}, 32)), 12345),
	} {
		before := CopyHeader(h)
		extraBefore := common.CopyBytes(h.Extra)
		_ = h.QKCSealHash()
		if !bytes.Equal(h.Extra, extraBefore) {
			t.Fatalf("seal hash mutated the extra data:\nhave %x\nwant %x", h.Extra, extraBefore)
		}
		if h.Hash() != before.Hash() {
			t.Fatal("seal hash mutated the header")
		}
	}
}

// TestQKCSealHashNonQKCHeader pins the fallback: a header whose extra data is
// not a QuarkChain extra is hashed unchanged. Rejecting such a header is the
// job of DecodeQKCExtra and the consensus engine, not of the hashing helper.
func TestQKCSealHashNonQKCHeader(t *testing.T) {
	for _, extra := range [][]byte{nil, {}, {1, 2, 3}, make([]byte, QKCExtraSize-1)} {
		plain := &Header{Number: big.NewInt(1), Difficulty: common.Big0, Extra: extra}
		if plain.QKCSealHash() != plain.Hash() {
			t.Fatalf("seal hash changed a header without a QuarkChain extra: %x", extra)
		}
	}
}

// qkcSealHashReference is the reference formula for the seal hash, computed from
// an explicit header field list instead of Header.EncodeRLP/Hash:
//
//	keccak256(RLP(ParentHash, UncleHash, Coinbase, Root, TxHash, ReceiptHash,
//	              Bloom, Difficulty, Number, GasLimit, GasUsed, Time,
//	              extra[seal zeroed], MixDigest, Nonce))
//
// Using it means the seal hash test does not depend on the same code path it is
// meant to verify.
func qkcSealHashReference(h *Header) (common.Hash, error) {
	extra := common.CopyBytes(h.Extra)
	if len(extra) == QKCExtraSize {
		for i := qkcExtraNonceOffset; i < QKCExtraSize; i++ {
			extra[i] = 0
		}
	}
	enc := []interface{}{
		h.ParentHash,
		h.UncleHash,
		h.Coinbase,
		h.Root,
		h.TxHash,
		h.ReceiptHash,
		h.Bloom,
		h.Difficulty,
		h.Number,
		h.GasLimit,
		h.GasUsed,
		h.Time,
		extra,
		h.MixDigest,
		h.Nonce,
	}
	var buf bytes.Buffer
	if err := rlp.Encode(&buf, enc); err != nil {
		return common.Hash{}, err
	}
	sum := crypto.NewKeccakState()
	if _, err := sum.Write(buf.Bytes()); err != nil {
		return common.Hash{}, err
	}
	var out common.Hash
	if _, err := sum.Read(out[:]); err != nil {
		return common.Hash{}, err
	}
	return out, nil
}

func TestQKCSealHashReference(t *testing.T) {
	cases := []*Header{
		qkcReferenceHeader(),
		sealHeader(EncodeNonce(0xdeadbeef), common.BytesToHash(bytes.Repeat([]byte{0xcc}, 32)), 1000),
		sealHeader(BlockNonce{}, common.Hash{}, 0),
		func() *Header {
			h := sealHeader(EncodeNonce(1), common.Hash{}, 1000)
			h.ParentHash = common.BytesToHash(bytes.Repeat([]byte{0x11}, 32))
			h.Time = 42
			return h
		}(),
	}
	for i, h := range cases {
		want, err := qkcSealHashReference(h)
		if err != nil {
			t.Fatalf("case %d: reference failed: %v", i, err)
		}
		if have := h.QKCSealHash(); have != want {
			t.Fatalf("case %d: seal hash mismatch:\nhave %x\nwant %x", i, have, want)
		}
		// Block hash must differ from the seal hash whenever the seal is set.
		if extra, err := h.QKCExtra(); err == nil {
			sealed := extra.Nonce != EncodeNonce(0) || extra.MixHash != (common.Hash{})
			if sealed && h.Hash() == want {
				t.Fatalf("case %d: block hash equals the seal hash despite a non-zero seal", i)
			}
		}
	}
}

// qkcReferenceHeader is a fully specified header used by the golden vector test.
func qkcReferenceHeader() *Header {
	h := &Header{
		ParentHash:  common.BytesToHash(bytes.Repeat([]byte{0x11}, 32)),
		UncleHash:   common.BytesToHash(bytes.Repeat([]byte{0x22}, 32)),
		Coinbase:    common.BytesToAddress(bytes.Repeat([]byte{0x33}, 20)),
		Root:        common.BytesToHash(bytes.Repeat([]byte{0x44}, 32)),
		TxHash:      common.BytesToHash(bytes.Repeat([]byte{0x55}, 32)),
		ReceiptHash: common.BytesToHash(bytes.Repeat([]byte{0x66}, 32)),
		Bloom:       BytesToBloom(bytes.Repeat([]byte{0x77}, 256)),
		Number:      big.NewInt(12345),
		GasLimit:    30_000_000,
		GasUsed:     21_000,
		Time:        1_600_000_000,
		Extra: (&QKCExtra{
			Version:        1,
			ShardID:        0x00010003,
			PrevRootBlock:  common.BytesToHash(bytes.Repeat([]byte{0x88}, 32)),
			Cursor:         XShardCursor{RootBlockHeight: 1, MinorBlockIndex: 2, XShardDepositIndex: 3},
			XShardGasUsed:  21_000,
			XShardGasLimit: 1_000_000,
		}).Encode(),
	}
	h.SetQKCDifficulty(big.NewInt(1000))
	return h
}

// TestQKCBlockHashGolden pins the exact bytes of the block hash and the seal
// hash of a fully specified header, so that any change to the field coverage,
// the ordering or the encoding is caught.
func TestQKCBlockHashGolden(t *testing.T) {
	const golden = "0xd6e3701b3d314251b417461c44a0e45575965606f523aff1309279f0ebff770b"
	h := qkcReferenceHeader()
	if have := h.Hash(); have != common.HexToHash(golden) {
		t.Fatalf("block hash mismatch:\nhave %x\nwant %s", have, golden)
	}
	// This header carries a zero seal, so both hashes coincide.
	if have := h.QKCSealHash(); have != common.HexToHash(golden) {
		t.Fatalf("seal hash mismatch:\nhave %x\nwant %s", have, golden)
	}
	// Writing a seal must leave the seal hash alone.
	sealed := CopyHeader(h)
	if err := sealed.SetQKCSeal(EncodeNonce(0x0102030405060708), common.BytesToHash(bytes.Repeat([]byte{0xcc}, 32))); err != nil {
		t.Fatalf("failed to set seal: %v", err)
	}
	if have := sealed.QKCSealHash(); have != common.HexToHash(golden) {
		t.Fatalf("seal hash changed after sealing:\nhave %x\nwant %s", have, golden)
	}
	if sealed.Hash() == common.HexToHash(golden) {
		t.Fatal("block hash unchanged after sealing")
	}
}
