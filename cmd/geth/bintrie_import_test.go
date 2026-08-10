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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
)

// importFixture converts alloc with both artifacts and returns everything an
// import test needs: the converter's database, the merkle anchor root, the
// binary root and the artifact paths.
func importFixture(t *testing.T, alloc types.GenesisAlloc) (ethdb.Database, common.Hash, common.Hash, string, string) {
	t.Helper()
	var (
		dir      = t.TempDir()
		snapPath = filepath.Join(dir, "snapshot.bin")
		prePath  = filepath.Join(dir, "preimages.bin")
	)
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

	src := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.ReadOnly,
	})
	defer src.Close()

	binRoot, err := convertState(chaindb, src, root, conversionOptions{
		tmpDir:       dir,
		snapshotPath: snapPath,
		preimagePath: prePath,
	})
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	return chaindb, root, binRoot, snapPath, prePath
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
		anchor: anchor, conversionOptions: conversionOptions{tmpDir: t.TempDir()}})
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

// The tamper harness: read the writer's valid artifacts, perform surgery,
// re-encode, and demand rejection.

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
		recs = append(recs, snapRecord{key: key, value: value})
	}
	return sr.root, recs
}

// writeSnapshotRecords re-encodes a snapshot. A zero root recomputes the
// honest one from the records, so mutations can choose which check meets
// them.
func writeSnapshotRecords(t *testing.T, path string, root common.Hash, count int, recs []snapRecord) {
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
	var buf bytes.Buffer
	header := make([]byte, snapshotHeaderSize)
	copy(header, root[:])
	binary.BigEndian.PutUint64(header[32:], uint64(count))
	buf.Write(header)
	for _, rec := range recs {
		blob, err := rlp.EncodeToBytes([]any{rec.key, common.TrimLeftZeroes(rec.value[:])})
		if err != nil {
			t.Fatal(err)
		}
		buf.Write(blob)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
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
		addr, slots, err := pr.next()
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

func writePreimageRecords(t *testing.T, path string, recs []preRecord) {
	t.Helper()
	slices.SortFunc(recs, func(a, b preRecord) int { return bytes.Compare(a.addr[:], b.addr[:]) })
	var buf bytes.Buffer
	for _, rec := range recs {
		slots := make([][]byte, 0, len(rec.slots))
		for _, slot := range rec.slots {
			slots = append(slots, common.TrimLeftZeroes(slot[:]))
		}
		blob, err := rlp.EncodeToBytes([]any{rec.addr[:], slots})
		if err != nil {
			t.Fatal(err)
		}
		buf.Write(blob)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
}

// TestImportRejects is the adversarial matrix: one surgery per way an
// artifact can lie, each of which must abort the import and leave nothing
// attested. Surgeries that recompute the claimed root reach past the
// internal-consistency check to the layer they target.
func TestImportRejects(t *testing.T) {
	var (
		contract  = common.HexToAddress("0x2000000000000000000000000000000000000002")
		delegated = common.HexToAddress("0x4000000000000000000000000000000000000004")
		eoa       = common.HexToAddress("0x1000000000000000000000000000000000000001")
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
				writeSnapshotRecords(t, path, bad, len(recs), recs)
			},
			wantErr: "rebuild to",
		},
		{
			name: "wrong leaf count",
			file: func(t *testing.T, path string) {
				claimed, recs := readSnapshotRecords(t, path)
				writeSnapshotRecords(t, path, claimed, len(recs)+1, recs)
			},
			wantErr: "header claims",
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
			name: "records out of order",
			snap: func(recs []snapRecord) []snapRecord {
				recs[0], recs[1] = recs[1], recs[0]
				return recs
			},
			wantErr: "out of order",
		},
		{
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
			wantErr: "does not decode",
		},
		{
			name: "reserved zone",
			snap: func(recs []snapRecord) []snapRecord {
				recs[0].key = bytes.Clone(recs[0].key)
				recs[0].key[0] = 0x02
				return recs
			},
			wantErr: "reserved zone",
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
						slices.SortFunc(recs[i].slots, func(a, b common.Hash) int { return bytes.Compare(a[:], b[:]) })
					}
				}
				return recs
			},
			wantErr: "surplus preimage",
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
			wantErr: "has no preimage",
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
			// The reserved bytes take part in no check but the tree's hash:
			// the MPT commits nothing there and the claimed root is the
			// attacker's to recompute, so only full-value canonicality
			// catches it.
			name:      "garbage in the basic-data reserved bytes",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				i := findKey(recs, bintrie.BasicDataKey(contract))
				recs[i].value[1] = 0xff
				return recs
			},
			wantErr: "canonical",
		},
		{
			// Likewise the nine padding bytes after a 23-byte designator.
			name:      "garbage in the delegation padding",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				i := findKey(recs, bintrie.DelegationKey(delegated))
				recs[i].value[23] = 0xff
				return recs
			},
			wantErr: "canonical",
		},
		{
			name:      "nonzero version",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				i := findKey(recs, bintrie.BasicDataKey(contract))
				recs[i].value[0] = 1
				return recs
			},
			wantErr: "version 1, must be 0",
		},
		{
			name:      "delegation and code hash together",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				return insertSorted(recs, snapRecord{key: bintrie.CodeHashKey(delegated), value: types.EmptyCodeHash})
			},
			wantErr: "holds both",
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
			name:      "empty code hash claiming code",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				i := findKey(recs, bintrie.BasicDataKey(eoa))
				recs[i].value[7] = 10
				return recs
			},
			wantErr: "assembled code hashes to",
		},
		{
			name:      "code hash claiming no code",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				i := findKey(recs, bintrie.BasicDataKey(loner))
				for b := 4; b < 8; b++ {
					recs[i].value[b] = 0
				}
				return recs
			},
			wantErr: "addressed by no account",
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
			name:      "account with neither code-hash nor delegation",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				i := findKey(recs, bintrie.CodeHashKey(eoa))
				return slices.Delete(recs, i, i+1)
			},
			wantErr: "holds neither",
		},
		{
			name:      "delegation with a wrong code size",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				i := findKey(recs, bintrie.BasicDataKey(delegated))
				recs[i].value[7] = 24
				return recs
			},
			wantErr: "must be 23",
		},
		{
			name:      "malformed designator",
			recompute: true,
			snap: func(recs []snapRecord) []snapRecord {
				i := findKey(recs, bintrie.DelegationKey(delegated))
				recs[i].value[0] ^= 1 // breaks the 0xef0100 marker
				return recs
			},
			wantErr: "malformed delegation",
		},
		{
			name:    "wrong anchor root",
			anchor:  common.HexToHash("0xdead"),
			wantErr: "re-derive merkle root",
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
				writeSnapshotRecords(t, badSnap, claimed, len(recs), recs)
			default:
				writeSnapshotRecords(t, badSnap, claimed, len(recs), recs)
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
