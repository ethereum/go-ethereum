package multistream

import (
	"io"
	"sync"
)

// lazyServerConn is an io.ReadWriteCloser adapter used for negotiating inbound
// streams (see NegotiateLazy).
//
// This is "lazy" because it doesn't wait for the write half to succeed before
// allowing us to read from the stream.
type lazyServerConn struct {
	waitForHandshake sync.Once
	werr             error

	con io.ReadWriteCloser
}

func (l *lazyServerConn) Write(b []byte) (int, error) {
	l.waitForHandshake.Do(func() { panic("didn't initiate handshake") })
	if l.werr != nil {
		return 0, l.werr
	}
	return l.con.Write(b)
}

func (l *lazyServerConn) Read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	return l.con.Read(b)
}

func (l *lazyServerConn) Close() error {
	return l.con.Close()
}
