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
// reference used instead is the explicit RLP field list in qkcSealHashReference,
// which does not go through Header.EncodeRLP/Hash.
//
// Headers are built the way the design prescribes, through qkcShardHeader: the
// QKC fields are assembled as a QKCExtra and encoded once into Extra, the PoW
// difficulty is put in MixDigest, and the header's own Difficulty and Nonce are
// left at zero. There is no setter for any of those header fields, by design.

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
		"00000001" + // version, offset 0
		"00010003" + // shardId, offset 4
		strings.Repeat("aa", 32) + // prevRootBlock, offset 8
		"0000000000000001" + "0000000000000002" + "0000000000000003" + // cursor, offset 40
		"0000000000005208" + // xShardGasUsed, offset 64
		"00000000000f4240" + // xShardGasLimit, offset 72
		"0102030405060708" + // nonce, offset 80, start of the PoW seal
		strings.Repeat("bb", 32)) // mixHash, offset 88

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

// TestQKCExtraRoundTrip checks that every field survives a round trip and that
// the encoding is always exactly QKCExtraSize bytes long, including for the
// extreme value of every field.
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
		enc := want.Encode()
		if len(enc) != QKCExtraSize {
			t.Fatalf("case %d: encoded to %d bytes, want %d", i, len(enc), QKCExtraSize)
		}
		have, err := DecodeQKCExtra(enc)
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

// TestQKCDifficulty checks that the PoW difficulty is carried by MixDigest and
// not by the geth Difficulty field, which a shard block has to leave at zero.
func TestQKCDifficulty(t *testing.T) {
	difficulties := []*big.Int{
		nil,
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(1_000_000),
		new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)),
	}
	for _, difficulty := range difficulties {
		want := new(big.Int)
		if difficulty != nil {
			want = difficulty
		}
		h := qkcShardHeader(qkcBaseExtra(EncodeNonce(1), common.Hash{}), difficulty)

		if have := h.QKCDifficulty(); have.Cmp(want) != 0 {
			t.Errorf("difficulty %v: have %v", want, have)
		}
		if wantHash := common.BigToHash(want); h.MixDigest != wantHash {
			t.Errorf("difficulty %v: MixDigest is %x, want %x", want, h.MixDigest, wantHash)
		}
		// The two geth fields a post-Merge block has to zero stay untouched.
		if h.Difficulty == nil || h.Difficulty.Sign() != 0 {
			t.Errorf("difficulty %v: header Difficulty is %v, want 0", want, h.Difficulty)
		}
		if h.Nonce != (BlockNonce{}) {
			t.Errorf("difficulty %v: header Nonce is %x, want zero", want, h.Nonce)
		}
	}
}

// TestQKCSealHashSealMatrix checks that the seal hash is invariant to the seal
// while the block hash is not, for every zero / non-zero combination of the two
// seal fields.
func TestQKCSealHashSealMatrix(t *testing.T) {
	mix := common.BytesToHash(bytes.Repeat([]byte{0xcc}, 32))
	cases := []struct {
		name  string
		nonce BlockNonce
		mix   common.Hash
	}{
		{"zero seal", BlockNonce{}, common.Hash{}},
		{"nonce only", EncodeNonce(1), common.Hash{}},
		{"mixhash only", BlockNonce{}, mix},
		{"both set", EncodeNonce(1), mix},
	}
	sealHashes := make([]common.Hash, len(cases))
	blockHashes := make([]common.Hash, len(cases))
	for i, c := range cases {
		h := qkcShardHeader(qkcBaseExtra(c.nonce, c.mix), big.NewInt(1000))
		sealHashes[i] = h.QKCSealHash()
		blockHashes[i] = h.Hash()
	}
	// The zero seal is hashed as it stands, so the two hashes coincide for it.
	if blockHashes[0] != sealHashes[0] {
		t.Errorf("%s: block hash and seal hash differ", cases[0].name)
	}
	for i := 1; i < len(cases); i++ {
		if sealHashes[i] != sealHashes[0] {
			t.Errorf("%s: seal hash depends on the seal:\nhave %x\nwant %x", cases[i].name, sealHashes[i], sealHashes[0])
		}
		if blockHashes[i] == blockHashes[0] {
			t.Errorf("%s: block hash ignores the seal", cases[i].name)
		}
	}
	// The three sealed cases must also be distinguishable from each other.
	for i := 1; i < len(blockHashes); i++ {
		for j := i + 1; j < len(blockHashes); j++ {
			if blockHashes[i] == blockHashes[j] {
				t.Errorf("%s and %s: block hash does not distinguish the seals", cases[i].name, cases[j].name)
			}
		}
	}
}

// TestQKCSealHashSensitivity checks that the seal hash commits to the PoW
// difficulty and to every other field that has to take part in it.
func TestQKCSealHashSensitivity(t *testing.T) {
	nonce := EncodeNonce(1)
	mix := common.BytesToHash(bytes.Repeat([]byte{0xcc}, 32))
	base := qkcShardHeader(qkcBaseExtra(nonce, mix), big.NewInt(1000))
	baseSeal := base.QKCSealHash()
	baseBlock := base.Hash()

	// The PoW difficulty participates through MixDigest, in place of Difficulty.
	harder := qkcShardHeader(qkcBaseExtra(nonce, mix), big.NewInt(2000))
	if harder.QKCSealHash() == baseSeal {
		t.Error("seal hash ignores the difficulty")
	}
	if harder.Hash() == baseBlock {
		t.Error("block hash ignores the difficulty")
	}
	// The QKC fields outside the seal region participate as well.
	other := qkcBaseExtra(nonce, mix)
	other.ShardID = 4
	otherShard := qkcShardHeader(other, big.NewInt(1000))
	if otherShard.QKCSealHash() == baseSeal {
		t.Error("seal hash ignores the QuarkChain fields outside the seal")
	}
	if otherShard.Hash() == baseBlock {
		t.Error("block hash ignores the QuarkChain fields outside the seal")
	}
	// And so do the plain geth consensus fields.
	mutations := map[string]func(*Header){
		"number":      func(h *Header) { h.Number = big.NewInt(2) },
		"time":        func(h *Header) { h.Time = 1 },
		"gasLimit":    func(h *Header) { h.GasLimit = 1 },
		"gasUsed":     func(h *Header) { h.GasUsed = 1 },
		"coinbase":    func(h *Header) { h.Coinbase = common.BytesToAddress([]byte{1}) },
		"root":        func(h *Header) { h.Root = common.BytesToHash([]byte{1}) },
		"txHash":      func(h *Header) { h.TxHash = common.BytesToHash([]byte{1}) },
		"receiptHash": func(h *Header) { h.ReceiptHash = common.BytesToHash([]byte{1}) },
		"parentHash":  func(h *Header) { h.ParentHash = common.BytesToHash([]byte{1}) },
		"uncleHash":   func(h *Header) { h.UncleHash = common.BytesToHash([]byte{1}) },
		"bloom":       func(h *Header) { h.Bloom[0] = 1 },
	}
	for name, mutate := range mutations {
		h := CopyHeader(base)
		mutate(h)
		if h.QKCSealHash() == baseSeal {
			t.Errorf("seal hash ignores the %s field", name)
		}
		if h.Hash() == baseBlock {
			t.Errorf("block hash ignores the %s field", name)
		}
	}
}

// TestQKCSealHashDoesNotMutate checks that computing the seal hash never changes
// the header it is called on, whether that header carries a seal or not.
func TestQKCSealHashDoesNotMutate(t *testing.T) {
	headers := []*Header{
		qkcReferenceHeader(),
		qkcShardHeader(qkcBaseExtra(EncodeNonce(9), common.BytesToHash(bytes.Repeat([]byte{0xee}, 32))), big.NewInt(12345)),
	}
	for i, h := range headers {
		extraBefore := common.CopyBytes(h.Extra)
		hashBefore := h.Hash()
		_ = h.QKCSealHash()
		if !bytes.Equal(h.Extra, extraBefore) {
			t.Fatalf("case %d: seal hash mutated the extra data:\nhave %x\nwant %x", i, h.Extra, extraBefore)
		}
		if h.Hash() != hashBefore {
			t.Fatalf("case %d: seal hash mutated the header", i)
		}
	}
}

// TestQKCSealHashNonQKCHeader pins the fallback: a header whose extra data is
// not exactly a QuarkChain extra is hashed as it stands, however long it is.
// Rejecting such a header is the job of DecodeQKCExtra and of the consensus
// engine, not of the hashing helper.
func TestQKCSealHashNonQKCHeader(t *testing.T) {
	extras := [][]byte{
		nil,
		{},
		{1, 2, 3},
		bytes.Repeat([]byte{0xaa}, QKCExtraSize-1),
		bytes.Repeat([]byte{0xaa}, QKCExtraSize+1),
	}
	for _, extra := range extras {
		plain := &Header{Number: big.NewInt(1), Difficulty: common.Big0, Extra: extra}
		if plain.QKCSealHash() != plain.Hash() {
			t.Errorf("seal hash changed a header without a QuarkChain extra: %x", extra)
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
	mix := common.BytesToHash(bytes.Repeat([]byte{0xcc}, 32))

	// A header that also carries geth state, not just the QKC fields.
	state := qkcShardHeader(qkcBaseExtra(EncodeNonce(1), common.Hash{}), big.NewInt(1000))
	state.ParentHash = common.BytesToHash(bytes.Repeat([]byte{0x11}, 32))
	state.Time = 42

	cases := []*Header{
		qkcReferenceHeader(),
		qkcShardHeader(qkcBaseExtra(EncodeNonce(0xdeadbeef), mix), big.NewInt(1000)),
		qkcShardHeader(qkcBaseExtra(BlockNonce{}, common.Hash{}), nil),
		state,
	}
	for i, h := range cases {
		want, err := qkcSealHashReference(h)
		if err != nil {
			t.Fatalf("case %d: reference failed: %v", i, err)
		}
		if have := h.QKCSealHash(); have != want {
			t.Fatalf("case %d: seal hash mismatch:\nhave %x\nwant %x", i, have, want)
		}
	}
}

// qkcBaseExtra returns a QKC extra with the given seal, plus a version and a
// shard id so that the region in front of the seal is non-zero too.
func qkcBaseExtra(nonce BlockNonce, mixHash common.Hash) *QKCExtra {
	return &QKCExtra{
		Version: 1,
		ShardID: 3,
		Nonce:   nonce,
		MixHash: mixHash,
	}
}

// qkcShardHeader builds a shard block header the way the design prescribes:
// assemble the QKC fields as a QKCExtra, encode it once into Extra, then
// initialise the header. The PoW difficulty goes into MixDigest; the header's
// own Difficulty and Nonce are left at zero, as a post-Merge block requires. A
// nil difficulty means no work, i.e. a zero difficulty.
func qkcShardHeader(fields *QKCExtra, difficulty *big.Int) *Header {
	h := &Header{
		Number:     big.NewInt(1),
		Difficulty: new(big.Int),
		Extra:      fields.Encode(),
	}
	if difficulty != nil {
		h.MixDigest = common.BigToHash(difficulty)
	}
	return h
}

// qkcReferenceHeader is the fully specified shard header used by the golden
// vector test. It carries a zero seal, so its block hash and seal hash coincide.
func qkcReferenceHeader() *Header {
	return &Header{
		ParentHash:  common.BytesToHash(bytes.Repeat([]byte{0x11}, 32)),
		UncleHash:   common.BytesToHash(bytes.Repeat([]byte{0x22}, 32)),
		Coinbase:    common.BytesToAddress(bytes.Repeat([]byte{0x33}, 20)),
		Root:        common.BytesToHash(bytes.Repeat([]byte{0x44}, 32)),
		TxHash:      common.BytesToHash(bytes.Repeat([]byte{0x55}, 32)),
		ReceiptHash: common.BytesToHash(bytes.Repeat([]byte{0x66}, 32)),
		Bloom:       BytesToBloom(bytes.Repeat([]byte{0x77}, 256)),
		Difficulty:  new(big.Int),
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
		MixDigest: common.BigToHash(big.NewInt(1000)),
	}
}

// TestQKCBlockHashGolden freezes the exact bytes of the block hash and the seal
// hash of a fully specified header, so that any change to the field coverage,
// the ordering or the encoding is caught. What checks that the frozen value is
// the correct one is the independent reference formula in
// TestQKCSealHashReference; this test only checks that it does not move.
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
	// Writing a seal must leave the seal hash alone and move the block hash.
	sealed := CopyHeader(h)
	fields, err := sealed.QKCExtra()
	if err != nil {
		t.Fatalf("failed to decode the extra data: %v", err)
	}
	fields.Nonce = EncodeNonce(0x0102030405060708)
	fields.MixHash = common.BytesToHash(bytes.Repeat([]byte{0xcc}, 32))
	sealed.Extra = fields.Encode()

	if have := sealed.QKCSealHash(); have != common.HexToHash(golden) {
		t.Fatalf("seal hash changed after sealing:\nhave %x\nwant %s", have, golden)
	}
	if sealed.Hash() == common.HexToHash(golden) {
		t.Fatal("block hash unchanged after sealing")
	}
}
