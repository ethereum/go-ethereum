package eth

import (
	"bytes"
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

type DiscReason byte

const (
	DiscReRequested  = 0x00
	DiscReTcpSysErr  = 0x01
	DiscBadProto     = 0x02
	DiscBadPeer      = 0x03
	DiscTooManyPeers = 0x04
)

var discReasonToString = []string{
	"Disconnect requested",
	"Disconnect TCP sys error",
	"Disconnect Bad protocol",
	"Disconnect Useless peer",
	"Disconnect Too many peers",
}

func (d DiscReason) String() string {
	if len(discReasonToString) > int(d) {
		return "Unknown"
	}

	return discReasonToString[d]
}

// Peer capabilities
type Caps byte

const (
	CapPeerDiscTy = 0x01
	CapTxTy       = 0x02
	CapChainTy    = 0x04

	CapDefault = CapChainTy | CapTxTy | CapPeerDiscTy
)

var capsToString = map[Caps]string{
	CapPeerDiscTy: "Peer discovery",
	CapTxTy:       "Transaction relaying",
	CapChainTy:    "Block chain relaying",
}

func (c Caps) IsCap(cap Caps) bool {
	return c&cap > 0
}

func (c Caps) String() string {
	var caps []string
	if c.IsCap(CapPeerDiscTy) {
		caps = append(caps, capsToString[CapPeerDiscTy])
	}
	if c.IsCap(CapChainTy) {
		caps = append(caps, capsToString[CapChainTy])
	}
	if c.IsCap(CapTxTy) {
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

	host []interface{}
	port uint16
	caps Caps

	pubkey []byte

	// Indicated whether the node is catching up or not
	catchingUp bool
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

func NewOutboundPeer(addr string, ethereum *Ethereum, caps Caps) *Peer {
	data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
	pubkey := ethutil.NewValueFromBytes(data).Get(2).Bytes()

	p := &Peer{
		outputQueue: make(chan *ethwire.Msg, outputBufferSize),
		quit:        make(chan bool),
		ethereum:    ethereum,
		inbound:     false,
		connected:   0,
		disconnect:  0,
		caps:        caps,
		pubkey:      pubkey,
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
	pingTimer := time.NewTicker(2 * time.Minute)
	serviceTimer := time.NewTicker(5 * time.Minute)
out:
	for {
		select {
		// Main message queue. All outbound messages are processed through here
		case msg := <-p.outputQueue:
			p.writeMessage(msg)

			p.lastSend = time.Now()

		// Ping timer sends a ping to the peer each 2 minutes
		case <-pingTimer.C:
			p.writeMessage(ethwire.NewMessage(ethwire.MsgPingTy, ""))

		// Service timer takes care of peer broadcasting, transaction
		// posting or block posting
		case <-serviceTimer.C:
			if p.caps&CapPeerDiscTy > 0 {
				msg := p.peersMessage()
				p.ethereum.BroadcastMsg(msg)
			}

		case <-p.quit:
			// Break out of the for loop if a quit message is posted
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
		// HMM?
		time.Sleep(500 * time.Millisecond)

		// Wait for a message from the peer
		msgs, err := ethwire.ReadMessages(p.conn)
		if err != nil {
			log.Println(err)

			break out
		}
		for _, msg := range msgs {
			switch msg.Type {
			case ethwire.MsgHandshakeTy:
				// Version message
				p.handleHandshake(msg)

				if p.caps.IsCap(CapPeerDiscTy) {
					p.QueueMessage(ethwire.NewMessage(ethwire.MsgGetPeersTy, ""))
				}
			case ethwire.MsgDiscTy:
				p.Stop()
				log.Println("Disconnect peer:", DiscReason(msg.Data.Get(0).AsUint()))
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
					// FIXME
					block := ethchain.NewBlockFromRlpValue(ethutil.NewValue(msg.Data.Get(i).AsRaw()))
					err := p.ethereum.BlockManager.ProcessBlock(block)

					if err != nil {
						log.Println(err)
					} else {
						if p.catchingUp && msg.Data.Length() > 1 {
							p.catchingUp = false
							p.CatchupWithPeer()
						}
					}
				}
			case ethwire.MsgTxTy:
				// If the message was a transaction queue the transaction
				// in the TxPool where it will undergo validation and
				// processing when a new block is found
				for i := 0; i < msg.Data.Length(); i++ {
					p.ethereum.TxPool.QueueTransaction(ethchain.NewTransactionFromData(msg.Data.Get(i).Encode()))
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
				//if p.requestedPeerList {
				data := msg.Data
				// Create new list of possible peers for the ethereum to process
				peers := make([]string, data.Length())
				// Parse each possible peer
				for i := 0; i < data.Length(); i++ {
					peers[i] = unpackAddr(data.Get(i).Get(0), data.Get(i).Get(1).AsUint())
				}

				// Connect to the list of peers
				p.ethereum.ProcessPeerList(peers)
				// Mark unrequested again
				p.requestedPeerList = false

				//}
			case ethwire.MsgGetChainTy:
				var parent *ethchain.Block
				// Length minus one since the very last element in the array is a count
				l := msg.Data.Length() - 1
				// Ignore empty get chains
				if l == 0 {
					break
				}

				// Amount of parents in the canonical chain
				//amountOfBlocks := msg.Data.Get(l).AsUint()
				amountOfBlocks := uint64(100)
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

func packAddr(address, port string) ([]interface{}, uint16) {
	addr := strings.Split(address, ".")
	a, _ := strconv.Atoi(addr[0])
	b, _ := strconv.Atoi(addr[1])
	c, _ := strconv.Atoi(addr[2])
	d, _ := strconv.Atoi(addr[3])
	host := []interface{}{byte(a), byte(b), byte(c), byte(d)}
	prt, _ := strconv.Atoi(port)

	return host, uint16(prt)
}

func unpackAddr(value *ethutil.RlpValue, p uint64) string {
	a := strconv.Itoa(int(value.Get(0).AsUint()))
	b := strconv.Itoa(int(value.Get(1).AsUint()))
	c := strconv.Itoa(int(value.Get(2).AsUint()))
	d := strconv.Itoa(int(value.Get(3).AsUint()))
	host := strings.Join([]string{a, b, c, d}, ".")
	port := strconv.Itoa(int(p))

	return net.JoinHostPort(host, port)
}

func (p *Peer) Start() {
	peerHost, peerPort, _ := net.SplitHostPort(p.conn.LocalAddr().String())
	servHost, servPort, _ := net.SplitHostPort(p.conn.RemoteAddr().String())

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
}

func (p *Peer) pushHandshake() error {
	msg := ethwire.NewMessage(ethwire.MsgHandshakeTy, []interface{}{
		uint32(3), uint32(0), "/Ethereum(G) v0.0.1/", byte(p.caps), p.port, p.pubkey,
	})

	p.QueueMessage(msg)

	return nil
}

func (p *Peer) peersMessage() *ethwire.Msg {
	outPeers := make([]interface{}, len(p.ethereum.InOutPeers()))
	// Serialise each peer
	for i, peer := range p.ethereum.InOutPeers() {
		outPeers[i] = peer.RlpData()
	}

	// Return the message to the peer with the known list of connected clients
	return ethwire.NewMessage(ethwire.MsgPeersTy, outPeers)
}

// Pushes the list of outbound peers to the client when requested
func (p *Peer) pushPeers() {
	p.QueueMessage(p.peersMessage())
}

func (p *Peer) handleHandshake(msg *ethwire.Msg) {
	c := msg.Data

	if c.Get(0).AsUint() != 2 {
		log.Println("Invalid peer version. Require protocol v 2")
		p.Stop()
		return
	}

	// [PROTOCOL_VERSION, NETWORK_ID, CLIENT_ID, CAPS, PORT, PUBKEY]
	p.versionKnown = true

	var istr string
	// If this is an inbound connection send an ack back
	if p.inbound {
		p.pubkey = c.Get(5).AsBytes()
		p.port = uint16(c.Get(4).AsUint())

		// Self connect detection
		data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
		pubkey := ethutil.NewValueFromBytes(data).Get(2).Bytes()
		if bytes.Compare(pubkey, p.pubkey) == 0 {
			p.Stop()

			return
		}

		istr = "inbound"
	} else {
		p.CatchupWithPeer()

		istr = "outbound"
	}

	p.caps = Caps(c.Get(3).AsByte())

	log.Printf("peer connect (%s) %v %s [%s]\n", istr, p.conn.RemoteAddr(), c.Get(2).AsString(), p.caps)
}

func (p *Peer) CatchupWithPeer() {
	if !p.catchingUp {
		p.catchingUp = true
		msg := ethwire.NewMessage(ethwire.MsgGetChainTy, []interface{}{p.ethereum.BlockManager.BlockChain().CurrentBlock.Hash(), uint64(50)})
		p.QueueMessage(msg)

		log.Printf("Requesting blockchain up from %x\n", p.ethereum.BlockManager.BlockChain().CurrentBlock.Hash())
	}
}

func (p *Peer) RlpData() []interface{} {
	return []interface{}{p.host, p.port, p.pubkey}
}
