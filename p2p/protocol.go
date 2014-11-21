package p2p

import (
	"bytes"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/ethutil"
)

// Protocol represents a P2P subprotocol implementation.
type Protocol struct {
	// Name should contain the official protocol name,
	// often a three-letter word.
	Name string

	// Version should contain the version number of the protocol.
	Version uint

	// Length should contain the number of message codes used
	// by the protocol.
	Length uint64

	// Run is called in a new groutine when the protocol has been
	// negotiated with a peer. It should read and write messages from
	// rw. The Payload for each message must be fully consumed.
	//
	// The peer connection is closed when Start returns. It should return
	// any protocol-level error (such as an I/O error) that is
	// encountered.
	Run func(peer *Peer, rw MsgReadWriter) error
}

func (p Protocol) cap() Cap {
	return Cap{p.Name, p.Version}
}

const (
	baseProtocolVersion    = 2
	baseProtocolLength     = uint64(16)
	baseProtocolMaxMsgSize = 10 * 1024 * 1024
)

const (
	// devp2p message codes
	handshakeMsg = 0x00
	discMsg      = 0x01
	pingMsg      = 0x02
	pongMsg      = 0x03
	getPeersMsg  = 0x04
	peersMsg     = 0x05
)

// handshake is the structure of a handshake list.
type handshake struct {
	Version    uint64
	ID         string
	Caps       []Cap
	ListenPort uint64
	NodeID     []byte
}

func (h *handshake) String() string {
	return h.ID
}
func (h *handshake) Pubkey() []byte {
	return h.NodeID
}

// Cap is the structure of a peer capability.
type Cap struct {
	Name    string
	Version uint
}

func (cap Cap) RlpData() interface{} {
	return []interface{}{cap.Name, cap.Version}
}

type capsByName []Cap

func (cs capsByName) Len() int           { return len(cs) }
func (cs capsByName) Less(i, j int) bool { return cs[i].Name < cs[j].Name }
func (cs capsByName) Swap(i, j int)      { cs[i], cs[j] = cs[j], cs[i] }

type baseProtocol struct {
	rw   MsgReadWriter
	peer *Peer
}

func runBaseProtocol(peer *Peer, rw MsgReadWriter) error {
	bp := &baseProtocol{rw, peer}

	// do handshake
	if err := rw.WriteMsg(bp.handshakeMsg()); err != nil {
		return err
	}
	msg, err := rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != handshakeMsg {
		return newPeerError(errProtocolBreach, "first message must be handshake, got %x", msg.Code)
	}
	data, err := msg.Data()
	if err != nil {
		return newPeerError(errInvalidMsg, "%v", err)
	}
	if err := bp.handleHandshake(data); err != nil {
		return err
	}

	// run main loop
	quit := make(chan error, 1)
	go func() {
		quit <- MsgLoop(rw, baseProtocolMaxMsgSize, bp.handle)
	}()
	return bp.loop(quit)
}

var pingTimeout = 2 * time.Second

func (bp *baseProtocol) loop(quit <-chan error) error {
	ping := time.NewTimer(pingTimeout)
	activity := bp.peer.activity.Subscribe(time.Time{})
	lastActive := time.Time{}
	defer ping.Stop()
	defer activity.Unsubscribe()

	getPeersTick := time.NewTicker(10 * time.Second)
	defer getPeersTick.Stop()
	err := bp.rw.EncodeMsg(getPeersMsg)

	for err == nil {
		select {
		case err = <-quit:
			return err
		case <-getPeersTick.C:
			err = bp.rw.EncodeMsg(getPeersMsg)
		case event := <-activity.Chan():
			ping.Reset(pingTimeout)
			lastActive = event.(time.Time)
		case t := <-ping.C:
			if lastActive.Add(pingTimeout * 2).Before(t) {
				err = newPeerError(errPingTimeout, "")
			} else if lastActive.Add(pingTimeout).Before(t) {
				err = bp.rw.EncodeMsg(pingMsg)
			}
		}
	}
	return err
}

func (bp *baseProtocol) handle(code uint64, data *ethutil.Value) error {
	switch code {
	case handshakeMsg:
		return newPeerError(errProtocolBreach, "extra handshake received")

	case discMsg:
		bp.peer.Disconnect(DiscReason(data.Get(0).Uint()))
		return nil

	case pingMsg:
		return bp.rw.EncodeMsg(pongMsg)

	case pongMsg:

	case getPeersMsg:
		peers := bp.peerList()
		// this is dangerous. the spec says that we should _delay_
		// sending the response if no new information is available.
		// this means that would need to send a response later when
		// new peers become available.
		//
		// TODO: add event mechanism to notify baseProtocol for new peers
		if len(peers) > 0 {
			return bp.rw.EncodeMsg(peersMsg, peers)
		}

	case peersMsg:
		bp.handlePeers(data)

	default:
		return newPeerError(errInvalidMsgCode, "unknown message code %v", code)
	}
	return nil
}

func (bp *baseProtocol) handlePeers(data *ethutil.Value) {
	it := data.NewIterator()
	for it.Next() {
		addr := &peerAddr{
			IP:     net.IP(it.Value().Get(0).Bytes()),
			Port:   it.Value().Get(1).Uint(),
			Pubkey: it.Value().Get(2).Bytes(),
		}
		bp.peer.Debugf("received peer suggestion: %v", addr)
		bp.peer.newPeerAddr <- addr
	}
}

func (bp *baseProtocol) handleHandshake(c *ethutil.Value) error {
	hs := handshake{
		Version:    c.Get(0).Uint(),
		ID:         c.Get(1).Str(),
		Caps:       nil, // decoded below
		ListenPort: c.Get(3).Uint(),
		NodeID:     c.Get(4).Bytes(),
	}
	if hs.Version != baseProtocolVersion {
		return newPeerError(errP2PVersionMismatch, "Require protocol %d, received %d\n",
			baseProtocolVersion, hs.Version)
	}
	if len(hs.NodeID) == 0 {
		return newPeerError(errPubkeyMissing, "")
	}
	if len(hs.NodeID) != 64 {
		return newPeerError(errPubkeyInvalid, "require 512 bit, got %v", len(hs.NodeID)*8)
	}
	if da := bp.peer.dialAddr; da != nil {
		// verify that the peer we wanted to connect to
		// actually holds the target public key.
		if da.Pubkey != nil && !bytes.Equal(da.Pubkey, hs.NodeID) {
			return newPeerError(errPubkeyForbidden, "dial address pubkey mismatch")
		}
	}
	pa := newPeerAddr(bp.peer.conn.RemoteAddr(), hs.NodeID)
	if err := bp.peer.pubkeyHook(pa); err != nil {
		return newPeerError(errPubkeyForbidden, "%v", err)
	}
	capsIt := c.Get(2).NewIterator()
	for capsIt.Next() {
		cap := capsIt.Value()
		name := cap.Get(0).Str()
		if name != "" {
			hs.Caps = append(hs.Caps, Cap{Name: name, Version: uint(cap.Get(1).Uint())})
		}
	}

	var addr *peerAddr
	if hs.ListenPort != 0 {
		addr = newPeerAddr(bp.peer.conn.RemoteAddr(), hs.NodeID)
		addr.Port = hs.ListenPort
	}
	bp.peer.setHandshakeInfo(&hs, addr, hs.Caps)
	bp.peer.startSubprotocols(hs.Caps)
	return nil
}

func (bp *baseProtocol) handshakeMsg() Msg {
	var (
		port uint64
		caps []interface{}
	)
	if bp.peer.ourListenAddr != nil {
		port = bp.peer.ourListenAddr.Port
	}
	for _, proto := range bp.peer.protocols {
		caps = append(caps, proto.cap())
	}
	return NewMsg(handshakeMsg,
		baseProtocolVersion,
		bp.peer.ourID.String(),
		caps,
		port,
		bp.peer.ourID.Pubkey()[1:],
	)
}

func (bp *baseProtocol) peerList() []ethutil.RlpEncodable {
	peers := bp.peer.otherPeers()
	ds := make([]ethutil.RlpEncodable, 0, len(peers))
	for _, p := range peers {
		p.infolock.Lock()
		addr := p.listenAddr
		p.infolock.Unlock()
		// filter out this peer and peers that are not listening or
		// have not completed the handshake.
		// TODO: track previously sent peers and exclude them as well.
		if p == bp.peer || addr == nil {
			continue
		}
		ds = append(ds, addr)
	}
	ourAddr := bp.peer.ourListenAddr
	if ourAddr != nil && !ourAddr.IP.IsLoopback() && !ourAddr.IP.IsUnspecified() {
		ds = append(ds, ourAddr)
	}
	return ds
}
