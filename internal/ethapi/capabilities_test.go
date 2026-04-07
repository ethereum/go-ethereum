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

package ethapi

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	corestate "github.com/ethereum/go-ethereum/core/state"
)

func TestBuildCapabilities(t *testing.T) {
	const (
		archiveHead uint64 = 3_000_000
		postmerge   uint64 = 15_537_393
	)
	headHash := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	// retentionWindow is a small helper for asserting on
	// CapabilityResource fields.
	retentionWindow := func(n uint64) *uint64 { return &n }

	tests := []struct {
		name     string
		headNum  uint64
		cutoff   uint64
		ret      HistoryRetention
		expected map[string]CapabilityResource // by JSON field name
	}{
		{
			name:    "archive node, path scheme, all defaults",
			headNum: archiveHead,
			cutoff:  0,
			ret: HistoryRetention{
				StateArchive: true,
				StateScheme:  rawdb.PathScheme,
			},
			expected: map[string]CapabilityResource{
				"blocks":      {OldestBlock: 0, DeleteStrategy: DeleteStrategy{Type: "none"}},
				"receipts":    {OldestBlock: 0, DeleteStrategy: DeleteStrategy{Type: "none"}},
				"tx":          {OldestBlock: 0, DeleteStrategy: DeleteStrategy{Type: "none"}},
				"logs":        {OldestBlock: 0, DeleteStrategy: DeleteStrategy{Type: "none"}},
				"state":       {OldestBlock: 0, DeleteStrategy: DeleteStrategy{Type: "none"}},
				"stateproofs": {OldestBlock: 0, DeleteStrategy: DeleteStrategy{Type: "none"}},
			},
		},
		{
			name:    "post-merge pruned chain",
			headNum: archiveHead,
			cutoff:  postmerge,
			ret: HistoryRetention{
				StateScheme: rawdb.PathScheme,
			},
			expected: map[string]CapabilityResource{
				// blocks/receipts honor the absolute cutoff with no
				// sliding window.
				"blocks":   {OldestBlock: hexUint(postmerge), DeleteStrategy: DeleteStrategy{Type: "none"}},
				"receipts": {OldestBlock: hexUint(postmerge), DeleteStrategy: DeleteStrategy{Type: "none"}},
			},
		},
		{
			name:    "default tx and log indices, head above window",
			headNum: 5_000_000,
			cutoff:  0,
			ret: HistoryRetention{
				StateScheme:     rawdb.PathScheme,
				TxIndexHistory:  2_350_000,
				LogIndexHistory: 2_350_000,
			},
			expected: map[string]CapabilityResource{
				"tx": {
					OldestBlock:    hexUint(5_000_000 - 2_350_000 + 1),
					DeleteStrategy: DeleteStrategy{Type: "window", RetentionBlocks: retentionWindow(2_350_000)},
				},
				"logs": {
					OldestBlock:    hexUint(5_000_000 - 2_350_000 + 1),
					DeleteStrategy: DeleteStrategy{Type: "window", RetentionBlocks: retentionWindow(2_350_000)},
				},
			},
		},
		{
			name:    "head below tx window: clamp to cutoff, no underflow",
			headNum: 100,
			cutoff:  0,
			ret: HistoryRetention{
				StateScheme:    rawdb.PathScheme,
				TxIndexHistory: 2_350_000,
			},
			expected: map[string]CapabilityResource{
				"tx": {
					OldestBlock:    0,
					DeleteStrategy: DeleteStrategy{Type: "window", RetentionBlocks: retentionWindow(2_350_000)},
				},
			},
		},
		{
			name:    "tx window oldest clamped to history pruning cutoff",
			headNum: 5_000_000,
			cutoff:  4_000_000,
			ret: HistoryRetention{
				StateScheme:    rawdb.PathScheme,
				TxIndexHistory: 2_350_000, // would otherwise reach back to 2.65M
			},
			expected: map[string]CapabilityResource{
				"tx": {
					OldestBlock:    hexUint(4_000_000),
					DeleteStrategy: DeleteStrategy{Type: "window", RetentionBlocks: retentionWindow(2_350_000)},
				},
			},
		},
		{
			name:    "state windows are not clamped to history pruning cutoff",
			headNum: 5_000_000,
			cutoff:  4_950_000,
			ret: HistoryRetention{
				StateArchive:    true,
				StateScheme:     rawdb.PathScheme,
				StateHistory:    90_000,
				TrienodeHistory: 100_000,
			},
			expected: map[string]CapabilityResource{
				"state": {
					OldestBlock:    hexUint(5_000_000 - 90_000 + 1),
					DeleteStrategy: DeleteStrategy{Type: "window", RetentionBlocks: retentionWindow(90_000)},
				},
				"stateproofs": {
					OldestBlock:    hexUint(5_000_000 - 100_000 + 1),
					DeleteStrategy: DeleteStrategy{Type: "window", RetentionBlocks: retentionWindow(100_000)},
				},
			},
		},
		{
			name:    "log index disabled",
			headNum: 5_000_000,
			cutoff:  0,
			ret: HistoryRetention{
				StateScheme:      rawdb.PathScheme,
				LogIndexHistory:  2_350_000,
				LogIndexDisabled: true,
			},
			expected: map[string]CapabilityResource{
				"logs": {
					Disabled:       true,
					OldestBlock:    hexUint(5_000_000 - 2_350_000 + 1),
					DeleteStrategy: DeleteStrategy{Type: "window", RetentionBlocks: retentionWindow(2_350_000)},
				},
			},
		},
		{
			name:    "path archive with separate state and trienode history windows",
			headNum: 5_000_000,
			cutoff:  0,
			ret: HistoryRetention{
				StateArchive:    true,
				StateScheme:     rawdb.PathScheme,
				StateHistory:    90_000,
				TrienodeHistory: 50_000,
			},
			expected: map[string]CapabilityResource{
				"state": {
					OldestBlock:    hexUint(5_000_000 - 90_000 + 1),
					DeleteStrategy: DeleteStrategy{Type: "window", RetentionBlocks: retentionWindow(90_000)},
				},
				"stateproofs": {
					OldestBlock:    hexUint(5_000_000 - 50_000 + 1),
					DeleteStrategy: DeleteStrategy{Type: "window", RetentionBlocks: retentionWindow(50_000)},
				},
			},
		},
		{
			name:    "path archive with trienode history disabled retains in-memory proofs",
			headNum: 5_000_000,
			cutoff:  0,
			ret: HistoryRetention{
				StateArchive:    true,
				StateScheme:     rawdb.PathScheme,
				StateHistory:    90_000,
				TrienodeHistory: -1,
			},
			expected: map[string]CapabilityResource{
				"state": {
					OldestBlock:    hexUint(5_000_000 - 90_000 + 1),
					DeleteStrategy: DeleteStrategy{Type: "window", RetentionBlocks: retentionWindow(90_000)},
				},
				"stateproofs": {
					OldestBlock:    hexUint(5_000_000 - corestate.TriesInMemory + 1),
					DeleteStrategy: DeleteStrategy{Type: "window", RetentionBlocks: retentionWindow(corestate.TriesInMemory)},
				},
			},
		},
		{
			name:    "hash scheme archive ignores StateHistory",
			headNum: 5_000_000,
			cutoff:  0,
			ret: HistoryRetention{
				StateArchive: true,
				StateScheme:  rawdb.HashScheme,
				StateHistory: 90_000,
			},
			expected: map[string]CapabilityResource{
				"state":       {OldestBlock: 0, DeleteStrategy: DeleteStrategy{Type: "none"}},
				"stateproofs": {OldestBlock: 0, DeleteStrategy: DeleteStrategy{Type: "none"}},
			},
		},
		{
			name:    "full mode hash scheme retains in-memory state window",
			headNum: 5_000_000,
			cutoff:  0,
			ret: HistoryRetention{
				StateScheme:  rawdb.HashScheme,
				StateHistory: 90_000, // ignored under hash scheme
			},
			expected: map[string]CapabilityResource{
				"state": {
					OldestBlock:    hexUint(5_000_000 - corestate.TriesInMemory + 1),
					DeleteStrategy: DeleteStrategy{Type: "window", RetentionBlocks: retentionWindow(corestate.TriesInMemory)},
				},
				"stateproofs": {
					OldestBlock:    hexUint(5_000_000 - corestate.TriesInMemory + 1),
					DeleteStrategy: DeleteStrategy{Type: "window", RetentionBlocks: retentionWindow(corestate.TriesInMemory)},
				},
			},
		},
		{
			name:    "full mode path scheme ignores StateHistory",
			headNum: 5_000_000,
			cutoff:  0,
			ret: HistoryRetention{
				StateScheme:  rawdb.PathScheme,
				StateHistory: 90_000,
			},
			expected: map[string]CapabilityResource{
				"state": {
					OldestBlock:    hexUint(5_000_000 - corestate.TriesInMemory + 1),
					DeleteStrategy: DeleteStrategy{Type: "window", RetentionBlocks: retentionWindow(corestate.TriesInMemory)},
				},
				"stateproofs": {
					OldestBlock:    hexUint(5_000_000 - corestate.TriesInMemory + 1),
					DeleteStrategy: DeleteStrategy{Type: "window", RetentionBlocks: retentionWindow(corestate.TriesInMemory)},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := buildCapabilities(tt.headNum, headHash, tt.cutoff, tt.ret)

			// Head is always present.
			if uint64(caps.Head.BlockNumber) != tt.headNum {
				t.Errorf("head.blockNumber = %d, want %d", uint64(caps.Head.BlockNumber), tt.headNum)
			}
			if caps.Head.BlockHash != headHash {
				t.Errorf("head.blockHash = %x, want %x", caps.Head.BlockHash, headHash)
			}

			actual := map[string]CapabilityResource{
				"state":       caps.State,
				"tx":          caps.Tx,
				"logs":        caps.Logs,
				"receipts":    caps.Receipts,
				"blocks":      caps.Blocks,
				"stateproofs": caps.StateProofs,
			}
			for name, want := range tt.expected {
				got := actual[name]
				if got.Disabled != want.Disabled {
					t.Errorf("%s.disabled = %v, want %v", name, got.Disabled, want.Disabled)
				}
				if got.OldestBlock != want.OldestBlock {
					t.Errorf("%s.oldestBlock = %d, want %d", name, uint64(got.OldestBlock), uint64(want.OldestBlock))
				}
				if got.DeleteStrategy.Type != want.DeleteStrategy.Type {
					t.Errorf("%s.deleteStrategy.type = %q, want %q", name, got.DeleteStrategy.Type, want.DeleteStrategy.Type)
				}
				switch {
				case want.DeleteStrategy.RetentionBlocks == nil && got.DeleteStrategy.RetentionBlocks != nil:
					t.Errorf("%s.deleteStrategy.retentionBlocks = %d, want absent",
						name, *got.DeleteStrategy.RetentionBlocks)
				case want.DeleteStrategy.RetentionBlocks != nil && got.DeleteStrategy.RetentionBlocks == nil:
					t.Errorf("%s.deleteStrategy.retentionBlocks absent, want %d",
						name, *want.DeleteStrategy.RetentionBlocks)
				case want.DeleteStrategy.RetentionBlocks != nil && got.DeleteStrategy.RetentionBlocks != nil:
					if *got.DeleteStrategy.RetentionBlocks != *want.DeleteStrategy.RetentionBlocks {
						t.Errorf("%s.deleteStrategy.retentionBlocks = %d, want %d",
							name, *got.DeleteStrategy.RetentionBlocks, *want.DeleteStrategy.RetentionBlocks)
					}
				}
			}
		})
	}
}

// TestCapabilitiesJSONShape verifies that the marshalled JSON conforms to
// the schema defined in https://github.com/ethereum/execution-apis/pull/755:
// "none" strategies must omit retentionBlocks, oldestBlock must be a hex
// quantity, retentionBlocks must be a decimal integer.
func TestCapabilitiesJSONShape(t *testing.T) {
	caps := buildCapabilities(
		5_000_000,
		common.HexToHash("0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3"),
		0,
		HistoryRetention{
			StateScheme:     rawdb.PathScheme,
			TxIndexHistory:  2_350_000,
			LogIndexHistory: 2_350_000,
			StateHistory:    90_000,
		},
	)

	raw, err := json.Marshal(caps)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Round-trip through a generic map so we can assert on key presence.
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Top-level keys must match the spec.
	required := []string{"head", "state", "tx", "logs", "receipts", "blocks", "stateproofs"}
	for _, k := range required {
		if _, ok := generic[k]; !ok {
			t.Errorf("missing top-level key %q", k)
		}
	}

	// head.blockNumber must be a hex string ("0x..."), blockHash must be a 0x hash.
	head := generic["head"].(map[string]any)
	if bn, ok := head["blockNumber"].(string); !ok || len(bn) < 3 || bn[:2] != "0x" {
		t.Errorf("head.blockNumber not hex string: %v", head["blockNumber"])
	}
	if bh, ok := head["blockHash"].(string); !ok || len(bh) != 66 {
		t.Errorf("head.blockHash not 32-byte hex string: %v", head["blockHash"])
	}

	// blocks.deleteStrategy is "none" → must NOT contain retentionBlocks.
	blocks := generic["blocks"].(map[string]any)
	bds := blocks["deleteStrategy"].(map[string]any)
	if bds["type"] != "none" {
		t.Errorf("blocks.deleteStrategy.type = %v, want none", bds["type"])
	}
	if _, present := bds["retentionBlocks"]; present {
		t.Errorf("blocks.deleteStrategy must not include retentionBlocks for type=none")
	}

	// tx.deleteStrategy is "window" → must contain retentionBlocks as a
	// decimal number, not a hex string.
	tx := generic["tx"].(map[string]any)
	tds := tx["deleteStrategy"].(map[string]any)
	if tds["type"] != "window" {
		t.Errorf("tx.deleteStrategy.type = %v, want window", tds["type"])
	}
	rb, ok := tds["retentionBlocks"].(float64)
	if !ok {
		t.Fatalf("tx.deleteStrategy.retentionBlocks not a JSON number: %T %v",
			tds["retentionBlocks"], tds["retentionBlocks"])
	}
	if uint64(rb) != 2_350_000 {
		t.Errorf("tx.deleteStrategy.retentionBlocks = %v, want 2350000", rb)
	}

	// tx.oldestBlock must be a hex string.
	if ob, ok := tx["oldestBlock"].(string); !ok || len(ob) < 3 || ob[:2] != "0x" {
		t.Errorf("tx.oldestBlock not hex string: %v", tx["oldestBlock"])
	}
}

// hexUint is a small helper to keep the test tables compact.
func hexUint(n uint64) hexutil.Uint64 { return hexutil.Uint64(n) }
