package net

import (
	"errors"
	"io"
	"time"
)

// EOFTimeout is the maximum amount of time to wait to successfully observe an
// EOF on the stream. Defaults to 60 seconds.
var EOFTimeout = time.Second * 60

// ErrExpectedEOF is returned when we read data while expecting an EOF.
var ErrExpectedEOF = errors.New("read data when expecting EOF")

// FullClose closes the stream and waits to read an EOF from the other side.
//
// * If it reads any data *before* the EOF, it resets the stream.
// * If it doesn't read an EOF within EOFTimeout, it resets the stream.
//
// You'll likely want to invoke this as `go FullClose(stream)` to close the
// stream in the background.
func FullClose(s Stream) error {
	if err := s.Close(); err != nil {
		s.Reset()
		return err
	}
	return AwaitEOF(s)
}

// AwaitEOF waits for an EOF on the given stream, returning an error if that
// fails. It waits at most EOFTimeout (defaults to 1 minute) after which it
// resets the stream.
func AwaitEOF(s Stream) error {
	// So we don't wait forever
	s.SetDeadline(time.Now().Add(EOFTimeout))

	// We *have* to observe the EOF. Otherwise, we leak the stream.
	// Now, technically, we should do this *before*
	// returning from SendMessage as the message
	// hasn't really been sent yet until we see the
	// EOF but we don't actually *know* what
	// protocol the other side is speaking.
	n, err := s.Read([]byte{0})
	if n > 0 || err == nil {
		s.Reset()
		return ErrExpectedEOF
	}
	if err != io.EOF {
		s.Reset()
		return err
	}
	return nil
}
