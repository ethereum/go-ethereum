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

package syncx

import (
	"context"
	"errors"
	"testing"
)

func TestLockContext(t *testing.T) {
	cm := NewClosableMutex()

	// A free mutex is taken even with a context that is done, and a held one
	// is waited for until the context is done.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := cm.LockContext(ctx); err != nil {
		t.Fatalf("Failed to lock a free mutex: %v", err)
	}
	if err := cm.LockContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Unexpected error locking a held mutex: %v", err)
	}
	// It's taken once it's unlocked.
	locked := make(chan error)
	go func() {
		locked <- cm.LockContext(context.Background())
	}()
	cm.Unlock()
	if err := <-locked; err != nil {
		t.Fatalf("Failed to lock an unlocked mutex: %v", err)
	}
	// A closed mutex can't be locked. One waiting while it's closed either takes
	// it before the close or fails.
	go func() {
		err := cm.LockContext(context.Background())
		if err == nil {
			cm.Unlock()
		}
		locked <- err
	}()
	cm.Unlock()
	cm.Close()
	if err := <-locked; err != nil && !errors.Is(err, ErrClosed) {
		t.Fatalf("Unexpected error locking a mutex being closed: %v", err)
	}
	if err := cm.LockContext(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatalf("Unexpected error locking a closed mutex: %v", err)
	}
}
