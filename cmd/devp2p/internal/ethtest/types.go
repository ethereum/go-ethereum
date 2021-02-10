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

type EthMessage interface {
	eth.Packet
}

// Status is the network packet for the status message for eth/64 and later.
type Status struct {
	*eth.StatusPacket
}

func (s Status) Code() int { return 16 }

// NewBlockHashes is the network packet for the block announcements.
type NewBlockHashes struct {
	*eth.NewBlockHashesPacket
}

func (nbh NewBlockHashes) Code() int { return 17 }

type Transactions struct {
	*eth.TransactionsPacket
}

func (t Transactions) Code() int { return 18 }

// GetBlockHeaders represents a block header query.
type GetBlockHeaders struct {
	*eth.GetBlockHeadersPacket
}

func (g GetBlockHeaders) Code() int { return 19 }

type BlockHeaders struct {
	*eth.BlockHeadersPacket
}

func (bh BlockHeaders) Code() int { return 20 }

// GetBlockBodies represents a GetBlockBodies request
type GetBlockBodies struct {
	*eth.GetBlockBodiesPacket
}

func (gbb GetBlockBodies) Code() int { return 21 }

// BlockBodies is the network packet for block content distribution.
type BlockBodies struct {
	*eth.BlockBodiesPacket
}

func (bb BlockBodies) Code() int { return 22 }

// NewBlock is the network packet for the block propagation message.
type NewBlock struct {
	*eth.NewBlockPacket
}

func (nb NewBlock) Code() int { return 23 }

// NewPooledTransactionHashes is the network packet for the tx hash propagation message.
type NewPooledTransactionHashes struct {
	*eth.NewPooledTransactionHashesPacket
}

func (nb NewPooledTransactionHashes) Code() int { return 24 }

// HashOrNumber is a combined field for specifying an origin block.
type hashOrNumber struct {
	*eth.HashOrNumber
}

// Conn represents an individual connection with a peer
type Conn struct {
	*rlpx.Conn
	ourKey             *ecdsa.PrivateKey
	ethProtocolVersion uint
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
		return decodeEthMessage(rawData, new(Status))
	case (GetBlockHeaders{}).Code():
		return decodeEthMessage(rawData, new(GetBlockHeaders))
	case (BlockHeaders{}).Code():
		return decodeEthMessage(rawData, new(BlockHeaders))
	case (GetBlockBodies{}).Code():
		return decodeEthMessage(rawData, new(GetBlockBodies))
	case (BlockBodies{}).Code():
		return decodeEthMessage(rawData, new(BlockBodies))
	case (NewBlock{}).Code():
		return decodeEthMessage(rawData, new(NewBlock))
	case (NewBlockHashes{}).Code():
		return decodeEthMessage(rawData, new(NewBlockHashes))
	case (Transactions{}).Code():
		return decodeEthMessage(rawData, new(Transactions))
	case (NewPooledTransactionHashes{}).Code():
		return decodeEthMessage(rawData, new(NewPooledTransactionHashes))
	default:
		return errorf("invalid message code: %d", code)
	}
	// if message is devp2p, decode here
	if err := rlp.DecodeBytes(rawData, msg); err != nil {
		return errorf("could not rlp decode message: %v", err)
	}
	return msg
}

func decodeEthMessage(rawData []byte, msg EthMessage) Message {
	switch msg.(type) {
	case *Status:
		decode := msg.(*Status)
		decode.StatusPacket = new(eth.StatusPacket)
		if err := rlp.DecodeBytes(rawData, decode.StatusPacket); err != nil {
			return errorf("could not rlp decode message: %v", err)
		}
		return decode
	case *NewBlockHashes:
		decode := msg.(*NewBlockHashes)
		decode.NewBlockHashesPacket = new(eth.NewBlockHashesPacket)
		if err := rlp.DecodeBytes(rawData, decode.NewBlockHashesPacket); err != nil {
			return errorf("could not rlp decode message: %v", err)
		}
		return decode
	case *Transactions:
		decode := msg.(*Transactions)
		decode.TransactionsPacket = new(eth.TransactionsPacket)
		if err := rlp.DecodeBytes(rawData, decode.TransactionsPacket); err != nil {
			return errorf("could not rlp decode message: %v", err)
		}
		return decode
	case *GetBlockHeaders:
		decode := msg.(*GetBlockHeaders)
		decode.GetBlockHeadersPacket = new(eth.GetBlockHeadersPacket)
		if err := rlp.DecodeBytes(rawData, decode.GetBlockHeadersPacket); err != nil {
			return errorf("could not rlp decode message: %v", err)
		}
		return decode
	case *BlockHeaders:
		decode := msg.(*BlockHeaders)
		decode.BlockHeadersPacket = new(eth.BlockHeadersPacket)
		if err := rlp.DecodeBytes(rawData, decode.BlockHeadersPacket); err != nil {
			return errorf("could not rlp decode message: %v", err)
		}
		return decode
	case *GetBlockBodies:
		decode := msg.(*GetBlockBodies)
		decode.GetBlockBodiesPacket = new(eth.GetBlockBodiesPacket)
		if err := rlp.DecodeBytes(rawData, decode.GetBlockBodiesPacket); err != nil {
			return errorf("could not rlp decode message: %v", err)
		}
		return decode
	case *BlockBodies:
		decode := msg.(*BlockBodies)
		decode.BlockBodiesPacket = new(eth.BlockBodiesPacket)
		if err := rlp.DecodeBytes(rawData, decode.BlockBodiesPacket); err != nil {
			return errorf("could not rlp decode message: %v", err)
		}
		return decode
	case *NewBlock:
		decode := msg.(*NewBlock)
		decode.NewBlockPacket = new(eth.NewBlockPacket)
		if err := rlp.DecodeBytes(rawData, decode.NewBlockPacket); err != nil {
			return errorf("could not rlp decode message: %v", err)
		}
		return decode
	case *NewPooledTransactionHashes:
		decode := msg.(*NewPooledTransactionHashes)
		decode.NewPooledTransactionHashesPacket = new(eth.NewPooledTransactionHashesPacket)
		if err := rlp.DecodeBytes(rawData, decode.NewPooledTransactionHashesPacket); err != nil {
			return errorf("could not rlp decode message: %v", err)
		}
		return decode
	default:
		return errorf("invalid message: %v", pretty.Sdump(msg))
	}
}

// ReadAndServe serves GetBlockHeaders requests while waiting
// on another message from the node.
func (c *Conn) ReadAndServe(chain *Chain, timeout time.Duration) Message {
	start := time.Now()
	for time.Since(start) < timeout {
		timeout := time.Now().Add(10 * time.Second)
		c.SetReadDeadline(timeout)
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
		err error
	)
	if ethMessage, ok := msg.(EthMessage); ok {
		payload, err = encodeEthMessage(ethMessage)
		if err != nil {
			return err
		}
	} else {
		payload, err = rlp.EncodeToBytes(msg)
		if err != nil {
			return err
		}
	}
	_, err = c.Conn.Write(uint64(msg.Code()), payload)
	return err
}

func encodeEthMessage(msg EthMessage) ([]byte, error) {
	switch msg.Name() {
	case "Status":
		packet := msg.(*Status).StatusPacket
		return rlp.EncodeToBytes(*packet)
	case "NewBlockHashes":
		packet := msg.(*NewBlockHashes).NewBlockHashesPacket
		return rlp.EncodeToBytes(packet)
	case "Transactions":
		packet := msg.(*Transactions).TransactionsPacket
		return rlp.EncodeToBytes(*packet)
	case "GetBlockHeaders":
		packet := msg.(*GetBlockHeaders).GetBlockHeadersPacket
		return rlp.EncodeToBytes(packet)
	case "BlockHeaders":
		packet := msg.(*BlockHeaders).BlockHeadersPacket
		return rlp.EncodeToBytes(*packet)
	case "GetBlockBodies":
		packet := msg.(*GetBlockBodies).GetBlockBodiesPacket
		return rlp.EncodeToBytes(*packet)
	case "BlockBodies":
		packet := msg.(*BlockBodies).BlockBodiesPacket
		return rlp.EncodeToBytes(*packet)
	case "NewBlock":
		packet := msg.(*NewBlock).NewBlockPacket
		return rlp.EncodeToBytes(*packet)
	case "NewPooledTransactionHashes":
		packet := msg.(*NewPooledTransactionHashes).NewPooledTransactionHashesPacket
		return rlp.EncodeToBytes(*packet)
	default:
		return nil, errorf("invalid message: %v", pretty.Sdump(msg))
	}
}

// handshake checks to make sure a `HELLO` is received.
func (c *Conn) handshake(t *utesting.T) Message {
	defer c.SetDeadline(time.Time{})
	c.SetDeadline(time.Now().Add(10 * time.Second))

	// write hello to client
	pub0 := crypto.FromECDSAPub(&c.ourKey.PublicKey)[1:]
	ourHandshake := &Hello{
		Version: 5,
		Caps: []p2p.Cap{
			{Name: "eth", Version: 64},
			{Name: "eth", Version: 65},
		},
		ID: pub0,
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
		if c.ethProtocolVersion == 0 {
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
		if capability.Version > highestEthVersion && capability.Version <= 65 {
			highestEthVersion = capability.Version
		}
	}
	c.ethProtocolVersion = highestEthVersion
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
			if msg.Head != chain.blocks[chain.Len()-1].Hash() {
				t.Fatalf("wrong head block in status: %s", msg.Head.String())
			}
			if msg.TD.Cmp(chain.TD(chain.Len())) != 0 {
				t.Fatalf("wrong TD in status: %v", msg.TD)
			}
			if !reflect.DeepEqual(msg.ForkID, chain.ForkID()) {
				t.Fatalf("wrong fork ID in status: %v", msg.ForkID)
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
	if c.ethProtocolVersion == 0 {
		t.Fatalf("eth protocol version must be set in Conn")
	}
	if status == nil {
		// write status message to client
		status = &Status{
			&eth.StatusPacket{
				ProtocolVersion: uint32(c.ethProtocolVersion),
				NetworkID:       chain.chainConfig.ChainID.Uint64(),
				TD:              chain.TD(chain.Len()),
				Head:            chain.blocks[chain.Len()-1].Hash(),
				Genesis:         chain.blocks[0].Hash(),
				ForkID:          chain.ForkID(),
			},
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

	timeout := time.Now().Add(20 * time.Second)
	c.SetReadDeadline(timeout)
	for {
		req := &GetBlockHeaders{&eth.GetBlockHeadersPacket{Origin: eth.HashOrNumber{Hash: block.Hash()}, Amount: 1}}
		if err := c.Write(req); err != nil {
			return err
		}
		switch msg := c.Read().(type) {
		case *BlockHeaders:
			if len(*msg.BlockHeadersPacket) > 0 {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		default:
			return fmt.Errorf("invalid message: %s", pretty.Sdump(msg))
		}
	}
}
