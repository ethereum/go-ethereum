// Copyright 2015 The go-ethereum Authors
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

package rlpx

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

var errAcquireTimeout = errors.New("acquisition timeout")

// bufSema is a counting semaphore.
type bufSema struct {
	val, cap, waiting uint32
	wakeup            chan struct{}
}

func newBufSema(cap uint32) *bufSema {
	return &bufSema{cap: cap, val: cap, wakeup: make(chan struct{})}
}

func (sem *bufSema) get() uint32 {
	return atomic.LoadUint32(&sem.val)
}

// release increments sem, potentially unblocking a call to
// waitAcquire if there is one. release never blocks.
func (sem *bufSema) release(n uint32) {
	new := atomic.AddUint32(&sem.val, n)
	if new > sem.cap {
		panic(fmt.Sprintf("semaphore count %d exceeds cap after release(%d)", new, n))
	}
	// Wake up a pending waitAcquire call if there is one.
	if atomic.CompareAndSwapUint32(&sem.waiting, 1, 0) {
		sem.wakeup <- struct{}{}
	}
}

// waitAcquire decrements the semaphore by n. If less than
// n units are available, waitAcquire blocks until release is called.
// It may only be called from one goroutine at a time.
func (sem *bufSema) waitAcquire(n uint32, timeout time.Duration) error {
	if n > sem.cap {
		return fmt.Errorf("requested amount %d exceeds semaphore cap of %d", n, sem.cap)
	}
	var timer *time.Timer
	for {
		// Set the waiting flag so release will try to wake us after
		// incrementing sem.val.
		if !atomic.CompareAndSwapUint32(&sem.waiting, 0, 1) {
			panic("concurrent call to waitAcquire")
		}
		// Decrement if sem.val if possible.
		if atomic.LoadUint32(&sem.val) >= n {
			atomic.AddUint32(&sem.val, ^(n - 1))
			// Gobble up wakeup signal in case release decremented sem.waiting.
			if !atomic.CompareAndSwapUint32(&sem.waiting, 1, 0) {
				<-sem.wakeup
			}
			return nil
		}
		// Start the timeout on the first iteration.
		if timer == nil {
			timer = time.NewTimer(timeout)
			defer timer.Stop()
		}
		select {
		case <-sem.wakeup:
			// Woken by release. It has decremented sem.waiting back to zero.
		case <-timer.C:
			// Gobble up wakeup signal in case release decremented sem.waiting.
			if !atomic.CompareAndSwapUint32(&sem.waiting, 1, 0) {
				<-sem.wakeup
			}
			return errAcquireTimeout
		}
	}
	return nil
}
