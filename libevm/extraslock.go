// Copyright 2025 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

package libevm

import (
	"errors"
	"sync"
	"sync/atomic"
)

var (
	extrasMu     sync.Mutex
	extrasHandle atomic.Uint64
)

// An ExtrasLock is a handle that proves a current call to
// [WithTemporaryExtrasLock].
type ExtrasLock struct {
	handle *uint64
}

// WithTemporaryExtrasLock takes a global lock and calls `fn` with a handle that
// can be used to prove that the lock is held. All package-specific temporary
// overrides require this proof.
//
// WithTemporaryExtrasLock MUST NOT be used on a live chain. It is solely
// intended for off-chain consumers that require access to extras.
func WithTemporaryExtrasLock(fn func(lock ExtrasLock) error) error {
	extrasMu.Lock()
	defer func() {
		extrasHandle.Add(1)
		extrasMu.Unlock()
	}()

	v := extrasHandle.Load()
	return fn(ExtrasLock{&v})
}

// ErrExpiredExtrasLock is returned by [ExtrasLock.Verify] if the lock has been
// persisted beyond the call to [WithTemporaryExtrasLock] that created it.
var ErrExpiredExtrasLock = errors.New("libevm.ExtrasLock expired")

// Verify verifies that the lock is valid.
func (l ExtrasLock) Verify() error {
	if *l.handle != extrasHandle.Load() {
		return ErrExpiredExtrasLock
	}
	return nil
}
