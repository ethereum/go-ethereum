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
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// Spec worked example (EIP-8304, pinned at
// ethereum/EIPs@a9b074cffb5bea63b201ee5fabf19553dbfe483a): blocks 40-43 of a
// chain whose 42-43 contain WETH/USDT transfers.
//
// Note on the spec's internal inconsistency: the chronological table lists the
// 42tx1 hash as 0xa75590c9... while the binary table encodes 0x75590c9c...
// This test pins the binary table (which is the authoritative encoding).
var (
	specHash39 = common.HexToHash("0xbf98e6cb26f6ff312586968d1f343a3d3c439a8c5c86233aff2a82f1a68263df")
	specHash40 = common.HexToHash("0x42f66a2e9f9c68e223e8d826145d7cfacb00520dba6a9555803121de29790b65")
	specHash41 = common.HexToHash("0x978ce0036b6d1c62d716045505587d15cc85a1def92f9f450937b6467295e517")
	specHash42 = common.HexToHash("0x66f42ef12b140e8004ad39a760191457f485204ee6c9f990c33f14014e521f20")

	specTx42_0 = common.HexToHash("0xca2d12d1b8132de09d0d668cc87349dc70134bee3010e03ddb2d83f7160bd6e3")
	specTx42_1 = common.HexToHash("0x75590c9ced72898d6207c917e11b452949043404a1802b893f7717a9c1c8f45d")
	specTx43_0 = common.HexToHash("0x7046035b326ab22a3142e4416ed4300a28a483a31a1e04cf62bec575b2e7cf09")

	specWETH = common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")
	specUSDT = common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7")

	specTransfer   = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	specWithdrawal = common.HexToHash("0x7fcf532c15f0a6db0bd6d0e038bea71d30d808c7d98cb3bf7268a95bf5081b65")
	specAlice      = common.HexToHash("0x0000000000000000000000004a5e9a3bf35df5e0ea4b4cd3289e0111f1deadf7")
	specBob        = common.HexToHash("0x000000000000000000000000f2a5fd4d5bb24b651d1198bc014941a16f6edde5")
	specCarol      = common.HexToHash("0x000000000000000000000000ac8cbe20039c2e9547da9107456179bd4f39a734")
)

// TestIndexBuilderSpecWorkedExample reproduces the spec's 22-entry worked
// example byte-for-byte: four single-block tables (blocks 40-43) whose entries
// are the lexicographically sorted binary encodings listed in the spec.
func TestIndexBuilderSpecWorkedExample(t *testing.T) {
	receipts42 := Receipts{
		{TxHash: specTx42_0, Logs: []*Log{
			{Address: specWETH, Topics: []common.Hash{specTransfer, specAlice, specBob}},
			{Address: specUSDT, Topics: []common.Hash{specTransfer, specBob, specAlice}},
		}},
		{TxHash: specTx42_1, Logs: []*Log{
			{Address: specWETH, Topics: []common.Hash{specWithdrawal, specBob}},
		}},
	}
	receipts43 := Receipts{
		{TxHash: specTx43_0, Logs: []*Log{
			{Address: specUSDT, Topics: []common.Hash{specTransfer, specAlice, specCarol}},
		}},
	}
	// The sorted binary table from the spec, verbatim (22 entries).
	want := []string{
		"000042f66a2e9f9c68e223e8d826145d7cfacb00520dba6a9555803121de29790b650000000000000028",
		"000066f42ef12b140e8004ad39a760191457f485204ee6c9f990c33f14014e521f20000000000000002a",
		"0000978ce0036b6d1c62d716045505587d15cc85a1def92f9f450937b6467295e5170000000000000029",
		"0000bf98e6cb26f6ff312586968d1f343a3d3c439a8c5c86233aff2a82f1a68263df0000000000000027",
		"00017046035b326ab22a3142e4416ed4300a28a483a31a1e04cf62bec575b2e7cf09000000000000002b0000000000000000",
		"000175590c9ced72898d6207c917e11b452949043404a1802b893f7717a9c1c8f45d000000000000002a0000000100000002",
		"0001ca2d12d1b8132de09d0d668cc87349dc70134bee3010e03ddb2d83f7160bd6e3000000000000002a0000000000000000",
		"0002c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2000000000000002a0000000000000000",
		"0002c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2000000000000002a0000000100000000",
		"0002dac17f958d2ee523a2206206994597c13d831ec7000000000000002a0000000000000001",
		"0002dac17f958d2ee523a2206206994597c13d831ec7000000000000002b0000000000000000",
		"00037fcf532c15f0a6db0bd6d0e038bea71d30d808c7d98cb3bf7268a95bf5081b65000000000000002a0000000100000000",
		"0003ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef000000000000002a0000000000000000",
		"0003ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef000000000000002a0000000000000001",
		"0003ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef000000000000002b0000000000000000",
		"00040000000000000000000000004a5e9a3bf35df5e0ea4b4cd3289e0111f1deadf7000000000000002a0000000000000000",
		"00040000000000000000000000004a5e9a3bf35df5e0ea4b4cd3289e0111f1deadf7000000000000002b0000000000000000",
		"0004000000000000000000000000f2a5fd4d5bb24b651d1198bc014941a16f6edde5000000000000002a0000000000000001",
		"0004000000000000000000000000f2a5fd4d5bb24b651d1198bc014941a16f6edde5000000000000002a0000000100000000",
		"00050000000000000000000000004a5e9a3bf35df5e0ea4b4cd3289e0111f1deadf7000000000000002a0000000000000001",
		"0005000000000000000000000000ac8cbe20039c2e9547da9107456179bd4f39a734000000000000002b0000000000000000",
		"0005000000000000000000000000f2a5fd4d5bb24b651d1198bc014941a16f6edde5000000000000002a0000000000000000",
	}

	b := NewIndexBuilder()
	b.AddBlockEntries(specHash39, 40, nil)
	b.AddBlockEntries(specHash40, 41, nil)
	b.AddBlockEntries(specHash41, 42, receipts42)
	b.AddBlockEntries(specHash42, 43, receipts43)
	if got := len(b.Entries()); got != 22 {
		t.Fatalf("entry count = %d, want 22", got)
	}
	b.Build() // sorts in place
	for i, e := range b.Entries() {
		if got := hex.EncodeToString(e); got != want[i] {
			t.Errorf("entry %d: encoded %s, want %s", i, got, want[i])
		}
	}

	// Semantics the prose describes, decoded from the sorted entries:
	// cumulative log counts, per-transaction log indices, one-block delay.
	checks := []struct {
		index int
		typ   EntryType
		value []byte
		block uint64
		tx    uint32
		log   uint32
	}{
		// 42tx1's transaction entry carries cumulative log count 2.
		{5, EntryTypeTransaction, specTx42_1[:], 42, 1, 2},
		// 42tx0's transaction entry has cumulative log count 0.
		{6, EntryTypeTransaction, specTx42_0[:], 42, 0, 0},
		// WETH log of 42tx1 has log index 0, not cumulative 2.
		{8, EntryTypeLogAddress, specWETH[:], 42, 1, 0},
		// The table for block 43 holds block 42's entry (one-block delay).
		{1, EntryTypeBlock, specHash42[:], 42, 0, 0},
	}
	for _, c := range checks {
		typ, content, block, tx, log, err := EntryFields(b.Entries()[c.index])
		if err != nil {
			t.Fatalf("entry %d: unexpected error: %v", c.index, err)
		}
		if typ != c.typ || !bytes.Equal(content, c.value) || block != c.block || tx != c.tx || log != c.log {
			t.Errorf("entry %d: decoded (type %d, %x, %d, %d, %d), want (%d, %x, %d, %d, %d)",
				c.index, typ, content, block, tx, log, c.typ, c.value, c.block, c.tx, c.log)
		}
	}
}

// TestIndexBuilderPerTableCounts pins the entry distribution across the four
// tables of the worked example: 1, 1, 14 and 6 entries.
func TestIndexBuilderPerTableCounts(t *testing.T) {
	receipts42 := Receipts{
		{TxHash: specTx42_0, Logs: []*Log{
			{Address: specWETH, Topics: []common.Hash{specTransfer, specAlice, specBob}},
			{Address: specUSDT, Topics: []common.Hash{specTransfer, specBob, specAlice}},
		}},
		{TxHash: specTx42_1, Logs: []*Log{
			{Address: specWETH, Topics: []common.Hash{specWithdrawal, specBob}},
		}},
	}
	receipts43 := Receipts{
		{TxHash: specTx43_0, Logs: []*Log{
			{Address: specUSDT, Topics: []common.Hash{specTransfer, specAlice, specCarol}},
		}},
	}
	tests := []struct {
		block    uint64
		parent   common.Hash
		receipts Receipts
		want     int
	}{
		{40, specHash39, nil, 1},
		{41, specHash40, nil, 1},
		{42, specHash41, receipts42, 14},
		{43, specHash42, receipts43, 6},
	}
	for _, tt := range tests {
		b := NewIndexBuilder()
		b.AddBlockEntries(tt.parent, tt.block, tt.receipts)
		if got := len(b.Entries()); got != tt.want {
			t.Errorf("block %d: entry count = %d, want %d", tt.block, got, tt.want)
		}
	}
}

// TestAddBlockEntriesGenesis covers the genesis edge cases: no block entry for
// block 0 and exactly the parent's entry for block 1.
func TestAddBlockEntriesGenesis(t *testing.T) {
	b := NewIndexBuilder()
	b.AddBlockEntries(specHash39, 0, nil)
	if got := len(b.Entries()); got != 0 {
		t.Fatalf("block 0: entry count = %d, want 0", got)
	}

	b = NewIndexBuilder()
	b.AddBlockEntries(specHash39, 1, nil)
	if got := len(b.Entries()); got != 1 {
		t.Fatalf("block 1: entry count = %d, want 1", got)
	}
	typ, content, block, tx, log, err := EntryFields(b.Entries()[0])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if typ != EntryTypeBlock || !bytes.Equal(content, specHash39[:]) || block != 0 || tx != 0 || log != 0 {
		t.Errorf("block 1 entry: decoded (type %d, %x, %d, %d, %d), want block entry of block 0", typ, content, block, tx, log)
	}

	// An empty builder still hashes (the genesis block never gets a table).
	if root := NewIndexBuilder().Build(); root == (common.Hash{}) {
		t.Errorf("empty builder root should be the keccak of an empty table")
	}
}

