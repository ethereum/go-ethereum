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
	"fmt"
	"net"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/rlpx"
	"github.com/stretchr/testify/assert"
)

var pretty = spew.ConfigState{
	Indent:                  "  ",
	DisableCapacities:       true,
	DisablePointerAddresses: true,
	SortKeys:                true,
}

// Suite represents a structure used to test the eth
// protocol of a node(s).
type Suite struct {
	Dest *enode.Node

	chain     *Chain
	fullChain *Chain
}

// NewSuite creates and returns a new eth-test suite that can
// be used to test the given node against the given blockchain
// data.
func NewSuite(dest *enode.Node, chainfile string, genesisfile string) *Suite {
	chain, err := loadChain(chainfile, genesisfile)
	if err != nil {
		panic(err)
	}
	return &Suite{
		Dest:      dest,
		chain:     chain.Shorten(1000),
		fullChain: chain,
	}
}

func (s *Suite) AllTests() []utesting.Test {
	return []utesting.Test{
		{Name: "Status", Fn: s.TestStatus},
		{Name: "GetBlockHeaders", Fn: s.TestGetBlockHeaders},
		{Name: "Broadcast", Fn: s.TestBroadcast},
		{Name: "GetBlockBodies", Fn: s.TestGetBlockBodies},
	}
}

// TestStatus attempts to connect to the given node and exchange
// a status message with it, and then check to make sure
// the chain head is correct.
func (s *Suite) TestStatus(t *utesting.T) {
	conn, err := s.dial()
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}
	// get protoHandshake
	conn.handshake(t)
	// get status
	switch msg := conn.statusExchange(t, s.chain).(type) {
	case *Status:
		t.Logf("got status message: %s", pretty.Sdump(msg))
	default:
		t.Fatalf("unexpected: %s", pretty.Sdump(msg))
	}
}

// TestGetBlockHeaders tests whether the given node can respond to
// a `GetBlockHeaders` request and that the response is accurate.
func (s *Suite) TestGetBlockHeaders(t *utesting.T) {
	conn, err := s.dial()
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}

	conn.handshake(t)
	conn.statusExchange(t, s.chain)

	// get block headers
	req := &GetBlockHeaders{
		Origin: hashOrNumber{
			Hash: s.chain.blocks[1].Hash(),
		},
		Amount:  2,
		Skip:    1,
		Reverse: false,
	}

	if err := conn.Write(req); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}

	timeout := 20 * time.Second
	switch msg := conn.ReadAndServe(s.chain, timeout).(type) {
	case *BlockHeaders:
		headers := msg
		for _, header := range *headers {
			num := header.Number.Uint64()
			t.Logf("received header (%d): %s", num, pretty.Sdump(header))
			assert.Equal(t, s.chain.blocks[int(num)].Header(), header)
		}
	default:
		t.Fatalf("unexpected: %s", pretty.Sdump(msg))
	}
}

// TestGetBlockBodies tests whether the given node can respond to
// a `GetBlockBodies` request and that the response is accurate.
func (s *Suite) TestGetBlockBodies(t *utesting.T) {
	conn, err := s.dial()
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}

	conn.handshake(t)
	conn.statusExchange(t, s.chain)
	// create block bodies request
	req := &GetBlockBodies{s.chain.blocks[54].Hash(), s.chain.blocks[75].Hash()}
	if err := conn.Write(req); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}

	timeout := 20 * time.Second
	switch msg := conn.ReadAndServe(s.chain, timeout).(type) {
	case *BlockBodies:
		t.Logf("received %d block bodies", len(*msg))
	default:
		t.Fatalf("unexpected: %s", pretty.Sdump(msg))
	}
}

// TestBroadcast tests whether a block announcement is correctly
// propagated to the given node's peer(s).
func (s *Suite) TestBroadcast(t *utesting.T) {
	// create conn to send block announcement
	sendConn, err := s.dial()
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}
	// create conn to receive block announcement
	receiveConn, err := s.dial()
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}

	sendConn.handshake(t)
	receiveConn.handshake(t)

	sendConn.statusExchange(t, s.chain)
	receiveConn.statusExchange(t, s.chain)

	// sendConn sends the block announcement
	blockAnnouncement := &NewBlock{
		Block: s.fullChain.blocks[1000],
		TD:    s.fullChain.TD(1001),
	}
	if err := sendConn.Write(blockAnnouncement); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}

	timeout := 20 * time.Second
	switch msg := receiveConn.ReadAndServe(s.chain, timeout).(type) {
	case *NewBlock:
		t.Logf("received NewBlock message: %s", pretty.Sdump(msg.Block))
		assert.Equal(t,
			blockAnnouncement.Block.Header(), msg.Block.Header(),
			"wrong block header in announcement",
		)
		assert.Equal(t,
			blockAnnouncement.TD, msg.TD,
			"wrong TD in announcement",
		)
	case *NewBlockHashes:
		hashes := *msg
		t.Logf("received NewBlockHashes message: %s", pretty.Sdump(hashes))
		assert.Equal(t,
			blockAnnouncement.Block.Hash(), hashes[0].Hash,
			"wrong block hash in announcement",
		)
	default:
		t.Fatalf("unexpected: %s", pretty.Sdump(msg))
	}
	// update test suite chain
	s.chain.blocks = append(s.chain.blocks, s.fullChain.blocks[1000])
	// wait for client to update its chain
	if err := receiveConn.waitForBlock(s.chain.Head()); err != nil {
		t.Fatal(err)
	}
}

// dial attempts to dial the given node and perform a handshake,
// returning the created Conn if successful.
func (s *Suite) dial() (*Conn, error) {
	var conn Conn

	fd, err := net.Dial("tcp", fmt.Sprintf("%v:%d", s.Dest.IP(), s.Dest.TCP()))
	if err != nil {
		return nil, err
	}
	conn.Conn = rlpx.NewConn(fd, s.Dest.Pubkey())

	// do encHandshake
	conn.ourKey, _ = crypto.GenerateKey()
	_, err = conn.Handshake(conn.ourKey)
	if err != nil {
		return nil, err
	}

	return &conn, nil
}
