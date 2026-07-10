// Copyright 2026-2027, QuarkChain.

package slave

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/qkc/cluster/wire"
	"github.com/ethereum/go-ethereum/qkc/serialize"
)

// writeRawFrame writes a raw frame directly to the underlying TCP connection,
// bypassing the connection's frame writer. Used to craft malformed/invalid frames
// for protocol-validation tests.
func writeRawFrame(t *testing.T, conn net.Conn, frame *wire.Frame) {
	t.Helper()
	if err := wire.WriteFrameNoMeta(conn, frame); err != nil {
		t.Fatalf("write raw frame: %v", err)
	}
}

// newTestConnPair creates a pair of XshardConns connected over a local TCP
// socket. The caller is responsible for calling cleanup.
func newTestConnPair(t *testing.T) (client, server *XshardConn, cleanup func()) {
	t.Helper()
	return newTestConnPairWithIdentity(t, []byte("client-slave"), []uint32{0x00010001}, []byte("server-slave"), []uint32{0x00030004})
}

func newTestConnPairWithIdentity(t *testing.T, clientID []byte, clientShards []uint32, serverID []byte, serverShards []uint32) (client, server *XshardConn, cleanup func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	var serverConn net.Conn
	var acceptErr error
	accepted := make(chan struct{})
	go func() {
		defer close(accepted)
		serverConn, acceptErr = ln.Accept()
		ln.Close()
	}()

	clientConn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	<-accepted
	if acceptErr != nil {
		t.Fatalf("accept: %v", acceptErr)
	}

	logger := log.New()
	client = NewXshardConnFromConn(clientConn, 0, clientID, clientShards, logger) // 0 = no limit (matches Python)
	server = NewXshardConnFromConn(serverConn, 0, serverID, serverShards, logger)
	cleanup = func() {
		client.Close()
		server.Close()
	}
	return
}

// TestXshardConn_DefaultPingHandler verifies that PING is handled internally
// even when the server does not register a PING handler. The server still
// records peer identity and returns a PONG with its own identity.
func TestXshardConn_DefaultPingHandler(t *testing.T) {
	clientID := []byte("client-slave")
	clientShards := []uint32{0x00010001}
	serverID := []byte("server-slave")
	serverShards := []uint32{0x00030004}

	client, server, cleanup := newTestConnPairWithIdentity(t, clientID, clientShards, serverID, serverShards)
	defer cleanup()

	// Server does NOT register any handler; PING should be handled internally.
	server.Start()
	client.Start()

	pingPayload, err := serialize.SerializeToBytes(&wire.PingRequest{
		ID:              clientID,
		FullShardIDList: clientShards,
		RootTip:         nil,
	})
	if err != nil {
		t.Fatalf("serialize ping: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.SendRPC(ctx, byte(wire.ClusterOpPing), pingPayload)
	if err != nil {
		t.Fatalf("send ping rpc: %v", err)
	}
	if resp.Opcode != byte(wire.ClusterOpPong) {
		t.Fatalf("expected opcode 0x%x, got 0x%x", wire.ClusterOpPong, resp.Opcode)
	}

	var pong wire.PongResponse
	if err := serialize.Deserialize(serialize.NewByteBuffer(resp.Payload), &pong); err != nil {
		t.Fatalf("deserialize pong: %v", err)
	}
	if string(pong.ID) != string(serverID) {
		t.Fatalf("pong id mismatch: got %s, expected %s", pong.ID, serverID)
	}
	if len(pong.FullShardIDList) != len(serverShards) {
		t.Fatalf("pong shard list mismatch: got %v", pong.FullShardIDList)
	}

	if !server.WaitUntilPingReceived() {
		t.Fatal("server did not receive ping")
	}
	if string(server.RemoteID()) != string(clientID) {
		t.Fatalf("server remote id mismatch: got %s", server.RemoteID())
	}
}

func TestXshardConn_RPCRoundTrip(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()

	clientID := []byte("client-slave")
	clientShards := []uint32{0x00010001, 0x00010002}
	serverID := []byte("server-slave")
	serverShards := []uint32{0x00030004}

	server.RegisterHandlers(map[byte]TypedHandler{
		byte(wire.ClusterOpPing): func(req any) (any, error) {
			_ = req.(*wire.PingRequest)
			return &wire.PongResponse{
				ID:              serverID,
				FullShardIDList: serverShards,
			}, nil
		},
	})
	server.Start()
	client.Start()

	pingPayload, err := serialize.SerializeToBytes(&wire.PingRequest{
		ID:              clientID,
		FullShardIDList: clientShards,
		RootTip:         nil, // OK for SlaveConnection (master doesn't use it)
	})
	if err != nil {
		t.Fatalf("serialize ping: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.SendRPC(ctx, byte(wire.ClusterOpPing), pingPayload)
	if err != nil {
		t.Fatalf("send ping rpc: %v", err)
	}
	if resp.Opcode != byte(wire.ClusterOpPong) {
		t.Fatalf("expected opcode 0x%x, got 0x%x", wire.ClusterOpPong, resp.Opcode)
	}

	var pong wire.PongResponse
	if err := serialize.Deserialize(serialize.NewByteBuffer(resp.Payload), &pong); err != nil {
		t.Fatalf("deserialize pong: %v", err)
	}
	if string(pong.ID) != string(serverID) {
		t.Fatalf("pong id mismatch: got %s", pong.ID)
	}

	if !server.WaitUntilPingReceived() {
		t.Fatal("server did not receive ping")
	}
	if string(server.RemoteID()) != string(clientID) {
		t.Fatalf("server remote id mismatch: got %s", server.RemoteID())
	}
	if len(server.RemoteFullShardIDList()) != len(clientShards) {
		t.Fatalf("server remote shard list mismatch: got %v", server.RemoteFullShardIDList())
	}
}

// TestXshardConn_RejectEmptyShardList verifies that empty shard list causes
// connection close (Python's close_with_error behavior). The peer ID is still
// recorded before closing, matching Python's handle_ping.
func TestXshardConn_RejectEmptyShardList(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()

	server.RegisterHandlers(map[byte]TypedHandler{
		byte(wire.ClusterOpPing): func(req any) (any, error) {
			// This handler won't be called because wrapper rejects empty shard list first.
			t.Fatal("user handler should not be called for empty shard list")
			return nil, nil
		},
	})
	server.Start()
	client.Start()

	pingPayload, err := serialize.SerializeToBytes(&wire.PingRequest{
		ID:              []byte("bad-slave"),
		FullShardIDList: []uint32{}, // empty list
		RootTip:         nil,
	})
	if err != nil {
		t.Fatalf("serialize ping: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Python: empty shard list causes close_with_error (connection close, no response).
	_, err = client.SendRPC(ctx, byte(wire.ClusterOpPing), pingPayload)
	if err == nil {
		t.Fatal("expected error due to connection close, got nil")
	}
	// The error should be connection closed (readLoop returns after handler error).
	if err != ErrConnectionClosed {
		t.Logf("got error: %v (expected ErrConnectionClosed or timeout)", err)
	}

	// Python: id is recorded BEFORE close_with_error is called.
	// The wrapper records the id first, then checks shard list.
	if string(server.RemoteID()) != "bad-slave" {
		t.Fatalf("expected remote ID 'bad-slave', got %v", server.RemoteID())
	}
}

// TestXshardConn_UnsupportedOpcodeClosesConnection verifies that unsupported
// opcode causes connection close (Python's close_with_error behavior).
func TestXshardConn_UnsupportedOpcodeClosesConnection(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()

	server.Start()
	client.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Send a request for an opcode that has no handler.
	_, err := client.SendRPC(ctx, byte(wire.ClusterOpAddRootBlockRequest), []byte("payload"))
	if err == nil {
		t.Fatal("expected error due to connection close, got nil")
	}
	// Connection should be closed by server due to unsupported opcode.
	if err != ErrConnectionClosed {
		t.Logf("got error: %v (expected ErrConnectionClosed or timeout)", err)
	}
}

// TestXshardConn_HandlerErrorClosesConnection verifies that handler error
// causes connection close (Python's close_with_error behavior).
func TestXshardConn_HandlerErrorClosesConnection(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()

	server.RegisterHandlers(map[byte]TypedHandler{
		byte(wire.ClusterOpAddRootBlockRequest): func(req any) (any, error) {
			_ = req
			return nil, fmt.Errorf("intentional error") //nolint:govet // test error
		},
	})
	server.Start()
	client.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.SendRPC(ctx, byte(wire.ClusterOpAddRootBlockRequest), []byte("payload"))
	if err == nil {
		t.Fatal("expected error due to connection close, got nil")
	}
}

// TestXshardConn_HandlerPanicClosesConnection verifies that handler panic
// causes connection close (Python's close_with_error behavior).
func TestXshardConn_HandlerPanicClosesConnection(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()

	server.RegisterHandlers(map[byte]TypedHandler{
		byte(wire.ClusterOpAddRootBlockRequest): func(req any) (any, error) {
			_ = req
			panic("intentional panic")
		},
	})
	server.Start()
	client.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.SendRPC(ctx, byte(wire.ClusterOpAddRootBlockRequest), []byte("payload"))
	if err == nil {
		t.Fatal("expected error due to connection close, got nil")
	}
}

// TestXshardConn_CloseWakesPendingRPC verifies that Close wakes all pending RPCs.
// Uses a sync channel instead of time.Sleep for reliable testing.
func TestXshardConn_CloseWakesPendingRPC(t *testing.T) {
	client, _, cleanup := newTestConnPair(t)
	defer cleanup()

	// Server intentionally left unstarted so it never replies.
	client.Start()

	var wg sync.WaitGroup
	wg.Add(1)
	errChan := make(chan error, 1)
	go func() {
		wg.Done() // Signal that goroutine is ready
		_, err := client.SendRPC(context.Background(), byte(wire.ClusterOpPing), []byte("ping"))
		errChan <- err
	}()

	wg.Wait() // Wait for goroutine to start (reliable synchronization)
	client.Close()

	select {
	case err := <-errChan:
		if err != ErrConnectionClosed {
			t.Fatalf("expected ErrConnectionClosed, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pending RPC was not woken by Close")
	}
}

// TestXshardConn_SendXshardTxList verifies RPC mode for AddXshardTxListRequest.
// The handler must return a proper response (AddXshardTxListResponse).
func TestXshardConn_SendXshardTxList(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()

	server.RegisterHandlers(map[byte]TypedHandler{
		byte(wire.ClusterOpAddXshardTxListRequest): func(req any) (any, error) {
			_ = req.(*wire.AddXshardTxListRequest)
			// Return success response (Python: AddXshardTxListResponse(error_code=0))
			return &wire.AddXshardTxListResponse{ErrorCode: 0}, nil
		},
	})
	server.Start()
	client.Start()

	txList := wire.RawBytes([]byte("tx-list"))
	req := &wire.AddXshardTxListRequest{
		Branch:         0x00010001,
		MinorBlockHash: [32]byte{1, 2, 3},
		TxList:         &txList,
	}
	payload, err := serialize.SerializeToBytes(req)
	if err != nil {
		t.Fatalf("serialize request: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.SendXshardTxList(ctx, payload)
	if err != nil {
		t.Fatalf("send xshard tx list: %v", err)
	}
	if resp.Opcode != byte(wire.ClusterOpAddXshardTxListResponse) {
		t.Fatalf("unexpected response opcode 0x%x", resp.Opcode)
	}

	var xshardResp wire.AddXshardTxListResponse
	if err := serialize.Deserialize(serialize.NewByteBuffer(resp.Payload), &xshardResp); err != nil {
		t.Fatalf("deserialize response: %v", err)
	}
	if xshardResp.ErrorCode != 0 {
		t.Fatalf("expected error_code 0, got %d", xshardResp.ErrorCode)
	}
}

// TestXshardConn_SendBatchXshardTxList verifies RPC mode for BatchAddXshardTxListRequest.
func TestXshardConn_SendBatchXshardTxList(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()

	server.RegisterHandlers(map[byte]TypedHandler{
		byte(wire.ClusterOpBatchAddXshardTxListRequest): func(req any) (any, error) {
			_ = req.(*wire.BatchAddXshardTxListRequest)
			return &wire.BatchAddXshardTxListResponse{ErrorCode: 0}, nil
		},
	})
	server.Start()
	client.Start()

	txList := wire.RawBytes([]byte("tx1"))
	req := &wire.BatchAddXshardTxListRequest{
		AddXshardTxListRequestList: []wire.AddXshardTxListRequest{
			{Branch: 0x00010001, MinorBlockHash: [32]byte{1, 2, 3}, TxList: &txList},
		},
	}
	payload, err := serialize.SerializeToBytes(req)
	if err != nil {
		t.Fatalf("serialize request: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.SendBatchXshardTxList(ctx, payload)
	if err != nil {
		t.Fatalf("send batch xshard tx list: %v", err)
	}
	if resp.Opcode != byte(wire.ClusterOpBatchAddXshardTxListResponse) {
		t.Fatalf("unexpected response opcode 0x%x", resp.Opcode)
	}

	var batchResp wire.BatchAddXshardTxListResponse
	if err := serialize.Deserialize(serialize.NewByteBuffer(resp.Payload), &batchResp); err != nil {
		t.Fatalf("deserialize response: %v", err)
	}
	if batchResp.ErrorCode != 0 {
		t.Fatalf("expected error_code 0, got %d", batchResp.ErrorCode)
	}
}

func TestXshardPool_AddGetRemove(t *testing.T) {
	pool := NewXshardPool(log.New())
	defer pool.Close()

	// Use stub connections that are never started.
	_, conn1, cleanup1 := newTestConnPair(t)
	defer cleanup1()
	_, conn2, cleanup2 := newTestConnPair(t)
	defer cleanup2()

	pool.Add(0x00010001, conn1)
	pool.Add(0x00010001, conn2)
	pool.Add(0x00020001, conn1)

	if got := pool.OutboundSize(); got != 3 {
		t.Fatalf("expected pool outbound size 3, got %d", got)
	}

	conns := pool.Get(0x00010001)
	if len(conns) != 2 {
		t.Fatalf("expected 2 conns for shard 0x00010001, got %d", len(conns))
	}

	pool.Remove(0x00010001, conn1)
	if got := pool.OutboundSize(); got != 2 {
		t.Fatalf("expected pool outbound size 2 after remove, got %d", got)
	}
	conns = pool.Get(0x00010001)
	if len(conns) != 1 || conns[0] != conn2 {
		t.Fatalf("expected only conn2 for shard 0x00010001")
	}

	targets := pool.Targets()
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}
}

func TestXshardPool_RemoveTargetClosesConnections(t *testing.T) {
	pool := NewXshardPool(log.New())
	defer pool.Close()

	_, conn, cleanup := newTestConnPair(t)
	defer cleanup()

	conn.Start()
	pool.Add(0x00010001, conn)
	pool.RemoveTarget(0x00010001)

	if pool.OutboundSize() != 0 {
		t.Fatalf("expected pool outbound size 0, got %d", pool.OutboundSize())
	}

	// A closed connection rejects further RPCs.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := conn.SendRPC(ctx, byte(wire.ClusterOpPing), []byte("ping"))
	if err != ErrConnectionClosed {
		t.Fatalf("expected ErrConnectionClosed, got %v", err)
	}
}

func TestXshardPool_TrackInboundClose(t *testing.T) {
	pool := NewXshardPool(log.New())

	_, conn, cleanup := newTestConnPair(t)
	defer cleanup()

	conn.Start()
	pool.TrackInbound(conn)
	pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := conn.SendRPC(ctx, byte(wire.ClusterOpPing), []byte("ping"))
	if err != ErrConnectionClosed {
		t.Fatalf("expected ErrConnectionClosed after pool close, got %v", err)
	}
}

func TestXshardPool_SendXshardTxNoConnection(t *testing.T) {
	pool := NewXshardPool(log.New())
	defer pool.Close()

	ctx := context.Background()
	_, err := pool.SendXshardTx(ctx, 0x00010001, []byte("tx"))
	if err == nil {
		t.Fatal("expected error when no connection exists")
	}
}

func TestXshardPool_ClosedPoolRejectsAdd(t *testing.T) {
	pool := NewXshardPool(log.New())
	pool.Close()

	_, conn, cleanup := newTestConnPair(t)
	defer cleanup()

	conn.Start()
	pool.Add(0x00010001, conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := conn.SendRPC(ctx, byte(wire.ClusterOpPing), []byte("ping"))
	if err != ErrConnectionClosed {
		t.Fatalf("expected ErrConnectionClosed, got %v", err)
	}
}

// TestXshardConn_RPCIDMonotonic verifies RPC ID monotonic validation.
// Sending a duplicate RPC ID causes the server to close the connection.
func TestXshardConn_RPCIDMonotonic(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()

	server.RegisterHandlers(map[byte]TypedHandler{
		byte(wire.ClusterOpPing): func(req any) (any, error) {
			_ = req.(*wire.PingRequest)
			return &wire.PongResponse{
				ID:              []byte("server"),
				FullShardIDList: []uint32{0x00030004},
			}, nil
		},
	})
	server.Start()
	client.Start()

	pingPayload, _ := serialize.SerializeToBytes(&wire.PingRequest{
		ID:              []byte("client"),
		FullShardIDList: []uint32{0x00010001},
	})

	// Manually send two PING frames with the same RPC ID (=1).
	writeRawFrame(t, client.conn, &wire.Frame{
		Opcode:  byte(wire.ClusterOpPing),
		RPCID:   1,
		Payload: pingPayload,
	})
	writeRawFrame(t, client.conn, &wire.Frame{
		Opcode:  byte(wire.ClusterOpPing),
		RPCID:   1, // duplicate rpc_id: should trigger close
		Payload: pingPayload,
	})

	// Wait for server to close the connection.
	select {
	case <-server.WaitUntilClosed():
	case <-time.After(2 * time.Second):
		t.Fatal("server did not close connection after duplicate rpc_id")
	}

	if !server.IsClosed() {
		t.Fatal("server should be closed")
	}
}

// TestXshardConn_RPCIDDecreasing verifies that a decreasing RPC ID closes the connection.
func TestXshardConn_RPCIDDecreasing(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()

	server.RegisterHandlers(map[byte]TypedHandler{
		byte(wire.ClusterOpPing): func(req any) (any, error) {
			_ = req.(*wire.PingRequest)
			return &wire.PongResponse{
				ID:              []byte("server"),
				FullShardIDList: []uint32{0x00030004},
			}, nil
		},
	})
	server.Start()
	client.Start()

	pingPayload, _ := serialize.SerializeToBytes(&wire.PingRequest{
		ID:              []byte("client"),
		FullShardIDList: []uint32{0x00010001},
	})

	// Send rpc_id=2 then rpc_id=1 (decreasing).
	writeRawFrame(t, client.conn, &wire.Frame{
		Opcode:  byte(wire.ClusterOpPing),
		RPCID:   2,
		Payload: pingPayload,
	})
	writeRawFrame(t, client.conn, &wire.Frame{
		Opcode:  byte(wire.ClusterOpPing),
		RPCID:   1, // decreasing rpc_id: should trigger close
		Payload: pingPayload,
	})

	select {
	case <-server.WaitUntilClosed():
	case <-time.After(2 * time.Second):
		t.Fatal("server did not close connection after decreasing rpc_id")
	}
}

// TestXshardConn_MultipleRPCs verifies multiple sequential RPCs work correctly.
func TestXshardConn_MultipleRPCs(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()

	callCount := 0
	server.RegisterHandlers(map[byte]TypedHandler{
		byte(wire.ClusterOpPing): func(req any) (any, error) {
			_ = req.(*wire.PingRequest)
			callCount++
			return &wire.PongResponse{
				ID:              []byte("server"),
				FullShardIDList: []uint32{0x00010001},
			}, nil
		},
	})
	server.Start()
	client.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pingPayload, _ := serialize.SerializeToBytes(&wire.PingRequest{
		ID:              []byte("client"),
		FullShardIDList: []uint32{0x00010001},
	})

	// Send multiple RPCs in sequence.
	for i := 0; i < 5; i++ {
		_, err := client.SendRPC(ctx, byte(wire.ClusterOpPing), pingPayload)
		if err != nil {
			t.Fatalf("rpc %d failed: %v", i+1, err)
		}
	}

	if callCount != 5 {
		t.Fatalf("expected 5 handler calls, got %d", callCount)
	}
}

// TestXshardConn_RecordPingOnlyOnce verifies that recordPing only updates
// on first PING (matches Python's handle_ping behavior).
func TestXshardConn_RecordPingOnlyOnce(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()

	server.RegisterHandlers(map[byte]TypedHandler{
		byte(wire.ClusterOpPing): func(req any) (any, error) {
			_ = req.(*wire.PingRequest)
			return &wire.PongResponse{
				ID:              []byte("server"),
				FullShardIDList: []uint32{0x00010001},
			}, nil
		},
	})
	server.Start()
	client.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First PING with one shard list.
	ping1, _ := serialize.SerializeToBytes(&wire.PingRequest{
		ID:              []byte("client1"),
		FullShardIDList: []uint32{0x00010001, 0x00010002},
	})
	_, err := client.SendRPC(ctx, byte(wire.ClusterOpPing), ping1)
	if err != nil {
		t.Fatalf("first ping failed: %v", err)
	}

	firstID := server.RemoteID()
	firstShards := server.RemoteFullShardIDList()

	// Second PING with different shard list (should not overwrite).
	ping2, _ := serialize.SerializeToBytes(&wire.PingRequest{
		ID:              []byte("client2"),
		FullShardIDList: []uint32{0x00030004},
	})
	_, err = client.SendRPC(ctx, byte(wire.ClusterOpPing), ping2)
	if err != nil {
		t.Fatalf("second ping failed: %v", err)
	}

	// RemoteID and RemoteFullShardIDList should NOT have changed.
	if string(server.RemoteID()) != string(firstID) {
		t.Fatalf("remote ID changed: got %s, expected %s", server.RemoteID(), firstID)
	}
	if len(server.RemoteFullShardIDList()) != len(firstShards) {
		t.Fatalf("remote shard list changed: got %v, expected %v", server.RemoteFullShardIDList(), firstShards)
	}
}
