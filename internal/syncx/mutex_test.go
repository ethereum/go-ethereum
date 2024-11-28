package syncx

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCancel(t *testing.T) {
	t.Skip("not a good test, also time-consuming")
	mu := NewClosableMutex()
	var wg sync.WaitGroup
	wg.Add(3)
	waiter := func(id int, waitTime time.Duration) {
		defer wg.Done()
		ctx, _ := context.WithTimeout(context.Background(), waitTime)
		if mu.TryLockWithContext(ctx) {
			fmt.Printf("%d. Sleeping\n", id)
			time.Sleep(10 * time.Second)
			fmt.Printf("%d. Waking\n", id)
			mu.Unlock()
		} else {
			fmt.Printf("%d. Cancelling\n", id)
		}
	}

	go waiter(1, 5*time.Second)
	time.Sleep(100 * time.Millisecond)
	go waiter(2, 5*time.Second)
	go waiter(3, 15*time.Second)
	wg.Wait()
}
