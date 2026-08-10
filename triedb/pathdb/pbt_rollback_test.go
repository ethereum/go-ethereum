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
	"strings"
	"testing"
)

// TestPBTRollbackUnsupported pins that the binary tree refuses to roll its
// persisted state back, and says why.
//
// Reverting works by replaying pre-transition account and storage values
// through the trie until the parent root reappears. The binary tree also
// stores contract code, and every code chunk is content-addressed, so
// whether such a leaf belongs at the parent root depends on whether any
// other account held the same bytecode - which no per-account history
// record describes. Callers fall back to re-executing
// blocks forward, so a false answer here costs reach, not correctness.
//
// The merkle half of this test is what makes the binary half mean anything:
// it establishes that the very same root would otherwise be reported as
// recoverable.
func TestPBTRollbackUnsupported(t *testing.T) {
	// Redefine the diff layer depth allowance for faster testing.
	maxDiffLayers = 4
	defer func() {
		maxDiffLayers = 128
	}()

	merkle := newTester(t, &testerConfig{layers: 12})
	defer merkle.release()

	target := merkle.roots[merkle.bottomIndex()-1]
	if !merkle.db.Recoverable(target) {
		t.Fatal("a state below the disk layer is not recoverable even on merkle; the binary tree case below would prove nothing")
	}

	binary := newTester(t, &testerConfig{layers: 12, isPBT: true})
	defer binary.release()

	target = binary.roots[binary.bottomIndex()-1]
	if binary.db.Recoverable(target) {
		t.Fatal("binary tree reports a state as recoverable, but reverting it is not possible")
	}
	err := binary.db.Recover(target)
	if err == nil {
		t.Fatal("binary tree accepted a rollback request")
	}
	if !strings.Contains(err.Error(), "binary tree") {
		t.Fatalf("rollback refusal does not name its cause: %v", err)
	}
}

// TestPBTSyncUnsupported pins that a binary tree database refuses to be handed
// a synced state.
//
// Snap sync delivers trie nodes and rebuilds flat state from them afterwards by
// walking the trie and recovering which account each leaf belongs to. That walk
// is impossible here: leaves are keyed by a hash of the address and no
// preimages are kept, so flat state cannot be regenerated. Accepting a sync
// would leave a database whose flat state is empty while attested complete -
// which reads back as a chain with no accounts in it.
//
// The refusal sits in resetForReactivation so it covers Enable and
// AdoptSyncedState alike, which is what this checks: both entry points, not
// just the one a caller happens to use.
func TestPBTSyncUnsupported(t *testing.T) {
	binary := newTester(t, &testerConfig{layers: 4, isPBT: true})
	defer binary.release()

	root := binary.roots[len(binary.roots)-1]

	if err := binary.db.AdoptSyncedState(root); err == nil {
		t.Fatal("binary tree accepted a synced state")
	} else if !strings.Contains(err.Error(), "binary tree") {
		t.Fatalf("sync refusal does not name its cause: %v", err)
	}
	if err := binary.db.Enable(root); err == nil {
		t.Fatal("binary tree accepted a state reset through Enable")
	} else if !strings.Contains(err.Error(), "binary tree") {
		t.Fatalf("enable refusal does not name its cause: %v", err)
	}

	// The merkle half, so the refusals above are known to be about the tree.
	// Enable validates the root against the stored one, so merkle rejects this
	// call too - but for a different reason, and that is the distinction being
	// drawn: the binary tree refuses the operation itself, merkle only refuses
	// this particular argument.
	merkle := newTester(t, &testerConfig{layers: 4})
	defer merkle.release()

	if err := merkle.db.Enable(merkle.roots[len(merkle.roots)-1]); err != nil {
		if strings.Contains(err.Error(), "binary tree") {
			t.Fatalf("merkle refused with the binary tree's reason: %v", err)
		}
	}
}
