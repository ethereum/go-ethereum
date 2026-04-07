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

package pathdb

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/bintrie"
)

// newTestBintrieCodec constructs a bintrieFlatCodec backed by an
// in-memory key-value store. Returns both the codec and the underlying
// store so tests can drive it directly.
func newTestBintrieCodec(t *testing.T) (*bintrieFlatCodec, ethdb.Database) {
	t.Helper()
	db := rawdb.NewMemoryDatabase()
	codec := newBintrieFlatCodec(db)
	return codec, db
}

// flushBatch commits a batch built against a memory database. Called
// after each codec write because the in-memory RMW of applyWrites reads
// from the store, not the batch.
func flushBatch(t *testing.T, batch interface{ Write() error }) {
	t.Helper()
	if err := batch.Write(); err != nil {
		t.Fatalf("batch write: %v", err)
	}
}

// TestBintrieCodecAccountRoundTrip verifies that an account written via
// WriteAccount (a two-slot BasicData||CodeHash blob) is persisted under
// the account's stem and can be read back by extracting the relevant
// offsets from the stem blob.
func TestBintrieCodecAccountRoundTrip(t *testing.T) {
	codec, db := newTestBintrieCodec(t)
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")

	basicData := bytes.Repeat([]byte{0xAB}, stemBlobValueSize)
	codeHash := bytes.Repeat([]byte{0xCD}, stemBlobValueSize)
	blob := append(append([]byte{}, basicData...), codeHash...)

	batch := db.NewBatch()
	codec.WriteAccount(batch, codec.AccountKey(addr), blob)
	flushBatch(t, batch)

	// Read back via ReadAccount — returns the raw stem blob, not the
	// decoded account. Extract offsets 0 and 1 manually.
	got := codec.ReadAccount(db, codec.AccountKey(addr))
	if len(got) == 0 {
		t.Fatal("ReadAccount returned empty for just-written account")
	}
	gotBasic, err := extractStemOffset(got, bintrie.BasicDataLeafKey)
	if err != nil || !bytes.Equal(gotBasic, basicData) {
		t.Fatalf("BasicData extract: got %x err=%v, want %x", gotBasic, err, basicData)
	}
	gotCode, err := extractStemOffset(got, bintrie.CodeHashLeafKey)
	if err != nil || !bytes.Equal(gotCode, codeHash) {
		t.Fatalf("CodeHash extract: got %x err=%v, want %x", gotCode, err, codeHash)
	}
}

// TestBintrieCodecStorageRoundTrip verifies that a storage slot written
// via WriteStorage is persisted at the correct stem+offset and can be
// read back via ReadStorage (which does offset extraction internally).
func TestBintrieCodecStorageRoundTrip(t *testing.T) {
	codec, db := newTestBintrieCodec(t)
	addr := common.HexToAddress("0x2222222222222222222222222222222222222222")
	slot := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000042")
	value := bytes.Repeat([]byte{0x77}, stemBlobValueSize)

	acctKey, storageKey := codec.StorageKey(addr, slot)
	batch := db.NewBatch()
	codec.WriteStorage(batch, acctKey, storageKey, value)
	flushBatch(t, batch)

	got := codec.ReadStorage(db, acctKey, storageKey)
	if !bytes.Equal(got, value) {
		t.Fatalf("ReadStorage: got %x, want %x", got, value)
	}
}

// TestBintrieCodecMultipleWritesSameStem verifies that two successive
// writes to DIFFERENT offsets at the same stem both persist — this is
// the common case when an account is updated (BasicData + CodeHash at
// stem X) and then a header storage slot at the same stem is written.
//
// Note: because the codec reads RMW from the store (not the batch), the
// caller must flush the batch between writes to the same stem for this
// to work correctly. This test exercises that pattern to ensure the
// per-call contract holds.
func TestBintrieCodecMultipleWritesSameStem(t *testing.T) {
	codec, db := newTestBintrieCodec(t)
	addr := common.HexToAddress("0x3333333333333333333333333333333333333333")

	// Write the account (offsets 0 and 1 at the BasicData stem).
	basicData := bytes.Repeat([]byte{0xAA}, stemBlobValueSize)
	codeHash := bytes.Repeat([]byte{0xBB}, stemBlobValueSize)
	blob := append(append([]byte{}, basicData...), codeHash...)
	batch := db.NewBatch()
	codec.WriteAccount(batch, codec.AccountKey(addr), blob)
	flushBatch(t, batch)

	// Now write a header storage slot. Slot 0 (per EIP-7864) lives at
	// offset 64 within the SAME stem as BasicData, so this is a
	// read-modify-write on the existing stem blob.
	slot := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
	storageValue := bytes.Repeat([]byte{0xCC}, stemBlobValueSize)
	acctKey, storageKey := codec.StorageKey(addr, slot)
	batch = db.NewBatch()
	codec.WriteStorage(batch, acctKey, storageKey, storageValue)
	flushBatch(t, batch)

	// All three offsets should now be readable.
	accountBlob := codec.ReadAccount(db, codec.AccountKey(addr))
	gotBasic, _ := extractStemOffset(accountBlob, bintrie.BasicDataLeafKey)
	if !bytes.Equal(gotBasic, basicData) {
		t.Fatalf("BasicData lost after storage write: got %x, want %x", gotBasic, basicData)
	}
	gotCode, _ := extractStemOffset(accountBlob, bintrie.CodeHashLeafKey)
	if !bytes.Equal(gotCode, codeHash) {
		t.Fatalf("CodeHash lost after storage write: got %x, want %x", gotCode, codeHash)
	}
	gotStorage := codec.ReadStorage(db, acctKey, storageKey)
	if !bytes.Equal(gotStorage, storageValue) {
		t.Fatalf("Storage: got %x, want %x", gotStorage, storageValue)
	}
}

// TestBintrieCodecDeleteAccount verifies that DeleteAccount clears only
// offsets 0 (BasicData) and 1 (CodeHash) at the account's stem, leaving
// any other offsets (e.g. header storage slots) at the same stem
// untouched. This mirrors BinaryTrie.DeleteAccount's intended semantics.
func TestBintrieCodecDeleteAccount(t *testing.T) {
	codec, db := newTestBintrieCodec(t)
	addr := common.HexToAddress("0x4444444444444444444444444444444444444444")

	// Populate account (offsets 0+1) and one header storage slot (offset 64).
	basicData := bytes.Repeat([]byte{0xAA}, stemBlobValueSize)
	codeHash := bytes.Repeat([]byte{0xBB}, stemBlobValueSize)
	batch := db.NewBatch()
	codec.WriteAccount(batch, codec.AccountKey(addr), append(basicData, codeHash...))
	flushBatch(t, batch)

	storageValue := bytes.Repeat([]byte{0xCC}, stemBlobValueSize)
	slot := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
	acctKey, storageKey := codec.StorageKey(addr, slot)
	batch = db.NewBatch()
	codec.WriteStorage(batch, acctKey, storageKey, storageValue)
	flushBatch(t, batch)

	// Delete the account. Offsets 0 and 1 should be cleared; the
	// header storage slot at offset 64 should survive.
	batch = db.NewBatch()
	codec.DeleteAccount(batch, codec.AccountKey(addr))
	flushBatch(t, batch)

	accountBlob := codec.ReadAccount(db, codec.AccountKey(addr))
	if len(accountBlob) == 0 {
		t.Fatal("stem blob was fully deleted; header storage should still be present")
	}
	if got, _ := extractStemOffset(accountBlob, bintrie.BasicDataLeafKey); got != nil {
		t.Fatalf("BasicData not cleared: %x", got)
	}
	if got, _ := extractStemOffset(accountBlob, bintrie.CodeHashLeafKey); got != nil {
		t.Fatalf("CodeHash not cleared: %x", got)
	}
	if got := codec.ReadStorage(db, acctKey, storageKey); !bytes.Equal(got, storageValue) {
		t.Fatalf("header storage lost after DeleteAccount: got %x, want %x", got, storageValue)
	}
}

// TestBintrieCodecDeleteLastOffsetRemovesKey verifies that when the
// final populated offset at a stem is cleared, the on-disk key is
// removed entirely (zero-length blobs are never persisted).
func TestBintrieCodecDeleteLastOffsetRemovesKey(t *testing.T) {
	codec, db := newTestBintrieCodec(t)
	addr := common.HexToAddress("0x5555555555555555555555555555555555555555")
	slot := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000080")
	value := bytes.Repeat([]byte{0xDD}, stemBlobValueSize)

	acctKey, storageKey := codec.StorageKey(addr, slot)

	// Write, verify, delete, verify absent.
	batch := db.NewBatch()
	codec.WriteStorage(batch, acctKey, storageKey, value)
	flushBatch(t, batch)

	if got := codec.ReadStorage(db, acctKey, storageKey); !bytes.Equal(got, value) {
		t.Fatalf("pre-delete read: got %x, want %x", got, value)
	}

	batch = db.NewBatch()
	codec.DeleteStorage(batch, acctKey, storageKey)
	flushBatch(t, batch)

	// The raw key should be gone from the store.
	raw := rawdb.ReadBinTrieStem(db, stemFromKey(storageKey))
	if raw != nil {
		t.Fatalf("stem blob should be deleted, got %x", raw)
	}
	// And ReadStorage returns nil.
	if got := codec.ReadStorage(db, acctKey, storageKey); got != nil {
		t.Fatalf("post-delete read: got %x, want nil", got)
	}
}

// TestBintrieCodecCacheKeysDisjoint verifies that the bintrie cache key
// prefix keeps it disjoint from merkle account keys. This is the
// collision check that Agent 2 flagged in the review.
func TestBintrieCodecCacheKeysDisjoint(t *testing.T) {
	codec := &bintrieFlatCodec{}
	merkle := &merkleFlatCodec{}

	// A 32-byte hash that, when passed to both codecs, would collide
	// if the bintrie codec didn't prefix-disambiguate its cache keys.
	hash := common.HexToHash("0xaabbccddeeff00112233445566778899aabbccddeeff00112233445566778899")

	binKey := codec.AccountCacheKey(hash)
	merkleKey := merkle.AccountCacheKey(hash)

	if bytes.Equal(binKey, merkleKey) {
		t.Fatalf("bintrie and merkle cache keys collided: both are %x", binKey)
	}
	if binKey[0] != bintrieCacheKeyPrefix {
		t.Fatalf("bintrie cache key missing prefix byte: %x", binKey)
	}
}

// TestBintrieCodecSplitMarker verifies the single-tier marker handling.
// For merkle the marker is a two-tier (account, account+storage) pair;
// for bintrie it's a single 32-byte stem key, so SplitMarker returns
// the same slice twice.
func TestBintrieCodecSplitMarker(t *testing.T) {
	codec := &bintrieFlatCodec{}

	// Nil marker.
	acc, full := codec.SplitMarker(nil)
	if acc != nil || full != nil {
		t.Fatalf("nil marker: acc=%v full=%v, want nil/nil", acc, full)
	}

	// A 32-byte marker. Both halves point to the same bytes.
	marker := bytes.Repeat([]byte{0xAA}, 32)
	acc, full = codec.SplitMarker(marker)
	if !bytes.Equal(acc, marker) || !bytes.Equal(full, marker) {
		t.Fatalf("SplitMarker: acc=%x full=%x, want both %x", acc, full, marker)
	}
}

// TestBintrieCodecFlushAggregates verifies the per-stem aggregation that
// the codec's Flush method performs. Two distinct offsets at the SAME stem
// should produce a single on-disk stem blob containing both offsets after
// one Flush call — proving the codec collapses what would have been N
// read-modify-writes into one.
//
// Three offsets are written across two stems (2 + 1) so we exercise both
// the multi-offset and single-offset paths in a single test.
func TestBintrieCodecFlushAggregates(t *testing.T) {
	codec, db := newTestBintrieCodec(t)

	// Build a per-offset accountData map mimicking what encodeBinary
	// produces from a binaryHasher.DrainStemWrites: the keys are full
	// 32-byte (stem || offset) tuples and the values are 32-byte leaves.
	addr := common.HexToAddress("0xCafeBabeDeadBeef00112233445566778899aabb")
	stem := bintrie.GetBinaryTreeKey(addr, make([]byte, 32))[:bintrie.StemSize]

	basicData := bytes.Repeat([]byte{0xAA}, stemBlobValueSize)
	codeHash := bytes.Repeat([]byte{0xBB}, stemBlobValueSize)
	storageVal := bytes.Repeat([]byte{0xCC}, stemBlobValueSize)
	otherStem := bytes.Repeat([]byte{0x42}, bintrie.StemSize)
	otherVal := bytes.Repeat([]byte{0xDD}, stemBlobValueSize)

	mkKey := func(stem []byte, offset byte) common.Hash {
		var k common.Hash
		copy(k[:bintrie.StemSize], stem)
		k[bintrie.StemSize] = offset
		return k
	}
	accountData := map[common.Hash][]byte{
		mkKey(stem, bintrie.BasicDataLeafKey):       basicData,
		mkKey(stem, bintrie.CodeHashLeafKey):        codeHash,
		mkKey(stem, 64):                             storageVal, // header storage slot
		mkKey(otherStem, bintrie.BasicDataLeafKey):  otherVal,
	}

	batch := db.NewBatch()
	accW, stoW := codec.Flush(batch, nil, accountData, nil, nil)
	flushBatch(t, batch)

	if accW != 4 {
		t.Errorf("account write count: got %d, want 4", accW)
	}
	if stoW != 0 {
		t.Errorf("storage write count: got %d, want 0 (no storage map)", stoW)
	}

	// All three offsets at `stem` should be readable from a single on-disk
	// blob; aggregation worked iff the second/third writes did not clobber
	// the first.
	blob := rawdb.ReadBinTrieStem(db, stem)
	if len(blob) == 0 {
		t.Fatal("stem blob missing after Flush")
	}
	for offset, want := range map[byte][]byte{
		bintrie.BasicDataLeafKey: basicData,
		bintrie.CodeHashLeafKey:  codeHash,
		64:                       storageVal,
	} {
		got, err := extractStemOffset(blob, offset)
		if err != nil {
			t.Fatalf("extract offset %d: %v", offset, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("offset %d: got %x, want %x", offset, got, want)
		}
	}

	// The other stem should also have its single offset.
	otherBlob := rawdb.ReadBinTrieStem(db, otherStem)
	if got, _ := extractStemOffset(otherBlob, bintrie.BasicDataLeafKey); !bytes.Equal(got, otherVal) {
		t.Errorf("other stem BasicData: got %x, want %x", got, otherVal)
	}
}

// TestBintrieCodecFlushDelete verifies that nil-valued entries in the
// accountData map clear the corresponding offset, and that clearing every
// populated offset at a stem removes the on-disk key entirely (matching
// the per-call DeleteStorage semantics tested elsewhere).
func TestBintrieCodecFlushDelete(t *testing.T) {
	codec, db := newTestBintrieCodec(t)

	// Seed: write two offsets at one stem.
	stem := bytes.Repeat([]byte{0x77}, bintrie.StemSize)
	v0 := bytes.Repeat([]byte{0x01}, stemBlobValueSize)
	v1 := bytes.Repeat([]byte{0x02}, stemBlobValueSize)

	mkKey := func(offset byte) common.Hash {
		var k common.Hash
		copy(k[:bintrie.StemSize], stem)
		k[bintrie.StemSize] = offset
		return k
	}
	batch := db.NewBatch()
	codec.Flush(batch, nil, map[common.Hash][]byte{
		mkKey(0): v0,
		mkKey(1): v1,
	}, nil, nil)
	flushBatch(t, batch)

	// Now flush a nil for offset 0 — only offset 1 should remain.
	batch = db.NewBatch()
	codec.Flush(batch, nil, map[common.Hash][]byte{mkKey(0): nil}, nil, nil)
	flushBatch(t, batch)

	blob := rawdb.ReadBinTrieStem(db, stem)
	if got, _ := extractStemOffset(blob, 0); got != nil {
		t.Errorf("offset 0 should be cleared, got %x", got)
	}
	if got, _ := extractStemOffset(blob, 1); !bytes.Equal(got, v1) {
		t.Errorf("offset 1 should survive, got %x want %x", got, v1)
	}

	// Clear the last remaining offset; the on-disk key should disappear.
	batch = db.NewBatch()
	codec.Flush(batch, nil, map[common.Hash][]byte{mkKey(1): nil}, nil, nil)
	flushBatch(t, batch)

	if raw := rawdb.ReadBinTrieStem(db, stem); raw != nil {
		t.Errorf("stem should be deleted, got %x", raw)
	}
}
