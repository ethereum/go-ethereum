// Copyright 2026-2027, QuarkChain.

package shardconn

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/qkc/conn"
	"github.com/ethereum/go-ethereum/qkc/wire"
)

// XshardHandler serves inbound xshard requests, implemented by the business
// layer. Implementations must be safe for concurrent calls.
//
// A returned error signals a connection-level failure and closes the
// connection via BaseConn; business failures must be encoded in the response
// ErrorCode instead.
type XshardHandler interface {
	AddXshardTxList(req *wire.AddXshardTxListRequest) (*wire.AddXshardTxListResponse, error)
}

// XshardConn is a direct TCP connection between shard nodes.
type XshardConn struct {
	base *conn.BaseConn

	handler XshardHandler

	localID uint32
	// peerID is the remote shard node's ChainID.
	peerID uint32

	outbound     bool
	pingReceived chan struct{} // closed after the first PING
	pingOnce     sync.Once     // records only the first PING

	// inboundRegistered is closed by the pool after it indexes an inbound
	// connection. The PING handler waits for it before returning PONG. It is
	// nil for outbound connections, which are indexed after receiving PONG.
	inboundRegistered chan struct{}
}

// newXshardConn creates a connection to another shard node. For outbound
// connections peerID is the expected ChainID; for inbound connections the
// remote ChainID is learned from the first PING.
func newXshardConn(nc net.Conn, maxPayloadSize uint32, localID, peerID uint32, outbound bool, handler XshardHandler, logger log.Logger) (*XshardConn, error) {
	if handler == nil {
		return nil, errors.New("xshard handler must not be nil")
	}
	if maxPayloadSize == 0 {
		maxPayloadSize = wire.MaxFramePayloadLen
	}
	xc := &XshardConn{handler: handler, localID: localID, peerID: peerID, outbound: outbound, pingReceived: make(chan struct{})}
	xc.base = conn.NewBaseConn(conn.Config{
		Transport: conn.NewTCPTransport(
			nc,
			func(r io.Reader) (*wire.ClusterFrame, error) {
				return wire.ReadClusterFrame(r, maxPayloadSize)
			},
			wire.WriteClusterFrame,
		),
		Serializers: map[byte]*conn.OpSerializer{
			byte(wire.ClusterOpPing):                   conn.OpSerializerFor[wire.PingRequest, wire.PongResponse](byte(wire.ClusterOpPong)),
			byte(wire.ClusterOpAddXshardTxListRequest): conn.OpSerializerFor[wire.AddXshardTxListRequest, wire.AddXshardTxListResponse](byte(wire.ClusterOpAddXshardTxListResponse)),
		},
		Handlers: map[byte]conn.TypedHandler{
			byte(wire.ClusterOpPing):                   xc.handlePing,
			byte(wire.ClusterOpAddXshardTxListRequest): xc.handleAddXshardTxList,
		},
		Logger: logger,
	})
	return xc, nil
}

// Public API

// RemoteChainID returns the chain ID established by the handshake.
func (x *XshardConn) RemoteChainID() uint32 { return x.peerID }

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

// Internal implementation

// handlePing records the remote ChainID for inbound connections and signals
// the pool. Inbound PONGs are sent after the pool registers the connection.
func (x *XshardConn) handlePing(req any) (any, error) {
	ping := req.(*wire.PingRequest)

	x.pingOnce.Do(func() {
		if !x.outbound {
			x.peerID = ping.ChainID
		}
		close(x.pingReceived)
	})

	// Inbound-only barrier: the pool must finish registering this connection
	// before the PONG is written, so the dialing peer can rely on the reverse
	// dial being skipped by the time it sees PONG. Every PING (not only the
	// first) must pass it, since any of them may carry the response the peer
	// observes. Pool failure paths close the connection, which wakes this
	// select with an error instead of leaking the goroutine.
	if x.inboundRegistered != nil {
		select {
		case <-x.inboundRegistered:
		case <-x.base.WaitUntilClosed():
			return nil, fmt.Errorf("connection closed before inbound registration")
		}
	}

	return &wire.PongResponse{ChainID: x.localID}, nil
}

// handleAddXshardTxList delegates to the business handler.
func (x *XshardConn) handleAddXshardTxList(req any) (any, error) {
	return x.handler.AddXshardTxList(req.(*wire.AddXshardTxListRequest))
}

// waitUntilPingReceived blocks until the first PING, connection close, or
// handshake timeout, returning false on close or timeout.
func (x *XshardConn) waitUntilPingReceived() bool {
	timer := time.NewTimer(xshardHandshakeTimeout)
	defer timer.Stop()

	select {
	case <-x.pingReceived:
		return !x.base.IsClosed()
	case <-x.base.WaitUntilClosed():
		return false
	case <-timer.C:
		x.base.Close()
		return false
	}
}

// sendPing sends PING and returns the peer's chain ID.
func (x *XshardConn) sendPing(ctx context.Context) (uint32, error) {
	resp, err := x.sendRPC(ctx, byte(wire.ClusterOpPing), &wire.PingRequest{ChainID: x.localID})
	if err != nil {
		return 0, err
	}
	pong, ok := resp.(*wire.PongResponse)
	if !ok {
		return 0, fmt.Errorf("unexpected ping response %T", resp)
	}
	return pong.ChainID, nil
}

func (x *XshardConn) sendRPC(ctx context.Context, opcode byte, req wire.SSZMessage) (any, error) {
	payload, err := req.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return x.base.SendRPC(ctx, opcode, payload)
}
