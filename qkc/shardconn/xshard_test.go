// Copyright 2026-2027, QuarkChain.

package shardconn

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/qkc/wire"
)

type testHandler struct {
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
			pool, err := NewXshardPool(0, []uint32{0, 1}, tc.limit, &testHandler{}, nil)
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

func TestRejectUnconfiguredChain(t *testing.T) {
	handler := &testHandler{requests: make(chan *wire.AddXshardTxListRequest, 1)}
	pool, err := NewXshardPool(0, []uint32{0, 1}, 0, handler, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.DialToChain(context.Background(), wire.ChainInfo{ChainID: 2}); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected unconfigured chain error, got %v", err)
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
		t.Fatal("unconfigured inbound chain received a PONG")
	}
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("inbound handler did not finish")
	}
	if len(pool.Lookup(2)) != 0 || pool.knownRemote(2) {
		t.Fatal("unconfigured chain was registered")
	}
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	if len(pool.connections) != 0 {
		t.Fatal("rejected connection remains tracked")
	}
}

func (h *testHandler) AddXshardTxList(req *wire.AddXshardTxListRequest) (*wire.AddXshardTxListResponse, error) {
	h.requests <- req
	return &wire.AddXshardTxListResponse{}, nil
}

func TestTCPChainHandshakeAndRequest(t *testing.T) {
	handler := &testHandler{requests: make(chan *wire.AddXshardTxListRequest, 1)}
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
	if err := client.DialToChain(ctx, wire.ChainInfo{Host: []byte("127.0.0.1"), Port: uint16(addr.Port), ChainID: 0}); err != nil {
		t.Fatal(err)
	}
	<-done
	if len(server.Lookup(1)) != 1 || len(client.Lookup(0)) != 1 {
		t.Fatal("connections were not indexed by chain ID")
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

func TestDialRejectsWrongChainID(t *testing.T) {
	handler := &testHandler{requests: make(chan *wire.AddXshardTxListRequest, 1)}
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
	if err := client.DialToChain(ctx, wire.ChainInfo{Host: []byte("127.0.0.1"), Port: uint16(addr.Port), ChainID: 3}); err == nil {
		t.Fatal("expected chain ID mismatch")
	}
	if len(client.Lookup(3)) != 0 {
		t.Fatal("mismatched connection was indexed")
	}
}
