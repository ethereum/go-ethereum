package conn

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	logging "github.com/ipfs/go-log"
	ic "github.com/libp2p/go-libp2p-crypto"
	iconn "github.com/libp2p/go-libp2p-interface-conn"
	lgbl "github.com/libp2p/go-libp2p-loggables"
	peer "github.com/libp2p/go-libp2p-peer"
	tpt "github.com/libp2p/go-libp2p-transport"
	mpool "github.com/libp2p/go-msgio/mpool"
	ma "github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("conn")

// ReleaseBuffer puts the given byte array back into the appropriate
// global buffer pool based on its capacity.
func ReleaseBuffer(b []byte) {
	log.Debugf("Releasing buffer! (cap,size = %d, %d)", cap(b), len(b))
	mpool.ByteSlicePool.Put(uint32(cap(b)), b)
}

// singleConn represents a single connection to another Peer (IPFS Node).
type singleConn struct {
	local  peer.ID
	remote peer.ID
	maconn tpt.Conn

	eventMu sync.Mutex
	event   io.Closer
}

// newConn constructs a new connection
func newSingleConn(ctx context.Context, local, remote peer.ID, maconn tpt.Conn) iconn.Conn {
	ml := lgbl.Dial("conn", local, remote, maconn.LocalMultiaddr(), maconn.RemoteMultiaddr())

	conn := &singleConn{
		local:  local,
		remote: remote,
		maconn: maconn,
		event:  log.EventBegin(ctx, "connLifetime", ml),
	}

	log.Debugf("newSingleConn %p: %v to %v", conn, local, remote)
	return conn
}

// close is the internal close function, called by ContextCloser.Close
func (c *singleConn) Close() error {
	c.eventMu.Lock()
	if c.event != nil {
		evt := c.event
		c.event = nil
		defer evt.Close()
	}
	c.eventMu.Unlock()

	// close underlying connection
	return c.maconn.Close()
}

// ID is an identifier unique to this connection.
func (c *singleConn) ID() string {
	return iconn.ID(c)
}

func (c *singleConn) String() string {
	return iconn.String(c, "singleConn")
}

func (c *singleConn) LocalAddr() net.Addr {
	return c.maconn.LocalAddr()
}

func (c *singleConn) RemoteAddr() net.Addr {
	return c.maconn.RemoteAddr()
}

func (c *singleConn) LocalPrivateKey() ic.PrivKey {
	return nil
}

func (c *singleConn) RemotePublicKey() ic.PubKey {
	return nil
}

func (c *singleConn) SetDeadline(t time.Time) error {
	return c.maconn.SetDeadline(t)
}
func (c *singleConn) SetReadDeadline(t time.Time) error {
	return c.maconn.SetReadDeadline(t)
}

func (c *singleConn) SetWriteDeadline(t time.Time) error {
	return c.maconn.SetWriteDeadline(t)
}

// LocalMultiaddr is the Multiaddr on this side
func (c *singleConn) LocalMultiaddr() ma.Multiaddr {
	return c.maconn.LocalMultiaddr()
}

// RemoteMultiaddr is the Multiaddr on the remote side
func (c *singleConn) RemoteMultiaddr() ma.Multiaddr {
	return c.maconn.RemoteMultiaddr()
}

func (c *singleConn) Transport() tpt.Transport {
	return c.maconn.Transport()
}

// LocalPeer is the Peer on this side
func (c *singleConn) LocalPeer() peer.ID {
	return c.local
}

// RemotePeer is the Peer on the remote side
func (c *singleConn) RemotePeer() peer.ID {
	return c.remote
}

// Read reads data, net.Conn style
func (c *singleConn) Read(buf []byte) (int, error) {
	return c.maconn.Read(buf)
}

// Write writes data, net.Conn style
func (c *singleConn) Write(buf []byte) (int, error) {
	return c.maconn.Write(buf)
}
