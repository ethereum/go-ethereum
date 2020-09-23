package ethtest

import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/rlpx"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
)

// Suite represents a structure used to test the eth
// protocol of a node(s).
type Suite struct {
	Dest *enode.Node

	chain     *Chain
	fullChain *Chain
}

type Conn struct {
	*rlpx.Conn
	ourKey *ecdsa.PrivateKey
}

func (c *Conn) Read() Message {
	code, rawData, _, err := c.Conn.Read()
	if err != nil {
		return &Error{fmt.Errorf("could not read from connection: %v", err)}
	}

	var msg Message
	switch int(code) {
	case (Hello{}).Code():
		msg = new(Hello)
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
	default:
		return &Error{fmt.Errorf("invalid message code: %d", code)}
	}

	if err := rlp.DecodeBytes(rawData, msg); err != nil {
		return &Error{fmt.Errorf("could not rlp decode message: %v", err)}
	}

	return msg
}

func (c *Conn) Write(msg Message) error {
	payload, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}
	_, err = c.Conn.Write(uint64(msg.Code()), payload)
	return err

}

// handshake checks to make sure a `HELLO` is received.
func (c *Conn) handshake(t *utesting.T) Message {
	// write protoHandshake to client
	pub0 := crypto.FromECDSAPub(&c.ourKey.PublicKey)[1:]
	ourHandshake := &Hello{
		Version: 5,
		Caps:    []p2p.Cap{{Name: "eth", Version: 64}, {Name: "eth", Version: 65}},
		ID:      pub0,
	}
	if err := c.Write(ourHandshake); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}
	// read protoHandshake from client
	switch msg := c.Read().(type) {
	case *Hello:
		return msg
	default:
		t.Fatalf("bad handshake: %v", msg)
		return nil
	}
}

// statusExchange performs a `Status` message exchange with the given
// node.
func (c *Conn) statusExchange(t *utesting.T, chain *Chain) Message {
	// read status message from client
	var message Message
	switch msg := c.Read().(type) {
	case *Status:
		if msg.Head != chain.blocks[chain.Len()-1].Hash() {
			t.Fatalf("wrong head in status: %v", msg.Head)
		}
		if msg.TD.Cmp(chain.TD(chain.Len())) != 0 {
			t.Fatalf("wrong TD in status: %v", msg.TD)
		}
		if !reflect.DeepEqual(msg.ForkID, chain.ForkID()) {
			t.Fatalf("wrong fork ID in status: %v", msg.ForkID)
		}
		message = msg
	default:
		t.Fatalf("bad status message: %v", msg)
	}
	// write status message to client
	status := Status{
		ProtocolVersion: 65,
		NetworkID:       1,
		TD:              chain.TD(chain.Len()),
		Head:            chain.blocks[chain.Len()-1].Hash(),
		Genesis:         chain.blocks[0].Hash(),
		ForkID:          chain.ForkID(),
	}
	if err := c.Write(status); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}

	return message
}

// waitForBlock waits for confirmation from the client that it has
// imported the given block.
func (c *Conn) waitForBlock(block *types.Block) error {
	for {
		req := &GetBlockHeaders{Origin: hashOrNumber{Hash: block.Hash()}, Amount: 1}
		if err := c.Write(req); err != nil {
			return err
		}

		switch msg := c.Read().(type) {
		case *BlockHeaders:
			if len(*msg) > 0 {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		default:
			return fmt.Errorf("invalid message: %v", msg)
		}
	}
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
		t.Logf("%+v\n", msg)
	default:
		t.Fatalf("error: %v", msg)
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

	switch msg := conn.Read().(type) {
	case *BlockHeaders:
		headers := msg
		for _, header := range *headers {
			num := header.Number.Uint64()
			assert.Equal(t, s.chain.blocks[int(num)].Header(), header)
			t.Logf("\nHEADER FOR BLOCK NUMBER %d: %+v\n", header.Number, header)
		}
	default:
		t.Fatalf("error: %v", msg)
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

	switch msg := conn.Read().(type) {
	case *BlockBodies:
		bodies := msg
		for _, body := range *bodies {
			t.Logf("\nBODY: %+v\n", body)
		}
	default:
		t.Fatalf("error: %v", msg)
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

	switch msg := receiveConn.Read().(type) {
	case *NewBlock:
		assert.Equal(t, blockAnnouncement.Block.Header(), msg.Block.Header(),
			"wrong block header in announcement")
		assert.Equal(t, blockAnnouncement.TD, msg.TD,
			"wrong TD in announcement")
	case *NewBlockHashes:
		hashes := *msg
		assert.Equal(t, blockAnnouncement.Block.Hash(), hashes[0].Hash,
			"wrong block hash in announcement")
	default:
		t.Fatal(msg)
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
