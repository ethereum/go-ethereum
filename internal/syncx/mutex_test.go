package syncx

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestTryWithContext(t *testing.T) {
	mu := NewClosableMutex()
	var wg sync.WaitGroup
	res := []bool{false, false}
	waiter := func(id int, waitTime time.Duration) {
		defer wg.Done()
		ctx, _ := context.WithTimeout(context.Background(), waitTime)
		if mu.TryLockWithContext(ctx) {
			mu.Unlock()
			res[id] = true
		}
	}
	// Obtain the lock
	if !mu.TryLock() {
		t.Fatalf("lock failed")
	}
	// Launch goroutines
	wg.Add(2)
	go waiter(0, 100*time.Millisecond) // This one should cancel
	go waiter(1, 5*time.Second)        // This one should make it
	// Sleep for a bit.
	time.Sleep(1 * time.Second)
	mu.Unlock()
	wg.Wait()
	if have, want := res[0], false; have != want {
		t.Fatalf("have %v want %v", have, want)
	}
	if have, want := res[1], true; have != want {
		t.Fatalf("have %v want %v", have, want)
	}
}
