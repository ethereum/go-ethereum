// Copyright 2026-2027, QuarkChain.

package slave

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/qkc/cluster/wire"
)

const defaultDialTimeout = 10 * time.Second

// XshardConn is a direct TCP connection to another slave node for cross-shard
// traffic. It uses 0-byte metadata (slave↔slave mode) and corresponds to Python's
// SlaveConnection.
//
// Architecture:
//
//	XshardConn  embeds  *rpcConn  embeds  *transport
//
// No forwarder — all frames are dispatched locally. RPC ID validation is
// global monotonic (the default in rpcConn).
type XshardConn struct {
	*rpcConn

	// local identity of this slave, used in PONG responses.
	localID              []byte
	localFullShardIDList []uint32

	// peer identity state, protected by its own mutex (not rpcConn.closeMu).
	stateMu               sync.Mutex
	remoteID              []byte
	remoteFullShardIDList []uint32
	pingReceived          chan struct{}
	pingOnce              sync.Once
}

// NewXshardConn dials another slave and returns an XshardConn.
// Call RegisterHandlers then Start before using the connection.
// maxPayloadSize controls frame payload size limit; 0 disables the limit.
// localID and localFullShardIDList identify this slave and are used in PONG responses.
func NewXshardConn(addr string, maxPayloadSize uint32, localID []byte, localFullShardIDList []uint32, logger log.Logger) (*XshardConn, error) {
	conn, err := net.DialTimeout("tcp", addr, defaultDialTimeout)
	if err != nil {
		return nil, fmt.Errorf("dial xshard slave %s: %w", addr, err)
	}
	return newXshardConn(conn, maxPayloadSize, localID, localFullShardIDList, logger), nil
}

// NewXshardConnFromConn wraps an accepted net.Conn as an XshardConn.
// maxPayloadSize controls frame payload size limit; 0 disables the limit.
// localID and localFullShardIDList identify this slave and are used in PONG responses.
func NewXshardConnFromConn(conn net.Conn, maxPayloadSize uint32, localID []byte, localFullShardIDList []uint32, logger log.Logger) *XshardConn {
	return newXshardConn(conn, maxPayloadSize, localID, localFullShardIDList, logger)
}

func newXshardConn(conn net.Conn, maxPayloadSize uint32, localID []byte, localFullShardIDList []uint32, logger log.Logger) *XshardConn {
	readFrame := func(r io.Reader) (*wire.Frame, error) {
		return wire.ReadFrameNoMeta(r, maxPayloadSize)
	}
	xc := &XshardConn{
		rpcConn:              newRPCConn(conn, readFrame, wire.WriteFrameNoMeta, logger),
		localID:              append([]byte(nil), localID...),
		localFullShardIDList: append([]uint32(nil), localFullShardIDList...),
		pingReceived:         make(chan struct{}),
	}

	// Register serializers for all opcodes that SlaveConnection understands.
	// This matches Python's SLAVE_OP_SERIALIZER_MAP.
	xc.rpcConn.RegisterOpSerializers(map[byte]*OpSerializer{
		byte(wire.ClusterOpPing):                        OpSerializerFor[wire.PingRequest, wire.PongResponse](),
		byte(wire.ClusterOpAddXshardTxListRequest):      OpSerializerFor[wire.AddXshardTxListRequest, wire.AddXshardTxListResponse](),
		byte(wire.ClusterOpBatchAddXshardTxListRequest): OpSerializerFor[wire.BatchAddXshardTxListRequest, wire.BatchAddXshardTxListResponse](),
	})

	// PING is handled internally by SlaveConnection in Python; register the
	// built-in handler immediately so it works even if the caller never calls
	// RegisterHandlers.
	xc.rpcConn.RegisterTypedHandlers(map[byte]TypedHandler{
		byte(wire.ClusterOpPing): xc.handlePing,
	})

	return xc
}

// handlePing is the built-in PING handler. It records peer identity, validates
// the shard list, and returns a PONG with this slave's identity.
func (x *XshardConn) handlePing(req any) (any, error) {
	ping := req.(*wire.PingRequest)

	// Record peer identity (only on first ping, matches Python's "if not self.id")
	x.stateMu.Lock()
	if len(x.remoteID) == 0 {
		x.remoteID = append([]byte(nil), ping.ID...)
		x.remoteFullShardIDList = append([]uint32(nil), ping.FullShardIDList...)
	}
	// Check stored shard list (matches Python's self.full_shard_id_list check)
	storedShardList := x.remoteFullShardIDList
	x.stateMu.Unlock()

	if len(storedShardList) == 0 {
		// Returning error causes rpcConn to close connection (Python's close_with_error)
		return nil, fmt.Errorf("empty shard list from slave %s", ping.ID)
	}

	// Signal ping received AFTER check passes (matches Python's ping_received_event.set())
	if !x.rpcConn.Closed() {
		x.pingOnce.Do(func() { close(x.pingReceived) })
	}

	return &wire.PongResponse{
		ID:              append([]byte(nil), x.localID...),
		FullShardIDList: append([]uint32(nil), x.localFullShardIDList...),
	}, nil
}

// RegisterHandlers registers user-provided opcode handlers. PING is always
// handled internally (see handlePing). If the user registers a PING handler,
// it is wrapped so that peer identity recording and empty-shard-list validation
// still happen first; the user's returned response object is then sent as the
// PONG body.
func (x *XshardConn) RegisterHandlers(handlers map[byte]TypedHandler) {
	wrapped := make(map[byte]TypedHandler, len(handlers))
	for opcode, handler := range handlers {
		if opcode != byte(wire.ClusterOpPing) {
			wrapped[opcode] = handler
		}
	}

	if userPingHandler, ok := handlers[byte(wire.ClusterOpPing)]; ok {
		wrapped[byte(wire.ClusterOpPing)] = func(req any) (any, error) {
			ping := req.(*wire.PingRequest)

			// Record peer identity (only on first ping)
			x.stateMu.Lock()
			if len(x.remoteID) == 0 {
				x.remoteID = append([]byte(nil), ping.ID...)
				x.remoteFullShardIDList = append([]uint32(nil), ping.FullShardIDList...)
			}
			// Check stored shard list
			storedShardList := x.remoteFullShardIDList
			x.stateMu.Unlock()

			if len(storedShardList) == 0 {
				return nil, fmt.Errorf("empty shard list from slave %s", ping.ID)
			}

			// Signal ping received AFTER check passes
			if !x.rpcConn.Closed() {
				x.pingOnce.Do(func() { close(x.pingReceived) })
			}

			return userPingHandler(req)
		}
	}

	x.rpcConn.RegisterTypedHandlers(wrapped)
}

// RemoteID returns the peer's slave ID, populated after the first PING.
func (x *XshardConn) RemoteID() []byte {
	x.stateMu.Lock()
	defer x.stateMu.Unlock()
	return append([]byte(nil), x.remoteID...)
}

// RemoteFullShardIDList returns the peer's full shard ID list, populated after
// the first PING.
func (x *XshardConn) RemoteFullShardIDList() []uint32 {
	x.stateMu.Lock()
	defer x.stateMu.Unlock()
	return append([]uint32(nil), x.remoteFullShardIDList...)
}

// WaitUntilPingReceived blocks until the first PING is received or the
// connection is closed. It returns true if the connection is still alive.
func (x *XshardConn) WaitUntilPingReceived() bool {
	select {
	case <-x.pingReceived:
		return !x.rpcConn.Closed()
	case <-x.rpcConn.Error():
		return false
	}
}

// SendPing sends a PING request and waits for PONG response. It returns the
// peer's id and full_shard_id_list from the PONG response.
// This is the outbound half of the slave-to-slave identity exchange,
// corresponding to Python's SlaveConnection.send_ping().
// The connection must have been started (Start() called).
func (x *XshardConn) SendPing(ctx context.Context) (id []byte, shardList []uint32, err error) {
	payload, err := serializeBytes(&wire.PingRequest{
		ID:              x.localID,
		FullShardIDList: x.localFullShardIDList,
		RootTip:         nil, // slave-to-slave: no root tip required
	})
	if err != nil {
		return nil, nil, fmt.Errorf("serialize ping: %w", err)
	}

	frame, err := x.rpcConn.SendRPC(ctx, byte(wire.ClusterOpPing), payload)
	if err != nil {
		return nil, nil, fmt.Errorf("send ping: %w", err)
	}

	var pong wire.PongResponse
	if err := deserializeBytes(frame.Payload, &pong); err != nil {
		return nil, nil, fmt.Errorf("deserialize pong: %w", err)
	}

	return pong.ID, pong.FullShardIDList, nil
}

// SendXshardTxList sends an AddXshardTxListRequest via RPC and returns the response.
// Python's ADD_XSHARD_TX_LIST_REQUEST is an RPC (in SLAVE_OP_RPC_MAP), not fire-and-forget.
func (x *XshardConn) SendXshardTxList(ctx context.Context, payload []byte) (*wire.Frame, error) {
	return x.rpcConn.SendRPC(ctx, byte(wire.ClusterOpAddXshardTxListRequest), payload)
}

// SendBatchXshardTxList sends a BatchAddXshardTxListRequest via RPC and returns the response.
// Python's BATCH_ADD_XSHARD_TX_LIST_REQUEST is an RPC (in SLAVE_OP_RPC_MAP).
func (x *XshardConn) SendBatchXshardTxList(ctx context.Context, payload []byte) (*wire.Frame, error) {
	return x.rpcConn.SendRPC(ctx, byte(wire.ClusterOpBatchAddXshardTxListRequest), payload)
}
