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

// Engine micro-benchmarks. First recorded against the EIP-7864 engine as the
// G2 baseline before its replacement by the EIP-8297 engine; the benchmark
// names are stable across the swap so benchstat can compare directly.

import (
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

func benchAddr(i int) common.Address {
	var a common.Address
	binary.BigEndian.PutUint64(a[12:], uint64(i)+1)
	return a
}

func benchAccount(i int) *types.StateAccount {
	return &types.StateAccount{
		Nonce:    uint64(i),
		Balance:  uint256.NewInt(uint64(i) + 1),
		CodeHash: types.EmptyCodeHash[:],
	}
}

// benchTrie returns an in-memory binary trie populated with n accounts, each
// carrying two storage slots (one header slot, one overflow slot).
func benchTrie(b *testing.B, n int) *bintrie.BinaryTrie {
	b.Helper()
	db := triedb.NewDatabase(rawdb.NewMemoryDatabase(), triedb.PBTDefaults)
	tr, err := bintrie.NewBinaryTrie(types.EmptyBinaryHash, db)
	if err != nil {
		b.Fatal(err)
	}
	var slot common.Hash
	for i := 0; i < n; i++ {
		addr := benchAddr(i)
		if err := tr.UpdateAccount(addr, benchAccount(i), 0, nil); err != nil {
			b.Fatal(err)
		}
		slot[31] = 1 // header slot 1
		if err := tr.UpdateStorage(addr, slot[:], slot[:]); err != nil {
			b.Fatal(err)
		}
		slot[31] = 100 // overflow slot 100
		if err := tr.UpdateStorage(addr, slot[:], slot[:]); err != nil {
			b.Fatal(err)
		}
	}
	return tr
}

// BenchmarkEngineInsertAccount measures amortized account insertion into a
// growing in-memory trie.
func BenchmarkEngineInsertAccount(b *testing.B) {
	db := triedb.NewDatabase(rawdb.NewMemoryDatabase(), triedb.PBTDefaults)
	tr, err := bintrie.NewBinaryTrie(types.EmptyBinaryHash, db)
	if err != nil {
		b.Fatal(err)
	}
	i := 0
	for b.Loop() {
		if err := tr.UpdateAccount(benchAddr(i), benchAccount(i), 0, nil); err != nil {
			b.Fatal(err)
		}
		i++
	}
}

// BenchmarkEngineGetAccount measures resolved in-memory account reads from a
// 100k-account trie.
func BenchmarkEngineGetAccount(b *testing.B) {
	tr := benchTrie(b, 100_000)
	i := 0
	b.ResetTimer()
	for b.Loop() {
		if _, err := tr.GetAccount(benchAddr(i % 100_000)); err != nil {
			b.Fatal(err)
		}
		i++
	}
}

// BenchmarkEngineUpdateStorage measures overflow-slot writes on a 100k-account
// trie.
func BenchmarkEngineUpdateStorage(b *testing.B) {
	tr := benchTrie(b, 100_000)
	var slot, val common.Hash
	slot[31] = 200
	i := 0
	b.ResetTimer()
	for b.Loop() {
		binary.BigEndian.PutUint64(val[24:], uint64(i)+1)
		if err := tr.UpdateStorage(benchAddr(i%100_000), slot[:], val[:]); err != nil {
			b.Fatal(err)
		}
		i++
	}
}

// BenchmarkEngineHashDirty measures root recomputation after touching 1k
// accounts of a 100k-account trie (the steady-state per-block hash workload).
func BenchmarkEngineHashDirty(b *testing.B) {
	tr := benchTrie(b, 100_000)
	tr.Hash()
	i := 0
	b.ResetTimer()
	for b.Loop() {
		b.StopTimer()
		for j := 0; j < 1000; j++ {
			acc := benchAccount(i + j)
			acc.Nonce = uint64(i + j + 7)
			if err := tr.UpdateAccount(benchAddr((i+j)%100_000), acc, 0, nil); err != nil {
				b.Fatal(err)
			}
		}
		i += 1000
		b.StartTimer()
		tr.Hash()
	}
}

// BenchmarkEngineCommit measures Commit (node collection + serialization) of
// 1k dirty accounts on a 100k-account trie. The trie is copied per iteration
// so every Commit sees the same dirty set.
func BenchmarkEngineCommit(b *testing.B) {
	tr := benchTrie(b, 100_000)
	tr.Hash()
	for j := 0; j < 1000; j++ {
		acc := benchAccount(j)
		acc.Nonce = uint64(j + 7)
		if err := tr.UpdateAccount(benchAddr(j), acc, 0, nil); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for b.Loop() {
		b.StopTimer()
		cp := tr.Copy()
		b.StartTimer()
		cp.Commit(false)
	}
}
