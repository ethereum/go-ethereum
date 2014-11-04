package p2p

import (
	"fmt"
	"net"
	"strconv"
)

type Peer struct {
	Inbound          bool // inbound (via listener) or outbound (via dialout)
	Address          net.Addr
	Host             []byte
	Port             uint16
	Pubkey           []byte
	Id               string
	Caps             []string
	peerErrorChan    chan error
	messenger        *messenger
	peerErrorHandler *PeerErrorHandler
	server           *Server
}

func NewPeer(conn net.Conn, address net.Addr, inbound bool, server *Server) *Peer {
	peerErrorChan := NewPeerErrorChannel()
	host, port, _ := net.SplitHostPort(address.String())
	intport, _ := strconv.Atoi(port)
	peer := &Peer{
		Inbound:       inbound,
		Address:       address,
		Port:          uint16(intport),
		Host:          net.ParseIP(host),
		peerErrorChan: peerErrorChan,
		server:        server,
	}
	peer.messenger = newMessenger(peer, conn, peerErrorChan, server.Handlers())
	peer.peerErrorHandler = NewPeerErrorHandler(address, server.PeerDisconnect(), peerErrorChan)
	return peer
}

func (self *Peer) String() string {
	var kind string
	if self.Inbound {
		kind = "inbound"
	} else {
		kind = "outbound"
	}
	return fmt.Sprintf("%v:%v (%s) v%v %v", self.Host, self.Port, kind, self.Id, self.Caps)
}

func (self *Peer) Write(protocol string, msg Msg) error {
	return self.messenger.writeProtoMsg(protocol, msg)
}

func (self *Peer) Start() {
	self.peerErrorHandler.Start()
	self.messenger.Start()
}

func (self *Peer) Stop() {
	self.peerErrorHandler.Stop()
	self.messenger.Stop()
}

func (p *Peer) Encode() []interface{} {
	return []interface{}{p.Host, p.Port, p.Pubkey}
}
