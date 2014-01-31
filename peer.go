package eth

import (
	"github.com/ethereum/ethchain-go"
	"github.com/ethereum/ethutil-go"
	"github.com/ethereum/ethwire-go"
	"log"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	// The size of the output buffer for writing messages
	outputBufferSize = 50
)

// Peer capabillities
type Caps byte

const (
	CapDiscoveryTy = 0x01
	CapTxTy        = 0x02
	CapChainTy     = 0x04
)

var capsToString = map[Caps]string{
	CapDiscoveryTy: "Peer discovery",
	CapTxTy:        "Transaction relaying",
	CapChainTy:     "Block chain relaying",
}

func (c Caps) String() string {
	var caps []string
	if c&CapDiscoveryTy > 0 {
		caps = append(caps, capsToString[CapDiscoveryTy])
	}
	if c&CapChainTy > 0 {
		caps = append(caps, capsToString[CapChainTy])
	}
	if c&CapTxTy > 0 {
		caps = append(caps, capsToString[CapTxTy])
	}

	return strings.Join(caps, " | ")
}

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

	// Determines whether this is a seed peer
	seed bool

	host []byte
	port uint16
	caps Caps
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
		port:        30303,
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

		p.Start(false)
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

	// XXX TMP CODE FOR TESTNET
	switch msg.Type {
	case ethwire.MsgPeersTy:
		if p.seed {
			p.Stop()
		}
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
		msgs, err := ethwire.ReadMessages(p.conn)
		for _, msg := range msgs {
			if err != nil {
				log.Println(err)

				break out
			}

			switch msg.Type {
			case ethwire.MsgHandshakeTy:
				// Version message
				p.handleHandshake(msg)

				p.QueueMessage(ethwire.NewMessage(ethwire.MsgGetPeersTy, ""))
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
				// Get all blocks and process them
				msg.Data = msg.Data
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
					data := msg.Data
					// Create new list of possible peers for the ethereum to process
					peers := make([]string, data.Length())
					// Parse each possible peer
					for i := 0; i < data.Length(); i++ {
						peers[i] = unpackAddr(data.Get(i).Get(0).AsBytes(), data.Get(i).Get(1).AsUint())
						log.Println(peers[i])
					}

					// Connect to the list of peers
					p.ethereum.ProcessPeerList(peers)
					// Mark unrequested again
					p.requestedPeerList = false

				}
			case ethwire.MsgGetChainTy:
				var parent *ethchain.Block
				// Length minus one since the very last element in the array is a count
				l := msg.Data.Length() - 1
				// Ignore empty get chains
				if l <= 1 {
					break
				}

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
					p.QueueMessage(ethwire.NewMessage(ethwire.MsgBlockTy, append(chain, amountOfBlocks)))
				} else {
					// If no blocks are found we send back a reply with msg not in chain
					// and the last hash from get chain
					lastHash := msg.Data.Get(l - 1)
					log.Printf("Sending not in chain with hash %x\n", lastHash.AsRaw())
					p.QueueMessage(ethwire.NewMessage(ethwire.MsgNotInChainTy, []interface{}{lastHash.AsRaw()}))
				}
			case ethwire.MsgNotInChainTy:
				log.Printf("Not in chain %x\n", msg.Data)
				// TODO

				// Unofficial but fun nonetheless
			case ethwire.MsgTalkTy:
				log.Printf("%v says: %s\n", p.conn.RemoteAddr(), msg.Data.AsString())
			}
		}
	}

	p.Stop()
}

func packAddr(address, port string) ([]byte, uint16) {
	addr := strings.Split(address, ".")
	a, _ := strconv.Atoi(addr[0])
	b, _ := strconv.Atoi(addr[1])
	c, _ := strconv.Atoi(addr[2])
	d, _ := strconv.Atoi(addr[3])
	host := []byte{byte(a), byte(b), byte(c), byte(d)}
	prt, _ := strconv.Atoi(port)

	return host, uint16(prt)
}

func unpackAddr(h []byte, p uint64) string {
	if len(h) != 4 {
		return ""
	}

	a := strconv.Itoa(int(h[0]))
	b := strconv.Itoa(int(h[1]))
	c := strconv.Itoa(int(h[2]))
	d := strconv.Itoa(int(h[3]))
	host := strings.Join([]string{a, b, c, d}, ".")
	port := strconv.Itoa(int(p))

	return net.JoinHostPort(host, port)
}

func (p *Peer) Start(seed bool) {
	p.seed = seed

	peerHost, peerPort, _ := net.SplitHostPort(p.conn.LocalAddr().String())
	servHost, servPort, _ := net.SplitHostPort(p.conn.RemoteAddr().String())
	if peerHost == servHost {
		log.Println("Connected to self")

		//p.Stop()

		//return
	}

	if p.inbound {
		p.host, p.port = packAddr(peerHost, peerPort)
	} else {
		p.host, p.port = packAddr(servHost, servPort)
	}

	err := p.pushHandshake()
	if err != nil {
		log.Printf("Peer can't send outbound version ack", err)

		p.Stop()

		return
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
	msg := ethwire.NewMessage(ethwire.MsgHandshakeTy, []interface{}{
		uint32(0), uint32(0), "/Ethereum(G) v0.0.1/", CapChainTy | CapTxTy | CapDiscoveryTy, p.port,
	})

	p.QueueMessage(msg)

	return nil
}

// Pushes the list of outbound peers to the client when requested
func (p *Peer) pushPeers() {
	outPeers := make([]interface{}, len(p.ethereum.InOutPeers()))
	// Serialise each peer
	for i, peer := range p.ethereum.InOutPeers() {
		outPeers[i] = peer.RlpData()
	}

	// Send message to the peer with the known list of connected clients
	msg := ethwire.NewMessage(ethwire.MsgPeersTy, outPeers)

	p.QueueMessage(msg)
}

func (p *Peer) handleHandshake(msg *ethwire.Msg) {
	c := msg.Data
	// [PROTOCOL_VERSION, NETWORK_ID, CLIENT_ID]
	p.versionKnown = true

	var istr string
	// If this is an inbound connection send an ack back
	if p.inbound {
		if port := c.Get(4).AsUint(); port != 0 {
			p.port = uint16(port)
		}

		istr = "inbound"
	} else {
		msg := ethwire.NewMessage(ethwire.MsgGetChainTy, []interface{}{p.ethereum.BlockManager.BlockChain().CurrentBlock.Hash(), uint64(100)})
		p.QueueMessage(msg)

		istr = "outbound"
	}

	if caps := Caps(c.Get(3).AsByte()); caps != 0 {
		p.caps = caps
	}

	log.Printf("peer connect (%s) %v %s [%s]\n", istr, p.conn.RemoteAddr(), c.Get(2).AsString(), p.caps)
}

func (p *Peer) RlpData() []interface{} {
	return []interface{}{p.host, p.port /*port*/}
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
