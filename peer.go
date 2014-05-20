package eth

import (
	"bytes"
	"container/list"
	"fmt"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethwire"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	// The size of the output buffer for writing messages
	outputBufferSize = 50
	// Current protocol version
	ProtocolVersion = 8
)

type DiscReason byte

const (
	// Values are given explicitly instead of by iota because these values are
	// defined by the wire protocol spec; it is easier for humans to ensure
	// correctness when values are explicit.
	DiscReRequested  = 0x00
	DiscReTcpSysErr  = 0x01
	DiscBadProto     = 0x02
	DiscBadPeer      = 0x03
	DiscTooManyPeers = 0x04
	DiscConnDup      = 0x05
	DiscGenesisErr   = 0x06
	DiscProtoErr     = 0x07
)

var discReasonToString = []string{
	"Disconnect requested",
	"Disconnect TCP sys error",
	"Disconnect bad protocol",
	"Disconnect useless peer",
	"Disconnect too many peers",
	"Disconnect already connected",
	"Disconnect wrong genesis block",
	"Disconnect incompatible network",
}

func (d DiscReason) String() string {
	if len(discReasonToString) < int(d) {
		return "Unknown"
	}

	return discReasonToString[d]
}

// Peer capabilities
type Caps byte

const (
	CapPeerDiscTy = 1 << iota
	CapTxTy
	CapChainTy

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
	catchingUp      bool
	diverted        bool
	blocksRequested int

	Version string
}

func NewPeer(conn net.Conn, ethereum *Ethereum, inbound bool) *Peer {
	data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
	pubkey := ethutil.NewValueFromBytes(data).Get(2).Bytes()

	return &Peer{
		outputQueue:     make(chan *ethwire.Msg, outputBufferSize),
		quit:            make(chan bool),
		ethereum:        ethereum,
		conn:            conn,
		inbound:         inbound,
		disconnect:      0,
		connected:       1,
		port:            30303,
		pubkey:          pubkey,
		blocksRequested: 10,
		caps:            ethereum.ServerCaps(),
	}
}

func NewOutboundPeer(addr string, ethereum *Ethereum, caps Caps) *Peer {

	p := &Peer{
		outputQueue: make(chan *ethwire.Msg, outputBufferSize),
		quit:        make(chan bool),
		ethereum:    ethereum,
		inbound:     false,
		connected:   0,
		disconnect:  0,
		caps:        caps,
		Version:     ethutil.Config.ClientString,
	}

	// Set up the connection in another goroutine so we don't block the main thread
	go func() {
		conn, err := net.DialTimeout("tcp", addr, 30*time.Second)

		if err != nil {
			ethutil.Config.Log.Debugln("Connection to peer failed", err)
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
	if atomic.LoadInt32(&p.connected) != 1 {
		return
	}
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
		ethutil.Config.Log.Debugln("Can't send message:", err)
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

	for atomic.LoadInt32(&p.disconnect) == 0 {
		// HMM?
		time.Sleep(500 * time.Millisecond)
		// Wait for a message from the peer
		msgs, err := ethwire.ReadMessages(p.conn)
		if err != nil {
			ethutil.Config.Log.Debugln(err)
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
				ethutil.Config.Log.Infoln("Disconnect peer:", DiscReason(msg.Data.Get(0).Uint()))
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
				var block, lastBlock *ethchain.Block
				var err error

				// Make sure we are actually receiving anything
				if msg.Data.Len()-1 > 1 && p.diverted {
					// We requested blocks and now we need to make sure we have a common ancestor somewhere in these blocks so we can find
					// common ground to start syncing from
					lastBlock = ethchain.NewBlockFromRlpValue(msg.Data.Get(msg.Data.Len() - 1))
					ethutil.Config.Log.Infof("[PEER] Last block: %x. Checking if we have it locally.\n", lastBlock.Hash())
					for i := msg.Data.Len() - 1; i >= 0; i-- {
						block = ethchain.NewBlockFromRlpValue(msg.Data.Get(i))
						// Do we have this block on our chain? If so we can continue
						if !p.ethereum.StateManager().BlockChain().HasBlock(block.Hash()) {
							// We don't have this block, but we do have a block with the same prevHash, diversion time!
							if p.ethereum.StateManager().BlockChain().HasBlockWithPrevHash(block.PrevHash) {
								p.diverted = false
								if !p.ethereum.StateManager().BlockChain().FindCanonicalChainFromMsg(msg, block.PrevHash) {
									p.SyncWithPeerToLastKnown()
								}
								break
							}
						}
					}
					if !p.ethereum.StateManager().BlockChain().HasBlock(lastBlock.Hash()) {
						// If we can't find a common ancenstor we need to request more blocks.
						// FIXME: At one point this won't scale anymore since we are not asking for an offset
						// we just keep increasing the amount of blocks.
						p.blocksRequested = p.blocksRequested * 2

						ethutil.Config.Log.Infof("[PEER] No common ancestor found, requesting %d more blocks.\n", p.blocksRequested)
						p.catchingUp = false
						p.FindCommonParentBlock()
						break
					}
				}

				for i := msg.Data.Len() - 1; i >= 0; i-- {
					block = ethchain.NewBlockFromRlpValue(msg.Data.Get(i))

					//p.ethereum.StateManager().PrepareDefault(block)
					state := p.ethereum.StateManager().CurrentState()
					err = p.ethereum.StateManager().ProcessBlock(state, block, false)

					if err != nil {
						if ethutil.Config.Debug {
							ethutil.Config.Log.Infof("[PEER] Block %x failed\n", block.Hash())
							ethutil.Config.Log.Infof("[PEER] %v\n", err)
						}
						break
					} else {
						lastBlock = block
					}
				}

				if msg.Data.Len() == 0 {
					// Set catching up to false if
					// the peer has nothing left to give
					p.catchingUp = false
				}

				if err != nil {
					// If the parent is unknown try to catch up with this peer
					if ethchain.IsParentErr(err) {
						ethutil.Config.Log.Infoln("Attempting to catch up since we don't know the parent")
						p.catchingUp = false
						p.CatchupWithPeer(p.ethereum.BlockChain().CurrentBlock.Hash())
					} else if ethchain.IsValidationErr(err) {
						fmt.Println("Err:", err)
						p.catchingUp = false
					}
				} else {
					// If we're catching up, try to catch up further.
					if p.catchingUp && msg.Data.Len() > 1 {
						if lastBlock != nil {
							blockInfo := lastBlock.BlockInfo()
							ethutil.Config.Log.Debugf("Synced to block height #%d %x %x\n", blockInfo.Number, lastBlock.Hash(), blockInfo.Hash)
						}

						p.catchingUp = false

						hash := p.ethereum.BlockChain().CurrentBlock.Hash()
						p.CatchupWithPeer(hash)
					}
				}

			case ethwire.MsgTxTy:
				// If the message was a transaction queue the transaction
				// in the TxPool where it will undergo validation and
				// processing when a new block is found
				for i := 0; i < msg.Data.Len(); i++ {
					tx := ethchain.NewTransactionFromValue(msg.Data.Get(i))
					p.ethereum.TxPool().QueueTransaction(tx)
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
					peers := make([]string, data.Len())
					// Parse each possible peer
					for i := 0; i < data.Len(); i++ {
						value := data.Get(i)
						peers[i] = unpackAddr(value.Get(0), value.Get(1).Uint())
					}

					// Connect to the list of peers
					p.ethereum.ProcessPeerList(peers)
					// Mark unrequested again
					p.requestedPeerList = false

				}
			case ethwire.MsgGetChainTy:
				var parent *ethchain.Block
				// Length minus one since the very last element in the array is a count
				l := msg.Data.Len() - 1
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
					if data := msg.Data.Get(i).Bytes(); p.ethereum.StateManager().BlockChain().HasBlock(data) {
						parent = p.ethereum.BlockChain().GetBlock(data)
						break
					}
				}

				// If a parent is found send back a reply
				if parent != nil {
					ethutil.Config.Log.Debugf("[PEER] Found conical block, returning chain from: %x ", parent.Hash())
					chain := p.ethereum.BlockChain().GetChainFromHash(parent.Hash(), amountOfBlocks)
					if len(chain) > 0 {
						ethutil.Config.Log.Debugf("[PEER] Returning %d blocks: %x ", len(chain), parent.Hash())
						p.QueueMessage(ethwire.NewMessage(ethwire.MsgBlockTy, chain))
					} else {
						p.QueueMessage(ethwire.NewMessage(ethwire.MsgBlockTy, []interface{}{}))
					}

				} else {
					//ethutil.Config.Log.Debugf("[PEER] Could not find a similar block")
					// If no blocks are found we send back a reply with msg not in chain
					// and the last hash from get chain
					lastHash := msg.Data.Get(l - 1)
					//log.Printf("Sending not in chain with hash %x\n", lastHash.AsRaw())
					p.QueueMessage(ethwire.NewMessage(ethwire.MsgNotInChainTy, []interface{}{lastHash.Raw()}))
				}
			case ethwire.MsgNotInChainTy:
				ethutil.Config.Log.Debugf("Not in chain: %x\n", msg.Data.Get(0).Bytes())
				if p.diverted == true {
					// If were already looking for a common parent and we get here again we need to go deeper
					p.blocksRequested = p.blocksRequested * 2
				}
				p.diverted = true
				p.catchingUp = false
				p.FindCommonParentBlock()
			case ethwire.MsgGetTxsTy:
				// Get the current transactions of the pool
				txs := p.ethereum.TxPool().CurrentTransactions()
				// Get the RlpData values from the txs
				txsInterface := make([]interface{}, len(txs))
				for i, tx := range txs {
					txsInterface[i] = tx.RlpData()
				}
				// Broadcast it back to the peer
				p.QueueMessage(ethwire.NewMessage(ethwire.MsgTxTy, txsInterface))

				// Unofficial but fun nonetheless
			case ethwire.MsgTalkTy:
				ethutil.Config.Log.Infoln("%v says: %s\n", p.conn.RemoteAddr(), msg.Data.Str())
			}
		}
	}
	p.Stop()
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
		ethutil.Config.Log.Debugln("Peer can't send outbound version ack", err)

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

	// Pre-emptively remove the peer; don't wait for reaping. We already know it's dead if we are here
	p.ethereum.peerMut.Lock()
	defer p.ethereum.peerMut.Unlock()
	eachPeer(p.ethereum.peers, func(peer *Peer, e *list.Element) {
		if peer == p {
			p.ethereum.peers.Remove(e)
		}
	})
}

func (p *Peer) pushHandshake() error {
	data, _ := ethutil.Config.Db.Get([]byte("KeyRing"))
	pubkey := ethutil.NewValueFromBytes(data).Get(2).Bytes()

	msg := ethwire.NewMessage(ethwire.MsgHandshakeTy, []interface{}{
		uint32(ProtocolVersion), uint32(0), p.Version, byte(p.caps), p.port, pubkey,
	})

	p.QueueMessage(msg)

	return nil
}

func (p *Peer) peersMessage() *ethwire.Msg {
	outPeers := make([]interface{}, len(p.ethereum.InOutPeers()))
	// Serialise each peer
	for i, peer := range p.ethereum.InOutPeers() {
		// Don't return localhost as valid peer
		if !net.ParseIP(peer.conn.RemoteAddr().String()).IsLoopback() {
			outPeers[i] = peer.RlpData()
		}
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

	if c.Get(0).Uint() != ProtocolVersion {
		ethutil.Config.Log.Debugln("Invalid peer version. Require protocol:", ProtocolVersion)
		p.Stop()
		return
	}

	// [PROTOCOL_VERSION, NETWORK_ID, CLIENT_ID, CAPS, PORT, PUBKEY]
	p.versionKnown = true

	// If this is an inbound connection send an ack back
	if p.inbound {
		p.pubkey = c.Get(5).Bytes()
		p.port = uint16(c.Get(4).Uint())

		// Self connect detection
		keyPair := ethutil.GetKeyRing().Get(0)
		if bytes.Compare(keyPair.PublicKey, p.pubkey) == 0 {
			p.Stop()

			return
		}

	}

	// Set the peer's caps
	p.caps = Caps(c.Get(3).Byte())
	// Get a reference to the peers version
	p.Version = c.Get(2).Str()

	// Catch up with the connected peer
	if !p.ethereum.IsUpToDate() {
		ethutil.Config.Log.Debugln("Already syncing up with a peer; sleeping")
		time.Sleep(10 * time.Second)
	}
	p.SyncWithPeerToLastKnown()

	ethutil.Config.Log.Debugln("[PEER]", p)
}

func (p *Peer) String() string {
	var strBoundType string
	if p.inbound {
		strBoundType = "inbound"
	} else {
		strBoundType = "outbound"
	}
	var strConnectType string
	if atomic.LoadInt32(&p.disconnect) == 0 {
		strConnectType = "connected"
	} else {
		strConnectType = "disconnected"
	}

	return fmt.Sprintf("[%s] (%s) %v %s [%s]", strConnectType, strBoundType, p.conn.RemoteAddr(), p.Version, p.caps)

}
func (p *Peer) SyncWithPeerToLastKnown() {
	p.catchingUp = false
	p.CatchupWithPeer(p.ethereum.BlockChain().CurrentBlock.Hash())
}

func (p *Peer) FindCommonParentBlock() {
	if p.catchingUp {
		return
	}

	p.catchingUp = true
	if p.blocksRequested == 0 {
		p.blocksRequested = 20
	}
	blocks := p.ethereum.BlockChain().GetChain(p.ethereum.BlockChain().CurrentBlock.Hash(), p.blocksRequested)

	var hashes []interface{}
	for _, block := range blocks {
		hashes = append(hashes, block.Hash())
	}

	msgInfo := append(hashes, uint64(len(hashes)))

	ethutil.Config.Log.Infof("Asking for block from %x (%d total) from %s\n", p.ethereum.BlockChain().CurrentBlock.Hash(), len(hashes), p.conn.RemoteAddr().String())

	msg := ethwire.NewMessage(ethwire.MsgGetChainTy, msgInfo)
	p.QueueMessage(msg)
}
func (p *Peer) CatchupWithPeer(blockHash []byte) {
	if !p.catchingUp {
		// Make sure nobody else is catching up when you want to do this
		p.catchingUp = true
		msg := ethwire.NewMessage(ethwire.MsgGetChainTy, []interface{}{blockHash, uint64(50)})
		p.QueueMessage(msg)

		ethutil.Config.Log.Debugf("Requesting blockchain %x... from peer %s\n", p.ethereum.BlockChain().CurrentBlock.Hash()[:4], p.conn.RemoteAddr())

		/*
			msg = ethwire.NewMessage(ethwire.MsgGetTxsTy, []interface{}{})
			p.QueueMessage(msg)
		*/
	}
}

func (p *Peer) RlpData() []interface{} {
	return []interface{}{p.host, p.port, p.pubkey}
}

func packAddr(address, port string) ([]interface{}, uint16) {
	addr := strings.Split(address, ".")
	a, _ := strconv.Atoi(addr[0])
	b, _ := strconv.Atoi(addr[1])
	c, _ := strconv.Atoi(addr[2])
	d, _ := strconv.Atoi(addr[3])
	host := []interface{}{int32(a), int32(b), int32(c), int32(d)}
	prt, _ := strconv.Atoi(port)

	return host, uint16(prt)
}

func unpackAddr(value *ethutil.Value, p uint64) string {
	a := strconv.Itoa(int(value.Get(0).Uint()))
	b := strconv.Itoa(int(value.Get(1).Uint()))
	c := strconv.Itoa(int(value.Get(2).Uint()))
	d := strconv.Itoa(int(value.Get(3).Uint()))
	host := strings.Join([]string{a, b, c, d}, ".")
	port := strconv.Itoa(int(p))

	return net.JoinHostPort(host, port)
}
