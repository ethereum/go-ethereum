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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/qkc/comm"
	"github.com/ethereum/go-ethereum/qkc/conn"
	qkctypes "github.com/ethereum/go-ethereum/qkc/types"
	"github.com/ethereum/go-ethereum/qkc/wire"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// MasterRequestHandler serves requests received from the master. Calls may run
// concurrently. Return *comm.RemoteError for business rejections; other errors
// indicate handler failures and close the connection.
type MasterRequestHandler interface {
	CreateShards(rootTip *qkctypes.RootBlock) error
	AddRootBlock(root *qkctypes.RootBlock) error
	GetRootChainStakes(address common.Address, shardBlockHash common.Hash) (*comm.RootChainStakesResult, error)
	GetUnconfirmedHeaders() ([]*types.Header, error)
}

// SlaveDialer establishes outbound connections to advertised slaves.
type SlaveDialer interface {
	DialToSlave(ctx context.Context, info wire.SlaveInfo) error
}

// MasterConnConfig configures the slave's connection to its master.
type MasterConnConfig struct {
	Conn           net.Conn
	LocalChainID   *big.Int
	MaxPayloadSize uint32 // Zero uses the protocol maximum.
	Handler        MasterRequestHandler
	SlaveDialer    SlaveDialer
	ShutdownSlave  func()
	Logger         log.Logger
}

// MasterConn handles master requests and sends shard block notifications.
// Any close, including the slave's own Close during shutdown, invokes
// ShutdownSlave once on a separate goroutine, so it must be idempotent.
type MasterConn struct {
	*conn.BaseConn

	localChainID *big.Int

	handler     MasterRequestHandler
	slaveDialer SlaveDialer

	shutdownSlave func()
	shutdownOnce  sync.Once
}

// NewMasterConn wraps an accepted master connection; the caller starts it with Start.
func NewMasterConn(cfg MasterConnConfig) (*MasterConn, error) {
	if cfg.Conn == nil {
		return nil, errors.New("master connection must not be nil")
	}
	if cfg.Handler == nil {
		return nil, errors.New("master handler must not be nil")
	}
	if cfg.SlaveDialer == nil {
		return nil, errors.New("slave dialer must not be nil")
	}
	if cfg.ShutdownSlave == nil {
		return nil, errors.New("master connection shutdown callback must not be nil")
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

	mc := &MasterConn{
		handler:       cfg.Handler,
		slaveDialer:   cfg.SlaveDialer,
		shutdownSlave: cfg.ShutdownSlave,
		localChainID:  new(big.Int).Set(cfg.LocalChainID),
	}

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
			byte(wire.ClusterOpAddShardBlockHeaderRequest): conn.OpSerializerFor[wire.AddShardBlockHeaderRequest, wire.AddShardBlockHeaderResponse](byte(wire.ClusterOpAddShardBlockHeaderResponse)),

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
	go mc.watchConnection()
	return mc, nil
}

// watchConnection shuts down the owning slave when its master connection closes.
func (mc *MasterConn) watchConnection() {
	<-mc.WaitUntilClosed()
	mc.shutdownOnce.Do(func() {
		mc.Logger().Error("master connection closed; shutting down slave")
		mc.shutdownSlave()
	})
}

// SendAddShardBlockHeader reports a shard header and its coinbase reward to
// the master. A nonzero business error code is returned as *comm.RemoteError.
func (mc *MasterConn) SendAddShardBlockHeader(ctx context.Context, header *types.Header, coinbaseAmount *uint256.Int) error {
	if header == nil {
		return errors.New("shard block header must not be nil")
	}
	headerData, err := rlp.EncodeToBytes(header)
	if err != nil {
		return fmt.Errorf("encode shard block header: %w", err)
	}
	request := &wire.AddShardBlockHeaderRequest{ShardBlockHeader: headerData}
	if coinbaseAmount != nil {
		encodedAmount, err := coinbaseAmount.MarshalSSZ()
		if err != nil {
			return fmt.Errorf("encode coinbase amount: %w", err)
		}
		copy(request.CoinbaseAmount[:], encodedAmount)
	}
	resp, err := mc.SendRPC(ctx, byte(wire.ClusterOpAddShardBlockHeaderRequest), request)
	if err != nil {
		return err
	}
	result, ok := resp.(*wire.AddShardBlockHeaderResponse)
	if !ok {
		return fmt.Errorf("unexpected add shard block header response %T", resp)
	}
	if result.ErrorCode != 0 {
		return &comm.RemoteError{Code: result.ErrorCode}
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
		root := new(qkctypes.RootBlock)
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
	request := req.(*wire.ConnectToSlavesRequest)
	result := &wire.ConnectToSlavesResponse{ResultList: make([][]byte, len(request.SlaveInfoList))}
	var wg sync.WaitGroup
	for i, info := range request.SlaveInfoList {
		wg.Add(1)
		go func(i int, info wire.SlaveInfo) {
			defer wg.Done()
			if err := mc.slaveDialer.DialToSlave(context.Background(), info); err != nil {
				result.ResultList[i] = []byte(err.Error())
			}
		}(i, info)
	}
	wg.Wait()
	return result, nil
}

// ── business RPC handlers ──────────────────────────────────────────────────
// Business RPCs operate on runtime-owned state (mining, accounts,
// transactions, queries) and delegate to MasterRequestHandler.

func (mc *MasterConn) handleAddRootBlock(req wire.SSZMessage) (wire.SSZMessage, error) {
	request := req.(*wire.AddRootBlockRequest)
	root := new(qkctypes.RootBlock)
	if err := root.UnmarshalSSZ(request.RootBlock); err != nil {
		return nil, fmt.Errorf("decode root block: %w", err)
	}
	err := mc.handler.AddRootBlock(root)
	code, err := wire.ErrorCode(err)
	if err != nil {
		return nil, err
	}
	return &wire.AddRootBlockResponse{ErrorCode: code}, nil
}

func (mc *MasterConn) handleGetRootChainStakes(req wire.SSZMessage) (wire.SSZMessage, error) {
	request := req.(*wire.GetRootChainStakesRequest)
	result, err := mc.handler.GetRootChainStakes(request.Address, request.ShardBlockHash)
	code, err := wire.ErrorCode(err)
	if err != nil {
		return nil, err
	}
	response := &wire.GetRootChainStakesResponse{ErrorCode: code}
	if code != 0 {
		return response, nil
	}
	if result == nil || result.Stakes == nil {
		return nil, errors.New("root chain stakes handler returned no result")
	}
	response.Signer = result.Signer
	encoded, err := result.Stakes.MarshalSSZ()
	if err != nil {
		return nil, fmt.Errorf("encode root chain stakes: %w", err)
	}
	copy(response.Stakes[:], encoded)
	return response, nil
}

func (mc *MasterConn) handleGetUnconfirmedHeaders(req wire.SSZMessage) (wire.SSZMessage, error) {
	headers, err := mc.handler.GetUnconfirmedHeaders()
	code, err := wire.ErrorCode(err)
	if err != nil {
		return nil, err
	}
	response := &wire.GetUnconfirmedHeadersResponse{ErrorCode: code}
	if code != 0 {
		return response, nil
	}
	response.HeaderList = make([]wire.Data, 0, len(headers))
	for _, header := range headers {
		encoded, err := rlp.EncodeToBytes(header)
		if err != nil {
			return nil, fmt.Errorf("encode unconfirmed header: %w", err)
		}
		response.HeaderList = append(response.HeaderList, encoded)
	}
	return response, nil
}
