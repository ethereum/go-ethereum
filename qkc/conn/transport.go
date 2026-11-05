// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package conn

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/qkc/wire"
)

// frameWriteTimeout bounds each WriteFrame call (encode + flush), following
// geth's bounded-write model: the same 20s constant geth uses for
// frameWriteTimeout (p2p/server.go, "Maximum amount of time allowed for
// writing a complete message") applied per message in rlpxTransport.WriteMsg
// (p2p/transport.go). A peer that stops reading cannot block a writer
// indefinitely: deadline expiry surfaces as a write error, which BaseConn
// treats as fatal and closes the connection. Var (not const) so tests can
// shorten it.
var frameWriteTimeout = 20 * time.Second

// FrameTransport is the frame I/O contract required by BaseConn.
//
// Close must be safe to call concurrently with ReadFrame/WriteFrame and must
// unblock any currently blocked I/O operation.
// BaseConn may call Close while holding writeMu after a write failure. Close
// must not acquire that lock, directly or indirectly, or wait for work that
// requires it (for example, by writing through the same BaseConn).
// A virtual transport's Close must close only that virtual connection and leave
// the shared physical transport available to other routes.
// WriteFrame may return wire.ErrInvalidClusterFrame only if no frame bytes were
// written, including through an error wrapping it.
type FrameTransport interface {
	ReadFrame() (*wire.ClusterFrame, error)
	WriteFrame(*wire.ClusterFrame) error
	Close() error
	RemoteAddr() string
}

type transport struct {
	conn net.Conn
	r    *bufio.Reader
	w    *bufio.Writer

	readFrameFn  func(io.Reader) (*wire.ClusterFrame, error)
	writeFrameFn func(io.Writer, *wire.ClusterFrame) error

	remoteAddr string
}

// NewTCPTransport creates a FrameTransport backed by a real TCP connection.
// readFrame and writeFrame are a matched codec pair selected for this
// connection (for example, ClusterFrame with or without ClusterPeerID).
func NewTCPTransport(
	conn net.Conn,
	readFrame func(io.Reader) (*wire.ClusterFrame, error),
	writeFrame func(io.Writer, *wire.ClusterFrame) error,
) FrameTransport {
	return &transport{
		conn:         conn,
		r:            bufio.NewReader(conn),
		w:            bufio.NewWriter(conn),
		readFrameFn:  readFrame,
		writeFrameFn: writeFrame,
		remoteAddr:   conn.RemoteAddr().String(),
	}
}

func (t *transport) ReadFrame() (*wire.ClusterFrame, error) {
	return t.readFrameFn(t.r)
}

func (t *transport) WriteFrame(f *wire.ClusterFrame) error {
	// Clear the deadline once this write finishes (success or error), so it
	// does not stay armed on the conn and break later operations. Failure to
	// clear must not fail an already-successful write.
	defer func() {
		_ = t.conn.SetWriteDeadline(time.Time{})
	}()

	if err := t.conn.SetWriteDeadline(time.Now().Add(frameWriteTimeout)); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}
	if err := t.writeFrameFn(t.w, f); err != nil {
		return fmt.Errorf("write frame: %w", err)
	}
	if err := t.w.Flush(); err != nil {
		return fmt.Errorf("flush: %w", err)
	}
	return nil
}

func (t *transport) Close() error {
	return t.conn.Close()
}

func (t *transport) RemoteAddr() string {
	return t.remoteAddr
}
