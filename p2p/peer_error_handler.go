package p2p

import (
	"net"
)

const (
	severityThreshold = 10
)

type DisconnectRequest struct {
	addr   net.Addr
	reason DiscReason
}

type PeerErrorHandler struct {
	quit           chan chan bool
	address        net.Addr
	peerDisconnect chan DisconnectRequest
	severity       int
	peerErrorChan  chan *PeerError
	blacklist      Blacklist
}

func NewPeerErrorHandler(address net.Addr, peerDisconnect chan DisconnectRequest, peerErrorChan chan *PeerError, blacklist Blacklist) *PeerErrorHandler {
	return &PeerErrorHandler{
		quit:           make(chan chan bool),
		address:        address,
		peerDisconnect: peerDisconnect,
		peerErrorChan:  peerErrorChan,
		blacklist:      blacklist,
	}
}

func (self *PeerErrorHandler) Start() {
	go self.listen()
}

func (self *PeerErrorHandler) Stop() {
	q := make(chan bool)
	self.quit <- q
	<-q
}

func (self *PeerErrorHandler) listen() {
	for {
		select {
		case peerError, ok := <-self.peerErrorChan:
			if ok {
				logger.Debugf("error %v\n", peerError)
				go self.handle(peerError)
			} else {
				return
			}
		case q := <-self.quit:
			q <- true
			return
		}
	}
}

func (self *PeerErrorHandler) handle(peerError *PeerError) {
	reason := DiscReason(' ')
	switch peerError.Code {
	case P2PVersionMismatch:
		reason = DiscIncompatibleVersion
	case PubkeyMissing, PubkeyInvalid:
		reason = DiscInvalidIdentity
	case PubkeyForbidden:
		reason = DiscUselessPeer
	case InvalidMsgCode, PacketTooShort, PayloadTooShort, MagicTokenMismatch, EmptyPayload, ProtocolBreach:
		reason = DiscProtocolError
	case PingTimeout:
		reason = DiscReadTimeout
	case WriteError, MiscError:
		reason = DiscNetworkError
	case InvalidGenesis, InvalidNetworkId, InvalidProtocolVersion:
		reason = DiscSubprotocolError
	default:
		self.severity += self.getSeverity(peerError)
	}

	if self.severity >= severityThreshold {
		reason = DiscSubprotocolError
	}
	if reason != DiscReason(' ') {
		self.peerDisconnect <- DisconnectRequest{
			addr:   self.address,
			reason: reason,
		}
	}
}

func (self *PeerErrorHandler) getSeverity(peerError *PeerError) int {
	switch peerError.Code {
	case ReadError:
		return 4 //tolerate 3 :)
	default:
		return 1
	}
}
