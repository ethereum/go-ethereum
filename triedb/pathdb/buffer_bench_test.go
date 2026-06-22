// Copyright 2025 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/testrand"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// makeBenchDiffSet builds a trie node set and a flat state set roughly modelling
// one block's worth of mutations: `accounts` account trie nodes and accounts,
// plus `owners` storage tries each holding `slots` nodes/slots.
func makeBenchDiffSet(accounts, owners, slots int) (*nodeSet, *stateSet) {
	var (
		nodes      = make(map[common.Hash]map[string]*trienode.Node)
		accountSet = make(map[common.Hash][]byte, accounts)
		storageSet = make(map[common.Hash]map[common.Hash][]byte, owners)
	)
	nodes[common.Hash{}] = make(map[string]*trienode.Node, accounts)
	for i := 0; i < accounts; i++ {
		blob := testrand.Bytes(100)
		nodes[common.Hash{}][string(testrand.Bytes(32))] = trienode.New(crypto.Keccak256Hash(blob), blob)
		accountSet[common.BytesToHash(testrand.Bytes(32))] = testrand.Bytes(70)
	}
	for i := 0; i < owners; i++ {
		owner := common.BytesToHash(testrand.Bytes(32))
		subset := make(map[string]*trienode.Node, slots)
		slotSet := make(map[common.Hash][]byte, slots)
		for j := 0; j < slots; j++ {
			blob := testrand.Bytes(100)
			subset[string(testrand.Bytes(32))] = trienode.New(crypto.Keccak256Hash(blob), blob)
			slotSet[common.BytesToHash(testrand.Bytes(32))] = testrand.Bytes(32)
		}
		nodes[owner] = subset
		storageSet[owner] = slotSet
	}
	return newNodeSet(nodes), newStates(accountSet, storageSet, false)
}

// commitSequential merges the node set and state set without the parallel /
// sharded fast paths, kept here purely as a benchmark baseline.
func commitSequential(b *buffer, nodes *nodeSet, states *stateSet) {
	b.layers++
	b.nodes.merge(nodes)
	b.states.merge(states)
}

func benchmarkBufferCommit(b *testing.B, parallel bool) {
	const (
		accounts = 3000
		owners   = 300 // above parallelMergeThreshold so the sharded path is exercised
		slots    = 20
	)
	diffs := make([]struct {
		nodes  *nodeSet
		states *stateSet
	}, b.N)
	for i := range diffs {
		diffs[i].nodes, diffs[i].states = makeBenchDiffSet(accounts, owners, slots)
	}
	buf := newBuffer(1<<62, nil, nil, 0) // huge limit so it never flushes

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if parallel {
			buf.commit(diffs[i].nodes, diffs[i].states)
		} else {
			commitSequential(buf, diffs[i].nodes, diffs[i].states)
		}
	}
}

func BenchmarkBufferCommitParallel(b *testing.B)   { benchmarkBufferCommit(b, true) }
func BenchmarkBufferCommitSequential(b *testing.B) { benchmarkBufferCommit(b, false) }
