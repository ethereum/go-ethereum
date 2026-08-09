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
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
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
	imported, err := importState(impDB, snapPath, prePath, root, false, conversionOptions{tmpDir: t.TempDir()})
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
	imported, err := importState(impDB, snapPath, prePath, root, true, conversionOptions{tmpDir: t.TempDir()})
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
			imported, err := importState(impDB, snapPath, prePath, root, false, conversionOptions{tmpDir: t.TempDir()})
			if err != nil {
				t.Fatalf("import failed: %v", err)
			}
			if want := common.HexToHash(sv.Root); imported != want {
				t.Fatalf("imported root %x, reference says %s", imported, sv.Root)
			}
		})
	}
}
