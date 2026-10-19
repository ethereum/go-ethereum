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
	"cmp"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

// convertState converts the MPT state at root in place, the self-converter's
// way, over a trie database, attested at opts.anchor or else at a header
// holding only root. With output paths in opts it first exports the
// artifacts, as a producer would, and holds both runs to one root.
func convertState(ctx context.Context, chaindb ethdb.Database, srcTriedb *triedb.Database, root common.Hash, opts conversionOptions) (common.Hash, error) {
	var exported common.Hash
	if opts.snapshotPath != "" || opts.preimagePath != "" {
		out := opts
		if out.snapshotPath == "" || out.preimagePath == "" {
			dir, err := os.MkdirTemp(opts.tmpDir, "convert-artifacts-")
			if err != nil {
				return common.Hash{}, err
			}
			defer os.RemoveAll(dir)
			out.snapshotPath = cmp.Or(out.snapshotPath, filepath.Join(dir, "snapshot"))
			out.preimagePath = cmp.Or(out.preimagePath, filepath.Join(dir, "preimages"))
		}
		var err error
		if exported, err = exportArtifacts(ctx, chaindb, srcTriedb, root, out); err != nil {
			return common.Hash{}, err
		}
	}
	plan, file, err := triedbPlan(srcTriedb, root, opts)
	if err != nil {
		return common.Hash{}, err
	}
	defer file.Close()
	if opts.anchor == nil {
		opts.anchor = &types.Header{Number: new(big.Int), Root: root}
	}
	converted, err := convertPlan(ctx, chaindb, plan, nil, opts)
	if err == nil && exported != (common.Hash{}) && converted != exported {
		return common.Hash{}, fmt.Errorf("export built %x, the conversion %x", exported, converted)
	}
	return converted, err
}

func TestBintrieConvert(t *testing.T) {
	var (
		addr1    = common.HexToAddress("0x1111111111111111111111111111111111111111")
		addr2    = common.HexToAddress("0x2222222222222222222222222222222222222222")
		slotKey1 = common.HexToHash("0x01")
		slotKey2 = common.HexToHash("0x02")
		slotVal1 = common.HexToHash("0xdeadbeef")
		slotVal2 = common.HexToHash("0xcafebabe")
		code     = []byte{0x60, 0x42, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3}
	)

	chaindb := rawdb.NewMemoryDatabase()

	srcTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.Defaults,
	})

	gspec := &core.Genesis{
		Config:  params.TestChainConfig,
		BaseFee: big.NewInt(params.InitialBaseFee),
		Alloc: types.GenesisAlloc{
			addr1: {
				Balance: big.NewInt(1000000),
				Nonce:   5,
			},
			addr2: {
				Balance: big.NewInt(2000000),
				Nonce:   10,
				Code:    code,
				Storage: map[common.Hash]common.Hash{
					slotKey1: slotVal1,
					slotKey2: slotVal2,
				},
			},
		},
	}

	genesisBlock := gspec.MustCommit(chaindb, srcTriedb)
	root := genesisBlock.Root()
	t.Logf("Genesis root: %x", root)
	srcTriedb.Close()

	srcTriedb2 := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.ReadOnly,
	})
	defer srcTriedb2.Close()

	currentRoot, err := convertState(context.Background(), chaindb, srcTriedb2, root, conversionOptions{})
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	t.Logf("Binary trie root: %x", currentRoot)

	// The conversion finalized the namespace on disk; a database opened over
	// it afterwards - the way a converted node starts - must pick the root up
	// from there.
	destTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
		IsPBT:  true,
		PathDB: pathdb.Defaults,
	})
	defer destTriedb.Close()

	bt2, err := bintrie.NewBinaryTrie(currentRoot, destTriedb)
	if err != nil {
		t.Fatalf("failed to reload binary trie: %v", err)
	}

	acc1, err := bt2.GetAccount(addr1)
	if err != nil {
		t.Fatalf("failed to get account1: %v", err)
	}
	if acc1 == nil {
		t.Fatal("account1 not found in binary trie")
	}
	if acc1.Nonce != 5 {
		t.Errorf("account1 nonce: got %d, want 5", acc1.Nonce)
	}
	wantBal1 := uint256.NewInt(1000000)
	if acc1.Balance.Cmp(wantBal1) != 0 {
		t.Errorf("account1 balance: got %s, want %s", acc1.Balance, wantBal1)
	}

	acc2, err := bt2.GetAccount(addr2)
	if err != nil {
		t.Fatalf("failed to get account2: %v", err)
	}
	if acc2 == nil {
		t.Fatal("account2 not found in binary trie")
	}
	if acc2.Nonce != 10 {
		t.Errorf("account2 nonce: got %d, want 10", acc2.Nonce)
	}
	wantBal2 := uint256.NewInt(2000000)
	if acc2.Balance.Cmp(wantBal2) != 0 {
		t.Errorf("account2 balance: got %s, want %s", acc2.Balance, wantBal2)
	}

	treeKey1 := bintrie.StorageSlotKey(addr2, slotKey1[:])
	val1, err := bt2.GetStemValue(treeKey1)
	if err != nil {
		t.Fatalf("failed to get storage slot1: %v", err)
	}
	if len(val1) == 0 {
		t.Fatal("storage slot1 not found")
	}
	got1 := common.BytesToHash(val1)
	if got1 != slotVal1 {
		t.Errorf("storage slot1: got %x, want %x", got1, slotVal1)
	}

	treeKey2 := bintrie.StorageSlotKey(addr2, slotKey2[:])
	val2, err := bt2.GetStemValue(treeKey2)
	if err != nil {
		t.Fatalf("failed to get storage slot2: %v", err)
	}
	if len(val2) == 0 {
		t.Fatal("storage slot2 not found")
	}
	got2 := common.BytesToHash(val2)
	if got2 != slotVal2 {
		t.Errorf("storage slot2: got %x, want %x", got2, slotVal2)
	}

	// Everything above reads the trie directly, which is not how a node reads
	// state. Read the same accounts back the way block processing does.
	assertConvertedStateReadable(t, chaindb, destTriedb, currentRoot, addr1, addr2, slotKey1, slotVal1)
}

// TestPBTDiskIsDetectable pins what MakeTrieDatabase's guard rests on: a
// binary tree database is recognisable from disk alone, and a merkle one is
// never mistaken for it.
func TestPBTDiskIsDetectable(t *testing.T) {
	merkle := rawdb.NewMemoryDatabase()
	mtdb := triedb.NewDatabase(merkle, &triedb.Config{PathDB: pathdb.Defaults})
	mtdb.Close()
	if rawdb.HasPBTState(merkle) {
		t.Fatal("a merkle-patricia database is marked as holding a binary tree")
	}

	binary := rawdb.NewMemoryDatabase()
	btdb := triedb.NewDatabase(binary, &triedb.Config{IsPBT: true, PathDB: pathdb.Defaults})
	btdb.Close()
	if !rawdb.HasPBTState(binary) {
		t.Fatal("a binary tree database is not detectable from disk; the guard can never fire")
	}
}

// assertConvertedStateReadable reads through the state reader a node uses:
// flat-first, where a miss is authoritative absence, so an empty flat store
// reads as empty while direct trie assertions still pass.
func assertConvertedStateReadable(t *testing.T, chaindb ethdb.Database, destTriedb *triedb.Database, root common.Hash, addr1, addr2 common.Address, slotKey, slotVal common.Hash) {
	t.Helper()

	statedb, err := state.New(root, state.NewPBTDatabase(destTriedb, state.NewCodeDB(chaindb)))
	if err != nil {
		t.Fatalf("failed to open the converted state: %v", err)
	}
	if got := statedb.GetNonce(addr1); got != 5 {
		t.Errorf("account1 nonce through the state reader: got %d, want 5", got)
	}
	if got := statedb.GetBalance(addr1).ToBig(); got.Cmp(big.NewInt(1000000)) != 0 {
		t.Errorf("account1 balance through the state reader: got %s, want 1000000", got)
	}
	if got := statedb.GetNonce(addr2); got != 10 {
		t.Errorf("account2 nonce through the state reader: got %d, want 10", got)
	}
	if got := statedb.GetState(addr2, slotKey); got != slotVal {
		t.Errorf("account2 slot through the state reader: got %x, want %x", got, slotVal)
	}
	// A flat-store miss is authoritative absence.
	absent := common.HexToAddress("0x00000000000000000000000000000000deadbeef")
	if got := statedb.GetBalance(absent); !got.IsZero() {
		t.Errorf("absent account has balance %s through the state reader", got)
	}
	if got := statedb.GetNonce(absent); got != 0 {
		t.Errorf("absent account has nonce %d through the state reader", got)
	}
	if got := statedb.GetState(addr1, common.HexToHash("0x77")); got != (common.Hash{}) {
		t.Errorf("absent slot reads %x through the state reader", got)
	}
}

// TestConvertRefusesDirtyNamespace pins the crash story: a dead run leaves
// namespace keys but no marker, and the converter must refuse that debris
// until --force wipes it. The open-time refusal is pinned in triedb/pathdb.
func TestConvertRefusesDirtyNamespace(t *testing.T) {
	addr1 := common.HexToAddress("0x4444444444444444444444444444444444444444")

	chaindb := rawdb.NewMemoryDatabase()
	srcTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.Defaults,
	})
	gspec := &core.Genesis{
		Config:  params.TestChainConfig,
		BaseFee: big.NewInt(params.InitialBaseFee),
		Alloc: types.GenesisAlloc{
			addr1: {Balance: big.NewInt(1000000), Nonce: 1},
		},
	}
	genesis := gspec.MustCommit(chaindb, srcTriedb)
	root := genesis.Root()
	srcTriedb.Close()

	src := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.ReadOnly,
	})
	defer src.Close()

	// Scan-phase debris: one flat record, nothing else.
	pbtdb := rawdb.NewTable(chaindb, string(rawdb.PBTPrefix))
	rawdb.WriteAccountSnapshot(pbtdb, common.Hash{0x01}, []byte{0x01})
	if dirty, err := hasBinaryTrieState(chaindb); err != nil {
		t.Fatalf("namespace probe failed: %v", err)
	} else if !dirty {
		t.Fatal("conversion debris is invisible to the namespace probe")
	}
	if _, err := convertState(context.Background(), chaindb, src, root, conversionOptions{}); err == nil {
		t.Fatal("conversion ran over the debris of a previous run")
	}

	// --force wipes; conversion must then run clean.
	if err := wipeBinaryTrieState(chaindb, ""); err != nil {
		t.Fatalf("wipe failed: %v", err)
	}
	if _, err := convertState(context.Background(), chaindb, src, root, conversionOptions{}); err != nil {
		t.Fatalf("conversion failed after wipe: %v", err)
	}
	if got := rawdb.ReadAccountSnapshot(pbtdb, common.Hash{0x01}); len(got) != 0 {
		t.Fatal("the wipe left the debris record in place")
	}
	// Chain data shares the prefix; the wipe must not touch it.
	if !rawdb.HasBody(chaindb, genesis.Hash(), 0) {
		t.Fatal("the wipe deleted the genesis block body along with the tree")
	}

	// A completed conversion also refuses without --force.
	if _, err := convertState(context.Background(), chaindb, src, root, conversionOptions{}); err == nil {
		t.Fatal("conversion ran over a completed conversion")
	}
}

// tamperDB passes every write through except the first record of one key
// family, which it drops, corrupts or doubles with a stray neighbour: a store
// that loses or mangles a write on its way in.
type tamperDB struct {
	ethdb.Database
	family []byte // full key prefix, the namespace's table prefix included
	mode   string // "drop", "corrupt" or "stray"
	done   atomic.Bool
}

func newTamperDB(db ethdb.Database, family []byte, mode string) *tamperDB {
	return &tamperDB{Database: db, family: append(common.CopyBytes(rawdb.PBTPrefix), family...), mode: mode}
}

func (db *tamperDB) NewBatch() ethdb.Batch { return tamperBatch{db.Database.NewBatch(), db} }

type tamperBatch struct {
	ethdb.Batch
	db *tamperDB
}

func (b tamperBatch) Put(key, value []byte) error {
	if !bytes.HasPrefix(key, b.db.family) || !b.db.done.CompareAndSwap(false, true) {
		return b.Batch.Put(key, value)
	}
	switch b.db.mode {
	case "drop":
		return nil
	case "corrupt":
		value = append(common.CopyBytes(value[:len(value)-1]), value[len(value)-1]^0xff)
	case "stray":
		if err := b.Batch.Put(append(common.CopyBytes(key), 0xff), value); err != nil {
			return err
		}
	}
	return b.Batch.Put(key, value)
}

// TestWrittenStateCatchesBadWrites pins both commands' read-back check: a
// record of any family that the store drops, corrupts or doubles on its way
// in fails the run, naming the family, and nothing is attested.
func TestWrittenStateCatchesBadWrites(t *testing.T) {
	_, root, _, snapPath, prePath := importFixture(t, artifactAlloc())
	caught := func(t *testing.T, err error, chaindb ethdb.Database, family string) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), family) {
			t.Fatalf("a bad %s write went through; err = %v", family, err)
		}
		if rawdb.HasPBTState(chaindb) {
			t.Fatal("a run with a bad write attested its namespace")
		}
	}
	for _, f := range []struct {
		name   string
		prefix []byte
	}{
		{"tree node", rawdb.TrieNodeAccountPrefix},
		{"flat account", rawdb.SnapshotAccountPrefix},
		{"flat storage", rawdb.SnapshotStoragePrefix},
	} {
		for _, mode := range []string{"drop", "corrupt", "stray"} {
			t.Run("convert/"+f.name+"/"+mode, func(t *testing.T) {
				chaindb := rawdb.NewMemoryDatabase()
				srcTriedb := triedb.NewDatabase(chaindb, &triedb.Config{Preimages: true, PathDB: pathdb.Defaults})
				gspec := &core.Genesis{Config: params.TestChainConfig, BaseFee: big.NewInt(params.InitialBaseFee), Alloc: artifactAlloc()}
				root := gspec.MustCommit(chaindb, srcTriedb).Root()
				srcTriedb.Close()
				src := triedb.NewDatabase(chaindb, &triedb.Config{Preimages: true, PathDB: pathdb.ReadOnly})
				defer src.Close()
				_, err := convertState(context.Background(), newTamperDB(chaindb, f.prefix, mode), src, root, conversionOptions{tmpDir: t.TempDir()})
				caught(t, err, chaindb, f.name)
			})
			t.Run("import/"+f.name+"/"+mode, func(t *testing.T) {
				chaindb := rawdb.NewMemoryDatabase()
				_, err := importState(context.Background(), newTamperDB(chaindb, f.prefix, mode), importOptions{snapshot: snapPath, preimages: prePath,
					anchor: &types.Header{Number: big.NewInt(1), Root: root}, conversionOptions: conversionOptions{tmpDir: t.TempDir()}})
				caught(t, err, chaindb, f.name)
			})
		}
	}
}

// TestWriteDigestSpansBatches pins the digest across its batch boundaries: a
// changed record in any batch changes the sum, and keys that stop ascending
// fail it, at a batch's first record as well as inside one.
func TestWriteDigestSpansBatches(t *testing.T) {
	n := 2*recordBatchSize + 5
	pool := newDigestPool(4)
	defer pool.close()
	digest := func(key func(i int) []byte, value func(i int) []byte) (*writeDigest, error) {
		d := newWriteDigest(pool)
		a := d.ahead()
		for i := range n {
			a.add(key(i), value(i))
		}
		return d, a.finish()
	}
	key := func(i int) []byte { return binary.BigEndian.AppendUint32(nil, uint32(i)) }
	value := func(i int) []byte { return []byte{byte(i)} }
	base, err := digest(key, value)
	if err != nil || base.count != uint64(n) {
		t.Fatalf("ascending records: count %d, err %v", base.count, err)
	}
	for _, at := range []int{0, recordBatchSize, n - 1} {
		changed, err := digest(key, func(i int) []byte {
			if i == at {
				return []byte{byte(i) + 1}
			}
			return value(i)
		})
		if err != nil || changed.sum() == base.sum() {
			t.Fatalf("record %d changed: err %v, sum unchanged %v", at, err, changed.sum() == base.sum())
		}
	}
	for _, at := range []int{1, recordBatchSize, recordBatchSize + 1} {
		_, err := digest(func(i int) []byte {
			if i == at {
				return key(i - 1)
			}
			return key(i)
		}, value)
		if err == nil || !strings.Contains(err.Error(), "out of key order") {
			t.Fatalf("record %d repeating its predecessor's key: err %v", at, err)
		}
	}
}

// TestConvertFoldsSparseStorage: storage spread thin, most of a fold batch's
// accounts holding none, still folds to the source root; the account trie
// never waits on storage roots no worker was handed.
func TestConvertFoldsSparseStorage(t *testing.T) {
	defer func(n int) { foldBatchSize = n }(foldBatchSize)
	foldBatchSize = 4
	alloc := make(types.GenesisAlloc)
	for i := range 300 {
		var addr common.Address
		binary.BigEndian.PutUint64(addr[12:], uint64(i)+1)
		acct := types.Account{Balance: big.NewInt(1)}
		if i%60 == 0 {
			acct.Storage = map[common.Hash]common.Hash{{31: 1}: {31: 1}}
		}
		alloc[addr] = acct
	}
	if _, _, _, err := convertGenesis(t, alloc, true, conversionOptions{workers: 4}); err != nil {
		t.Fatal(err)
	}
}

// TestConvertRejectsCorruptCode: the code store has no integrity of its own,
// so code whose bytes do not hash to its key must fail the dual-check rather
// than convert into wrong code leaves.
func TestConvertRejectsCorruptCode(t *testing.T) {
	alloc := artifactAlloc()
	chaindb := rawdb.NewMemoryDatabase()
	srcTriedb := triedb.NewDatabase(chaindb, &triedb.Config{Preimages: true, PathDB: pathdb.Defaults})
	gspec := &core.Genesis{Config: params.TestChainConfig, BaseFee: big.NewInt(params.InitialBaseFee), Alloc: alloc}
	root := gspec.MustCommit(chaindb, srcTriedb).Root()
	srcTriedb.Close()
	for _, a := range alloc {
		if _, delegated := types.ParseDelegation(a.Code); len(a.Code) > 0 && !delegated {
			code := bytes.Clone(a.Code)
			code[len(code)-1] ^= 0xff
			rawdb.WriteCode(chaindb, crypto.Keccak256Hash(a.Code), code)
			break
		}
	}
	src := triedb.NewDatabase(chaindb, &triedb.Config{Preimages: true, PathDB: pathdb.ReadOnly})
	defer src.Close()
	if _, err := convertState(context.Background(), chaindb, src, root, conversionOptions{tmpDir: t.TempDir()}); err == nil || !strings.Contains(err.Error(), "assembled code hashes to") {
		t.Fatalf("converted corrupt code; err = %v", err)
	}
	if rawdb.HasPBTState(chaindb) {
		t.Fatal("a conversion of corrupt code attested its namespace")
	}
}

// TestWipeRestoresVirginNamespace: the wipe restores the exact
// pre-conversion key set (pinning rawdb.PBTKeyFamilies against drift) and
// re-conversion reproduces identical root and artifacts.
func TestWipeRestoresVirginNamespace(t *testing.T) {
	chaindb := rawdb.NewMemoryDatabase()
	srcTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.Defaults,
	})
	gspec := &core.Genesis{
		Config:  params.TestChainConfig,
		BaseFee: big.NewInt(params.InitialBaseFee),
		Alloc:   artifactAlloc(),
	}
	root := gspec.MustCommit(chaindb, srcTriedb).Root()
	srcTriedb.Close()

	// Pre-conversion prefix keys are chain data the wipe must not touch.
	prefixKeys := func() map[string]struct{} {
		keys := make(map[string]struct{})
		it := chaindb.NewIterator(rawdb.PBTPrefix, nil)
		defer it.Release()
		for it.Next() {
			keys[string(it.Key())] = struct{}{}
		}
		return keys
	}
	before := prefixKeys()

	src := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.ReadOnly,
	})
	defer src.Close()

	convert := func(dir string) (common.Hash, []byte, []byte) {
		t.Helper()
		binRoot, err := convertState(context.Background(), chaindb, src, root, conversionOptions{
			tmpDir:       dir,
			snapshotPath: filepath.Join(dir, "snapshot.bin"),
			preimagePath: filepath.Join(dir, "preimages.bin"),
		})
		if err != nil {
			t.Fatalf("conversion failed: %v", err)
		}
		snap, err := os.ReadFile(filepath.Join(dir, "snapshot.bin"))
		if err != nil {
			t.Fatal(err)
		}
		pre, err := os.ReadFile(filepath.Join(dir, "preimages.bin"))
		if err != nil {
			t.Fatal(err)
		}
		return binRoot, snap, pre
	}
	root1, snap1, pre1 := convert(t.TempDir())

	if err := wipeBinaryTrieState(chaindb, ""); err != nil {
		t.Fatalf("wipe failed: %v", err)
	}
	after := prefixKeys()
	if len(after) != len(before) {
		t.Fatalf("wipe left %d keys under the namespace prefix, started with %d", len(after), len(before))
	}
	for key := range after {
		if _, ok := before[key]; !ok {
			t.Fatalf("wipe left a converted key behind: %x", key)
		}
	}

	root2, snap2, pre2 := convert(t.TempDir())
	if root1 != root2 {
		t.Fatalf("re-conversion produced root %x, first run %x", root2, root1)
	}
	if !bytes.Equal(snap1, snap2) {
		t.Fatal("re-conversion produced a different snapshot artifact")
	}
	if !bytes.Equal(pre1, pre2) {
		t.Fatal("re-conversion produced a different preimage file")
	}
}

// TestWipeDropsAnchorFirst: a wipe killed after the cursors go must not leave
// an anchor the follower would bind to the state left behind.
func TestWipeDropsAnchorFirst(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	pbtdb := rawdb.NewTable(db, string(rawdb.PBTPrefix))
	rawdb.WritePBTAnchor(pbtdb, 0, common.Hash{0x01})
	if err := wipeBinaryTrieState(refusedBatchDB{db}, ""); err == nil {
		t.Fatal("wipe succeeded through refused batches")
	}
	if _, _, ok := rawdb.ReadPBTAnchor(pbtdb); ok {
		t.Fatal("anchor outlived the refused wipe")
	}
}

// refusedBatchDB fails every batch write, leaving direct writes alone.
type refusedBatchDB struct{ ethdb.Database }

func (db refusedBatchDB) NewBatch() ethdb.Batch { return refusedBatch{db.Database.NewBatch()} }

type refusedBatch struct{ ethdb.Batch }

func (refusedBatch) Write() error { return errors.New("batch refused") }

// TestConvertCorruptPreimageRefused: a corrupt preimage store entry must
// abort the conversion with nothing surviving - no artifacts, no
// attestation.
func TestConvertCorruptPreimageRefused(t *testing.T) {
	newFixture := func(t *testing.T) (ethdb.Database, *triedb.Database, common.Hash) {
		t.Helper()
		chaindb := rawdb.NewMemoryDatabase()
		srcTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
			Preimages: true,
			PathDB:    pathdb.Defaults,
		})
		gspec := &core.Genesis{
			Config:  params.TestChainConfig,
			BaseFee: big.NewInt(params.InitialBaseFee),
			Alloc:   artifactAlloc(),
		}
		root := gspec.MustCommit(chaindb, srcTriedb).Root()
		srcTriedb.Close()
		src := triedb.NewDatabase(chaindb, &triedb.Config{
			Preimages: true,
			PathDB:    pathdb.ReadOnly,
		})
		t.Cleanup(func() { src.Close() })
		return chaindb, src, root
	}
	convert := func(t *testing.T, chaindb ethdb.Database, src *triedb.Database, root common.Hash) (error, string, string) {
		t.Helper()
		dir := t.TempDir()
		snapPath, prePath := filepath.Join(dir, "snapshot.bin"), filepath.Join(dir, "preimages.bin")
		_, err := convertState(context.Background(), chaindb, src, root, conversionOptions{
			tmpDir:       dir,
			snapshotPath: snapPath,
			preimagePath: prePath,
		})
		return err, snapPath, prePath
	}
	t.Run("wrong account preimage", func(t *testing.T) {
		chaindb, src, root := newFixture(t)
		// Well-formed length, wrong value.
		addr := common.HexToAddress("0x1000000000000000000000000000000000000001")
		rawdb.WritePreimages(chaindb, map[common.Hash][]byte{
			crypto.Keccak256Hash(addr.Bytes()): common.HexToAddress("0x9999999999999999999999999999999999999999").Bytes(),
		})
		err, snapPath, prePath := convert(t, chaindb, src, root)
		if err == nil || !strings.Contains(err.Error(), "corrupt preimage") {
			t.Fatalf("a corrupt account preimage converted; err = %v", err)
		}
		for _, path := range []string{snapPath, prePath} {
			if _, statErr := os.Stat(path); statErr == nil {
				t.Fatalf("a refused conversion left the artifact %s behind", path)
			}
		}
		if rawdb.HasPBTState(chaindb) {
			t.Fatal("a refused conversion attested its namespace")
		}
	})

	t.Run("wrong-length slot preimage", func(t *testing.T) {
		chaindb, src, root := newFixture(t)
		// 33 bytes: used to panic in the key deriver.
		slot := common.BigToHash(big.NewInt(1))
		rawdb.WritePreimages(chaindb, map[common.Hash][]byte{
			crypto.Keccak256Hash(slot[:]): make([]byte, 33),
		})
		err, snapPath, _ := convert(t, chaindb, src, root)
		if err == nil || !strings.Contains(err.Error(), "corrupt preimage") {
			t.Fatalf("a wrong-length slot preimage converted; err = %v", err)
		}
		if _, statErr := os.Stat(snapPath); statErr == nil {
			t.Fatal("a refused conversion left the snapshot artifact behind")
		}
		if rawdb.HasPBTState(chaindb) {
			t.Fatal("a refused conversion attested its namespace")
		}
	})
}

// raw-namespace cursors and records must not outlive the state they name.

// TestConvertDetectsLostRecords pins the check that ties a conversion's
// output to its input. The converter's dual-check joins its own leaves to
// candidates from the same scan, so a record the scan loses goes missing from
// both and the join stays consistent: only check 2, re-deriving the source
// root from the leaves, can catch it.
func TestConvertDetectsLostRecords(t *testing.T) {
	alloc := artifactAlloc()

	chaindb := rawdb.NewMemoryDatabase()
	srcTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.Defaults,
	})
	gspec := &core.Genesis{
		Config:  params.TestChainConfig,
		BaseFee: big.NewInt(params.InitialBaseFee),
		Alloc:   alloc,
	}
	root := gspec.MustCommit(chaindb, srcTriedb).Root()
	srcTriedb.Close()

	// scan derives the leaves and candidates deriveLeaves would, minus what
	// drop rejects: the account for (addr, nil), a slot for (addr, &key).
	scan := func(t *testing.T, drop func(addr common.Address, slot *common.Hash) bool) *ownLeaves {
		t.Helper()
		leaves := bintrie.NewLeafSorter(t.TempDir(), 0)
		cands := bintrie.NewRecordSorter(t.TempDir(), 0, nil)
		t.Cleanup(func() {
			leaves.Close()
			cands.Close()
		})
		emit := func(key []byte, value [32]byte) error {
			if value == ([32]byte{}) {
				return nil
			}
			return leaves.Add(key, value[:])
		}
		codes, stats := newCodeSet(), newStats("Scanning")
		for addr, account := range alloc {
			if drop != nil && drop(addr, nil) {
				continue
			}
			codeHash := types.EmptyCodeHash
			if len(account.Code) != 0 {
				codeHash = crypto.Keccak256Hash(account.Code)
			}
			balance := new(uint256.Int)
			if account.Balance != nil {
				balance = uint256.MustFromBig(account.Balance)
			}
			if err := emitAccountHeader(chaindb, addr, account.Nonce, balance, codeHash, codes, stats, emit); err != nil {
				t.Fatal(err)
			}
			c := accountCandidates{cand: cands, buf: new(candidateBuf), addr: addr}
			for key, value := range account.Storage {
				if (drop != nil && drop(addr, &key)) || value == (common.Hash{}) {
					continue
				}
				if err := emit(bintrie.StorageSlotKey(addr, key[:]), value); err != nil {
					t.Fatal(err)
				}
				if err := c.slot(key); err != nil {
					t.Fatal(err)
				}
			}
			if err := c.finish(); err != nil {
				t.Fatal(err)
			}
		}
		stream, err := leaves.Sort()
		if err != nil {
			t.Fatal(err)
		}
		return &ownLeaves{leaves: stream, candidates: cands}
	}
	convert := func(t *testing.T, own *ownLeaves) (ethdb.Database, error) {
		db := rawdb.NewMemoryDatabase()
		_, err := importState(context.Background(), db, importOptions{own: own,
			anchor:            &types.Header{Number: big.NewInt(1), Root: root},
			conversionOptions: conversionOptions{tmpDir: t.TempDir()}})
		return db, err
	}

	// The account holding storage, so both loss cases land on it.
	var (
		victim   = common.HexToAddress("0x2000000000000000000000000000000000000002")
		slotOne  = common.BigToHash(big.NewInt(1))
		mismatch = "re-derive merkle root"
	)
	if _, ok := alloc[victim]; !ok {
		t.Fatalf("fixture no longer holds %x", victim)
	}

	t.Run("complete set", func(t *testing.T) {
		db, err := convert(t, scan(t, nil))
		if err != nil {
			t.Fatalf("the complete leaf set failed the dual-check: %v", err)
		}
		if !rawdb.HasPBTState(db) {
			t.Fatal("a passing conversion attested nothing")
		}
	})

	t.Run("lost account", func(t *testing.T) {
		db, err := convert(t, scan(t, func(addr common.Address, _ *common.Hash) bool {
			return addr == victim
		}))
		if err == nil || !strings.Contains(err.Error(), mismatch) {
			t.Fatalf("a dropped account survived check 2; err = %v", err)
		}
		if rawdb.HasPBTState(db) {
			t.Fatal("a refused conversion attested its namespace")
		}
	})

	t.Run("lost slot", func(t *testing.T) {
		db, err := convert(t, scan(t, func(addr common.Address, slot *common.Hash) bool {
			return addr == victim && slot != nil && *slot == slotOne
		}))
		if err == nil || !strings.Contains(err.Error(), mismatch) {
			t.Fatalf("a dropped storage slot survived check 2; err = %v", err)
		}
		if rawdb.HasPBTState(db) {
			t.Fatal("a refused conversion attested its namespace")
		}
	})
}

// TestConvertYieldsToBlocks: the self-converter's scan makes no progress while
// the node processes a block, even over accounts without storage, and a
// conversion pausing between accounts and after every slot lands on the root
// and namespace of one that never pauses.
func TestConvertYieldsToBlocks(t *testing.T) {
	defer func(n int) { scanPauseSlots = n }(scanPauseSlots)
	scanPauseSlots = 1

	source := func(alloc types.GenesisAlloc) (ethdb.Database, *triedb.Database, common.Hash) {
		chaindb := rawdb.NewMemoryDatabase()
		tdb := triedb.NewDatabase(chaindb, &triedb.Config{Preimages: true, PathDB: pathdb.Defaults})
		gspec := &core.Genesis{Config: params.TestChainConfig, BaseFee: big.NewInt(params.InitialBaseFee), Alloc: alloc}
		root := gspec.MustCommit(chaindb, tdb).Root()
		tdb.Close()
		src := triedb.NewDatabase(chaindb, &triedb.Config{Preimages: true, PathDB: pathdb.ReadOnly})
		t.Cleanup(func() { src.Close() })
		return chaindb, src, root
	}
	options := func(root common.Hash) conversionOptions {
		return conversionOptions{tmpDir: t.TempDir(), sortBudget: 512, workers: 4, anchor: &types.Header{Number: new(big.Int), Root: root}}
	}

	t.Run("scan waits through a block", func(t *testing.T) {
		eoas := make(types.GenesisAlloc)
		for i := range 16 {
			eoas[common.Address{byte(i + 1)}] = types.Account{Balance: big.NewInt(int64(i + 1))}
		}
		chaindb, src, root := source(eoas)
		opts := options(root)
		plan, file, err := triedbPlan(src, root, opts)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		y := newBlockYield()
		y.set(true)
		done := make(chan error, 1)
		go func() {
			_, _, closeScan, err := scanState(context.Background(), chaindb, plan, nil, true, opts, newStats("Converting state"), y)
			if err == nil {
				closeScan()
			}
			done <- err
		}()
		select {
		case err := <-done:
			t.Fatalf("the scan ran through a block: %v", err)
		case <-time.After(100 * time.Millisecond):
		}
		y.set(false)
		if err := <-done; err != nil {
			t.Fatalf("scan after the block: %v", err)
		}
	})

	t.Run("pausing changes nothing", func(t *testing.T) {
		convert := func(y *blockYield) (ethdb.Database, common.Hash) {
			chaindb, src, root := source(mixedAlloc(8347))
			opts := options(root)
			plan, file, err := triedbPlan(src, root, opts)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			got, err := convertPlan(context.Background(), chaindb, plan, y, opts)
			if err != nil {
				t.Fatalf("conversion failed: %v", err)
			}
			return chaindb, got
		}
		y := newBlockYield()
		stop := make(chan struct{})
		defer close(stop)
		go func() {
			for {
				select {
				case <-stop:
					return
				default:
				}
				y.set(true)
				time.Sleep(200 * time.Microsecond)
				y.set(false)
				time.Sleep(200 * time.Microsecond)
			}
		}()
		wantDB, want := convert(nil)
		gotDB, got := convert(y)
		if got != want {
			t.Fatalf("pausing conversion built %x, the uninterrupted one %x", got, want)
		}
		assertNamespaceEqual(t, wantDB, gotDB)
	})
}
