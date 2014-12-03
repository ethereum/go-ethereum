package eth

import (
	"bytes"
	"container/list"
	"fmt"
	"math"
	"math/big"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/chain/types"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/wire"
)

var peerlogger = logger.NewLogger("PEER")

const (
	// The size of the output buffer for writing messages
	outputBufferSize = 50
	// Current protocol version
	ProtocolVersion = 46
	// Current P2P version
	P2PVersion = 2
	// Ethereum network version
	NetVersion = 0
	// Interval for ping/pong message
	pingPongTimer = 2 * time.Second
)

type DiscReason byte

const (
	// Values are given explicitly instead of by iota because these values are
	// defined by the wire protocol spec; it is easier for humans to ensure
	// correctness when values are explicit.
	DiscRequested DiscReason = iota
	DiscReTcpSysErr
	DiscBadProto
	DiscBadPeer
	DiscTooManyPeers
	DiscConnDup
	DiscGenesisErr
	DiscProtoErr
	DiscQuitting
)

var discReasonToString = []string{
	"requested",
	"TCP sys error",
	"bad protocol",
	"useless peer",
	"too many peers",
	"already connected",
	"wrong genesis block",
	"incompatible network",
	"quitting",
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
	CapPeerDiscTy Caps = 1 << iota
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
	outputQueue chan *wire.Msg
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
	statusKnown  bool

	// Last received pong message
	lastPong           int64
	lastBlockReceived  time.Time
	doneFetchingHashes bool

	host             []byte
	port             uint16
	caps             Caps
	td               *big.Int
	bestHash         []byte
	lastReceivedHash []byte
	requestedHashes  [][]byte

	// This peer's public key
	pubkey []byte

	// Indicated whether the node is catching up or not
	catchingUp      bool
	diverted        bool
	blocksRequested int

	version string

	// We use this to give some kind of pingtime to a node, not very accurate, could be improved.
	pingTime      time.Duration
	pingStartTime time.Time

	lastRequestedBlock *types.Block

	protocolCaps *ethutil.Value
}

func NewPeer(conn net.Conn, ethereum *Ethereum, inbound bool) *Peer {
	pubkey := ethereum.KeyManager().PublicKey()[1:]

	return &Peer{
		outputQueue:        make(chan *wire.Msg, outputBufferSize),
		quit:               make(chan bool),
		ethereum:           ethereum,
		conn:               conn,
		inbound:            inbound,
		disconnect:         0,
		connected:          1,
		port:               30303,
		pubkey:             pubkey,
		blocksRequested:    10,
		caps:               ethereum.ServerCaps(),
		version:            ethereum.ClientIdentity().String(),
		protocolCaps:       ethutil.NewValue(nil),
		td:                 big.NewInt(0),
		doneFetchingHashes: true,
	}
}

func NewOutboundPeer(addr string, ethereum *Ethereum, caps Caps) *Peer {
	p := &Peer{
		outputQueue:        make(chan *wire.Msg, outputBufferSize),
		quit:               make(chan bool),
		ethereum:           ethereum,
		inbound:            false,
		connected:          0,
		disconnect:         0,
		port:               30303,
		caps:               caps,
		version:            ethereum.ClientIdentity().String(),
		protocolCaps:       ethutil.NewValue(nil),
		td:                 big.NewInt(0),
		doneFetchingHashes: true,
	}

	// Set up the connection in another goroutine so we don't block the main thread
	go func() {
		conn, err := p.Connect(addr)
		if err != nil {
			//peerlogger.Debugln("Connection to peer failed. Giving up.", err)
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

func (self *Peer) Connect(addr string) (conn net.Conn, err error) {
	const maxTries = 3
	for attempts := 0; attempts < maxTries; attempts++ {
		conn, err = net.DialTimeout("tcp", addr, 10*time.Second)
		if err != nil {
			time.Sleep(time.Duration(attempts*20) * time.Second)
			continue
		}

		// Success
		return
	}

	return
}

// Getters
func (p *Peer) PingTime() string {
	return p.pingTime.String()
}
func (p *Peer) Inbound() bool {
	return p.inbound
}
func (p *Peer) LastSend() time.Time {
	return p.lastSend
}
func (p *Peer) LastPong() int64 {
	return p.lastPong
}
func (p *Peer) Host() []byte {
	return p.host
}
func (p *Peer) Port() uint16 {
	return p.port
}
func (p *Peer) Version() string {
	return p.version
}
func (p *Peer) Connected() *int32 {
	return &p.connected
}

// Setters
func (p *Peer) SetVersion(version string) {
	p.version = version
}

// Outputs any RLP encoded data to the peer
func (p *Peer) QueueMessage(msg *wire.Msg) {
	if atomic.LoadInt32(&p.connected) != 1 {
		return
	}
	p.outputQueue <- msg
}

func (p *Peer) writeMessage(msg *wire.Msg) {
	// Ignore the write if we're not connected
	if atomic.LoadInt32(&p.connected) != 1 {
		return
	}

	if !p.versionKnown {
		switch msg.Type {
		case wire.MsgHandshakeTy: // Ok
		default: // Anything but ack is allowed
			return
		}
	} else {
		/*
			if !p.statusKnown {
				switch msg.Type {
				case wire.MsgStatusTy: // Ok
				default: // Anything but ack is allowed
					return
				}
			}
		*/
	}

	peerlogger.DebugDetailf("(%v) <= %v\n", p.conn.RemoteAddr(), formatMessage(msg))

	err := wire.WriteMessage(p.conn, msg)
	if err != nil {
		peerlogger.Debugln(" Can't send message:", err)
		// Stop the client if there was an error writing to it
		p.Stop()
		return
	}
}

// Outbound message handler. Outbound messages are handled here
func (p *Peer) HandleOutbound() {
	// The ping timer. Makes sure that every 2 minutes a ping is send to the peer
	pingTimer := time.NewTicker(pingPongTimer)
	serviceTimer := time.NewTicker(10 * time.Second)

out:
	for {
	skip:
		select {
		// Main message queue. All outbound messages are processed through here
		case msg := <-p.outputQueue:
			if !p.statusKnown {
				switch msg.Type {
				case wire.MsgTxTy, wire.MsgGetBlockHashesTy, wire.MsgBlockHashesTy, wire.MsgGetBlocksTy, wire.MsgBlockTy:
					break skip
				}
			}

			p.writeMessage(msg)
			p.lastSend = time.Now()

		// Ping timer
		case <-pingTimer.C:
			/*
				timeSince := time.Since(time.Unix(p.lastPong, 0))
				if !p.pingStartTime.IsZero() && p.lastPong != 0 && timeSince > (pingPongTimer+30*time.Second) {
					peerlogger.Infof("Peer did not respond to latest pong fast enough, it took %s, disconnecting.\n", timeSince)
					p.Stop()
					return
				}
			*/
			p.writeMessage(wire.NewMessage(wire.MsgPingTy, ""))
			p.pingStartTime = time.Now()

		// Service timer takes care of peer broadcasting, transaction
		// posting or block posting
		case <-serviceTimer.C:
			p.QueueMessage(wire.NewMessage(wire.MsgGetPeersTy, ""))

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

func formatMessage(msg *wire.Msg) (ret string) {
	ret = fmt.Sprintf("%v %v", msg.Type, msg.Data)

	/*
		XXX Commented out because I need the log level here to determine
		if i should or shouldn't generate this message
	*/
	/*
		switch msg.Type {
		case wire.MsgPeersTy:
			ret += fmt.Sprintf("(%d entries)", msg.Data.Len())
		case wire.MsgBlockTy:
			b1, b2 := chain.NewBlockFromRlpValue(msg.Data.Get(0)), ethchain.NewBlockFromRlpValue(msg.Data.Get(msg.Data.Len()-1))
			ret += fmt.Sprintf("(%d entries) %x - %x", msg.Data.Len(), b1.Hash()[0:4], b2.Hash()[0:4])
		case wire.MsgBlockHashesTy:
			h1, h2 := msg.Data.Get(0).Bytes(), msg.Data.Get(msg.Data.Len()-1).Bytes()
			ret += fmt.Sprintf("(%d entries) %x - %x", msg.Data.Len(), h1, h2)
		}
	*/

	return
}

// Inbound handler. Inbound messages are received here and passed to the appropriate methods
func (p *Peer) HandleInbound() {
	for atomic.LoadInt32(&p.disconnect) == 0 {

		// HMM?
		time.Sleep(50 * time.Millisecond)
		// Wait for a message from the peer
		msgs, err := wire.ReadMessages(p.conn)
		if err != nil {
			peerlogger.Debugln(err)
		}
		for _, msg := range msgs {
			peerlogger.DebugDetailf("(%v) => %v\n", p.conn.RemoteAddr(), formatMessage(msg))

			switch msg.Type {
			case wire.MsgHandshakeTy:
				// Version message
				p.handleHandshake(msg)

				//if p.caps.IsCap(CapPeerDiscTy) {
				p.QueueMessage(wire.NewMessage(wire.MsgGetPeersTy, ""))
				//}

			case wire.MsgDiscTy:
				p.Stop()
				peerlogger.Infoln("Disconnect peer: ", DiscReason(msg.Data.Get(0).Uint()))
			case wire.MsgPingTy:
				// Respond back with pong
				p.QueueMessage(wire.NewMessage(wire.MsgPongTy, ""))
			case wire.MsgPongTy:
				// If we received a pong back from a peer we set the
				// last pong so the peer handler knows this peer is still
				// active.
				p.lastPong = time.Now().Unix()
				p.pingTime = time.Since(p.pingStartTime)
			case wire.MsgTxTy:
				// If the message was a transaction queue the transaction
				// in the TxPool where it will undergo validation and
				// processing when a new block is found
				for i := 0; i < msg.Data.Len(); i++ {
					tx := types.NewTransactionFromValue(msg.Data.Get(i))
					err := p.ethereum.TxPool().Add(tx)
					if err != nil {
						peerlogger.Infoln(err)
					} else {
						peerlogger.Infof("tx OK (%x)\n", tx.Hash()[0:4])
					}
				}
			case wire.MsgGetPeersTy:
				// Peer asked for list of connected peers
				//p.pushPeers()
			case wire.MsgPeersTy:
				// Received a list of peers (probably because MsgGetPeersTy was send)
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

			case wire.MsgStatusTy:
				// Handle peer's status msg
				p.handleStatus(msg)
			}

			// TMP
			if p.statusKnown {
				switch msg.Type {
				/*
					case wire.MsgGetTxsTy:
						// Get the current transactions of the pool
						txs := p.ethereum.TxPool().CurrentTransactions()
						// Get the RlpData values from the txs
						txsInterface := make([]interface{}, len(txs))
						for i, tx := range txs {
							txsInterface[i] = tx.RlpData()
						}
						// Broadcast it back to the peer
						p.QueueMessage(wire.NewMessage(wire.MsgTxTy, txsInterface))
				*/

				case wire.MsgGetBlockHashesTy:
					if msg.Data.Len() < 2 {
						peerlogger.Debugln("err: argument length invalid ", msg.Data.Len())
					}

					hash := msg.Data.Get(0).Bytes()
					amount := msg.Data.Get(1).Uint()

					hashes := p.ethereum.ChainManager().GetChainHashesFromHash(hash, amount)

					p.QueueMessage(wire.NewMessage(wire.MsgBlockHashesTy, ethutil.ByteSliceToInterface(hashes)))

				case wire.MsgGetBlocksTy:
					// Limit to max 300 blocks
					max := int(math.Min(float64(msg.Data.Len()), 300.0))
					var blocks []interface{}

					for i := 0; i < max; i++ {
						hash := msg.Data.Get(i).Bytes()
						block := p.ethereum.ChainManager().GetBlock(hash)
						if block != nil {
							blocks = append(blocks, block.Value().Raw())
						}
					}

					p.QueueMessage(wire.NewMessage(wire.MsgBlockTy, blocks))

				case wire.MsgBlockHashesTy:
					p.catchingUp = true

					blockPool := p.ethereum.blockPool

					foundCommonHash := false

					it := msg.Data.NewIterator()
					for it.Next() {
						hash := it.Value().Bytes()
						p.lastReceivedHash = hash

						if blockPool.HasCommonHash(hash) {
							foundCommonHash = true

							break
						}

						blockPool.AddHash(hash, p)
					}

					if !foundCommonHash {
						//if !p.FetchHashes() {
						//	p.doneFetchingHashes = true
						//}
						p.FetchHashes()
					} else {
						peerlogger.Infof("Found common hash (%x...)\n", p.lastReceivedHash[0:4])
						p.doneFetchingHashes = true
					}

				case wire.MsgBlockTy:
					p.catchingUp = true

					blockPool := p.ethereum.blockPool

					it := msg.Data.NewIterator()
					for it.Next() {
						block := types.NewBlockFromRlpValue(it.Value())
						blockPool.Add(block, p)

						p.lastBlockReceived = time.Now()
					}
				case wire.MsgNewBlockTy:
					var (
						blockPool = p.ethereum.blockPool
						block     = types.NewBlockFromRlpValue(msg.Data.Get(0))
						td        = msg.Data.Get(1).BigInt()
					)

					if td.Cmp(blockPool.td) > 0 {
						p.ethereum.blockPool.AddNew(block, p)
					}
				}

			}
		}
	}

	p.Stop()
}

func (self *Peer) FetchBlocks(hashes [][]byte) {
	if len(hashes) > 0 {
		peerlogger.Debugf("Fetching blocks (%d)\n", len(hashes))

		self.QueueMessage(wire.NewMessage(wire.MsgGetBlocksTy, ethutil.ByteSliceToInterface(hashes)))
	}
}

func (self *Peer) FetchHashes() bool {
	blockPool := self.ethereum.blockPool

	return blockPool.FetchHashes(self)
}

func (self *Peer) FetchingHashes() bool {
	return !self.doneFetchingHashes
}

// General update method
func (self *Peer) update() {
	serviceTimer := time.NewTicker(100 * time.Millisecond)

out:
	for {
		select {
		case <-serviceTimer.C:
			if self.IsCap("eth") {
				var (
					sinceBlock = time.Since(self.lastBlockReceived)
				)

				if sinceBlock > 5*time.Second {
					self.catchingUp = false
				}
			}
		case <-self.quit:
			break out
		}
	}

	serviceTimer.Stop()
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
		peerlogger.Debugln("Peer can't send outbound version ack", err)

		p.Stop()

		return
	}

	go p.HandleOutbound()
	// Run the inbound handler in a new goroutine
	go p.HandleInbound()
	// Run the general update handler
	go p.update()

	// Wait a few seconds for startup and then ask for an initial ping
	time.Sleep(2 * time.Second)
	p.writeMessage(wire.NewMessage(wire.MsgPingTy, ""))
	p.pingStartTime = time.Now()

}

func (p *Peer) Stop() {
	p.StopWithReason(DiscRequested)
}

func (p *Peer) StopWithReason(reason DiscReason) {
	if atomic.AddInt32(&p.disconnect, 1) != 1 {
		return
	}

	// Pre-emptively remove the peer; don't wait for reaping. We already know it's dead if we are here
	p.ethereum.RemovePeer(p)

	close(p.quit)
	if atomic.LoadInt32(&p.connected) != 0 {
		p.writeMessage(wire.NewMessage(wire.MsgDiscTy, reason))
		p.conn.Close()
	}
}

func (p *Peer) peersMessage() *wire.Msg {
	outPeers := make([]interface{}, len(p.ethereum.InOutPeers()))
	// Serialise each peer
	for i, peer := range p.ethereum.InOutPeers() {
		// Don't return localhost as valid peer
		if !net.ParseIP(peer.conn.RemoteAddr().String()).IsLoopback() {
			outPeers[i] = peer.RlpData()
		}
	}

	// Return the message to the peer with the known list of connected clients
	return wire.NewMessage(wire.MsgPeersTy, outPeers)
}

// Pushes the list of outbound peers to the client when requested
func (p *Peer) pushPeers() {
	p.QueueMessage(p.peersMessage())
}

func (self *Peer) pushStatus() {
	msg := wire.NewMessage(wire.MsgStatusTy, []interface{}{
		uint32(ProtocolVersion),
		uint32(NetVersion),
		self.ethereum.ChainManager().TD,
		self.ethereum.ChainManager().CurrentBlock.Hash(),
		self.ethereum.ChainManager().Genesis().Hash(),
	})

	self.QueueMessage(msg)
}

func (self *Peer) handleStatus(msg *wire.Msg) {
	c := msg.Data

	var (
		//protoVersion = c.Get(0).Uint()
		netVersion = c.Get(1).Uint()
		td         = c.Get(2).BigInt()
		bestHash   = c.Get(3).Bytes()
		genesis    = c.Get(4).Bytes()
	)

	if bytes.Compare(self.ethereum.ChainManager().Genesis().Hash(), genesis) != 0 {
		loggerger.Warnf("Invalid genisis hash %x. Disabling [eth]\n", genesis)
		return
	}

	if netVersion != NetVersion {
		loggerger.Warnf("Invalid network version %d. Disabling [eth]\n", netVersion)
		return
	}

	/*
		if protoVersion != ProtocolVersion {
			loggerger.Warnf("Invalid protocol version %d. Disabling [eth]\n", protoVersion)
			return
		}
	*/

	// Get the td and last hash
	self.td = td
	self.bestHash = bestHash
	self.lastReceivedHash = bestHash

	self.statusKnown = true

	// Compare the total TD with the blockchain TD. If remote is higher
	// fetch hashes from highest TD node.
	self.FetchHashes()

	loggerger.Infof("Peer is [eth] capable. (TD = %v ~ %x)", self.td, self.bestHash)

}

func (p *Peer) pushHandshake() error {
	pubkey := p.ethereum.KeyManager().PublicKey()
	msg := wire.NewMessage(wire.MsgHandshakeTy, []interface{}{
		P2PVersion, []byte(p.version), []interface{}{[]interface{}{"eth", ProtocolVersion}}, p.port, pubkey[1:],
	})

	p.QueueMessage(msg)

	return nil
}

func (p *Peer) handleHandshake(msg *wire.Msg) {
	c := msg.Data

	var (
		p2pVersion = c.Get(0).Uint()
		clientId   = c.Get(1).Str()
		caps       = c.Get(2)
		port       = c.Get(3).Uint()
		pub        = c.Get(4).Bytes()
	)

	// Check correctness of p2p protocol version
	if p2pVersion != P2PVersion {
		fmt.Println(p)
		peerlogger.Debugf("Invalid P2P version. Require protocol %d, received %d\n", P2PVersion, p2pVersion)
		p.Stop()
		return
	}

	// Handle the pub key (validation, uniqueness)
	if len(pub) == 0 {
		peerlogger.Warnln("Pubkey required, not supplied in handshake.")
		p.Stop()
		return
	}

	// Self connect detection
	pubkey := p.ethereum.KeyManager().PublicKey()
	if bytes.Compare(pubkey[1:], pub) == 0 {
		p.Stop()

		return
	}

	// Check for blacklisting
	for _, pk := range p.ethereum.blacklist {
		if bytes.Compare(pk, pub) == 0 {
			peerlogger.Debugf("Blacklisted peer tried to connect (%x...)\n", pubkey[0:4])
			p.StopWithReason(DiscBadPeer)

			return
		}
	}

	usedPub := 0
	// This peer is already added to the peerlist so we expect to find a double pubkey at least once
	eachPeer(p.ethereum.Peers(), func(peer *Peer, e *list.Element) {
		if bytes.Compare(pub, peer.pubkey) == 0 {
			usedPub++
		}
	})

	if usedPub > 0 {
		peerlogger.Debugf("Pubkey %x found more then once. Already connected to client.", p.pubkey)
		p.Stop()
		return
	}
	p.pubkey = pub

	// If this is an inbound connection send an ack back
	if p.inbound {
		p.port = uint16(port)
	}

	p.SetVersion(clientId)

	p.versionKnown = true

	p.ethereum.PushPeer(p)
	p.ethereum.eventMux.Post(PeerListEvent{p.ethereum.Peers()})

	p.protocolCaps = caps

	it := caps.NewIterator()
	var capsStrs []string
	for it.Next() {
		cap := it.Value().Get(0).Str()
		ver := it.Value().Get(1).Uint()
		switch cap {
		case "eth":
			if ver != ProtocolVersion {
				loggerger.Warnf("Invalid protocol version %d. Disabling [eth]\n", ver)
				continue
			}
			p.pushStatus()
		}

		capsStrs = append(capsStrs, fmt.Sprintf("%s/%d", cap, ver))
	}

	peerlogger.Infof("Added peer (%s) %d / %d (%v)\n", p.conn.RemoteAddr(), p.ethereum.Peers().Len(), p.ethereum.MaxPeers, capsStrs)

	peerlogger.Debugln(p)
}

func (self *Peer) IsCap(cap string) bool {
	capsIt := self.protocolCaps.NewIterator()
	for capsIt.Next() {
		if capsIt.Value().Str() == cap {
			return true
		}
	}

	return false
}

func (self *Peer) Caps() *ethutil.Value {
	return self.protocolCaps
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

	return fmt.Sprintf("[%s] (%s) %v %s", strConnectType, strBoundType, p.conn.RemoteAddr(), p.version)

}

func (p *Peer) RlpData() []interface{} {
	return []interface{}{p.host, p.port, p.pubkey}
}

func packAddr(address, _port string) (host []byte, port uint16) {
	p, _ := strconv.Atoi(_port)
	port = uint16(p)

	h := net.ParseIP(address)
	if ip := h.To4(); ip != nil {
		host = []byte(ip)
	} else {
		host = []byte(h)
	}

	return
}

func unpackAddr(value *ethutil.Value, p uint64) string {
	host, _ := net.IP(value.Bytes()).MarshalText()
	prt := strconv.Itoa(int(p))

	return net.JoinHostPort(string(host), prt)
}
