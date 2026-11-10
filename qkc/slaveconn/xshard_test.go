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
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/qkc/comm"
	"github.com/ethereum/go-ethereum/qkc/conn"
	"github.com/ethereum/go-ethereum/qkc/wire"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

type testXshardHandler struct {
	requests chan xshardRequest
}

type xshardRequest struct {
	blockHash common.Hash
	deposits  []*types.DepositTx
}

func noopShutdown() {}

func TestXshardConnManagerPayloadLimit(t *testing.T) {
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
			manager, err := NewXshardConnManager(big.NewInt(1), []*big.Int{big.NewInt(1), big.NewInt(2)}, tc.limit, &testXshardHandler{}, noopShutdown, nil)
			if manager != nil {
				defer manager.Close()
			}
			if (err != nil) != tc.wantErr {
				t.Fatalf("NewXshardConnManager error = %v, want error: %v", err, tc.wantErr)
			}
			if tc.wantErr && manager != nil {
				t.Fatal("invalid payload limit returned a manager")
			}
		})
	}
}

func TestRejectUnconfiguredSlave(t *testing.T) {
	handler := &testXshardHandler{requests: make(chan xshardRequest, 1)}
	manager, err := NewXshardConnManager(big.NewInt(1), []*big.Int{big.NewInt(1), big.NewInt(2)}, 0, handler, noopShutdown, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	if err := manager.DialToSlave(context.Background(), wire.SlaveInfo{ChainID: [wire.UInt256Length]byte{3}}); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected unconfigured slave error, got %v", err)
	}

	local, remote := net.Pipe()
	done := make(chan struct{})
	go func() { defer close(done); manager.HandleInbound(local) }()
	peer, err := newXshardConn(remote, 0, big.NewInt(3), big.NewInt(1), true, handler, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	peer.Start()
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
	if manager.Lookup(big.NewInt(3)) != nil || manager.knownSlave(big.NewInt(3)) {
		t.Fatal("unconfigured slave was registered")
	}
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	if len(manager.connections) != 0 {
		t.Fatal("rejected connection remains tracked")
	}
}

func (h *testXshardHandler) AddXshardTxList(blockHash common.Hash, deposits []*types.DepositTx) error {
	h.requests <- xshardRequest{blockHash: blockHash, deposits: deposits}
	return nil
}

type xshardHandlerFunc func(common.Hash, []*types.DepositTx) error

func (h xshardHandlerFunc) AddXshardTxList(blockHash common.Hash, deposits []*types.DepositTx) error {
	return h(blockHash, deposits)
}

// newTestXshardPeer completes an inbound handshake through a manager so requests
// exercise the same handler registration as real TCP connections.
func newTestXshardPeer(t *testing.T, handler XshardRequestHandler) *XshardConn {
	t.Helper()
	manager, err := NewXshardConnManager(big.NewInt(1), []*big.Int{big.NewInt(1), big.NewInt(2)}, 0, handler, noopShutdown, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(manager.Close)
	local, remote := net.Pipe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		manager.HandleInbound(local)
	}()
	peer, err := newXshardConn(remote, 0, big.NewInt(2), big.NewInt(1), true, handler, nil)
	if err != nil {
		remote.Close()
		t.Fatal(err)
	}
	t.Cleanup(peer.Close)
	peer.Start()
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
	peer := newTestXshardPeer(t, xshardHandlerFunc(func(hash common.Hash, _ []*types.DepositTx) error {
		if hash == (common.Hash{1}) {
			return &comm.RemoteError{Code: errorCode}
		}
		return nil
	}))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := peer.SendAddXshardTxList(ctx, common.Hash{1}, nil)
	var remoteErr *comm.RemoteError
	if !errors.As(err, &remoteErr) || remoteErr.Code != errorCode {
		t.Fatalf("expected RemoteError with code %d, got %T: %v", errorCode, err, err)
	}
	if err := peer.SendAddXshardTxList(ctx, common.Hash{}, nil); err != nil {
		t.Fatalf("request after remote execution failure: %v", err)
	}
}

func TestXshardConnRejectsUnsupportedDepositVersion(t *testing.T) {
	called := false
	chainID := big.NewInt(1)
	x := &XshardConn{localChainID: chainID, handler: xshardHandlerFunc(func(common.Hash, []*types.DepositTx) error {
		called = true
		return nil
	})}
	encoded, err := rlp.EncodeToBytes(&types.DepositTx{Version: types.DepositTxVersion + 1, Value: uint256.NewInt(1), GasPrice: uint256.NewInt(1)})
	if err != nil {
		t.Fatal(err)
	}
	sszChainID, err := wire.ChainIDToSSZ(chainID)
	if err != nil {
		t.Fatal(err)
	}
	response, err := x.handleAddXshardTxList(&wire.AddXshardTxListRequest{ChainID: sszChainID, DepositTxList: []wire.Data{encoded}})
	if err == nil || !strings.Contains(err.Error(), "unsupported deposit version 1") {
		t.Fatalf("expected unsupported deposit version error, got %v", err)
	}
	if response != nil || called {
		t.Fatalf("unsupported deposit was acknowledged or passed to handler: response %v, called %t", response, called)
	}
}

func TestXshardConnHandlerErrorClosesConnection(t *testing.T) {
	for _, tc := range []struct {
		name string
	}{
		{"handler error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			peer := newTestXshardPeer(t, xshardHandlerFunc(func(common.Hash, []*types.DepositTx) error {
				return errors.New("handler unavailable")
			}))
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := peer.SendAddXshardTxList(ctx, common.Hash{}, nil)
			if !errors.Is(err, conn.ErrConnectionClosed) {
				t.Fatalf("expected connection closure after handler error, got %v", err)
			}
			if err := peer.SendAddXshardTxList(ctx, common.Hash{}, nil); !errors.Is(err, conn.ErrConnectionClosed) {
				t.Fatalf("request after handler failure: expected connection closed, got %v", err)
			}
		})
	}
}

func TestXshardConnContextErrorsKeepConnection(t *testing.T) {
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	peer := newTestXshardPeer(t, xshardHandlerFunc(func(hash common.Hash, _ []*types.DepositTx) error {
		if hash[0] == 2 {
			entered <- struct{}{}
			<-release
		}
		return nil
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
				result <- peer.SendAddXshardTxList(ctx, common.Hash{2}, nil)
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
			var remoteErr *comm.RemoteError
			if !errors.Is(err, tc.wantErr) || errors.As(err, &remoteErr) {
				t.Fatalf("expected context error %v, got %T: %v", tc.wantErr, err, err)
			}
			ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := peer.SendAddXshardTxList(ctx, common.Hash{}, nil); err != nil {
				t.Fatalf("request after context error: %v", err)
			}
		})
	}
}

func TestTCPXshardHandshakeAndRequest(t *testing.T) {
	handler := &testXshardHandler{requests: make(chan xshardRequest, 1)}
	serverID := new(big.Int).Lsh(big.NewInt(1), 80)
	clientID := new(big.Int).Add(serverID, big.NewInt(1))
	configuredIDs := []*big.Int{serverID, clientID}
	serverWireID, err := wire.ChainIDToSSZ(serverID)
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewXshardConnManager(serverID, configuredIDs, 0, handler, noopShutdown, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	client, err := NewXshardConnManager(clientID, configuredIDs, 0, handler, noopShutdown, nil)
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
	serverID.SetInt64(0)
	serverID = wire.ChainIDFromSSZ(serverWireID)
	if err := client.DialToSlave(ctx, wire.SlaveInfo{Host: []byte("127.0.0.1"), Port: uint16(addr.Port), ChainID: serverWireID}); err != nil {
		t.Fatal(err)
	}
	<-done
	if server.Lookup(new(big.Int).Set(clientID)) == nil || client.Lookup(new(big.Int).Set(serverID)) == nil {
		t.Fatal("slave connections were not indexed by chain ID")
	}
	remoteID := client.Lookup(serverID).RemoteChainID()
	remoteID.SetInt64(0)
	if client.Lookup(serverID).RemoteChainID().Cmp(serverID) != 0 {
		t.Fatal("remote chain ID changed through its accessor")
	}
	deposit := &types.DepositTx{Value: uint256.NewInt(42), GasPrice: uint256.NewInt(3), SourceBlockHash: common.Hash{0xaa}}
	blockHash := common.Hash{0xbb}
	if err := client.Lookup(serverID).SendAddXshardTxList(ctx, blockHash, []*types.DepositTx{deposit}); err != nil {
		t.Fatal(err)
	}
	select {
	case req := <-handler.requests:
		if req.blockHash != blockHash || len(req.deposits) != 1 || req.deposits[0].SourceBlockHash != deposit.SourceBlockHash || req.deposits[0].Value.Cmp(deposit.Value) != 0 {
			t.Fatalf("unexpected handler request: %+v", req)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestDialToSlaveRejectsWrongChainID(t *testing.T) {
	handler := &testXshardHandler{requests: make(chan xshardRequest, 1)}
	server, err := NewXshardConnManager(big.NewInt(3), []*big.Int{big.NewInt(2), big.NewInt(3), big.NewInt(4)}, 0, handler, noopShutdown, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	client, err := NewXshardConnManager(big.NewInt(2), []*big.Int{big.NewInt(2), big.NewInt(3), big.NewInt(4)}, 0, handler, noopShutdown, nil)
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
	if err := client.DialToSlave(ctx, wire.SlaveInfo{Host: []byte("127.0.0.1"), Port: uint16(addr.Port), ChainID: [wire.UInt256Length]byte{4}}); err == nil {
		t.Fatal("expected slave chain ID mismatch")
	}
	if client.Lookup(big.NewInt(4)) != nil {
		t.Fatal("mismatched connection was indexed")
	}
}

func TestXshardConnManagerRejectsDuplicateConnection(t *testing.T) {
	handler := &testXshardHandler{}
	manager, err := NewXshardConnManager(big.NewInt(1), []*big.Int{big.NewInt(1), big.NewInt(2)}, 0, handler, noopShutdown, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	firstLocal, firstRemote := net.Pipe()
	defer firstRemote.Close()
	first, err := newXshardConn(firstLocal, 0, big.NewInt(1), big.NewInt(2), true, handler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !manager.trackConnection(first) {
		t.Fatal("could not track first connection")
	}
	if err := manager.registerConnection(first); err != nil {
		t.Fatal(err)
	}

	duplicateLocal, duplicateRemote := net.Pipe()
	defer duplicateRemote.Close()
	duplicate, err := newXshardConn(duplicateLocal, 0, big.NewInt(1), big.NewInt(2), true, handler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !manager.trackConnection(duplicate) {
		t.Fatal("could not track duplicate connection")
	}
	if err := manager.registerConnection(duplicate); err == nil || !strings.Contains(err.Error(), "already connected") {
		t.Fatalf("expected duplicate connection error, got %v", err)
	}
	if manager.Lookup(big.NewInt(2)) != first || !duplicate.IsClosed() {
		t.Fatal("duplicate connection replaced the registered connection or remained open")
	}
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	if len(manager.connections) != 1 {
		t.Fatalf("tracked connections = %d, want 1", len(manager.connections))
	}
}

func TestXshardConnManagerShutsDownOnIndexedConnectionClose(t *testing.T) {
	shutdown := make(chan struct{}, 1)
	manager, err := NewXshardConnManager(
		big.NewInt(1), []*big.Int{big.NewInt(1), big.NewInt(2)}, 0,
		&testXshardHandler{}, func() { shutdown <- struct{}{} }, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	local, remote := net.Pipe()
	defer remote.Close()
	indexed, err := newXshardConn(local, 0, big.NewInt(1), big.NewInt(2), true, &testXshardHandler{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !manager.trackConnection(indexed) {
		t.Fatal("could not track connection")
	}
	if err := manager.registerConnection(indexed); err != nil {
		t.Fatal(err)
	}

	indexed.Close()
	select {
	case <-shutdown:
	case <-time.After(5 * time.Second):
		t.Fatal("indexed connection close did not shut down slave")
	}
}

func TestXshardConnManagerCloseDoesNotShutDownSlave(t *testing.T) {
	shutdown := make(chan struct{}, 1)
	manager, err := NewXshardConnManager(
		big.NewInt(1), []*big.Int{big.NewInt(1), big.NewInt(2)}, 0,
		&testXshardHandler{}, func() { shutdown <- struct{}{} }, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	local, remote := net.Pipe()
	defer remote.Close()
	indexed, err := newXshardConn(local, 0, big.NewInt(1), big.NewInt(2), true, &testXshardHandler{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !manager.trackConnection(indexed) {
		t.Fatal("could not track connection")
	}
	if err := manager.registerConnection(indexed); err != nil {
		t.Fatal(err)
	}

	manager.Close()
	select {
	case <-shutdown:
		t.Fatal("normal manager close shut down slave")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestXshardConnManagerRejectsInvalidChainIDs(t *testing.T) {
	for _, id := range []*big.Int{nil, new(big.Int), big.NewInt(-1), new(big.Int).Lsh(big.NewInt(1), 256)} {
		if manager, err := NewXshardConnManager(id, []*big.Int{big.NewInt(1)}, 0, &testXshardHandler{}, noopShutdown, nil); err == nil {
			manager.Close()
			t.Fatalf("accepted invalid local chain ID %v", id)
		}
		if manager, err := NewXshardConnManager(big.NewInt(1), []*big.Int{big.NewInt(1), id}, 0, &testXshardHandler{}, noopShutdown, nil); err == nil {
			manager.Close()
			t.Fatalf("accepted invalid configured chain ID %v", id)
		}
	}
	if manager, err := NewXshardConnManager(big.NewInt(1), []*big.Int{big.NewInt(2)}, 0, &testXshardHandler{}, noopShutdown, nil); err == nil {
		manager.Close()
		t.Fatal("accepted local chain ID missing from configuration")
	}
}
