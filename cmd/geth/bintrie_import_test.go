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
	"encoding/binary"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

// importFixture converts alloc with both artifacts and returns everything an
// import test needs: the converter's database, the merkle anchor root, the
// binary root and the artifact paths.
func importFixture(t *testing.T, alloc types.GenesisAlloc) (ethdb.Database, common.Hash, common.Hash, string, string) {
	t.Helper()
	opts := artifactOptions(t)
	chaindb, root, binRoot, err := convertGenesis(t, alloc, true, opts)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	return chaindb, root, binRoot, opts.snapshotPath, opts.preimagePath
}

// namespaceFamily collects one key family of the binary tree namespace.
func namespaceFamily(t *testing.T, chaindb ethdb.Database, prefix []byte) map[string][]byte {
	t.Helper()
	pbtdb := rawdb.NewTable(chaindb, string(rawdb.PBTPrefix))
	it := pbtdb.NewIterator(prefix, nil)
	defer it.Release()
	records := make(map[string][]byte)
	for it.Next() {
		records[string(it.Key())] = common.CopyBytes(it.Value())
	}
	if err := it.Error(); err != nil {
		t.Fatal(err)
	}
	return records
}

// TestImportRoundTrip pins the consumer against the producer: importing the
// converter's own artifacts into a fresh database must reproduce, byte for
// byte, the namespace the converter built - and the imported state must read
// through the node's stack, code and preimages included, on a database that
// held neither.
func TestImportRoundTrip(t *testing.T) {
	alloc := mixedAlloc(8347)
	convDB, root, binRoot, snapPath, prePath := importFixture(t, alloc)

	impDB := rawdb.NewMemoryDatabase()
	anchor := &types.Header{Number: big.NewInt(7), Root: root}
	imported, err := importState(impDB, importOptions{snapshot: snapPath, preimages: prePath,
		anchor: anchor, keepPreimages: true, conversionOptions: conversionOptions{tmpDir: t.TempDir()}})
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if imported != binRoot {
		t.Fatalf("imported root %x, the converter built %x", imported, binRoot)
	}
	// The namespace must be byte-identical: tree nodes, flat accounts, flat
	// slots.
	for _, family := range [][]byte{rawdb.TrieNodeAccountPrefix, rawdb.SnapshotAccountPrefix, rawdb.SnapshotStoragePrefix} {
		want := namespaceFamily(t, convDB, family)
		got := namespaceFamily(t, impDB, family)
		if len(got) != len(want) {
			t.Fatalf("family %x holds %d records imported, %d converted", family, len(got), len(want))
		}
		for key, value := range want {
			if !bytes.Equal(got[key], value) {
				t.Fatalf("family %x record %x diverges", family, key)
			}
		}
	}
	// The anchor must be recoverable: catching up from an imported state
	// starts at the block it commits, and the tree does not say which.
	pbtdb := rawdb.NewTable(impDB, string(rawdb.PBTPrefix))
	if number, hash, ok := rawdb.ReadPBTAnchor(pbtdb); !ok {
		t.Fatal("the import recorded no anchor")
	} else if number != 7 || hash != anchor.Hash() {
		t.Fatalf("anchor reads back as %d/%x, imported at %d/%x", number, hash, 7, anchor.Hash())
	}

	// The state must read through the stack a node uses.
	destTriedb := triedb.NewDatabase(impDB, &triedb.Config{IsPBT: true, PathDB: pathdb.Defaults})
	defer destTriedb.Close()
	statedb, err := state.New(imported, state.NewPBTDatabase(destTriedb, state.NewCodeDB(impDB)))
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for addr, acct := range alloc {
		if got := statedb.GetNonce(addr); got != acct.Nonce {
			t.Fatalf("account %x nonce %d, want %d", addr, got, acct.Nonce)
		}
		if got := statedb.GetCode(addr); !bytes.Equal(got, acct.Code) {
			t.Fatalf("account %x code %d bytes, want %d", addr, len(got), len(acct.Code))
		}
		for slot, value := range acct.Storage {
			if got := statedb.GetState(addr, slot); got != value {
				t.Fatalf("slot %x of %x reads %x, want %x", slot, addr, got, value)
			}
			if got := rawdb.ReadPreimage(impDB, crypto.Keccak256Hash(slot[:])); !bytes.Equal(got, slot[:]) {
				t.Fatalf("slot %x preimage missing after import", slot)
			}
		}
		// The preimages travelled too.
		if got := rawdb.ReadPreimage(impDB, crypto.Keccak256Hash(addr.Bytes())); !bytes.Equal(got, addr.Bytes()) {
			t.Fatalf("account %x preimage missing after import", addr)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("the fixture held no accounts")
	}
}

// TestImportVerifyOnly runs the whole pipeline with the writers disarmed:
// both checks pass and the database stays untouched.
func TestImportVerifyOnly(t *testing.T) {
	_, root, binRoot, snapPath, prePath := importFixture(t, artifactAlloc())

	impDB := rawdb.NewMemoryDatabase()
	imported, err := importState(impDB, importOptions{snapshot: snapPath, preimages: prePath,
		anchor: &types.Header{Number: new(big.Int), Root: root}, verifyOnly: true, conversionOptions: conversionOptions{tmpDir: t.TempDir()}})
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}
	if imported != binRoot {
		t.Fatalf("verified root %x, the converter built %x", imported, binRoot)
	}
	it := impDB.NewIterator(nil, nil)
	defer it.Release()
	if it.Next() {
		t.Fatalf("verify-only wrote key %x", it.Key())
	}
}

// TestImportMatchesReference closes the loop against the execution-specs
// reference: every state vector, converted, exported and re-imported, must
// land on the reference-computed root.
func TestImportMatchesReference(t *testing.T) {
	for _, sv := range loadStateVectors(t) {
		t.Run(sv.Name, func(t *testing.T) {
			if sv.Name == "zero_basic_data_with_storage" {
				// EIP-7523: a snapshot cannot carry an empty account. The
				// writer refuses it rather than silently drop it; DB-only
				// conversion (no snapshot requested) is unaffected, which is
				// why this vector still exists for the tree-parity tests.
				_, _, _, err := convertGenesis(t, allocOf(t, sv), true, artifactOptions(t))
				if err == nil {
					t.Fatal("converting an EIP-7523 empty account into a snapshot succeeded")
				}
				if !strings.Contains(err.Error(), "EIP-7523") {
					t.Fatalf("rejection %q does not name the fault", err)
				}
				return
			}
			_, root, _, snapPath, prePath := importFixture(t, allocOf(t, sv))

			impDB := rawdb.NewMemoryDatabase()
			imported, err := importState(impDB, importOptions{snapshot: snapPath, preimages: prePath,
				anchor: &types.Header{Number: new(big.Int), Root: root}, conversionOptions: conversionOptions{tmpDir: t.TempDir()}})
			if err != nil {
				t.Fatalf("import failed: %v", err)
			}
			if want := common.HexToHash(sv.Root); imported != want {
				t.Fatalf("imported root %x, reference says %s", imported, sv.Root)
			}
		})
	}
}

// TestImportEmptyState covers the empty state end to end: a zero-account
// genesis converts to the bare 33-byte snapshot (the end tag, then a zero
// binary root) and an empty preimage file, verify-only accepts it against the
// anchor whose root a genuinely empty state commits and rejects it against
// any other, and a full import leaves a namespace that reopens like any
// other.
func TestImportEmptyState(t *testing.T) {
	_, mptRoot, binRoot, snapPath, prePath := importFixture(t, types.GenesisAlloc{})
	if binRoot != (common.Hash{}) {
		t.Fatalf("empty state's binary root is %x, want zero", binRoot)
	}

	blob, err := os.ReadFile(snapPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(blob) != snapshotOverhead {
		t.Fatalf("empty snapshot is %d bytes, want %d", len(blob), snapshotOverhead)
	}
	preBlob, err := os.ReadFile(prePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(preBlob) != 0 {
		t.Fatalf("empty preimage file holds %d bytes, want 0", len(preBlob))
	}

	// verify-only accepts the empty state's own root.
	imported, err := importState(rawdb.NewMemoryDatabase(), importOptions{snapshot: snapPath, preimages: prePath,
		anchor: &types.Header{Number: new(big.Int), Root: mptRoot}, verifyOnly: true, conversionOptions: conversionOptions{tmpDir: t.TempDir()}})
	if err != nil {
		t.Fatalf("verify-only rejected the empty state against its own anchor: %v", err)
	}
	if imported != binRoot {
		t.Fatalf("verify-only reported root %x, converter built %x", imported, binRoot)
	}

	// verify-only rejects a non-empty anchor.
	if _, err := importState(rawdb.NewMemoryDatabase(), importOptions{snapshot: snapPath, preimages: prePath,
		anchor: &types.Header{Number: new(big.Int), Root: common.HexToHash("0xdead")}, verifyOnly: true, conversionOptions: conversionOptions{tmpDir: t.TempDir()}}); err == nil {
		t.Fatal("verify-only accepted the empty state against a mismatched anchor")
	}

	// A full import writes the attestation and reopens.
	impDB := rawdb.NewMemoryDatabase()
	anchor := &types.Header{Number: big.NewInt(1), Root: mptRoot}
	imported, err = importState(impDB, importOptions{snapshot: snapPath, preimages: prePath,
		anchor: anchor, conversionOptions: conversionOptions{tmpDir: t.TempDir()}})
	if err != nil {
		t.Fatalf("full import of the empty state failed: %v", err)
	}
	if imported != binRoot {
		t.Fatalf("import reported root %x, converter built %x", imported, binRoot)
	}
	if !rawdb.HasPBTState(impDB) {
		t.Fatal("the empty import attested no namespace")
	}
	pbtdb := rawdb.NewTable(impDB, string(rawdb.PBTPrefix))
	if number, hash, ok := rawdb.ReadPBTAnchor(pbtdb); !ok || number != 1 || hash != anchor.Hash() {
		t.Fatalf("anchor reads back as %d/%x/%v, imported at %d/%x", number, hash, ok, 1, anchor.Hash())
	}
	destTriedb := triedb.NewDatabase(impDB, &triedb.Config{IsPBT: true, PathDB: pathdb.Defaults})
	defer destTriedb.Close()
	if _, err := state.New(imported, state.NewPBTDatabase(destTriedb, state.NewCodeDB(impDB))); err != nil {
		t.Fatalf("reopening the PBT namespace on an empty state: %v", err)
	}
}

// The tamper harness: read the writer's valid artifacts, perform surgery,
// re-encode, and demand rejection.

// The tamper harness reads a valid snapshot as derived leaves, lets a case
// mutate that leaf set, and re-encodes it: a general, unguarded grouping
// encoder that mirrors the production writer's stem/zone grouping but
// trusts whatever leaves it is given, since a case's whole point is to
// construct leaf sets the production writer's canonical-form guard would
// itself refuse to emit. Byte-level corruption that isn't expressible as a
// leaf mutation goes through the typed builders directly (rawHeader,
// rawGroup, rawSnapshot, from bintrie_artifacts_test.go).

type snapRecord struct {
	key   []byte
	value [32]byte
}

func readSnapshotRecords(t *testing.T, path string) (common.Hash, []snapRecord) {
	t.Helper()
	sr, err := openSnapshot(path)
	if err != nil {
		t.Fatal(err)
	}
	defer sr.close()
	var recs []snapRecord
	for {
		key, value, err := sr.next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		recs = append(recs, snapRecord{key: bytes.Clone(key), value: value})
	}
	return sr.root, recs
}

// writeSnapshotRecords re-encodes a leaf set, grouped by contiguous
// same-stem runs in whatever order recs holds them - so a case that
// reorders whole stems produces a genuinely out-of-order file. A zero root
// recomputes the honest one from the records, so mutations can choose which
// check meets them.
func writeSnapshotRecords(t *testing.T, path string, root common.Hash, recs []snapRecord) {
	t.Helper()
	if root == (common.Hash{}) {
		rebuild := bintrie.NewStackBuilder(nil)
		for _, rec := range recs {
			if err := rebuild.Add(rec.key, rec.value[:]); err != nil {
				t.Fatalf("tampered records do not fold: %v", err)
			}
		}
		root = rebuild.Finish()
	}
	snap := rawSnapshot{root: root}
	for i := 0; i < len(recs); {
		key := recs[i].key
		switch key[0] {
		case bintrie.AccountZone:
			stem := key[:33]
			h := rawHeader{addrHash: [32]byte(stem[1:33])}
			var codeSize []byte
			for i < len(recs) && len(recs[i].key) == 34 && bytes.Equal(recs[i].key[:33], stem) {
				sub, v := recs[i].key[33], recs[i].value
				switch {
				case sub == bintrie.BasicDataLeafKey:
					h.nonce = common.TrimLeftZeroes(v[8:16])
					h.balance = common.TrimLeftZeroes(v[16:32])
					codeSize = common.TrimLeftZeroes(v[4:8])
				case sub == bintrie.CodeHashLeafKey:
					if v == types.EmptyCodeHash {
						h.kind = 0
					} else {
						h.kind = 1
						h.codeRef = append(append([]byte{}, v[:]...), rawVarint(codeSize...)...)
					}
				case sub == bintrie.DelegationLeafKey:
					h.kind = 2
					h.codeRef = append([]byte{}, v[len(types.DelegationPrefix):23]...)
				default:
					h.slots = append(h.slots, rawEntry{sub: sub - bintrie.HeaderStorageOffset, value: common.TrimLeftZeroes(v[:])})
				}
				i++
			}
			snap.records = append(snap.records, h.bytes()...)
		case bintrie.CodeZone:
			stem := key[:33]
			g := rawGroup{stemHash: [32]byte(stem[1:33])}
			for i < len(recs) && len(recs[i].key) == 34 && bytes.Equal(recs[i].key[:33], stem) {
				g.entries = append(g.entries, rawEntry{sub: recs[i].key[33], value: common.TrimLeftZeroes(recs[i].value[:])})
				i++
			}
			snap.records = append(snap.records, rawCodeGroup(g)...)
		case bintrie.StorageZone:
			addrHash := key[1:33]
			snap.records = append(snap.records, rawStorageAccount([32]byte(addrHash))...)
			for i < len(recs) && len(recs[i].key) == 66 && bytes.Equal(recs[i].key[1:33], addrHash) {
				gstem := recs[i].key[:65]
				g := rawGroup{stemHash: [32]byte(gstem[33:65])}
				for i < len(recs) && len(recs[i].key) == 66 && bytes.Equal(recs[i].key[:65], gstem) {
					g.entries = append(g.entries, rawEntry{sub: recs[i].key[65], value: common.TrimLeftZeroes(recs[i].value[:])})
					i++
				}
				snap.records = append(snap.records, rawStorageGroup(g)...)
			}
		default:
			t.Fatalf("record %x sits in an unknown zone", key)
		}
	}
	if err := os.WriteFile(path, snap.encode(), 0600); err != nil {
		t.Fatal(err)
	}
}

type preRecord struct {
	addr  common.Address
	slots []common.Hash
}

func readPreimageRecords(t *testing.T, path string) []preRecord {
	t.Helper()
	pr, err := openPreimages(path)
	if err != nil {
		t.Fatal(err)
	}
	defer pr.close()
	var recs []preRecord
	for {
		addr, _, slots, err := pr.next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		recs = append(recs, preRecord{addr: addr, slots: slots})
	}
	return recs
}

// encodePreimageRecords lays recs out in the EIP-8347 layout and hashed-key
// order, so cases only choose the set.
func encodePreimageRecords(recs []preRecord) []byte {
	byHash := func(a, b []byte) int { return bytes.Compare(crypto.Keccak256(a), crypto.Keccak256(b)) }
	recs = slices.Clone(recs)
	slices.SortFunc(recs, func(a, b preRecord) int { return byHash(a.addr[:], b.addr[:]) })
	var buf bytes.Buffer
	for _, rec := range recs {
		slots := slices.Clone(rec.slots)
		slices.SortFunc(slots, func(a, b common.Hash) int { return byHash(a[:], b[:]) })
		buf.Write(rec.addr[:])
		buf.Write(binary.BigEndian.AppendUint32(nil, uint32(len(slots))))
		for _, slot := range slots {
			buf.Write(slot[:])
		}
	}
	return buf.Bytes()
}

func writePreimageRecords(t *testing.T, path string, recs []preRecord) {
	t.Helper()
	if err := os.WriteFile(path, encodePreimageRecords(recs), 0600); err != nil {
		t.Fatal(err)
	}
}

// TestImportRejects is the adversarial matrix: one surgery per way an
// artifact can lie, each of which must abort the import and leave nothing
// attested. Surgeries that recompute the claimed root reach past the
// internal-consistency check to the layer they target.
func TestImportRejects(t *testing.T) {
	var (
		contract = common.HexToAddress("0x2000000000000000000000000000000000000002")
		eoa      = common.HexToAddress("0x1000000000000000000000000000000000000001")
		// The one account whose code no other account shares, so a surgery
		// on its size or chunks reaches the code limb rather than tripping
		// the shared-hash size conflict first.
		loner     = common.HexToAddress("0x5000000000000000000000000000000000000005")
		alloc     = artifactAlloc()
		lonerHash = crypto.Keccak256Hash(alloc[loner].Code)
		lonerCode = uint32(len(alloc[loner].Code))
	)
	_, root, _, snapPath, prePath := importFixture(t, alloc)

	findKey := func(recs []snapRecord, key []byte) int {
		t.Helper()
		for i, rec := range recs {
			if bytes.Equal(rec.key, key) {
				return i
			}
		}
		t.Fatalf("fixture lacks the targeted key %x", key)
		return -1
	}
	insertSorted := func(recs []snapRecord, rec snapRecord) []snapRecord {
		i, _ := slices.BinarySearchFunc(recs, rec, func(a, b snapRecord) int { return bytes.Compare(a.key, b.key) })
		return slices.Insert(recs, i, rec)
	}
	firstCode := func(recs []snapRecord) int {
		for i, rec := range recs {
			if rec.key[0] == bintrie.CodeZone {
				return i
			}
		}
		t.Fatal("fixture holds no code leaves")
		return -1
	}
	// swapHeaderSlots swaps two of one account's header-range storage
	// leaves. They share a stem, so the preimage candidate join (which
	// matches on the stem alone) sees no mismatch; only the reader's own
	// key-ordering check can catch it. Swapping whole stems instead would
	// also desync the candidate join, which reports its own, earlier
	// mismatch first.
	swapHeaderSlots := func(recs []snapRecord) []snapRecord {
		recs = slices.Clone(recs)
		i := findKey(recs, bintrie.HeaderKey(contract, bintrie.HeaderStorageOffset))
		j := findKey(recs, bintrie.HeaderKey(contract, bintrie.HeaderStorageOffset+63))
		recs[i], recs[j] = recs[j], recs[i]
		return recs
	}

	for _, tc := range []struct {
		name      string
		recompute bool
		snap      func([]snapRecord) []snapRecord
		pre       func([]preRecord) []preRecord
		file      func(t *testing.T, snapPath string)
		anchor    common.Hash
		wantErr   string
	}{
		{
			name: "wrong claimed root",
			file: func(t *testing.T, path string) {
				_, recs := readSnapshotRecords(t, path)
				bad := root
				bad[0] ^= 1
				writeSnapshotRecords(t, path, bad, recs)
			},
			wantErr: "rebuild to",
		},
		{
			name: "flipped leaf value",
			snap: func(recs []snapRecord) []snapRecord {
				recs[0].value[31] ^= 1
				return recs
			},
			wantErr: "rebuild to",
		},
		{
			name:    "records out of order",
			snap:    swapHeaderSlots,
			wantErr: "out of order",
		},
		{
			// Appended after an otherwise complete, valid snapshot: one stray
			// byte past the root.
			name: "trailing garbage",
			file: func(t *testing.T, path string) {
				blob, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, append(blob, 0x00), 0600); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "trailing byte",
		},
		{
			name: "missing preimage record",
			pre: func(recs []preRecord) []preRecord {
				return slices.DeleteFunc(recs, func(r preRecord) bool { return r.addr == eoa })
			},
			wantErr: "has no preimage",
		},
		{
			name: "surplus preimage address",
			pre: func(recs []preRecord) []preRecord {
				return append(recs, preRecord{addr: common.HexToAddress("0x9999999999999999999999999999999999999999")})
			},
			wantErr: "the state does not hold",
		},
		{
			name: "surplus preimage slot",
			pre: func(recs []preRecord) []preRecord {
				for i := range recs {
					if recs[i].addr == contract {
						recs[i].slots = append(recs[i].slots, common.HexToHash("0x99"))
					}
				}
				return recs
			},
			wantErr: "names slot",
		},
		{
			name: "missing preimage slot",
			pre: func(recs []preRecord) []preRecord {
				for i := range recs {
					if recs[i].addr == contract {
						recs[i].slots = recs[i].slots[1:]
					}
				}
				return recs
			},
			wantErr: "does not name",
		},
		{
			name:      "flipped push-data offset",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				recs[firstCode(recs)].value[0] ^= 1
				return recs
			},
			wantErr: "re-chunking",
		},
		{
			name:      "truncated code",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				return slices.Delete(recs, firstCode(recs), firstCode(recs)+1)
			},
			wantErr: "assembled code hashes to",
		},
		{
			name:      "wrong code size",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				i := findKey(recs, bintrie.BasicDataKey(contract))
				recs[i].value[7]-- // code_size low byte
				return recs
			},
			wantErr: "claimed with sizes",
		},
		{
			name:      "surplus code leaf",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				return insertSorted(recs, snapRecord{key: bintrie.CodeChunkKey(crypto.Keccak256Hash([]byte("junk")), 0), value: common.Hash{31: 1}})
			},
			wantErr: "addressed by no account",
		},
		{
			// A header-range slot's number is derived from its sub-index, so
			// the two directions are exact rather than inferred.
			name: "header slot the preimage file omits",
			pre: func(recs []preRecord) []preRecord {
				for i := range recs {
					if recs[i].addr == contract {
						recs[i].slots = slices.DeleteFunc(recs[i].slots, func(h common.Hash) bool {
							return h == common.BigToHash(big.NewInt(63))
						})
					}
				}
				return recs
			},
			wantErr: "holds slot 63",
		},
		{
			name: "header slot the state does not hold",
			pre: func(recs []preRecord) []preRecord {
				for i := range recs {
					if recs[i].addr == contract {
						recs[i].slots = append(recs[i].slots, common.BigToHash(big.NewInt(7)))
					}
				}
				return recs
			},
			wantErr: "names slot 7",
		},
		{
			name:      "forged code size",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				i := findKey(recs, bintrie.BasicDataKey(loner))
				// A four-byte field claiming 4 GB of code: the buffer and the
				// candidate set are sized from it before anything checks it.
				for b := 4; b < 8; b++ {
					recs[i].value[b] = 0xff
				}
				return recs
			},
			wantErr: "import bound",
		},
		{
			name:      "whole bytecode absent",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				return slices.DeleteFunc(recs, func(r snapRecord) bool {
					return r.key[0] == bintrie.CodeZone &&
						bytes.Equal(r.key[:33], bintrie.CodeChunkStem(lonerHash, 0))
				})
			},
			wantErr: "assembled code hashes to",
		},
		{
			name:      "chunk past the code's last",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				chunks := (lonerCode + 30) / 31
				return insertSorted(recs, snapRecord{
					key:   bintrie.CodeChunkKey(lonerHash, uint64(chunks)+1),
					value: common.Hash{31: 1},
				})
			},
			wantErr: "beyond the",
		},
		{
			name:    "wrong anchor root",
			anchor:  common.HexToHash("0xdead"),
			wantErr: "re-derive merkle root",
		},
		{
			// A kind-1 account's bytecode must not itself be an EIP-7702
			// delegation indicator: that account must carry kind 2 instead.
			name:      "delegation bytecode without kind 2",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				target := common.HexToAddress("0x9000000000000000000000000000000000000009")
				newCode := types.AddressToDelegation(target)
				newHash := crypto.Keccak256Hash(newCode)
				chunked := bintrie.ChunkifyCode(newCode)
				i := findKey(recs, bintrie.BasicDataKey(loner))
				recs[i].value[7] = byte(len(newCode))
				j := findKey(recs, bintrie.CodeHashKey(loner))
				recs[j].value = newHash
				recs = slices.DeleteFunc(recs, func(r snapRecord) bool {
					return r.key[0] == bintrie.CodeZone && bytes.Equal(r.key[:33], bintrie.CodeChunkStem(lonerHash, 0))
				})
				return insertSorted(recs, snapRecord{key: bintrie.CodeChunkKey(newHash, 0), value: [32]byte(chunked[:32])})
			},
			wantErr: "recovers to a delegation indicator",
		},
		{
			// A slot number below 64 belongs in the header stem; encoding
			// the same slot again through the overflow path produces a
			// leaf no preimage candidate can match - the preimage file
			// never registers a header-range slot as an overflow candidate.
			name:      "header slot repeated as a storage entry",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				var treeIndex uint256.Int
				key := append(bintrie.StorageStem(contract, &treeIndex), 0)
				return insertSorted(recs, snapRecord{key: key, value: common.Hash{31: 1}})
			},
			wantErr: "has no preimage",
		},
		{
			// Every storage record must name an account that has a header
			// record. With the header gone, the account's storage and its
			// preimage remain, and the preimage names an address the state
			// does not hold.
			name:      "storage record without its header record",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				header := bintrie.HeaderStem(contract)
				return slices.DeleteFunc(recs, func(r snapRecord) bool { return bytes.HasPrefix(r.key, header) })
			},
			wantErr: "the state does not hold",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			badSnap := filepath.Join(dir, "snapshot.bin")
			badPre := filepath.Join(dir, "preimages.bin")

			claimed, recs := readSnapshotRecords(t, snapPath)
			switch {
			case tc.file != nil:
				blob, err := os.ReadFile(snapPath)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(badSnap, blob, 0600); err != nil {
					t.Fatal(err)
				}
				tc.file(t, badSnap)
			case tc.snap != nil:
				recs = tc.snap(recs)
				if tc.recompute {
					claimed = common.Hash{}
				}
				writeSnapshotRecords(t, badSnap, claimed, recs)
			default:
				writeSnapshotRecords(t, badSnap, claimed, recs)
			}
			preRecs := readPreimageRecords(t, prePath)
			if tc.pre != nil {
				preRecs = tc.pre(preRecs)
			}
			writePreimageRecords(t, badPre, preRecs)

			anchor := root
			if tc.anchor != (common.Hash{}) {
				anchor = tc.anchor
			}
			impDB := rawdb.NewMemoryDatabase()
			_, err := importState(impDB, importOptions{snapshot: badSnap, preimages: badPre,
				anchor: &types.Header{Number: new(big.Int), Root: anchor}, conversionOptions: conversionOptions{tmpDir: dir}})
			if err == nil {
				t.Fatal("a tampered artifact was imported")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("rejection %q does not name the fault %q", err, tc.wantErr)
			}
			if rawdb.HasPBTState(impDB) {
				t.Fatal("a rejected import attested its namespace")
			}
		})
	}
}
