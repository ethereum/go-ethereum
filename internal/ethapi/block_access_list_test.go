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
	"errors"
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
// matches the core/types/bal package output.
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
			if len(*al) == 0 {
				t.Fatal("expected a non-empty access list for a block with transactions")
			}
			// The list is sorted lexicographically by address; the coinbase
			// (recorded even when untouched) and both transfer accounts must
			// be present.
			var found [3]bool
			want := []common.Address{{}, accounts[0].addr, accounts[1].addr}
			for i, entry := range *al {
				if i > 0 && bytes.Compare((*al)[i-1].Address[:], entry.Address[:]) >= 0 {
					t.Fatalf("access list not sorted at index %d: %s >= %s", i, (*al)[i-1].Address, entry.Address)
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

	// The JSON encoding must match the core/types/bal package output: all
	// change lists are arrays (never null) and the inner objects carry the
	// bal package's field names.
	al, err := api.GetBlockAccessList(context.Background(), latest)
	if err != nil {
		t.Fatalf("GetBlockAccessList error: %v", err)
	}
	blob, err := json.Marshal(al)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	if bytes.Contains(blob, []byte("null")) {
		t.Fatalf("marshaled access list contains null: %s", blob)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(blob, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}
	for _, entry := range decoded {
		for _, want := range []string{"address", "balanceChanges", "nonceChanges", "codeChanges", "storageChanges", "storageReads"} {
			if _, ok := entry[want]; !ok {
				t.Fatalf("access list entry missing field %q: %s", want, blob)
			}
		}
		assertChangeKeys(t, entry["storageChanges"], false, "slot", "slotChanges")
		if slots, ok := entry["storageChanges"].([]any); ok {
			for _, slot := range slots {
				assertChangeKeys(t, slot.(map[string]any)["slotChanges"], false, "blockAccessIndex", "postValue")
			}
		}
		assertChangeKeys(t, entry["codeChanges"], false, "blockAccessIndex", "newCode")
	}
	// The sender has balance and nonce changes in every block; pin their
	// exact key sets.
	sender := findEntry(t, decoded, accounts[0].addr)
	assertChangeKeys(t, sender["balanceChanges"], true, "blockAccessIndex", "postBalance")
	assertChangeKeys(t, sender["nonceChanges"], true, "blockAccessIndex", "postNonce")

	// The pending tag serves null: the pending block's access list is not
	// durable.
	if al, err := api.GetBlockAccessList(context.Background(), rpc.BlockNumberOrHashWithNumber(rpc.PendingBlockNumber)); al != nil || err != nil {
		t.Fatalf("expected null for pending block, got list=%v err=%v", al, err)
	}

	// Unknown blocks must yield null, not an error -- by number and by hash.
	if al, err := api.GetBlockAccessList(context.Background(), rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(genBlocks+100))); al != nil || err != nil {
		t.Fatalf("expected null for unknown block, got list=%v err=%v", al, err)
	}
	if al, err := api.GetBlockAccessList(context.Background(), rpc.BlockNumberOrHashWithHash(common.Hash{0xaa}, false)); al != nil || err != nil {
		t.Fatalf("expected null for unknown hash, got list=%v err=%v", al, err)
	}

	// Header lookup failures pass through, except the unknown-hash case which
	// the RPC spec requires to be null.
	errAPI := NewBlockChainAPI(errHeaderBackend{api.b})
	if al, err := errAPI.GetBlockAccessList(context.Background(), rpc.BlockNumberOrHashWithHash(block.Hash(), false)); al != nil || err != nil {
		t.Fatalf("expected null for unknown hash, got list=%v err=%v", al, err)
	}
	if _, err := errAPI.GetBlockAccessList(context.Background(), rpc.BlockNumberOrHashWithHash(block.Hash(), true)); err == nil {
		t.Fatal("expected requireCanonical to propagate the header error")
	}
	if _, err := errAPI.GetBlockAccessList(context.Background(), rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(1))); err == nil {
		t.Fatal("expected number-path error to propagate")
	}
}

// assertChangeKeys verifies the exact key set of the first element of a
// marshaled change array, catching both renames and added fields. With
// nonEmpty it fails on empty arrays instead of skipping them.
func assertChangeKeys(t *testing.T, v any, nonEmpty bool, keys ...string) {
	t.Helper()
	arr, ok := v.([]any)
	if !ok || len(arr) == 0 {
		if nonEmpty {
			t.Fatalf("expected non-empty change array, got %v", v)
		}
		return
	}
	entry, ok := arr[0].(map[string]any)
	if !ok {
		t.Fatalf("expected object entries, got %v", arr[0])
	}
	if len(entry) != len(keys) {
		t.Fatalf("entry has keys %v, want exactly %v", entry, keys)
	}
	for _, key := range keys {
		if _, ok := entry[key]; !ok {
			t.Fatalf("entry missing key %q", key)
		}
	}
}

// findEntry returns the access list entry of the given account.
func findEntry(t *testing.T, decoded []map[string]any, addr common.Address) map[string]any {
	t.Helper()
	for _, entry := range decoded {
		if common.HexToAddress(entry["address"].(string)) == addr {
			return entry
		}
	}
	t.Fatalf("no access list entry for %s", addr)
	return nil
}

// errHeaderBackend wraps a backend, failing every header lookup like the real
// one does for unknown hashes and canonicality violations.
type errHeaderBackend struct {
	Backend
}

func (b errHeaderBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	return nil, errors.New("header for hash not found")
}

func uint64Ptr(v uint64) *uint64 { return &v }
