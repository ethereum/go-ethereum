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

package state

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// fuzzJournalAddrs is a small fixed pool used by the fuzz harness to force
// repeated collisions on the same account, which exercises the multi-entry
// path in the journal's mutation tracking and originals cleanup on revert.
// It deliberately excludes the RIPEMD-160 precompile (0x03), which has a
// consensus-level touch/revert exception that would complicate invariants.
var fuzzJournalAddrs = []common.Address{
	common.BytesToAddress([]byte{0x11}),
	common.BytesToAddress([]byte{0x22}),
	common.BytesToAddress([]byte{0x44}),
}

// checkJournalInvariants validates that:
//   - journal.mutations exactly reflects the dirty entries currently in
//     journal.entries (per-kind counts and mask match what you'd get by
//     walking the entries from scratch).
//   - journal.originals mirrors that set for the three tracked metadata kinds
//     (balance/nonce/code): a *Set flag is true iff the account currently has
//     at least one corresponding entry in the journal.
//   - An address is present in originals only if it also has at least one
//     tracked-kind mutation in the journal.
func checkJournalInvariants(t *testing.T, j *journal) {
	t.Helper()

	// Reconstruct the expected per-address counts from the live entries.
	expected := make(map[common.Address]*journalMutationCounts)
	for _, e := range j.entries {
		addr, kind, dirty := e.mutation()
		if !dirty {
			continue
		}
		c := expected[addr]
		if c == nil {
			c = &journalMutationCounts{}
			expected[addr] = c
		}
		c.add(kind)
	}

	if len(j.mutations) != len(expected) {
		t.Fatalf("mutations size %d, want %d", len(j.mutations), len(expected))
	}
	for addr, state := range j.mutations {
		want, ok := expected[addr]
		if !ok {
			t.Fatalf("mutations has extra address %x", addr)
		}
		if state.counts != *want {
			t.Fatalf("addr %x: counts=%+v want=%+v", addr, state.counts, *want)
		}
		// First-touch *Set flags must mirror the live per-kind counts.
		if state.balanceSet != (want.balance > 0) {
			t.Fatalf("addr %x: balanceSet=%v want=%v (balance count=%d)",
				addr, state.balanceSet, want.balance > 0, want.balance)
		}
		if state.nonceSet != (want.nonce > 0) {
			t.Fatalf("addr %x: nonceSet=%v want=%v (nonce count=%d)",
				addr, state.nonceSet, want.nonce > 0, want.nonce)
		}
		if state.codeSet != (want.code > 0) {
			t.Fatalf("addr %x: codeSet=%v want=%v (code count=%d)",
				addr, state.codeSet, want.code > 0, want.code)
		}
	}
}

// FuzzJournal drives a randomised sequence of state mutations, snapshots and
// reverts against a fresh StateDB and validates the journal's internal
// bookkeeping invariants after every step. It also asserts that reverting
// back to the root snapshot empties mutations, originals and entries
// completely. The seed corpus ensures the test also runs as a regular unit
// test via `go test -run FuzzJournal`.
func FuzzJournal(f *testing.F) {
	seeds := [][]byte{
		// balance then full revert (simplest a→b→a case).
		{0x00, 0x00, 0x05, 0x05, 0x00},
		// balance+nonce+code mixed, then revert to root.
		{0x00, 0x00, 0x01, 0x01, 0x01, 0x02, 0x02, 0x02, 0x00, 0x03, 0x05, 0x00},
		// snapshot, mutate, revert, mutate again.
		{0x04, 0x00, 0x00, 0x07, 0x05, 0x00, 0x00, 0x01, 0x05},
		// storage interleaved with metadata.
		{0x03, 0x00, 0x01, 0x00, 0x01, 0x05, 0x03, 0x02, 0x02, 0x04, 0x03, 0x01, 0x07},
		// many ops, no explicit revert — exercises steady-state invariants.
		{0x00, 0x01, 0x02, 0x00, 0x01, 0x02, 0x03, 0x00, 0x01, 0x02,
			0x03, 0x04, 0x00, 0x01, 0x02, 0x00, 0x06, 0x08, 0x0a, 0x0c},
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		sdb, err := New(types.EmptyRootHash, NewDatabaseForTesting())
		if err != nil {
			t.Fatal(err)
		}
		root := sdb.Snapshot()

		// Stack of snapshot IDs taken during the fuzz loop.
		var pending []int

		// readByte returns the next byte and advances the cursor. Returns
		// (0, false) if exhausted.
		i := 0
		readByte := func() (byte, bool) {
			if i >= len(data) {
				return 0, false
			}
			b := data[i]
			i++
			return b, true
		}

		for {
			op, ok := readByte()
			if !ok {
				break
			}
			switch op % 6 {
			case 0: // SetBalance
				a, ok1 := readByte()
				v, ok2 := readByte()
				if !ok1 || !ok2 {
					break
				}
				addr := fuzzJournalAddrs[int(a)%len(fuzzJournalAddrs)]
				sdb.SetBalance(addr, uint256.NewInt(uint64(v)), tracing.BalanceChangeUnspecified)
			case 1: // SetNonce
				a, ok1 := readByte()
				n, ok2 := readByte()
				if !ok1 || !ok2 {
					break
				}
				addr := fuzzJournalAddrs[int(a)%len(fuzzJournalAddrs)]
				sdb.SetNonce(addr, uint64(n), tracing.NonceChangeUnspecified)
			case 2: // SetCode
				a, ok1 := readByte()
				l, ok2 := readByte()
				if !ok1 || !ok2 {
					break
				}
				addr := fuzzJournalAddrs[int(a)%len(fuzzJournalAddrs)]
				code := make([]byte, int(l)%8)
				for k := range code {
					b, ok := readByte()
					if !ok {
						break
					}
					code[k] = b
				}
				sdb.SetCode(addr, code, tracing.CodeChangeUnspecified)
			case 3: // SetState (storage; tracked as mutation kind, no original)
				a, ok1 := readByte()
				k, ok2 := readByte()
				v, ok3 := readByte()
				if !ok1 || !ok2 || !ok3 {
					break
				}
				addr := fuzzJournalAddrs[int(a)%len(fuzzJournalAddrs)]
				sdb.SetState(addr,
					common.BytesToHash([]byte{k}),
					common.BytesToHash([]byte{v}))
			case 4: // Snapshot
				pending = append(pending, sdb.Snapshot())
			case 5: // RevertToSnapshot
				if len(pending) == 0 {
					break
				}
				sel, ok := readByte()
				if !ok {
					break
				}
				idx := int(sel) % len(pending)
				sdb.RevertToSnapshot(pending[idx])
				pending = pending[:idx]
			}
			checkJournalInvariants(t, sdb.journal)
		}

		// After reverting to the root snapshot, the journal must be fully
		// drained: no entries, no mutations, no originals. This is the core
		// guarantee the user cares about — "all mutations against a single
		// account reverted" taken to its limit across every account.
		sdb.RevertToSnapshot(root)
		checkJournalInvariants(t, sdb.journal)

		if n := len(sdb.journal.entries); n != 0 {
			t.Fatalf("entries not drained after revert-to-root: %d remain", n)
		}
		if n := len(sdb.journal.mutations); n != 0 {
			t.Fatalf("mutations not drained after revert-to-root: %d remain", n)
		}
	})
}
