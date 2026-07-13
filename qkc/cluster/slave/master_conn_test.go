// Copyright 2026-2027, QuarkChain.

package slave

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/qkc/account"
	"github.com/ethereum/go-ethereum/qkc/cluster/wire"
	"github.com/ethereum/go-ethereum/qkc/serialize"
)

// newMasterTestConnPair creates a pair of MasterConns connected over a local TCP
// socket. The caller is responsible for calling cleanup.
func newMasterTestConnPair(t *testing.T) (client, server *MasterConn, cleanup func()) {
	t.Helper()
	return newMasterTestConnPairWithIdentity(
		t,
		[]byte("go-slave-client"), []uint32{0x00010001},
		[]byte("go-slave-server"), []uint32{0x00010001, 0x00020001},
	)
}

func newMasterTestConnPairWithIdentity(
	t *testing.T,
	clientID []byte, clientShards []uint32,
	serverID []byte, serverShards []uint32,
) (client, server *MasterConn, cleanup func()) {
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
	client = NewMasterConnFromConn(clientConn, 0, clientID, clientShards, logger)
	server = NewMasterConnFromConn(serverConn, 0, serverID, serverShards, logger)
	cleanup = func() {
		client.Close()
		server.Close()
	}
	return
}

// writeRawMasterFrame writes a raw ClusterMetadata frame directly to the
// underlying TCP connection, bypassing the connection's frame writer.
func writeRawMasterFrame(t *testing.T, conn net.Conn, frame *wire.Frame) {
	t.Helper()
	if err := wire.WriteFrame(conn, frame); err != nil {
		t.Fatalf("write raw frame: %v", err)
	}
}

// hasHandler reports whether the connection has a typed handler for opcode.
func hasHandler(c *MasterConn, opcode byte) bool {
	rv := reflect.ValueOf(c.rpcConn).Elem()
	handlers := rv.FieldByName("typedHandlers").MapKeys()
	for _, k := range handlers {
		if k.Uint() == uint64(opcode) {
			return true
		}
	}
	return false
}

// hasSerializer reports whether the connection has an OpSerializer for opcode.
func hasSerializer(c *MasterConn, opcode byte) bool {
	rv := reflect.ValueOf(c.rpcConn).Elem()
	serializers := rv.FieldByName("serializers").MapKeys()
	for _, k := range serializers {
		if k.Uint() == uint64(opcode) {
			return true
		}
	}
	return false
}

// TestMasterConn_AllMasterHandlersRegistered verifies that every master→slave
// request opcode has a handler registered and that the fire-and-forget opcode
// is marked as non-RPC.
func TestMasterConn_AllMasterHandlersRegistered(t *testing.T) {
	_, server, cleanup := newMasterTestConnPair(t)
	defer cleanup()

	masterRPCOps := []wire.ClusterOp{
		wire.ClusterOpPing,
		wire.ClusterOpConnectToSlavesRequest,
		wire.ClusterOpMineRequest,
		wire.ClusterOpGenTxRequest,
		wire.ClusterOpAddRootBlockRequest,
		wire.ClusterOpGetEcoInfoListRequest,
		wire.ClusterOpGetNextBlockToMineRequest,
		wire.ClusterOpAddMinorBlockRequest,
		wire.ClusterOpGetUnconfirmedHeadersRequest,
		wire.ClusterOpGetAccountDataRequest,
		wire.ClusterOpAddTransactionRequest,
		wire.ClusterOpCreateClusterPeerConnectionRequest,
		wire.ClusterOpGetMinorBlockRequest,
		wire.ClusterOpGetTransactionRequest,
		wire.ClusterOpSyncMinorBlockListRequest,
		wire.ClusterOpExecuteTransactionRequest,
		wire.ClusterOpGetTransactionReceiptRequest,
		wire.ClusterOpGetTransactionListByAddressRequest,
		wire.ClusterOpGetLogRequest,
		wire.ClusterOpEstimateGasRequest,
		wire.ClusterOpGetStorageRequest,
		wire.ClusterOpGetCodeRequest,
		wire.ClusterOpGasPriceRequest,
		wire.ClusterOpGetWorkRequest,
		wire.ClusterOpSubmitWorkRequest,
		wire.ClusterOpCheckMinorBlockRequest,
		wire.ClusterOpGetAllTransactionsRequest,
		wire.ClusterOpGetRootChainStakesRequest,
		wire.ClusterOpGetTotalBalanceRequest,
	}

	for _, op := range masterRPCOps {
		if !hasHandler(server, byte(op)) {
			t.Fatalf("missing handler for opcode 0x%02x (%v)", op, op)
		}
	}

	if !isNonRPC(server, byte(wire.ClusterOpDestroyClusterPeerConnectionCommand)) {
		t.Fatalf("DESTROY_CLUSTER_PEER_CONNECTION_COMMAND is not marked as non-RPC")
	}
}

// isNonRPC reports whether opcode is registered as fire-and-forget.
func isNonRPC(c *MasterConn, opcode byte) bool {
	rv := reflect.ValueOf(c.rpcConn).Elem()
	nonRPCOps := rv.FieldByName("nonRPCOps").MapKeys()
	for _, k := range nonRPCOps {
		if k.Uint() == uint64(opcode) {
			return true
		}
	}
	return false
}

// TestMasterConn_AllSerializersRegistered verifies that every ClusterOp defined
// in wire/opcode.go has a registered OpSerializer.
func TestMasterConn_AllSerializersRegistered(t *testing.T) {
	_, server, cleanup := newMasterTestConnPair(t)
	defer cleanup()

	for op := wire.ClusterOpPing; op <= wire.ClusterOpGetTotalBalanceResponse; op++ {
		if op == 0x9C { // 28 is intentionally skipped in Python
			continue
		}
		if !hasSerializer(server, byte(op)) {
			t.Fatalf("missing serializer for opcode 0x%02x (%v)", op, op)
		}
	}
}

// TestMasterConn_Ping verifies the master→slave PING handshake.
func TestMasterConn_Ping(t *testing.T) {
	clientID := []byte("go-slave-client")
	clientShards := []uint32{0x00010001}
	serverID := []byte("go-slave-server")
	serverShards := []uint32{0x00010001, 0x00020001}

	client, server, cleanup := newMasterTestConnPairWithIdentity(t, clientID, clientShards, serverID, serverShards)
	defer cleanup()

	server.Start()
	client.Start()

	pingPayload, err := serialize.SerializeToBytes(&wire.PingRequest{
		ID:              []byte("master"),
		FullShardIDList: []uint32{0x00010001},
		RootTip:         nil,
	})
	if err != nil {
		t.Fatalf("serialize ping: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.SendRPCMeta(ctx, byte(wire.ClusterOpPing), pingPayload, wire.ClusterMetadata{})
	if err != nil {
		t.Fatalf("send ping: %v", err)
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
}

// TestMasterConn_RPCRoundTrip verifies request/response dispatch for a
// representative set of master→slave RPCs.
func TestMasterConn_RPCRoundTrip(t *testing.T) {
	client, server, cleanup := newMasterTestConnPair(t)
	defer cleanup()

	server.Start()
	client.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cases := []struct {
		name       string
		opcode     wire.ClusterOp
		req        any
		resp       any
		respOpcode wire.ClusterOp
	}{
		{
			name:       "add_root_block",
			opcode:     wire.ClusterOpAddRootBlockRequest,
			req:        &wire.AddRootBlockRequest{RootBlock: emptyRawBytes(), ExpectSwitch: false},
			resp:       &wire.AddRootBlockResponse{},
			respOpcode: wire.ClusterOpAddRootBlockResponse,
		},
		{
			name:       "get_eco_info_list",
			opcode:     wire.ClusterOpGetEcoInfoListRequest,
			req:        &wire.GetEcoInfoListRequest{},
			resp:       &wire.GetEcoInfoListResponse{},
			respOpcode: wire.ClusterOpGetEcoInfoListResponse,
		},
		{
			name:       "add_transaction",
			opcode:     wire.ClusterOpAddTransactionRequest,
			req:        &wire.AddTransactionRequest{Tx: emptyRawBytes()},
			resp:       &wire.AddTransactionResponse{},
			respOpcode: wire.ClusterOpAddTransactionResponse,
		},
		{
			name:       "get_minor_block",
			opcode:     wire.ClusterOpGetMinorBlockRequest,
			req:        &wire.GetMinorBlockRequest{Branch: 0x00010001, Height: 1, NeedExtraInfo: false},
			resp:       &wire.GetMinorBlockResponse{},
			respOpcode: wire.ClusterOpGetMinorBlockResponse,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := serialize.SerializeToBytes(tc.req)
			if err != nil {
				t.Fatalf("serialize request: %v", err)
			}

			frame, err := client.SendRPCMeta(ctx, byte(tc.opcode), payload, wire.ClusterMetadata{Branch: 0x00010001})
			if err != nil {
				t.Fatalf("send rpc: %v", err)
			}
			if frame.Opcode != byte(tc.respOpcode) {
				t.Fatalf("expected response opcode 0x%x, got 0x%x", tc.respOpcode, frame.Opcode)
			}
			if frame.Meta.Branch != 0x00010001 {
				t.Fatalf("metadata branch not preserved: got %d", frame.Meta.Branch)
			}

			if err := serialize.Deserialize(serialize.NewByteBuffer(frame.Payload), tc.resp); err != nil {
				t.Fatalf("deserialize response: %v", err)
			}
		})
	}
}

// TestMasterConn_NonRPCDispatch verifies that the fire-and-forget
// DESTROY_CLUSTER_PEER_CONNECTION_COMMAND is accepted with rpc_id == 0 and does
// not produce a response or close the connection.
func TestMasterConn_NonRPCDispatch(t *testing.T) {
	client, server, cleanup := newMasterTestConnPair(t)
	defer cleanup()

	server.Start()
	client.Start()

	payload, err := serialize.SerializeToBytes(&wire.DestroyClusterPeerConnectionCommand{ClusterPeerID: 42})
	if err != nil {
		t.Fatalf("serialize command: %v", err)
	}

	// Write a non-RPC frame directly; no response should come back, but the
	// connection must remain usable for a subsequent RPC.
	writeRawMasterFrame(t, client.conn, &wire.Frame{
		Meta:    wire.ClusterMetadata{Branch: 0x00010001},
		Opcode:  byte(wire.ClusterOpDestroyClusterPeerConnectionCommand),
		RPCID:   0,
		Payload: payload,
	})

	// Give the server a moment to process the command.
	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pingPayload, _ := serialize.SerializeToBytes(&wire.PingRequest{
		ID:              []byte("master"),
		FullShardIDList: []uint32{0x00010001},
	})
	resp, err := client.SendRPCMeta(ctx, byte(wire.ClusterOpPing), pingPayload, wire.ClusterMetadata{})
	if err != nil {
		t.Fatalf("ping after non-rpc command failed: %v", err)
	}
	if resp.Opcode != byte(wire.ClusterOpPong) {
		t.Fatalf("expected pong, got opcode 0x%x", resp.Opcode)
	}
}

// TestMasterConn_NonRPCWithNonZeroRPCID verifies that a non-RPC command with a
// non-zero rpc_id causes the server to close the connection.
func TestMasterConn_NonRPCWithNonZeroRPCID(t *testing.T) {
	client, server, cleanup := newMasterTestConnPair(t)
	defer cleanup()

	server.Start()
	client.Start()

	payload, _ := serialize.SerializeToBytes(&wire.DestroyClusterPeerConnectionCommand{ClusterPeerID: 42})

	writeRawMasterFrame(t, client.conn, &wire.Frame{
		Meta:    wire.ClusterMetadata{Branch: 0x00010001},
		Opcode:  byte(wire.ClusterOpDestroyClusterPeerConnectionCommand),
		RPCID:   1, // non-RPC must have rpc_id == 0
		Payload: payload,
	})

	select {
	case <-server.WaitUntilClosed():
	case <-time.After(2 * time.Second):
		t.Fatal("server did not close after non-rpc with non-zero rpc_id")
	}
}

// TestMasterConn_Forwarder verifies that frames with cluster_peer_id != 0 are
// routed through the forwarder hook and are not dispatched locally.
func TestMasterConn_Forwarder(t *testing.T) {
	client, server, cleanup := newMasterTestConnPair(t)
	defer cleanup()

	var forwardedMu sync.Mutex
	var forwarded []*wire.Frame
	server.SetForwarder(func(frame *wire.Frame) bool {
		if frame.Meta.ClusterPeerID == 0 {
			return false
		}
		forwardedMu.Lock()
		forwarded = append(forwarded, frame)
		forwardedMu.Unlock()
		return true
	})

	server.Start()
	client.Start()

	payload, _ := serialize.SerializeToBytes(&wire.GetMinorBlockRequest{Branch: 0x00010001, Height: 1})

	// Peer-originated frame: cluster_peer_id != 0.
	writeRawMasterFrame(t, client.conn, &wire.Frame{
		Meta:    wire.ClusterMetadata{Branch: 0x00010001, ClusterPeerID: 123},
		Opcode:  byte(wire.ClusterOpGetMinorBlockRequest),
		RPCID:   7,
		Payload: payload,
	})

	time.Sleep(100 * time.Millisecond)

	forwardedMu.Lock()
	count := len(forwarded)
	forwardedMu.Unlock()
	if count != 1 {
		t.Fatalf("expected 1 forwarded frame, got %d", count)
	}

	// Connection should still be open; a subsequent master RPC works.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pingPayload, _ := serialize.SerializeToBytes(&wire.PingRequest{ID: []byte("m"), FullShardIDList: []uint32{1}})
	resp, err := client.SendRPCMeta(ctx, byte(wire.ClusterOpPing), pingPayload, wire.ClusterMetadata{})
	if err != nil {
		t.Fatalf("ping after forwarded frame failed: %v", err)
	}
	if resp.Opcode != byte(wire.ClusterOpPong) {
		t.Fatalf("expected pong, got 0x%x", resp.Opcode)
	}
}

// TestMasterConn_UnsupportedOpcodeClosesConnection verifies that an opcode
// without a registered handler and rpc_id == 0 causes the server to close the
// connection.
func TestMasterConn_UnsupportedOpcodeClosesConnection(t *testing.T) {
	client, server, cleanup := newMasterTestConnPair(t)
	defer cleanup()

	server.Start()
	client.Start()

	// 0x01 is a CommandOp with no handler registered on the master side.
	// rpc_id must be 0 for the server to treat it as a non-RPC unsupported
	// command and close the connection.
	writeRawMasterFrame(t, client.conn, &wire.Frame{
		Meta:    wire.ClusterMetadata{},
		Opcode:  0x01,
		RPCID:   0,
		Payload: []byte("payload"),
	})

	select {
	case <-server.WaitUntilClosed():
	case <-time.After(2 * time.Second):
		t.Fatal("server did not close after unsupported opcode")
	}
}

// TestMasterConn_RPCIDMonotonic verifies that duplicate RPC IDs cause the
// server to close the connection.
func TestMasterConn_RPCIDMonotonic(t *testing.T) {
	client, server, cleanup := newMasterTestConnPair(t)
	defer cleanup()

	server.Start()
	client.Start()

	pingPayload, _ := serialize.SerializeToBytes(&wire.PingRequest{
		ID:              []byte("master"),
		FullShardIDList: []uint32{0x00010001},
	})

	// Manually send two PING frames with the same RPC ID.
	writeRawMasterFrame(t, client.conn, &wire.Frame{
		Meta:    wire.ClusterMetadata{},
		Opcode:  byte(wire.ClusterOpPing),
		RPCID:   1,
		Payload: pingPayload,
	})
	writeRawMasterFrame(t, client.conn, &wire.Frame{
		Meta:    wire.ClusterMetadata{},
		Opcode:  byte(wire.ClusterOpPing),
		RPCID:   1, // duplicate
		Payload: pingPayload,
	})

	select {
	case <-server.WaitUntilClosed():
	case <-time.After(2 * time.Second):
		t.Fatal("server did not close after duplicate rpc_id")
	}
}

// TestMasterConn_RPCIDDecreasing verifies that a decreasing RPC ID causes the
// server to close the connection.
func TestMasterConn_RPCIDDecreasing(t *testing.T) {
	client, server, cleanup := newMasterTestConnPair(t)
	defer cleanup()

	server.Start()
	client.Start()

	pingPayload, _ := serialize.SerializeToBytes(&wire.PingRequest{
		ID:              []byte("master"),
		FullShardIDList: []uint32{0x00010001},
	})

	writeRawMasterFrame(t, client.conn, &wire.Frame{
		Meta:    wire.ClusterMetadata{},
		Opcode:  byte(wire.ClusterOpPing),
		RPCID:   2,
		Payload: pingPayload,
	})
	writeRawMasterFrame(t, client.conn, &wire.Frame{
		Meta:    wire.ClusterMetadata{},
		Opcode:  byte(wire.ClusterOpPing),
		RPCID:   1, // decreasing
		Payload: pingPayload,
	})

	select {
	case <-server.WaitUntilClosed():
	case <-time.After(2 * time.Second):
		t.Fatal("server did not close after decreasing rpc_id")
	}
}

// TestMasterConn_CloseWakesPendingRPC verifies that Close wakes all pending
// outbound RPCs with ErrConnectionClosed.
func TestMasterConn_CloseWakesPendingRPC(t *testing.T) {
	client, _, cleanup := newMasterTestConnPair(t)
	defer cleanup()

	// Server intentionally left unstarted so it never replies.
	client.Start()

	var wg sync.WaitGroup
	wg.Add(1)
	errChan := make(chan error, 1)
	go func() {
		wg.Done()
		_, err := client.SendRPCMeta(context.Background(), byte(wire.ClusterOpPing), []byte("ping"), wire.ClusterMetadata{})
		errChan <- err
	}()

	wg.Wait()
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

// TestMasterConn_OutboundRPCMeta verifies that outbound RPCs from the slave
// encode ClusterMetadata correctly on the wire.
func TestMasterConn_OutboundRPCMeta(t *testing.T) {
	client, server, cleanup := newMasterTestConnPair(t)
	defer cleanup()

	// Server echoes the request opcode + 1 and preserves metadata.
	server.RegisterTypedHandlers(map[byte]TypedHandler{
		byte(wire.ClusterOpGetEcoInfoListRequest): func(req any) (any, error) {
			_ = req.(*wire.GetEcoInfoListRequest)
			return &wire.GetEcoInfoListResponse{ErrorCode: 0}, nil
		},
	})

	server.Start()
	client.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	meta := wire.ClusterMetadata{Branch: 0x00010001, ClusterPeerID: 99}
	payload, _ := serialize.SerializeToBytes(&wire.GetEcoInfoListRequest{})
	resp, err := client.SendRPCMeta(ctx, byte(wire.ClusterOpGetEcoInfoListRequest), payload, meta)
	if err != nil {
		t.Fatalf("send rpc: %v", err)
	}
	if resp.Opcode != byte(wire.ClusterOpGetEcoInfoListResponse) {
		t.Fatalf("unexpected response opcode 0x%x", resp.Opcode)
	}
	if resp.Meta.Branch != meta.Branch || resp.Meta.ClusterPeerID != meta.ClusterPeerID {
		t.Fatalf("response metadata mismatch: got %+v, want %+v", resp.Meta, meta)
	}
}

// TestMasterConn_SendAddMinorBlockHeader verifies the typed outbound helper.
func TestMasterConn_SendAddMinorBlockHeader(t *testing.T) {
	client, server, cleanup := newMasterTestConnPair(t)
	defer cleanup()

	server.RegisterTypedHandlers(map[byte]TypedHandler{
		byte(wire.ClusterOpAddMinorBlockHeaderRequest): func(req any) (any, error) {
			r := req.(*wire.AddMinorBlockHeaderRequest)
			if r.TxCount != 5 {
				t.Fatalf("unexpected tx_count: %d", r.TxCount)
			}
			return &wire.AddMinorBlockHeaderResponse{ErrorCode: 0}, nil
		},
	})

	server.Start()
	client.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &wire.AddMinorBlockHeaderRequest{
		MinorBlockHeader:  emptyRawBytes(),
		TxCount:           5,
		XShardTxCount:     0,
		CoinbaseAmountMap: emptyRawBytes(),
		ShardStats:        wire.ShardStats{Branch: 0x00010001},
	}
	resp, err := client.SendAddMinorBlockHeader(ctx, req)
	if err != nil {
		t.Fatalf("SendAddMinorBlockHeader: %v", err)
	}
	if resp.ErrorCode != 0 {
		t.Fatalf("unexpected error_code: %d", resp.ErrorCode)
	}
}

// TestMasterConn_12ByteMetadata verifies that ClusterMetadata is encoded as
// 4-byte branch followed by 8-byte cluster_peer_id.
func TestMasterConn_12ByteMetadata(t *testing.T) {
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
	defer clientConn.Close()
	defer serverConn.Close()

	mc := NewMasterConnFromConn(clientConn, 0, []byte("s"), []uint32{1}, log.New())
	defer mc.Close()

	// Read the raw first frame written by the client to inspect metadata layout.
	go func() {
		// Accept but do not respond; we only need the wire bytes.
		buf := make([]byte, 1024)
		_, _ = serverConn.Read(buf)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	payload, _ := serialize.SerializeToBytes(&wire.GetEcoInfoListRequest{})
	// This RPC will time out because the fake server does not reply, but the
	// request bytes are still written to the wire before the timeout.
	_, _ = mc.SendRPCMeta(ctx, byte(wire.ClusterOpGetEcoInfoListRequest), payload, wire.ClusterMetadata{Branch: 0x01020304, ClusterPeerID: 0x1122334455667788})

	// Read back what the client wrote from serverConn using a fresh connection
	// is not straightforward; instead verify the metadata marshal helper.
	meta := wire.ClusterMetadata{Branch: 0x01020304, ClusterPeerID: 0x1122334455667788}
	b := wire.MarshalClusterMetadata(meta)
	if len(b) != 12 {
		t.Fatalf("metadata length: got %d, want 12", len(b))
	}
	if binary.BigEndian.Uint32(b[0:4]) != meta.Branch {
		t.Fatalf("branch mismatch")
	}
	if binary.BigEndian.Uint64(b[4:12]) != meta.ClusterPeerID {
		t.Fatalf("cluster_peer_id mismatch")
	}
}

// TestMasterConn_StubResponsesAreValidBytes verifies that every master handler
// stub returns a response that can be serialized.
func TestMasterConn_StubResponsesAreValidBytes(t *testing.T) {
	_, server, cleanup := newMasterTestConnPair(t)
	defer cleanup()

	server.Start()

	cases := []struct {
		opcode wire.ClusterOp
		req    any
		resp   any
	}{
		{wire.ClusterOpPing, &wire.PingRequest{ID: []byte("m"), FullShardIDList: []uint32{1}}, &wire.PongResponse{}},
		{wire.ClusterOpConnectToSlavesRequest, &wire.ConnectToSlavesRequest{SlaveInfoList: []wire.SlaveInfo{}}, &wire.ConnectToSlavesResponse{}},
		{wire.ClusterOpMineRequest, &wire.MineRequest{}, &wire.MineResponse{}},
		{wire.ClusterOpGenTxRequest, &wire.GenTxRequest{Tx: emptyRawBytes()}, &wire.GenTxResponse{}},
		{wire.ClusterOpAddRootBlockRequest, &wire.AddRootBlockRequest{RootBlock: emptyRawBytes()}, &wire.AddRootBlockResponse{}},
		{wire.ClusterOpGetEcoInfoListRequest, &wire.GetEcoInfoListRequest{}, &wire.GetEcoInfoListResponse{}},
		{wire.ClusterOpGetNextBlockToMineRequest, &wire.GetNextBlockToMineRequest{Address: account.Address{}}, &wire.GetNextBlockToMineResponse{}},
		{wire.ClusterOpAddMinorBlockRequest, &wire.AddMinorBlockRequest{MinorBlockData: []byte{}}, &wire.AddMinorBlockResponse{}},
		{wire.ClusterOpGetUnconfirmedHeadersRequest, &wire.GetUnconfirmedHeadersRequest{}, &wire.GetUnconfirmedHeadersResponse{}},
		{wire.ClusterOpGetAccountDataRequest, &wire.GetAccountDataRequest{}, &wire.GetAccountDataResponse{}},
		{wire.ClusterOpAddTransactionRequest, &wire.AddTransactionRequest{Tx: emptyRawBytes()}, &wire.AddTransactionResponse{}},
		{wire.ClusterOpCreateClusterPeerConnectionRequest, &wire.CreateClusterPeerConnectionRequest{ClusterPeerID: 1}, &wire.CreateClusterPeerConnectionResponse{}},
		{wire.ClusterOpGetMinorBlockRequest, &wire.GetMinorBlockRequest{}, &wire.GetMinorBlockResponse{}},
		{wire.ClusterOpGetTransactionRequest, &wire.GetTransactionRequest{}, &wire.GetTransactionResponse{}},
		{wire.ClusterOpSyncMinorBlockListRequest, &wire.SyncMinorBlockListRequest{MinorBlockHashList: [][wire.HashLength]byte{}}, &wire.SyncMinorBlockListResponse{}},
		{wire.ClusterOpExecuteTransactionRequest, &wire.ExecuteTransactionRequest{Tx: emptyRawBytes()}, &wire.ExecuteTransactionResponse{}},
		{wire.ClusterOpGetTransactionReceiptRequest, &wire.GetTransactionReceiptRequest{}, &wire.GetTransactionReceiptResponse{}},
		{wire.ClusterOpGetTransactionListByAddressRequest, &wire.GetTransactionListByAddressRequest{}, &wire.GetTransactionListByAddressResponse{}},
		{wire.ClusterOpGetLogRequest, &wire.GetLogRequest{}, &wire.GetLogResponse{}},
		{wire.ClusterOpEstimateGasRequest, &wire.EstimateGasRequest{Tx: emptyRawBytes()}, &wire.EstimateGasResponse{}},
		{wire.ClusterOpGetStorageRequest, &wire.GetStorageRequest{}, &wire.GetStorageResponse{}},
		{wire.ClusterOpGetCodeRequest, &wire.GetCodeRequest{}, &wire.GetCodeResponse{}},
		{wire.ClusterOpGasPriceRequest, &wire.GasPriceRequest{}, &wire.GasPriceResponse{}},
		{wire.ClusterOpGetWorkRequest, &wire.GetWorkRequest{}, &wire.GetWorkResponse{}},
		{wire.ClusterOpSubmitWorkRequest, &wire.SubmitWorkRequest{}, &wire.SubmitWorkResponse{}},
		{wire.ClusterOpCheckMinorBlockRequest, &wire.CheckMinorBlockRequest{MinorBlockHeader: emptyRawBytes()}, &wire.CheckMinorBlockResponse{}},
		{wire.ClusterOpGetAllTransactionsRequest, &wire.GetAllTransactionsRequest{}, &wire.GetAllTransactionsResponse{}},
		{wire.ClusterOpGetRootChainStakesRequest, &wire.GetRootChainStakesRequest{}, &wire.GetRootChainStakesResponse{}},
		{wire.ClusterOpGetTotalBalanceRequest, &wire.GetTotalBalanceRequest{}, &wire.GetTotalBalanceResponse{}},
	}

	for _, tc := range cases {
		// Serialize the request bytes.
		reqBytes, err := serialize.SerializeToBytes(tc.req)
		if err != nil {
			t.Fatalf("serialize request for opcode 0x%x: %v", tc.opcode, err)
		}

		// Ask the server to process the request by writing a raw frame.
		// We use a fresh connection per case to avoid ordering issues.
		client, srv, cleanupPair := newMasterTestConnPair(t)
		srv.Start()

		writeRawMasterFrame(t, client.conn, &wire.Frame{
			Meta:    wire.ClusterMetadata{Branch: 0x00010001},
			Opcode:  byte(tc.opcode),
			RPCID:   1,
			Payload: reqBytes,
		})

		// Read the raw response from the connection.
		clientConn := client.conn
		clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		frame, err := wire.ReadFrame(clientConn, 0)
		if err != nil {
			t.Fatalf("read response for opcode 0x%x: %v", tc.opcode, err)
		}
		if frame.Opcode != byte(tc.opcode)+1 {
			t.Fatalf("opcode 0x%x: expected response opcode 0x%x, got 0x%x", tc.opcode, byte(tc.opcode)+1, frame.Opcode)
		}
		if err := serialize.Deserialize(serialize.NewByteBuffer(frame.Payload), tc.resp); err != nil {
			t.Fatalf("deserialize response for opcode 0x%x: %v", tc.opcode, err)
		}

		cleanupPair()
	}
}

// TestMasterConn_EmptyPayloadDeserialization verifies that request types with
// empty bodies deserialize correctly.
func TestMasterConn_EmptyPayloadDeserialization(t *testing.T) {
	client, server, cleanup := newMasterTestConnPair(t)
	defer cleanup()

	// Only start the server; the client connection is used as a bare socket so
	// we can observe the raw response frame without the client's readLoop
	// competing for bytes.
	server.Start()

	// Empty payload should deserialize to an empty GetEcoInfoListRequest.
	writeRawMasterFrame(t, client.conn, &wire.Frame{
		Meta:    wire.ClusterMetadata{},
		Opcode:  byte(wire.ClusterOpGetEcoInfoListRequest),
		RPCID:   1,
		Payload: []byte{},
	})

	// Read the response from the same connection the client wrote on.
	client.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	frame, err := wire.ReadFrame(client.conn, 0)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if frame.Opcode != byte(wire.ClusterOpGetEcoInfoListResponse) {
		t.Fatalf("expected GetEcoInfoListResponse, got 0x%x", frame.Opcode)
	}
}

// TestMasterConn_MetadataPreserved verifies that request metadata is echoed
// back in the response.
func TestMasterConn_MetadataPreserved(t *testing.T) {
	client, server, cleanup := newMasterTestConnPair(t)
	defer cleanup()

	server.Start()
	client.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	meta := wire.ClusterMetadata{Branch: 0xDEADBEEF, ClusterPeerID: 0xCAFEBABECAFEBABE}
	payload, _ := serialize.SerializeToBytes(&wire.GetEcoInfoListRequest{})
	resp, err := client.SendRPCMeta(ctx, byte(wire.ClusterOpGetEcoInfoListRequest), payload, meta)
	if err != nil {
		t.Fatalf("send rpc: %v", err)
	}
	if resp.Meta != meta {
		t.Fatalf("metadata not preserved: got %+v, want %+v", resp.Meta, meta)
	}
}

// TestMasterConn_FrameWireLayout verifies the full ClusterMetadata frame layout
// written by MasterConn matches the Python protocol.
func TestMasterConn_FrameWireLayout(t *testing.T) {
	var buf bytes.Buffer
	frame := &wire.Frame{
		Meta:    wire.ClusterMetadata{Branch: 0x01020304, ClusterPeerID: 0x1122334455667788},
		Opcode:  byte(wire.ClusterOpPing),
		RPCID:   0xAABBCCDDEEFF0011,
		Payload: []byte{0xAA, 0xBB},
	}
	if err := wire.WriteFrame(&buf, frame); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}

	wireBytes := buf.Bytes()
	if len(wireBytes) != 4+12+1+8+2 {
		t.Fatalf("frame length: got %d, want %d", len(wireBytes), 4+12+1+8+2)
	}

	if got := binary.BigEndian.Uint32(wireBytes[0:4]); got != 2 {
		t.Fatalf("payload_len: got %d, want 2", got)
	}
	if got := binary.BigEndian.Uint32(wireBytes[4:8]); got != frame.Meta.Branch {
		t.Fatalf("branch mismatch: got 0x%x", got)
	}
	if got := binary.BigEndian.Uint64(wireBytes[8:16]); got != frame.Meta.ClusterPeerID {
		t.Fatalf("cluster_peer_id mismatch: got 0x%x", got)
	}
	if wireBytes[16] != frame.Opcode {
		t.Fatalf("opcode mismatch: got 0x%x", wireBytes[16])
	}
	if got := binary.BigEndian.Uint64(wireBytes[17:25]); got != frame.RPCID {
		t.Fatalf("rpc_id mismatch: got 0x%x", got)
	}
	if !bytes.Equal(wireBytes[25:], frame.Payload) {
		t.Fatalf("payload mismatch: got %x", wireBytes[25:])
	}
}
