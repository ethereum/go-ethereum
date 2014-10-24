package p2p

import (
	"bytes"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"
)

type Protocol interface {
	Start()
	Stop()
	HandleIn(*Msg, chan *Msg)
	HandleOut(*Msg) bool
	Offset() MsgCode
	Name() string
}

const (
	P2PVersion      = 0
	pingTimeout     = 2
	pingGracePeriod = 2
)

const (
	HandshakeMsg = iota
	DiscMsg
	PingMsg
	PongMsg
	GetPeersMsg
	PeersMsg
	offset = 16
)

type ProtocolState uint8

const (
	nullState = iota
	handshakeReceived
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

var discReasonToString = map[DiscReason]string{
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

type BaseProtocol struct {
	peer      *Peer
	state     ProtocolState
	stateLock sync.RWMutex
}

func NewBaseProtocol(peer *Peer) *BaseProtocol {
	self := &BaseProtocol{
		peer: peer,
	}

	return self
}

func (self *BaseProtocol) Start() {
	if self.peer != nil {
		self.peer.Write("", self.peer.Server().Handshake())
		go self.peer.Messenger().PingPong(
			pingTimeout*time.Second,
			pingGracePeriod*time.Second,
			self.Ping,
			self.Timeout,
		)
	}
}

func (self *BaseProtocol) Stop() {
}

func (self *BaseProtocol) Ping() {
	msg, _ := NewMsg(PingMsg)
	self.peer.Write("", msg)
}

func (self *BaseProtocol) Timeout() {
	self.peerError(PingTimeout, "")
}

func (self *BaseProtocol) Name() string {
	return ""
}

func (self *BaseProtocol) Offset() MsgCode {
	return offset
}

func (self *BaseProtocol) CheckState(state ProtocolState) bool {
	self.stateLock.RLock()
	self.stateLock.RUnlock()
	if self.state != state {
		return false
	} else {
		return true
	}
}

func (self *BaseProtocol) HandleIn(msg *Msg, response chan *Msg) {
	if msg.Code() == HandshakeMsg {
		self.handleHandshake(msg)
	} else {
		if !self.CheckState(handshakeReceived) {
			self.peerError(ProtocolBreach, "message code %v not allowed", msg.Code())
			close(response)
			return
		}
		switch msg.Code() {
		case DiscMsg:
			logger.Infof("Disconnect requested from peer %v, reason", DiscReason(msg.Data().Get(0).Uint()))
			self.peer.Server().PeerDisconnect() <- DisconnectRequest{
				addr:   self.peer.Address,
				reason: DiscRequested,
			}
		case PingMsg:
			out, _ := NewMsg(PongMsg)
			response <- out
		case PongMsg:
		case GetPeersMsg:
			// Peer asked for list of connected peers
			if out, err := self.peer.Server().PeersMessage(); err != nil {
				response <- out
			}
		case PeersMsg:
			self.handlePeers(msg)
		default:
			self.peerError(InvalidMsgCode, "unknown message code %v", msg.Code())
		}
	}
	close(response)
}

func (self *BaseProtocol) HandleOut(msg *Msg) (allowed bool) {
	// somewhat overly paranoid
	allowed = msg.Code() == HandshakeMsg || msg.Code() == DiscMsg || msg.Code() < self.Offset() && self.CheckState(handshakeReceived)
	return
}

func (self *BaseProtocol) peerError(errorCode ErrorCode, format string, v ...interface{}) {
	err := NewPeerError(errorCode, format, v...)
	logger.Warnln(err)
	fmt.Println(self.peer, err)
	if self.peer != nil {
		self.peer.PeerErrorChan() <- err
	}
}

func (self *BaseProtocol) handlePeers(msg *Msg) {
	it := msg.Data().NewIterator()
	for it.Next() {
		ip := net.IP(it.Value().Get(0).Bytes())
		port := it.Value().Get(1).Uint()
		address := &net.TCPAddr{IP: ip, Port: int(port)}
		go self.peer.Server().PeerConnect(address)
	}
}

func (self *BaseProtocol) handleHandshake(msg *Msg) {
	self.stateLock.Lock()
	defer self.stateLock.Unlock()
	if self.state != nullState {
		self.peerError(ProtocolBreach, "extra handshake")
		return
	}

	c := msg.Data()

	var (
		p2pVersion = c.Get(0).Uint()
		id         = c.Get(1).Str()
		caps       = c.Get(2)
		port       = c.Get(3).Uint()
		pubkey     = c.Get(4).Bytes()
	)
	fmt.Printf("handshake received %v, %v, %v, %v, %v ", p2pVersion, id, caps, port, pubkey)

	// Check correctness of p2p protocol version
	if p2pVersion != P2PVersion {
		self.peerError(P2PVersionMismatch, "Require protocol %d, received %d\n", P2PVersion, p2pVersion)
		return
	}

	// Handle the pub key (validation, uniqueness)
	if len(pubkey) == 0 {
		self.peerError(PubkeyMissing, "not supplied in handshake.")
		return
	}

	if len(pubkey) != 64 {
		self.peerError(PubkeyInvalid, "require 512 bit, got %v", len(pubkey)*8)
		return
	}

	// Self connect detection
	if bytes.Compare(self.peer.Server().ClientIdentity().Pubkey()[1:], pubkey) == 0 {
		self.peerError(PubkeyForbidden, "not allowed to connect to self")
		return
	}

	// register pubkey on server. this also sets the pubkey on the peer (need lock)
	if err := self.peer.Server().RegisterPubkey(self.peer, pubkey); err != nil {
		self.peerError(PubkeyForbidden, err.Error())
		return
	}

	// check port
	if self.peer.Inbound {
		uint16port := uint16(port)
		if self.peer.Port > 0 && self.peer.Port != uint16port {
			self.peerError(PortMismatch, "port mismatch: %v != %v", self.peer.Port, port)
			return
		} else {
			self.peer.Port = uint16port
		}
	}

	capsIt := caps.NewIterator()
	for capsIt.Next() {
		cap := capsIt.Value().Str()
		self.peer.Caps = append(self.peer.Caps, cap)
	}
	sort.Strings(self.peer.Caps)
	self.peer.Messenger().AddProtocols(self.peer.Caps)

	self.peer.Id = id

	self.state = handshakeReceived

	//p.ethereum.PushPeer(p)
	// p.ethereum.reactor.Post("peerList", p.ethereum.Peers())
	return
}
