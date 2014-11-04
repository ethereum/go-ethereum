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
	errc           chan error
}

func NewPeerErrorHandler(address net.Addr, peerDisconnect chan DisconnectRequest, errc chan error) *PeerErrorHandler {
	return &PeerErrorHandler{
		quit:           make(chan chan bool),
		address:        address,
		peerDisconnect: peerDisconnect,
		errc:           errc,
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
		case err, ok := <-self.errc:
			if ok {
				logger.Debugf("error %v\n", err)
				go self.handle(err)
			} else {
				return
			}
		case q := <-self.quit:
			q <- true
			return
		}
	}
}

func (self *PeerErrorHandler) handle(err error) {
	reason := DiscReason(' ')
	peerError, ok := err.(*PeerError)
	if !ok {
		peerError = NewPeerError(MiscError, " %v", err)
	}
	switch peerError.Code {
	case P2PVersionMismatch:
		reason = DiscIncompatibleVersion
	case PubkeyMissing, PubkeyInvalid:
		reason = DiscInvalidIdentity
	case PubkeyForbidden:
		reason = DiscUselessPeer
	case InvalidMsgCode, PacketTooLong, PayloadTooShort, MagicTokenMismatch, ProtocolBreach:
		reason = DiscProtocolError
	case PingTimeout:
		reason = DiscReadTimeout
	case ReadError, WriteError, MiscError:
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
	return 1
}
