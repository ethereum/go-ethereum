// Copyright 2022 The go-ethereum Authors
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

package tracers

import (
	"fmt"
	"sync"
	"time"
)

// stateTracker is an auxiliary tool used to cache the release functions of all
// used trace states, and to determine whether the creation of trace state needs
// to be paused in case there are too many states waiting for tracing.
type stateTracker struct {
	limit    int                // Maximum number of states allowed waiting for tracing
	head     uint64             // The number of the first trace state which isn't used up
	used     []bool             // List of flags indicating whether the trace state has been used up
	releases []StateReleaseFunc // List of trace state release functions waiting to be called
	lock     sync.RWMutex
}

// newStateTracker initializes the tracker with provided state limits and
// head state number.
func newStateTracker(limit int, head uint64) *stateTracker {
	return &stateTracker{
		limit: limit,
		head:  head,
		used:  make([]bool, limit),
	}
}

// releaseState marks the state specified by the number is released and caches
// the corresponding release functions internally.
func (t *stateTracker) releaseState(number uint64, release StateReleaseFunc) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.used[int(number-t.head)] = true
	if number == t.head {
		var count int
		for i := 0; i < len(t.used); i++ {
			if !t.used[i] {
				break
			}
			count += 1
		}
		t.head += uint64(count)
		copy(t.used, t.used[count:])
	}
	t.releases = append(t.releases, release)
}

// callReleases invokes all cached release functions.
func (t *stateTracker) callReleases() {
	t.lock.Lock()
	defer t.lock.Unlock()

	for _, release := range t.releases {
		release()
	}
	t.releases = t.releases[:0]
}

// wait blocks until the accumulated trace states are less than the limit.
func (t *stateTracker) wait(number uint64) error {
	for {
		t.lock.RLock()
		head := t.head
		t.lock.RUnlock()

		if number < t.head {
			return fmt.Errorf("invalid state number %d head %d", number, t.head)
		}
		if int(number-head) < t.limit {
			return nil
		}
		time.Sleep(time.Millisecond * 100)
	}
}
