package syncx

import (
	"testing"
	"time"
)

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

func TestClosableMutex_Concurrent(t *testing.T) {
	cm := NewClosableMutex()
	done := make(chan struct{})
	go func() {
		cm.MustLock()
		time.Sleep(100 * time.Millisecond)
		cm.Unlock()
		close(done)
	}()
	time.Sleep(50 * time.Millisecond)
	if cm.TryLock() {
		t.Fatal("expected TryLock to fail when locked by another goroutine")
	}
	<-done
	if !cm.TryLock() {
		t.Fatal("expected TryLock to succeed after other goroutine unlocks")
	}
	cm.Unlock()
}
