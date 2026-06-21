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

// ParallelTxGroupingByStorageOverlap controls grouping in Process and
// BuildTransactionStorageParallelGroups. Despite the name, grouping uses
// declared address disjointness (from, to, access-list addresses). When true
// (default), txs whose declared address sets are pairwise disjoint may share a
// wave. When false, each tx is its own group [[0],[1],...].
var ParallelTxGroupingByStorageOverlap = true

// ParallelTxWaveExecution runs txs in the same wave concurrently when true: one
// goroutine and one vm.EVM per tx. StateDB, GasPool, and any tracers must be safe
// for concurrent use (or txs must be disjoint and externally synchronized).
// When false (default), txs in a wave still run strictly in ascending index order
// on the shared EVM — consensus-compatible with sequential Ethereum execution.
var ParallelTxWaveExecution = true

// ParallelTxDebug enables debug logging for parallel transaction execution.
var ParallelTxDebug = false
