// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package slaveconn

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/qkc/conn"
	"github.com/ethereum/go-ethereum/qkc/wire"
)

type testXshardHandler struct {
	requests chan *wire.AddXshardTxListRequest
}

func TestXshardPoolPayloadLimit(t *testing.T) {
	for _, tc := range []struct {
		name    string
		limit   uint32
		wantErr bool
	}{
		{"default", 0, false},
		{"maximum", wire.MaxFramePayloadLen, false},
		{"above maximum", wire.MaxFramePayloadLen + 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := NewXshardPool(0, []uint32{0, 1}, tc.limit, &testXshardHandler{}, nil)
			if pool != nil {
				defer pool.Close()
			}
			if (err != nil) != tc.wantErr {
				t.Fatalf("NewXshardPool error = %v, want error: %v", err, tc.wantErr)
			}
			if tc.wantErr && pool != nil {
				t.Fatal("invalid payload limit returned a pool")
			}
		})
	}
}

func TestRejectUnconfiguredSlave(t *testing.T) {
	handler := &testXshardHandler{requests: make(chan *wire.AddXshardTxListRequest, 1)}
	pool, err := NewXshardPool(0, []uint32{0, 1}, 0, handler, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.DialToSlave(context.Background(), wire.ChainInfo{ChainID: 2}); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected unconfigured slave error, got %v", err)
	}

	local, remote := net.Pipe()
	done := make(chan struct{})
	go func() { defer close(done); pool.HandleInbound(local) }()
	peer, err := newXshardConn(remote, 0, 2, 0, true, handler, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer peer.base.Close()
	peer.base.Start()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := peer.sendPing(ctx); err == nil {
		t.Fatal("unconfigured inbound slave received a PONG")
	}
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("inbound handler did not finish")
	}
	if len(pool.Lookup(2)) != 0 || pool.knownSlave(2) {
		t.Fatal("unconfigured slave was registered")
	}
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	if len(pool.connections) != 0 {
		t.Fatal("rejected connection remains tracked")
	}
}

func (h *testXshardHandler) AddXshardTxList(req *wire.AddXshardTxListRequest) (*wire.AddXshardTxListResponse, error) {
	h.requests <- req
	return &wire.AddXshardTxListResponse{}, nil
}

type xshardHandlerFunc func(*wire.AddXshardTxListRequest) (*wire.AddXshardTxListResponse, error)

func (h xshardHandlerFunc) AddXshardTxList(req *wire.AddXshardTxListRequest) (*wire.AddXshardTxListResponse, error) {
	return h(req)
}

// newTestXshardPeer completes an inbound handshake through a pool so requests
// exercise the same handler registration as real TCP connections.
func newTestXshardPeer(t *testing.T, handler XshardHandler) *XshardConn {
	t.Helper()
	pool, err := NewXshardPool(0, []uint32{0, 1}, 0, handler, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	local, remote := net.Pipe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		pool.HandleInbound(local)
	}()
	peer, err := newXshardConn(remote, 0, 1, 0, true, handler, nil)
	if err != nil {
		remote.Close()
		t.Fatal(err)
	}
	t.Cleanup(peer.base.Close)
	peer.base.Start()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := peer.sendPing(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	return peer
}

func TestXshardConnRemoteErrorKeepsConnection(t *testing.T) {
	const errorCode = 404
	peer := newTestXshardPeer(t, xshardHandlerFunc(func(req *wire.AddXshardTxListRequest) (*wire.AddXshardTxListResponse, error) {
		if req.ChainID != 0 {
			return &wire.AddXshardTxListResponse{ErrorCode: errorCode}, nil
		}
		return &wire.AddXshardTxListResponse{}, nil
	}))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := peer.SendAddXshardTxList(ctx, &wire.AddXshardTxListRequest{ChainID: 2})
	var remoteErr *wire.RemoteError
	if !errors.As(err, &remoteErr) || remoteErr.Code != errorCode {
		t.Fatalf("expected RemoteError with code %d, got %T: %v", errorCode, err, err)
	}
	if err := peer.SendAddXshardTxList(ctx, &wire.AddXshardTxListRequest{ChainID: 0}); err != nil {
		t.Fatalf("request after remote execution failure: %v", err)
	}
}

func TestXshardConnHandlerErrorClosesConnection(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response *wire.AddXshardTxListResponse
	}{
		{"nil response", nil},
		{"success response with error", &wire.AddXshardTxListResponse{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			peer := newTestXshardPeer(t, xshardHandlerFunc(func(*wire.AddXshardTxListRequest) (*wire.AddXshardTxListResponse, error) {
				return tc.response, errors.New("handler unavailable")
			}))
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := peer.SendAddXshardTxList(ctx, &wire.AddXshardTxListRequest{ChainID: 0})
			if !errors.Is(err, conn.ErrConnectionClosed) {
				t.Fatalf("expected connection closure after handler error, got %v", err)
			}
			if err := peer.SendAddXshardTxList(ctx, &wire.AddXshardTxListRequest{ChainID: 0}); !errors.Is(err, conn.ErrConnectionClosed) {
				t.Fatalf("request after handler failure: expected connection closed, got %v", err)
			}
		})
	}
}

func TestXshardConnContextErrorsKeepConnection(t *testing.T) {
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	peer := newTestXshardPeer(t, xshardHandlerFunc(func(req *wire.AddXshardTxListRequest) (*wire.AddXshardTxListResponse, error) {
		if req.ChainID != 0 {
			entered <- struct{}{}
			<-release
		}
		return &wire.AddXshardTxListResponse{}, nil
	}))
	t.Cleanup(func() { close(release) })
	for _, tc := range []struct {
		name    string
		timeout time.Duration
		wantErr error
	}{
		{"canceled", 5 * time.Second, context.Canceled},
		{"deadline", 100 * time.Millisecond, context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()
			result := make(chan error, 1)
			go func() {
				result <- peer.SendAddXshardTxList(ctx, &wire.AddXshardTxListRequest{ChainID: 1})
			}()
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("business handler was not called")
			}
			if tc.wantErr == context.Canceled {
				cancel()
			}
			err := <-result
			var remoteErr *wire.RemoteError
			if !errors.Is(err, tc.wantErr) || errors.As(err, &remoteErr) {
				t.Fatalf("expected context error %v, got %T: %v", tc.wantErr, err, err)
			}
			ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := peer.SendAddXshardTxList(ctx, &wire.AddXshardTxListRequest{ChainID: 0}); err != nil {
				t.Fatalf("request after context error: %v", err)
			}
		})
	}
}

func TestTCPXshardHandshakeAndRequest(t *testing.T) {
	handler := &testXshardHandler{requests: make(chan *wire.AddXshardTxListRequest, 1)}
	server, err := NewXshardPool(0, []uint32{0, 1}, 0, handler, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	client, err := NewXshardPool(1, []uint32{0, 1}, 0, handler, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		c, err := listener.Accept()
		if err == nil {
			server.HandleInbound(c)
		}
	}()
	addr := listener.Addr().(*net.TCPAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.DialToSlave(ctx, wire.ChainInfo{Host: []byte("127.0.0.1"), Port: uint16(addr.Port), ChainID: 0}); err != nil {
		t.Fatal(err)
	}
	<-done
	if len(server.Lookup(1)) != 1 || len(client.Lookup(0)) != 1 {
		t.Fatal("slave connections were not indexed by chain ID")
	}
	if err := client.Lookup(0)[0].SendAddXshardTxList(ctx, &wire.AddXshardTxListRequest{ChainID: 0}); err != nil {
		t.Fatal(err)
	}
	select {
	case req := <-handler.requests:
		if req.ChainID != 0 {
			t.Fatalf("chain ID = %d", req.ChainID)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestDialToSlaveRejectsWrongChainID(t *testing.T) {
	handler := &testXshardHandler{requests: make(chan *wire.AddXshardTxListRequest, 1)}
	server, err := NewXshardPool(2, []uint32{1, 2, 3}, 0, handler, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	client, err := NewXshardPool(1, []uint32{1, 2, 3}, 0, handler, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		c, err := listener.Accept()
		if err == nil {
			server.HandleInbound(c)
		}
	}()
	addr := listener.Addr().(*net.TCPAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.DialToSlave(ctx, wire.ChainInfo{Host: []byte("127.0.0.1"), Port: uint16(addr.Port), ChainID: 3}); err == nil {
		t.Fatal("expected slave chain ID mismatch")
	}
	if len(client.Lookup(3)) != 0 {
		t.Fatal("mismatched connection was indexed")
	}
}
