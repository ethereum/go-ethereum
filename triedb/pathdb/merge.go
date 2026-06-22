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
	"runtime"
	"sync"
)

// parallelMergeThreshold is the minimum number of storage owners required for
// the buffer merge to fan out across goroutines. Below this, the goroutine
// setup overhead outweighs the benefit and the merge runs single-threaded.
const parallelMergeThreshold = 256

// mergeWorkers returns the number of goroutines to use for a parallel merge of
// the given number of owners. It is capped well below the core count because
// the node-set and state-set merges already run concurrently with each other,
// so each should only claim a fraction of the available parallelism.
func mergeWorkers(owners int) int {
	n := runtime.GOMAXPROCS(0)
	if n > 4 {
		n = 4
	}
	if n > owners {
		n = owners
	}
	if n < 1 {
		n = 1
	}
	return n
}

// parallelChunks splits [0,total) into `workers` contiguous ranges and invokes
// fn for each concurrently, passing the chunk index and its [lo,hi) bounds. It
// blocks until every chunk has been processed. fn must be safe to run
// concurrently for disjoint ranges.
func parallelChunks(total, workers int, fn func(idx, lo, hi int)) {
	var wg sync.WaitGroup
	chunk := (total + workers - 1) / workers
	for i := 0; i < workers; i++ {
		lo := i * chunk
		if lo >= total {
			break
		}
		hi := lo + chunk
		if hi > total {
			hi = total
		}
		wg.Add(1)
		go func(idx, lo, hi int) {
			defer wg.Done()
			fn(idx, lo, hi)
		}(i, lo, hi)
	}
	wg.Wait()
}
