package swarm

import (
	"context"
	"fmt"

	"github.com/jbenet/goprocess"
	ipnet "github.com/libp2p/go-libp2p-interface-pnet"
	metrics "github.com/libp2p/go-libp2p-metrics"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

// Network implements the inet.Network interface.
// It is simply a swarm, with a few different functions
// to implement inet.Network.
type Network Swarm

func NewNetwork(ctx context.Context, listen []ma.Multiaddr, local peer.ID,
	peers pstore.Peerstore, bwc metrics.Reporter) (*Network, error) {

	return NewNetworkWithProtector(ctx, listen, local, peers, nil, bwc)
}

// NewNetwork constructs a new network and starts listening on given addresses.
func NewNetworkWithProtector(ctx context.Context, listen []ma.Multiaddr, local peer.ID,
	peers pstore.Peerstore, protec ipnet.Protector, bwc metrics.Reporter) (*Network, error) {

	s, err := NewSwarmWithProtector(ctx, listen, local, peers, protec, PSTransport, bwc)
	if err != nil {
		return nil, err
	}

	return (*Network)(s), nil
}

// DialPeer attempts to establish a connection to a given peer.
// Respects the context.
func (n *Network) DialPeer(ctx context.Context, p peer.ID) (inet.Conn, error) {
	log.Debugf("[%s] network dialing peer [%s]", n.local, p)
	sc, err := n.Swarm().Dial(ctx, p)
	if err != nil {
		return nil, err
	}

	log.Debugf("network for %s finished dialing %s", n.local, p)
	return inet.Conn(sc), nil
}

// Process returns the network's Process
func (n *Network) Process() goprocess.Process {
	return n.proc
}

// Swarm returns the network's peerstream.Swarm
func (n *Network) Swarm() *Swarm {
	return (*Swarm)(n)
}

// LocalPeer the network's LocalPeer
func (n *Network) LocalPeer() peer.ID {
	return n.Swarm().LocalPeer()
}

// Peers returns the known peer IDs from the Peerstore
func (n *Network) Peers() []peer.ID {
	return n.Swarm().Peers()
}

// Peerstore returns the Peerstore, which tracks known peers
func (n *Network) Peerstore() pstore.Peerstore {
	return n.Swarm().peers
}

// Conns returns the connected peers
func (n *Network) Conns() []inet.Conn {
	conns1 := n.Swarm().Connections()
	out := make([]inet.Conn, len(conns1))
	for i, c := range conns1 {
		out[i] = inet.Conn(c)
	}
	return out
}

// ConnsToPeer returns the connections in this Netowrk for given peer.
func (n *Network) ConnsToPeer(p peer.ID) []inet.Conn {
	conns1 := n.Swarm().ConnectionsToPeer(p)
	out := make([]inet.Conn, len(conns1))
	for i, c := range conns1 {
		out[i] = inet.Conn(c)
	}
	return out
}

// ClosePeer connection to peer
func (n *Network) ClosePeer(p peer.ID) error {
	return n.Swarm().CloseConnection(p)
}

// close is the real teardown function
func (n *Network) close() error {
	return n.Swarm().Close()
}

// Close calls the ContextCloser func
func (n *Network) Close() error {
	return n.Swarm().proc.Close()
}

// Listen tells the network to start listening on given multiaddrs.
func (n *Network) Listen(addrs ...ma.Multiaddr) error {
	return n.Swarm().Listen(addrs...)
}

// ListenAddresses returns a list of addresses at which this network listens.
func (n *Network) ListenAddresses() []ma.Multiaddr {
	return n.Swarm().ListenAddresses()
}

// InterfaceListenAddresses returns a list of addresses at which this network
// listens. It expands "any interface" addresses (/ip4/0.0.0.0, /ip6/::) to
// use the known local interfaces.
func (n *Network) InterfaceListenAddresses() ([]ma.Multiaddr, error) {
	return n.Swarm().InterfaceListenAddresses()
}

// Connectedness returns a state signaling connection capabilities
// For now only returns Connected || NotConnected. Expand into more later.
func (n *Network) Connectedness(p peer.ID) inet.Connectedness {
	if n.Swarm().HaveConnsToPeer(p) {
		return inet.Connected
	}
	return inet.NotConnected
}

// NewStream returns a new stream to given peer p.
// If there is no connection to p, attempts to create one.
func (n *Network) NewStream(ctx context.Context, p peer.ID) (inet.Stream, error) {
	log.Debugf("[%s] network opening stream to peer [%s]", n.local, p)
	s, err := n.Swarm().NewStreamWithPeer(ctx, p)
	if err != nil {
		return nil, err
	}

	return inet.Stream(s), nil
}

// SetStreamHandler sets the protocol handler on the Network's Muxer.
// This operation is threadsafe.
func (n *Network) SetStreamHandler(h inet.StreamHandler) {
	n.Swarm().SetStreamHandler(h)
}

// SetConnHandler sets the conn handler on the Network.
// This operation is threadsafe.
func (n *Network) SetConnHandler(h inet.ConnHandler) {
	n.Swarm().SetConnHandler(func(c *Conn) {
		h(inet.Conn(c))
	})
}

// String returns a string representation of Network.
func (n *Network) String() string {
	return fmt.Sprintf("<Network %s>", n.LocalPeer())
}

// Notify signs up Notifiee to receive signals when events happen
func (n *Network) Notify(f inet.Notifiee) {
	n.Swarm().Notify(f)
}

// StopNotify unregisters Notifiee fromr receiving signals
func (n *Network) StopNotify(f inet.Notifiee) {
	n.Swarm().StopNotify(f)
}
