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
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

// TestGetBlockAccessList exercises eth_getBlockAccessList against a chain
// with the Amsterdam fork active, asserting that the stored block access list
// is returned for number, tag and hash lookups, and that the JSON encoding
// follows the execution-apis spec field names.
func TestGetBlockAccessList(t *testing.T) {
	t.Parallel()
	accounts := newAccounts(2)

	cfg := *params.MergedTestChainConfig
	cfg.AmsterdamTime = uint64Ptr(0) // activate the block access list fork

	genBlocks := 3
	api := NewBlockChainAPI(newTestBackend(t, genBlocks, &core.Genesis{
		Config: &cfg,
		Alloc:  types.GenesisAlloc{accounts[0].addr: {Balance: big.NewInt(params.Ether)}},
	}, beacon.New(ethash.NewFaker()), func(i int, b *core.BlockGen) {
		tx, _ := types.SignTx(types.NewTx(&types.LegacyTx{
			Nonce:    uint64(i),
			To:       &accounts[1].addr,
			Value:    big.NewInt(1000),
			Gas:      params.TxGas,
			GasPrice: b.BaseFee(),
		}), types.HomesteadSigner{}, accounts[0].key)
		b.AddTx(tx)
		b.SetPoS()
	}))

	latest := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)

	// Resolve the latest block once for the by-hash lookup below.
	block, err := api.b.BlockByNumberOrHash(context.Background(), latest)
	if err != nil {
		t.Fatalf("failed to resolve latest block: %v", err)
	}

	var testSuite = []struct {
		name string
		ref  rpc.BlockNumberOrHash
	}{
		{"by-number", latest},
		{"by-hash", rpc.BlockNumberOrHashWithHash(block.Hash(), false)},
	}
	for _, tc := range testSuite {
		t.Run(tc.name, func(t *testing.T) {
			al, err := api.GetBlockAccessList(context.Background(), tc.ref)
			if err != nil {
				t.Fatalf("GetBlockAccessList error: %v", err)
			}
			if al == nil {
				t.Fatal("expected a non-nil access list")
			}
			if len(al) == 0 {
				t.Fatal("expected a non-empty access list for a block with transactions")
			}
			// The list is sorted lexicographically by address; the coinbase
			// (fee recipient) and both transfer accounts must be present.
			var found [3]bool
			want := []common.Address{{}, accounts[0].addr, accounts[1].addr}
			for i, entry := range al {
				if i > 0 && bytes.Compare(al[i-1].Address[:], entry.Address[:]) >= 0 {
					t.Fatalf("access list not sorted at index %d: %s >= %s", i, al[i-1].Address, entry.Address)
				}
				for j, addr := range want {
					if entry.Address == addr {
						found[j] = true
					}
				}
			}
			for j, addr := range want {
				if !found[j] {
					t.Fatalf("access list missing expected account %s", addr)
				}
			}
		})
	}

	// The JSON encoding must follow the execution-apis spec field names.
	al, err := api.GetBlockAccessList(context.Background(), latest)
	if err != nil {
		t.Fatalf("GetBlockAccessList error: %v", err)
	}
	blob, err := json.Marshal(al)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(blob, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}
	for _, want := range []string{"address", "balanceChanges", "nonceChanges", "codeChanges", "storageChanges", "storageReads"} {
		if _, ok := decoded[0][want]; !ok {
			t.Fatalf("access list entry missing spec field %q: %s", want, blob)
		}
	}
	for _, field := range []string{"balanceChanges", "nonceChanges", "codeChanges", "storageChanges"} {
		if !hasSpecInnerFields(decoded[0][field]) {
			t.Fatalf("spec field %q has wrong inner shape: %s", field, blob)
		}
	}

	// Unknown blocks must yield null, not an error.
	if al, err := api.GetBlockAccessList(context.Background(), rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(genBlocks+100))); al != nil || err != nil {
		t.Fatalf("expected null for unknown block, got list=%v err=%v", al, err)
	}
}

// hasSpecInnerFields checks that the change array uses the execution-apis
// index/value (or key/changes for storage) naming rather than the internal
// encoding naming.
func hasSpecInnerFields(v any) bool {
	entries, ok := v.([]any)
	if !ok || len(entries) == 0 {
		return true // empty arrays are fine
	}
	entry, ok := entries[0].(map[string]any)
	if !ok {
		return false
	}
	if _, ok := entry["index"]; ok {
		return true
	}
	if _, ok := entry["key"]; ok {
		if _, ok := entry["changes"]; ok {
			return true
		}
	}
	return false
}

func uint64Ptr(v uint64) *uint64 { return &v }
