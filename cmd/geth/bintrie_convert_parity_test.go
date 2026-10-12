// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

// The parity oracles: read-back tests cannot see a converter bug (the tree
// hashes whatever was written), so the converter is pinned by root equality
// against independent provenance - the execution-specs state_vectors and the
// tree's own typed writers.

// stateVector mirrors the state_vectors entries of eip8297_vectors.json.
type stateVector struct {
	Name     string `json:"name"`
	Accounts []struct {
		Address string            `json:"address"`
		Nonce   uint64            `json:"nonce"`
		Balance *big.Int          `json:"balance"`
		Code    string            `json:"code"`
		Storage map[string]uint64 `json:"storage"`
	} `json:"accounts"`
	Root string `json:"root"`
}

func loadStateVectors(t *testing.T) []stateVector {
	t.Helper()
	blob, err := os.ReadFile("../../trie/bintrie/testdata/eip8297_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		StateVectors []stateVector `json:"state_vectors"`
	}
	if err := json.Unmarshal(blob, &file); err != nil {
		t.Fatal(err)
	}
	if len(file.StateVectors) == 0 {
		t.Fatal("no state vectors in the file; the exporter did not write them")
	}
	return file.StateVectors
}

// allocOf converts a vector's accounts into a genesis allocation.
func allocOf(t *testing.T, sv stateVector) types.GenesisAlloc {
	t.Helper()
	alloc := make(types.GenesisAlloc, len(sv.Accounts))
	for _, a := range sv.Accounts {
		acct := types.Account{Nonce: a.Nonce, Balance: a.Balance}
		if acct.Balance == nil {
			acct.Balance = new(big.Int)
		}
		if a.Code != "" {
			acct.Code = common.FromHex(a.Code)
		}
		if len(a.Storage) > 0 {
			acct.Storage = make(map[common.Hash]common.Hash, len(a.Storage))
			for slot, val := range a.Storage {
				key, ok := new(big.Int).SetString(slot, 10)
				if !ok {
					t.Fatalf("bad storage key %q", slot)
				}
				var k, v common.Hash
				key.FillBytes(k[:])
				new(big.Int).SetUint64(val).FillBytes(v[:])
				acct.Storage[k] = v
			}
		}
		alloc[common.HexToAddress(a.Address)] = acct
	}
	return alloc
}

// convertAlloc converts alloc from a merkle genesis and returns the binary
// root and database.
func convertAlloc(t *testing.T, alloc types.GenesisAlloc) (common.Hash, ethdb.Database) {
	t.Helper()
	return convertAllocOpts(t, alloc, conversionOptions{})
}

// convertAllocOpts is convertAlloc under explicit conversion options.
func convertAllocOpts(t *testing.T, alloc types.GenesisAlloc, opts conversionOptions) (common.Hash, ethdb.Database) {
	t.Helper()
	chaindb, _, binRoot, err := convertGenesis(t, alloc, true, opts)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	return binRoot, chaindb
}

// convertGenesis commits alloc as a merkle genesis, under the path scheme or
// else the hash scheme, and converts it under opts. The source keeps a
// preimage store unless opts names a preimage file. It returns the source
// database, its merkle root and the conversion's outcome.
func convertGenesis(t *testing.T, alloc types.GenesisAlloc, pathScheme bool, opts conversionOptions) (chaindb ethdb.Database, root, binRoot common.Hash, err error) {
	t.Helper()
	keep := opts.preimageIn == ""
	cfg, roCfg := &triedb.Config{Preimages: keep}, &triedb.Config{Preimages: keep}
	if pathScheme {
		cfg.PathDB, roCfg.PathDB = pathdb.Defaults, pathdb.ReadOnly
	}
	chaindb = rawdb.NewMemoryDatabase()
	srcTriedb := triedb.NewDatabase(chaindb, cfg)
	gspec := &core.Genesis{Config: params.TestChainConfig, BaseFee: big.NewInt(params.InitialBaseFee), Alloc: alloc}
	root = gspec.MustCommit(chaindb, srcTriedb).Root()
	srcTriedb.Close()

	src := triedb.NewDatabase(chaindb, roCfg)
	defer src.Close()
	// The flat scan needs a completed snapshot; without one a path-scheme
	// conversion would silently fall back to the node walk.
	if pathScheme && !src.SnapshotCompleted() {
		t.Fatal("path-scheme source has no completed snapshot")
	}
	binRoot, err = convertState(chaindb, src, root, opts)
	return chaindb, root, binRoot, err
}

// artifactOptions writes both artifacts, and spills, to a fresh temp dir.
func artifactOptions(t *testing.T) conversionOptions {
	dir := t.TempDir()
	return conversionOptions{
		tmpDir:       dir,
		snapshotPath: filepath.Join(dir, "snapshot.bin"),
		preimagePath: filepath.Join(dir, "preimages.bin"),
	}
}

// embedAlloc writes alloc through the typed writers - what replay does - and
// returns the root.
func embedAlloc(t *testing.T, alloc types.GenesisAlloc) common.Hash {
	t.Helper()
	tr, err := bintrie.NewBinaryTrie(types.EmptyBinaryHash, triedb.NewDatabase(rawdb.NewMemoryDatabase(), triedb.PBTDefaults))
	if err != nil {
		t.Fatal(err)
	}
	for addr, a := range alloc {
		code := a.Code
		acc := &types.StateAccount{
			Nonce:    a.Nonce,
			Balance:  balanceOf(t, a.Balance),
			Root:     types.EmptyRootHash,
			CodeHash: crypto.Keccak256(code),
		}
		var delegation []byte
		if _, ok := types.ParseDelegation(code); ok {
			delegation = code
		}
		if err := tr.UpdateAccount(addr, acc, len(code), delegation); err != nil {
			t.Fatalf("account %x: %v", addr, err)
		}
		if err := tr.UpdateContractCode(addr, common.BytesToHash(acc.CodeHash), code); err != nil {
			t.Fatalf("code for %x: %v", addr, err)
		}
		for k, v := range a.Storage {
			if err := tr.UpdateStorage(addr, k[:], common.TrimLeftZeroes(v[:])); err != nil {
				t.Fatalf("storage %x of %x: %v", k, addr, err)
			}
		}
	}
	return tr.Hash()
}

// balanceOf converts a genesis balance for the typed writers.
func balanceOf(t *testing.T, b *big.Int) *uint256.Int {
	t.Helper()
	if b == nil {
		return new(uint256.Int)
	}
	v, overflow := uint256.FromBig(b)
	if overflow {
		t.Fatalf("balance %v overflows", b)
	}
	return v
}

// assertConvertedTreeClean checks EIP-8297's invariant: no leaf holds 32
// zero bytes.
func assertConvertedTreeClean(t *testing.T, chaindb ethdb.Database, root common.Hash) {
	t.Helper()
	db := triedb.NewDatabase(chaindb, &triedb.Config{IsPBT: true, PathDB: pathdb.Defaults})
	defer db.Close()
	tr, err := bintrie.NewBinaryTrie(root, db)
	if err != nil {
		t.Fatal(err)
	}
	it, err := tr.NodeIterator(nil)
	if err != nil {
		t.Fatal(err)
	}
	var zero [32]byte
	for it.Next(true) {
		if it.Leaf() && bytes.Equal(it.LeafBlob(), zero[:]) {
			t.Fatalf("leaf %x holds 32 zero bytes, which no key in the state's tree may", it.LeafKey())
		}
	}
	if err := it.Error(); err != nil {
		t.Fatal(err)
	}
}

// TestConvertMatchesReference pins the converter against the execution-specs
// reference: each vector's allocation, converted, must hit the
// reference-computed root.
func TestConvertMatchesReference(t *testing.T) {
	for _, sv := range loadStateVectors(t) {
		t.Run(sv.Name, func(t *testing.T) {
			if sv.Name == "zero_basic_data_with_storage" {
				t.Skip("EIP-7523: no anchor state holds an empty account; TestImportMatchesReference pins the refusal")
			}
			binRoot, chaindb := convertAlloc(t, allocOf(t, sv))
			if want := common.HexToHash(sv.Root); binRoot != want {
				t.Fatalf("converted root %x, reference says %s", binRoot, sv.Root)
			}
			assertConvertedTreeClean(t, chaindb, binRoot)
		})
	}
}

// TestConvertMatchesEmbedding is the randomized differential: a mixed
// allocation converted from a merkle source must hash identically to the
// same allocation written through the typed writers.
func TestConvertMatchesEmbedding(t *testing.T) {
	alloc := mixedAlloc(8347)
	want := embedAlloc(t, alloc)

	// In memory and spilling: the sort path must not change the answer.
	for _, budget := range []int{0, 512} {
		binRoot, chaindb := convertAllocOpts(t, alloc, conversionOptions{
			sortBudget: budget,
			tmpDir:     t.TempDir(),
		})
		if binRoot != want {
			t.Fatalf("budget %d: converted root %x, the typed writers produce %x", budget, binRoot, want)
		}
		assertConvertedTreeClean(t, chaindb, binRoot)
		assertFlatStateMatchesAlloc(t, chaindb, alloc)
	}
}

// TestConvertParallelMatchesSerial: a scan split across workers converts the
// mixed allocation to the typed writers' root from either source, with code
// held by accounts in different ranges; the artifacts come out byte-identical
// to a serial run's, and one failing worker fails the conversion.
func TestConvertParallelMatchesSerial(t *testing.T) {
	alloc := mixedAlloc(8347)
	want := embedAlloc(t, alloc)

	// The case the shared code set exists for: one code in two ranges.
	ranges, holders := splitHashSpace(4), make(map[common.Hash]map[int]bool)
	for addr, acct := range alloc {
		if len(acct.Code) == 0 {
			continue
		}
		hash, codeHash := crypto.Keccak256Hash(addr[:]), crypto.Keccak256Hash(acct.Code)
		i := 0
		for ranges[i].past(hash) {
			i++
		}
		if holders[codeHash] == nil {
			holders[codeHash] = make(map[int]bool)
		}
		holders[codeHash][i] = true
	}
	shared := false
	for _, in := range holders {
		shared = shared || len(in) > 1
	}
	if !shared {
		t.Fatal("fixture holds no code in more than one of four ranges")
	}

	for _, pathScheme := range []bool{true, false} {
		for _, budget := range []int{0, 512} {
			for _, workers := range []int{2, 4, 8} {
				chaindb, _, binRoot, err := convertGenesis(t, alloc, pathScheme, conversionOptions{
					sortBudget: budget,
					tmpDir:     t.TempDir(),
					workers:    workers,
				})
				if err != nil {
					t.Fatalf("path scheme %v, budget %d, %d workers: %v", pathScheme, budget, workers, err)
				}
				if binRoot != want {
					t.Fatalf("path scheme %v, budget %d, %d workers: root %x, want %x", pathScheme, budget, workers, binRoot, want)
				}
				assertFlatStateMatchesAlloc(t, chaindb, alloc)
			}
		}
	}

	// Every worker writes its range of the preimage file; the joined file,
	// like the snapshot, must not depend on how many there were. The small
	// allocation leaves most of eight ranges, and their fragments, empty.
	t.Run("artifacts", func(t *testing.T) {
		artifacts := func(alloc types.GenesisAlloc, workers int) (snap, pre []byte) {
			opts := artifactOptions(t)
			opts.workers = workers
			convertAllocOpts(t, alloc, opts)
			snap, err := os.ReadFile(opts.snapshotPath)
			if err != nil {
				t.Fatal(err)
			}
			pre, err = os.ReadFile(opts.preimagePath)
			if err != nil {
				t.Fatal(err)
			}
			return snap, pre
		}
		for _, alloc := range []types.GenesisAlloc{alloc, artifactAlloc()} {
			snap1, pre1 := artifacts(alloc, 1)
			for _, workers := range []int{4, 8} {
				snap, pre := artifacts(alloc, workers)
				if !bytes.Equal(snap1, snap) || !bytes.Equal(pre1, pre) {
					t.Fatalf("%d accounts: artifacts differ between one and %d workers", len(alloc), workers)
				}
			}
		}
	})

	t.Run("failing worker", func(t *testing.T) {
		chaindb := rawdb.NewMemoryDatabase()
		srcTriedb := triedb.NewDatabase(chaindb, &triedb.Config{Preimages: true, PathDB: pathdb.Defaults})
		gspec := &core.Genesis{Config: params.TestChainConfig, BaseFee: big.NewInt(params.InitialBaseFee), Alloc: alloc}
		root := gspec.MustCommit(chaindb, srcTriedb).Root()
		srcTriedb.Close()

		// Slot 1 is in {0xed, 3}'s storage; 33 bytes is no slot key.
		slot := common.BigToHash(big.NewInt(1))
		rawdb.WritePreimages(chaindb, map[common.Hash][]byte{crypto.Keccak256Hash(slot[:]): make([]byte, 33)})

		src := triedb.NewDatabase(chaindb, &triedb.Config{Preimages: true, PathDB: pathdb.ReadOnly})
		defer src.Close()
		_, err := convertState(chaindb, src, root, conversionOptions{tmpDir: t.TempDir(), workers: 4})
		if err == nil || !strings.Contains(err.Error(), "corrupt preimage") {
			t.Fatalf("a corrupt slot preimage converted on four workers; err = %v", err)
		}
		if rawdb.HasPBTState(chaindb) {
			t.Fatal("a failed parallel conversion attested its namespace")
		}
	})
}

// TestConvertFromPreimageFile: a source holding no preimages converts from a
// preimage file, read in step with the scan, to the root, flat state and
// artifacts a conversion from the node's store produces, at any worker
// count and from either source walk; a file that names a key the state does
// not hold, or lacks one it does, is refused.
func TestConvertFromPreimageFile(t *testing.T) {
	alloc := mixedAlloc(8347)
	want := embedAlloc(t, alloc)
	_, _, _, snapPath, prePath := importFixture(t, alloc)
	wantSnap, wantPre := mustReadFile(t, snapPath), mustReadFile(t, prePath)

	for _, pathScheme := range []bool{true, false} {
		for _, workers := range []int{1, 4} {
			opts := artifactOptions(t)
			opts.workers, opts.preimageIn = workers, prePath
			chaindb, _, binRoot, err := convertGenesis(t, alloc, pathScheme, opts)
			if err != nil {
				t.Fatalf("path scheme %v, %d workers: %v", pathScheme, workers, err)
			}
			if binRoot != want {
				t.Fatalf("path scheme %v, %d workers: root %x, want %x", pathScheme, workers, binRoot, want)
			}
			assertFlatStateMatchesAlloc(t, chaindb, alloc)
			if !bytes.Equal(mustReadFile(t, opts.snapshotPath), wantSnap) || !bytes.Equal(mustReadFile(t, opts.preimagePath), wantPre) {
				t.Fatalf("path scheme %v, %d workers: artifacts differ from a conversion from the store", pathScheme, workers)
			}
		}
	}

	t.Run("more workers than records", func(t *testing.T) {
		small := artifactAlloc()
		_, _, smallRoot, _, smallPre := importFixture(t, small)
		if _, _, binRoot, err := convertGenesis(t, small, true, conversionOptions{tmpDir: t.TempDir(), workers: 8, preimageIn: smallPre}); err != nil || binRoot != smallRoot {
			t.Fatalf("root %x, err %v; a conversion from the store builds %x", binRoot, err, smallRoot)
		}
	})

	t.Run("rejects", func(t *testing.T) {
		recs := readPreimageRecords(t, prePath)
		withSlots := slices.IndexFunc(recs, func(r preRecord) bool { return len(r.slots) > 0 })
		if withSlots < 0 {
			t.Fatal("the fixture's preimage file holds no slot keys")
		}
		edit := func(f func(recs []preRecord) []preRecord) []preRecord {
			cloned := make([]preRecord, len(recs))
			for i, r := range recs {
				cloned[i] = preRecord{addr: r.addr, slots: slices.Clone(r.slots)}
			}
			return f(cloned)
		}
		for _, tc := range []struct {
			name string
			recs []preRecord
			want string
		}{
			{"missing account", edit(func(r []preRecord) []preRecord { return slices.Delete(r, 0, 1) }),
				"missing preimage for account hash"},
			{"surplus account", edit(func(r []preRecord) []preRecord {
				return append(r, preRecord{addr: common.HexToAddress("0xdeadbeef00000000000000000000000000000001")})
			}), "which the state does not hold"},
			{"missing slot", edit(func(r []preRecord) []preRecord {
				r[withSlots].slots = r[withSlots].slots[1:]
				return r
			}), "missing preimage for storage key"},
			{"surplus slot", edit(func(r []preRecord) []preRecord {
				r[withSlots].slots = append(r[withSlots].slots, common.Hash{0xde, 0xad})
				return r
			}), "which the state does not hold"},
		} {
			for _, workers := range []int{1, 4} {
				t.Run(fmt.Sprintf("%s/%d workers", tc.name, workers), func(t *testing.T) {
					bad := filepath.Join(t.TempDir(), "preimages.bin")
					writePreimageRecords(t, bad, tc.recs)
					chaindb, _, _, err := convertGenesis(t, alloc, true, conversionOptions{tmpDir: t.TempDir(), workers: workers, preimageIn: bad})
					if err == nil || !strings.Contains(err.Error(), tc.want) {
						t.Fatalf("err = %v, want %q", err, tc.want)
					}
					if rawdb.HasPBTState(chaindb) {
						t.Fatal("a refused conversion attested its namespace")
					}
				})
			}
		}
	})
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	blob, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return blob
}

// mixedAlloc builds a deterministic randomized allocation: shared code,
// delegations, boundary slots, code spanning multiple code-zone stems.
func mixedAlloc(seed int64) types.GenesisAlloc {
	rng := rand.New(rand.NewSource(seed))
	alloc := make(types.GenesisAlloc)

	// Shared blobs, a designator, a zero tail, a blob past 256 chunks.
	codes := [][]byte{
		nil,
		bytes.Repeat([]byte{0x5b}, 40),
		bytes.Repeat([]byte{0x5b}, 40), // deliberately identical to the previous
		append([]byte{0x60, 0x01, 0x00}, make([]byte, 80)...),
		bytes.Repeat([]byte{0x60, 0x01}, 700), // spans several code-zone groups
		bytes.Repeat([]byte{0x5b}, 10*1024),   // spans code-zone stems
		types.AddressToDelegation(common.Address{0xde, 0x1e}),
	}
	for i := 0; i < 60; i++ {
		var addr common.Address
		rng.Read(addr[:])
		acct := types.Account{
			Nonce:   uint64(rng.Intn(3)),
			Balance: big.NewInt(int64(rng.Intn(1 << 30))),
			Code:    codes[rng.Intn(len(codes))],
		}
		if rng.Intn(2) == 0 {
			acct.Storage = make(map[common.Hash]common.Hash)
			for j := 0; j < rng.Intn(6); j++ {
				var k common.Hash
				// Half the slots below the header boundary, half above.
				if rng.Intn(2) == 0 {
					k = common.BigToHash(big.NewInt(int64(rng.Intn(64))))
				} else {
					k = common.BigToHash(big.NewInt(int64(64 + rng.Intn(4096))))
				}
				acct.Storage[k] = common.BigToHash(big.NewInt(int64(rng.Intn(1<<30) + 1)))
			}
		}
		// The genesis writer refuses entirely absent accounts.
		if acct.Nonce == 0 && acct.Balance.Sign() == 0 && acct.Code == nil && len(acct.Storage) == 0 {
			acct.Nonce = 1
		}
		alloc[addr] = acct
	}
	// Shapes the draw above may miss: a code of exactly one full 256-leaf
	// group, a code whose only chunk is zero, and a delegation with header
	// and overflow storage.
	alloc[common.Address{0xed, 1}] = types.Account{Nonce: 1, Code: bytes.Repeat([]byte{0x5b}, 256*31)}
	alloc[common.Address{0xed, 2}] = types.Account{Nonce: 1, Code: []byte{0x00}}
	alloc[common.Address{0xed, 3}] = types.Account{
		Nonce:   1,
		Code:    types.AddressToDelegation(common.Address{0xde, 0x1e}),
		Storage: map[common.Hash]common.Hash{{31: 1}: {31: 1}, {30: 1}: {31: 2}},
	}
	return alloc
}

// assertFlatStateMatchesAlloc requires the flat store to be byte-exact, in
// both directions, against what a replaying node would hold for alloc.
func assertFlatStateMatchesAlloc(t *testing.T, chaindb ethdb.Database, alloc types.GenesisAlloc) {
	t.Helper()
	var (
		wantAccounts = make(map[common.Hash][]byte, len(alloc))
		wantSlots    = make(map[string][]byte)
	)
	for addr, acct := range alloc {
		balance, _ := uint256.FromBig(acct.Balance)
		wantAccounts[crypto.Keccak256Hash(addr.Bytes())] = types.SlimAccountRLP(types.StateAccount{
			Nonce:    acct.Nonce,
			Balance:  balance,
			Root:     types.EmptyRootHash,
			CodeHash: crypto.Keccak256(acct.Code),
		})
		for slot, val := range acct.Storage {
			blob, err := rlp.EncodeToBytes(common.TrimLeftZeroes(val[:]))
			if err != nil {
				t.Fatal(err)
			}
			key := string(crypto.Keccak256(addr.Bytes())) + string(crypto.Keccak256(slot[:]))
			wantSlots[key] = blob
		}
	}
	pbtdb := rawdb.NewTable(chaindb, string(rawdb.PBTPrefix))

	acctIt := pbtdb.NewIterator(rawdb.SnapshotAccountPrefix, nil)
	defer acctIt.Release()
	for acctIt.Next() {
		hash := common.BytesToHash(acctIt.Key()[len(rawdb.SnapshotAccountPrefix):])
		want, ok := wantAccounts[hash]
		if !ok {
			t.Fatalf("flat account %x has no counterpart in the allocation", hash)
		}
		if !bytes.Equal(acctIt.Value(), want) {
			t.Fatalf("flat account %x is %x, a replaying node writes %x", hash, acctIt.Value(), want)
		}
		delete(wantAccounts, hash)
	}
	if err := acctIt.Error(); err != nil {
		t.Fatal(err)
	}
	if len(wantAccounts) != 0 {
		t.Fatalf("%d accounts converted but missing from the flat store", len(wantAccounts))
	}

	slotIt := pbtdb.NewIterator(rawdb.SnapshotStoragePrefix, nil)
	defer slotIt.Release()
	for slotIt.Next() {
		key := string(slotIt.Key()[len(rawdb.SnapshotStoragePrefix):])
		want, ok := wantSlots[key]
		if !ok {
			t.Fatalf("flat slot %x has no counterpart in the allocation", slotIt.Key())
		}
		if !bytes.Equal(slotIt.Value(), want) {
			t.Fatalf("flat slot %x is %x, a replaying node writes %x", slotIt.Key(), slotIt.Value(), want)
		}
		delete(wantSlots, key)
	}
	if err := slotIt.Error(); err != nil {
		t.Fatal(err)
	}
	if len(wantSlots) != 0 {
		t.Fatalf("%d slots converted but missing from the flat store", len(wantSlots))
	}
}
