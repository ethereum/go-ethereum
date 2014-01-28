package eth

import (
	"github.com/ethereum/ethchain-go"
	"github.com/ethereum/ethutil-go"
	"github.com/ethereum/ethwire-go"
	"log"
	"net"
	"strconv"
	"sync/atomic"
	"time"
)

const (
	// The size of the output buffer for writing messages
	outputBufferSize = 50
)

type Peer struct {
	// Ethereum interface
	ethereum *Ethereum
	// Net connection
	conn net.Conn
	// Output queue which is used to communicate and handle messages
	outputQueue chan *ethwire.Msg
	// Quit channel
	quit chan bool
	// Determines whether it's an inbound or outbound peer
	inbound bool
	// Flag for checking the peer's connectivity state
	connected  int32
	disconnect int32
	// Last known message send
	lastSend time.Time
	// Indicated whether a verack has been send or not
	// This flag is used by writeMessage to check if messages are allowed
	// to be send or not. If no version is known all messages are ignored.
	versionKnown bool

	// Last received pong message
	lastPong int64
	// Indicates whether a MsgGetPeersTy was requested of the peer
	// this to prevent receiving false peers.
	requestedPeerList bool
}

func NewPeer(conn net.Conn, ethereum *Ethereum, inbound bool) *Peer {
	return &Peer{
		outputQueue: make(chan *ethwire.Msg, outputBufferSize),
		quit:        make(chan bool),
		ethereum:    ethereum,
		conn:        conn,
		inbound:     inbound,
		disconnect:  0,
		connected:   1,
	}
}

func NewOutboundPeer(addr string, ethereum *Ethereum) *Peer {
	p := &Peer{
		outputQueue: make(chan *ethwire.Msg, outputBufferSize),
		quit:        make(chan bool),
		ethereum:    ethereum,
		inbound:     false,
		connected:   0,
		disconnect:  0,
	}

	// Set up the connection in another goroutine so we don't block the main thread
	go func() {
		conn, err := net.DialTimeout("tcp", addr, 30*time.Second)

		if err != nil {
			log.Println("Connection to peer failed", err)
			p.Stop()
			return
		}
		p.conn = conn

		// Atomically set the connection state
		atomic.StoreInt32(&p.connected, 1)
		atomic.StoreInt32(&p.disconnect, 0)

		log.Println("Connected to peer ::", conn.RemoteAddr())

		p.Start()
	}()

	return p
}

// Outputs any RLP encoded data to the peer
func (p *Peer) QueueMessage(msg *ethwire.Msg) {
	p.outputQueue <- msg
}

func (p *Peer) writeMessage(msg *ethwire.Msg) {
	// Ignore the write if we're not connected
	if atomic.LoadInt32(&p.connected) != 1 {
		return
	}

	if !p.versionKnown {
		switch msg.Type {
		case ethwire.MsgHandshakeTy: // Ok
		default: // Anything but ack is allowed
			return
		}
	}

	err := ethwire.WriteMessage(p.conn, msg)
	if err != nil {
		log.Println("Can't send message:", err)
		// Stop the client if there was an error writing to it
		p.Stop()
		return
	}
}

// Outbound message handler. Outbound messages are handled here
func (p *Peer) HandleOutbound() {
	// The ping timer. Makes sure that every 2 minutes a ping is send to the peer
	tickleTimer := time.NewTicker(2 * time.Minute)
out:
	for {
		select {
		// Main message queue. All outbound messages are processed through here
		case msg := <-p.outputQueue:
			p.writeMessage(msg)

			p.lastSend = time.Now()

		case <-tickleTimer.C:
			p.writeMessage(ethwire.NewMessage(ethwire.MsgPingTy, ""))

		// Break out of the for loop if a quit message is posted
		case <-p.quit:
			break out
		}
	}

clean:
	// This loop is for draining the output queue and anybody waiting for us
	for {
		select {
		case <-p.outputQueue:
			// TODO
		default:
			break clean
		}
	}
}

// Inbound handler. Inbound messages are received here and passed to the appropriate methods
func (p *Peer) HandleInbound() {

out:
	for atomic.LoadInt32(&p.disconnect) == 0 {
		// Wait for a message from the peer
		msg, err := ethwire.ReadMessage(p.conn)
		if err != nil {
			log.Println(err)

			break out
		}

		if ethutil.Config.Debug {
			log.Printf("Received %s\n", msg.Type.String())
		}

		switch msg.Type {
		case ethwire.MsgHandshakeTy:
			// Version message
			p.handleHandshake(msg)
		case ethwire.MsgDiscTy:
			p.Stop()
		case ethwire.MsgPingTy:
			// Respond back with pong
			p.QueueMessage(ethwire.NewMessage(ethwire.MsgPongTy, ""))
		case ethwire.MsgPongTy:
			// If we received a pong back from a peer we set the
			// last pong so the peer handler knows this peer is still
			// active.
			p.lastPong = time.Now().Unix()
		case ethwire.MsgBlockTy:
			// Get all blocks and process them (TODO reverse order?)
			msg.Data = msg.Data.Get(0)
			for i := msg.Data.Length() - 1; i >= 0; i-- {
				block := ethchain.NewBlockFromRlpValue(msg.Data.Get(i))
				err := p.ethereum.BlockManager.ProcessBlock(block)

				if err != nil {
					log.Println(err)
				}
			}
		case ethwire.MsgTxTy:
			// If the message was a transaction queue the transaction
			// in the TxPool where it will undergo validation and
			// processing when a new block is found
			for i := 0; i < msg.Data.Length(); i++ {
				p.ethereum.TxPool.QueueTransaction(ethchain.NewTransactionFromRlpValue(msg.Data.Get(i)))
			}
		case ethwire.MsgGetPeersTy:
			// Flag this peer as a 'requested of new peers' this to
			// prevent malicious peers being forced.
			p.requestedPeerList = true
			// Peer asked for list of connected peers
			p.pushPeers()
		case ethwire.MsgPeersTy:
			// Received a list of peers (probably because MsgGetPeersTy was send)
			// Only act on message if we actually requested for a peers list
			if p.requestedPeerList {
				data := ethutil.Conv(msg.Data)
				// Create new list of possible peers for the ethereum to process
				peers := make([]string, data.Length())
				// Parse each possible peer
				for i := 0; i < data.Length(); i++ {
					peers[i] = data.Get(i).AsString() + strconv.Itoa(int(data.Get(i).AsUint()))
				}

				// Connect to the list of peers
				p.ethereum.ProcessPeerList(peers)
				// Mark unrequested again
				p.requestedPeerList = false
			}
		case ethwire.MsgGetChainTy:
			var parent *ethchain.Block
			// FIXME
			msg.Data = msg.Data.Get(0)
			// Length minus one since the very last element in the array is a count
			l := msg.Data.Length() - 1
			// Amount of parents in the canonical chain
			amountOfBlocks := msg.Data.Get(l).AsUint()
			// Check each SHA block hash from the message and determine whether
			// the SHA is in the database
			for i := 0; i < l; i++ {
				if data := msg.Data.Get(i).AsBytes(); p.ethereum.BlockManager.BlockChain().HasBlock(data) {
					parent = p.ethereum.BlockManager.BlockChain().GetBlock(data)
					break
				}
			}

			// If a parent is found send back a reply
			if parent != nil {
				chain := p.ethereum.BlockManager.BlockChain().GetChainFromHash(parent.Hash(), amountOfBlocks)
				p.QueueMessage(ethwire.NewMessage(ethwire.MsgBlockTy, chain))
			} else {
				// If no blocks are found we send back a reply with msg not in chain
				// and the last hash from get chain
				lastHash := msg.Data.Get(l)
				p.QueueMessage(ethwire.NewMessage(ethwire.MsgNotInChainTy, lastHash.AsRaw()))
			}
		case ethwire.MsgNotInChainTy:
			log.Println("Not in chain, not yet implemented")
			// TODO

		// Unofficial but fun nonetheless
		case ethwire.MsgTalkTy:
			log.Printf("%v says: %s\n", p.conn.RemoteAddr(), msg.Data.Get(0).AsString())
		}
	}

	p.Stop()
}

func (p *Peer) Start() {
	if !p.inbound {
		err := p.pushHandshake()
		if err != nil {
			log.Printf("Peer can't send outbound version ack", err)

			p.Stop()
		}
	}

	// Run the outbound handler in a new goroutine
	go p.HandleOutbound()
	// Run the inbound handler in a new goroutine
	go p.HandleInbound()
}

func (p *Peer) Stop() {
	if atomic.AddInt32(&p.disconnect, 1) != 1 {
		return
	}

	close(p.quit)
	if atomic.LoadInt32(&p.connected) != 0 {
		p.writeMessage(ethwire.NewMessage(ethwire.MsgDiscTy, ""))
		p.conn.Close()
	}

	log.Println("Peer shutdown")
}

func (p *Peer) pushHandshake() error {
	msg := ethwire.NewMessage(ethwire.MsgHandshakeTy, ethutil.Encode([]interface{}{
		1, 0, p.ethereum.Nonce,
	}))

	p.QueueMessage(msg)

	return nil
}

// Pushes the list of outbound peers to the client when requested
func (p *Peer) pushPeers() {
	outPeers := make([]interface{}, len(p.ethereum.OutboundPeers()))
	// Serialise each peer
	for i, peer := range p.ethereum.OutboundPeers() {
		outPeers[i] = peer.RlpEncode()
	}

	// Send message to the peer with the known list of connected clients
	msg := ethwire.NewMessage(ethwire.MsgPeersTy, ethutil.Encode(outPeers))

	p.QueueMessage(msg)
}

func (p *Peer) handleHandshake(msg *ethwire.Msg) {
	c := msg.Data
	// [PROTOCOL_VERSION, NETWORK_ID, CLIENT_ID]
	if c.Get(2).AsUint() == p.ethereum.Nonce {
		//if msg.Nonce == p.ethereum.Nonce {
		log.Println("Peer connected to self, disconnecting")

		p.Stop()

		return
	}

	p.versionKnown = true

	// If this is an inbound connection send an ack back
	if p.inbound {
		err := p.pushHandshake()
		if err != nil {
			log.Println("Peer can't send ack back")

			p.Stop()
		}
	} else {
		msg := ethwire.NewMessage(ethwire.MsgGetChainTy, []interface{}{p.ethereum.BlockManager.BlockChain().CurrentBlock.Hash(), uint64(100)})
		p.QueueMessage(msg)
	}
}

func (p *Peer) RlpEncode() []byte {
	host, prt, err := net.SplitHostPort(p.conn.RemoteAddr().String())
	if err != nil {
		return nil
	}

	i, err := strconv.Atoi(prt)
	if err != nil {
		return nil
	}

	port := ethutil.NumberToBytes(uint16(i), 16)

	return ethutil.Encode([]interface{}{host, port})
}
