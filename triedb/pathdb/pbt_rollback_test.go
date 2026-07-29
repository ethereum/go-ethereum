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
// stores contract code, and code chunks past the first 128 are
// content-addressed, so whether such a leaf belongs at the parent root
// depends on whether any other account held the same bytecode - which no
// per-account history record describes. Callers fall back to re-executing
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
