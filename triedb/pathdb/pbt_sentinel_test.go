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

package pathdb

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// TestTipLookupZeroRootIsNotStale pins that a lookup landing on a zero-rooted
// disk layer reports a hit, not staleness.
//
// The binary tree's empty root is 32 zero bytes, the same value these lookups
// used to return to mean "the requested state is stale". A hit on an empty disk
// layer was therefore byte-identical to a miss, and the caller turned every one
// of them into errSnapshotStale.
func TestTipLookupZeroRootIsNotStale(t *testing.T) {
	if types.EmptyBinaryHash != (common.Hash{}) {
		t.Skip("the binary tree's empty root is no longer the zero hash; this collision is gone")
	}
	var (
		account = common.Hash{0xaa}
		slot    = common.Hash{0xbb}
		base    = types.EmptyBinaryHash // an empty binary-tree disk layer
	)
	l := &lookup{
		accounts: make(map[common.Hash][]common.Hash),
		storages: make(map[[64]byte][]common.Hash),
		// Only the disk layer exists, so nothing descends from anything.
		descendant: func(state common.Hash, ancestor common.Hash) bool { return false },
	}
	// Requesting the disk layer's own state: an unmodified account resolves to
	// the disk layer, which happens to be the zero root.
	tip, found := l.accountTip(account, base, base)
	if !found {
		t.Fatal("a zero-rooted disk layer was reported stale for an account read")
	}
	if tip != base {
		t.Fatalf("account tip is %x, want the disk layer %x", tip, base)
	}
	tip, found = l.storageTip(account, slot, base, base)
	if !found {
		t.Fatal("a zero-rooted disk layer was reported stale for a storage read")
	}
	if tip != base {
		t.Fatalf("storage tip is %x, want the disk layer %x", tip, base)
	}
	// A genuinely unrelated state is still reported stale, so the flag is not
	// simply always true.
	if _, found = l.accountTip(account, common.Hash{0xcc}, base); found {
		t.Fatal("an unrelated state was reported as reachable")
	}
	if _, found = l.storageTip(account, slot, common.Hash{0xcc}, base); found {
		t.Fatal("an unrelated state was reported as reachable")
	}
}

// TestGeneratorRequiresRecordedSnapshotRoot pins that a persisted generator is
// only trusted when a snapshot root was actually written.
//
// The consistency check compares the trie root against the recorded snapshot
// root, but a missing snapshot root reads back as the zero hash - which is also
// the binary tree's empty root. Comparing values alone therefore let an
// unwritten snapshot "match" an empty tree, validating a stale generator,
// including one claiming the flat state was fully generated.
func TestGeneratorRequiresRecordedSnapshotRoot(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	// A generator claiming completion, with no snapshot root recorded, over a
	// database whose account trie root node is absent - so the binary hasher
	// yields the zero hash, matching the missing snapshot root.
	blob, err := rlp.EncodeToBytes(&journalGenerator{Done: true})
	if err != nil {
		t.Fatal(err)
	}
	rawdb.WriteSnapshotGenerator(db, blob)

	generator, _, err := loadGenerator(db, binaryNodeHasher)
	if err != nil {
		t.Fatal(err)
	}
	if generator != nil {
		t.Fatal("a generator was accepted although no snapshot root was ever recorded")
	}

	// With a snapshot root recorded and matching, the same generator is kept.
	rawdb.WriteSnapshotRoot(db, common.Hash{})
	generator, _, err = loadGenerator(db, binaryNodeHasher)
	if err != nil {
		t.Fatal(err)
	}
	if generator == nil || !generator.Done {
		t.Fatal("a consistent generator was discarded")
	}
}
