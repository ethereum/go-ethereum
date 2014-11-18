package p2p

import (
	"bytes"
	"net"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/ethutil"
)

// Protocol is implemented by P2P subprotocols.
type Protocol interface {
	// Start is called when the protocol becomes active.
	// It should read and write messages from rw.
	// Messages must be fully consumed.
	//
	// The connection is closed when Start returns. It should return
	// any protocol-level error (such as an I/O error) that is
	// encountered.
	Start(peer *Peer, rw MsgReadWriter) error

	// Offset should return the number of message codes
	// used by the protocol.
	Offset() MsgCode
}

type MsgReader interface {
	ReadMsg() (Msg, error)
}

type MsgWriter interface {
	WriteMsg(Msg) error
}

// MsgReadWriter is passed to protocols. Protocol implementations can
// use it to write messages back to a connected peer.
type MsgReadWriter interface {
	MsgReader
	MsgWriter
}

type MsgHandler func(code MsgCode, data *ethutil.Value) error

// MsgLoop reads messages off the given reader and
// calls the handler function for each decoded message until
// it returns an error or the peer connection is closed.
//
// If a message is larger than the given maximum size, RunProtocol
// returns an appropriate error.n
func MsgLoop(r MsgReader, maxsize uint32, handler MsgHandler) error {
	for {
		msg, err := r.ReadMsg()
		if err != nil {
			return err
		}
		if msg.Size > maxsize {
			return NewPeerError(InvalidMsg, "size %d exceeds maximum size of %d", msg.Size, maxsize)
		}
		value, err := msg.Data()
		if err != nil {
			return err
		}
		if err := handler(msg.Code, value); err != nil {
			return err
		}
	}
}

// the ÐΞVp2p base protocol
type baseProtocol struct {
	rw   MsgReadWriter
	peer *Peer
}

type bpMsg struct {
	code MsgCode
	data *ethutil.Value
}

const (
	p2pVersion      = 0
	pingTimeout     = 2 * time.Second
	pingGracePeriod = 2 * time.Second
)

const (
	// message codes
	handshakeMsg = iota
	discMsg
	pingMsg
	pongMsg
	getPeersMsg
	peersMsg
)

const (
	baseProtocolOffset     MsgCode = 16
	baseProtocolMaxMsgSize         = 500 * 1024
)

type DiscReason byte

const (
	// Values are given explicitly instead of by iota because these values are
	// defined by the wire protocol spec; it is easier for humans to ensure
	// correctness when values are explicit.
	DiscRequested           = 0x00
	DiscNetworkError        = 0x01
	DiscProtocolError       = 0x02
	DiscUselessPeer         = 0x03
	DiscTooManyPeers        = 0x04
	DiscAlreadyConnected    = 0x05
	DiscIncompatibleVersion = 0x06
	DiscInvalidIdentity     = 0x07
	DiscQuitting            = 0x08
	DiscUnexpectedIdentity  = 0x09
	DiscSelf                = 0x0a
	DiscReadTimeout         = 0x0b
	DiscSubprotocolError    = 0x10
)

var discReasonToString = [DiscSubprotocolError + 1]string{
	DiscRequested:           "Disconnect requested",
	DiscNetworkError:        "Network error",
	DiscProtocolError:       "Breach of protocol",
	DiscUselessPeer:         "Useless peer",
	DiscTooManyPeers:        "Too many peers",
	DiscAlreadyConnected:    "Already connected",
	DiscIncompatibleVersion: "Incompatible P2P protocol version",
	DiscInvalidIdentity:     "Invalid node identity",
	DiscQuitting:            "Client quitting",
	DiscUnexpectedIdentity:  "Unexpected identity",
	DiscSelf:                "Connected to self",
	DiscReadTimeout:         "Read timeout",
	DiscSubprotocolError:    "Subprotocol error",
}

func (d DiscReason) String() string {
	if len(discReasonToString) < int(d) {
		return "Unknown"
	}
	return discReasonToString[d]
}

func (bp *baseProtocol) Offset() MsgCode {
	return baseProtocolOffset
}

func (bp *baseProtocol) Start(peer *Peer, rw MsgReadWriter) error {
	bp.peer, bp.rw = peer, rw

	// Do the handshake.
	// TODO: disconnect is valid before handshake, too.
	rw.WriteMsg(bp.peer.server.handshakeMsg())
	msg, err := rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != handshakeMsg {
		return NewPeerError(ProtocolBreach, " first message must be handshake")
	}
	data, err := msg.Data()
	if err != nil {
		return NewPeerError(InvalidMsg, "%v", err)
	}
	if err := bp.handleHandshake(data); err != nil {
		return err
	}

	msgin := make(chan bpMsg)
	done := make(chan error, 1)
	go func() {
		done <- MsgLoop(rw, baseProtocolMaxMsgSize,
			func(code MsgCode, data *ethutil.Value) error {
				msgin <- bpMsg{code, data}
				return nil
			})
	}()
	return bp.loop(msgin, done)
}

func (bp *baseProtocol) loop(msgin <-chan bpMsg, quit <-chan error) error {
	logger.Debugf("pingpong keepalive started at %v\n", time.Now())
	messenger := bp.rw.(*proto).messenger
	pingTimer := time.NewTimer(pingTimeout)
	pinged := true

	for {
		select {
		case msg := <-msgin:
			if err := bp.handle(msg.code, msg.data); err != nil {
				return err
			}
		case err := <-quit:
			return err
		case <-messenger.pulse:
			pingTimer.Reset(pingTimeout)
			pinged = false
		case <-pingTimer.C:
			if pinged {
				return NewPeerError(PingTimeout, "")
			}
			logger.Debugf("pinging at %v\n", time.Now())
			if err := bp.rw.WriteMsg(NewMsg(pingMsg)); err != nil {
				return NewPeerError(WriteError, "%v", err)
			}
			pinged = true
			pingTimer.Reset(pingTimeout)
		}
	}
}

func (bp *baseProtocol) handle(code MsgCode, data *ethutil.Value) error {
	switch code {
	case handshakeMsg:
		return NewPeerError(ProtocolBreach, " extra handshake received")

	case discMsg:
		logger.Infof("Disconnect requested from peer %v, reason", DiscReason(data.Get(0).Uint()))
		bp.peer.server.PeerDisconnect() <- DisconnectRequest{
			addr:   bp.peer.Address,
			reason: DiscRequested,
		}

	case pingMsg:
		return bp.rw.WriteMsg(NewMsg(pongMsg))

	case pongMsg:
		// reply for ping

	case getPeersMsg:
		// Peer asked for list of connected peers.
		peersRLP := bp.peer.server.encodedPeerList()
		if peersRLP != nil {
			msg := Msg{
				Code:    peersMsg,
				Size:    uint32(len(peersRLP)),
				Payload: bytes.NewReader(peersRLP),
			}
			return bp.rw.WriteMsg(msg)
		}

	case peersMsg:
		bp.handlePeers(data)

	default:
		return NewPeerError(InvalidMsgCode, "unknown message code %v", code)
	}
	return nil
}

func (bp *baseProtocol) handlePeers(data *ethutil.Value) {
	it := data.NewIterator()
	for it.Next() {
		ip := net.IP(it.Value().Get(0).Bytes())
		port := it.Value().Get(1).Uint()
		address := &net.TCPAddr{IP: ip, Port: int(port)}
		go bp.peer.server.PeerConnect(address)
	}
}

func (bp *baseProtocol) handleHandshake(c *ethutil.Value) error {
	var (
		remoteVersion = c.Get(0).Uint()
		id            = c.Get(1).Str()
		caps          = c.Get(2)
		port          = c.Get(3).Uint()
		pubkey        = c.Get(4).Bytes()
	)
	// Check correctness of p2p protocol version
	if remoteVersion != p2pVersion {
		return NewPeerError(P2PVersionMismatch, "Require protocol %d, received %d\n", p2pVersion, remoteVersion)
	}

	// Handle the pub key (validation, uniqueness)
	if len(pubkey) == 0 {
		return NewPeerError(PubkeyMissing, "not supplied in handshake.")
	}

	if len(pubkey) != 64 {
		return NewPeerError(PubkeyInvalid, "require 512 bit, got %v", len(pubkey)*8)
	}

	// self connect detection
	if bytes.Compare(bp.peer.server.ClientIdentity().Pubkey()[1:], pubkey) == 0 {
		return NewPeerError(PubkeyForbidden, "not allowed to connect to self")
	}

	// register pubkey on server. this also sets the pubkey on the peer (need lock)
	if err := bp.peer.server.RegisterPubkey(bp.peer, pubkey); err != nil {
		return NewPeerError(PubkeyForbidden, err.Error())
	}

	// check port
	if bp.peer.Inbound {
		uint16port := uint16(port)
		if bp.peer.Port > 0 && bp.peer.Port != uint16port {
			return NewPeerError(PortMismatch, "port mismatch: %v != %v", bp.peer.Port, port)
		} else {
			bp.peer.Port = uint16port
		}
	}

	capsIt := caps.NewIterator()
	for capsIt.Next() {
		cap := capsIt.Value().Str()
		bp.peer.Caps = append(bp.peer.Caps, cap)
	}
	sort.Strings(bp.peer.Caps)
	bp.rw.(*proto).messenger.setRemoteProtocols(bp.peer.Caps)
	bp.peer.Id = id
	return nil
}
