// Package reuseport provides Listen and Dial functions that set socket options
// in order to be able to reuse ports. You should only use this package if you
// know what SO_REUSEADDR and SO_REUSEPORT are.
//
// For example:
//
//  // listen on the same port. oh yeah.
//  l1, _ := reuse.Listen("tcp", "127.0.0.1:1234")
//  l2, _ := reuse.Listen("tcp", "127.0.0.1:1234")
//
//  // dial from the same port. oh yeah.
//  l1, _ := reuse.Listen("tcp", "127.0.0.1:1234")
//  l2, _ := reuse.Listen("tcp", "127.0.0.1:1235")
//  c, _ := reuse.Dial("tcp", "127.0.0.1:1234", "127.0.0.1:1235")
//
// Note: cant dial self because tcp/ip stacks use 4-tuples to identify connections,
// and doing so would clash.
package reuseport

import (
	"context"
	"errors"
	"net"
	"syscall"
	"time"
)

// Available returns whether or not SO_REUSEPORT is available in the OS.
// It does so by attepting to open a tcp listener, setting the option, and
// checking ENOPROTOOPT on error. After checking, the decision is cached
// for the rest of the process run.
func Available() bool {
	return available()
}

// ErrUnsuportedProtocol signals that the protocol is not currently
// supported by this package. This package currently only supports TCP.
var ErrUnsupportedProtocol = errors.New("protocol not yet supported")

// ErrReuseFailed is returned if a reuse attempt was unsuccessful.
var ErrReuseFailed = errors.New("reuse failed")

// ErrDialSelf is returned if we connect to our own source address.
var ErrDialSelf = errors.New("dialed our own socket")

// Listen listens at the given network and address. see net.Listen
// Returns a net.Listener created from a file discriptor for a socket
// with SO_REUSEPORT and SO_REUSEADDR option set.
func Listen(network, address string) (net.Listener, error) {
	if !available() {
		return nil, syscall.ENOPROTOOPT
	}

	return listenStream(network, address)
}

// ListenPacket listens at the given network and address. see net.ListenPacket
// Returns a net.Listener created from a file discriptor for a socket
// with SO_REUSEPORT and SO_REUSEADDR option set.
func ListenPacket(network, address string) (net.PacketConn, error) {
	if !available() {
		return nil, syscall.ENOPROTOOPT
	}

	return listenPacket(network, address)
}

// Dial dials the given network and address. see net.Dialer.Dial
// Returns a net.Conn created from a file discriptor for a socket
// with SO_REUSEPORT and SO_REUSEADDR option set.
func Dial(network, laddr, raddr string) (net.Conn, error) {
	if !available() {
		return nil, syscall.ENOPROTOOPT
	}

	var d Dialer
	if laddr != "" {
		netladdr, err := ResolveAddr(network, laddr)
		if err != nil {
			return nil, err
		}
		d.D.LocalAddr = netladdr
	}

	return d.Dial(network, raddr)
}

// Dialer is used to specify the Dial options, much like net.Dialer.
// We simply wrap a net.Dialer.
type Dialer struct {
	D net.Dialer
}

// Dial dials the given network and address. see net.Dialer.Dial
// Returns a net.Conn created from a file discriptor for a socket
// with SO_REUSEPORT and SO_REUSEADDR option set.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if !available() {
		return nil, syscall.ENOPROTOOPT
	}

	return dial(ctx, d.D, network, address)
}

func (d *Dialer) deadline(def time.Duration) time.Time {
	switch {
	case !d.D.Deadline.IsZero():
		return d.D.Deadline
	case d.D.Timeout != 0:
		return time.Now().Add(d.D.Timeout)
	default:
		return time.Now().Add(def)
	}
}
