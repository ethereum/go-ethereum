package heimdall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/network"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/checkpoint"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/milestone"

	"github.com/stretchr/testify/require"
)

// HttpHandlerFake defines the handler functions required to serve
// requests to the mock heimdal server for specific functions. Add more handlers
// according to requirements.
type HttpHandlerFake struct {
	handleFetchCheckpoint         http.HandlerFunc
	handleFetchMilestone          http.HandlerFunc
	handleFetchNoAckMilestone     http.HandlerFunc
	handleFetchLastNoAckMilestone http.HandlerFunc
}

func (h *HttpHandlerFake) GetCheckpointHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.handleFetchCheckpoint.ServeHTTP(w, r)
	}
}

func (h *HttpHandlerFake) GetMilestoneHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.handleFetchMilestone.ServeHTTP(w, r)
	}
}

func (h *HttpHandlerFake) GetNoAckMilestoneHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.handleFetchNoAckMilestone.ServeHTTP(w, r)
	}
}

func (h *HttpHandlerFake) GetLastNoAckMilestoneHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.handleFetchLastNoAckMilestone.ServeHTTP(w, r)
	}
}

func CreateMockHeimdallServer(wg *sync.WaitGroup, port int, listener net.Listener, handler *HttpHandlerFake) (*http.Server, error) {
	// Create a new server mux
	mux := http.NewServeMux()

	// Create a route for fetching latest checkpoint
	mux.HandleFunc("/checkpoints/latest", func(w http.ResponseWriter, r *http.Request) {
		handler.GetCheckpointHandler()(w, r)
	})

	// Create a route for fetching milestone
	mux.HandleFunc("/milestone/latest", func(w http.ResponseWriter, r *http.Request) {
		handler.GetMilestoneHandler()(w, r)
	})

	// Create a route for fetching milestone
	mux.HandleFunc("/milestone/noAck/{id}", func(w http.ResponseWriter, r *http.Request) {
		handler.GetNoAckMilestoneHandler()(w, r)
	})

	// Create a route for fetching milestone
	mux.HandleFunc("/milestone/lastNoAck", func(w http.ResponseWriter, r *http.Request) {
		handler.GetLastNoAckMilestoneHandler()(w, r)
	})

	// Add other routes as per requirement

	// Create the server with given port and mux
	srv := &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", port),
		Handler: mux,
	}

	// Close the listener using the port and immediately consume it below
	err := listener.Close()
	if err != nil {
		return nil, err
	}

	go func() {
		defer wg.Done()

		// always returns error. ErrServerClosed on graceful close
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("error in server.ListenAndServe(): %v", err)
		}
	}()

	return srv, nil
}

// TestFetchCheckpointFromMockHeimdall tests the heimdall client side logic
// to fetch checkpoints (latest for the scope of test) from a mock heimdall server.
// It can be used for debugging purpose (like response fields, marshalling/unmarshalling, etc).
func TestFetchCheckpointFromMockHeimdall(t *testing.T) {
	t.Parallel()

	// Create a wait group for sending across the mock server
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Initialize the fake handler and add a fake checkpoint handler function
	handler := &HttpHandlerFake{}
	handler.handleFetchCheckpoint = func(w http.ResponseWriter, _ *http.Request) {
		err := json.NewEncoder(w).Encode(checkpoint.CheckpointResponse{
			Height: "0",
			Result: checkpoint.Checkpoint{
				Proposer:   common.Address{},
				StartBlock: big.NewInt(0),
				EndBlock:   big.NewInt(512),
				RootHash:   common.Hash{},
				BorChainID: "15001",
				Timestamp:  0,
			},
		})

		if err != nil {
			w.WriteHeader(500) // Return 500 Internal Server Error.
		}
	}

	// Fetch available port
	port, listener, err := network.FindAvailablePort()
	require.NoError(t, err, "expect no error in finding available port")

	// Create mock heimdall server and pass handler instance for setting up the routes
	srv, err := CreateMockHeimdallServer(wg, port, listener, handler)
	require.NoError(t, err, "expect no error in starting mock heimdall server")

	// Create a new heimdall client and use same port for connection
	client := NewHeimdallClient(fmt.Sprintf("http://localhost:%d", port))
	_, err = client.FetchCheckpoint(context.Background(), -1)
	require.NoError(t, err, "expect no error in fetching checkpoint")

	// Shutdown the server
	err = srv.Shutdown(context.TODO())
	require.NoError(t, err, "expect no error in shutting down mock heimdall server")

	// Wait for `wg.Done()` to be called in the mock server's routine.
	wg.Wait()
}

// TestFetchMilestoneFromMockHeimdall tests the heimdall client side logic
// to fetch milestone from a mock heimdall server.
// It can be used for debugging purpose (like response fields, marshalling/unmarshalling, etc).
func TestFetchMilestoneFromMockHeimdall(t *testing.T) {
	t.Parallel()

	// Create a wait group for sending across the mock server
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Initialize the fake handler and add a fake milestone handler function
	handler := &HttpHandlerFake{}
	handler.handleFetchMilestone = func(w http.ResponseWriter, _ *http.Request) {
		err := json.NewEncoder(w).Encode(milestone.MilestoneResponse{
			Height: "0",
			Result: milestone.Milestone{
				Proposer:   common.Address{},
				StartBlock: big.NewInt(0),
				EndBlock:   big.NewInt(512),
				Hash:       common.Hash{},
				BorChainID: "15001",
				Timestamp:  0,
			},
		})

		if err != nil {
			w.WriteHeader(500) // Return 500 Internal Server Error.
		}
	}

	// Fetch available port
	port, listener, err := network.FindAvailablePort()
	require.NoError(t, err, "expect no error in finding available port")

	// Create mock heimdall server and pass handler instance for setting up the routes
	srv, err := CreateMockHeimdallServer(wg, port, listener, handler)
	require.NoError(t, err, "expect no error in starting mock heimdall server")

	// Create a new heimdall client and use same port for connection
	client := NewHeimdallClient(fmt.Sprintf("http://localhost:%d", port))
	_, err = client.FetchMilestone(context.Background())
	require.NoError(t, err, "expect no error in fetching milestone")

	// Shutdown the server
	err = srv.Shutdown(context.TODO())
	require.NoError(t, err, "expect no error in shutting down mock heimdall server")

	// Wait for `wg.Done()` to be called in the mock server's routine.
	wg.Wait()
}

// TestFetchShutdown tests the heimdall client side logic for context timeout and
// interrupt handling while fetching data from a mock heimdall server.
func TestFetchShutdown(t *testing.T) {
	t.Parallel()

	// Create a wait group for sending across the mock server
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Initialize the fake handler and add a fake checkpoint handler function
	handler := &HttpHandlerFake{}

	// Case1 - Testing context timeout: Create delay in serving requests for simulating timeout. Add delay slightly
	// greater than `retryDelay`. This should cause the request to timeout and trigger shutdown
	// due to `ctx.Done()`. Expect context timeout error.
	handler.handleFetchCheckpoint = func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)

		err := json.NewEncoder(w).Encode(checkpoint.CheckpointResponse{
			Height: "0",
			Result: checkpoint.Checkpoint{
				Proposer:   common.Address{},
				StartBlock: big.NewInt(0),
				EndBlock:   big.NewInt(512),
				RootHash:   common.Hash{},
				BorChainID: "15001",
				Timestamp:  0,
			},
		})

		if err != nil {
			w.WriteHeader(500) // Return 500 Internal Server Error.
		}
	}

	// Fetch available port
	port, listener, err := network.FindAvailablePort()
	require.NoError(t, err, "expect no error in finding available port")

	// Create mock heimdall server and pass handler instance for setting up the routes
	srv, err := CreateMockHeimdallServer(wg, port, listener, handler)
	require.NoError(t, err, "expect no error in starting mock heimdall server")

	// Create a new heimdall client and use same port for connection
	client := NewHeimdallClient(fmt.Sprintf("http://localhost:%d", port))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)

	// Expect this to fail due to timeout
	_, err = client.FetchCheckpoint(ctx, -1)
	require.Equal(t, "context deadline exceeded", err.Error(), "expect the function error to be a context deadline exceeded error")
	require.Equal(t, "context deadline exceeded", ctx.Err().Error(), "expect the ctx error to be a context deadline exceeded error")

	cancel()

	// Case2 - Testing context cancellation. Pass a context with timeout to the request and
	// cancel it before timeout. This should cause the request to timeout and trigger shutdown
	// due to `ctx.Done()`. Expect context cancellation error.
	handler.handleFetchCheckpoint = func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(500) // Return 500 Internal Server Error.
	}

	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond) // Use some high value for timeout

	// Cancel the context after a delay until we make request
	go func(cancel context.CancelFunc) {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}(cancel)

	// Expect this to fail due to cancellation
	_, err = client.FetchCheckpoint(ctx, -1)
	require.Equal(t, "context canceled", err.Error(), "expect the function error to be a context cancelled error")
	require.Equal(t, "context canceled", ctx.Err().Error(), "expect the ctx error to be a context cancelled error")

	// Case3 - Testing interrupt: Closing the `closeCh` in heimdall client simulating interrupt. This
	// should cause the request to fail and throw an error due to `<-closeCh` in fetchWithRetry.
	// Expect shutdown detected error.
	handler.handleFetchCheckpoint = func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500) // Return 500 Internal Server Error.
	}

	// Close the channel after a delay until we make request
	go func() {
		time.Sleep(1 * time.Second)
		close(client.closeCh)
	}()

	// Expect this to fail due to shutdown
	_, err = client.FetchCheckpoint(context.Background(), -1)
	require.Equal(t, ErrShutdownDetected.Error(), err.Error(), "expect the function error to be a shutdown detected error")

	// Shutdown the server
	err = srv.Shutdown(context.TODO())
	require.NoError(t, err, "expect no error in shutting down mock heimdall server")

	// Wait for `wg.Done()` to be called in the mock server's routine.
	wg.Wait()
}

// TestContext includes bunch of simple tests to verify the working of timeout
// based context and cancellation.
func TestContext(t *testing.T) {
	t.Parallel()

	ctx, cancel1 := context.WithTimeout(context.Background(), 1*time.Second)

	// Case1: Done is not yet closed, so Err returns nil.
	require.NoError(t, ctx.Err(), "expect nil error")

	wg := &sync.WaitGroup{}

	// Case2: Check if timeout is being handled
	wg.Add(1)

	go func(ctx context.Context, wg *sync.WaitGroup) {
		defer wg.Done()
		select {
		case <-ctx.Done():
			// Expect context deadline exceeded error
			require.Equal(t, "context deadline exceeded", ctx.Err().Error(), "expect the ctx error to be a context deadline exceeded error")
		case <-time.After(2 * time.Second):
			// Case for safely exiting the tests
			return
		}
	}(ctx, wg)

	// Case3: Check normal case
	ctx, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)

	wg.Add(1)

	errCh := make(chan error, 1)

	go func(ctx context.Context, wg *sync.WaitGroup) {
		defer wg.Done()
		select {
		case <-ctx.Done():
			// Expect this to never occur, throw explicit error
			errCh <- errors.New("unexpected call to `ctx.Done()`")
		case <-time.After(2 * time.Second):
			// Case for safely exiting the tests
			errCh <- nil
			return
		}
	}(ctx, wg)

	if err := <-errCh; err != nil {
		t.Fatalf("err: %v", err)
	}

	// Case4: Check if cancellation is being handled
	ctx, cancel3 := context.WithTimeout(context.Background(), 1*time.Second)

	wg.Add(1)

	go func(cancel context.CancelFunc) {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}(cancel3)

	go func(ctx context.Context, wg *sync.WaitGroup) {
		defer wg.Done()
		select {
		case <-ctx.Done():
			// Expect context canceled error
			require.Equal(t, "context canceled", ctx.Err().Error(), "expect the ctx error to be a context canceled error")
		case <-time.After(2 * time.Second):
			// Case for safely exiting the tests
			return
		}
	}(ctx, wg)

	// Wait for all tests to pass
	wg.Wait()

	// Cancel all remaining contexts
	cancel1()
	cancel2()
}

func TestSpanURL(t *testing.T) {
	t.Parallel()

	url, err := spanURL("http://bor0", 1)
	if err != nil {
		t.Fatal("got an error", err)
	}

	const expected = "http://bor0/bor/span/1"

	if url.String() != expected {
		t.Fatalf("expected URL %q, got %q", expected, url.String())
	}
}

func TestStateSyncURL(t *testing.T) {
	t.Parallel()

	url, err := stateSyncURL("http://bor0", 10, 100)
	if err != nil {
		t.Fatal("got an error", err)
	}

	const expected = "http://bor0/clerk/event-record/list?from-id=10&to-time=100&limit=50"

	if url.String() != expected {
		t.Fatalf("expected URL %q, got %q", expected, url.String())
	}
}
