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

// ClosableMutex is a mutex that can also be closed.
// Once closed, it can never be taken again.
type ClosableMutex struct {
	ch chan struct{}
}

// NewClosableMutex creates a new closable mutex.
func NewClosableMutex() *ClosableMutex {
	ch := make(chan struct{}, 1)
	ch <- struct{}{}
	return &ClosableMutex{ch: ch}
}

// TryLock attempts to lock cm.
// If the mutex is closed, TryLock returns false.
func (cm *ClosableMutex) TryLock() bool {
	select {
	case <-cm.ch:
		return true
	default:
		return false
	}
}

// MustLock locks cm.
// If the mutex is closed, MustLock panics.
func (cm *ClosableMutex) MustLock() {
	select {
	case <-cm.ch:
		return
	default:
		panic("mutex is already locked")
	}
}

// Unlock unlocks cm.
func (cm *ClosableMutex) Unlock() {
	select {
	case cm.ch <- struct{}{}:
	default:
		panic("Unlock of already-unlocked ClosableMutex")
	}
}

// Close locks the mutex, then closes it.
func (cm *ClosableMutex) Close() {
	select {
	case <-cm.ch:
	default:
	}
	close(cm.ch)
}
