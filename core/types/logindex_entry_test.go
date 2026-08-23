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
	"encoding/hex"
	"strings"
	"testing"
)

// TestEncodeEntry pins the exact wire layout of every entry type: 2-byte
// big-endian type id, content (32 bytes, 20 for addresses), then the
// position fields. Non-trivial values prove the big-endian byte order.
func TestEncodeEntry(t *testing.T) {
	var (
		blockNum = uint64(0x0102030405060708)
		txIdx    = uint32(0x01020304)
		logIdx   = uint32(0x0a0b0c0d)
		hashCtx  = bytes.Repeat([]byte{0x11}, 32)
		addrCtx  = bytes.Repeat([]byte{0x22}, 20)
		hashHex  = strings.Repeat("11", 32)
		addrHex  = strings.Repeat("22", 20)
		tailHex  = "0102030405060708" + "01020304" + "0a0b0c0d"
	)
	tests := []struct {
		typ     EntryType
		content []byte
		want    string
	}{
		{EntryTypeBlock, hashCtx, "0000" + hashHex + "0102030405060708"},
		{EntryTypeTransaction, hashCtx, "0001" + hashHex + tailHex},
		{EntryTypeLogAddress, addrCtx, "0002" + addrHex + tailHex},
		{EntryTypeLogTopic0, hashCtx, "0003" + hashHex + tailHex},
		{EntryTypeLogTopic1, hashCtx, "0004" + hashHex + tailHex},
		{EntryTypeLogTopic2, hashCtx, "0005" + hashHex + tailHex},
		{EntryTypeLogTopic3, hashCtx, "0006" + hashHex + tailHex},
	}
	for _, tt := range tests {
		e := EncodeEntry(tt.typ, tt.content, blockNum, txIdx, logIdx)
		if got := hex.EncodeToString(e); got != tt.want {
			t.Errorf("type %d: encoded %s, want %s", tt.typ, got, tt.want)
		}
		if len(e) != entrySize(tt.typ) {
			t.Errorf("type %d: length %d, want %d", tt.typ, len(e), entrySize(tt.typ))
		}
	}
}

// TestEncodeEntryPanics asserts the encoder rejects wrong content sizes and
// unknown types instead of silently mis-encoding.
func TestEncodeEntryPanics(t *testing.T) {
	mustPanic := func(name string, f func()) {
		defer func() {
			if recover() == nil {
				t.Errorf("%s: expected panic", name)
			}
		}()
		f()
	}
	mustPanic("block with 20-byte content", func() { EncodeEntry(EntryTypeBlock, make([]byte, 20), 0, 0, 0) })
	mustPanic("address with 32-byte content", func() { EncodeEntry(EntryTypeLogAddress, make([]byte, 32), 0, 0, 0) })
	mustPanic("topic with 20-byte content", func() { EncodeEntry(EntryTypeLogTopic2, make([]byte, 20), 0, 0, 0) })
	mustPanic("unknown type", func() { EncodeEntry(EntryType(7), make([]byte, 32), 0, 0, 0) })
}

// TestEntryFields round-trips every entry type and asserts the decoder
// rejects malformed and unknown-type entries without panicking.
func TestEntryFields(t *testing.T) {
	var (
		blockNum = uint64(0x0102030405060708)
		txIdx    = uint32(0x01020304)
		logIdx   = uint32(0x0a0b0c0d)
	)
	tests := []EntryType{
		EntryTypeBlock, EntryTypeTransaction, EntryTypeLogAddress,
		EntryTypeLogTopic0, EntryTypeLogTopic1, EntryTypeLogTopic2, EntryTypeLogTopic3,
	}
	for _, typ := range tests {
		content := make([]byte, contentSize(typ))
		for i := range content {
			content[i] = byte(i)
		}
		e := EncodeEntry(typ, content, blockNum, txIdx, logIdx)
		gotTyp, gotContent, gotBlock, gotTx, gotLog, err := EntryFields(e)
		if err != nil {
			t.Fatalf("type %d: unexpected error: %v", typ, err)
		}
		// Block entries have no position tail; txIdx and logIdx are ignored
		// for them, so skip the tail comparison.
		wantTx, wantLog := txIdx, logIdx
		if typ == EntryTypeBlock {
			wantTx, wantLog = 0, 0
		}
		if gotTyp != typ || !bytes.Equal(gotContent, content) || gotBlock != blockNum || gotTx != wantTx || gotLog != wantLog {
			t.Errorf("type %d: round-trip mismatch: typ %d content %x block %d tx %d log %d", typ, gotTyp, gotContent, gotBlock, gotTx, gotLog)
		}
		// The decoded content must not alias the entry.
		gotContent[0] ^= 0xff
		if bytes.Equal(gotContent, content) {
			t.Errorf("type %d: decoded content aliases the entry", typ)
		}
	}

	// Malformed entries: too short, truncated, oversized, unknown type id.
	valid := EncodeEntry(EntryTypeTransaction, make([]byte, TxHashSize), 0, 0, 0)
	unknownID := EncodeEntry(EntryTypeLogTopic3, make([]byte, TopicSize), 0, 0, 0)
	unknownID[0], unknownID[1] = 0x00, 0x07
	bad := []IndexEntry{
		nil,
		valid[:1],
		valid[:len(valid)-1],
		append(IndexEntry{}, append(valid, 0)...),
		unknownID,
		{0xff, 0xff, 0x00},
	}
	for _, e := range bad {
		if _, _, _, _, _, err := EntryFields(e); err == nil {
			t.Errorf("entry %x: expected error", e)
		}
	}
}

// TestCompareEntries asserts the ordering implied by the spec's
// lexicographical sort: type id, then content, then block number, then tx
// index, then log index.
func TestCompareEntries(t *testing.T) {
	hashOf := func(v byte) []byte { return bytes.Repeat([]byte{v}, 32) }
	addrOf := func(v byte) []byte { return bytes.Repeat([]byte{v}, 20) }
	tests := []struct {
		name string
		a, b IndexEntry
		want int
	}{
		{"type ordering", EncodeEntry(EntryTypeBlock, hashOf(0), 0, 0, 0), EncodeEntry(EntryTypeTransaction, hashOf(0), 0, 0, 0), -1},
		{"type ordering 2", EncodeEntry(EntryTypeTransaction, hashOf(0), 0, 0, 0), EncodeEntry(EntryTypeLogAddress, addrOf(0), 0, 0, 0), -1},
		{"type ordering 3", EncodeEntry(EntryTypeLogAddress, addrOf(0), 0, 0, 0), EncodeEntry(EntryTypeLogTopic0, hashOf(0), 0, 0, 0), -1},
		{"topic ordering", EncodeEntry(EntryTypeLogTopic1, hashOf(0), 0, 0, 0), EncodeEntry(EntryTypeLogTopic2, hashOf(0), 0, 0, 0), -1},
		{"content ordering", EncodeEntry(EntryTypeBlock, hashOf(0), 0, 0, 0), EncodeEntry(EntryTypeBlock, hashOf(1), 0, 0, 0), -1},
		{"block ordering", EncodeEntry(EntryTypeBlock, hashOf(0), 1, 0, 0), EncodeEntry(EntryTypeBlock, hashOf(0), 2, 0, 0), -1},
		{"tx ordering", EncodeEntry(EntryTypeTransaction, hashOf(0), 0, 1, 0), EncodeEntry(EntryTypeTransaction, hashOf(0), 0, 2, 0), -1},
		{"log ordering", EncodeEntry(EntryTypeLogAddress, addrOf(0), 0, 0, 1), EncodeEntry(EntryTypeLogAddress, addrOf(0), 0, 0, 2), -1},
		{"equal", EncodeEntry(EntryTypeBlock, hashOf(0), 0, 0, 0), EncodeEntry(EntryTypeBlock, hashOf(0), 0, 0, 0), 0},
	}
	for _, tt := range tests {
		if got := CompareEntries(tt.a, tt.b); got != tt.want {
			t.Errorf("%s: CompareEntries = %d, want %d", tt.name, got, tt.want)
		}
		if got := CompareEntries(tt.b, tt.a); got != -tt.want {
			t.Errorf("%s: comparison not antisymmetric", tt.name)
		}
	}
}

// TestEntryTypeString covers the human-readable names used in logs.
func TestEntryTypeString(t *testing.T) {
	want := map[EntryType]string{
		EntryTypeBlock: "block", EntryTypeTransaction: "transaction",
		EntryTypeLogAddress: "log_address", EntryTypeLogTopic0: "log_topic0",
		EntryTypeLogTopic1: "log_topic1", EntryTypeLogTopic2: "log_topic2",
		EntryTypeLogTopic3: "log_topic3", EntryType(7): "unknown",
	}
	for typ, wantStr := range want {
		if got := EntryTypeString(typ); got != wantStr {
			t.Errorf("type %d: got %q, want %q", typ, got, wantStr)
		}
	}
}

// TestEntrySizes guards the derived totals against the spec's fixed sizes.
func TestEntrySizes(t *testing.T) {
	if BlockEntrySize != 42 || TransactionEntrySize != 50 || LogAddressEntrySize != 38 || LogTopicEntrySize != 50 {
		t.Errorf("derived sizes: block %d, tx %d, address %d, topic %d; want 42, 50, 38, 50", BlockEntrySize, TransactionEntrySize, LogAddressEntrySize, LogTopicEntrySize)
	}
}
