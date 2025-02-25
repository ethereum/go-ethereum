// Copyright 2021 The go-ethereum Authors
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

package syncx

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestClosableMutex tests the ClosableMutex type.
func TestClosableMutex(t *testing.T) {
	t.Run("basic lock/unlock", func(t *testing.T) {
		cm := NewClosableMutex()
		if !cm.TryLock() {
			t.Error("TryLock should succeed on new mutex")
		}
		cm.Unlock()
	})

	t.Run("multiple lock/unlock", func(t *testing.T) {
		cm := NewClosableMutex()
		for i := 0; i < 3; i++ {
			if !cm.TryLock() {
				t.Error("TryLock should succeed")
			}
			cm.Unlock()
		}
	})

	t.Run("concurrent lock/unlock", func(t *testing.T) {
		cm := NewClosableMutex()
		var wg sync.WaitGroup

		// lockCount is used to count how many goroutines have acquired the lock
		var lockCount atomic.Int32

		// Start a goroutine that holds the lock for a short time
		wg.Add(2)
		go func() {
			defer wg.Done()
			if !cm.TryLock() {
				t.Error("First TryLock should succeed")
				return
			}
			time.Sleep(3 * time.Second)
			lockCount.Add(1)
			cm.Unlock()
		}()

		go func() {
			defer wg.Done()
			// if main goroutine acquires the lock, it will increment lockCount
			// so check the lockCount at the second to see if the main goroutine has acquired the lock
			time.Sleep(2 * time.Second)
			if lockCount.Load() != 0 {
				t.Error("Second TryLock should not succeed while the first goroutine holds the lock")
			}
		}()

		// Try to acquire the lock while it's held
		time.Sleep(time.Second) // Wait for the first goroutine to acquire the lock

		// NOTE: will block here until the first goroutine releases the lock
		if !cm.TryLock() {
			t.Error("Second TryLock should block here unitl the first goroutine releases the lock, but it didn't")
			cm.Unlock()
		}

		lockCount.Add(1)

		// the main goroutine will increment lockCount once the first gourotine releases the lock,
		// so lockCount should be 2
		if lockCount.Load() != 2 {
			t.Error("main gourotine should have acquired the lock")
		}

		wg.Wait()
	})

	t.Run("must lock", func(t *testing.T) {
		cm := NewClosableMutex()
		cm.MustLock()
		cm.Unlock()
	})

	t.Run("close mutex", func(t *testing.T) {
		cm := NewClosableMutex()
		cm.Close()
		if cm.TryLock() {
			t.Error("TryLock should fail on closed mutex")
		}
	})

	t.Run("panic on unlock of unlocked mutex", func(t *testing.T) {
		cm := NewClosableMutex()
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic on Unlock of unlocked mutex")
			}
		}()
		cm.Unlock() // Should panic
	})

	t.Run("panic on must lock of closed mutex", func(t *testing.T) {
		cm := NewClosableMutex()
		cm.Close()
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic on MustLock of closed mutex")
			}
		}()
		cm.MustLock() // Should panic
	})

	t.Run("panic on close of closed mutex", func(t *testing.T) {
		cm := NewClosableMutex()
		cm.Close()
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic on Close of closed mutex")
			}
		}()
		cm.Close() // Should panic
	})
}
