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
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// tagEntry is a minimal journalEntry used by journal tests. It carries an
// integer tag so frameEntries iteration order can be verified, and is a no-op
// on revert so the surrounding StateDB can be a zero value.
type tagEntry struct {
	tag int
}

func (t tagEntry) revert(*StateDB)                 {}
func (t tagEntry) dirtied() (common.Address, bool) { return common.Address{}, false }
func (t tagEntry) copy() journalEntry              { return t }

// frameTags drives frameEntries and returns the visited tags in order.
func frameTags(j *journal) []int {
	var got []int
	j.frameEntries(func(e journalEntry) {
		got = append(got, e.(tagEntry).tag)
	})
	return got
}

// didPanic reports whether fn panicked.
func didPanic(fn func()) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	fn()
	return false
}

// TestJournalFrameTracking covers the happy paths of closeSnapshot and
// frameEntries together: basic single-child filtering, empty-range elision,
// multiple siblings, transitive descendant absorption, and the no-open-frame
// edge case for frameEntries. Building one composite scenario and asserting
// at each step keeps the expected behaviour as a connected story rather than
// scattering it across many tiny tests.
func TestJournalFrameTracking(t *testing.T) {
	j := newJournal()

	// frameEntries on an empty journal is a no-op.
	if got := frameTags(j); len(got) != 0 {
		t.Fatalf("empty journal frameEntries: have %v, want []", got)
	}

	j.snapshot()
	j.append(tagEntry{1}) // outer

	// Closing an empty child frame must not record a degenerate range.
	empty := j.snapshot()
	j.closeSnapshot(empty)
	if got := j.validRevisions[0].closedChildren; len(got) != 0 {
		t.Fatalf("empty child should not propagate, have %+v", got)
	}

	// First sibling child: two entries, then close. Range goes onto outer.
	c1 := j.snapshot()
	c1Start := len(j.entries)
	j.append(tagEntry{10})
	j.append(tagEntry{11})
	c1End := len(j.entries)
	j.closeSnapshot(c1)

	j.append(tagEntry{2}) // outer between siblings

	// Second sibling, with a grandchild closed inside it. After the
	// grandchild closes, more entries appear in the child before it itself
	// closes. The outer must end up with a single range that covers the
	// child (which transitively covers the grandchild).
	c2 := j.snapshot()
	c2Start := len(j.entries)
	j.append(tagEntry{20})

	gc := j.snapshot()
	j.append(tagEntry{300})
	j.closeSnapshot(gc)

	j.append(tagEntry{21})
	c2End := len(j.entries)
	j.closeSnapshot(c2)

	j.append(tagEntry{3}) // outer after both siblings

	got := j.validRevisions[0].closedChildren
	want := []frameRange{{c1Start, c1End}, {c2Start, c2End}}
	if !slices.Equal(got, want) {
		t.Fatalf("closedChildren: have %+v, want %+v", got, want)
	}
	if tags := frameTags(j); !slices.Equal(tags, []int{1, 2, 3}) {
		t.Fatalf("frameEntries: have %v, want [1 2 3]", tags)
	}

	// Closing the outermost (no-parent) frame is allowed: there is nothing
	// to populate, but the revision is still popped and its range silently
	// dropped. The journal ends up with no open frames.
	outer := j.validRevisions[0].id
	j.closeSnapshot(outer)
	if len(j.validRevisions) != 0 {
		t.Fatalf("after closing outermost, have %d open revisions, want 0", len(j.validRevisions))
	}
}

// TestJournalCloseSnapshotPanics asserts the LIFO precondition: closing when
// no snapshot is open, or closing a revision while a more recent snapshot is
// still open above it, must panic rather than silently mutate state. Closing
// the outermost (no-parent) frame *is* permitted and is covered in
// TestJournalFrameTracking.
func TestJournalCloseSnapshotPanics(t *testing.T) {
	j := newJournal()
	if !didPanic(func() { j.closeSnapshot(0) }) {
		t.Fatal("closing with no open snapshot should panic")
	}
	bottom := j.snapshot()
	j.snapshot() // a more recent snapshot is now on top
	if !didPanic(func() { j.closeSnapshot(bottom) }) {
		t.Fatal("closing a snapshot that is not the most recent should panic")
	}
}

// TestJournalRevertInteractions verifies the two cross-cuts between revert
// and close: reverting a parent that has absorbed closed children also
// throws away the children's entries, and reverting a child (rather than
// closing it) leaves no closed-child range on the parent.
func TestJournalRevertInteractions(t *testing.T) {
	t.Run("revertParentWithClosedChild", func(t *testing.T) {
		j := newJournal()
		outer := j.snapshot()
		j.append(tagEntry{1})

		c := j.snapshot()
		j.append(tagEntry{10})
		j.append(tagEntry{11})
		j.closeSnapshot(c)

		j.append(tagEntry{2})
		j.revertToSnapshot(outer, &StateDB{})

		if len(j.entries) != 0 || len(j.validRevisions) != 0 {
			t.Fatalf("after revert have entries=%d revisions=%d, want both 0",
				len(j.entries), len(j.validRevisions))
		}
	})
	t.Run("revertedChildLeavesNoRange", func(t *testing.T) {
		j := newJournal()
		j.snapshot()
		j.append(tagEntry{1})

		c := j.snapshot()
		j.append(tagEntry{10})
		j.revertToSnapshot(c, &StateDB{})
		j.append(tagEntry{2})

		if got := j.validRevisions[0].closedChildren; len(got) != 0 {
			t.Fatalf("reverted child should not appear in closedChildren, have %+v", got)
		}
		if tags := frameTags(j); !slices.Equal(tags, []int{1, 2}) {
			t.Fatalf("frameEntries: have %v, want [1 2]", tags)
		}
	})
}

// TestJournalStateCreationBytes exercises the slot-creation accounting in
// closeSnapshot and the matching refund returned by revertToSnapshot.
//
// It uses a real StateDB (so SetState/GetState are wired up) and walks
// through the cases the docstring promises:
//   - a slot transitioning 0→X within a frame contributes +stateBytesPerSlot;
//   - a slot transitioning X→0 within a frame whose tx-original was 0
//     contributes -stateBytesPerSlot;
//   - bytes attributed to a successful child frame are NOT re-counted by the
//     parent's own closeSnapshot (descendant filtering);
//   - when a parent is reverted, RevertToSnapshot returns the cumulative
//     bytes that successful children inside the reverted scope had emitted,
//     so the caller can undo whatever bookkeeping it kept.
func TestJournalStateCreationBytes(t *testing.T) {
	addr := common.HexToAddress("0xaa")
	keyA := common.HexToHash("0x1")
	keyB := common.HexToHash("0x2")
	nonZero := common.HexToHash("0x42")
	otherNonZero := common.HexToHash("0x99")

	// seedExistingAccount returns a fresh StateDB whose `addr` already exists
	// (so subsequent SetState calls won't journal a createObjectChange). Used
	// by storage-focused subtests so the account-creation contribution does
	// not bleed into the slot-accounting assertions.
	seedExistingAccount := func() *StateDB {
		st := newStateEnv().state
		st.getOrNewStateObject(addr)
		// The createObjectChange just journaled is at index 0; the upcoming
		// st.Snapshot() starts at index 1, so the createObject entry sits
		// outside any test scope and contributes nothing.
		return st
	}

	t.Run("slotCreationInDirectFrame", func(t *testing.T) {
		st := seedExistingAccount()
		p := st.Snapshot()
		st.SetState(addr, keyA, nonZero)
		if got := st.CloseSnapshot(p); got != stateBytesPerSlot {
			t.Fatalf("0→X creation: have %d, want %d", got, stateBytesPerSlot)
		}
	})

	t.Run("slotClearingRefundsCreation", func(t *testing.T) {
		st := seedExistingAccount()
		// Set the slot once so it has a non-zero value to clear, but make
		// the tx-original 0 by doing the set inside the tested scope.
		p := st.Snapshot()
		st.SetState(addr, keyA, nonZero)       // 0 → X (creation)
		st.SetState(addr, keyA, common.Hash{}) // X → 0 (clear)
		// Net: nothing changed; the in-frame creation was undone, so
		// closeSnapshot must report -stateBytesPerSlot to refund the
		// would-be creation, but since +stateBytesPerSlot is also
		// counted... wait: the journal stores only the FIRST prevvalue
		// per slot, which is 0 for this slot in this frame. Current
		// state is 0. Per the rules: prev==0 && current==0 — neither
		// rule fires, so 0 bytes net. That correctly reflects no growth.
		if got := st.CloseSnapshot(p); got != 0 {
			t.Fatalf("0→X→0 net: have %d, want 0", got)
		}
	})

	t.Run("childContributionNotDoubleCounted", func(t *testing.T) {
		st := seedExistingAccount()
		p := st.Snapshot()
		// Child creates slot A.
		c := st.Snapshot()
		st.SetState(addr, keyA, nonZero)
		childBytes := st.CloseSnapshot(c)
		if childBytes != stateBytesPerSlot {
			t.Fatalf("child closeSnapshot: have %d, want %d", childBytes, stateBytesPerSlot)
		}
		// Parent itself does not touch any slot. Its own closeSnapshot
		// must NOT re-count slot A — that contribution was already
		// reported by the child and lives in childStateBytes for the
		// purpose of revert refunds.
		parentBytes := st.CloseSnapshot(p)
		if parentBytes != 0 {
			t.Fatalf("parent closeSnapshot (no direct slots): have %d, want 0", parentBytes)
		}
	})

	t.Run("parentSlotChangeIndependentOfChild", func(t *testing.T) {
		st := seedExistingAccount()
		p := st.Snapshot()
		// Parent creates slot A directly.
		st.SetState(addr, keyA, nonZero)
		// Child creates a different slot B.
		c := st.Snapshot()
		st.SetState(addr, keyB, otherNonZero)
		if got := st.CloseSnapshot(c); got != stateBytesPerSlot {
			t.Fatalf("child slot B creation: have %d, want %d", got, stateBytesPerSlot)
		}
		// Parent's own closeSnapshot must report only slot A (the child's
		// slot B was filtered via the closed-child range).
		if got := st.CloseSnapshot(p); got != stateBytesPerSlot {
			t.Fatalf("parent slot A creation: have %d, want %d", got, stateBytesPerSlot)
		}
	})

	t.Run("revertReturnsAccumulatedChildBytes", func(t *testing.T) {
		st := seedExistingAccount()
		p := st.Snapshot()
		// Two successful children, each creating one slot.
		c1 := st.Snapshot()
		st.SetState(addr, keyA, nonZero)
		st.CloseSnapshot(c1)
		c2 := st.Snapshot()
		st.SetState(addr, keyB, otherNonZero)
		st.CloseSnapshot(c2)
		// Now revert the parent. The two children together emitted
		// 2 * stateBytesPerSlot, all of which should come back so the
		// caller can undo whatever was billed at close time.
		refund := st.RevertToSnapshot(p)
		want := 2 * stateBytesPerSlot
		if refund != want {
			t.Fatalf("revert refund: have %d, want %d", refund, want)
		}
	})

	t.Run("perStepComposesWhenParentAndChildShareSlot", func(t *testing.T) {
		// The interleaved-slot case that used to diverge under the
		// "first-touch + current state" rule. Per-step accounting makes
		// each SSTORE carry its own delta, so the per-frame numbers may
		// look bigger but their sum is exactly what a whole-frame walk
		// over the entire subtree would produce.
		//
		//   parent SSTORE S = X    →  entry: prev=0, new=X        → +stateBytesPerSlot
		//     child SSTORE S = 0   →  entry: prev=X, new=0, ori=0 → -stateBytesPerSlot
		//   parent SSTORE S = Y    →  entry: prev=0, new=Y        → +stateBytesPerSlot
		//
		//   child   bytes = -stateBytesPerSlot
		//   parent  bytes = +2 * stateBytesPerSlot   (two parent SSTOREs)
		//   sum           = +stateBytesPerSlot       (= net 0→Y)
		st := seedExistingAccount()
		p := st.Snapshot()
		st.SetState(addr, keyA, nonZero) // parent direct: 0 → X
		c := st.Snapshot()
		st.SetState(addr, keyA, common.Hash{}) // child: X → 0
		childBytes := st.CloseSnapshot(c)
		st.SetState(addr, keyA, otherNonZero) // parent direct: 0 → Y
		parentBytes := st.CloseSnapshot(p)

		if childBytes != -stateBytesPerSlot {
			t.Fatalf("child bytes (X→0 with origin 0): have %d, want %d",
				childBytes, -stateBytesPerSlot)
		}
		if parentBytes != 2*stateBytesPerSlot {
			t.Fatalf("parent bytes (two 0→nonZero SSTOREs): have %d, want %d",
				parentBytes, 2*stateBytesPerSlot)
		}
		if sum := childBytes + parentBytes; sum != stateBytesPerSlot {
			t.Fatalf("per-frame sum: have %d, want %d (= whole-frame net 0→Y)",
				sum, stateBytesPerSlot)
		}
	})

	t.Run("perStepComposesAcrossSiblings", func(t *testing.T) {
		// Two siblings sharing a slot: A creates, B clears. Per-step,
		// each sibling's delta is independent and the parent's sum
		// matches the whole-frame net (0).
		st := seedExistingAccount()
		p := st.Snapshot()

		a := st.Snapshot()
		st.SetState(addr, keyA, nonZero) // 0 → X
		aBytes := st.CloseSnapshot(a)

		b := st.Snapshot()
		st.SetState(addr, keyA, common.Hash{}) // X → 0
		bBytes := st.CloseSnapshot(b)

		pBytes := st.CloseSnapshot(p)

		if aBytes != stateBytesPerSlot {
			t.Fatalf("sibling A bytes: have %d, want %d", aBytes, stateBytesPerSlot)
		}
		if bBytes != -stateBytesPerSlot {
			t.Fatalf("sibling B bytes: have %d, want %d", bBytes, -stateBytesPerSlot)
		}
		if pBytes != 0 {
			t.Fatalf("parent bytes (no own slots): have %d, want 0", pBytes)
		}
		if sum := aBytes + bBytes + pBytes; sum != 0 {
			t.Fatalf("per-frame sum: have %d, want 0", sum)
		}
	})

	t.Run("perStepComposesAcrossDeepNesting", func(t *testing.T) {
		// Three-deep version of the divergence: grandparent SSTOREs S
		// before and after a child clears it. Each SSTORE contributes
		// independently and the sum equals the whole-frame net.
		st := seedExistingAccount()
		gp := st.Snapshot()
		st.SetState(addr, keyA, nonZero) // grandparent: 0 → X
		p := st.Snapshot()
		c := st.Snapshot()
		st.SetState(addr, keyA, common.Hash{}) // child: X → 0
		cBytes := st.CloseSnapshot(c)
		pBytes := st.CloseSnapshot(p)
		st.SetState(addr, keyA, otherNonZero) // grandparent: 0 → Y
		gpBytes := st.CloseSnapshot(gp)

		if cBytes != -stateBytesPerSlot {
			t.Fatalf("child bytes: have %d, want %d", cBytes, -stateBytesPerSlot)
		}
		if pBytes != 0 {
			t.Fatalf("parent (no own SSTORE) bytes: have %d, want 0", pBytes)
		}
		if gpBytes != 2*stateBytesPerSlot {
			t.Fatalf("grandparent bytes: have %d, want %d", gpBytes, 2*stateBytesPerSlot)
		}
		if sum := cBytes + pBytes + gpBytes; sum != stateBytesPerSlot {
			t.Fatalf("per-frame sum: have %d, want %d", sum, stateBytesPerSlot)
		}
	})

	t.Run("nonZeroOriginBouncesContributeNothing", func(t *testing.T) {
		// A slot whose tx-original is non-zero is not subject to creation
		// accounting at all: any in-tx transitions (X → 0 in parent, then
		// 0 → X in child) are merely rearranging pre-existing storage.
		// Both per-step deltas must be 0, and so must the per-frame sum.
		st := newStateEnv().state

		// Seed the slot with a non-zero tx-original by writing directly
		// into the origin cache, simulating storage that was committed
		// before this transaction began.
		obj := st.getOrNewStateObject(addr)
		obj.originStorage[keyA] = nonZero

		p := st.Snapshot()
		st.SetState(addr, keyA, common.Hash{}) // parent: X → 0 (origin = X)
		c := st.Snapshot()
		st.SetState(addr, keyA, nonZero) // child:  0 → X (origin = X)
		cBytes := st.CloseSnapshot(c)
		pBytes := st.CloseSnapshot(p)

		if cBytes != 0 {
			t.Fatalf("child bytes (origin non-zero): have %d, want 0", cBytes)
		}
		if pBytes != 0 {
			t.Fatalf("parent bytes (origin non-zero): have %d, want 0", pBytes)
		}
		if sum := cBytes + pBytes; sum != 0 {
			t.Fatalf("sum (net X→X): have %d, want 0", sum)
		}
	})

	t.Run("nestedDescendantsBubbleUp", func(t *testing.T) {
		st := seedExistingAccount()
		p := st.Snapshot()
		c := st.Snapshot()
		gc := st.Snapshot()
		// Grandchild creates a slot.
		st.SetState(addr, keyA, nonZero)
		if got := st.CloseSnapshot(gc); got != stateBytesPerSlot {
			t.Fatalf("grandchild close: have %d, want %d", got, stateBytesPerSlot)
		}
		// Child closes with no own direct slot work.
		if got := st.CloseSnapshot(c); got != 0 {
			t.Fatalf("child close (no own slots): have %d, want 0", got)
		}
		// If the parent is reverted now, the grandchild's bytes should
		// surface even though they were inherited via the intermediate
		// child.
		if refund := st.RevertToSnapshot(p); refund != stateBytesPerSlot {
			t.Fatalf("nested revert refund: have %d, want %d", refund, stateBytesPerSlot)
		}
	})

	t.Run("accountCreationContributesPerAccountOverhead", func(t *testing.T) {
		// CreateAccount on a fresh address journals a createObjectChange,
		// which contributes +stateBytesPerAccount.
		st := newStateEnv().state
		p := st.Snapshot()
		st.CreateAccount(addr)
		if got := st.CloseSnapshot(p); got != stateBytesPerAccount {
			t.Fatalf("account creation: have %d, want %d", got, stateBytesPerAccount)
		}
	})

	t.Run("codeCreationContributesCodeLength", func(t *testing.T) {
		// SetCode on an account whose previous code is empty contributes
		// +len(newCode); the inverse transition (non-empty → empty) refunds.
		st := seedExistingAccount()
		code := []byte{0x60, 0x00, 0x60, 0x00, 0xfd} // arbitrary 5 bytes

		p := st.Snapshot()
		st.SetCode(addr, code, 0)
		if got := st.CloseSnapshot(p); got != len(code) {
			t.Fatalf("code creation (empty → %d bytes): have %d, want %d",
				len(code), got, len(code))
		}

		// Now clear it again in a fresh frame: -len(code).
		p2 := st.Snapshot()
		st.SetCode(addr, nil, 0)
		if got := st.CloseSnapshot(p2); got != -len(code) {
			t.Fatalf("code clear (%d → empty bytes): have %d, want %d",
				len(code), got, -len(code))
		}
	})

	t.Run("createAndDeployComposesAcrossFrames", func(t *testing.T) {
		// A typical CREATE: outer frame allocates an account in a child
		// frame, the child writes code and a slot. Per-step bytes:
		//   child:  +stateBytesPerAccount + len(code) + stateBytesPerSlot
		//   outer:  0 (no own direct entries)
		//   sum  =  child total
		st := newStateEnv().state
		code := []byte{0x60, 0x42, 0x60, 0x00, 0x55} // arbitrary 5 bytes
		p := st.Snapshot()
		c := st.Snapshot()
		st.CreateAccount(addr)
		st.SetCode(addr, code, 0)
		st.SetState(addr, keyA, nonZero)
		childBytes := st.CloseSnapshot(c)
		parentBytes := st.CloseSnapshot(p)

		want := stateBytesPerAccount + len(code) + stateBytesPerSlot
		if childBytes != want {
			t.Fatalf("child bytes (account+code+slot): have %d, want %d",
				childBytes, want)
		}
		if parentBytes != 0 {
			t.Fatalf("parent bytes (no own work): have %d, want 0", parentBytes)
		}
		if sum := childBytes + parentBytes; sum != want {
			t.Fatalf("sum: have %d, want %d", sum, want)
		}
	})
}

// TestJournalCopyAndReset checks that the bookkeeping for closed-child ranges
// participates in journal.copy (deep-copied, not aliased) and journal.reset
// (cleared along with everything else).
func TestJournalCopyAndReset(t *testing.T) {
	j := newJournal()
	j.snapshot()
	j.append(tagEntry{1})
	c := j.snapshot()
	j.append(tagEntry{10})
	j.closeSnapshot(c)

	cp := j.copy()
	if !slices.Equal(cp.validRevisions[0].closedChildren, j.validRevisions[0].closedChildren) {
		t.Fatalf("copy lost closedChildren: orig=%+v copy=%+v",
			j.validRevisions[0].closedChildren, cp.validRevisions[0].closedChildren)
	}
	cp.validRevisions[0].closedChildren = append(cp.validRevisions[0].closedChildren, frameRange{99, 100})
	if len(j.validRevisions[0].closedChildren) != 1 {
		t.Fatal("original aliased copy's closedChildren slice")
	}

	j.reset()
	if len(j.entries) != 0 || len(j.validRevisions) != 0 {
		t.Fatalf("after reset have entries=%d revisions=%d, want both 0",
			len(j.entries), len(j.validRevisions))
	}
}
