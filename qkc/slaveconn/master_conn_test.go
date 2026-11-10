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
	"io"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	coretypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/qkc/comm"
	"github.com/ethereum/go-ethereum/qkc/conn"
	"github.com/ethereum/go-ethereum/qkc/types"
	"github.com/ethereum/go-ethereum/qkc/wire"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

type masterTestHandler struct {
	roots  chan *types.RootBlock
	addErr error
	pool   *XshardConnManager
}

type testSlaveDialer struct{}

func (testSlaveDialer) DialToSlave(context.Context, wire.SlaveInfo) error { return nil }

func (h *masterTestHandler) CreateShards(root *types.RootBlock) error { h.roots <- root; return nil }
func (h *masterTestHandler) AddRootBlock(req *types.RootBlock) error {
	if h.addErr != nil {
		return h.addErr
	}
	return &comm.RemoteError{Code: 7}
}
func (h *masterTestHandler) GetRootChainStakes(common.Address, common.Hash) (*comm.RootChainStakesResult, error) {
	return nil, &comm.RemoteError{Code: 8}
}
func (h *masterTestHandler) GetUnconfirmedHeaders() ([]*coretypes.Header, error) {
	return []*coretypes.Header{{Difficulty: big.NewInt(1), Number: big.NewInt(1)}}, nil
}

func masterTestPair(t *testing.T, h *masterTestHandler, headerCode uint32, chainID *big.Int) (*MasterConn, *conn.BaseConn, *XshardConnManager) {
	t.Helper()
	pool, err := NewXshardConnManager(chainID, []*big.Int{chainID, big.NewInt(2)}, 0, &testXshardHandler{}, noopShutdown, nil)
	if err != nil {
		t.Fatal(err)
	}
	h.pool = pool
	a, b := net.Pipe()
	mc, err := NewMasterConn(MasterConnConfig{Conn: a, LocalChainID: chainID, Handler: h, SlaveDialer: pool, ShutdownSlave: noopShutdown})
	if err != nil {
		t.Fatal(err)
	}
	remote := conn.NewBaseConn(conn.Config{Transport: conn.NewTCPTransport(b, func(r io.Reader) (*wire.ClusterFrame, error) {
		return wire.ReadClusterFrameWithPeerID(r, wire.MaxFramePayloadLen)
	}, wire.WriteClusterFrameWithPeerID),
		Serializers: map[byte]*conn.OpSerializer{
			byte(wire.ClusterOpPing):                         conn.OpSerializerFor[wire.PingRequest, wire.PongResponse](byte(wire.ClusterOpPong)),
			byte(wire.ClusterOpConnectToSlavesRequest):       conn.OpSerializerFor[wire.ConnectToSlavesRequest, wire.ConnectToSlavesResponse](byte(wire.ClusterOpConnectToSlavesResponse)),
			byte(wire.ClusterOpAddRootBlockRequest):          conn.OpSerializerFor[wire.AddRootBlockRequest, wire.AddRootBlockResponse](byte(wire.ClusterOpAddRootBlockResponse)),
			byte(wire.ClusterOpGetRootChainStakesRequest):    conn.OpSerializerFor[wire.GetRootChainStakesRequest, wire.GetRootChainStakesResponse](byte(wire.ClusterOpGetRootChainStakesResponse)),
			byte(wire.ClusterOpGetUnconfirmedHeadersRequest): conn.OpSerializerFor[wire.GetUnconfirmedHeadersRequest, wire.GetUnconfirmedHeadersResponse](byte(wire.ClusterOpGetUnconfirmedHeadersResponse)),
			byte(wire.ClusterOpAddMinorBlockHeaderRequest):   conn.OpSerializerFor[wire.AddMinorBlockHeaderRequest, wire.AddMinorBlockHeaderResponse](byte(wire.ClusterOpAddMinorBlockHeaderResponse)),
		}, Handlers: map[byte]conn.TypedHandler{byte(wire.ClusterOpAddMinorBlockHeaderRequest): func(wire.SSZMessage) (wire.SSZMessage, error) {
			return &wire.AddMinorBlockHeaderResponse{ErrorCode: headerCode}, nil
		}}})
	mc.Start()
	remote.Start()
	t.Cleanup(func() { mc.Close(); remote.Close(); pool.Close() })
	return mc, remote, pool
}

func TestMasterConnRejectsInvalidChainIDs(t *testing.T) {
	for _, chainID := range []*big.Int{nil, new(big.Int), big.NewInt(-1), new(big.Int).Lsh(big.NewInt(1), 256)} {
		local, remote := net.Pipe()
		mc, err := NewMasterConn(MasterConnConfig{Conn: local, LocalChainID: chainID, Handler: &masterTestHandler{}, SlaveDialer: new(testSlaveDialer), ShutdownSlave: noopShutdown})
		local.Close()
		remote.Close()
		if err == nil {
			mc.Close()
			t.Fatalf("accepted invalid chain ID %v", chainID)
		}
	}
}

func TestMasterConnCloseShutsDownSlave(t *testing.T) {
	a, b := net.Pipe()
	defer b.Close()
	shutdown := make(chan struct{}, 2)
	mc, err := NewMasterConn(MasterConnConfig{
		Conn: a, LocalChainID: big.NewInt(1), Handler: &masterTestHandler{},
		SlaveDialer: new(testSlaveDialer), ShutdownSlave: func() { shutdown <- struct{}{} },
	})
	if err != nil {
		t.Fatal(err)
	}
	mc.Start()
	mc.Close()
	select {
	case <-shutdown:
	case <-time.After(5 * time.Second):
		t.Fatal("slave shutdown callback was not called")
	}
	// Closing an already closed connection must not notify shutdown again.
	mc.Close()
	select {
	case <-shutdown:
		t.Fatal("slave shutdown callback called more than once")
	case <-time.After(10 * time.Millisecond):
	}
}

func TestMasterConnRPC(t *testing.T) {
	h := &masterTestHandler{roots: make(chan *types.RootBlock, 1)}
	chainID := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 100), big.NewInt(258))
	encodedID, err := wire.ChainIDToSSZ(chainID)
	if err != nil {
		t.Fatal(err)
	}
	mc, remote, _ := masterTestPair(t, h, 9, chainID)
	chainID.SetInt64(0)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := remote.SendRPC(ctx, byte(wire.ClusterOpPing), &wire.PingRequest{})
	if err != nil || resp.(*wire.PongResponse).ChainID != encodedID {
		t.Fatalf("ping response %v error %v", resp, err)
	}
	root := types.NewRootBlockWithHeader(&types.RootBlockHeader{})
	encoded, err := root.MarshalSSZ()
	if err != nil {
		t.Fatal(err)
	}
	_, err = remote.SendRPC(ctx, byte(wire.ClusterOpPing), &wire.PingRequest{HasRootTip: true, RootTip: encoded})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-h.roots:
	case <-ctx.Done():
		t.Fatal("root tip not handled")
	}
	resp, err = remote.SendRPC(ctx, byte(wire.ClusterOpConnectToSlavesRequest), &wire.ConnectToSlavesRequest{SlaveInfoList: []wire.SlaveInfo{{ChainID: encodedID}, {ChainID: [wire.UInt256Length]byte{3}}}})
	if err != nil {
		t.Fatal(err)
	}
	results := resp.(*wire.ConnectToSlavesResponse).ResultList
	if len(results) != 2 || len(results[0]) != 0 || len(results[1]) == 0 {
		t.Fatalf("unexpected connection results: %q", results)
	}
	rootData, err := types.NewRootBlockWithHeader(&types.RootBlockHeader{}).MarshalSSZ()
	if err != nil {
		t.Fatal(err)
	}
	resp, err = remote.SendRPC(ctx, byte(wire.ClusterOpAddRootBlockRequest), &wire.AddRootBlockRequest{RootBlock: rootData})
	if err != nil || resp.(*wire.AddRootBlockResponse).ErrorCode != 7 {
		t.Fatalf("add root %v, %v", resp, err)
	}
	resp, err = remote.SendRPC(ctx, byte(wire.ClusterOpGetRootChainStakesRequest), &wire.GetRootChainStakesRequest{})
	if err != nil || resp.(*wire.GetRootChainStakesResponse).ErrorCode != 8 {
		t.Fatalf("stakes %v, %v", resp, err)
	}
	resp, err = remote.SendRPC(ctx, byte(wire.ClusterOpGetUnconfirmedHeadersRequest), &wire.GetUnconfirmedHeadersRequest{})
	if err != nil {
		t.Fatal(err)
	}
	headers := resp.(*wire.GetUnconfirmedHeadersResponse).HeaderList
	if len(headers) != 1 {
		t.Fatalf("unconfirmed headers %v, %v", resp, err)
	}
	var decodedHeader coretypes.Header
	if err := rlp.DecodeBytes(headers[0], &decodedHeader); err != nil || decodedHeader.Number.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("decode unconfirmed header: %v, %v", &decodedHeader, err)
	}
	err = mc.SendAddMinorBlockHeader(ctx, &coretypes.Header{Difficulty: big.NewInt(1), Number: big.NewInt(1)}, uint256.NewInt(0))
	var remoteErr *comm.RemoteError
	if !errors.As(err, &remoteErr) || remoteErr.Code != 9 {
		t.Fatalf("expected remote error 9, got %v", err)
	}
	if _, err = remote.SendRPC(ctx, byte(wire.ClusterOpPing), &wire.PingRequest{}); err != nil {
		t.Fatalf("connection closed after business error: %v", err)
	}
}

func TestMasterConnHandlerErrorClosesConnection(t *testing.T) {
	mc, remote, _ := masterTestPair(t, &masterTestHandler{addErr: errors.New("failure")}, 0, big.NewInt(1))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rootData, marshalErr := types.NewRootBlockWithHeader(&types.RootBlockHeader{}).MarshalSSZ()
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if _, err := remote.SendRPC(ctx, byte(wire.ClusterOpAddRootBlockRequest), &wire.AddRootBlockRequest{RootBlock: rootData}); err == nil {
		t.Fatal("expected handler error")
	}
	select {
	case <-mc.WaitUntilClosed():
	case <-ctx.Done():
		t.Fatal("connection did not close")
	}
}

func TestMasterConnIgnoresPeerFrames(t *testing.T) {
	a, b := net.Pipe()
	defer b.Close()
	mc, err := NewMasterConn(MasterConnConfig{Conn: a, LocalChainID: big.NewInt(1), Handler: &masterTestHandler{}, SlaveDialer: new(testSlaveDialer), ShutdownSlave: noopShutdown})
	if err != nil {
		t.Fatal(err)
	}
	mc.Start()
	defer mc.Close()
	if err = wire.WriteClusterFrameWithPeerID(b, &wire.ClusterFrame{ClusterPeerID: 1, Opcode: byte(wire.ClusterOpPing), RPCID: 1}); err != nil {
		t.Fatal(err)
	}
	if err := b.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	payload, err := (&wire.PingRequest{}).MarshalSSZ()
	if err != nil {
		t.Fatal(err)
	}
	if err := wire.WriteClusterFrameWithPeerID(b, &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPing), RPCID: 2, Payload: payload}); err != nil {
		t.Fatal(err)
	}
	response, err := wire.ReadClusterFrameWithPeerID(b, wire.MaxFramePayloadLen)
	if err != nil {
		t.Fatal(err)
	}
	if response.Opcode != byte(wire.ClusterOpPong) || response.RPCID != 2 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestMasterConnWritesPeerFrame(t *testing.T) {
	a, b := net.Pipe()
	defer b.Close()
	mc, err := NewMasterConn(MasterConnConfig{Conn: a, LocalChainID: big.NewInt(1), Handler: &masterTestHandler{}, SlaveDialer: new(testSlaveDialer), ShutdownSlave: noopShutdown})
	if err != nil {
		t.Fatal(err)
	}
	mc.Start()
	defer mc.Close()
	if err := b.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	writeErr := make(chan error, 1)
	go func() {
		writeErr <- mc.WriteFrame(&wire.ClusterFrame{ClusterPeerID: 42, Opcode: 3, RPCID: 7, Payload: []byte("peer")})
	}()
	frame, err := wire.ReadClusterFrameWithPeerID(b, wire.MaxFramePayloadLen)
	if err != nil {
		t.Fatal(err)
	}
	if frame.ClusterPeerID != 42 || frame.Opcode != 3 || frame.RPCID != 7 || string(frame.Payload) != "peer" {
		t.Fatalf("unexpected peer frame: %+v", frame)
	}
	if err := <-writeErr; err != nil {
		t.Fatal(err)
	}
}

func TestMasterConnConnectToSlaves(t *testing.T) {
	_, remote, pool := masterTestPair(t, &masterTestHandler{}, 0, big.NewInt(1))
	other, err := NewXshardConnManager(big.NewInt(2), []*big.Int{big.NewInt(1), big.NewInt(2)}, 0, &testXshardHandler{}, noopShutdown, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		if nc, err := listener.Accept(); err == nil {
			other.HandleInbound(nc)
		}
	}()
	addr := listener.Addr().(*net.TCPAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := remote.SendRPC(ctx, byte(wire.ClusterOpConnectToSlavesRequest), &wire.ConnectToSlavesRequest{SlaveInfoList: []wire.SlaveInfo{
		{ChainID: [wire.UInt256Length]byte{1}}, {ChainID: [wire.UInt256Length]byte{2}, Host: []byte("127.0.0.1"), Port: uint16(addr.Port)},
	}})
	if err != nil {
		t.Fatal(err)
	}
	results := resp.(*wire.ConnectToSlavesResponse).ResultList
	if len(results) != 2 || len(results[0]) != 0 || len(results[1]) != 0 {
		t.Fatalf("unexpected connection results: %q", results)
	}
	if pool.Lookup(big.NewInt(2)) == nil || other.Lookup(big.NewInt(1)) == nil {
		t.Fatal("both sides should have registered the connection")
	}
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("inbound handshake did not finish")
	}
}
