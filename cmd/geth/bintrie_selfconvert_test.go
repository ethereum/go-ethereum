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
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/pebble"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

// TestDriftKeys: hashes a scan at P meets that the preimage file made at A
// lacks resolve through the writes of A+1..P, and file keys the state at P
// lacks pass only as writes. Anything else fails, as against a file made at
// another state. Every case also runs with records read one slot key at a
// time, so the deleted contract's storage is read past across windows.
func TestDriftKeys(t *testing.T) {
	defer func(n int) { fileKeysWindow = n }(fileKeysWindow)
	type edit = func(*bal.ConstructionBlockAccessList)
	var (
		kept, created, deleted     = common.Address{1}, common.Address{2}, common.Address{3}
		keptSlot, newSlot, cleared = common.Hash{4}, common.Hash{5}, common.Hash{6}

		file  = []preRecord{{addr: kept, slots: []common.Hash{keptSlot, cleared}}, {addr: deleted, slots: []common.Hash{{7}, {8}}}}
		state = []preRecord{{addr: kept, slots: []common.Hash{keptSlot, newSlot}}, {addr: created}}

		create edit = func(l *bal.ConstructionBlockAccessList) { l.BalanceChange(0, created, uint256.NewInt(1)) }
		remove edit = func(l *bal.ConstructionBlockAccessList) { l.BalanceChange(0, deleted, new(uint256.Int)) }
		store  edit = func(l *bal.ConstructionBlockAccessList) { l.StorageWrite(0, kept, newSlot, common.Hash{1}) }
		clear  edit = func(l *bal.ConstructionBlockAccessList) { l.StorageWrite(0, kept, cleared, common.Hash{}) }
		read   edit = func(l *bal.ConstructionBlockAccessList) { l.AccountRead(deleted); l.StorageRead(kept, cleared) }
	)
	for _, tc := range []struct {
		name  string
		edits []edit
		want  string // error substring; empty resolves
	}{
		{"drift resolves", []edit{create, remove, store, clear}, ""},
		{"created account unwritten", []edit{remove, store, clear}, "missing preimage for account hash"},
		{"created slot unwritten", []edit{create, remove, clear}, "missing preimage for storage key"},
		{"deleted account unwritten", []edit{create, store, clear}, "which the state does not hold"},
		{"cleared slot unwritten", []edit{create, remove, store}, "which the state does not hold"},
		{"reads are not writes", []edit{create, store, read}, "which the state does not hold"},
	} {
		for _, window := range []int{1, 4096} {
			t.Run(fmt.Sprintf("%s/window=%d", tc.name, window), func(t *testing.T) {
				fileKeysWindow = window
				path := filepath.Join(t.TempDir(), "preimages")
				writePreimageRecords(t, path, file)
				list := bal.NewConstructionBlockAccessList()
				for _, e := range tc.edits {
					e(list)
				}
				f, ranges, pieces, err := splitPreimages(path, 1)
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				lists, err := accessListKeys(context.Background(), func(uint64) (*bal.BlockAccessList, error) {
					return list.ToEncodingObj(), nil
				}, 0, 1, ranges, t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				defer lists[0].close()
				keys, err := newDriftKeys(pieces[0], lists[0])
				if err != nil {
					t.Fatal(err)
				}
				err = resolveState(keys, state)
				if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
					t.Fatalf("err = %v, want %q", err, tc.want)
				}
			})
		}
	}
}

// resolveState asks keys for state's hashes in scan order, accounts and
// their slots ascending by hash, and checks what they resolve to.
func resolveState(keys keySource, state []preRecord) error {
	byHash := func(a, b []byte) int { return bytes.Compare(crypto.Keccak256(a), crypto.Keccak256(b)) }
	recs := slices.Clone(state)
	slices.SortFunc(recs, func(a, b preRecord) int { return byHash(a.addr[:], b.addr[:]) })
	for _, rec := range recs {
		addr, err := keys.account(crypto.Keccak256Hash(rec.addr[:]))
		if err != nil {
			return err
		}
		if addr != rec.addr {
			return fmt.Errorf("account resolved to %x, want %x", addr, rec.addr)
		}
		slots := slices.Clone(rec.slots)
		slices.SortFunc(slots, func(a, b common.Hash) int { return byHash(a[:], b[:]) })
		for _, slot := range slots {
			got, err := keys.slot(addr, crypto.Keccak256Hash(slot[:]))
			if err != nil {
				return err
			}
			if got != slot {
				return fmt.Errorf("slot resolved to %x, want %x", got, slot)
			}
		}
		if err := keys.done(addr); err != nil {
			return err
		}
	}
	return keys.close()
}

// TestCapturedStateOutlivesFlushes: the sources capturePersisted opens keep
// reading the state they captured, range by range, while a later flush
// rewrites the flat state under them.
func TestCapturedStateOutlivesFlushes(t *testing.T) {
	kv, err := pebble.New(t.TempDir(), 0, 0, "", false)
	if err != nil {
		t.Fatal(err)
	}
	db := rawdb.NewDatabase(kv)
	defer db.Close()

	type slot struct{ account, slot common.Hash }
	flush := func(id uint64, root common.Hash, accounts map[common.Hash][]byte, slots map[slot][]byte) {
		batch := db.NewBatch()
		for hash, data := range accounts {
			if data == nil {
				rawdb.DeleteAccountSnapshot(batch, hash)
			} else {
				rawdb.WriteAccountSnapshot(batch, hash, data)
			}
		}
		for key, data := range slots {
			if data == nil {
				rawdb.DeleteStorageSnapshot(batch, key.account, key.slot)
			} else {
				rawdb.WriteStorageSnapshot(batch, key.account, key.slot, data)
			}
		}
		rawdb.WritePersistentStateID(batch, id)
		rawdb.WriteSnapshotRoot(batch, root)
		if err := batch.Write(); err != nil {
			t.Fatal(err)
		}
	}
	a, b, c := common.Hash{0x10}, common.Hash{0x20}, common.Hash{0x30}
	flush(1, common.Hash{1},
		map[common.Hash][]byte{a: {0xa1}, b: {0xb1}},
		map[slot][]byte{{a, common.Hash{1}}: {0x01}, {b, common.Hash{2}}: {0x02}})

	ranges := []hashRange{{end: b}, {start: b}}
	root, sources, err := capturePersisted(db, ranges)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, s := range sources {
			s.release()
		}
	}()
	if root != (common.Hash{1}) {
		t.Fatalf("captured root %x, want the persisted one", root)
	}
	flush(2, common.Hash{2},
		map[common.Hash][]byte{a: nil, c: {0xc2}},
		map[slot][]byte{{a, common.Hash{1}}: nil, {b, common.Hash{2}}: {0x22}, {b, common.Hash{3}}: {0x03}})

	var got []string
	for i, r := range ranges {
		accts, _ := sources[i].accounts(r.start)
		for accts.Next() && !r.past(accts.Hash()) {
			got = append(got, fmt.Sprintf("%x=%x", accts.Hash().Bytes()[:1], accts.Account()))
			slots, _ := sources[i].storage(accts.Hash(), common.Hash{})
			for slots.Next() {
				got = append(got, fmt.Sprintf("%x/%x=%x", accts.Hash().Bytes()[:1], slots.Hash().Bytes()[:1], slots.Slot()))
			}
			if err := slots.Error(); err != nil {
				t.Fatal(err)
			}
		}
		if err := accts.Error(); err != nil {
			t.Fatal(err)
		}
	}
	if want := []string{"10=a1", "10/01=01", "20=b1", "20/02=02"}; !slices.Equal(got, want) {
		t.Fatalf("scanned %v, want the captured state %v", got, want)
	}
}

// driftChain is anchorTestChain's migrating chain with keys coming and
// going: every block stores a new slot in a contract and pays a fresh
// account, and block 3 clears one of the contract's genesis slots. It
// returns the contract too. No account goes: post-merge, none can that
// existed before the block deleting it (EIP-6780, EIP-7523).
func driftChain(t *testing.T, n int) (*core.Genesis, consensus.Engine, []*types.Block, *core.BlockChainConfig, common.Address) {
	t.Helper()
	gspec, engine, _, cfg := anchorTestChain(t, 0)
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291") // anchorTestChain's sender
	var (
		sender   = crypto.PubkeyToAddress(key.PublicKey)
		store    = common.Address{0x5e}
		signer   = types.LatestSigner(gspec.Config)
		gasPrice = big.NewInt(params.InitialBaseFee + 1)
	)
	gspec.Alloc[store] = types.Account{
		// SSTORE(calldata[0:32], calldata[32:64])
		Code:    []byte{0x60, 0x20, 0x35, 0x60, 0x00, 0x35, 0x55, 0x00},
		Storage: map[common.Hash]common.Hash{{1}: {0x11}, {2}: {0x22}},
		Balance: new(big.Int),
	}

	_, blocks, _ := core.GenerateChainWithGenesis(gspec, engine, n, func(i int, b *core.BlockGen) {
		send := func(to common.Address, value int64, data []byte) {
			b.AddTx(types.MustSignNewTx(key, signer, &types.LegacyTx{
				Nonce: b.TxNonce(sender), Gas: 100_000, GasPrice: gasPrice, To: &to, Value: big.NewInt(value), Data: data,
			}))
		}
		send(store, 0, append(common.BigToHash(big.NewInt(int64(0x10+i))).Bytes(), common.BigToHash(big.NewInt(int64(i+1))).Bytes()...))
		send(common.Address{0xf0, byte(i)}, 1, nil)
		if i == 2 {
			send(store, 0, append(common.Hash{1}.Bytes(), common.Hash{}.Bytes()...))
		}
	})
	return gspec, engine, blocks, cfg, store
}

// TestOnlineSelfConvert: a node following the chain converts its own state
// with a preimage file made at an earlier block A. It converts the block P
// it has persisted, resolving the keys that came and went since A through
// the access lists, attests P once finalized, and the follower takes the
// tree from P to the head, to the root a genesis-seeded follower produced.
// The scan consumes the preimage file, which a failed scan leaves whole.
// Restarting with the same flags converts nothing.
func TestOnlineSelfConvert(t *testing.T) {
	gspec, engine, blocks, cfg, store := driftChain(t, 8)
	var (
		a, p = blocks[1], blocks[4]
		head = blocks[len(blocks)-1]
	)
	ref := rawdb.NewMemoryDatabase()
	chain, err := core.NewBlockChain(ref, gspec, engine, cfg)
	if err != nil {
		t.Fatal(err)
	}
	insertBlocks(t, chain, blocks)
	want := awaitRecord(t, ref, head.NumberU64(), head.Hash())
	chain.Stop()

	// The preimage file at A holds the slot block 3 clears and none of the
	// keys blocks after A create.
	prePath := filepath.Join(t.TempDir(), "preimages")
	if err := wipeBinaryTrieState(ref, ""); err != nil {
		t.Fatal(err)
	}
	src := triedb.NewDatabase(ref, &triedb.Config{Preimages: true, PathDB: pathdb.ReadOnly})
	if _, err := convertState(context.Background(), ref, src, a.Root(), conversionOptions{preimagePath: prePath}); err != nil {
		t.Fatal(err)
	}
	src.Close()
	var (
		named      = make(map[common.Address]bool)
		storeSlots []common.Hash
	)
	for _, rec := range readPreimageRecords(t, prePath) {
		named[rec.addr] = true
		if rec.addr == store {
			storeSlots = rec.slots
		}
	}
	if named[common.Address{0xf0, 2}] || !slices.Contains(storeSlots, common.Hash{1}) || slices.Contains(storeSlots, common.BigToHash(big.NewInt(0x12))) {
		t.Fatal("the preimage file at A does not exercise keys coming and going")
	}

	// The node persists P, as a full write buffer would, and runs on while it
	// converts.
	node := *cfg
	node.PBTImport = true
	importer := func(chain *core.BlockChain, db ethdb.Database, preimages string) *pbtImporter {
		return &pbtImporter{
			chain:     chain,
			db:        db,
			cfg:       ethconfig.PBTImportConfig{Preimages: preimages, Anchor: a.Hash().Hex()},
			triedbDir: t.TempDir(),
			tmpDir:    filepath.Join(t.TempDir(), "pbt-import"),
			poll:      10 * time.Millisecond,
		}
	}
	convert := func(preimages string) (ethdb.Database, *core.BlockChain, *pbtImporter) {
		db := rawdb.NewMemoryDatabase()
		chain, err := core.NewBlockChain(db, gspec, engine, &node)
		if err != nil {
			t.Fatal(err)
		}
		insertBlocks(t, chain, blocks[:5])
		if err := chain.TrieDB().Commit(p.Root(), false); err != nil {
			t.Fatal(err)
		}
		insertBlocks(t, chain, blocks[5:6])
		chain.SetFinalized(blocks[5].Header())
		imp := importer(chain, db, preimages)
		if err := imp.Start(); err != nil {
			t.Fatal(err)
		}
		insertBlocks(t, chain, blocks[6:])
		<-imp.done
		return db, chain, imp
	}

	// Half the file, as the devnet's fault run cuts it: the conversion fails,
	// leaves nothing behind and the tree held, and the node follows on.
	whole, err := os.ReadFile(prePath)
	if err != nil {
		t.Fatal(err)
	}
	half := filepath.Join(t.TempDir(), "half")
	if err := os.WriteFile(half, whole[:len(whole)/2], 0o600); err != nil {
		t.Fatal(err)
	}
	db, chain, imp := convert(half)
	if prog := chain.MigrationProgress(); prog.Phase != "importFailed" {
		t.Fatalf("progress %+v after a conversion from half a file, want importFailed", prog)
	}
	if dirty, err := hasBinaryTrieState(db); err != nil || dirty {
		t.Fatalf("the failed conversion's writes survived (err %v)", err)
	}
	if !rawdb.ReadPBTImportPending(db) {
		t.Fatal("a failed conversion released the tree")
	}
	if got := chain.CurrentBlock().Number.Uint64(); got != head.NumberU64() {
		t.Fatalf("head %d after the failed conversion, want %d", got, head.NumberU64())
	}
	if info, err := os.Stat(half); err != nil || info.Size() != int64(len(whole)/2) {
		t.Fatalf("a failed conversion consumed its preimage file (err %v)", err)
	}
	imp.Stop()
	chain.Stop()

	db, chain, imp = convert(prePath)
	if num, hash, _ := rawdb.ReadPBTAnchor(rawdb.NewTable(db, string(rawdb.PBTPrefix))); num != p.NumberU64() || hash != p.Hash() {
		t.Fatalf("attested at block %d (%x), want the persisted block %d", num, hash, p.NumberU64())
	}
	if got := awaitRecord(t, db, head.NumberU64(), head.Hash()); got != want {
		t.Fatalf("self-converted tree's catch-up root %x, genesis-seeded said %x", got, want)
	}
	if _, ok := rawdb.ReadShadowStateRoot(db, blocks[3].Hash(), blocks[3].NumberU64()); ok {
		t.Fatal("follower replayed below the conversion's block")
	}
	if info, err := os.Stat(prePath); err != nil || info.Size() != 0 {
		t.Fatalf("the conversion did not consume its preimage file (err %v)", err)
	}
	imp.Stop()
	chain.Stop()

	chain, err = core.NewBlockChain(db, gspec, engine, &node)
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()
	imp = importer(chain, db, prePath)
	if err := imp.Start(); err != nil {
		t.Fatal(err)
	}
	if imp.done != nil || rawdb.ReadPBTImportPending(db) {
		t.Fatal("restarting with the same flags converted again")
	}
}
