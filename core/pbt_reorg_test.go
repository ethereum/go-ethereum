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

package core

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

// TestPBTReportsRollbackUnsupported pins the chain-level half of how the binary
// tree handles reorgs: it reports itself as unable to roll state back, which is
// what routes every caller onto re-executing the winning branch forward from
// the closest ancestor whose state is still live.
//
// Reverting works by replaying pre-transition account and storage values
// through the trie until the parent root reappears, which is only valid while
// the trie is a pure function of those values. The binary tree also stores
// contract code, and code chunks past the first 128 are content-addressed, so
// whether such a leaf belongs at the parent root depends on whether any other
// account held the same bytecode - a question no per-account history answers.
//
// `stateRecoverable` returning false is therefore the honest answer, not a
// degradation: `recoverAncestors` and `insertSideChain` both already treat
// rollback as a fast path and fall back to re-execution.
//
// Scope: this pins the reporting and the refusal. That the refusal is
// meaningful - that the same root would otherwise have been accepted - is
// pinned in pathdb.TestPBTRollbackUnsupported against a merkle database. A
// reorg whose fork point sits below the persisted disk layer cannot be built
// here at all: GenerateChain force-commits every block's trie
// (`chain_makers.go:432`), which pathdb rejects, so no core test can currently
// produce a multi-block binary-tree chain.
func TestPBTReportsRollbackUnsupported(t *testing.T) {
	var (
		genesis = pbtSchemeGenesis()
		engine  = beacon.New(ethash.NewFaker())
	)
	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), genesis, engine, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	if !chain.TrieDB().IsPBT() {
		t.Fatal("chain is not running on the binary tree")
	}
	root := chain.CurrentBlock().Root

	// Live state is still served; only rolling back is out of reach.
	if _, err := chain.StateAt(root); err != nil {
		t.Fatalf("live state is unavailable on the binary tree: %v", err)
	}
	if chain.stateRecoverable(root) {
		t.Fatal("chain reports its state as recoverable; the binary tree cannot revert")
	}
	err = chain.triedb.Recover(root)
	if err == nil {
		t.Fatal("binary tree accepted a rollback request")
	}
	if !strings.Contains(err.Error(), "binary tree") {
		t.Fatalf("rollback refusal does not name its cause: %v", err)
	}
}
