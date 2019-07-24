package sockets

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/Microsoft/go-winio"
)

func configureUnixTransport(tr *http.Transport, proto, addr string) error {
	return ErrProtocolNotAvailable
}

func configureNpipeTransport(tr *http.Transport, proto, addr string) error {
	// No need for compression in local communications.
	tr.DisableCompression = true
	tr.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		// DialPipeContext() has been added to winio:
		// https://github.com/Microsoft/go-winio/commit/5fdbdcc2ae1c7e1073157fa7cb34a15eab472e1d
		// However, a new version of winio with this commit has not been released yet.
		// Continue to use DialPipe() until DialPipeContext() becomes available.
		//return winio.DialPipeContext(ctx, addr)
		return DialPipe(addr, defaultTimeout)
	}
	return nil
}

// DialPipe connects to a Windows named pipe.
func DialPipe(addr string, timeout time.Duration) (net.Conn, error) {
	return winio.DialPipe(addr, &timeout)
}
