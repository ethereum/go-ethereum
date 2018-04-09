package peerstream

import (
	"fmt"
	"time"

	protocol "github.com/libp2p/go-libp2p-protocol"
	smux "github.com/libp2p/go-stream-muxer"
)

// StreamHandler is a function which receives a Stream. It
// allows clients to set a function to receive newly created
// streams, and decide whether to continue adding them.
// It works sort of like a http.HandleFunc.
// Note: the StreamHandler is called sequentially, so spawn
// goroutines or pass the Stream. See EchoHandler.
type StreamHandler func(s *Stream)

// Stream is an io.{Read,Write,Close}r to a remote counterpart.
// It wraps a spdystream.Stream, and links it to a Conn and groups
type Stream struct {
	smuxStream smux.Stream

	conn     *Conn
	groups   groupSet
	protocol protocol.ID
}

var _ smux.Stream = &Stream{}

func newStream(ss smux.Stream, c *Conn) *Stream {
	s := &Stream{
		conn:       c,
		smuxStream: ss,
		groups:     groupSet{m: make(map[Group]struct{})},
	}
	s.groups.AddSet(&c.groups) // inherit groups
	return s
}

// String returns a string representation of the Stream
func (s *Stream) String() string {
	f := "<peerstream.Stream %s <--> %s>"
	return fmt.Sprintf(f, s.conn.NetConn().LocalAddr(), s.conn.NetConn().RemoteAddr())
}

// Stream returns the underlying stream muxer Stream
func (s *Stream) Stream() smux.Stream {
	return s.smuxStream
}

// Conn returns the Conn associated with this Stream
func (s *Stream) Conn() *Conn {
	return s.conn
}

// Swarm returns the Swarm asociated with this Stream
func (s *Stream) Swarm() *Swarm {
	return s.conn.swarm
}

// Groups returns the Groups this Stream belongs to
func (s *Stream) Groups() []Group {
	return s.groups.Groups()
}

// InGroup returns whether this stream belongs to a Group
func (s *Stream) InGroup(g Group) bool {
	return s.groups.Has(g)
}

// AddGroup assigns given Group to Stream
func (s *Stream) AddGroup(g Group) {
	s.groups.Add(g)
}

// Read reads from the stream and returns the number
// of bytes read. It implements the io.Reader interface.
func (s *Stream) Read(p []byte) (n int, err error) {
	return s.smuxStream.Read(p)
}

// Write writes to the stream and returns the number
// of bytes written. It implements the io.Writer interface.
func (s *Stream) Write(p []byte) (n int, err error) {
	return s.smuxStream.Write(p)
}

// Reset resets the stream and removes it from the swarm.
func (s *Stream) Reset() error {
	return s.conn.swarm.removeStream(s, true)
}

// Close closes the write end of the stream.
// NOTE: This currently removes the stream from the swarm as well. We shouldn't
// do this but not doing this will result in bad memory leaks so we're punting
// for now until we have some way to know when a stream has been closed by the
// remote side.
func (s *Stream) Close() error {
	return s.conn.swarm.removeStream(s, false)
}

// Protocol returns the protocol identifier associated to this Stream.
func (s *Stream) Protocol() protocol.ID {
	return s.protocol
}

// SetProtocol sets the protocol identifier for this Stream.
func (s *Stream) SetProtocol(p protocol.ID) {
	s.protocol = p
}

// SetDeadline sets the read and write deadlines associated
// with the Stream. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail with a timeout (see type Error) instead of
// blocking.
func (s *Stream) SetDeadline(t time.Time) error {
	return s.smuxStream.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (s *Stream) SetReadDeadline(t time.Time) error {
	return s.smuxStream.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (s *Stream) SetWriteDeadline(t time.Time) error {
	return s.smuxStream.SetWriteDeadline(t)
}

// StreamsWithGroup narrows down a set of streams to those in given group.
func StreamsWithGroup(g Group, streams []*Stream) []*Stream {
	var out []*Stream
	for _, s := range streams {
		if s.InGroup(g) {
			out = append(out, s)
		}
	}
	return out
}
