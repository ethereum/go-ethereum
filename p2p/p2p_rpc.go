package p2p

import (
	rpc "github.com/ethereum/go-ethereum/rpc/v2"
	"fmt"
)
type NetService struct {
	net            *Server
	networkVersion int
}

func NewNetService(net *Server, networkVersion int) *NetService {
	return &NetService{net, networkVersion}
}

// Listening returns an indication if the node is listening for network connections.
func (s *NetService) Listening() bool {
	return true // always listening
}

// Peercount returns the number of connected peers
func (s *NetService) PeerCount() *rpc.HexNumber {
	return rpc.NewHexNumber(s.net.PeerCount())
}

// ProtocolVersion returns the current ethereum protocol version.
func (s *NetService) Version() string {
	return fmt.Sprintf("%d", s.networkVersion)
}
