package peerstream_spdystream

import (
	"errors"
	"net"
	"net/http"
	"time"

	ss "github.com/docker/spdystream"
	smux "github.com/libp2p/go-stream-muxer"
)

// errClosed is returned when trying to accept a stream from a closed connection
var errClosed = errors.New("conn closed")

// stream implements smux.Stream using a ss.Stream
type stream ss.Stream

func (s *stream) spdyStream() *ss.Stream {
	return (*ss.Stream)(s)
}

func (s *stream) Read(buf []byte) (int, error) {
	return s.spdyStream().Read(buf)
}

func (s *stream) Write(buf []byte) (int, error) {
	return s.spdyStream().Write(buf)
}

func (s *stream) Close() error {
	// Reset is spdystream's full bidirectional close.
	// We expose bidirectional close as our `Close`.
	// To close only half of the connection, and use other
	// spdystream options, just get the stream with:
	//  ssStream := (*ss.Stream)(stream)
	return s.spdyStream().Close()
}

func (s *stream) Reset() error {
	return s.spdyStream().Reset()
}

func (s *stream) SetDeadline(t time.Time) error {
	return (*ss.Stream)(s).SetDeadline(t)
}

func (s *stream) SetReadDeadline(t time.Time) error {
	return (*ss.Stream)(s).SetReadDeadline(t)
}

func (s *stream) SetWriteDeadline(t time.Time) error {
	return (*ss.Stream)(s).SetWriteDeadline(t)
}

// StreamQueueLen is the length of the stream queue.
const StreamQueueLen = 10

// Conn is a connection to a remote peer.
type conn struct {
	sc *ss.Connection

	streamQueue chan *ss.Stream

	closed chan struct{}
}

func (c *conn) spdyConn() *ss.Connection {
	return c.sc
}

func (c *conn) Close() error {
	err := c.spdyConn().CloseWait()
	if !c.IsClosed() {
		close(c.closed)
	}
	return err
}

func (c *conn) IsClosed() bool {
	select {
	case <-c.closed:
		return true
	case <-c.sc.CloseChan():
		return true
	default:
		return false
	}
}

// OpenStream creates a new stream.
func (c *conn) OpenStream() (smux.Stream, error) {
	s, err := c.spdyConn().CreateStream(http.Header{
		":method": []string{"POST"}, // this is here for HTTP/SPDY interop
		":path":   []string{"/"},    // this is here for HTTP/SPDY interop
	}, nil, false)
	if err != nil {
		return nil, err
	}

	// wait for a response before writing. for some reason
	// spdystream does not make forward progress unless you do this.
	s.Wait()
	return (*stream)(s), nil
}

// AcceptStream accepts a stream opened by the other side.
func (c *conn) AcceptStream() (smux.Stream, error) {
	if c.IsClosed() {
		return nil, errClosed
	}

	select {
	case <-c.closed:
		return nil, errClosed
	case <-c.sc.CloseChan():
		return nil, errClosed
	case s := <-c.streamQueue:
		return s, nil
	}
}

// serve accepts incoming streams and places them in the streamQueue
func (c *conn) serve() {
	c.spdyConn().Serve(func(s *ss.Stream) {
		// Flow control and backpressure of Opening streams is broken.
		// I believe that spdystream has one set of workers that both send
		// data AND accept new streams (as it's just more data). there
		// is a problem where if the new stream handlers want to throttle,
		// they also eliminate the ability to read/write data, which makes
		// forward-progress impossible. Thus, throttling this function is
		// -- at this moment -- not the solution. Either spdystream must
		// change, or we must throttle another way. go-peerstream handles
		// every new stream in its own goroutine.
		err := s.SendReply(http.Header{
			":status": []string{"200"},
		}, false)
		if err != nil {
			// this _could_ error out. not sure how to handle this failure.
			// don't return, and let the caller handle a broken stream.
			// better than _hiding_ an error.
			// return
		}
		c.streamQueue <- s
	})
}

type transport struct{}

// Transport is a go-peerstream transport that constructs
// spdystream-backed connections.
var Transport = transport{}

func (t transport) NewConn(nc net.Conn, isServer bool) (smux.Conn, error) {
	sc, err := ss.NewConnection(nc, isServer)
	if err != nil {
		return nil, err
	}
	c := &conn{
		sc:     sc,
		closed: make(chan struct{}),
	}
	c.streamQueue = make(chan *ss.Stream, StreamQueueLen)
	go c.serve()
	return c, nil
}
