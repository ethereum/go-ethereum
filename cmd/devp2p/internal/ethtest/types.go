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
	"crypto/ecdsa"
	"fmt"
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/rlpx"
	"github.com/ethereum/go-ethereum/rlp"
)

type Message interface {
	Code() int
}

type Error struct {
	err error
}

func (e *Error) Unwrap() error  { return e.err }
func (e *Error) Error() string  { return e.err.Error() }
func (e *Error) Code() int      { return -1 }
func (e *Error) String() string { return e.Error() }

func errorf(format string, args ...interface{}) *Error {
	return &Error{fmt.Errorf(format, args...)}
}

// Hello is the RLP structure of the protocol handshake.
type Hello struct {
	Version    uint64
	Name       string
	Caps       []p2p.Cap
	ListenPort uint64
	ID         []byte // secp256k1 public key

	// Ignore additional fields (for forward compatibility).
	Rest []rlp.RawValue `rlp:"tail"`
}

func (h Hello) Code() int { return 0x00 }

// Disconnect is the RLP structure for a disconnect message.
type Disconnect struct {
	Reason p2p.DiscReason
}

func (d Disconnect) Code() int { return 0x01 }

type Ping struct{}

func (p Ping) Code() int { return 0x02 }

type Pong struct{}

func (p Pong) Code() int { return 0x03 }

// Status is the network packet for the status message for eth/64 and later.
type Status eth.StatusPacket

func (s Status) Code() int { return 16 }

// NewBlockHashes is the network packet for the block announcements.
type NewBlockHashes eth.NewBlockHashesPacket

func (nbh NewBlockHashes) Code() int { return 17 }

type Transactions eth.TransactionsPacket

func (t Transactions) Code() int { return 18 }

// GetBlockHeaders represents a block header query.
type GetBlockHeaders eth.GetBlockHeadersPacket

func (g GetBlockHeaders) Code() int { return 19 }

type BlockHeaders eth.BlockHeadersPacket

func (bh BlockHeaders) Code() int { return 20 }

// GetBlockBodies represents a GetBlockBodies request
type GetBlockBodies eth.GetBlockBodiesPacket

func (gbb GetBlockBodies) Code() int { return 21 }

// BlockBodies is the network packet for block content distribution.
type BlockBodies eth.BlockBodiesPacket

func (bb BlockBodies) Code() int { return 22 }

// NewBlock is the network packet for the block propagation message.
type NewBlock eth.NewBlockPacket

func (nb NewBlock) Code() int { return 23 }

// NewPooledTransactionHashes is the network packet for the tx hash propagation message.
type NewPooledTransactionHashes eth.NewPooledTransactionHashesPacket

func (nb NewPooledTransactionHashes) Code() int { return 24 }

type GetPooledTransactions eth.GetPooledTransactionsPacket

func (gpt GetPooledTransactions) Code() int { return 25 }

type PooledTransactions eth.PooledTransactionsPacket

func (pt PooledTransactions) Code() int { return 26 }

// Conn represents an individual connection with a peer
type Conn struct {
	*rlpx.Conn
	ourKey                 *ecdsa.PrivateKey
	negotiatedProtoVersion uint
	ourHighestProtoVersion uint
	caps                   []p2p.Cap
}

func (c *Conn) Read() Message {
	code, rawData, _, err := c.Conn.Read()
	if err != nil {
		return errorf("could not read from connection: %v", err)
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
		msg = new(GetBlockHeaders)
	case (BlockHeaders{}).Code():
		msg = new(BlockHeaders)
	case (GetBlockBodies{}).Code():
		msg = new(GetBlockBodies)
	case (BlockBodies{}).Code():
		msg = new(BlockBodies)
	case (NewBlock{}).Code():
		msg = new(NewBlock)
	case (NewBlockHashes{}).Code():
		msg = new(NewBlockHashes)
	case (Transactions{}).Code():
		msg = new(Transactions)
	case (NewPooledTransactionHashes{}).Code():
		msg = new(NewPooledTransactionHashes)
	case (GetPooledTransactions{}.Code()):
		msg = new(GetPooledTransactions)
	case (PooledTransactions{}.Code()):
		msg = new(PooledTransactions)
	default:
		return errorf("invalid message code: %d", code)
	}
	// if message is devp2p, decode here
	if err := rlp.DecodeBytes(rawData, msg); err != nil {
		return errorf("could not rlp decode message: %v", err)
	}
	return msg
}

// ReadAndServe serves GetBlockHeaders requests while waiting
// on another message from the node.
func (c *Conn) ReadAndServe(chain *Chain, timeout time.Duration) Message {
	start := time.Now()
	for time.Since(start) < timeout {
		c.SetReadDeadline(time.Now().Add(5 * time.Second))
		switch msg := c.Read().(type) {
		case *Ping:
			c.Write(&Pong{})
		case *GetBlockHeaders:
			req := *msg
			headers, err := chain.GetHeaders(req)
			if err != nil {
				return errorf("could not get headers for inbound header request: %v", err)
			}

			if err := c.Write(headers); err != nil {
				return errorf("could not write to connection: %v", err)
			}
		default:
			return msg
		}
	}
	return errorf("no message received within %v", timeout)
}

func (c *Conn) Write(msg Message) error {
	// check if message is eth protocol message
	var (
		payload []byte
		err     error
	)
	payload, err = rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}
	_, err = c.Conn.Write(uint64(msg.Code()), payload)
	return err
}

// handshake checks to make sure a `HELLO` is received.
func (c *Conn) handshake(t *utesting.T) Message {
	defer c.SetDeadline(time.Time{})
	c.SetDeadline(time.Now().Add(10 * time.Second))

	// write hello to client
	pub0 := crypto.FromECDSAPub(&c.ourKey.PublicKey)[1:]
	ourHandshake := &Hello{
		Version: 5,
		Caps:    c.caps,
		ID:      pub0,
	}
	if err := c.Write(ourHandshake); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}
	// read hello from client
	switch msg := c.Read().(type) {
	case *Hello:
		// set snappy if version is at least 5
		if msg.Version >= 5 {
			c.SetSnappy(true)
		}
		c.negotiateEthProtocol(msg.Caps)
		if c.negotiatedProtoVersion == 0 {
			t.Fatalf("unexpected eth protocol version")
		}
		return msg
	default:
		t.Fatalf("bad handshake: %#v", msg)
		return nil
	}
}

// negotiateEthProtocol sets the Conn's eth protocol version
// to highest advertised capability from peer
func (c *Conn) negotiateEthProtocol(caps []p2p.Cap) {
	var highestEthVersion uint
	for _, capability := range caps {
		if capability.Name != "eth" {
			continue
		}
		if capability.Version > highestEthVersion && capability.Version <= c.ourHighestProtoVersion {
			highestEthVersion = capability.Version
		}
	}
	c.negotiatedProtoVersion = highestEthVersion
}

// statusExchange performs a `Status` message exchange with the given
// node.
func (c *Conn) statusExchange(t *utesting.T, chain *Chain, status *Status) Message {
	defer c.SetDeadline(time.Time{})
	c.SetDeadline(time.Now().Add(20 * time.Second))

	// read status message from client
	var message Message
loop:
	for {
		switch msg := c.Read().(type) {
		case *Status:
			if have, want := msg.Head, chain.blocks[chain.Len()-1].Hash(); have != want {
				t.Fatalf("wrong head block in status, want:  %#x (block %d) have %#x",
					want, chain.blocks[chain.Len()-1].NumberU64(), have)
			}
			if have, want := msg.TD.Cmp(chain.TD(chain.Len())), 0; have != want {
				t.Fatalf("wrong TD in status: have %v want %v", have, want)
			}
			if have, want := msg.ForkID, chain.ForkID(); !reflect.DeepEqual(have, want) {
				t.Fatalf("wrong fork ID in status: have %v, want %v", have, want)
			}
			message = msg
			break loop
		case *Disconnect:
			t.Fatalf("disconnect received: %v", msg.Reason)
		case *Ping:
			c.Write(&Pong{}) // TODO (renaynay): in the future, this should be an error
			// (PINGs should not be a response upon fresh connection)
		default:
			t.Fatalf("bad status message: %s", pretty.Sdump(msg))
		}
	}
	// make sure eth protocol version is set for negotiation
	if c.negotiatedProtoVersion == 0 {
		t.Fatalf("eth protocol version must be set in Conn")
	}
	if status == nil {
		// write status message to client
		status = &Status{
			ProtocolVersion: uint32(c.negotiatedProtoVersion),
			NetworkID:       chain.chainConfig.ChainID.Uint64(),
			TD:              chain.TD(chain.Len()),
			Head:            chain.blocks[chain.Len()-1].Hash(),
			Genesis:         chain.blocks[0].Hash(),
			ForkID:          chain.ForkID(),
		}
	}

	if err := c.Write(status); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}

	return message
}

// waitForBlock waits for confirmation from the client that it has
// imported the given block.
func (c *Conn) waitForBlock(block *types.Block) error {
	defer c.SetReadDeadline(time.Time{})

	c.SetReadDeadline(time.Now().Add(20 * time.Second))
	// note: if the node has not yet imported the block, it will respond
	// to the GetBlockHeaders request with an empty BlockHeaders response,
	// so the GetBlockHeaders request must be sent again until the BlockHeaders
	// response contains the desired header.
	for {
		req := &GetBlockHeaders{Origin: eth.HashOrNumber{Hash: block.Hash()}, Amount: 1}
		if err := c.Write(req); err != nil {
			return err
		}
		switch msg := c.Read().(type) {
		case *BlockHeaders:
			for _, header := range *msg {
				if header.Number.Uint64() == block.NumberU64() {
					return nil
				}
			}
			time.Sleep(100 * time.Millisecond)
		default:
			return fmt.Errorf("invalid message: %s", pretty.Sdump(msg))
		}
	}
}
