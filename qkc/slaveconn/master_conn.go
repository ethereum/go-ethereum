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

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/qkc/conn"
	"github.com/ethereum/go-ethereum/qkc/types"
	"github.com/ethereum/go-ethereum/qkc/wire"
)

// MasterHandler is implemented by the slave's composition layer. Calls may run
// concurrently. Business failures belong in response error codes; non-nil
// errors indicate handler failures and close the connection.
type MasterHandler interface {
	CreateShards(rootTip *types.RootBlock) error

	// ConnectToSlaves connects the advertised slaves through the node's connection manager.
	ConnectToSlaves(req *wire.ConnectToSlavesRequest) (*wire.ConnectToSlavesResponse, error)

	AddRootBlock(req *wire.AddRootBlockRequest) (*wire.AddRootBlockResponse, error)
	GetRootChainStakes(req *wire.GetRootChainStakesRequest) (*wire.GetRootChainStakesResponse, error)
	GetUnconfirmedHeaders(req *wire.GetUnconfirmedHeadersRequest) (*wire.GetUnconfirmedHeadersResponse, error)
}

// MasterConnConfig configures the slave's connection to its master.
type MasterConnConfig struct {
	Conn           net.Conn
	LocalChainID   *big.Int
	MaxPayloadSize uint32 // Zero uses the protocol maximum.
	Handler        MasterHandler
	Logger         log.Logger
}

// MasterConn handles master requests and sends shard block notifications.
// Closing the master connection must be observed by the owning slave service,
// which is responsible for shutting down the cluster.
type MasterConn struct {
	*conn.BaseConn

	handler      MasterHandler
	localChainID *big.Int
}

// NewMasterConn wraps an accepted master connection; the caller starts it with Start.
func NewMasterConn(cfg MasterConnConfig) (*MasterConn, error) {
	if cfg.Conn == nil {
		return nil, errors.New("master connection must not be nil")
	}
	if cfg.Handler == nil {
		return nil, errors.New("master handler must not be nil")
	}
	if cfg.LocalChainID == nil || cfg.LocalChainID.Sign() <= 0 || cfg.LocalChainID.BitLen() > 256 {
		return nil, fmt.Errorf("invalid slave chain ID: %v", cfg.LocalChainID)
	}
	maxPayloadSize := cfg.MaxPayloadSize
	if maxPayloadSize == 0 {
		maxPayloadSize = wire.MaxFramePayloadLen
	} else if maxPayloadSize > wire.MaxFramePayloadLen {
		return nil, fmt.Errorf("max payload size %d exceeds protocol limit %d", maxPayloadSize, wire.MaxFramePayloadLen)
	}

	mc := &MasterConn{handler: cfg.Handler, localChainID: new(big.Int).Set(cfg.LocalChainID)}

	// The forwarder runs on the reader goroutine and must not block it.
	forwarder := mc.routeFrame

	mc.BaseConn = conn.NewBaseConn(conn.Config{
		Transport: conn.NewTCPTransport(cfg.Conn, func(r io.Reader) (*wire.ClusterFrame, error) {
			return wire.ReadClusterFrameWithPeerID(r, maxPayloadSize)
		}, wire.WriteClusterFrameWithPeerID),
		Forwarder: forwarder,
		Serializers: map[byte]*conn.OpSerializer{
			// Communication layer: connection setup and slave topology.
			byte(wire.ClusterOpPing):                   conn.OpSerializerFor[wire.PingRequest, wire.PongResponse](byte(wire.ClusterOpPong)),
			byte(wire.ClusterOpConnectToSlavesRequest): conn.OpSerializerFor[wire.ConnectToSlavesRequest, wire.ConnectToSlavesResponse](byte(wire.ClusterOpConnectToSlavesResponse)),

			// Business layer: requests sent to the master.
			byte(wire.ClusterOpAddMinorBlockHeaderRequest): conn.OpSerializerFor[wire.AddMinorBlockHeaderRequest, wire.AddMinorBlockHeaderResponse](byte(wire.ClusterOpAddMinorBlockHeaderResponse)),

			// Business layer: requests received from the master.
			byte(wire.ClusterOpAddRootBlockRequest):          conn.OpSerializerFor[wire.AddRootBlockRequest, wire.AddRootBlockResponse](byte(wire.ClusterOpAddRootBlockResponse)),
			byte(wire.ClusterOpGetRootChainStakesRequest):    conn.OpSerializerFor[wire.GetRootChainStakesRequest, wire.GetRootChainStakesResponse](byte(wire.ClusterOpGetRootChainStakesResponse)),
			byte(wire.ClusterOpGetUnconfirmedHeadersRequest): conn.OpSerializerFor[wire.GetUnconfirmedHeadersRequest, wire.GetUnconfirmedHeadersResponse](byte(wire.ClusterOpGetUnconfirmedHeadersResponse)),
		},
		Handlers: map[byte]conn.TypedHandler{
			// Handlers process received requests only.
			// Communication layer.
			byte(wire.ClusterOpPing):                   mc.handlePing,
			byte(wire.ClusterOpConnectToSlavesRequest): mc.handleConnectToSlaves,

			// Business layer.
			byte(wire.ClusterOpAddRootBlockRequest):          mc.handleAddRootBlock,
			byte(wire.ClusterOpGetRootChainStakesRequest):    mc.handleGetRootChainStakes,
			byte(wire.ClusterOpGetUnconfirmedHeadersRequest): mc.handleGetUnconfirmedHeaders,
		}, Logger: cfg.Logger,
	})
	return mc, nil
}

// SendAddMinorBlockHeader reports a shard block to the master. A nonzero
// business error code is returned as *wire.RemoteError.
func (mc *MasterConn) SendAddMinorBlockHeader(ctx context.Context, req *wire.AddMinorBlockHeaderRequest) error {
	resp, err := mc.SendRPC(ctx, byte(wire.ClusterOpAddMinorBlockHeaderRequest), req)
	if err != nil {
		return err
	}
	result, ok := resp.(*wire.AddMinorBlockHeaderResponse)
	if !ok {
		return fmt.Errorf("unexpected add minor block header response %T", resp)
	}
	if result.ErrorCode != 0 {
		return &wire.RemoteError{Code: result.ErrorCode}
	}
	return nil
}

// ── Frame routing ───────────────────────────────────────────────────────

// routeFrame leaves master-local frames for normal dispatch and consumes peer
// frames without dispatching them until virtual peer connections are supported.
func (mc *MasterConn) routeFrame(frame *wire.ClusterFrame) bool {
	if frame.ClusterPeerID == 0 {
		return false
	}

	// TODO: Route peer frames to virtual connections when peerconn is implemented.
	// Consume them until then so they cannot reach master RPC dispatch.
	return true
}

// ── topology & shard activation handlers ───────────────────────────────────

// handlePing handles the master's PING.
func (mc *MasterConn) handlePing(req wire.SSZMessage) (wire.SSZMessage, error) {
	ping := req.(*wire.PingRequest)
	if ping.HasRootTip {
		root := new(types.RootBlock)
		if err := root.UnmarshalSSZ(ping.RootTip); err != nil {
			return nil, fmt.Errorf("decode ping root tip: %w", err)
		}
		if err := mc.handler.CreateShards(root); err != nil {
			return nil, err
		}
	}
	chainID, err := wire.ChainIDToSSZ(mc.localChainID)
	if err != nil {
		return nil, err
	}
	return &wire.PongResponse{ChainID: chainID}, nil
}

// handleConnectToSlaves connects to the slaves advertised by the master.
func (mc *MasterConn) handleConnectToSlaves(req wire.SSZMessage) (wire.SSZMessage, error) {
	return mc.handler.ConnectToSlaves(req.(*wire.ConnectToSlavesRequest))
}

// ── business RPC handlers ──────────────────────────────────────────────────
// Business RPCs operate on runtime-owned state (mining, accounts,
// transactions, queries) and delegate to MasterHandler.

func (mc *MasterConn) handleAddRootBlock(req wire.SSZMessage) (wire.SSZMessage, error) {
	return mc.handler.AddRootBlock(req.(*wire.AddRootBlockRequest))
}

func (mc *MasterConn) handleGetRootChainStakes(req wire.SSZMessage) (wire.SSZMessage, error) {
	return mc.handler.GetRootChainStakes(req.(*wire.GetRootChainStakesRequest))
}

func (mc *MasterConn) handleGetUnconfirmedHeaders(req wire.SSZMessage) (wire.SSZMessage, error) {
	return mc.handler.GetUnconfirmedHeaders(req.(*wire.GetUnconfirmedHeadersRequest))
}
