// Copyright 2020 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package ethtest

import (
	"context"
	"crypto/rand"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/holiman/uint256"
)

// Suite represents a structure used to test a node's conformance
// to the eth protocol.
type Suite struct {
	Dest   *enode.Node
	chain  *Chain
	engine *EngineClient
}

// NewSuite creates and returns a new eth-test suite that can
// be used to test the given node against the given blockchain
// data.
func NewSuite(dest *enode.Node, chainDir, engineURL, jwt string) (*Suite, error) {
	chain, err := NewChain(chainDir)
	if err != nil {
		return nil, err
	}
	engine, err := NewEngineClient(chainDir, engineURL, jwt)
	if err != nil {
		return nil, err
	}

	return &Suite{
		Dest:   dest,
		chain:  chain,
		engine: engine,
	}, nil
}

func (s *Suite) EthTests() []utesting.Test {
	return []utesting.Test{
		// status
		{Name: "Status", Fn: s.TestStatus},
		// get block headers
		{Name: "GetBlockHeaders", Fn: s.TestGetBlockHeaders},
		{Name: "GetNonexistentBlockHeaders", Fn: s.TestGetNonexistentBlockHeaders},
		{Name: "SimultaneousRequests", Fn: s.TestSimultaneousRequests},
		{Name: "SameRequestID", Fn: s.TestSameRequestID},
		{Name: "ZeroRequestID", Fn: s.TestZeroRequestID},
		// get block bodies
		{Name: "GetBlockBodies", Fn: s.TestGetBlockBodies},
		// // malicious handshakes + status
		{Name: "MaliciousHandshake", Fn: s.TestMaliciousHandshake},
		// test transactions
		{Name: "LargeTxRequest", Fn: s.TestLargeTxRequest, Slow: true},
		{Name: "Transaction", Fn: s.TestTransaction},
		{Name: "InvalidTxs", Fn: s.TestInvalidTxs},
		{Name: "NewPooledTxs", Fn: s.TestNewPooledTxs},
		{Name: "BlobViolations", Fn: s.TestBlobViolations},
		{Name: "TestBlobTxWithoutSidecar", Fn: s.TestBlobTxWithoutSidecar},
		{Name: "TestBlobTxWithMismatchedSidecar", Fn: s.TestBlobTxWithMismatchedSidecar},
	}
}

func (s *Suite) SnapTests() []utesting.Test {
	return []utesting.Test{
		{Name: "Status", Fn: s.TestSnapStatus},
		{Name: "AccountRange", Fn: s.TestSnapGetAccountRange},
		{Name: "GetByteCodes", Fn: s.TestSnapGetByteCodes},
		{Name: "GetTrieNodes", Fn: s.TestSnapTrieNodes},
		{Name: "GetStorageRanges", Fn: s.TestSnapGetStorageRanges},
	}
}

func (s *Suite) TestStatus(t *utesting.T) {
	t.Log(`This test is just a sanity check. It performs an eth protocol handshake.`)

	conn, err := s.dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
}

// headersMatch returns whether the received headers match the given request
func headersMatch(expected []*types.Header, headers []*types.Header) bool {
	return reflect.DeepEqual(expected, headers)
}

func (s *Suite) TestGetBlockHeaders(t *utesting.T) {
	t.Log(`This test requests block headers from the node.`)

	conn, err := s.dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err = conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
	// Send headers request.
	req := &eth.GetBlockHeadersPacket{
		RequestId: 33,
		GetBlockHeadersRequest: &eth.GetBlockHeadersRequest{
			Origin:  eth.HashOrNumber{Hash: s.chain.blocks[1].Hash()},
			Amount:  2,
			Skip:    1,
			Reverse: false,
		},
	}
	// Read headers response.
	if err := conn.Write(ethProto, eth.GetBlockHeadersMsg, req); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}
	headers := new(eth.BlockHeadersPacket)
	if err := conn.ReadMsg(ethProto, eth.BlockHeadersMsg, &headers); err != nil {
		t.Fatalf("error reading msg: %v", err)
	}
	if got, want := headers.RequestId, req.RequestId; got != want {
		t.Fatalf("unexpected request id")
	}
	// Check for correct headers.
	expected, err := s.chain.GetHeaders(req)
	if err != nil {
		t.Fatalf("failed to get headers for given request: %v", err)
	}
	if !headersMatch(expected, headers.BlockHeadersRequest) {
		t.Fatalf("header mismatch: \nexpected %v \ngot %v", expected, headers)
	}
}

func (s *Suite) TestGetNonexistentBlockHeaders(t *utesting.T) {
	t.Log(`This test sends GetBlockHeaders requests for nonexistent blocks (using max uint64 value) 
to check if the node disconnects after receiving multiple invalid requests.`)

	conn, err := s.dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}

	// Create request with max uint64 value for a nonexistent block
	badReq := &eth.GetBlockHeadersPacket{
		GetBlockHeadersRequest: &eth.GetBlockHeadersRequest{
			Origin:  eth.HashOrNumber{Number: ^uint64(0)},
			Amount:  1,
			Skip:    0,
			Reverse: false,
		},
	}

	// Send request 10 times. Some clients are lient on the first few invalids.
	for i := 0; i < 10; i++ {
		badReq.RequestId = uint64(i)
		if err := conn.Write(ethProto, eth.GetBlockHeadersMsg, badReq); err != nil {
			if err == errDisc {
				t.Fatalf("peer disconnected after %d requests", i+1)
			}
			t.Fatalf("write failed: %v", err)
		}
	}

	// Check if peer disconnects at the end.
	code, _, err := conn.Read()
	if err == errDisc || code == discMsg {
		t.Fatal("peer improperly disconnected")
	}
}

func (s *Suite) TestSimultaneousRequests(t *utesting.T) {
	t.Log(`This test requests blocks headers from the node, performing two requests
concurrently, with different request IDs.`)

	conn, err := s.dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}

	// Create two different requests.
	req1 := &eth.GetBlockHeadersPacket{
		RequestId: uint64(111),
		GetBlockHeadersRequest: &eth.GetBlockHeadersRequest{
			Origin: eth.HashOrNumber{
				Hash: s.chain.blocks[1].Hash(),
			},
			Amount:  2,
			Skip:    1,
			Reverse: false,
		},
	}
	req2 := &eth.GetBlockHeadersPacket{
		RequestId: uint64(222),
		GetBlockHeadersRequest: &eth.GetBlockHeadersRequest{
			Origin: eth.HashOrNumber{
				Hash: s.chain.blocks[1].Hash(),
			},
			Amount:  4,
			Skip:    1,
			Reverse: false,
		},
	}

	// Send both requests.
	if err := conn.Write(ethProto, eth.GetBlockHeadersMsg, req1); err != nil {
		t.Fatalf("failed to write to connection: %v", err)
	}
	if err := conn.Write(ethProto, eth.GetBlockHeadersMsg, req2); err != nil {
		t.Fatalf("failed to write to connection: %v", err)
	}

	// Wait for responses.
	headers1 := new(eth.BlockHeadersPacket)
	if err := conn.ReadMsg(ethProto, eth.BlockHeadersMsg, &headers1); err != nil {
		t.Fatalf("error reading block headers msg: %v", err)
	}
	if got, want := headers1.RequestId, req1.RequestId; got != want {
		t.Fatalf("unexpected request id in response: got %d, want %d", got, want)
	}
	headers2 := new(eth.BlockHeadersPacket)
	if err := conn.ReadMsg(ethProto, eth.BlockHeadersMsg, &headers2); err != nil {
		t.Fatalf("error reading block headers msg: %v", err)
	}
	if got, want := headers2.RequestId, req2.RequestId; got != want {
		t.Fatalf("unexpected request id in response: got %d, want %d", got, want)
	}

	// Check received headers for accuracy.
	if expected, err := s.chain.GetHeaders(req1); err != nil {
		t.Fatalf("failed to get expected headers for request 1: %v", err)
	} else if !headersMatch(expected, headers1.BlockHeadersRequest) {
		t.Fatalf("header mismatch: \nexpected %v \ngot %v", expected, headers1)
	}
	if expected, err := s.chain.GetHeaders(req2); err != nil {
		t.Fatalf("failed to get expected headers for request 2: %v", err)
	} else if !headersMatch(expected, headers2.BlockHeadersRequest) {
		t.Fatalf("header mismatch: \nexpected %v \ngot %v", expected, headers2)
	}
}

func (s *Suite) TestSameRequestID(t *utesting.T) {
	t.Log(`This test requests block headers, performing two concurrent requests with the
same request ID. The node should handle the request by responding to both requests.`)

	conn, err := s.dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}

	// Create two different requests with the same ID.
	reqID := uint64(1234)
	request1 := &eth.GetBlockHeadersPacket{
		RequestId: reqID,
		GetBlockHeadersRequest: &eth.GetBlockHeadersRequest{
			Origin: eth.HashOrNumber{
				Number: 1,
			},
			Amount: 2,
		},
	}
	request2 := &eth.GetBlockHeadersPacket{
		RequestId: reqID,
		GetBlockHeadersRequest: &eth.GetBlockHeadersRequest{
			Origin: eth.HashOrNumber{
				Number: 33,
			},
			Amount: 2,
		},
	}

	// Send the requests.
	if err = conn.Write(ethProto, eth.GetBlockHeadersMsg, request1); err != nil {
		t.Fatalf("failed to write to connection: %v", err)
	}
	if err = conn.Write(ethProto, eth.GetBlockHeadersMsg, request2); err != nil {
		t.Fatalf("failed to write to connection: %v", err)
	}

	// Wait for the responses.
	headers1 := new(eth.BlockHeadersPacket)
	if err := conn.ReadMsg(ethProto, eth.BlockHeadersMsg, &headers1); err != nil {
		t.Fatalf("error reading from connection: %v", err)
	}
	if got, want := headers1.RequestId, request1.RequestId; got != want {
		t.Fatalf("unexpected request id: got %d, want %d", got, want)
	}
	headers2 := new(eth.BlockHeadersPacket)
	if err := conn.ReadMsg(ethProto, eth.BlockHeadersMsg, &headers2); err != nil {
		t.Fatalf("error reading from connection: %v", err)
	}
	if got, want := headers2.RequestId, request2.RequestId; got != want {
		t.Fatalf("unexpected request id: got %d, want %d", got, want)
	}

	// Check if headers match.
	if expected, err := s.chain.GetHeaders(request1); err != nil {
		t.Fatalf("failed to get expected block headers: %v", err)
	} else if !headersMatch(expected, headers1.BlockHeadersRequest) {
		t.Fatalf("header mismatch: \nexpected %v \ngot %v", expected, headers1)
	}
	if expected, err := s.chain.GetHeaders(request2); err != nil {
		t.Fatalf("failed to get expected block headers: %v", err)
	} else if !headersMatch(expected, headers2.BlockHeadersRequest) {
		t.Fatalf("header mismatch: \nexpected %v \ngot %v", expected, headers2)
	}
}

func (s *Suite) TestZeroRequestID(t *utesting.T) {
	t.Log(`This test sends a GetBlockHeaders message with a request-id of zero,
and expects a response.`)

	conn, err := s.dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
	req := &eth.GetBlockHeadersPacket{
		GetBlockHeadersRequest: &eth.GetBlockHeadersRequest{
			Origin: eth.HashOrNumber{Number: 0},
			Amount: 2,
		},
	}
	// Read headers response.
	if err := conn.Write(ethProto, eth.GetBlockHeadersMsg, req); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}
	headers := new(eth.BlockHeadersPacket)
	if err := conn.ReadMsg(ethProto, eth.BlockHeadersMsg, &headers); err != nil {
		t.Fatalf("error reading msg: %v", err)
	}
	if got, want := headers.RequestId, req.RequestId; got != want {
		t.Fatalf("unexpected request id")
	}
	if expected, err := s.chain.GetHeaders(req); err != nil {
		t.Fatalf("failed to get expected block headers: %v", err)
	} else if !headersMatch(expected, headers.BlockHeadersRequest) {
		t.Fatalf("header mismatch: \nexpected %v \ngot %v", expected, headers)
	}
}

func (s *Suite) TestGetBlockBodies(t *utesting.T) {
	t.Log(`This test sends GetBlockBodies requests to the node for known blocks in the test chain.`)

	conn, err := s.dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
	// Create block bodies request.
	req := &eth.GetBlockBodiesPacket{
		RequestId: 55,
		GetBlockBodiesRequest: eth.GetBlockBodiesRequest{
			s.chain.blocks[54].Hash(),
			s.chain.blocks[75].Hash(),
		},
	}
	if err := conn.Write(ethProto, eth.GetBlockBodiesMsg, req); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}
	// Wait for response.
	resp := new(eth.BlockBodiesPacket)
	if err := conn.ReadMsg(ethProto, eth.BlockBodiesMsg, &resp); err != nil {
		t.Fatalf("error reading block bodies msg: %v", err)
	}
	if got, want := resp.RequestId, req.RequestId; got != want {
		t.Fatalf("unexpected request id in respond", got, want)
	}
	bodies := resp.BlockBodiesResponse
	if len(bodies) != len(req.GetBlockBodiesRequest) {
		t.Fatalf("wrong bodies in response: expected %d bodies, got %d", len(req.GetBlockBodiesRequest), len(bodies))
	}
}

// randBuf makes a random buffer size kilobytes large.
func randBuf(size int) []byte {
	buf := make([]byte, size*1024)
	rand.Read(buf)
	return buf
}

func (s *Suite) TestMaliciousHandshake(t *utesting.T) {
	t.Log(`This test tries to send malicious data during the devp2p handshake, in various ways.`)

	// Write hello to client.
	var (
		key, _  = crypto.GenerateKey()
		pub0    = crypto.FromECDSAPub(&key.PublicKey)[1:]
		version = eth.ProtocolVersions[0]
	)
	handshakes := []*protoHandshake{
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: string(randBuf(2)), Version: version},
			},
			ID: pub0,
		},
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: "eth", Version: version},
			},
			ID: append(pub0, byte(0)),
		},
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: "eth", Version: version},
			},
			ID: append(pub0, pub0...),
		},
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: "eth", Version: version},
			},
			ID: randBuf(2),
		},
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: string(randBuf(2)), Version: version},
			},
			ID: randBuf(2),
		},
	}
	for _, handshake := range handshakes {
		conn, err := s.dialAs(key)
		if err != nil {
			t.Fatalf("dial failed: %v", err)
		}
		defer conn.Close()

		if err := conn.Write(ethProto, handshakeMsg, handshake); err != nil {
			t.Fatalf("could not write to connection: %v", err)
		}
		// Check that the peer disconnected
		for i := 0; i < 2; i++ {
			code, _, err := conn.Read()
			if err != nil {
				// Client may have disconnected without sending disconnect msg.
				continue
			}
			switch code {
			case discMsg:
			case handshakeMsg:
				// Discard one hello as Hello's are sent concurrently
				continue
			default:
				t.Fatalf("unexpected msg: code %d", code)
			}
		}
	}
}

func (s *Suite) TestTransaction(t *utesting.T) {
	t.Log(`This test sends a valid transaction to the node and checks if the
transaction gets propagated.`)

	// Nudge client out of syncing mode to accept pending txs.
	if err := s.engine.sendForkchoiceUpdated(); err != nil {
		t.Fatalf("failed to send next block: %v", err)
	}
	from, nonce := s.chain.GetSender(0)
	inner := &types.DynamicFeeTx{
		ChainID:   s.chain.config.ChainID,
		Nonce:     nonce,
		GasTipCap: common.Big1,
		GasFeeCap: s.chain.Head().BaseFee(),
		Gas:       30000,
		To:        &common.Address{0xaa},
		Value:     common.Big1,
	}
	tx, err := s.chain.SignTx(from, types.NewTx(inner))
	if err != nil {
		t.Fatalf("failed to sign tx: %v", err)
	}
	if err := s.sendTxs(t, []*types.Transaction{tx}); err != nil {
		t.Fatal(err)
	}
	s.chain.IncNonce(from, 1)
}

func (s *Suite) TestInvalidTxs(t *utesting.T) {
	t.Log(`This test sends several kinds of invalid transactions and checks that the node
does not propagate them.`)

	// Nudge client out of syncing mode to accept pending txs.
	if err := s.engine.sendForkchoiceUpdated(); err != nil {
		t.Fatalf("failed to send next block: %v", err)
	}

	from, nonce := s.chain.GetSender(0)
	inner := &types.DynamicFeeTx{
		ChainID:   s.chain.config.ChainID,
		Nonce:     nonce,
		GasTipCap: common.Big1,
		GasFeeCap: s.chain.Head().BaseFee(),
		Gas:       30000,
		To:        &common.Address{0xaa},
	}
	tx, err := s.chain.SignTx(from, types.NewTx(inner))
	if err != nil {
		t.Fatalf("failed to sign tx: %v", err)
	}
	if err := s.sendTxs(t, []*types.Transaction{tx}); err != nil {
		t.Fatalf("failed to send txs: %v", err)
	}
	s.chain.IncNonce(from, 1)

	inners := []*types.DynamicFeeTx{
		// Nonce already used
		{
			ChainID:   s.chain.config.ChainID,
			Nonce:     nonce - 1,
			GasTipCap: common.Big1,
			GasFeeCap: s.chain.Head().BaseFee(),
			Gas:       100000,
		},
		// Value exceeds balance
		{
			Nonce:     nonce,
			GasTipCap: common.Big1,
			GasFeeCap: s.chain.Head().BaseFee(),
			Gas:       100000,
			Value:     s.chain.Balance(from),
		},
		// Gas limit too low
		{
			Nonce:     nonce,
			GasTipCap: common.Big1,
			GasFeeCap: s.chain.Head().BaseFee(),
			Gas:       1337,
		},
		// Code size too large
		{
			Nonce:     nonce,
			GasTipCap: common.Big1,
			GasFeeCap: s.chain.Head().BaseFee(),
			Data:      randBuf(50),
			Gas:       1_000_000,
		},
		// Data too large
		{
			Nonce:     nonce,
			GasTipCap: common.Big1,
			GasFeeCap: s.chain.Head().BaseFee(),
			To:        &common.Address{0xaa},
			Data:      randBuf(128),
			Gas:       5_000_000,
		},
	}

	var txs []*types.Transaction
	for _, inner := range inners {
		tx, err := s.chain.SignTx(from, types.NewTx(inner))
		if err != nil {
			t.Fatalf("failed to sign tx: %v", err)
		}
		txs = append(txs, tx)
	}
	if err := s.sendInvalidTxs(t, txs); err != nil {
		t.Fatalf("failed to send invalid txs: %v", err)
	}
}

func (s *Suite) TestLargeTxRequest(t *utesting.T) {
	t.Log(`This test first send ~2000 transactions to the node, then requests them
on another peer connection using GetPooledTransactions.`)

	// Nudge client out of syncing mode to accept pending txs.
	if err := s.engine.sendForkchoiceUpdated(); err != nil {
		t.Fatalf("failed to send next block: %v", err)
	}

	// Generate many transactions to seed target with.
	var (
		from, nonce = s.chain.GetSender(1)
		count       = 2000
		txs         []*types.Transaction
		hashes      []common.Hash
		set         = make(map[common.Hash]struct{})
	)
	for i := 0; i < count; i++ {
		inner := &types.DynamicFeeTx{
			ChainID:   s.chain.config.ChainID,
			Nonce:     nonce + uint64(i),
			GasTipCap: common.Big1,
			GasFeeCap: s.chain.Head().BaseFee(),
			Gas:       75000,
		}
		tx, err := s.chain.SignTx(from, types.NewTx(inner))
		if err != nil {
			t.Fatalf("failed to sign tx: err")
		}
		txs = append(txs, tx)
		set[tx.Hash()] = struct{}{}
		hashes = append(hashes, tx.Hash())
	}
	s.chain.IncNonce(from, uint64(count))

	// Send txs.
	if err := s.sendTxs(t, txs); err != nil {
		t.Fatalf("failed to send txs: %v", err)
	}

	// Set up receive connection to ensure node is peered with the receiving
	// connection before tx request is sent.
	conn, err := s.dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err = conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
	// Create and send pooled tx request.
	req := &eth.GetPooledTransactionsPacket{
		RequestId:                    1234,
		GetPooledTransactionsRequest: hashes,
	}
	if err = conn.Write(ethProto, eth.GetPooledTransactionsMsg, req); err != nil {
		t.Fatalf("could not write to conn: %v", err)
	}
	// Check that all received transactions match those that were sent to node.
	msg := new(eth.PooledTransactionsPacket)
	if err := conn.ReadMsg(ethProto, eth.PooledTransactionsMsg, &msg); err != nil {
		t.Fatalf("error reading from connection: %v", err)
	}
	if got, want := msg.RequestId, req.RequestId; got != want {
		t.Fatalf("unexpected request id in response: got %d, want %d", got, want)
	}
	for _, got := range msg.PooledTransactionsResponse {
		if _, exists := set[got.Hash()]; !exists {
			t.Fatalf("unexpected tx received: %v", got.Hash())
		}
	}
}

func (s *Suite) TestNewPooledTxs(t *utesting.T) {
	t.Log(`This test announces transaction hashes to the node and expects it to fetch
the transactions using a GetPooledTransactions request.`)

	// Nudge client out of syncing mode to accept pending txs.
	if err := s.engine.sendForkchoiceUpdated(); err != nil {
		t.Fatalf("failed to send next block: %v", err)
	}

	var (
		count       = 50
		from, nonce = s.chain.GetSender(1)
		hashes      = make([]common.Hash, count)
		txTypes     = make([]byte, count)
		sizes       = make([]uint32, count)
	)
	for i := 0; i < count; i++ {
		inner := &types.DynamicFeeTx{
			ChainID:   s.chain.config.ChainID,
			Nonce:     nonce + uint64(i),
			GasTipCap: common.Big1,
			GasFeeCap: s.chain.Head().BaseFee(),
			Gas:       75000,
		}
		tx, err := s.chain.SignTx(from, types.NewTx(inner))
		if err != nil {
			t.Fatalf("failed to sign tx: err")
		}
		hashes[i] = tx.Hash()
		txTypes[i] = tx.Type()
		sizes[i] = uint32(tx.Size())
	}
	s.chain.IncNonce(from, uint64(count))

	// Connect to peer.
	conn, err := s.dial()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err = conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}

	// Send announcement.
	ann := eth.NewPooledTransactionHashesPacket{Types: txTypes, Sizes: sizes, Hashes: hashes}
	err = conn.Write(ethProto, eth.NewPooledTransactionHashesMsg, ann)
	if err != nil {
		t.Fatalf("failed to write to connection: %v", err)
	}

	// Wait for GetPooledTxs request.
	for {
		msg, err := conn.ReadEth()
		if err != nil {
			t.Fatalf("failed to read eth msg: %v", err)
		}
		switch msg := msg.(type) {
		case *eth.GetPooledTransactionsPacket:
			if len(msg.GetPooledTransactionsRequest) != len(hashes) {
				t.Fatalf("unexpected number of txs requested: wanted %d, got %d", len(hashes), len(msg.GetPooledTransactionsRequest))
			}
			return
		case *eth.NewPooledTransactionHashesPacket:
			continue
		case *eth.TransactionsPacket:
			continue
		default:
			t.Fatalf("unexpected %s", pretty.Sdump(msg))
		}
	}
}

func makeSidecar(data ...byte) *types.BlobTxSidecar {
	var (
		blobs       = make([]kzg4844.Blob, len(data))
		commitments []kzg4844.Commitment
		proofs      []kzg4844.Proof
	)
	for i := range blobs {
		blobs[i][0] = data[i]
		c, _ := kzg4844.BlobToCommitment(&blobs[i])
		p, _ := kzg4844.ComputeBlobProof(&blobs[i], c)
		commitments = append(commitments, c)
		proofs = append(proofs, p)
	}
	return &types.BlobTxSidecar{
		Blobs:       blobs,
		Commitments: commitments,
		Proofs:      proofs,
	}
}

func (s *Suite) makeBlobTxs(count, blobs int, discriminator byte) (txs types.Transactions) {
	from, nonce := s.chain.GetSender(5)
	for i := 0; i < count; i++ {
		// Make blob data, max of 2 blobs per tx.
		blobdata := make([]byte, blobs%3)
		for i := range blobdata {
			blobdata[i] = discriminator
			blobs -= 1
		}
		inner := &types.BlobTx{
			ChainID:    uint256.MustFromBig(s.chain.config.ChainID),
			Nonce:      nonce + uint64(i),
			GasTipCap:  uint256.NewInt(1),
			GasFeeCap:  uint256.MustFromBig(s.chain.Head().BaseFee()),
			Gas:        100000,
			BlobFeeCap: uint256.MustFromBig(eip4844.CalcBlobFee(s.chain.config, s.chain.Head().Header())),
			BlobHashes: makeSidecar(blobdata...).BlobHashes(),
			Sidecar:    makeSidecar(blobdata...),
		}
		tx, err := s.chain.SignTx(from, types.NewTx(inner))
		if err != nil {
			panic("blob tx signing failed")
		}
		txs = append(txs, tx)
	}
	return txs
}

func (s *Suite) TestBlobViolations(t *utesting.T) {
	t.Log(`This test sends some invalid blob tx announcements and expects the node to disconnect.`)

	if err := s.engine.sendForkchoiceUpdated(); err != nil {
		t.Fatalf("send fcu failed: %v", err)
	}
	// Create blob txs for each tests with unique tx hashes.
	var (
		t1 = s.makeBlobTxs(2, 3, 0x1)
		t2 = s.makeBlobTxs(2, 3, 0x2)
	)
	for _, test := range []struct {
		ann  eth.NewPooledTransactionHashesPacket
		resp eth.PooledTransactionsResponse
	}{
		// Invalid tx size.
		{
			ann: eth.NewPooledTransactionHashesPacket{
				Types:  []byte{types.BlobTxType, types.BlobTxType},
				Sizes:  []uint32{uint32(t1[0].Size()), uint32(t1[1].Size() + 10)},
				Hashes: []common.Hash{t1[0].Hash(), t1[1].Hash()},
			},
			resp: eth.PooledTransactionsResponse(t1),
		},
		// Wrong tx type.
		{
			ann: eth.NewPooledTransactionHashesPacket{
				Types:  []byte{types.DynamicFeeTxType, types.BlobTxType},
				Sizes:  []uint32{uint32(t2[0].Size()), uint32(t2[1].Size())},
				Hashes: []common.Hash{t2[0].Hash(), t2[1].Hash()},
			},
			resp: eth.PooledTransactionsResponse(t2),
		},
	} {
		conn, err := s.dial()
		if err != nil {
			t.Fatalf("dial fail: %v", err)
		}
		if err := conn.peer(s.chain, nil); err != nil {
			t.Fatalf("peering failed: %v", err)
		}
		if err := conn.Write(ethProto, eth.NewPooledTransactionHashesMsg, test.ann); err != nil {
			t.Fatalf("sending announcement failed: %v", err)
		}
		req := new(eth.GetPooledTransactionsPacket)
		if err := conn.ReadMsg(ethProto, eth.GetPooledTransactionsMsg, req); err != nil {
			t.Fatalf("reading pooled tx request failed: %v", err)
		}
		resp := eth.PooledTransactionsPacket{RequestId: req.RequestId, PooledTransactionsResponse: test.resp}
		if err := conn.Write(ethProto, eth.PooledTransactionsMsg, resp); err != nil {
			t.Fatalf("writing pooled tx response failed: %v", err)
		}
		if code, _, err := conn.Read(); err != nil {
			t.Fatalf("expected disconnect on blob violation, got err: %v", err)
		} else if code != discMsg {
			if code == protoOffset(ethProto)+eth.NewPooledTransactionHashesMsg {
				// sometimes we'll get a blob transaction hashes announcement before the disconnect
				// because blob transactions are scheduled to be fetched right away.
				if code, _, err = conn.Read(); err != nil {
					t.Fatalf("expected disconnect on blob violation, got err on second read: %v", err)
				}
			}
			if code != discMsg {
				t.Fatalf("expected disconnect on blob violation, got msg code: %d", code)
			}
		}
		conn.Close()
	}
}

// mangleSidecar returns a copy of the given blob transaction where the sidecar
// data has been modified to produce a different commitment hash.
func mangleSidecar(tx *types.Transaction) *types.Transaction {
	sidecar := tx.BlobTxSidecar()
	copy := types.BlobTxSidecar{
		Blobs:       append([]kzg4844.Blob{}, sidecar.Blobs...),
		Commitments: append([]kzg4844.Commitment{}, sidecar.Commitments...),
		Proofs:      append([]kzg4844.Proof{}, sidecar.Proofs...),
	}
	// zero the first commitment to alter the sidecar hash
	copy.Commitments[0] = kzg4844.Commitment{}
	return tx.WithBlobTxSidecar(&copy)
}

func (s *Suite) TestBlobTxWithoutSidecar(t *utesting.T) {
	t.Log(`This test checks that a blob transaction first advertised/transmitted without blobs will result in the sending peer being disconnected, and the full transaction should be successfully retrieved from another peer.`)
	tx := s.makeBlobTxs(1, 2, 42)[0]
	badTx := tx.WithoutBlobTxSidecar()
	s.testBadBlobTx(t, tx, badTx)
}

func (s *Suite) TestBlobTxWithMismatchedSidecar(t *utesting.T) {
	t.Log(`This test checks that a blob transaction first advertised/transmitted without blobs, whose commitment don't correspond to the blob_versioned_hashes in the transaction, will result in the sending peer being disconnected, and the full transaction should be successfully retrieved from another peer.`)
	tx := s.makeBlobTxs(1, 2, 43)[0]
	badTx := mangleSidecar(tx)
	s.testBadBlobTx(t, tx, badTx)
}

// readUntil reads eth protocol messages until a message of the target type is
// received.  It returns an error if there is a disconnect, or if the context
// is cancelled before a message of the desired type can be read.
func readUntil[T any](ctx context.Context, conn *Conn) (*T, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, context.Canceled
		default:
		}
		received, err := conn.ReadEth()
		if err != nil {
			if err == errDisc {
				return nil, errDisc
			}
			continue
		}

		switch res := received.(type) {
		case *T:
			return res, nil
		}
	}
}

// readUntilDisconnect reads eth protocol messages until the peer disconnects.
// It returns whether the peer disconnects in the next 100ms.
func readUntilDisconnect(conn *Conn) (disconnected bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := readUntil[struct{}](ctx, conn)
	return err == errDisc
}

func (s *Suite) testBadBlobTx(t *utesting.T, tx *types.Transaction, badTx *types.Transaction) {
	stage1, stage2, stage3 := new(sync.WaitGroup), new(sync.WaitGroup), new(sync.WaitGroup)
	stage1.Add(1)
	stage2.Add(1)
	stage3.Add(1)

	errc := make(chan error)

	badPeer := func() {
		// announce the correct hash from the bad peer.
		// when the transaction is first requested before transmitting it from the bad peer,
		// trigger step 2: connection and announcement by good peers

		conn, err := s.dial()
		if err != nil {
			errc <- fmt.Errorf("dial fail: %v", err)
			return
		}
		defer conn.Close()

		if err := conn.peer(s.chain, nil); err != nil {
			errc <- fmt.Errorf("bad peer: peering failed: %v", err)
			return
		}

		ann := eth.NewPooledTransactionHashesPacket{
			Types:  []byte{types.BlobTxType},
			Sizes:  []uint32{uint32(badTx.Size())},
			Hashes: []common.Hash{badTx.Hash()},
		}

		if err := conn.Write(ethProto, eth.NewPooledTransactionHashesMsg, ann); err != nil {
			errc <- fmt.Errorf("sending announcement failed: %v", err)
			return
		}

		req, err := readUntil[eth.GetPooledTransactionsPacket](context.Background(), conn)
		if err != nil {
			errc <- fmt.Errorf("failed to read GetPooledTransactions message: %v", err)
			return
		}

		stage1.Done()
		stage2.Wait()

		// the good peer is connected, and has announced the tx.
		// proceed to send the incorrect one from the bad peer.

		resp := eth.PooledTransactionsPacket{RequestId: req.RequestId, PooledTransactionsResponse: eth.PooledTransactionsResponse(types.Transactions{badTx})}
		if err := conn.Write(ethProto, eth.PooledTransactionsMsg, resp); err != nil {
			errc <- fmt.Errorf("writing pooled tx response failed: %v", err)
			return
		}
		if !readUntilDisconnect(conn) {
			errc <- fmt.Errorf("expected bad peer to be disconnected")
			return
		}
		stage3.Done()
	}

	goodPeer := func() {
		stage1.Wait()

		conn, err := s.dial()
		if err != nil {
			errc <- fmt.Errorf("dial fail: %v", err)
			return
		}
		defer conn.Close()

		if err := conn.peer(s.chain, nil); err != nil {
			errc <- fmt.Errorf("peering failed: %v", err)
			return
		}

		ann := eth.NewPooledTransactionHashesPacket{
			Types:  []byte{types.BlobTxType},
			Sizes:  []uint32{uint32(tx.Size())},
			Hashes: []common.Hash{tx.Hash()},
		}

		if err := conn.Write(ethProto, eth.NewPooledTransactionHashesMsg, ann); err != nil {
			errc <- fmt.Errorf("sending announcement failed: %v", err)
			return
		}

		// wait until the bad peer has transmitted the incorrect transaction
		stage2.Done()
		stage3.Wait()

		// the bad peer has transmitted the bad tx, and been disconnected.
		// transmit the same tx but with correct sidecar from the good peer.

		var req *eth.GetPooledTransactionsPacket
		req, err = readUntil[eth.GetPooledTransactionsPacket](context.Background(), conn)
		if err != nil {
			errc <- fmt.Errorf("reading pooled tx request failed: %v", err)
			return
		}

		if req.GetPooledTransactionsRequest[0] != tx.Hash() {
			errc <- fmt.Errorf("requested unknown tx hash")
			return
		}

		resp := eth.PooledTransactionsPacket{RequestId: req.RequestId, PooledTransactionsResponse: eth.PooledTransactionsResponse(types.Transactions{tx})}
		if err := conn.Write(ethProto, eth.PooledTransactionsMsg, resp); err != nil {
			errc <- fmt.Errorf("writing pooled tx response failed: %v", err)
			return
		}
		if readUntilDisconnect(conn) {
			errc <- fmt.Errorf("unexpected disconnect")
			return
		}
		close(errc)
	}

	if err := s.engine.sendForkchoiceUpdated(); err != nil {
		t.Fatalf("send fcu failed: %v", err)
	}

	go goodPeer()
	go badPeer()
	err := <-errc
	if err != nil {
		t.Fatalf("%v", err)
	}
}
