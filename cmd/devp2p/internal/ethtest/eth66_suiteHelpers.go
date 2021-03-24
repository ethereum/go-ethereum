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
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
)

func (c *Conn) statusExchange66(t *utesting.T, chain *Chain) Message {
	status := &Status{
		ProtocolVersion: uint32(66),
		NetworkID:       chain.chainConfig.ChainID.Uint64(),
		TD:              chain.TD(chain.Len()),
		Head:            chain.blocks[chain.Len()-1].Hash(),
		Genesis:         chain.blocks[0].Hash(),
		ForkID:          chain.ForkID(),
	}
	return c.statusExchange(t, chain, status)
}

func (s *Suite) dial66(t *utesting.T) *Conn {
	conn, err := s.dial()
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}
	conn.caps = append(conn.caps, p2p.Cap{Name: "eth", Version: 66})
	conn.ourHighestProtoVersion = 66
	return conn
}

func (c *Conn) write66(req eth.Packet, code int) error {
	payload, err := rlp.EncodeToBytes(req)
	if err != nil {
		return err
	}
	_, err = c.Conn.Write(uint64(code), payload)
	return err
}

func (c *Conn) read66() (uint64, Message) {
	code, rawData, _, err := c.Conn.Read()
	if err != nil {
		return 0, errorf("could not read from connection: %v", err)
	}

	var msg Message

	switch int(code) {
	case (Hello{}).Code():
		msg = new(Hello)

	case (Ping{}).Code():
		msg = new(Ping)
	case (Pong{}).Code():
		msg = new(Pong)
	case (Disconnect{}).Code():
		msg = new(Disconnect)
	case (Status{}).Code():
		msg = new(Status)
	case (GetBlockHeaders{}).Code():
		ethMsg := new(eth.GetBlockHeadersPacket66)
		if err := rlp.DecodeBytes(rawData, ethMsg); err != nil {
			return 0, errorf("could not rlp decode message: %v", err)
		}
		return ethMsg.RequestId, GetBlockHeaders(*ethMsg.GetBlockHeadersPacket)
	case (BlockHeaders{}).Code():
		ethMsg := new(eth.BlockHeadersPacket66)
		if err := rlp.DecodeBytes(rawData, ethMsg); err != nil {
			return 0, errorf("could not rlp decode message: %v", err)
		}
		return ethMsg.RequestId, BlockHeaders(ethMsg.BlockHeadersPacket)
	case (GetBlockBodies{}).Code():
		ethMsg := new(eth.GetBlockBodiesPacket66)
		if err := rlp.DecodeBytes(rawData, ethMsg); err != nil {
			return 0, errorf("could not rlp decode message: %v", err)
		}
		return ethMsg.RequestId, GetBlockBodies(ethMsg.GetBlockBodiesPacket)
	case (BlockBodies{}).Code():
		ethMsg := new(eth.BlockBodiesPacket66)
		if err := rlp.DecodeBytes(rawData, ethMsg); err != nil {
			return 0, errorf("could not rlp decode message: %v", err)
		}
		return ethMsg.RequestId, BlockBodies(ethMsg.BlockBodiesPacket)
	case (NewBlock{}).Code():
		msg = new(NewBlock)
	case (NewBlockHashes{}).Code():
		msg = new(NewBlockHashes)
	case (Transactions{}).Code():
		msg = new(Transactions)
	case (NewPooledTransactionHashes{}).Code():
		msg = new(NewPooledTransactionHashes)
	default:
		msg = errorf("invalid message code: %d", code)
	}

	if msg != nil {
		if err := rlp.DecodeBytes(rawData, msg); err != nil {
			return 0, errorf("could not rlp decode message: %v", err)
		}
		return 0, msg
	}
	return 0, errorf("invalid message: %s", string(rawData))
}

// ReadAndServe serves GetBlockHeaders requests while waiting
// on another message from the node.
func (c *Conn) readAndServe66(chain *Chain, timeout time.Duration) (uint64, Message) {
	start := time.Now()
	for time.Since(start) < timeout {
		timeout := time.Now().Add(10 * time.Second)
		c.SetReadDeadline(timeout)

		reqID, msg := c.read66()

		switch msg := msg.(type) {
		case *Ping:
			c.Write(&Pong{})
		case *GetBlockHeaders:
			headers, err := chain.GetHeaders(*msg)
			if err != nil {
				return 0, errorf("could not get headers for inbound header request: %v", err)
			}
			resp := &eth.BlockHeadersPacket66{
				RequestId:          reqID,
				BlockHeadersPacket: eth.BlockHeadersPacket(headers),
			}
			if err := c.write66(resp, BlockHeaders{}.Code()); err != nil {
				return 0, errorf("could not write to connection: %v", err)
			}
		default:
			return reqID, msg
		}
	}
	return 0, errorf("no message received within %v", timeout)
}

func (s *Suite) setupConnection66(t *utesting.T) *Conn {
	// create conn
	sendConn := s.dial66(t)
	sendConn.handshake(t)
	sendConn.statusExchange66(t, s.chain)
	return sendConn
}

func (s *Suite) testAnnounce66(t *utesting.T, sendConn, receiveConn *Conn, blockAnnouncement *NewBlock) {
	// Announce the block.
	if err := sendConn.Write(blockAnnouncement); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}
	s.waitAnnounce66(t, receiveConn, blockAnnouncement)
}

func (s *Suite) waitAnnounce66(t *utesting.T, conn *Conn, blockAnnouncement *NewBlock) {
	timeout := 20 * time.Second
	_, msg := conn.readAndServe66(s.chain, timeout)
	switch msg := msg.(type) {
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
		blockHashes := *msg
		t.Logf("received NewBlockHashes message: %s", pretty.Sdump(blockHashes))
		assert.Equal(t, blockAnnouncement.Block.Hash(), blockHashes[0].Hash,
			"wrong block hash in announcement",
		)
	default:
		t.Fatalf("unexpected: %s", pretty.Sdump(msg))
	}
}

// waitForBlock66 waits for confirmation from the client that it has
// imported the given block.
func (c *Conn) waitForBlock66(block *types.Block) error {
	defer c.SetReadDeadline(time.Time{})

	timeout := time.Now().Add(20 * time.Second)
	c.SetReadDeadline(timeout)
	for {
		req := eth.GetBlockHeadersPacket66{
			RequestId: 54,
			GetBlockHeadersPacket: &eth.GetBlockHeadersPacket{
				Origin: eth.HashOrNumber{
					Hash: block.Hash(),
				},
				Amount: 1,
			},
		}
		if err := c.write66(req, GetBlockHeaders{}.Code()); err != nil {
			return err
		}

		reqID, msg := c.read66()
		// check message
		switch msg := msg.(type) {
		case BlockHeaders:
			// check request ID
			if reqID != req.RequestId {
				return fmt.Errorf("request ID mismatch: wanted %d, got %d", req.RequestId, reqID)
			}
			if len(msg) > 0 {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		default:
			return fmt.Errorf("invalid message: %s", pretty.Sdump(msg))
		}
	}
}

func sendSuccessfulTx66(t *utesting.T, s *Suite, tx *types.Transaction) {
	sendConn := s.setupConnection66(t)
	sendSuccessfulTxWithConn(t, s, tx, sendConn)
}

func sendFailingTx66(t *utesting.T, s *Suite, tx *types.Transaction) {
	sendConn, recvConn := s.setupConnection66(t), s.setupConnection66(t)
	sendFailingTxWithConns(t, s, tx, sendConn, recvConn)
}

func (s *Suite) getBlockHeaders66(t *utesting.T, conn *Conn, req eth.Packet, expectedID uint64) BlockHeaders {
	if err := conn.write66(req, GetBlockHeaders{}.Code()); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}
	// check block headers response
	reqID, msg := conn.readAndServe66(s.chain, timeout)

	switch msg := msg.(type) {
	case BlockHeaders:
		if reqID != expectedID {
			t.Fatalf("request ID mismatch: wanted %d, got %d", expectedID, reqID)
		}
		return msg
	default:
		t.Fatalf("unexpected: %s", pretty.Sdump(msg))
		return nil
	}
}

func headersMatch(t *utesting.T, chain *Chain, headers BlockHeaders) {
	for _, header := range headers {
		num := header.Number.Uint64()
		t.Logf("received header (%d): %s", num, pretty.Sdump(header.Hash()))
		assert.Equal(t, chain.blocks[int(num)].Header(), header)
	}
}
