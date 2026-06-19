// Copyright 2026 go-ethereum Authors
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

package trie

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

// TestProbeHeightFullNodeBranch is a regression test for an over-estimate bug in
// nodeHeight's *fullNode case. The previous guard
//
//	if maxH+1 > maxHeight { return maxHeight + 1 }
//
// fired once the running max height reached maxHeight via one child, so the next
// hashNode child made a genuine height-3 branch report as height 4. Combined with
// the "archive only height == 3" predicate, dense branch-heavy tries (notably the
// account trie, whose height-3 nodes are all multi-child branches) archived
// nothing. Pre-fix probeHeight returns 4 here; post-fix it must return 3.
//
// The trie built below has a height-3 ROOT branch:
//
//	root  (branch @ nibble0, children at 0 and 1)            height 3
//	 ├─ child0 (branch @ nibble1)  ├─ leafA ├─ leafB         height 2
//	 └─ child1 (branch @ nibble1)  ├─ leafC ├─ leafD         height 2
//
// Values are 40 bytes so leaves are NOT embedded; root's children are hashNodes,
// which is exactly what exercises the buggy branch of nodeHeight.
func TestProbeHeightFullNodeBranch(t *testing.T) {
	big := func(b byte) []byte { return bytes.Repeat([]byte{b}, 40) }

	tr := NewEmpty(nil)
	keys := [][]byte{
		{0x00, 0x11, 0x11, 0x11}, // nibbles 0,0,...
		{0x01, 0x22, 0x22, 0x22}, // nibbles 0,1,...
		{0x10, 0x33, 0x33, 0x33}, // nibbles 1,0,...
		{0x11, 0x44, 0x44, 0x44}, // nibbles 1,1,...
	}
	for i, k := range keys {
		if err := tr.Update(k, big(byte('A'+i))); err != nil {
			t.Fatalf("update %x: %v", k, err)
		}
	}
	root, nodes := tr.Commit(false)
	if nodes == nil {
		t.Fatal("nil node set after commit")
	}

	// Persist the committed nodes under account-trie path keys so the archiver's
	// raw-DB reader (readNodeBlob -> ReadAccountTrieNode) can resolve them.
	raw := rawdb.NewMemoryDatabase()
	for path, n := range nodes.Nodes {
		if n == nil || len(n.Blob) == 0 { // skip deletions
			continue
		}
		rawdb.WriteAccountTrieNode(raw, []byte(path), n.Blob)
	}
	a := &Archiver{db: raw}

	if got := a.probeHeight(common.Hash{}, nil, root, 3); got != 3 {
		t.Fatalf("probeHeight(height-3 root branch) = %d, want 3 "+
			"(a height-3 branch must be detected as height 3, not over-estimated)", got)
	}
}
