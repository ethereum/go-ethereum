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

package bintrie_test

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
)

// reencode round-trips a proof through its wire form: the canonical-bitmap rule
// lives in the decoder, so skipping it never checks that the prover emits the
// one form its own verifier accepts.
func reencode(t *testing.T, mp *bintrie.Multiproof) *bintrie.Multiproof {
	t.Helper()
	decoded, err := bintrie.DecodeMultiproof(mp.Encode())
	if err != nil {
		t.Fatalf("the prover emitted a proof its own decoder refuses: %v", err)
	}
	return decoded
}

// recordsDB writes the records of tr into a fresh proof-only database and
// returns a trie opened on the same root, which is what a consumer does with a
// proof it has checked.
func recordsDB(t *testing.T, tr *bintrie.BinaryTrie, root common.Hash) (*triedb.Database, map[string][]byte) {
	t.Helper()

	disk := rawdb.NewMemoryDatabase()
	tbl := rawdb.NewTable(disk, string(rawdb.PBTPrefix))
	written := make(map[string][]byte)
	if err := tr.WriteRecords(func(path []byte, _ common.Hash, blob []byte) {
		written[string(path)] = blob
		rawdb.WriteAccountTrieNode(tbl, path, blob)
	}); err != nil {
		t.Fatalf("writing records: %v", err)
	}
	rawdb.WriteSnapshotRoot(tbl, root)

	db := triedb.NewDatabase(disk, triedb.PBTWitnessDefaults)
	t.Cleanup(func() { db.Close() })
	return db, written
}

// TestWriteRecordsRoundTrip: a verified proof written out as records and read
// back must answer what it covered and fault on the rest. Only here does the
// writer's path arithmetic meet the reader's, which nothing downstream checks.
func TestWriteRecordsRoundTrip(t *testing.T) {
	srcDB, root, addr, codeHash := mpFixture(t, 24576, 64)

	var (
		basic     = bintrie.BasicDataKey(addr)
		chunk     = bintrie.CodeChunkKey(codeHash, 0)
		uncovered = bintrie.CodeChunkKey(codeHash, 200)
	)
	// A stem request as well as keys, so the proof carries a whole group and an
	// expanded one, which serialize through different record forms.
	mp, err := openTrie(t, srcDB, root).ProveRequests(bintrie.ProofRequests{
		Keys:  [][]byte{chunk},
		Stems: [][]byte{basic[:bintrie.AccountKeyLength-1]},
	})
	if err != nil {
		t.Fatal(err)
	}
	verified, err := bintrie.VerifyMultiproof(root, reencode(t, mp))
	if err != nil {
		t.Fatal(err)
	}
	db, written := recordsDB(t, verified, root)
	if len(written) == 0 {
		t.Fatal("the verified tree produced no records")
	}
	// The root has to be there: NewBinaryTrie resolves it eagerly, so a tree whose
	// root record is missing cannot be opened at all.
	if _, ok := written[""]; !ok {
		t.Fatal("no record at the empty path, so the root cannot be resolved")
	}
	reloaded := openTrie(t, db, root)
	if got := reloaded.Hash(); got != root {
		t.Fatalf("reloaded root is %x, want %x", got, root)
	}
	// Everything the proof covered reads back as the source tree holds it.
	ref := openTrie(t, srcDB, root)
	for _, key := range [][]byte{basic, bintrie.CodeHashKey(addr), chunk} {
		want, err := ref.GetStemValue(key)
		if err != nil {
			t.Fatal(err)
		}
		got, err := reloaded.GetStemValue(key)
		if err != nil {
			t.Fatalf("key %x: %v", key, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("key %x: got %x, want %x", key, got, want)
		}
	}
	// The stem request has to have kept its group whole, or a write to it would be
	// refused. That is the difference between a proof a block can replay against
	// and one it can only read.
	if _, err := reloaded.GetAccount(addr); err != nil {
		t.Fatalf("account read over a proved stem: %v", err)
	}
	// A chunk the proof did not cover has to fault rather than read as absent: the
	// record was never written, and answering zero would let a replay compute a
	// root over state that was never there.
	if _, err := reloaded.GetStemValue(uncovered); err == nil {
		t.Fatal("an uncovered chunk read back without error")
	}
}

// TestWriteRecordsSkipsStubs pins that the records stop where the proof does.
func TestWriteRecordsSkipsStubs(t *testing.T) {
	srcDB, root, addr, codeHash := mpFixture(t, 24576, 64)

	// One chunk of a 256-leaf code stem: most of the tree is a stub.
	narrow, err := openTrie(t, srcDB, root).ProveMulti([][]byte{
		bintrie.BasicDataKey(addr), bintrie.CodeChunkKey(codeHash, 0),
	})
	if err != nil {
		t.Fatal(err)
	}
	wide, err := openTrie(t, srcDB, root).ProveRequests(bintrie.ProofRequests{
		Keys:  [][]byte{bintrie.BasicDataKey(addr)},
		Stems: [][]byte{bintrie.CodeChunkStem(codeHash, 0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	verifiedNarrow, err := bintrie.VerifyMultiproof(root, reencode(t, narrow))
	if err != nil {
		t.Fatal(err)
	}
	verifiedWide, err := bintrie.VerifyMultiproof(root, reencode(t, wide))
	if err != nil {
		t.Fatal(err)
	}
	_, narrowRecords := recordsDB(t, verifiedNarrow, root)
	_, wideRecords := recordsDB(t, verifiedWide, root)

	// A whole-stem proof holds the group as one record; a one-chunk proof holds
	// the expansion around one leaf. Equal counts would mean the stem request
	// stopped meaning anything.
	if len(narrowRecords) == len(wideRecords) {
		t.Fatalf("a whole-stem proof and a one-chunk proof produced %d records each", len(wideRecords))
	}
}

// TestProofRecorderRequests pins the recorder's contract: deduplicated, sorted,
// and never empty.
func TestProofRecorderRequests(t *testing.T) {
	rec := bintrie.NewProofRecorder()

	// A proof over no requests collapses the tree to a single hash, which verifies
	// against any root and is refused - so an empty block still has to record the
	// root.
	if got := rec.Requests(); len(got.Paths) != 1 || got.Paths[0].Bits != 0 {
		t.Fatalf("a fresh recorder holds %v, want just the root path", got.Paths)
	}
	key := bintrie.BasicDataKey(common.Address{0xaa})
	rec.AddKey(key)
	rec.AddKey(key)
	rec.AddStem(key[:bintrie.AccountKeyLength-1])
	rec.AddPath(key, 8)
	rec.AddPath(key, 8)

	req := rec.Requests()
	if len(req.Keys) != 1 || len(req.Stems) != 1 || len(req.Paths) != 2 {
		t.Fatalf("got %d keys, %d stems, %d paths; want 1, 1, 2", len(req.Keys), len(req.Stems), len(req.Paths))
	}
	// Sorted, so one set of accesses always produces one proof: the request set is
	// built from maps, and a proof that varied run to run could not be compared.
	if req.Paths[0].Bits > req.Paths[1].Bits {
		t.Fatalf("paths came back unsorted: %v", req.Paths)
	}
	// A nil recorder is the "recording off" case every hook relies on.
	var off *bintrie.ProofRecorder
	off.AddKey(key)
	off.AddStem(key)
	off.AddPath(key, 8)
	if off.Len() != 0 {
		t.Fatal("a nil recorder accumulated something")
	}
}
