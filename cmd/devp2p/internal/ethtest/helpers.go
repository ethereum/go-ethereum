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
	"reflect"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/rlpx"
)

var (
	pretty = spew.ConfigState{
		Indent:                  "  ",
		DisableCapacities:       true,
		DisablePointerAddresses: true,
		SortKeys:                true,
	}
	timeout = 20 * time.Second
)

// Is_66 checks if the node supports the eth66 protocol version,
// and if not, exists the test suite
func (s *Suite) Is_66(t *utesting.T) {
	conn, err := s.dial66()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	if err := conn.handshake(); err != nil {
		t.Fatalf("handshake failed: %v", err)
	}
	if conn.negotiatedProtoVersion < 66 {
		t.Fail()
	}
}

// dial attempts to dial the given node and perform a handshake,
// returning the created Conn if successful.
func (s *Suite) dial() (*Conn, error) {
	// dial
	fd, err := net.Dial("tcp", fmt.Sprintf("%v:%d", s.Dest.IP(), s.Dest.TCP()))
	if err != nil {
		return nil, err
	}
	conn := Conn{Conn: rlpx.NewConn(fd, s.Dest.Pubkey())}
	// do encHandshake
	conn.ourKey, _ = crypto.GenerateKey()
	_, err = conn.Handshake(conn.ourKey)
	if err != nil {
		conn.Close()
		return nil, err
	}
	// set default p2p capabilities
	conn.caps = []p2p.Cap{
		{Name: "eth", Version: 64},
		{Name: "eth", Version: 65},
	}
	conn.ourHighestProtoVersion = 65
	return &conn, nil
}

// dial66 attempts to dial the given node and perform a handshake,
// returning the created Conn with additional eth66 capabilities if
// successful
func (s *Suite) dial66() (*Conn, error) {
	conn, err := s.dial()
	if err != nil {
		return nil, fmt.Errorf("dial failed: %v", err)
	}
	conn.caps = append(conn.caps, p2p.Cap{Name: "eth", Version: 66})
	conn.ourHighestProtoVersion = 66
	return conn, nil
}

// dial66 attempts to dial the given node and perform a handshake,
// returning the created Conn with additional snap/1 capabilities if
// successful.
func (s *Suite) dialSnap() (*Conn, error) {
	conn, err := s.dial66()
	if err != nil {
		return nil, fmt.Errorf("dial failed: %v", err)
	}
	conn.caps = append(conn.caps, p2p.Cap{Name: "snap", Version: 1})
	conn.ourHighestSnapProtoVersion = 1
	return conn, nil
}

// peer performs both the protocol handshake and the status message
// exchange with the node in order to peer with it.
func (c *Conn) peer(chain *Chain, status *Status) error {
	if err := c.handshake(); err != nil {
		return fmt.Errorf("handshake failed: %v", err)
	}
	if _, err := c.statusExchange(chain, status); err != nil {
		return fmt.Errorf("status exchange failed: %v", err)
	}
	return nil
}

// handshake performs a protocol handshake with the node.
func (c *Conn) handshake() error {
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
		return fmt.Errorf("write to connection failed: %v", err)
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
			return fmt.Errorf("could not negotiate eth protocol (remote caps: %v, local eth version: %v)", msg.Caps, c.ourHighestProtoVersion)
		}
		// If we require snap, verify that it was negotiated
		if c.ourHighestSnapProtoVersion != c.negotiatedSnapProtoVersion {
			return fmt.Errorf("could not negotiate snap protocol (remote caps: %v, local snap version: %v)", msg.Caps, c.ourHighestSnapProtoVersion)
		}
		return nil
	default:
		return fmt.Errorf("bad handshake: %#v", msg)
	}
}

// negotiateEthProtocol sets the Conn's eth protocol version to highest
// advertised capability from peer.
func (c *Conn) negotiateEthProtocol(caps []p2p.Cap) {
	var highestEthVersion uint
	var highestSnapVersion uint
	for _, capability := range caps {
		switch capability.Name {
		case "eth":
			if capability.Version > highestEthVersion && capability.Version <= c.ourHighestProtoVersion {
				highestEthVersion = capability.Version
			}
		case "snap":
			if capability.Version > highestSnapVersion && capability.Version <= c.ourHighestSnapProtoVersion {
				highestSnapVersion = capability.Version
			}
		}
	}
	c.negotiatedProtoVersion = highestEthVersion
	c.negotiatedSnapProtoVersion = highestSnapVersion
}

// statusExchange performs a `Status` message exchange with the given node.
func (c *Conn) statusExchange(chain *Chain, status *Status) (Message, error) {
	defer c.SetDeadline(time.Time{})
	c.SetDeadline(time.Now().Add(20 * time.Second))

	// read status message from client
	var message Message
loop:
	for {
		switch msg := c.Read().(type) {
		case *Status:
			if have, want := msg.Head, chain.blocks[chain.Len()-1].Hash(); have != want {
				return nil, fmt.Errorf("wrong head block in status, want:  %#x (block %d) have %#x",
					want, chain.blocks[chain.Len()-1].NumberU64(), have)
			}
			if have, want := msg.TD.Cmp(chain.TD()), 0; have != want {
				return nil, fmt.Errorf("wrong TD in status: have %v want %v", have, want)
			}
			if have, want := msg.ForkID, chain.ForkID(); !reflect.DeepEqual(have, want) {
				return nil, fmt.Errorf("wrong fork ID in status: have %v, want %v", have, want)
			}
			if have, want := msg.ProtocolVersion, c.ourHighestProtoVersion; have != uint32(want) {
				return nil, fmt.Errorf("wrong protocol version: have %v, want %v", have, want)
			}
			message = msg
			break loop
		case *Disconnect:
			return nil, fmt.Errorf("disconnect received: %v", msg.Reason)
		case *Ping:
			c.Write(&Pong{}) // TODO (renaynay): in the future, this should be an error
			// (PINGs should not be a response upon fresh connection)
		default:
			return nil, fmt.Errorf("bad status message: %s", pretty.Sdump(msg))
		}
	}
	// make sure eth protocol version is set for negotiation
	if c.negotiatedProtoVersion == 0 {
		return nil, fmt.Errorf("eth protocol version must be set in Conn")
	}
	if status == nil {
		// default status message
		status = &Status{
			ProtocolVersion: uint32(c.negotiatedProtoVersion),
			NetworkID:       chain.chainConfig.ChainID.Uint64(),
			TD:              chain.TD(),
			Head:            chain.blocks[chain.Len()-1].Hash(),
			Genesis:         chain.blocks[0].Hash(),
			ForkID:          chain.ForkID(),
		}
	}
	if err := c.Write(status); err != nil {
		return nil, fmt.Errorf("write to connection failed: %v", err)
	}
	return message, nil
}

// createSendAndRecvConns creates two connections, one for sending messages to the
// node, and one for receiving messages from the node.
func (s *Suite) createSendAndRecvConns(isEth66 bool) (*Conn, *Conn, error) {
	var (
		sendConn *Conn
		recvConn *Conn
		err      error
	)
	if isEth66 {
		sendConn, err = s.dial66()
		if err != nil {
			return nil, nil, fmt.Errorf("dial failed: %v", err)
		}
		recvConn, err = s.dial66()
		if err != nil {
			sendConn.Close()
			return nil, nil, fmt.Errorf("dial failed: %v", err)
		}
	} else {
		sendConn, err = s.dial()
		if err != nil {
			return nil, nil, fmt.Errorf("dial failed: %v", err)
		}
		recvConn, err = s.dial()
		if err != nil {
			sendConn.Close()
			return nil, nil, fmt.Errorf("dial failed: %v", err)
		}
	}
	return sendConn, recvConn, nil
}

func (c *Conn) readAndServe(chain *Chain, timeout time.Duration) Message {
	if c.negotiatedProtoVersion == 66 {
		_, msg := c.readAndServe66(chain, timeout)
		return msg
	}
	return c.readAndServe65(chain, timeout)
}

// readAndServe serves GetBlockHeaders requests while waiting
// on another message from the node.
func (c *Conn) readAndServe65(chain *Chain, timeout time.Duration) Message {
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

// readAndServe66 serves eth66 GetBlockHeaders requests while waiting
// on another message from the node.
func (c *Conn) readAndServe66(chain *Chain, timeout time.Duration) (uint64, Message) {
	start := time.Now()
	for time.Since(start) < timeout {
		c.SetReadDeadline(time.Now().Add(10 * time.Second))

		reqID, msg := c.Read66()

		switch msg := msg.(type) {
		case *Ping:
			c.Write(&Pong{})
		case GetBlockHeaders:
			headers, err := chain.GetHeaders(msg)
			if err != nil {
				return 0, errorf("could not get headers for inbound header request: %v", err)
			}
			resp := &eth.BlockHeadersPacket66{
				RequestId:          reqID,
				BlockHeadersPacket: eth.BlockHeadersPacket(headers),
			}
			if err := c.Write66(resp, BlockHeaders{}.Code()); err != nil {
				return 0, errorf("could not write to connection: %v", err)
			}
		default:
			return reqID, msg
		}
	}
	return 0, errorf("no message received within %v", timeout)
}

// headersRequest executes the given `GetBlockHeaders` request.
func (c *Conn) headersRequest(request *GetBlockHeaders, chain *Chain, isEth66 bool, reqID uint64) (BlockHeaders, error) {
	defer c.SetReadDeadline(time.Time{})
	c.SetReadDeadline(time.Now().Add(20 * time.Second))
	// if on eth66 connection, perform eth66 GetBlockHeaders request
	if isEth66 {
		return getBlockHeaders66(chain, c, request, reqID)
	}
	if err := c.Write(request); err != nil {
		return nil, err
	}
	switch msg := c.readAndServe(chain, timeout).(type) {
	case *BlockHeaders:
		return *msg, nil
	default:
		return nil, fmt.Errorf("invalid message: %s", pretty.Sdump(msg))
	}
}

func (c *Conn) snapRequest(msg Message, id uint64, chain *Chain) (Message, error) {
	defer c.SetReadDeadline(time.Time{})
	c.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err := c.Write(msg); err != nil {
		return nil, fmt.Errorf("could not write to connection: %v", err)
	}
	return c.ReadSnap(id)
}

// getBlockHeaders66 executes the given `GetBlockHeaders` request over the eth66 protocol.
func getBlockHeaders66(chain *Chain, conn *Conn, request *GetBlockHeaders, id uint64) (BlockHeaders, error) {
	// write request
	packet := eth.GetBlockHeadersPacket(*request)
	req := &eth.GetBlockHeadersPacket66{
		RequestId:             id,
		GetBlockHeadersPacket: &packet,
	}
	if err := conn.Write66(req, GetBlockHeaders{}.Code()); err != nil {
		return nil, fmt.Errorf("could not write to connection: %v", err)
	}
	// wait for response
	msg := conn.waitForResponse(chain, timeout, req.RequestId)
	headers, ok := msg.(BlockHeaders)
	if !ok {
		return nil, fmt.Errorf("unexpected message received: %s", pretty.Sdump(msg))
	}
	return headers, nil
}

// headersMatch returns whether the received headers match the given request
func headersMatch(expected BlockHeaders, headers BlockHeaders) bool {
	return reflect.DeepEqual(expected, headers)
}

// waitForResponse reads from the connection until a response with the expected
// request ID is received.
func (c *Conn) waitForResponse(chain *Chain, timeout time.Duration, requestID uint64) Message {
	for {
		id, msg := c.readAndServe66(chain, timeout)
		if id == requestID {
			return msg
		}
	}
}

// sendNextBlock broadcasts the next block in the chain and waits
// for the node to propagate the block and import it into its chain.
func (s *Suite) sendNextBlock(isEth66 bool) error {
	// set up sending and receiving connections
	sendConn, recvConn, err := s.createSendAndRecvConns(isEth66)
	if err != nil {
		return err
	}
	defer sendConn.Close()
	defer recvConn.Close()
	if err = sendConn.peer(s.chain, nil); err != nil {
		return fmt.Errorf("peering failed: %v", err)
	}
	if err = recvConn.peer(s.chain, nil); err != nil {
		return fmt.Errorf("peering failed: %v", err)
	}
	// create new block announcement
	nextBlock := s.fullChain.blocks[s.chain.Len()]
	blockAnnouncement := &NewBlock{
		Block: nextBlock,
		TD:    s.fullChain.TotalDifficultyAt(s.chain.Len()),
	}
	// send announcement and wait for node to request the header
	if err = s.testAnnounce(sendConn, recvConn, blockAnnouncement); err != nil {
		return fmt.Errorf("failed to announce block: %v", err)
	}
	// wait for client to update its chain
	if err = s.waitForBlockImport(recvConn, nextBlock, isEth66); err != nil {
		return fmt.Errorf("failed to receive confirmation of block import: %v", err)
	}
	// update test suite chain
	s.chain.blocks = append(s.chain.blocks, nextBlock)
	return nil
}

// testAnnounce writes a block announcement to the node and waits for the node
// to propagate it.
func (s *Suite) testAnnounce(sendConn, receiveConn *Conn, blockAnnouncement *NewBlock) error {
	if err := sendConn.Write(blockAnnouncement); err != nil {
		return fmt.Errorf("could not write to connection: %v", err)
	}
	return s.waitAnnounce(receiveConn, blockAnnouncement)
}

// waitAnnounce waits for a NewBlock or NewBlockHashes announcement from the node.
func (s *Suite) waitAnnounce(conn *Conn, blockAnnouncement *NewBlock) error {
	for {
		switch msg := conn.readAndServe(s.chain, timeout).(type) {
		case *NewBlock:
			if !reflect.DeepEqual(blockAnnouncement.Block.Header(), msg.Block.Header()) {
				return fmt.Errorf("wrong header in block announcement: \nexpected %v "+
					"\ngot %v", blockAnnouncement.Block.Header(), msg.Block.Header())
			}
			if !reflect.DeepEqual(blockAnnouncement.TD, msg.TD) {
				return fmt.Errorf("wrong TD in announcement: expected %v, got %v", blockAnnouncement.TD, msg.TD)
			}
			return nil
		case *NewBlockHashes:
			hashes := *msg
			if blockAnnouncement.Block.Hash() != hashes[0].Hash {
				return fmt.Errorf("wrong block hash in announcement: expected %v, got %v", blockAnnouncement.Block.Hash(), hashes[0].Hash)
			}
			return nil
		case *NewPooledTransactionHashes:
			// ignore tx announcements from previous tests
			continue
		default:
			return fmt.Errorf("unexpected: %s", pretty.Sdump(msg))
		}
	}
}

func (s *Suite) waitForBlockImport(conn *Conn, block *types.Block, isEth66 bool) error {
	defer conn.SetReadDeadline(time.Time{})
	conn.SetReadDeadline(time.Now().Add(20 * time.Second))
	// create request
	req := &GetBlockHeaders{
		Origin: eth.HashOrNumber{
			Hash: block.Hash(),
		},
		Amount: 1,
	}
	// loop until BlockHeaders response contains desired block, confirming the
	// node imported the block
	for {
		var (
			headers BlockHeaders
			err     error
		)
		if isEth66 {
			requestID := uint64(54)
			headers, err = conn.headersRequest(req, s.chain, eth66, requestID)
		} else {
			headers, err = conn.headersRequest(req, s.chain, eth65, 0)
		}
		if err != nil {
			return fmt.Errorf("GetBlockHeader request failed: %v", err)
		}
		// if headers response is empty, node hasn't imported block yet, try again
		if len(headers) == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if !reflect.DeepEqual(block.Header(), headers[0]) {
			return fmt.Errorf("wrong header returned: wanted %v, got %v", block.Header(), headers[0])
		}
		return nil
	}
}

func (s *Suite) oldAnnounce(isEth66 bool) error {
	sendConn, receiveConn, err := s.createSendAndRecvConns(isEth66)
	if err != nil {
		return err
	}
	defer sendConn.Close()
	defer receiveConn.Close()
	if err := sendConn.peer(s.chain, nil); err != nil {
		return fmt.Errorf("peering failed: %v", err)
	}
	if err := receiveConn.peer(s.chain, nil); err != nil {
		return fmt.Errorf("peering failed: %v", err)
	}
	// create old block announcement
	oldBlockAnnounce := &NewBlock{
		Block: s.chain.blocks[len(s.chain.blocks)/2],
		TD:    s.chain.blocks[len(s.chain.blocks)/2].Difficulty(),
	}
	if err := sendConn.Write(oldBlockAnnounce); err != nil {
		return fmt.Errorf("could not write to connection: %v", err)
	}
	// wait to see if the announcement is propagated
	switch msg := receiveConn.readAndServe(s.chain, time.Second*8).(type) {
	case *NewBlock:
		block := *msg
		if block.Block.Hash() == oldBlockAnnounce.Block.Hash() {
			return fmt.Errorf("unexpected: block propagated: %s", pretty.Sdump(msg))
		}
	case *NewBlockHashes:
		hashes := *msg
		for _, hash := range hashes {
			if hash.Hash == oldBlockAnnounce.Block.Hash() {
				return fmt.Errorf("unexpected: block announced: %s", pretty.Sdump(msg))
			}
		}
	case *Error:
		errMsg := *msg
		// check to make sure error is timeout (propagation didn't come through == test successful)
		if !strings.Contains(errMsg.String(), "timeout") {
			return fmt.Errorf("unexpected error: %v", pretty.Sdump(msg))
		}
	default:
		return fmt.Errorf("unexpected: %s", pretty.Sdump(msg))
	}
	return nil
}

func (s *Suite) maliciousHandshakes(t *utesting.T, isEth66 bool) error {
	var (
		conn *Conn
		err  error
	)
	if isEth66 {
		conn, err = s.dial66()
		if err != nil {
			return fmt.Errorf("dial failed: %v", err)
		}
	} else {
		conn, err = s.dial()
		if err != nil {
			return fmt.Errorf("dial failed: %v", err)
		}
	}
	defer conn.Close()
	// write hello to client
	pub0 := crypto.FromECDSAPub(&conn.ourKey.PublicKey)[1:]
	handshakes := []*Hello{
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: largeString(2), Version: 64},
			},
			ID: pub0,
		},
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: "eth", Version: 64},
				{Name: "eth", Version: 65},
			},
			ID: append(pub0, byte(0)),
		},
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: "eth", Version: 64},
				{Name: "eth", Version: 65},
			},
			ID: append(pub0, pub0...),
		},
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: "eth", Version: 64},
				{Name: "eth", Version: 65},
			},
			ID: largeBuffer(2),
		},
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: largeString(2), Version: 64},
			},
			ID: largeBuffer(2),
		},
	}
	for i, handshake := range handshakes {
		t.Logf("Testing malicious handshake %v\n", i)
		if err := conn.Write(handshake); err != nil {
			return fmt.Errorf("could not write to connection: %v", err)
		}
		// check that the peer disconnected
		for i := 0; i < 2; i++ {
			switch msg := conn.readAndServe(s.chain, 20*time.Second).(type) {
			case *Disconnect:
			case *Error:
			case *Hello:
				// Discard one hello as Hello's are sent concurrently
				continue
			default:
				return fmt.Errorf("unexpected: %s", pretty.Sdump(msg))
			}
		}
		// dial for the next round
		if isEth66 {
			conn, err = s.dial66()
			if err != nil {
				return fmt.Errorf("dial failed: %v", err)
			}
		} else {
			conn, err = s.dial()
			if err != nil {
				return fmt.Errorf("dial failed: %v", err)
			}
		}
	}
	return nil
}

func (s *Suite) maliciousStatus(conn *Conn) error {
	if err := conn.handshake(); err != nil {
		return fmt.Errorf("handshake failed: %v", err)
	}
	status := &Status{
		ProtocolVersion: uint32(conn.negotiatedProtoVersion),
		NetworkID:       s.chain.chainConfig.ChainID.Uint64(),
		TD:              largeNumber(2),
		Head:            s.chain.blocks[s.chain.Len()-1].Hash(),
		Genesis:         s.chain.blocks[0].Hash(),
		ForkID:          s.chain.ForkID(),
	}
	// get status
	msg, err := conn.statusExchange(s.chain, status)
	if err != nil {
		return fmt.Errorf("status exchange failed: %v", err)
	}
	switch msg := msg.(type) {
	case *Status:
	default:
		return fmt.Errorf("expected status, got: %#v ", msg)
	}
	// wait for disconnect
	switch msg := conn.readAndServe(s.chain, timeout).(type) {
	case *Disconnect:
		return nil
	case *Error:
		return nil
	default:
		return fmt.Errorf("expected disconnect, got: %s", pretty.Sdump(msg))
	}
}

func (s *Suite) hashAnnounce(isEth66 bool) error {
	// create connections
	sendConn, recvConn, err := s.createSendAndRecvConns(isEth66)
	if err != nil {
		return fmt.Errorf("failed to create connections: %v", err)
	}
	defer sendConn.Close()
	defer recvConn.Close()
	if err := sendConn.peer(s.chain, nil); err != nil {
		return fmt.Errorf("peering failed: %v", err)
	}
	if err := recvConn.peer(s.chain, nil); err != nil {
		return fmt.Errorf("peering failed: %v", err)
	}
	// create NewBlockHashes announcement
	type anno struct {
		Hash   common.Hash // Hash of one particular block being announced
		Number uint64      // Number of one particular block being announced
	}
	nextBlock := s.fullChain.blocks[s.chain.Len()]
	announcement := anno{Hash: nextBlock.Hash(), Number: nextBlock.Number().Uint64()}
	newBlockHash := &NewBlockHashes{announcement}
	if err := sendConn.Write(newBlockHash); err != nil {
		return fmt.Errorf("failed to write to connection: %v", err)
	}
	// Announcement sent, now wait for a header request
	var (
		id             uint64
		msg            Message
		blockHeaderReq GetBlockHeaders
	)
	if isEth66 {
		id, msg = sendConn.Read66()
		switch msg := msg.(type) {
		case GetBlockHeaders:
			blockHeaderReq = msg
		default:
			return fmt.Errorf("unexpected %s", pretty.Sdump(msg))
		}
		if blockHeaderReq.Amount != 1 {
			return fmt.Errorf("unexpected number of block headers requested: %v", blockHeaderReq.Amount)
		}
		if blockHeaderReq.Origin.Hash != announcement.Hash {
			return fmt.Errorf("unexpected block header requested. Announced:\n %v\n Remote request:\n%v",
				pretty.Sdump(announcement),
				pretty.Sdump(blockHeaderReq))
		}
		if err := sendConn.Write66(&eth.BlockHeadersPacket66{
			RequestId: id,
			BlockHeadersPacket: eth.BlockHeadersPacket{
				nextBlock.Header(),
			},
		}, BlockHeaders{}.Code()); err != nil {
			return fmt.Errorf("failed to write to connection: %v", err)
		}
	} else {
		msg = sendConn.Read()
		switch msg := msg.(type) {
		case *GetBlockHeaders:
			blockHeaderReq = *msg
		default:
			return fmt.Errorf("unexpected %s", pretty.Sdump(msg))
		}
		if blockHeaderReq.Amount != 1 {
			return fmt.Errorf("unexpected number of block headers requested: %v", blockHeaderReq.Amount)
		}
		if blockHeaderReq.Origin.Hash != announcement.Hash {
			return fmt.Errorf("unexpected block header requested. Announced:\n %v\n Remote request:\n%v",
				pretty.Sdump(announcement),
				pretty.Sdump(blockHeaderReq))
		}
		if err := sendConn.Write(&BlockHeaders{nextBlock.Header()}); err != nil {
			return fmt.Errorf("failed to write to connection: %v", err)
		}
	}
	// wait for block announcement
	msg = recvConn.readAndServe(s.chain, timeout)
	switch msg := msg.(type) {
	case *NewBlockHashes:
		hashes := *msg
		if len(hashes) != 1 {
			return fmt.Errorf("unexpected new block hash announcement: wanted 1 announcement, got %d", len(hashes))
		}
		if nextBlock.Hash() != hashes[0].Hash {
			return fmt.Errorf("unexpected block hash announcement, wanted %v, got %v", nextBlock.Hash(),
				hashes[0].Hash)
		}
	case *NewBlock:
		// node should only propagate NewBlock without having requested the body if the body is empty
		nextBlockBody := nextBlock.Body()
		if len(nextBlockBody.Transactions) != 0 || len(nextBlockBody.Uncles) != 0 {
			return fmt.Errorf("unexpected non-empty new block propagated: %s", pretty.Sdump(msg))
		}
		if msg.Block.Hash() != nextBlock.Hash() {
			return fmt.Errorf("mismatched hash of propagated new block: wanted %v, got %v",
				nextBlock.Hash(), msg.Block.Hash())
		}
		// check to make sure header matches header that was sent to the node
		if !reflect.DeepEqual(nextBlock.Header(), msg.Block.Header()) {
			return fmt.Errorf("incorrect header received: wanted %v, got %v", nextBlock.Header(), msg.Block.Header())
		}
	default:
		return fmt.Errorf("unexpected: %s", pretty.Sdump(msg))
	}
	// confirm node imported block
	if err := s.waitForBlockImport(recvConn, nextBlock, isEth66); err != nil {
		return fmt.Errorf("error waiting for node to import new block: %v", err)
	}
	// update the chain
	s.chain.blocks = append(s.chain.blocks, nextBlock)
	return nil
}
