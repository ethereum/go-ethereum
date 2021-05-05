// Copyright 2021 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/p2p"
)

// Is_66 checks if the node supports the eth66 protocol version,
// and if not, exists the test suite
func (s *Suite) Is_66(t *utesting.T) {
	conn := s.dial66(t)
	conn.handshake(t)
	if conn.negotiatedProtoVersion < 66 {
		t.Fail()
	}
}

// TestStatus_66 attempts to connect to the given node and exchange
// a status message with it on the eth66 protocol, and then check to
// make sure the chain head is correct.
func (s *Suite) TestStatus_66(t *utesting.T) {
	conn := s.dial66(t)
	defer conn.Close()
	// get protoHandshake
	conn.handshake(t)
	// get status
	switch msg := conn.statusExchange66(t, s.chain).(type) {
	case *Status:
		status := *msg
		if status.ProtocolVersion != uint32(66) {
			t.Fatalf("mismatch in version: wanted 66, got %d", status.ProtocolVersion)
		}
		t.Logf("got status message: %s", pretty.Sdump(msg))
	default:
		t.Fatalf("unexpected: %s", pretty.Sdump(msg))
	}
}

// TestGetBlockHeaders_66 tests whether the given node can respond to
// an eth66 `GetBlockHeaders` request and that the response is accurate.
func (s *Suite) TestGetBlockHeaders_66(t *utesting.T) {
	conn := s.setupConnection66(t)
	defer conn.Close()
	// get block headers
	req := &eth.GetBlockHeadersPacket66{
		RequestId: 3,
		GetBlockHeadersPacket: &eth.GetBlockHeadersPacket{
			Origin: eth.HashOrNumber{
				Hash: s.chain.blocks[1].Hash(),
			},
			Amount:  2,
			Skip:    1,
			Reverse: false,
		},
	}
	// write message
	headers, err := s.getBlockHeaders66(conn, req, req.RequestId)
	if err != nil {
		t.Fatalf("could not get block headers: %v", err)
	}
	// check for correct headers
	if !headersMatch(t, s.chain, headers) {
		t.Fatal("received wrong header(s)")
	}
}

// TestSimultaneousRequests_66 sends two simultaneous `GetBlockHeader` requests
// with different request IDs and checks to make sure the node responds with the correct
// headers per request.
func (s *Suite) TestSimultaneousRequests_66(t *utesting.T) {
	// create two connections
	conn := s.setupConnection66(t)
	defer conn.Close()
	// create two requests
	req1 := &eth.GetBlockHeadersPacket66{
		RequestId: 111,
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
		RequestId: 222,
		GetBlockHeadersPacket: &eth.GetBlockHeadersPacket{
			Origin: eth.HashOrNumber{
				Hash: s.chain.blocks[1].Hash(),
			},
			Amount:  4,
			Skip:    1,
			Reverse: false,
		},
	}
	// write first request
	if err := conn.write66(req1, GetBlockHeaders{}.Code()); err != nil {
		t.Fatalf("failed to write to connection: %v", err)
	}
	// write second request
	if err := conn.write66(req2, GetBlockHeaders{}.Code()); err != nil {
		t.Fatalf("failed to write to connection: %v", err)
	}
	// wait for responses
	headers1, err := s.waitForBlockHeadersResponse66(conn, req1.RequestId)
	if err != nil {
		t.Fatalf("error while waiting for block headers: %v", err)
	}
	headers2, err := s.waitForBlockHeadersResponse66(conn, req2.RequestId)
	if err != nil {
		t.Fatalf("error while waiting for block headers: %v", err)
	}
	// check headers of both responses
	if !headersMatch(t, s.chain, headers1) {
		t.Fatalf("wrong header(s) in response to req1: got %v", headers1)
	}
	if !headersMatch(t, s.chain, headers2) {
		t.Fatalf("wrong header(s) in response to req2: got %v", headers2)
	}
}

// TestBroadcast_66 tests whether a block announcement is correctly
// propagated to the given node's peer(s) on the eth66 protocol.
func (s *Suite) TestBroadcast_66(t *utesting.T) {
	s.sendNextBlock66(t)
}

// TestGetBlockBodies_66 tests whether the given node can respond to
// a `GetBlockBodies` request and that the response is accurate over
// the eth66 protocol.
func (s *Suite) TestGetBlockBodies_66(t *utesting.T) {
	conn := s.setupConnection66(t)
	defer conn.Close()
	// create block bodies request
	id := uint64(55)
	req := &eth.GetBlockBodiesPacket66{
		RequestId: id,
		GetBlockBodiesPacket: eth.GetBlockBodiesPacket{
			s.chain.blocks[54].Hash(),
			s.chain.blocks[75].Hash(),
		},
	}
	if err := conn.write66(req, GetBlockBodies{}.Code()); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}

	reqID, msg := conn.readAndServe66(s.chain, timeout)
	switch msg := msg.(type) {
	case BlockBodies:
		if reqID != req.RequestId {
			t.Fatalf("request ID mismatch: wanted %d, got %d", req.RequestId, reqID)
		}
		t.Logf("received %d block bodies", len(msg))
	default:
		t.Fatalf("unexpected: %s", pretty.Sdump(msg))
	}
}

// TestLargeAnnounce_66 tests the announcement mechanism with a large block.
func (s *Suite) TestLargeAnnounce_66(t *utesting.T) {
	nextBlock := len(s.chain.blocks)
	blocks := []*NewBlock{
		{
			Block: largeBlock(),
			TD:    s.fullChain.TD(nextBlock + 1),
		},
		{
			Block: s.fullChain.blocks[nextBlock],
			TD:    largeNumber(2),
		},
		{
			Block: largeBlock(),
			TD:    largeNumber(2),
		},
		{
			Block: s.fullChain.blocks[nextBlock],
			TD:    s.fullChain.TD(nextBlock + 1),
		},
	}

	for i, blockAnnouncement := range blocks[0:3] {
		t.Logf("Testing malicious announcement: %v\n", i)
		sendConn := s.setupConnection66(t)
		if err := sendConn.Write(blockAnnouncement); err != nil {
			t.Fatalf("could not write to connection: %v", err)
		}
		// Invalid announcement, check that peer disconnected
		switch msg := sendConn.ReadAndServe(s.chain, time.Second*8).(type) {
		case *Disconnect:
		case *Error:
			break
		default:
			t.Fatalf("unexpected: %s wanted disconnect", pretty.Sdump(msg))
		}
		sendConn.Close()
	}
	// Test the last block as a valid block
	s.sendNextBlock66(t)
}

func (s *Suite) TestOldAnnounce_66(t *utesting.T) {
	sendConn, recvConn := s.setupConnection66(t), s.setupConnection66(t)
	defer sendConn.Close()
	defer recvConn.Close()

	s.oldAnnounce(t, sendConn, recvConn)
}

// TestMaliciousHandshake_66 tries to send malicious data during the handshake.
func (s *Suite) TestMaliciousHandshake_66(t *utesting.T) {
	conn := s.dial66(t)
	defer conn.Close()
	// write hello to client
	pub0 := crypto.FromECDSAPub(&conn.ourKey.PublicKey)[1:]
	handshakes := []*Hello{
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: largeString(2), Version: 66},
			},
			ID: pub0,
		},
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: "eth", Version: 64},
				{Name: "eth", Version: 65},
				{Name: "eth", Version: 66},
			},
			ID: append(pub0, byte(0)),
		},
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: "eth", Version: 64},
				{Name: "eth", Version: 65},
				{Name: "eth", Version: 66},
			},
			ID: append(pub0, pub0...),
		},
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: "eth", Version: 64},
				{Name: "eth", Version: 65},
				{Name: "eth", Version: 66},
			},
			ID: largeBuffer(2),
		},
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: largeString(2), Version: 66},
			},
			ID: largeBuffer(2),
		},
	}
	for i, handshake := range handshakes {
		t.Logf("Testing malicious handshake %v\n", i)
		// Init the handshake
		if err := conn.Write(handshake); err != nil {
			t.Fatalf("could not write to connection: %v", err)
		}
		// check that the peer disconnected
		timeout := 20 * time.Second
		// Discard one hello
		for i := 0; i < 2; i++ {
			switch msg := conn.ReadAndServe(s.chain, timeout).(type) {
			case *Disconnect:
			case *Error:
			case *Hello:
				// Hello's are sent concurrently, so ignore them
				continue
			default:
				t.Fatalf("unexpected: %s", pretty.Sdump(msg))
			}
		}
		// Dial for the next round
		conn = s.dial66(t)
	}
}

// TestMaliciousStatus_66 sends a status package with a large total difficulty.
func (s *Suite) TestMaliciousStatus_66(t *utesting.T) {
	conn := s.dial66(t)
	defer conn.Close()
	// get protoHandshake
	conn.handshake(t)
	status := &Status{
		ProtocolVersion: uint32(66),
		NetworkID:       s.chain.chainConfig.ChainID.Uint64(),
		TD:              largeNumber(2),
		Head:            s.chain.blocks[s.chain.Len()-1].Hash(),
		Genesis:         s.chain.blocks[0].Hash(),
		ForkID:          s.chain.ForkID(),
	}
	// get status
	switch msg := conn.statusExchange(t, s.chain, status).(type) {
	case *Status:
		t.Logf("%+v\n", msg)
	default:
		t.Fatalf("expected status, got: %#v ", msg)
	}
	// wait for disconnect
	switch msg := conn.ReadAndServe(s.chain, timeout).(type) {
	case *Disconnect:
	case *Error:
		return
	default:
		t.Fatalf("expected disconnect, got: %s", pretty.Sdump(msg))
	}
}

func (s *Suite) TestTransaction_66(t *utesting.T) {
	tests := []*types.Transaction{
		getNextTxFromChain(t, s),
		unknownTx(t, s),
	}
	for i, tx := range tests {
		t.Logf("Testing tx propagation: %v\n", i)
		sendSuccessfulTx66(t, s, tx)
	}
}

func (s *Suite) TestMaliciousTx_66(t *utesting.T) {
	badTxs := []*types.Transaction{
		getOldTxFromChain(t, s),
		invalidNonceTx(t, s),
		hugeAmount(t, s),
		hugeGasPrice(t, s),
		hugeData(t, s),
	}
	sendConn := s.setupConnection66(t)
	defer sendConn.Close()
	// set up receiving connection before sending txs to make sure
	// no announcements are missed
	recvConn := s.setupConnection66(t)
	defer recvConn.Close()

	for i, tx := range badTxs {
		t.Logf("Testing malicious tx propagation: %v\n", i)
		if err := sendConn.Write(&Transactions{tx}); err != nil {
			t.Fatalf("could not write to connection: %v", err)
		}

	}
	// check to make sure bad txs aren't propagated
	waitForTxPropagation(t, s, badTxs, recvConn)
}

// TestZeroRequestID_66 checks that a request ID of zero is still handled
// by the node.
func (s *Suite) TestZeroRequestID_66(t *utesting.T) {
	conn := s.setupConnection66(t)
	defer conn.Close()

	req := &eth.GetBlockHeadersPacket66{
		RequestId: 0,
		GetBlockHeadersPacket: &eth.GetBlockHeadersPacket{
			Origin: eth.HashOrNumber{
				Number: 0,
			},
			Amount: 2,
		},
	}
	headers, err := s.getBlockHeaders66(conn, req, req.RequestId)
	if err != nil {
		t.Fatalf("could not get block headers: %v", err)
	}
	if !headersMatch(t, s.chain, headers) {
		t.Fatal("received wrong header(s)")
	}
}

// TestSameRequestID_66 sends two requests with the same request ID
// concurrently to a single node.
func (s *Suite) TestSameRequestID_66(t *utesting.T) {
	conn := s.setupConnection66(t)
	// create two requests with the same request ID
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
	// write the first request
	err := conn.write66(request1, GetBlockHeaders{}.Code())
	if err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}
	// perform second request
	headers2, err := s.getBlockHeaders66(conn, request2, reqID)
	if err != nil {
		t.Fatalf("could not get block headers: %v", err)
		return
	}
	// wait for response to first request
	headers1, err := s.waitForBlockHeadersResponse66(conn, reqID)
	if err != nil {
		t.Fatalf("could not get BlockHeaders response: %v", err)
	}
	// check if headers match
	if !headersMatch(t, s.chain, headers1) || !headersMatch(t, s.chain, headers2) {
		t.Fatal("received wrong header(s)")
	}
}

// TestLargeTxRequest_66 tests whether a node can fulfill a large GetPooledTransactions
// request.
func (s *Suite) TestLargeTxRequest_66(t *utesting.T) {
	// send the next block to ensure the node is no longer syncing and is able to accept
	// txs
	s.sendNextBlock66(t)
	// send 2000 transactions to the node
	hashMap, txs := generateTxs(t, s, 2000)
	sendConn := s.setupConnection66(t)
	defer sendConn.Close()

	sendMultipleSuccessfulTxs(t, s, sendConn, txs)
	// set up connection to receive to ensure node is peered with the receiving connection
	// before tx request is sent
	recvConn := s.setupConnection66(t)
	defer recvConn.Close()
	// create and send pooled tx request
	hashes := make([]common.Hash, 0)
	for _, hash := range hashMap {
		hashes = append(hashes, hash)
	}
	getTxReq := &eth.GetPooledTransactionsPacket66{
		RequestId:                   1234,
		GetPooledTransactionsPacket: hashes,
	}
	if err := recvConn.write66(getTxReq, GetPooledTransactions{}.Code()); err != nil {
		t.Fatalf("could not write to conn: %v", err)
	}
	// check that all received transactions match those that were sent to node
	switch msg := recvConn.waitForResponse(s.chain, timeout, getTxReq.RequestId).(type) {
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
func (s *Suite) TestNewPooledTxs_66(t *utesting.T) {
	// send the next block to ensure the node is no longer syncing and is able to accept
	// txs
	s.sendNextBlock66(t)
	// generate 50 txs
	hashMap, _ := generateTxs(t, s, 50)
	// create new pooled tx hashes announcement
	hashes := make([]common.Hash, 0)
	for _, hash := range hashMap {
		hashes = append(hashes, hash)
	}
	announce := NewPooledTransactionHashes(hashes)
	// send announcement
	conn := s.setupConnection66(t)
	defer conn.Close()
	if err := conn.Write(announce); err != nil {
		t.Fatalf("could not write to connection: %v", err)
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
		case *NewPooledTransactionHashes, *NewBlock, *NewBlockHashes:
			// ignore propagated txs and blocks from old tests
			continue
		default:
			t.Fatalf("unexpected %s", pretty.Sdump(msg))
		}
	}
}
