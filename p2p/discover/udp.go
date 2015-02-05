package discover

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/rlp"
)

var log = logger.NewLogger("P2P Discovery")

// Errors
var (
	errPacketTooSmall = errors.New("too small")
	errBadHash        = errors.New("bad hash")
	errExpired        = errors.New("expired")
	errTimeout        = errors.New("RPC timeout")
	errClosed         = errors.New("socket closed")
)

// Timeouts
const (
	respTimeout = 300 * time.Millisecond
	sendTimeout = 300 * time.Millisecond
	expiration  = 3 * time.Second

	refreshInterval = 1 * time.Hour
)

// RPC packet types
const (
	pingPacket = iota + 1 // zero is 'reserved'
	pongPacket
	findnodePacket
	neighborsPacket
)

// RPC request structures
type (
	ping struct {
		IP         string // our IP
		Port       uint16 // our port
		Expiration uint64
	}

	// reply to Ping
	pong struct {
		ReplyTok   []byte
		Expiration uint64
	}

	findnode struct {
		// Id to look up. The responding node will send back nodes
		// closest to the target.
		Target     NodeID
		Expiration uint64
	}

	// reply to findnode
	neighbors struct {
		Nodes      []*Node
		Expiration uint64
	}
)

// udp implements the RPC protocol.
type udp struct {
	conn       *net.UDPConn
	priv       *ecdsa.PrivateKey
	addpending chan *pending
	replies    chan reply
	closing    chan struct{}

	*Table
}

// pending represents a pending reply.
//
// some implementations of the protocol wish to send more than one
// reply packet to findnode. in general, any neighbors packet cannot
// be matched up with a specific findnode packet.
//
// our implementation handles this by storing a callback function for
// each pending reply. incoming packets from a node are dispatched
// to all the callback functions for that node.
type pending struct {
	// these fields must match in the reply.
	from  NodeID
	ptype byte

	// time when the request must complete
	deadline time.Time

	// callback is called when a matching reply arrives. if it returns
	// true, the callback is removed from the pending reply queue.
	// if it returns false, the reply is considered incomplete and
	// the callback will be invoked again for the next matching reply.
	callback func(resp interface{}) (done bool)

	// errc receives nil when the callback indicates completion or an
	// error if no further reply is received within the timeout.
	errc chan<- error
}

type reply struct {
	from  NodeID
	ptype byte
	data  interface{}
}

// ListenUDP returns a new table that listens for UDP packets on laddr.
func ListenUDP(priv *ecdsa.PrivateKey, laddr string) (*Table, error) {
	net, realaddr, err := listen(priv, laddr)
	if err != nil {
		return nil, err
	}
	net.Table = newTable(net, newNodeID(priv), realaddr)
	log.DebugDetailf("Listening on %v, my ID %x\n", realaddr, net.self.ID[:])
	return net.Table, nil
}

func listen(priv *ecdsa.PrivateKey, laddr string) (*udp, *net.UDPAddr, error) {
	addr, err := net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		return nil, nil, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, nil, err
	}
	realaddr := conn.LocalAddr().(*net.UDPAddr)

	udp := &udp{
		conn:       conn,
		priv:       priv,
		closing:    make(chan struct{}),
		addpending: make(chan *pending),
		replies:    make(chan reply),
	}
	go udp.loop()
	go udp.readLoop()
	return udp, realaddr, nil
}

func (t *udp) close() {
	close(t.closing)
	t.conn.Close()
	// TODO: wait for the loops to end.
}

// ping sends a ping message to the given node and waits for a reply.
func (t *udp) ping(e *Node) error {
	// TODO: maybe check for ReplyTo field in callback to measure RTT
	errc := t.pending(e.ID, pongPacket, func(interface{}) bool { return true })
	t.send(e, pingPacket, ping{
		IP:         t.self.Addr.String(),
		Port:       uint16(t.self.Addr.Port),
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	})
	return <-errc
}

// findnode sends a findnode request to the given node and waits until
// the node has sent up to k neighbors.
func (t *udp) findnode(to *Node, target NodeID) ([]*Node, error) {
	nodes := make([]*Node, 0, bucketSize)
	nreceived := 0
	errc := t.pending(to.ID, neighborsPacket, func(r interface{}) bool {
		reply := r.(*neighbors)
		for i := 0; i < len(reply.Nodes); i++ {
			nreceived++
			n := reply.Nodes[i]
			if validAddr(n.Addr) && n.ID != t.self.ID {
				nodes = append(nodes, n)
			}
		}
		return nreceived == bucketSize
	})

	t.send(to, findnodePacket, findnode{
		Target:     target,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	})
	err := <-errc
	return nodes, err
}

func validAddr(a *net.UDPAddr) bool {
	return !a.IP.IsMulticast() && !a.IP.IsUnspecified() && a.Port != 0
}

// pending adds a reply callback to the pending reply queue.
// see the documentation of type pending for a detailed explanation.
func (t *udp) pending(id NodeID, ptype byte, callback func(interface{}) bool) <-chan error {
	ch := make(chan error, 1)
	p := &pending{from: id, ptype: ptype, callback: callback, errc: ch}
	select {
	case t.addpending <- p:
		// loop will handle it
	case <-t.closing:
		ch <- errClosed
	}
	return ch
}

// loop runs in its own goroutin. it keeps track of
// the refresh timer and the pending reply queue.
func (t *udp) loop() {
	var (
		pending      []*pending
		nextDeadline time.Time
		timeout      = time.NewTimer(0)
		refresh      = time.NewTicker(refreshInterval)
	)
	<-timeout.C // ignore first timeout
	defer refresh.Stop()
	defer timeout.Stop()

	rearmTimeout := func() {
		if len(pending) == 0 || nextDeadline == pending[0].deadline {
			return
		}
		nextDeadline = pending[0].deadline
		timeout.Reset(nextDeadline.Sub(time.Now()))
	}

	for {
		select {
		case <-refresh.C:
			go t.refresh()

		case <-t.closing:
			for _, p := range pending {
				p.errc <- errClosed
			}
			return

		case p := <-t.addpending:
			p.deadline = time.Now().Add(respTimeout)
			pending = append(pending, p)
			rearmTimeout()

		case reply := <-t.replies:
			// run matching callbacks, remove if they return false.
			for i, p := range pending {
				if reply.from == p.from && reply.ptype == p.ptype && p.callback(reply.data) {
					p.errc <- nil
					copy(pending[i:], pending[i+1:])
					pending = pending[:len(pending)-1]
					i--
				}
			}
			rearmTimeout()

		case now := <-timeout.C:
			// notify and remove callbacks whose deadline is in the past.
			i := 0
			for ; i < len(pending) && now.After(pending[i].deadline); i++ {
				pending[i].errc <- errTimeout
			}
			if i > 0 {
				copy(pending, pending[i:])
				pending = pending[:len(pending)-i]
			}
			rearmTimeout()
		}
	}
}

const (
	macSize  = 256 / 8
	sigSize  = 520 / 8
	headSize = macSize + sigSize // space of packet frame data
)

var headSpace = make([]byte, headSize)

func (t *udp) send(to *Node, ptype byte, req interface{}) error {
	b := new(bytes.Buffer)
	b.Write(headSpace)
	b.WriteByte(ptype)
	if err := rlp.Encode(b, req); err != nil {
		log.Errorln("error encoding packet:", err)
		return err
	}

	packet := b.Bytes()
	sig, err := crypto.Sign(crypto.Sha3(packet[headSize:]), t.priv)
	if err != nil {
		log.Errorln("could not sign packet:", err)
		return err
	}
	copy(packet[macSize:], sig)
	// add the hash to the front. Note: this doesn't protect the
	// packet in any way. Our public key will be part of this hash in
	// the future.
	copy(packet, crypto.Sha3(packet[macSize:]))

	log.DebugDetailf(">>> %v %T %v\n", to.Addr, req, req)
	if _, err = t.conn.WriteToUDP(packet, to.Addr); err != nil {
		log.DebugDetailln("UDP send failed:", err)
	}
	return err
}

// readLoop runs in its own goroutine. it handles incoming UDP packets.
func (t *udp) readLoop() {
	defer t.conn.Close()
	buf := make([]byte, 4096) // TODO: good buffer size
	for {
		nbytes, from, err := t.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		if err := t.packetIn(from, buf[:nbytes]); err != nil {
			log.Debugf("Bad packet from %v: %v\n", from, err)
		}
	}
}

func (t *udp) packetIn(from *net.UDPAddr, buf []byte) error {
	if len(buf) < headSize+1 {
		return errPacketTooSmall
	}
	hash, sig, sigdata := buf[:macSize], buf[macSize:headSize], buf[headSize:]
	shouldhash := crypto.Sha3(buf[macSize:])
	if !bytes.Equal(hash, shouldhash) {
		return errBadHash
	}
	fromID, err := recoverNodeID(crypto.Sha3(buf[headSize:]), sig)
	if err != nil {
		return err
	}

	var req interface {
		handle(t *udp, from *net.UDPAddr, fromID NodeID, mac []byte) error
	}
	switch ptype := sigdata[0]; ptype {
	case pingPacket:
		req = new(ping)
	case pongPacket:
		req = new(pong)
	case findnodePacket:
		req = new(findnode)
	case neighborsPacket:
		req = new(neighbors)
	default:
		return fmt.Errorf("unknown type: %d", ptype)
	}
	if err := rlp.Decode(bytes.NewReader(sigdata[1:]), req); err != nil {
		return err
	}
	log.DebugDetailf("<<< %v %T %v\n", from, req, req)
	return req.handle(t, from, fromID, hash)
}

func (req *ping) handle(t *udp, from *net.UDPAddr, fromID NodeID, mac []byte) error {
	if expired(req.Expiration) {
		return errExpired
	}
	t.mutex.Lock()
	// Note: we're ignoring the provided IP/Port right now.
	e := t.bumpOrAdd(fromID, from)
	t.mutex.Unlock()

	t.send(e, pongPacket, pong{
		ReplyTok:   mac,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	})
	return nil
}

func (req *pong) handle(t *udp, from *net.UDPAddr, fromID NodeID, mac []byte) error {
	if expired(req.Expiration) {
		return errExpired
	}
	t.mutex.Lock()
	t.bump(fromID)
	t.mutex.Unlock()

	t.replies <- reply{fromID, pongPacket, req}
	return nil
}

func (req *findnode) handle(t *udp, from *net.UDPAddr, fromID NodeID, mac []byte) error {
	if expired(req.Expiration) {
		return errExpired
	}
	t.mutex.Lock()
	e := t.bumpOrAdd(fromID, from)
	closest := t.closest(req.Target, bucketSize).entries
	t.mutex.Unlock()

	t.send(e, neighborsPacket, neighbors{
		Nodes:      closest,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	})
	return nil
}

func (req *neighbors) handle(t *udp, from *net.UDPAddr, fromID NodeID, mac []byte) error {
	if expired(req.Expiration) {
		return errExpired
	}
	t.mutex.Lock()
	t.bump(fromID)
	t.add(req.Nodes)
	t.mutex.Unlock()

	t.replies <- reply{fromID, neighborsPacket, req}
	return nil
}

func expired(ts uint64) bool {
	return time.Unix(int64(ts), 0).Before(time.Now())
}
