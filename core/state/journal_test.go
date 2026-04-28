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
