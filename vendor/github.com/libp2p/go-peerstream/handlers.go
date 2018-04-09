package peerstream

import (
	"io"
	"math/rand"
)

// SelectRandomConn defines a function which takes
// a slice of Conns and returns a randomly selected one.
// Note that it is user's responsability to rand.Seed before
// using.
var SelectRandomConn = func(conns []*Conn) *Conn {
	if len(conns) == 0 {
		return nil
	}

	return conns[rand.Intn(len(conns))]
}

// EchoHandler launches a StreamHandling go-routine
// which echoes everything read from the stream
// back into it. It closes the stream at the end.
func EchoHandler(s *Stream) {
	go func() {
		io.Copy(s, s)
		s.Close()
	}()
}

// ResetHandler is a StreamHandler which simply resets
// the stream.
func ResetHandler(s *Stream) {
	s.Reset()
}

// NoOpStreamHandler is a StreamHandler which does nothing.
func NoOpStreamHandler(s *Stream) {}

// NoOpConnHandler is a connection handler which does nothing.
func NoOpConnHandler(c *Conn) {}
