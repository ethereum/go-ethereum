package syncx

import (
	"testing"
	"time"
)

// TestClosableMutex_TryLock tests the TryLock functionality of ClosableMutex.
// It verifies that TryLock succeeds when the mutex is unlocked, fails when already locked,
// succeeds after unlocking, and fails after the mutex is closed.
func TestClosableMutex_TryLock(t *testing.T) {
	cm := NewClosableMutex()
	if !cm.TryLock() {
		t.Fatal("expected TryLock to succeed")
	}
	if cm.TryLock() {
		t.Fatal("expected TryLock to fail when already locked")
	}
	cm.Unlock()
	if !cm.TryLock() {
		t.Fatal("expected TryLock to succeed after unlock")
	}
	cm.Close()
	if cm.TryLock() {
		t.Fatal("expected TryLock to fail after close")
	}
}

// TestClosableMutex_MustLock tests the MustLock functionality of ClosableMutex.
// It verifies that MustLock succeeds when the mutex is unlocked and panics when already locked.
func TestClosableMutex_MustLock(t *testing.T) {
	cm := NewClosableMutex()
	cm.MustLock()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected MustLock to panic when already locked")
		}
	}()
	cm.MustLock()
}

// TestClosableMutex_Unlock tests the Unlock functionality of ClosableMutex.
// It verifies that Unlock succeeds when the mutex is locked and panics when already unlocked.
func TestClosableMutex_Unlock(t *testing.T) {
	cm := NewClosableMutex()
	cm.MustLock()
	cm.Unlock()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected Unlock to panic when already unlocked")
		}
	}()
	cm.Unlock()
}

// TestClosableMutex_Close tests the Close functionality of ClosableMutex.
// It verifies that Close succeeds when the mutex is locked and panics when already closed.
func TestClosableMutex_Close(t *testing.T) {
	cm := NewClosableMutex()
	cm.MustLock()
	cm.Close()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected Close to panic when already closed")
		}
	}()
	cm.Close()
}

// TestClosableMutex_Concurrent tests the concurrent behavior of ClosableMutex.
// It verifies that TryLock fails when the mutex is locked by another goroutine
// and succeeds after the other goroutine unlocks it.
func TestClosableMutex_Concurrent(t *testing.T) {
	cm := NewClosableMutex()
	done := make(chan struct{})
	go func() {
		cm.MustLock()
		time.Sleep(100 * time.Millisecond) // Simulate work while holding the lock
		cm.Unlock()
		close(done)
	}()
	time.Sleep(50 * time.Millisecond) // Wait for the goroutine to acquire the lock
	if cm.TryLock() {
		t.Fatal("expected TryLock to fail when locked by another goroutine")
	}
	<-done // Wait for the goroutine to release the lock
	if !cm.TryLock() {
		t.Fatal("expected TryLock to succeed after other goroutine unlocks")
	}
	cm.Unlock()
}
