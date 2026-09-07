// Copyright 2026-2027, QuarkChain.

package slave

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/qkc/cluster/conn"
	"github.com/ethereum/go-ethereum/qkc/cluster/wire"
	"github.com/ethereum/go-ethereum/qkc/serialize"
	"github.com/ethereum/go-ethereum/qkc/types"
)

// XshardHandler serves inbound xshard requests, implemented by the business
// layer. Implementations must be safe for concurrent calls.
//
// A returned error signals a connection-level failure and closes the
// connection via BaseConn; business failures must be encoded in the response
// ErrorCode instead.
type XshardHandler interface {
	AddXshardTxList(req *wire.AddXshardTxListRequest) (*wire.AddXshardTxListResponse, error)

	BatchAddXshardTxList(req *wire.BatchAddXshardTxListRequest) (*wire.BatchAddXshardTxListResponse, error)
}

// XshardConn is a direct TCP connection to another slave using 0-byte
// metadata (slave↔slave mode). It embeds conn.BaseConn and adds slave peer
// identity plus xshard-specific operations.
//
// Connections are owned by XshardPool and obtained via XshardPool.Lookup. A
// looked-up connection may be closed concurrently at any time; callers must
// tolerate operating on closed connections (sends just fail).
type XshardConn struct {
	*conn.BaseConn

	handler XshardHandler

	localID              []byte // this slave's identity, sent in PING/PONG
	localFullShardIDList []uint32

	// Peer identity, immutable once published: constructor-injected for
	// outbound (master-advertised SlaveInfo), recorded on the first PING for
	// inbound. close(pingReceived) under pingOnce.Do both marks completion
	// and happens-before publishes the fields to all lock-free readers.
	peerID              []byte
	peerFullShardIDList []uint32
	pingReceived        chan struct{} // closed on the first PING (py: ping_received_event)
	pingOnce            sync.Once     // keeps the close exactly-once under concurrent PINGs

	// inboundRegistered, set only by the owning pool's HandleInbound before
	// Start, is closed once the pool has committed the inbound connection to
	// its routing index. handlePing blocks on it before returning PONG, so a
	// successful PONG is only observable after registration completes. This
	// re-establishes explicitly the ordering Python gets implicitly from its
	// single asyncio event loop (its ready-queue FIFO runs the awakened
	// handle_new_connection — including _add_slave_connection — before any
	// command that causally follows the PONG). Nil for outbound connections
	// and bare test connections: no barrier, DialToSlave registers those
	// itself after the handshake.
	inboundRegistered chan struct{}
}

// newXshardConn creates a slave-to-slave connection. Inbound callers pass nil
// peer identity; it is then recorded from the first PING.
func newXshardConn(nc net.Conn, maxPayloadSize uint32, localID []byte, localFullShardIDList []uint32, peerID []byte, peerShardList []uint32, handler XshardHandler, logger log.Logger) (*XshardConn, error) {
	if handler == nil {
		return nil, errors.New("xshard handler must not be nil")
	}
	xc := &XshardConn{
		handler:              handler,
		localID:              append([]byte(nil), localID...),
		localFullShardIDList: append([]uint32(nil), localFullShardIDList...),
		peerID:               append([]byte(nil), peerID...),
		peerFullShardIDList:  append([]uint32(nil), peerShardList...),
		pingReceived:         make(chan struct{}),
	}
	xc.BaseConn = conn.NewBaseConn(conn.Config{
		Transport: conn.NewTCPTransport(
			nc,
			func(r io.Reader) (*wire.Frame, error) {
				return wire.ReadFrameNoMeta(r, maxPayloadSize)
			},
			wire.WriteFrameNoMeta,
		),
		Serializers: map[byte]*conn.OpSerializer{
			byte(wire.ClusterOpPing):                        conn.OpSerializerFor[wire.PingRequest, wire.PongResponse](byte(wire.ClusterOpPong)),
			byte(wire.ClusterOpAddXshardTxListRequest):      conn.OpSerializerFor[wire.AddXshardTxListRequest, wire.AddXshardTxListResponse](byte(wire.ClusterOpAddXshardTxListResponse)),
			byte(wire.ClusterOpBatchAddXshardTxListRequest): conn.OpSerializerFor[wire.BatchAddXshardTxListRequest, wire.BatchAddXshardTxListResponse](byte(wire.ClusterOpBatchAddXshardTxListResponse)),
		},
		Handlers: map[byte]conn.TypedHandler{
			byte(wire.ClusterOpPing):                        xc.handlePing,
			byte(wire.ClusterOpAddXshardTxListRequest):      xc.handleAddXshardTxList,
			byte(wire.ClusterOpBatchAddXshardTxListRequest): xc.handleBatchAddXshardTxList,
		},
		Logger: logger,
	})
	return xc, nil
}

// Public API

// RemoteID returns a copy of the peer's id. Metadata is immutable once
// published (see the peerID field), so the read needs no lock.
func (x *XshardConn) RemoteID() []byte {
	return append([]byte(nil), x.peerID...)
}

// RemoteFullShardIDList returns a copy of the peer's full shard ID list,
// subject to the same guarantees as RemoteID.
func (x *XshardConn) RemoteFullShardIDList() []uint32 {
	return append([]uint32(nil), x.peerFullShardIDList...)
}

// SendAddXshardTxList sends an AddXshardTxListRequest to the peer.
func (x *XshardConn) SendAddXshardTxList(ctx context.Context, req *wire.AddXshardTxListRequest) error {
	resp, err := x.sendRPC(ctx, byte(wire.ClusterOpAddXshardTxListRequest), req)
	if err != nil {
		return err
	}

	r, ok := resp.(*wire.AddXshardTxListResponse)
	if !ok {
		return fmt.Errorf("unexpected response %T", resp)
	}
	if r.ErrorCode != 0 {
		return fmt.Errorf("AddXshardTxList failed: %d", r.ErrorCode)
	}

	return nil
}

// SendBatchAddXshardTxList sends a BatchAddXshardTxListRequest to the peer.
func (x *XshardConn) SendBatchAddXshardTxList(ctx context.Context, req *wire.BatchAddXshardTxListRequest) error {
	resp, err := x.sendRPC(ctx, byte(wire.ClusterOpBatchAddXshardTxListRequest), req)
	if err != nil {
		return err
	}

	r, ok := resp.(*wire.BatchAddXshardTxListResponse)
	if !ok {
		return fmt.Errorf("unexpected response %T", resp)
	}

	if r.ErrorCode != 0 {
		return fmt.Errorf("BatchAddXshardTxList failed: %d", r.ErrorCode)
	}

	return nil
}

// Internal implementation

// handlePing performs the slave identity handshake. Peer metadata is recorded
// at most once by pingOnce.Do (see the peerID field); an empty inbound shard
// list publishes nothing and is rejected below. For inbound connections the
// handler then waits on inboundRegistered before returning PONG, making pool
// registration happen-before the PONG write (see the field comment).
func (x *XshardConn) handlePing(req any) (any, error) {
	ping := req.(*wire.PingRequest)

	x.pingOnce.Do(func() {
		if len(x.peerID) == 0 {
			if len(ping.FullShardIDList) == 0 {
				// An invalid inbound identity must not complete the handshake.
				return
			}

			x.peerID = append([]byte(nil), ping.ID...)
			x.peerFullShardIDList = append([]uint32(nil), ping.FullShardIDList...)
		}

		close(x.pingReceived)
	})

	if len(x.peerFullShardIDList) == 0 {
		return nil, fmt.Errorf("empty shard list from slave %s", ping.ID)
	}

	// Inbound-only barrier: the pool must finish registering this connection
	// before the PONG is written, so the dialing peer can rely on the reverse
	// dial being skipped by the time it sees PONG. Every PING (not only the
	// first) must pass it, since any of them may carry the response the peer
	// observes. Pool failure paths close the connection, which wakes this
	// select with an error instead of leaking the goroutine.
	if x.inboundRegistered != nil {
		select {
		case <-x.inboundRegistered:
		case <-x.WaitUntilClosed():
			return nil, fmt.Errorf("connection closed before inbound registration")
		}
	}

	return &wire.PongResponse{
		ID:              append([]byte(nil), x.localID...),
		FullShardIDList: append([]uint32(nil), x.localFullShardIDList...),
	}, nil
}

// handleAddXshardTxList delegates to the business handler.
func (x *XshardConn) handleAddXshardTxList(req any) (any, error) {
	return x.handler.AddXshardTxList(req.(*wire.AddXshardTxListRequest))
}

// handleBatchAddXshardTxList delegates to the business handler.
func (x *XshardConn) handleBatchAddXshardTxList(req any) (any, error) {
	return x.handler.BatchAddXshardTxList(req.(*wire.BatchAddXshardTxListRequest))
}

// waitUntilPingReceived blocks until the first PING, connection close, or
// handshake timeout, returning false on close or timeout.
func (x *XshardConn) waitUntilPingReceived() bool {
	timer := time.NewTimer(xshardHandshakeTimeout)
	defer timer.Stop()

	select {
	case <-x.pingReceived:
		return !x.IsClosed()
	case <-x.WaitUntilClosed():
		return false
	case <-timer.C:
		x.Close()
		return false
	}
}

// sendPing sends PING and returns the peer's id and shard list from PONG.
func (x *XshardConn) sendPing(ctx context.Context) ([]byte, []uint32, error) {
	req := &wire.PingRequest{
		ID:              x.localID,
		FullShardIDList: x.localFullShardIDList,
		RootTip:         types.NewRootBlockWithHeader(&types.RootBlockHeader{}),
	}
	resp, err := x.sendRPC(ctx, byte(wire.ClusterOpPing), req)
	if err != nil {
		return nil, nil, err
	}

	pong, ok := resp.(*wire.PongResponse)
	if !ok {
		return nil, nil, fmt.Errorf("unexpected ping response %T", resp)
	}
	if len(pong.ID) == 0 || len(pong.FullShardIDList) == 0 {
		return nil, nil, errors.New("invalid pong")
	}
	return pong.ID, pong.FullShardIDList, nil
}

// sendRPC serializes req and delegates to BaseConn.SendRPC.
func (x *XshardConn) sendRPC(ctx context.Context, opcode byte, req any) (any, error) {
	payload, err := serialize.SerializeToBytes(req)
	if err != nil {
		return nil, err
	}
	return x.BaseConn.SendRPC(ctx, opcode, payload)
}
