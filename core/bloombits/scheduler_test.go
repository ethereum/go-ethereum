package bloombits

import (
	"bytes"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
)

// Tests that the scheduler can deduplicate and forward retrieval requests to
// underlying fetchers and serve responses back, irrelevant of the concurrency
// of the requesting clients or serving data fetchers.
func TestSchedulerSingleClientSingleFetcher(t *testing.T) { testScheduler(t, 1, 1, 5000) }
func TestSchedulerSingleClientMultiFetcher(t *testing.T)  { testScheduler(t, 1, 10, 5000) }
func TestSchedulerMultiClientSingleFetcher(t *testing.T)  { testScheduler(t, 10, 1, 5000) }
func TestSchedulerMultiClientMultiFetcher(t *testing.T)   { testScheduler(t, 10, 10, 5000) }

func testScheduler(t *testing.T, clients int, fetchers int, requests int) {
	t.Parallel()
	f := newScheduler(0)

	// Create a batch of handler goroutines that respond to bloom bit requests and
	// deliver them to the scheduler.
	var fetchPend sync.WaitGroup
	fetchPend.Add(fetchers)
	defer fetchPend.Wait()

	fetch := make(chan *request, 16)
	defer close(fetch)

	var delivered atomic.Uint32
	for i := 0; i < fetchers; i++ {
		go func() {
			defer fetchPend.Done()

			for req := range fetch {
				delivered.Add(1)

				f.deliver([]uint64{
					req.section + uint64(requests), // Non-requested data (ensure it doesn't go out of bounds)
					req.section,                    // Requested data
					req.section,                    // Duplicated data (ensure it doesn't double close anything)
				}, [][]byte{
					{},
					new(big.Int).SetUint64(req.section).Bytes(),
					new(big.Int).SetUint64(req.section).Bytes(),
				})
			}
		}()
	}
	// Start a batch of goroutines to concurrently run scheduling tasks
	quit := make(chan struct{})

	var pend sync.WaitGroup
	pend.Add(clients)

	for i := 0; i < clients; i++ {
		go func() {
			defer pend.Done()

			in := make(chan uint64, 16)
			out := make(chan []byte, 16)

			f.run(in, fetch, out, quit, &pend)

			go func() {
				for j := 0; j < requests; j++ {
					in <- uint64(j)
				}
				close(in)
			}()
			b := new(big.Int)
			for j := 0; j < requests; j++ {
				bits := <-out
				if want := b.SetUint64(uint64(j)).Bytes(); !bytes.Equal(bits, want) {
					t.Errorf("vector %d: delivered content mismatch: have %x, want %x", j, bits, want)
				}
			}
		}()
	}
	pend.Wait()

	if have := delivered.Load(); int(have) != requests {
		t.Errorf("request count mismatch: have %v, want %v", have, requests)
	}
}

// TestSchedulerEdgeCases tests edge cases for the scheduler.
func TestSchedulerEdgeCases(t *testing.T) {
	t.Parallel()

	// Test with zero sections
	_, err := newScheduler(0)
	if err == nil {
		t.Fatal("expected error for zero sections, got nil")
	}

	// Test with non-multiple of 8 sections
	_, err = newScheduler(7)
	if err == nil {
		t.Fatal("expected error for non-multiple of 8 sections, got nil")
	}

	// Test with valid sections
	scheduler := newScheduler(8)
	if scheduler == nil {
		t.Fatal("failed to create scheduler with valid sections")
	}

	// Test running the scheduler with invalid channels
	in := make(chan uint64, 16)
	out := make(chan []byte, 16)
	quit := make(chan struct{})
	var pend sync.WaitGroup
	pend.Add(1)
	go func() {
		defer pend.Done()
		scheduler.run(in, nil, out, quit, &pend)
	}()

	// Test sending invalid data to the scheduler
	in <- 1
	close(in)
	pend.Wait()
}
