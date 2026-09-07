// Copyright 2026-2027, QuarkChain.

package slave

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/qkc/cluster/wire"
	"github.com/ethereum/go-ethereum/qkc/serialize"
	"github.com/ethereum/go-ethereum/qkc/types"
)

// ── pool test helpers (white-box, same package) ──────────────────────────────

// hasSlaveID reports whether the pool tracks the given peer identity.
func (p *XshardPool) hasSlaveID(id []byte) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.slaveIDs[string(id)]
	return ok
}

// connectionsSize returns the number of tracked connections (including conns
// still in the PING handshake).
func (p *XshardPool) connectionsSize() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.connections)
}

// ── business handler test hook ────────────────────────────────────────────────

// testXshardHandler is a test hook standing in for the business handler: it
// acknowledges every request so tests without real business logic do not fail.
type testXshardHandler struct{}

func (testXshardHandler) AddXshardTxList(*wire.AddXshardTxListRequest) (*wire.AddXshardTxListResponse, error) {
	return &wire.AddXshardTxListResponse{}, nil
}

func (testXshardHandler) BatchAddXshardTxList(*wire.BatchAddXshardTxListRequest) (*wire.BatchAddXshardTxListResponse, error) {
	return &wire.BatchAddXshardTxListResponse{}, nil
}

// mustNewXshardPool creates a pool with the test hook (maxPayloadSize 0) and
// the given cluster-wide configured shard set.
func mustNewXshardPool(t *testing.T, selfID []byte, shards, clusterShardIDs []uint32) *XshardPool {
	t.Helper()
	pool, err := NewXshardPool(selfID, shards, clusterShardIDs, 0, testXshardHandler{}, log.New())
	if err != nil {
		t.Fatalf("new xshard pool: %v", err)
	}
	return pool
}

// ── TCP test pair helpers ─────────────────────────────────────────────────────

// newTestConnPair creates a pair of XshardConn connected over a local TCP
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
	client, err = newXshardConn(clientConn, 0, clientID, clientShards, nil, nil, testXshardHandler{}, logger) // 0 = no limit (matches Python)
	if err != nil {
		t.Fatalf("new client conn: %v", err)
	}
	server, err = newXshardConn(serverConn, 0, serverID, serverShards, nil, nil, testXshardHandler{}, logger)
	if err != nil {
		t.Fatalf("new server conn: %v", err)
	}
	cleanup = func() {
		client.Close()
		server.Close()
	}
	return
}

// newRawConnPair establishes a TCP connection and returns both ends. It is used
// to drive the pool's HandleInbound with a raw accepted net.Conn.
func newRawConnPair(t *testing.T) (client, server net.Conn) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	accepted := make(chan net.Conn, 1)
	acceptErr := make(chan error, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			acceptErr <- err
			return
		}
		accepted <- c
	}()

	client, err = net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	select {
	case server = <-accepted:
	case err := <-acceptErr:
		t.Fatalf("accept: %v", err)
	}
	return client, server
}

// establishInbound drives one inbound connection through HandleInbound and the
// PING handshake: the client side acts as the remote slave sending PING, the
// server side is handed to the pool. It returns once the connection is indexed.
func establishInbound(t *testing.T, pool *XshardPool, remoteID []byte, remoteShards []uint32) {
	t.Helper()
	clientConn, serverConn := newRawConnPair(t)

	done := make(chan struct{})
	go func() {
		pool.HandleInbound(serverConn)
		close(done)
	}()

	client, err := newXshardConn(clientConn, 0, remoteID, remoteShards, nil, nil, testXshardHandler{}, log.New())
	if err != nil {
		clientConn.Close()
		t.Fatalf("new inbound client conn: %v", err)
	}
	client.Start()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, _, err := client.sendPing(ctx); err != nil {
		client.Close()
		t.Fatalf("inbound ping: %v", err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("HandleInbound did not return after PING")
	}
	client.Close()
}

// ── XshardConn layer tests ───────────────────────────────────────────────────

func TestXshardConn_RPCRoundTrip(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()

	clientID := []byte("client-slave")
	clientShards := []uint32{0x00010001, 0x00010002}
	serverID := []byte("server-slave")

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

	pong, ok := resp.(*wire.PongResponse)
	if !ok {
		t.Fatalf("expected *wire.PongResponse, got %T", resp)
	}
	if string(pong.ID) != string(serverID) {
		t.Fatalf("pong id mismatch: got %s, expected %s", pong.ID, serverID)
	}

	if !server.waitUntilPingReceived() {
		t.Fatal("server did not receive ping")
	}
	if string(server.RemoteID()) != string(clientID) {
		t.Fatalf("server remote id mismatch: got %s", server.RemoteID())
	}
	if len(server.RemoteFullShardIDList()) != len(clientShards) {
		t.Fatalf("server remote shard list mismatch: got %v", server.RemoteFullShardIDList())
	}
}

// TestXshardConn_XshardTxListServedByHandler verifies AddXshardTxList is
// served by the injected business handler and keeps the connection open.
func TestXshardConn_XshardTxListServedByHandler(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()
	server.Start()
	client.Start()

	txList := types.CrossShardTransactionList{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.SendAddXshardTxList(ctx, &wire.AddXshardTxListRequest{
		Branch: 1,
		TxList: &txList,
	}); err != nil {
		t.Fatalf("sendAddXshardTxList: %v", err)
	}
	if client.IsClosed() || server.IsClosed() {
		t.Fatal("connection should stay open after AddXshardTxList")
	}
}

// TestXshardConn_BatchAddXshardTxListServedByHandler verifies
// BatchAddXshardTxList is served by the injected business handler and keeps
// the connection open.
func TestXshardConn_BatchAddXshardTxListServedByHandler(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()
	server.Start()
	client.Start()

	txList := types.CrossShardTransactionList{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.SendBatchAddXshardTxList(ctx, &wire.BatchAddXshardTxListRequest{
		AddXshardTxListRequestList: []wire.AddXshardTxListRequest{
			{Branch: 1, TxList: &txList},
			{Branch: 2, TxList: &txList},
		},
	}); err != nil {
		t.Fatalf("sendBatchAddXshardTxList: %v", err)
	}
	if client.IsClosed() || server.IsClosed() {
		t.Fatal("connection should stay open after BatchAddXshardTxList")
	}
}

// TestXshardConn_SendPingRejectsWrongResponseOpcode verifies a wrong-opcode
// PONG is rejected by sendPing's opcode check but does not close the connection.
func TestXshardConn_SendPingRejectsWrongResponseOpcode(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client, err := newXshardConn(clientConn, 0, []byte("client"), []uint32{1}, nil, nil, testXshardHandler{}, log.New())
	if err != nil {
		t.Fatalf("new conn: %v", err)
	}
	client.Start()
	// Reply with an AddXshardTxListResponse (wrong opcode for PING).
	peerDone := rawXshardPeer(t, serverConn, 0)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _, err = client.sendPing(ctx)
	if err == nil {
		t.Fatal("expected wrong PING response opcode error")
	}
	if err := <-peerDone; err != nil {
		t.Fatalf("raw peer failed: %v", err)
	}
	if client.IsClosed() {
		t.Fatal("wrong PING response opcode should not close the connection")
	}
}

func TestXshardConn_WaitUntilPingReceivedReturnsAfterClose(t *testing.T) {
	_, server, cleanup := newTestConnPair(t)
	defer cleanup()

	result := make(chan bool, 1)
	go func() {
		result <- server.waitUntilPingReceived()
	}()
	server.Close()
	select {
	case got := <-result:
		if got {
			t.Fatal("expected false after close before PING")
		}
	case <-time.After(time.Second):
		t.Fatal("waitUntilPingReceived did not return after close")
	}
}

func TestXshardConn_RejectEmptyShardList(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()

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

	_, err = client.SendRPC(ctx, byte(wire.ClusterOpPing), pingPayload)
	if err == nil {
		t.Fatal("expected error due to connection close, got nil")
	}
	// An illegal empty first PING publishes nothing (Python parity for the
	// observable protocol): the handshake never completes and no identity is
	// recorded, so a getter can never race a partial initialization.
	if got := server.RemoteID(); len(got) != 0 {
		t.Fatalf("expected no recorded identity for empty shard list, got %q", got)
	}
	select {
	case <-server.pingReceived:
		t.Fatal("empty shard list must not complete the handshake")
	default:
	}
}

// TestXshardConn_RecordPingOnlyOnce verifies the first PING records identity
// and later PINGs do not overwrite it (matches Python's handle_ping).
func TestXshardConn_RecordPingOnlyOnce(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()

	server.Start()
	client.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ping1, _ := serialize.SerializeToBytes(&wire.PingRequest{
		ID:              []byte("client1"),
		FullShardIDList: []uint32{0x00010001, 0x00010002},
	})
	if _, err := client.SendRPC(ctx, byte(wire.ClusterOpPing), ping1); err != nil {
		t.Fatalf("first ping failed: %v", err)
	}

	firstID := server.RemoteID()
	firstShards := server.RemoteFullShardIDList()

	ping2, _ := serialize.SerializeToBytes(&wire.PingRequest{
		ID:              []byte("client2"),
		FullShardIDList: []uint32{0x00030004},
	})
	if _, err := client.SendRPC(ctx, byte(wire.ClusterOpPing), ping2); err != nil {
		t.Fatalf("second ping failed: %v", err)
	}

	if string(server.RemoteID()) != string(firstID) {
		t.Fatalf("remote ID changed: got %s, expected %s", server.RemoteID(), firstID)
	}
	if len(server.RemoteFullShardIDList()) != len(firstShards) {
		t.Fatalf("remote shard list changed: got %v, expected %v", server.RemoteFullShardIDList(), firstShards)
	}
}

// TestXshardConn_HandlePingBarrierUnblockedByClose verifies the inbound
// registration barrier's fallback: if the barrier is never released (pool
// failure path), closing the connection unblocks a pending handlePing with an
// error instead of leaking the goroutine — no successful PONG is produced.
func TestXshardConn_HandlePingBarrierUnblockedByClose(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()

	server.Start()
	client.Start()

	// Arm the barrier as HandleInbound would, but never close it: the only
	// way out of the barrier must then be connection close.
	server.inboundRegistered = make(chan struct{})

	result := make(chan error, 1)
	go func() {
		_, err := server.handlePing(&wire.PingRequest{
			ID:              []byte("client-slave"),
			FullShardIDList: []uint32{0x00010001},
		})
		result <- err
	}()

	server.Close()

	select {
	case err := <-result:
		if err == nil {
			t.Fatal("expected error when barrier is not released before close")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handlePing leaked: barrier not unblocked by connection close")
	}
}

// TestXshardConn_HandlePingAfterCloseReturnsImmediately verifies the other
// ordering of the same fallback: when the connection is closed BEFORE handlePing
// reaches the (unreleased) barrier, the call must return an error promptly
// instead of blocking forever on the nil-progress channel.
func TestXshardConn_HandlePingAfterCloseReturnsImmediately(t *testing.T) {
	_, server, cleanup := newTestConnPair(t)
	defer cleanup()

	server.Start()

	// Arm the barrier but never release it, then close the connection first.
	server.inboundRegistered = make(chan struct{})
	server.Close()

	result := make(chan error, 1)
	go func() {
		_, err := server.handlePing(&wire.PingRequest{
			ID:              []byte("client-slave"),
			FullShardIDList: []uint32{0x00010001},
		})
		result <- err
	}()

	select {
	case err := <-result:
		if err == nil {
			t.Fatal("expected error for handlePing on a closed connection")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handlePing blocked forever on closed connection")
	}
}

// TestXshardConn_AcceptEmptyPingID verifies a PING with an empty slave ID is
// accepted (Python only rejects an empty shard list).
func TestXshardConn_AcceptEmptyPingID(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()

	client.Start()
	server.Start()

	pingPayload, err := serialize.SerializeToBytes(&wire.PingRequest{
		ID:              []byte{},
		FullShardIDList: []uint32{0x00010001},
		RootTip:         nil,
	})
	if err != nil {
		t.Fatalf("serialize ping: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if _, err = client.SendRPC(ctx, byte(wire.ClusterOpPing), pingPayload); err != nil {
		t.Fatalf("expected PING with empty ID to be accepted, got %v", err)
	}
	if len(server.RemoteID()) != 0 {
		t.Fatalf("expected empty remote ID, got %s", server.RemoteID())
	}
	if server.IsClosed() {
		t.Fatal("server connection should remain open after empty-ID PING")
	}
}

// TestXshardConn_ConcurrentPingPublishesOnce drives several PINGs through
// BaseConn's per-request goroutine dispatch at once, verifying the handshake
// publishes peer metadata exactly once (via pingOnce) and that the result is
// immutable afterward. Run under -race this guards the lock-free peer metadata
// read model against the concurrent handler execution.
func TestXshardConn_ConcurrentPingPublishesOnce(t *testing.T) {
	client, server, cleanup := newTestConnPair(t)
	defer cleanup()

	server.Start()
	client.Start()

	const count = 8
	ids := make([][]byte, count)
	shards := make([][]uint32, count)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		i := i
		ids[i] = []byte(fmt.Sprintf("peer-%d", i))
		shards[i] = []uint32{uint32(0x00010000 + i)}
		wg.Add(1)
		go func() {
			defer wg.Done()
			payload, err := serialize.SerializeToBytes(&wire.PingRequest{
				ID:              ids[i],
				FullShardIDList: shards[i],
			})
			if err != nil {
				t.Errorf("serialize ping %d: %v", i, err)
				return
			}
			if _, err := client.SendRPC(ctx, byte(wire.ClusterOpPing), payload); err != nil {
				t.Errorf("ping %d: %v", i, err)
			}
		}()
	}
	wg.Wait()

	// Exactly one of the concurrent PINGs records its identity.
	found := 0
	for i := 0; i < count; i++ {
		if bytes.Equal(server.RemoteID(), ids[i]) {
			found++
		}
	}
	if found != 1 {
		t.Fatalf("expected exactly one recorded identity, got %d", found)
	}

	// After publication the metadata is immutable across reads.
	firstID := server.RemoteID()
	firstShards := server.RemoteFullShardIDList()
	if len(firstShards) != 1 {
		t.Fatalf("expected a single shard, got %v", firstShards)
	}
	for i := 0; i < 100; i++ {
		if !bytes.Equal(server.RemoteID(), firstID) {
			t.Fatal("remote ID changed after publication")
		}
		if len(server.RemoteFullShardIDList()) != 1 || server.RemoteFullShardIDList()[0] != firstShards[0] {
			t.Fatal("remote shard list changed after publication")
		}
	}
}

// ── XshardPool indexing tests ─────────────────────────────────────────────────

// TestXshardPool_ClosedConnectionStaysIndexed verifies Python parity: a CLOSED
// connection is never evicted from the routing index or slave ID registry.
func TestXshardPool_ClosedConnectionStaysIndexed(t *testing.T) {
	rs := startRemoteSlave(t, []byte("server-slave"), []uint32{0x00030004, 0x00030005})
	defer rs.close()

	pool := mustNewXshardPool(t, []byte("local-slave"), []uint32{0x00030004}, []uint32{0x00030004, 0x00030005})
	defer pool.Close()

	if err := pool.DialToSlave(context.Background(), rs.slaveInfo([]byte("server-slave"), []uint32{0x00030004, 0x00030005})); err != nil {
		t.Fatalf("dial: %v", err)
	}

	// Grab the indexed outbound connection and close it directly.
	var target *XshardConn
	for _, shardID := range []uint32{0x00030004, 0x00030005} {
		conns := pool.Lookup(shardID)
		if len(conns) != 1 {
			t.Fatalf("expected 1 conn for shard 0x%x, got %d", shardID, len(conns))
		}
		target = conns[0]
	}
	target.Close()

	for _, shardID := range []uint32{0x00030004, 0x00030005} {
		if conns := pool.Lookup(shardID); len(conns) != 1 || conns[0] != target {
			t.Fatalf("route 0x%x no longer contains the closed connection: %v", shardID, conns)
		}
	}
	if !pool.hasSlaveID([]byte("server-slave")) {
		t.Fatal("slave ID was removed after connection close")
	}
}

// ── inbound tests ─────────────────────────────────────────────────────────────

// TestXshardPool_RouteFilteredByClusterShardSet verifies the Python parity fix:
// route keys are restricted to the cluster-wide configured shard set, so a
// peer advertising an out-of-config id cannot create a route for it. The
// peer's slave id is still tracked regardless of the filtered shards.
func TestXshardPool_RouteFilteredByClusterShardSet(t *testing.T) {
	pool, err := NewXshardPool([]byte("local-slave"), []uint32{0x00030004}, []uint32{0x00010001}, 0, testXshardHandler{}, log.New())
	if err != nil {
		t.Fatalf("new xshard pool: %v", err)
	}
	defer pool.Close()

	configuredShard := uint32(0x00010001)
	rogueShard := uint32(0x00BAD00F) // advertised but not configured
	establishInbound(t, pool, []byte("remote-slave"), []uint32{configuredShard, rogueShard})

	if conns := pool.Lookup(configuredShard); len(conns) != 1 {
		t.Fatalf("expected 1 conn for configured shard 0x%x, got %d", configuredShard, len(conns))
	}
	if conns := pool.Lookup(rogueShard); len(conns) != 0 {
		t.Fatalf("rogue shard 0x%x must not be routed, got %d conns", rogueShard, len(conns))
	}
	if !pool.hasSlaveID([]byte("remote-slave")) {
		t.Fatal("slave ID must still be tracked even when some shards are filtered")
	}
}

// TestXshardPool_SequentialDialLeavesOneLiveRoute reproduces the Python master's
// sequential orchestration of mutual xshard routing: first S0 is told to dial S1
// (producing an S0 outbound and an S1 inbound that register each other over a
// real TCP connection); only afterwards S1 is told to dial S0. Because S1
// already knows S0 from the inbound handshake, DialToSlave's pre-check skips the
// second dial, so each pool keeps exactly one live route and no duplicate TCP
// connection is opened. Duplicate inbound connections are not a required
// behavior; the topology to preserve is that skip. Unlike the old helper-based
// tests (which closed the client before asserting), this keeps both pool-owned
// ends of the retained connection open and round-trips a real request over it to
// prove it is live.
func TestXshardPool_SequentialDialLeavesOneLiveRoute(t *testing.T) {
	s0ID, s1ID := []byte("s0"), []byte("s1")
	s0Shards := []uint32{1, 2}
	s1Shards := []uint32{3, 4}
	clusterShards := []uint32{1, 2, 3, 4}

	ln0, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen s0: %v", err)
	}
	defer ln0.Close()
	ln1, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen s1: %v", err)
	}
	defer ln1.Close()

	pool0 := mustNewXshardPool(t, s0ID, s0Shards, clusterShards)
	defer pool0.Close()
	pool1 := mustNewXshardPool(t, s1ID, s1Shards, clusterShards)
	defer pool1.Close()

	// Accept loops standing in for each slave's inbound path.
	for ln, pool := range map[net.Listener]*XshardPool{ln0: pool0, ln1: pool1} {
		go func(ln net.Listener, pool *XshardPool) {
			for {
				c, err := ln.Accept()
				if err != nil {
					return
				}
				pool.HandleInbound(c)
			}
		}(ln, pool)
	}

	a0 := ln0.Addr().(*net.TCPAddr)
	a1 := ln1.Addr().(*net.TCPAddr)
	info0 := wire.SlaveInfo{ID: s0ID, Host: []byte(a0.IP.String()), Port: uint16(a0.Port), FullShardIDList: s0Shards}
	info1 := wire.SlaveInfo{ID: s1ID, Host: []byte(a1.IP.String()), Port: uint16(a1.Port), FullShardIDList: s1Shards}

	// Step 1: master tells S0 to dial S1 (S0 outbound / S1 inbound). When this
	// returns, both registration sides are already complete: S1's inbound
	// registration happens-before its PONG is written (inboundRegistered
	// barrier), and S0 registers synchronously before DialToSlave returns. No
	// polling is needed — this verifies the production guarantee.
	if err := pool0.DialToSlave(context.Background(), info1); err != nil {
		t.Fatalf("s0 dial s1: %v", err)
	}

	// Step 2: master tells S1 to dial S0; the pre-check skips it because S0 is
	// already known from the inbound handshake, so no second TCP connection is made.
	if err := pool1.DialToSlave(context.Background(), info0); err != nil {
		t.Fatalf("s1 dial s0 should be skipped, got error: %v", err)
	}

	// Each peer shard keeps exactly one live route in each pool.
	for _, shard := range s1Shards {
		conns := pool0.Lookup(shard)
		if len(conns) != 1 {
			t.Fatalf("s0 route 0x%x: expected exactly 1 connection, got %d", shard, len(conns))
		}
		if conns[0].IsClosed() {
			t.Fatalf("s0 route 0x%x: retained connection is closed", shard)
		}
	}
	for _, shard := range s0Shards {
		conns := pool1.Lookup(shard)
		if len(conns) != 1 {
			t.Fatalf("s1 route 0x%x: expected exactly 1 connection, got %d", shard, len(conns))
		}
		if conns[0].IsClosed() {
			t.Fatalf("s1 route 0x%x: retained connection is closed", shard)
		}
	}

	// The retained route is genuinely live: round-trip a real request from S1's
	// retained (inbound) end back to S0 across the kept connection.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := pool1.Lookup(s0Shards[0])[0].SendAddXshardTxList(ctx, &wire.AddXshardTxListRequest{Branch: s0Shards[0], TxList: &types.CrossShardTransactionList{}}); err != nil {
		t.Fatalf("round-trip over retained route failed: %v", err)
	}
}

// TestXshardPool_HandleInboundDeadConnEvicted verifies that an inbound conn
// which closes before sending PING is evicted from the tracking set: dead
// connections must not accumulate.
func TestXshardPool_HandleInboundDeadConnEvicted(t *testing.T) {
	pool := mustNewXshardPool(t, []byte("local-slave"), []uint32{0x00030004}, []uint32{0x00030004})
	defer pool.Close()

	clientConn, serverConn := newRawConnPair(t)
	defer clientConn.Close()

	done := make(chan struct{})
	go func() {
		pool.HandleInbound(serverConn)
		close(done)
	}()

	// Wait until HandleInbound registers the connection as pending inbound.
	deadline := time.Now().Add(2 * time.Second)
	for pool.connectionsSize() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("HandleInbound did not register pending inbound")
		}
		time.Sleep(time.Millisecond)
	}

	// Remote disconnects without ever sending PING.
	clientConn.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("HandleInbound did not return after remote close")
	}

	if pool.connectionsSize() != 0 {
		t.Fatalf("expected dead inbound conn to be evicted, connectionsSize=%d", pool.connectionsSize())
	}
}

// TestXshardPool_HandleInboundPendingClose verifies the Go safety enhancement:
// a pending inbound connection (PING not yet received) is closed by pool Close,
// whereas Python leaks it.
func TestXshardPool_HandleInboundPendingClose(t *testing.T) {
	pool := mustNewXshardPool(t, []byte("local-slave"), []uint32{0x00030004}, []uint32{0x00030004})

	clientConn, serverConn := newRawConnPair(t)
	defer clientConn.Close()

	done := make(chan struct{})
	go func() {
		pool.HandleInbound(serverConn)
		close(done)
	}()

	// Wait until HandleInbound registers the connection as pending inbound.
	deadline := time.Now().Add(2 * time.Second)
	for pool.connectionsSize() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("HandleInbound did not register pending inbound")
		}
		time.Sleep(time.Millisecond)
	}

	pool.Close()

	// HandleInbound must unblock (waitUntilPingReceived returns false on close).
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("HandleInbound did not return after pool Close")
	}

	// The remote side observes EOF: the pending inbound was closed by Close.
	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := clientConn.Read(make([]byte, 1)); err == nil {
		t.Fatal("expected the pending inbound connection to be closed")
	}
}

// TestXshardPool_InboundDoesNotSkipSelf verifies the deliberate Python-parity
// divergence: HandleInbound does NOT skip a self connection, whereas DialToSlave
// does. Only the outbound pre-check guards against self; inbound is allowed to
// index a remote that claims our own identity (Python's handle_new_connection
// performs no self check).
func TestXshardPool_InboundDoesNotSkipSelf(t *testing.T) {
	pool := mustNewXshardPool(t, []byte("local-slave"), []uint32{0x00030004}, []uint32{0x00030004})
	defer pool.Close()

	// Inbound connection claiming to be self must still be indexed.
	establishInbound(t, pool, []byte("local-slave"), []uint32{0x00030004})

	if conns := pool.Lookup(0x00030004); len(conns) != 1 {
		t.Fatalf("expected self inbound connection to be indexed, got %d", len(conns))
	}
	if !pool.hasSlaveID([]byte("local-slave")) {
		t.Fatal("self ID should be tracked for inbound")
	}
}

// ── send-side response verification tests ────────────────────────────────────

// rawXshardPeer answers a single inbound request frame with an
// AddXshardTxListResponse carrying the given error code, over a net.Pipe.
func rawXshardPeer(t *testing.T, serverConn net.Conn, errCode uint32) <-chan error {
	t.Helper()
	peerDone := make(chan error, 1)
	go func() {
		request, err := wire.ReadFrameNoMeta(serverConn, 0)
		if err != nil {
			peerDone <- err
			return
		}
		payload, err := serialize.SerializeToBytes(&wire.AddXshardTxListResponse{ErrorCode: errCode})
		if err == nil {
			err = wire.WriteFrameNoMeta(serverConn, &wire.Frame{
				Opcode:  byte(wire.ClusterOpAddXshardTxListResponse),
				RPCID:   request.RPCID,
				Payload: payload,
			})
		}
		peerDone <- err
	}()
	return peerDone
}

func TestXshardConn_SendXshardTxListErrorCode(t *testing.T) {
	for _, tc := range []struct {
		name    string
		errCode uint32
		wantErr bool
	}{
		{"zero error code succeeds", 0, false},
		{"non-zero error code fails", 2, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clientConn, serverConn := net.Pipe()
			defer clientConn.Close()
			defer serverConn.Close()

			client, err := newXshardConn(clientConn, 0, []byte("client"), []uint32{1}, nil, nil, testXshardHandler{}, log.New())
			if err != nil {
				t.Fatalf("new conn: %v", err)
			}
			client.Start()
			peerDone := rawXshardPeer(t, serverConn, tc.errCode)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err = client.SendAddXshardTxList(ctx, &wire.AddXshardTxListRequest{Branch: 1, TxList: &types.CrossShardTransactionList{}})
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error for non-zero error_code, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("expected success for error_code 0, got: %v", err)
				}
			}
			if err := <-peerDone; err != nil {
				t.Fatalf("raw peer failed: %v", err)
			}
			if client.IsClosed() {
				t.Fatal("error_code should not close the connection")
			}
		})
	}
}

func TestNewXshardPool_NilLogger(t *testing.T) {
	pool, err := NewXshardPool(nil, nil, []uint32{1}, 0, testXshardHandler{}, nil)
	if err != nil {
		t.Fatalf("nil logger should be accepted: %v", err)
	}
	if pool == nil {
		t.Fatal("NewXshardPool returned nil")
	}
	pool.Close()
}

func TestNewXshardPool_NilHandler(t *testing.T) {
	if _, err := NewXshardPool(nil, nil, []uint32{1}, 0, nil, log.New()); err == nil {
		t.Fatal("expected error for nil handler")
	}
}

// ── remote slave helper ───────────────────────────────────────────────────────

// remoteSlave simulates a remote slave that answers PING with PONG. It counts
// accepted connections so tests can assert whether a dial happened.
type remoteSlave struct {
	ln       net.Listener
	host     string
	port     uint16
	accepted int32 // atomic

	mu    sync.Mutex
	conns []*XshardConn
	wg    sync.WaitGroup
}

func startRemoteSlave(t *testing.T, remoteID []byte, remoteShards []uint32) *remoteSlave {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	rs := &remoteSlave{ln: ln, host: addr.IP.String(), port: uint16(addr.Port)}
	rs.wg.Add(1)
	go rs.acceptLoop(remoteID, remoteShards)
	return rs
}

// slaveInfo builds a wire.SlaveInfo describing the simulated remote.
func (rs *remoteSlave) slaveInfo(id []byte, shards []uint32) wire.SlaveInfo {
	return wire.SlaveInfo{
		ID:              id,
		Host:            []byte(rs.host),
		Port:            rs.port,
		FullShardIDList: shards,
	}
}

func (rs *remoteSlave) acceptLoop(remoteID []byte, remoteShards []uint32) {
	defer rs.wg.Done()
	for {
		c, err := rs.ln.Accept()
		if err != nil {
			return
		}
		atomic.AddInt32(&rs.accepted, 1)
		conn, err := newXshardConn(c, 0, remoteID, remoteShards, nil, nil, testXshardHandler{}, log.New())
		if err != nil {
			c.Close()
			continue
		}
		conn.Start()
		rs.mu.Lock()
		rs.conns = append(rs.conns, conn)
		rs.mu.Unlock()
	}
}

func (rs *remoteSlave) acceptedCount() int {
	return int(atomic.LoadInt32(&rs.accepted))
}

func (rs *remoteSlave) close() {
	rs.ln.Close()
	rs.wg.Wait()
	rs.mu.Lock()
	for _, c := range rs.conns {
		c.Close()
	}
	rs.mu.Unlock()
}

// ── DialToSlave tests ─────────────────────────────────────────────────────────

// TestXshardPool_DialToSlaveSkipsExistingRemote verifies pre-dial dedup: dialing
// an already-tracked remote does not open a new TCP connection.
func TestXshardPool_DialToSlaveSkipsExistingRemote(t *testing.T) {
	rs := startRemoteSlave(t, []byte("remote-slave"), []uint32{0x00010001})
	defer rs.close()

	pool := mustNewXshardPool(t, []byte("local-slave"), []uint32{0x00030004}, []uint32{0x00010001})
	defer pool.Close()

	ctx := context.Background()

	if err := pool.DialToSlave(ctx, rs.slaveInfo([]byte("remote-slave"), []uint32{0x00010001})); err != nil {
		t.Fatalf("first dial: %v", err)
	}
	if rs.acceptedCount() != 1 {
		t.Fatalf("expected 1 accepted connection, got %d", rs.acceptedCount())
	}

	if err := pool.DialToSlave(ctx, rs.slaveInfo([]byte("remote-slave"), []uint32{0x00010001})); err != nil {
		t.Fatalf("second dial should be skipped: %v", err)
	}
	if rs.acceptedCount() != 1 {
		t.Fatalf("expected no new connection, got %d accepted", rs.acceptedCount())
	}
}

// TestXshardPool_DialToSlaveSkipsSelf verifies the pre-dial self guard.
func TestXshardPool_DialToSlaveSkipsSelf(t *testing.T) {
	rs := startRemoteSlave(t, []byte("local-slave"), []uint32{0x00030004})
	defer rs.close()

	pool := mustNewXshardPool(t, []byte("local-slave"), []uint32{0x00030004}, []uint32{0x00030004})
	defer pool.Close()

	ctx := context.Background()
	if err := pool.DialToSlave(ctx, rs.slaveInfo([]byte("local-slave"), []uint32{0x00030004})); err != nil {
		t.Fatalf("self dial should be skipped: %v", err)
	}
	if rs.acceptedCount() != 0 {
		t.Fatalf("expected no connection for self, got %d", rs.acceptedCount())
	}
	if pool.hasSlaveID([]byte("local-slave")) {
		t.Fatal("self ID should not be registered")
	}
}

// TestXshardPool_DialToSlaveConcurrentDialsRemainConnected verifies that
// concurrent dials to the same remote must not partition the pair nor lose
// the remote: both DialToSlave calls return success, the remote slave is
// registered, and the shard keeps at least one live delivery path. It does
// not mandate the number of retained connections, so it stays compatible
// both with keeping both duplicates (current entry-only dedup) and with a
// future deterministic convergence to a single logical route.
func TestXshardPool_DialToSlaveConcurrentDialsRemainConnected(t *testing.T) {
	rs := startRemoteSlave(t, []byte("remote-slave"), []uint32{0x00010001})
	defer rs.close()

	pool := mustNewXshardPool(t, []byte("local-slave"), []uint32{0x00030004}, []uint32{0x00010001})
	defer pool.Close()

	ctx := context.Background()

	const n = 2
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = pool.DialToSlave(ctx, rs.slaveInfo([]byte("remote-slave"), []uint32{0x00010001}))
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("dial %d: %v", i, err)
		}
	}

	if !pool.hasSlaveID([]byte("remote-slave")) {
		t.Fatal("remote-slave should be tracked")
	}
	// At least one live, routable connection must exist for the shard.
	conns := pool.Lookup(0x00010001)
	if len(conns) == 0 {
		t.Fatal("expected at least one route for the shard")
	}
	live := 0
	for _, c := range conns {
		if !c.IsClosed() {
			live++
		}
	}
	if live == 0 {
		t.Fatal("expected at least one live connection for the shard")
	}
}

// TestXshardPool_DialToSlaveRejectsMismatchedIdentity verifies the outbound
// identity validation branches: a PONG whose id or shard list does not match
// the master-advertised SlaveInfo must reject (close+evict) the tracked
// connection, leave no pool residue, and allow a later retry to succeed
// (Python slave.py connect_to_slave compares at py:885-890 but leaks the conn;
// Go rejects it instead).
func TestXshardPool_DialToSlaveRejectsMismatchedIdentity(t *testing.T) {
	for _, tc := range []struct {
		name          string
		pongID        []byte
		pongShards    []uint32
		wantSubstrErr string
	}{
		{"id mismatch", []byte("impostor"), []uint32{0x00010001}, "slave id mismatch"},
		{"shard list mismatch", []byte("remote-slave"), []uint32{0x00010002}, "shard list mismatch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool := mustNewXshardPool(t, []byte("local-slave"), []uint32{0x00030004}, []uint32{0x00010001})

			// One-shot raw responder: accepts a single connection, reads the
			// outbound PING frame, replies with a deliberately mismatched PONG,
			// and stops listening.
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("listen: %v", err)
			}
			addr := ln.Addr().(*net.TCPAddr)
			info := wire.SlaveInfo{
				ID:              []byte("remote-slave"),
				Host:            []byte(addr.IP.String()),
				Port:            uint16(addr.Port),
				FullShardIDList: []uint32{0x00010001},
			}

			go func() {
				c, acceptErr := ln.Accept()
				if acceptErr != nil {
					return
				}
				defer c.Close()

				frame, readErr := wire.ReadFrameNoMeta(c, 0)
				if readErr != nil {
					return
				}
				req := &wire.PingRequest{}
				buf := serialize.NewByteBuffer(frame.Payload)
				if deserializeErr := serialize.Deserialize(buf, req); deserializeErr != nil {
					return
				}
				payload, serErr := serialize.SerializeToBytes(&wire.PongResponse{
					ID:              tc.pongID,
					FullShardIDList: tc.pongShards,
				})
				if serErr != nil {
					return
				}
				_ = wire.WriteFrameNoMeta(c, &wire.Frame{
					Opcode:  byte(wire.ClusterOpPong),
					RPCID:   frame.RPCID,
					Payload: payload,
				})
			}()

			err = pool.DialToSlave(context.Background(), info)
			if err == nil || !strings.Contains(err.Error(), tc.wantSubstrErr) {
				t.Fatalf("expected %q error, got %v", tc.wantSubstrErr, err)
			}
			ln.Close() // responder exits on next accept or is already done

			// The rejected connection leaves no trace in any pool registry.
			if got := pool.connectionsSize(); got != 0 {
				t.Fatalf("rejected conn still tracked, connectionsSize=%d", got)
			}
			if pool.hasSlaveID([]byte("remote-slave")) {
				t.Fatal("slaveIDs polluted by rejected conn")
			}
			if conns := pool.Lookup(0x00010001); len(conns) != 0 {
				t.Fatalf("routing index polluted by rejected conn: %v", conns)
			}
			pool.Close()
		})
	}
}

// TestXshardPool_DialToSlaveRetryAfterFailure verifies a failed dial does not
// register the remote, so a later retry can still connect.
func TestXshardPool_DialToSlaveRetryAfterFailure(t *testing.T) {
	pool := mustNewXshardPool(t, []byte("local-slave"), []uint32{0x00030004}, []uint32{0x00010001})
	defer pool.Close()

	ctx := context.Background()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	deadAddr := ln.Addr().(*net.TCPAddr)
	ln.Close()
	deadInfo := wire.SlaveInfo{
		ID:              []byte("remote-slave"),
		Host:            []byte(deadAddr.IP.String()),
		Port:            uint16(deadAddr.Port),
		FullShardIDList: []uint32{0x00010001},
	}

	if err := pool.DialToSlave(ctx, deadInfo); err == nil {
		t.Fatal("expected dial failure to dead address")
	}
	if pool.hasSlaveID([]byte("remote-slave")) {
		t.Fatal("failed dial should not register the remote")
	}

	rs := startRemoteSlave(t, []byte("remote-slave"), []uint32{0x00010001})
	defer rs.close()
	if err := pool.DialToSlave(ctx, rs.slaveInfo([]byte("remote-slave"), []uint32{0x00010001})); err != nil {
		t.Fatalf("retry dial: %v", err)
	}
	if !pool.hasSlaveID([]byte("remote-slave")) {
		t.Fatal("retry should register the remote")
	}
}

// TestXshardPool_MutualDialKeepsLiveRoute verifies that when master tells both
// slaves to connect, mutual dial must not partition the pair: each side stays
// able to reach the peer with at least one live connection per shard. It does
// not mandate the connection count, so it is compatible both with keeping the
// inbound plus outbound duplicate (current entry-only dedup) and with a future
// deterministic convergence to a single logical route — the key invariant is
// that no indexed connection is ever a dead zombie (which would wedge the
// add-only slave ID registry and prevent any redial).
func TestXshardPool_MutualDialKeepsLiveRoute(t *testing.T) {
	s0ID, s1ID := []byte("s0"), []byte("s1")
	s0Shards := []uint32{1, 3, 5, 7}
	s1Shards := []uint32{2, 4, 6, 8}

	ln0, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen s0: %v", err)
	}
	defer ln0.Close()
	ln1, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen s1: %v", err)
	}
	defer ln1.Close()

	pool0 := mustNewXshardPool(t, s0ID, s0Shards, []uint32{1, 2, 3, 4, 5, 6, 7, 8})
	defer pool0.Close()
	pool1 := mustNewXshardPool(t, s1ID, s1Shards, []uint32{1, 2, 3, 4, 5, 6, 7, 8})
	defer pool1.Close()

	// Accept loops standing in for each slave's server port.
	go func() {
		for {
			c, err := ln0.Accept()
			if err != nil {
				return
			}
			pool0.HandleInbound(c)
		}
	}()
	go func() {
		for {
			c, err := ln1.Accept()
			if err != nil {
				return
			}
			pool1.HandleInbound(c)
		}
	}()

	a0 := ln0.Addr().(*net.TCPAddr)
	a1 := ln1.Addr().(*net.TCPAddr)
	info0 := wire.SlaveInfo{ID: s0ID, Host: []byte(a0.IP.String()), Port: uint16(a0.Port), FullShardIDList: s0Shards}
	info1 := wire.SlaveInfo{ID: s1ID, Host: []byte(a1.IP.String()), Port: uint16(a1.Port), FullShardIDList: s1Shards}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, pool := range []*XshardPool{pool0, pool1} {
		info := info0 // pool1 dials s0
		if i == 0 {
			info = info1 // pool0 dials s1
		}
		wg.Add(1)
		go func(pool *XshardPool, info wire.SlaveInfo, i int) {
			defer wg.Done()
			errs[i] = pool.DialToSlave(ctx, info)
		}(pool, info, i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("dial %d: %v", i, err)
		}
	}

	// Own shards must not be routed via this peer; each peer shard needs at
	// least one live connection and must never hold a dead zombie.
	assertShardState := func(pool *XshardPool, peerShards []uint32, peerID []byte) {
		t.Helper()
		if !pool.hasSlaveID(peerID) {
			t.Fatalf("peer %s should be tracked", peerID)
		}
		for _, shard := range peerShards {
			conns := pool.Lookup(shard)
			live := 0
			for _, c := range conns {
				if c.IsClosed() {
					t.Fatalf("shard %d: indexed connection is closed (zombie)", shard)
				}
				live++
			}
			if live == 0 {
				t.Fatalf("shard %d: no live connection to peer", shard)
			}
		}
	}
	assertShardState(pool0, s1Shards, s1ID)
	assertShardState(pool1, s0Shards, s0ID)
}
