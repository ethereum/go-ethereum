// Copyright 2025 The go-ethereum Authors
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

package trie

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/holiman/uint256"
)

// TestInspect inspects a randomly generated account trie. It's useful for
// quickly verifying changes to the results display.
func TestInspect(t *testing.T) {
	db := newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme)
	trie, err := NewStateTrie(TrieID(types.EmptyRootHash), db)
	if err != nil {
		t.Fatalf("failed to create state trie: %v", err)
	}
	// Create a realistic looking account trie with storage.
	addresses, accounts := makeAccountsWithStorage(db, 11, true)
	for i := 0; i < len(addresses); i++ {
		trie.MustUpdate(crypto.Keccak256(addresses[i][:]), accounts[i])
	}
	// Insert the accounts into the trie and hash it
	root, nodes := trie.Commit(true)
	db.Update(root, types.EmptyRootHash, trienode.NewWithNodeSet(nodes))
	db.Commit(root)

	tempDir := t.TempDir()
	dumpPath := filepath.Join(tempDir, "trie-dump.bin")
	if err := Inspect(db, root, &InspectConfig{
		TopN:     1,
		DumpPath: dumpPath,
		Path:     filepath.Join(tempDir, "trie-summary.json"),
	}); err != nil {
		t.Fatalf("inspect failed: %v", err)
	}
	reanalysisPath := filepath.Join(tempDir, "trie-summary-reanalysis.json")
	if err := Summarize(dumpPath, &InspectConfig{
		TopN: 1,
		Path: reanalysisPath,
	}); err != nil {
		t.Fatalf("summarize failed: %v", err)
	}

	inspectSummaryPath := filepath.Join(tempDir, "trie-summary.json")
	inspectOut := loadInspectJSON(t, inspectSummaryPath)
	reanalysisOut := loadInspectJSON(t, reanalysisPath)

	if len(inspectOut.StorageSummary.Levels) == 0 {
		t.Fatal("expected StorageSummary.Levels to be populated")
	}
	if inspectOut.AccountTrie.Summary.Size == 0 {
		t.Fatal("expected account trie size summary to be populated")
	}
	if inspectOut.StorageSummary.Totals.Size == 0 {
		t.Fatal("expected storage trie size summary to be populated")
	}
	if !reflect.DeepEqual(inspectOut.AccountTrie, reanalysisOut.AccountTrie) {
		t.Fatal("account trie summary mismatch between inspect and summarize")
	}
	if !reflect.DeepEqual(inspectOut.StorageSummary, reanalysisOut.StorageSummary) {
		t.Fatal("storage summary mismatch between inspect and summarize")
	}

	assertStorageTotalsMatchLevels(t, inspectOut)
	assertStorageTotalsMatchLevels(t, reanalysisOut)
	assertAccountTotalsMatchLevels(t, inspectOut.AccountTrie)
	assertAccountTotalsMatchLevels(t, reanalysisOut.AccountTrie)

	var histogramTotal uint64
	for _, count := range inspectOut.StorageSummary.DepthHistogram {
		histogramTotal += count
	}
	if histogramTotal != inspectOut.StorageSummary.TotalStorageTries {
		t.Fatalf("depth histogram total %d does not match total storage tries %d", histogramTotal, inspectOut.StorageSummary.TotalStorageTries)
	}
}

type inspectJSONOutput struct {
	// Reuse storageStats for AccountTrie JSON to avoid introducing a parallel
	// account summary test type. AccountTrie JSON includes Levels+Summary,
	// which map directly; other storageStats fields remain zero-values.
	AccountTrie storageStats `json:"AccountTrie"`

	StorageSummary struct {
		TotalStorageTries uint64                 `json:"TotalStorageTries"`
		Totals            jsonLevel              `json:"Totals"`
		Levels            []jsonLevel            `json:"Levels"`
		DepthHistogram    [trieStatLevels]uint64 `json:"DepthHistogram"`
	} `json:"StorageSummary"`
}

func loadInspectJSON(t *testing.T, path string) inspectJSONOutput {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	var out inspectJSONOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("failed to decode %s: %v", path, err)
	}
	return out
}

func assertStorageTotalsMatchLevels(t *testing.T, out inspectJSONOutput) {
	t.Helper()
	var fromLevels jsonLevel
	for _, level := range out.StorageSummary.Levels {
		fromLevels.Short += level.Short
		fromLevels.Full += level.Full
		fromLevels.Value += level.Value
		fromLevels.Size += level.Size
	}
	if fromLevels.Short != out.StorageSummary.Totals.Short || fromLevels.Full != out.StorageSummary.Totals.Full || fromLevels.Value != out.StorageSummary.Totals.Value || fromLevels.Size != out.StorageSummary.Totals.Size {
		t.Fatalf("storage totals mismatch: levels=%+v totals=%+v", fromLevels, out.StorageSummary.Totals)
	}
}

func assertAccountTotalsMatchLevels(t *testing.T, account storageStats) {
	t.Helper()
	var fromLevels jsonLevel
	for _, level := range account.Levels {
		fromLevels.Short += level.Short
		fromLevels.Full += level.Full
		fromLevels.Value += level.Value
		fromLevels.Size += level.Size
	}
	if fromLevels.Short != account.Summary.Short || fromLevels.Full != account.Summary.Full || fromLevels.Value != account.Summary.Value || fromLevels.Size != account.Summary.Size {
		t.Fatalf("account totals mismatch: levels=%+v totals=%+v", fromLevels, account.Summary)
	}
}

func makeAccountsWithStorage(db *testDb, size int, storage bool) (addresses [][20]byte, accounts [][]byte) {
	// Make the random benchmark deterministic
	random := rand.New(rand.NewSource(0))

	addresses = make([][20]byte, size)
	for i := 0; i < len(addresses); i++ {
		data := make([]byte, 20)
		random.Read(data)
		copy(addresses[i][:], data)
	}
	accounts = make([][]byte, len(addresses))
	for i := 0; i < len(accounts); i++ {
		var (
			nonce = uint64(random.Int63())
			root  = types.EmptyRootHash
			code  = crypto.Keccak256(nil)
		)
		if storage {
			trie := NewEmpty(db)
			for range random.Uint32()%256 + 1 { // non-zero
				k, v := make([]byte, 32), make([]byte, 32)
				random.Read(k)
				random.Read(v)
				trie.MustUpdate(k, v)
			}
			var nodes *trienode.NodeSet
			root, nodes = trie.Commit(true)
			db.Update(root, types.EmptyRootHash, trienode.NewWithNodeSet(nodes))
			db.Commit(root)
		}
		numBytes := random.Uint32() % 33 // [0, 32] bytes
		balanceBytes := make([]byte, numBytes)
		random.Read(balanceBytes)
		balance := new(uint256.Int).SetBytes(balanceBytes)
		data, _ := rlp.EncodeToBytes(&types.StateAccount{
			Nonce:    nonce,
			Balance:  balance,
			Root:     root,
			CodeHash: code,
		})
		accounts[i] = data
	}
	return addresses, accounts
}
