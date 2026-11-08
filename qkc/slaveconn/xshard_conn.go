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

// Package slaveconn implements slave-side cluster connections.
package slaveconn

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/qkc/conn"
	"github.com/ethereum/go-ethereum/qkc/wire"
)

// XshardHandler serves cross-shard requests received from other slaves,
// implemented by the business layer. Implementations must be safe for concurrent
// calls and return a non-nil response when the error is nil.
//
// Business failures must be encoded in the response ErrorCode with a nil error,
// so they are sent to the peer without closing the connection. A non-nil error
// signals a handler failure and closes the connection via BaseConn.
type XshardHandler interface {
	AddXshardTxList(req *wire.AddXshardTxListRequest) (*wire.AddXshardTxListResponse, error)
}

// XshardConn is a slave-to-slave TCP connection for cross-shard RPCs.
type XshardConn struct {
	*conn.BaseConn

	handler XshardHandler

	// Each slave hosts one chain, whose ChainID identifies it in the cluster.
	localChainID  *big.Int
	remoteChainID *big.Int

	outbound     bool
	pingReceived chan struct{} // closed after the first PING
	pingOnce     sync.Once     // records only the first PING

	// inboundRegistered is closed by the manager after it indexes an inbound
	// connection. The PING handler waits for it before returning PONG. It is
	// nil for outbound connections, which are indexed after receiving PONG.
	inboundRegistered chan struct{}
}

// newXshardConn creates a cross-shard connection to another slave. For outbound
// connections remoteChainID is that slave's expected chain ID; for inbound
// connections it is learned from the first PING.
func newXshardConn(nc net.Conn, maxPayloadSize uint32, localChainID, remoteChainID *big.Int, outbound bool, handler XshardHandler, logger log.Logger) (*XshardConn, error) {
	if handler == nil {
		return nil, errors.New("xshard handler must not be nil")
	}
	if outbound {
		remoteChainID = new(big.Int).Set(remoteChainID)
	}
	if maxPayloadSize == 0 {
		maxPayloadSize = wire.MaxFramePayloadLen
	}
	xc := &XshardConn{handler: handler, localChainID: new(big.Int).Set(localChainID), remoteChainID: remoteChainID, outbound: outbound, pingReceived: make(chan struct{})}
	xc.BaseConn = conn.NewBaseConn(conn.Config{
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
		// Connections are established only between trusted cluster slaves. We do
		// not gate business handlers on handshake completion because unvalidated
		// peers cannot reach this listener in the current deployment.
		Handlers: map[byte]conn.TypedHandler{
			byte(wire.ClusterOpPing):                   xc.handlePing,
			byte(wire.ClusterOpAddXshardTxListRequest): xc.handleAddXshardTxList,
		},
		Logger: logger,
	})
	return xc, nil
}

// Public API

// RemoteChainID returns the remote slave's chain ID established by the handshake.
func (x *XshardConn) RemoteChainID() *big.Int {
	if x.remoteChainID == nil {
		return nil
	}
	return new(big.Int).Set(x.remoteChainID)
}

// SendAddXshardTxList sends an AddXshardTxListRequest to the peer. A nonzero
// ErrorCode is returned as *wire.RemoteError; transport and context errors are
// returned unchanged.
func (x *XshardConn) SendAddXshardTxList(ctx context.Context, req *wire.AddXshardTxListRequest) error {
	resp, err := x.SendRPC(ctx, byte(wire.ClusterOpAddXshardTxListRequest), req)
	if err != nil {
		return err
	}

	r, ok := resp.(*wire.AddXshardTxListResponse)
	if !ok {
		return fmt.Errorf("unexpected response %T", resp)
	}
	if r.ErrorCode != 0 {
		return &wire.RemoteError{Code: r.ErrorCode}
	}

	return nil
}

// Internal implementation

// handlePing records the remote slave's chain ID for inbound connections and
// signals the manager. Inbound PONGs are sent after the manager registers the connection.
func (x *XshardConn) handlePing(req wire.SSZMessage) (wire.SSZMessage, error) {
	ping := req.(*wire.PingRequest)
	id := wire.ChainIDFromSSZ(ping.ChainID)
	if err := validateXshardChainID(id); err != nil {
		return nil, err
	}

	x.pingOnce.Do(func() {
		if !x.outbound {
			x.remoteChainID = id
		}
		close(x.pingReceived)
	})

	// Inbound-only barrier: the manager must finish registering this connection
	// before the PONG is written, so the dialing peer can rely on the reverse
	// dial being skipped by the time it sees PONG. Every PING (not only the
	// first) must pass it, since any of them may carry the response the peer
	// observes. Manager failure paths close the connection, which wakes this
	// select with an error instead of leaking the goroutine.
	if x.inboundRegistered != nil {
		select {
		case <-x.inboundRegistered:
		case <-x.WaitUntilClosed():
			return nil, fmt.Errorf("connection closed before inbound registration")
		}
	}

	chainID, err := wire.ChainIDToSSZ(x.localChainID)
	if err != nil {
		return nil, err
	}
	return &wire.PongResponse{ChainID: chainID}, nil
}

// handleAddXshardTxList delegates to the business handler.
func (x *XshardConn) handleAddXshardTxList(req wire.SSZMessage) (wire.SSZMessage, error) {
	return x.handler.AddXshardTxList(req.(*wire.AddXshardTxListRequest))
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

// sendPing sends PING and returns the remote slave's chain ID.
func (x *XshardConn) sendPing(ctx context.Context) (*big.Int, error) {
	chainID, err := wire.ChainIDToSSZ(x.localChainID)
	if err != nil {
		return nil, err
	}
	resp, err := x.SendRPC(ctx, byte(wire.ClusterOpPing), &wire.PingRequest{ChainID: chainID})
	if err != nil {
		return nil, err
	}
	pong, ok := resp.(*wire.PongResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected ping response %T", resp)
	}
	return wire.ChainIDFromSSZ(pong.ChainID), nil
}

func validateXshardChainID(id *big.Int) error {
	if id == nil || id.Sign() <= 0 || id.BitLen() > 256 {
		return fmt.Errorf("invalid xshard chain ID: %v", id)
	}
	return nil
}
