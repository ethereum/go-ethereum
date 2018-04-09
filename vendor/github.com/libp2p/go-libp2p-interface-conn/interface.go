package conn

import (
	"fmt"
	"io"
	"net"
	"time"

	u "github.com/ipfs/go-ipfs-util"
	ic "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	tpt "github.com/libp2p/go-libp2p-transport"
	filter "github.com/libp2p/go-maddr-filter"
	ma "github.com/multiformats/go-multiaddr"
)

type PeerConn interface {
	io.Closer

	// LocalPeer (this side) ID, PrivateKey, and Address
	LocalPeer() peer.ID
	LocalPrivateKey() ic.PrivKey
	LocalMultiaddr() ma.Multiaddr

	// RemotePeer ID, PublicKey, and Address
	RemotePeer() peer.ID
	RemotePublicKey() ic.PubKey
	RemoteMultiaddr() ma.Multiaddr
}

// Conn is a generic message-based Peer-to-Peer connection.
type Conn interface {
	PeerConn

	// ID is an identifier unique to this connection.
	ID() string

	// can't just say "net.Conn" cause we have duplicate methods.
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	Transport() tpt.Transport

	io.Reader
	io.Writer
}

// Listener is an object that can accept connections. It matches net.Listener
type Listener interface {

	// Accept waits for and returns the next connection to the listener.
	Accept() (tpt.Conn, error)

	// Addr is the local address
	Addr() net.Addr

	// Multiaddr is the local multiaddr address
	Multiaddr() ma.Multiaddr

	// LocalPeer is the identity of the local Peer.
	LocalPeer() peer.ID

	SetAddrFilters(*filter.Filters)

	// Close closes the listener.
	// Any blocked Accept operations will be unblocked and return errors.
	Close() error
}

// EncryptConnections is a global parameter because it should either be
// enabled or _completely disabled_. I.e. a node should only be able to talk
// to proper (encrypted) networks if it is encrypting all its transports.
// Running a node with disabled transport encryption is useful to debug the
// protocols, achieve implementation interop, or for private networks which
// -- for whatever reason -- _must_ run unencrypted.
var EncryptConnections = true

// ID returns the ID of a given Conn.
func ID(c Conn) string {
	l := fmt.Sprintf("%s/%s", c.LocalMultiaddr(), c.LocalPeer().Pretty())
	r := fmt.Sprintf("%s/%s", c.RemoteMultiaddr(), c.RemotePeer().Pretty())
	lh := u.Hash([]byte(l))
	rh := u.Hash([]byte(r))
	ch := u.XOR(lh, rh)
	return peer.ID(ch).Pretty()
}

// String returns the user-friendly String representation of a conn
func String(c Conn, typ string) string {
	return fmt.Sprintf("%s (%s) <-- %s %p --> (%s) %s",
		c.LocalPeer(), c.LocalMultiaddr(), typ, c, c.RemoteMultiaddr(), c.RemotePeer())
}
