package multistream

import (
	"bufio"
	"fmt"
	"io"
	"sync"
)

// Multistream represents in essense a ReadWriteCloser, or a single
// communication wire which supports multiple streams on it. Each
// stream is identified by a protocol tag.
type Multistream interface {
	io.ReadWriteCloser
}

// NewMSSelect returns a new Multistream which is able to perform
// protocol selection with a MultistreamMuxer.
func NewMSSelect(c io.ReadWriteCloser, proto string) Multistream {
	return &lazyClientConn{
		protos: []string{ProtocolID, proto},
		con:    c,
	}
}

// NewMultistream returns a multistream for the given protocol. This will not
// perform any protocol selection. If you are using a MultistreamMuxer, use
// NewMSSelect.
func NewMultistream(c io.ReadWriteCloser, proto string) Multistream {
	return &lazyClientConn{
		protos: []string{proto},
		con:    c,
	}
}

// lazyClientConn is a ReadWriteCloser adapter that lazily negotiates a protocol
// using multistream-select on first use.
//
// It *does not* block writes waiting for the other end to respond. Instead, it
// simply assumes the negotiation went successfully and starts writing data.
// See: https://github.com/multiformats/go-multistream/issues/20
type lazyClientConn struct {
	// Used to ensure we only trigger the write half of the handshake once.
	rhandshakeOnce sync.Once
	rerr           error

	// Used to ensure we only trigger the read half of the handshake once.
	whandshakeOnce sync.Once
	werr           error

	// The sequence of protocols to negotiate.
	protos []string

	// The inner connection.
	con io.ReadWriteCloser
}

// Read reads data from the io.ReadWriteCloser.
//
// If the protocol hasn't yet been negotiated, this method triggers the write
// half of the handshake and then waits for the read half to complete.
//
// It returns an error if the read half of the handshake fails.
func (l *lazyClientConn) Read(b []byte) (int, error) {
	l.rhandshakeOnce.Do(func() {
		go l.whandshakeOnce.Do(l.doWriteHandshake)
		l.doReadHandshake()
	})
	if l.rerr != nil {
		return 0, l.rerr
	}
	if len(b) == 0 {
		return 0, nil
	}

	return l.con.Read(b)
}

func (l *lazyClientConn) doReadHandshake() {
	for _, proto := range l.protos {
		// read protocol
		tok, err := ReadNextToken(l.con)
		if err != nil {
			l.rerr = err
			return
		}

		if tok != proto {
			l.rerr = fmt.Errorf("protocol mismatch in lazy handshake ( %s != %s )", tok, proto)
			return
		}
	}
}

func (l *lazyClientConn) doWriteHandshake() {
	l.doWriteHandshakeWithData(nil)
}

// Perform the write handshake but *also* write some extra data.
func (l *lazyClientConn) doWriteHandshakeWithData(extra []byte) int {
	buf := bufio.NewWriter(l.con)
	for _, proto := range l.protos {
		l.werr = delimWrite(buf, []byte(proto))
		if l.werr != nil {
			return 0
		}
	}

	n := 0
	if len(extra) > 0 {
		n, l.werr = buf.Write(extra)
		if l.werr != nil {
			return n
		}
	}
	l.werr = buf.Flush()
	return n
}

// Write writes the given buffer to the underlying connection.
//
// If the protocol has not yet been negotiated, write waits for the write half
// of the handshake to complete triggers (but does not wait for) the read half.
//
// Write *also* ignores errors from the read half of the handshake (in case the
// stream is actually write only).
func (l *lazyClientConn) Write(b []byte) (int, error) {
	n := 0
	l.whandshakeOnce.Do(func() {
		go l.rhandshakeOnce.Do(l.doReadHandshake)
		n = l.doWriteHandshakeWithData(b)
	})
	if l.werr != nil || n > 0 {
		return n, l.werr
	}
	return l.con.Write(b)
}

// Close closes the underlying io.ReadWriteCloser
func (l *lazyClientConn) Close() error {
	return l.con.Close()
}
