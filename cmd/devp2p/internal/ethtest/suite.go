// Copyright 2020 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package ethtest

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

// Suite represents a structure used to test a node's conformance
// to the eth protocol.
type Suite struct {
	Dest *enode.Node

	chain     *Chain
	fullChain *Chain
}

// NewSuite creates and returns a new eth-test suite that can
// be used to test the given node against the given blockchain
// data.
func NewSuite(dest *enode.Node, chainfile string, genesisfile string) (*Suite, error) {
	chain, err := loadChain(chainfile, genesisfile)
	if err != nil {
		return nil, err
	}
	return &Suite{
		Dest:      dest,
		chain:     chain.Shorten(1000),
		fullChain: chain,
	}, nil
}

func (s *Suite) AllEthTests() []utesting.Test {
	return []utesting.Test{
		// status
		{Name: "TestStatus", Fn: s.TestStatus},
		{Name: "TestStatus66", Fn: s.TestStatus66},
		// get block headers
		{Name: "TestGetBlockHeaders", Fn: s.TestGetBlockHeaders},
		{Name: "TestGetBlockHeaders66", Fn: s.TestGetBlockHeaders66},
		{Name: "TestSimultaneousRequests66", Fn: s.TestSimultaneousRequests66},
		{Name: "TestSameRequestID66", Fn: s.TestSameRequestID66},
		{Name: "TestZeroRequestID66", Fn: s.TestZeroRequestID66},
		// get block bodies
		{Name: "TestGetBlockBodies", Fn: s.TestGetBlockBodies},
		{Name: "TestGetBlockBodies66", Fn: s.TestGetBlockBodies66},
		// broadcast
		{Name: "TestBroadcast", Fn: s.TestBroadcast},
		{Name: "TestBroadcast66", Fn: s.TestBroadcast66},
		{Name: "TestLargeAnnounce", Fn: s.TestLargeAnnounce},
		{Name: "TestLargeAnnounce66", Fn: s.TestLargeAnnounce66},
		{Name: "TestOldAnnounce", Fn: s.TestOldAnnounce},
		{Name: "TestOldAnnounce66", Fn: s.TestOldAnnounce66},
		{Name: "TestBlockHashAnnounce", Fn: s.TestBlockHashAnnounce},
		{Name: "TestBlockHashAnnounce66", Fn: s.TestBlockHashAnnounce66},
		// malicious handshakes + status
		{Name: "TestMaliciousHandshake", Fn: s.TestMaliciousHandshake},
		{Name: "TestMaliciousStatus", Fn: s.TestMaliciousStatus},
		{Name: "TestMaliciousHandshake66", Fn: s.TestMaliciousHandshake66},
		{Name: "TestMaliciousStatus66", Fn: s.TestMaliciousStatus66},
		// test transactions
		{Name: "TestTransaction", Fn: s.TestTransaction},
		{Name: "TestTransaction66", Fn: s.TestTransaction66},
		{Name: "TestMaliciousTx", Fn: s.TestMaliciousTx},
		{Name: "TestMaliciousTx66", Fn: s.TestMaliciousTx66},
		{Name: "TestLargeTxRequest66", Fn: s.TestLargeTxRequest66},
		{Name: "TestNewPooledTxs66", Fn: s.TestNewPooledTxs66},
	}
}

func (s *Suite) EthTests() []utesting.Test {
	return []utesting.Test{
		{Name: "TestStatus", Fn: s.TestStatus},
		{Name: "TestGetBlockHeaders", Fn: s.TestGetBlockHeaders},
		{Name: "TestGetBlockBodies", Fn: s.TestGetBlockBodies},
		{Name: "TestBroadcast", Fn: s.TestBroadcast},
		{Name: "TestLargeAnnounce", Fn: s.TestLargeAnnounce},
		{Name: "TestOldAnnounce", Fn: s.TestOldAnnounce},
		{Name: "TestBlockHashAnnounce", Fn: s.TestBlockHashAnnounce},
		{Name: "TestMaliciousHandshake", Fn: s.TestMaliciousHandshake},
		{Name: "TestMaliciousStatus", Fn: s.TestMaliciousStatus},
		{Name: "TestTransaction", Fn: s.TestTransaction},
		{Name: "TestMaliciousTx", Fn: s.TestMaliciousTx},
	}
}

func (s *Suite) Eth66Tests() []utesting.Test {
	return []utesting.Test{
		// only proceed with eth66 test suite if node supports eth 66 protocol
		{Name: "TestStatus66", Fn: s.TestStatus66},
		{Name: "TestGetBlockHeaders66", Fn: s.TestGetBlockHeaders66},
		{Name: "TestSimultaneousRequests66", Fn: s.TestSimultaneousRequests66},
		{Name: "TestSameRequestID66", Fn: s.TestSameRequestID66},
		{Name: "TestZeroRequestID66", Fn: s.TestZeroRequestID66},
		{Name: "TestGetBlockBodies66", Fn: s.TestGetBlockBodies66},
		{Name: "TestBroadcast66", Fn: s.TestBroadcast66},
		{Name: "TestLargeAnnounce66", Fn: s.TestLargeAnnounce66},
		{Name: "TestOldAnnounce66", Fn: s.TestOldAnnounce66},
		{Name: "TestBlockHashAnnounce66", Fn: s.TestBlockHashAnnounce66},
		{Name: "TestMaliciousHandshake66", Fn: s.TestMaliciousHandshake66},
		{Name: "TestMaliciousStatus66", Fn: s.TestMaliciousStatus66},
		{Name: "TestTransaction66", Fn: s.TestTransaction66},
		{Name: "TestMaliciousTx66", Fn: s.TestMaliciousTx66},
		{Name: "TestLargeTxRequest66", Fn: s.TestLargeTxRequest66},
		{Name: "TestNewPooledTxs66", Fn: s.TestNewPooledTxs66},
	}
}

var (
	eth66 = true  // indicates whether suite should negotiate eth66 connection
	eth65 = false // indicates whether suite should negotiate eth65 connection or below.
)

// TestStatus attempts to connect to the given node and exchange
// a status message with it.
func (s *Suite) TestStatus(t *utesting.T) {
	conn, err := s.dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
}

// TestStatus66 attempts to connect to the given node and exchange
// a status message with it on the eth66 protocol.
func (s *Suite) TestStatus66(t *utesting.T) {
	conn, err := s.dial66()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
}

// TestGetBlockHeaders tests whether the given node can respond to
// a `GetBlockHeaders` request accurately.
func (s *Suite) TestGetBlockHeaders(t *utesting.T) {
	conn, err := s.dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("handshake(s) failed: %v", err)
	}
	// write request
	req := &GetBlockHeaders{
		Origin: eth.HashOrNumber{
			Hash: s.chain.blocks[1].Hash(),
		},
		Amount:  2,
		Skip:    1,
		Reverse: false,
	}
	headers, err := conn.headersRequest(req, s.chain, eth65, 0)
	if err != nil {
		t.Fatalf("GetBlockHeaders request failed: %v", err)
	}
	// check for correct headers
	expected, err := s.chain.GetHeaders(*req)
	if err != nil {
		t.Fatalf("failed to get headers for given request: %v", err)
	}
	if !headersMatch(expected, headers) {
		t.Fatalf("header mismatch: \nexpected %v \ngot %v", expected, headers)
	}
}

// TestGetBlockHeaders66 tests whether the given node can respond to
// an eth66 `GetBlockHeaders` request and that the response is accurate.
func (s *Suite) TestGetBlockHeaders66(t *utesting.T) {
	conn, err := s.dial66()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err = conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
	// write request
	req := &GetBlockHeaders{
		Origin: eth.HashOrNumber{
			Hash: s.chain.blocks[1].Hash(),
		},
		Amount:  2,
		Skip:    1,
		Reverse: false,
	}
	headers, err := conn.headersRequest(req, s.chain, eth66, 33)
	if err != nil {
		t.Fatalf("could not get block headers: %v", err)
	}
	// check for correct headers
	expected, err := s.chain.GetHeaders(*req)
	if err != nil {
		t.Fatalf("failed to get headers for given request: %v", err)
	}
	if !headersMatch(expected, headers) {
		t.Fatalf("header mismatch: \nexpected %v \ngot %v", expected, headers)
	}
}

// TestSimultaneousRequests66 sends two simultaneous `GetBlockHeader` requests from
// the same connection with different request IDs and checks to make sure the node
// responds with the correct headers per request.
func (s *Suite) TestSimultaneousRequests66(t *utesting.T) {
	// create a connection
	conn, err := s.dial66()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
	// create two requests
	req1 := &eth.GetBlockHeadersPacket66{
		RequestId: uint64(111),
		GetBlockHeadersPacket: &eth.GetBlockHeadersPacket{
			Origin: eth.HashOrNumber{
				Hash: s.chain.blocks[1].Hash(),
			},
			Amount:  2,
			Skip:    1,
			Reverse: false,
		},
	}
	req2 := &eth.GetBlockHeadersPacket66{
		RequestId: uint64(222),
		GetBlockHeadersPacket: &eth.GetBlockHeadersPacket{
			Origin: eth.HashOrNumber{
				Hash: s.chain.blocks[1].Hash(),
			},
			Amount:  4,
			Skip:    1,
			Reverse: false,
		},
	}
	// write the first request
	if err := conn.Write66(req1, GetBlockHeaders{}.Code()); err != nil {
		t.Fatalf("failed to write to connection: %v", err)
	}
	// write the second request
	if err := conn.Write66(req2, GetBlockHeaders{}.Code()); err != nil {
		t.Fatalf("failed to write to connection: %v", err)
	}
	// wait for responses
	msg := conn.waitForResponse(s.chain, timeout, req1.RequestId)
	headers1, ok := msg.(BlockHeaders)
	if !ok {
		t.Fatalf("unexpected %s", pretty.Sdump(msg))
	}
	msg = conn.waitForResponse(s.chain, timeout, req2.RequestId)
	headers2, ok := msg.(BlockHeaders)
	if !ok {
		t.Fatalf("unexpected %s", pretty.Sdump(msg))
	}
	// check received headers for accuracy
	expected1, err := s.chain.GetHeaders(GetBlockHeaders(*req1.GetBlockHeadersPacket))
	if err != nil {
		t.Fatalf("failed to get expected headers for request 1: %v", err)
	}
	expected2, err := s.chain.GetHeaders(GetBlockHeaders(*req2.GetBlockHeadersPacket))
	if err != nil {
		t.Fatalf("failed to get expected headers for request 2: %v", err)
	}
	if !headersMatch(expected1, headers1) {
		t.Fatalf("header mismatch: \nexpected %v \ngot %v", expected1, headers1)
	}
	if !headersMatch(expected2, headers2) {
		t.Fatalf("header mismatch: \nexpected %v \ngot %v", expected2, headers2)
	}
}

// TestSameRequestID66 sends two requests with the same request ID to a
// single node.
func (s *Suite) TestSameRequestID66(t *utesting.T) {
	conn, err := s.dial66()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
	// create requests
	reqID := uint64(1234)
	request1 := &eth.GetBlockHeadersPacket66{
		RequestId: reqID,
		GetBlockHeadersPacket: &eth.GetBlockHeadersPacket{
			Origin: eth.HashOrNumber{
				Number: 1,
			},
			Amount: 2,
		},
	}
	request2 := &eth.GetBlockHeadersPacket66{
		RequestId: reqID,
		GetBlockHeadersPacket: &eth.GetBlockHeadersPacket{
			Origin: eth.HashOrNumber{
				Number: 33,
			},
			Amount: 2,
		},
	}
	// write the requests
	if err = conn.Write66(request1, GetBlockHeaders{}.Code()); err != nil {
		t.Fatalf("failed to write to connection: %v", err)
	}
	if err = conn.Write66(request2, GetBlockHeaders{}.Code()); err != nil {
		t.Fatalf("failed to write to connection: %v", err)
	}
	// wait for responses
	msg := conn.waitForResponse(s.chain, timeout, reqID)
	headers1, ok := msg.(BlockHeaders)
	if !ok {
		t.Fatalf("unexpected %s", pretty.Sdump(msg))
	}
	msg = conn.waitForResponse(s.chain, timeout, reqID)
	headers2, ok := msg.(BlockHeaders)
	if !ok {
		t.Fatalf("unexpected %s", pretty.Sdump(msg))
	}
	// check if headers match
	expected1, err := s.chain.GetHeaders(GetBlockHeaders(*request1.GetBlockHeadersPacket))
	if err != nil {
		t.Fatalf("failed to get expected block headers: %v", err)
	}
	expected2, err := s.chain.GetHeaders(GetBlockHeaders(*request2.GetBlockHeadersPacket))
	if err != nil {
		t.Fatalf("failed to get expected block headers: %v", err)
	}
	if !headersMatch(expected1, headers1) {
		t.Fatalf("header mismatch: \nexpected %v \ngot %v", expected1, headers1)
	}
	if !headersMatch(expected2, headers2) {
		t.Fatalf("header mismatch: \nexpected %v \ngot %v", expected2, headers2)
	}
}

// TestZeroRequestID_66 checks that a message with a request ID of zero is still handled
// by the node.
func (s *Suite) TestZeroRequestID66(t *utesting.T) {
	conn, err := s.dial66()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
	req := &GetBlockHeaders{
		Origin: eth.HashOrNumber{
			Number: 0,
		},
		Amount: 2,
	}
	headers, err := conn.headersRequest(req, s.chain, eth66, 0)
	if err != nil {
		t.Fatalf("failed to get block headers: %v", err)
	}
	expected, err := s.chain.GetHeaders(*req)
	if err != nil {
		t.Fatalf("failed to get expected block headers: %v", err)
	}
	if !headersMatch(expected, headers) {
		t.Fatalf("header mismatch: \nexpected %v \ngot %v", expected, headers)
	}
}

// TestGetBlockBodies tests whether the given node can respond to
// a `GetBlockBodies` request and that the response is accurate.
func (s *Suite) TestGetBlockBodies(t *utesting.T) {
	conn, err := s.dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
	// create block bodies request
	req := &GetBlockBodies{
		s.chain.blocks[54].Hash(),
		s.chain.blocks[75].Hash(),
	}
	if err := conn.Write(req); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}
	// wait for response
	switch msg := conn.readAndServe(s.chain, timeout).(type) {
	case *BlockBodies:
		t.Logf("received %d block bodies", len(*msg))
		if len(*msg) != len(*req) {
			t.Fatalf("wrong bodies in response: expected %d bodies, "+
				"got %d", len(*req), len(*msg))
		}
	default:
		t.Fatalf("unexpected: %s", pretty.Sdump(msg))
	}
}

// TestGetBlockBodies66 tests whether the given node can respond to
// a `GetBlockBodies` request and that the response is accurate over
// the eth66 protocol.
func (s *Suite) TestGetBlockBodies66(t *utesting.T) {
	conn, err := s.dial66()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
	// create block bodies request
	req := &eth.GetBlockBodiesPacket66{
		RequestId: uint64(55),
		GetBlockBodiesPacket: eth.GetBlockBodiesPacket{
			s.chain.blocks[54].Hash(),
			s.chain.blocks[75].Hash(),
		},
	}
	if err := conn.Write66(req, GetBlockBodies{}.Code()); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}
	// wait for block bodies response
	msg := conn.waitForResponse(s.chain, timeout, req.RequestId)
	blockBodies, ok := msg.(BlockBodies)
	if !ok {
		t.Fatalf("unexpected: %s", pretty.Sdump(msg))
	}
	t.Logf("received %d block bodies", len(blockBodies))
	if len(blockBodies) != len(req.GetBlockBodiesPacket) {
		t.Fatalf("wrong bodies in response: expected %d bodies, "+
			"got %d", len(req.GetBlockBodiesPacket), len(blockBodies))
	}
}

// TestBroadcast tests whether a block announcement is correctly
// propagated to the given node's peer(s).
func (s *Suite) TestBroadcast(t *utesting.T) {
	if err := s.sendNextBlock(eth65); err != nil {
		t.Fatalf("block broadcast failed: %v", err)
	}
}

// TestBroadcast66 tests whether a block announcement is correctly
// propagated to the given node's peer(s) on the eth66 protocol.
func (s *Suite) TestBroadcast66(t *utesting.T) {
	if err := s.sendNextBlock(eth66); err != nil {
		t.Fatalf("block broadcast failed: %v", err)
	}
}

// TestLargeAnnounce tests the announcement mechanism with a large block.
func (s *Suite) TestLargeAnnounce(t *utesting.T) {
	nextBlock := len(s.chain.blocks)
	blocks := []*NewBlock{
		{
			Block: largeBlock(),
			TD:    s.fullChain.TotalDifficultyAt(nextBlock),
		},
		{
			Block: s.fullChain.blocks[nextBlock],
			TD:    largeNumber(2),
		},
		{
			Block: largeBlock(),
			TD:    largeNumber(2),
		},
	}

	for i, blockAnnouncement := range blocks {
		t.Logf("Testing malicious announcement: %v\n", i)
		conn, err := s.dial()
		if err != nil {
			t.Fatalf("dial failed: %v", err)
		}
		if err = conn.peer(s.chain, nil); err != nil {
			t.Fatalf("peering failed: %v", err)
		}
		if err = conn.Write(blockAnnouncement); err != nil {
			t.Fatalf("could not write to connection: %v", err)
		}
		// Invalid announcement, check that peer disconnected
		switch msg := conn.readAndServe(s.chain, time.Second*8).(type) {
		case *Disconnect:
		case *Error:
			break
		default:
			t.Fatalf("unexpected: %s wanted disconnect", pretty.Sdump(msg))
		}
		conn.Close()
	}
	// Test the last block as a valid block
	if err := s.sendNextBlock(eth65); err != nil {
		t.Fatalf("failed to broadcast next block: %v", err)
	}
}

// TestLargeAnnounce66 tests the announcement mechanism with a large
// block over the eth66 protocol.
func (s *Suite) TestLargeAnnounce66(t *utesting.T) {
	nextBlock := len(s.chain.blocks)
	blocks := []*NewBlock{
		{
			Block: largeBlock(),
			TD:    s.fullChain.TotalDifficultyAt(nextBlock),
		},
		{
			Block: s.fullChain.blocks[nextBlock],
			TD:    largeNumber(2),
		},
		{
			Block: largeBlock(),
			TD:    largeNumber(2),
		},
	}

	for i, blockAnnouncement := range blocks[0:3] {
		t.Logf("Testing malicious announcement: %v\n", i)
		conn, err := s.dial66()
		if err != nil {
			t.Fatalf("dial failed: %v", err)
		}
		if err := conn.peer(s.chain, nil); err != nil {
			t.Fatalf("peering failed: %v", err)
		}
		if err := conn.Write(blockAnnouncement); err != nil {
			t.Fatalf("could not write to connection: %v", err)
		}
		// Invalid announcement, check that peer disconnected
		switch msg := conn.readAndServe(s.chain, time.Second*8).(type) {
		case *Disconnect:
		case *Error:
			break
		default:
			t.Fatalf("unexpected: %s wanted disconnect", pretty.Sdump(msg))
		}
		conn.Close()
	}
	// Test the last block as a valid block
	if err := s.sendNextBlock(eth66); err != nil {
		t.Fatalf("failed to broadcast next block: %v", err)
	}
}

// TestOldAnnounce tests the announcement mechanism with an old block.
func (s *Suite) TestOldAnnounce(t *utesting.T) {
	if err := s.oldAnnounce(eth65); err != nil {
		t.Fatal(err)
	}
}

// TestOldAnnounce66 tests the announcement mechanism with an old block,
// over the eth66 protocol.
func (s *Suite) TestOldAnnounce66(t *utesting.T) {
	if err := s.oldAnnounce(eth66); err != nil {
		t.Fatal(err)
	}
}

// TestBlockHashAnnounce sends a new block hash announcement and expects
// the node to perform a `GetBlockHeaders` request.
func (s *Suite) TestBlockHashAnnounce(t *utesting.T) {
	if err := s.hashAnnounce(eth65); err != nil {
		t.Fatalf("block hash announcement failed: %v", err)
	}
}

// TestBlockHashAnnounce66 sends a new block hash announcement and expects
// the node to perform a `GetBlockHeaders` request.
func (s *Suite) TestBlockHashAnnounce66(t *utesting.T) {
	if err := s.hashAnnounce(eth66); err != nil {
		t.Fatalf("block hash announcement failed: %v", err)
	}
}

// TestMaliciousHandshake tries to send malicious data during the handshake.
func (s *Suite) TestMaliciousHandshake(t *utesting.T) {
	if err := s.maliciousHandshakes(t, eth65); err != nil {
		t.Fatal(err)
	}
}

// TestMaliciousHandshake66 tries to send malicious data during the handshake.
func (s *Suite) TestMaliciousHandshake66(t *utesting.T) {
	if err := s.maliciousHandshakes(t, eth66); err != nil {
		t.Fatal(err)
	}
}

// TestMaliciousStatus sends a status package with a large total difficulty.
func (s *Suite) TestMaliciousStatus(t *utesting.T) {
	conn, err := s.dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	if err := s.maliciousStatus(conn); err != nil {
		t.Fatal(err)
	}
}

// TestMaliciousStatus66 sends a status package with a large total
// difficulty over the eth66 protocol.
func (s *Suite) TestMaliciousStatus66(t *utesting.T) {
	conn, err := s.dial66()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	if err := s.maliciousStatus(conn); err != nil {
		t.Fatal(err)
	}
}

// TestTransaction sends a valid transaction to the node and
// checks if the transaction gets propagated.
func (s *Suite) TestTransaction(t *utesting.T) {
	if err := s.sendSuccessfulTxs(t, eth65); err != nil {
		t.Fatal(err)
	}
}

// TestTransaction66 sends a valid transaction to the node and
// checks if the transaction gets propagated.
func (s *Suite) TestTransaction66(t *utesting.T) {
	if err := s.sendSuccessfulTxs(t, eth66); err != nil {
		t.Fatal(err)
	}
}

// TestMaliciousTx sends several invalid transactions and tests whether
// the node will propagate them.
func (s *Suite) TestMaliciousTx(t *utesting.T) {
	if err := s.sendMaliciousTxs(t, eth65); err != nil {
		t.Fatal(err)
	}
}

// TestMaliciousTx66 sends several invalid transactions and tests whether
// the node will propagate them.
func (s *Suite) TestMaliciousTx66(t *utesting.T) {
	if err := s.sendMaliciousTxs(t, eth66); err != nil {
		t.Fatal(err)
	}
}

// TestLargeTxRequest66 tests whether a node can fulfill a large GetPooledTransactions
// request.
func (s *Suite) TestLargeTxRequest66(t *utesting.T) {
	// send the next block to ensure the node is no longer syncing and
	// is able to accept txs
	if err := s.sendNextBlock(eth66); err != nil {
		t.Fatalf("failed to send next block: %v", err)
	}
	// send 2000 transactions to the node
	hashMap, txs, err := generateTxs(s, 2000)
	if err != nil {
		t.Fatalf("failed to generate transactions: %v", err)
	}
	if err = sendMultipleSuccessfulTxs(t, s, txs); err != nil {
		t.Fatalf("failed to send multiple txs: %v", err)
	}
	// set up connection to receive to ensure node is peered with the receiving connection
	// before tx request is sent
	conn, err := s.dial66()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err = conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
	// create and send pooled tx request
	hashes := make([]common.Hash, 0)
	for _, hash := range hashMap {
		hashes = append(hashes, hash)
	}
	getTxReq := &eth.GetPooledTransactionsPacket66{
		RequestId:                   1234,
		GetPooledTransactionsPacket: hashes,
	}
	if err = conn.Write66(getTxReq, GetPooledTransactions{}.Code()); err != nil {
		t.Fatalf("could not write to conn: %v", err)
	}
	// check that all received transactions match those that were sent to node
	switch msg := conn.waitForResponse(s.chain, timeout, getTxReq.RequestId).(type) {
	case PooledTransactions:
		for _, gotTx := range msg {
			if _, exists := hashMap[gotTx.Hash()]; !exists {
				t.Fatalf("unexpected tx received: %v", gotTx.Hash())
			}
		}
	default:
		t.Fatalf("unexpected %s", pretty.Sdump(msg))
	}
}

// TestNewPooledTxs_66 tests whether a node will do a GetPooledTransactions
// request upon receiving a NewPooledTransactionHashes announcement.
func (s *Suite) TestNewPooledTxs66(t *utesting.T) {
	// send the next block to ensure the node is no longer syncing and
	// is able to accept txs
	if err := s.sendNextBlock(eth66); err != nil {
		t.Fatalf("failed to send next block: %v", err)
	}

	// generate 50 txs
	hashMap, _, err := generateTxs(s, 50)
	if err != nil {
		t.Fatalf("failed to generate transactions: %v", err)
	}

	// create new pooled tx hashes announcement
	hashes := make([]common.Hash, 0)
	for _, hash := range hashMap {
		hashes = append(hashes, hash)
	}
	announce := NewPooledTransactionHashes(hashes)

	// send announcement
	conn, err := s.dial66()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err = conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
	if err = conn.Write(announce); err != nil {
		t.Fatalf("failed to write to connection: %v", err)
	}

	// wait for GetPooledTxs request
	for {
		_, msg := conn.readAndServe66(s.chain, timeout)
		switch msg := msg.(type) {
		case GetPooledTransactions:
			if len(msg) != len(hashes) {
				t.Fatalf("unexpected number of txs requested: wanted %d, got %d", len(hashes), len(msg))
			}
			return
		// ignore propagated txs from previous tests
		case *NewPooledTransactionHashes:
			continue
		// ignore block announcements from previous tests
		case *NewBlockHashes:
			continue
		case *NewBlock:
			continue
		default:
			t.Fatalf("unexpected %s", pretty.Sdump(msg))
		}
	}
}
