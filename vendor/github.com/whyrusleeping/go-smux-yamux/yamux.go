package sm_yamux

import (
	"io/ioutil"
	"net"
	"time"

	smux "github.com/libp2p/go-stream-muxer"
	yamux "github.com/whyrusleeping/yamux"
)

// Conn is a connection to a remote peer.
type conn yamux.Session

func (c *conn) yamuxSession() *yamux.Session {
	return (*yamux.Session)(c)
}

func (c *conn) Close() error {
	return c.yamuxSession().Close()
}

func (c *conn) IsClosed() bool {
	return c.yamuxSession().IsClosed()
}

// OpenStream creates a new stream.
func (c *conn) OpenStream() (smux.Stream, error) {
	s, err := c.yamuxSession().OpenStream()
	if err != nil {
		return nil, err
	}

	return s, nil
}

// AcceptStream accepts a stream opened by the other side.
func (c *conn) AcceptStream() (smux.Stream, error) {
	s, err := c.yamuxSession().AcceptStream()
	return s, err
}

// Transport is a go-peerstream transport that constructs
// yamux-backed connections.
type Transport yamux.Config

// DefaultTransport has default settings for yamux
var DefaultTransport = (*Transport)(&yamux.Config{
	AcceptBacklog:          256,                // from yamux.DefaultConfig
	EnableKeepAlive:        true,               // from yamux.DefaultConfig
	KeepAliveInterval:      30 * time.Second,   // from yamux.DefaultConfig
	ConnectionWriteTimeout: 10 * time.Second,   // from yamux.DefaultConfig
	MaxStreamWindowSize:    uint32(256 * 1024), // from yamux.DefaultConfig
	LogOutput:              ioutil.Discard,
})

func (t *Transport) NewConn(nc net.Conn, isServer bool) (smux.Conn, error) {
	var s *yamux.Session
	var err error
	if isServer {
		s, err = yamux.Server(nc, t.Config())
	} else {
		s, err = yamux.Client(nc, t.Config())
	}
	return (*conn)(s), err
}

func (t *Transport) Config() *yamux.Config {
	return (*yamux.Config)(t)
}
