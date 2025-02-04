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

// Package syncx contains exotic synchronization primitives.
package syncx

import (
	"sync"
)

// ClosableMutex is a mutex that can be closed. Once closed, it cannot be locked again.
type ClosableMutex struct {
	mu     sync.Mutex // Protects the following fields
	closed bool
	ch     chan struct{}
}

// NewClosableMutex creates a new closable mutex.
func NewClosableMutex() *ClosableMutex {
	ch := make(chan struct{}, 1)
	ch <- struct{}{}
	return &ClosableMutex{ch: ch}
}

// TryLock attempts to acquire the lock. Returns true if successful, false if the lock is closed or unavailable.
func (cm *ClosableMutex) TryLock() bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.closed {
		return false
	}

	select {
	case <-cm.ch:
		return true
	default:
		return false
	}
}

// MustLock acquires the lock. Panics if the lock is already closed.
func (cm *ClosableMutex) MustLock() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.closed {
		panic("mutex closed")
	}
	select {
	case <-cm.ch:
		return
	default:
		panic("mutex is already locked")
	}
}

// Unlock releases the lock. Panics if the lock is already closed or if called without holding the lock.
func (cm *ClosableMutex) Unlock() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.closed {
		panic("Unlock after Close")
	}
	select {
	case cm.ch <- struct{}{}:
	default:
		panic("Unlock of already-unlocked ClosableMutex")
	}
}

// Close closes the mutex, preventing further lock operations. Panics if called on an already-closed mutex.
func (cm *ClosableMutex) Close() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.closed {
		panic("Close of already-closed ClosableMutex")
	}
	cm.closed = true
	close(cm.ch) // Closing the channel will cause subsequent send operations to panic
}
