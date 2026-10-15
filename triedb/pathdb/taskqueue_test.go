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
	"testing"
	"time"
)

// TestTaskQueue checks the tasks are run in the order of scheduling, and the
// scheduling blocks once the pending tasks reach the limit.
func TestTaskQueue(t *testing.T) {
	var (
		q       = newTaskQueue()
		ran     []int
		release = make(chan struct{})
	)
	// The first task blocks the thread, filling up the queue behind it
	q.schedule(func() { <-release })
	for i := 0; i < taskQueueLimit; i++ {
		q.schedule(func() { ran = append(ran, i) })
	}
	blocked := make(chan struct{})
	go func() {
		q.schedule(func() { ran = append(ran, taskQueueLimit) })
		close(blocked)
	}()
	select {
	case <-blocked:
		t.Fatal("scheduling beyond the limit did not block")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	select {
	case <-blocked:
	case <-time.After(5 * time.Second):
		t.Fatal("scheduling did not resume")
	}
	q.wait()
	if len(ran) != taskQueueLimit+1 {
		t.Fatalf("task count mismatch, want %d, got %d", taskQueueLimit+1, len(ran))
	}
	for i, v := range ran {
		if v != i {
			t.Fatalf("task order mismatch at %d, got %d", i, v)
		}
	}
	// The thread terminates once drained and is relaunched on demand
	q.schedule(func() { ran = append(ran, -1) })
	q.wait()
	if ran[len(ran)-1] != -1 {
		t.Fatal("task not run after relaunch")
	}
}
