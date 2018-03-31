// Package manet provides Multiaddr
// (https://github.com/multiformats/go-multiaddr) specific versions of common
// functions in Go's standard `net` package. This means wrappers of standard
// net symbols like `net.Dial` and `net.Listen`, as well as conversion to
// and from `net.Addr`.
package manet

import (
	"context"
	"fmt"
	"net"

	ma "github.com/multiformats/go-multiaddr"
)

// Conn is the equivalent of a net.Conn object. It is the
// result of calling the Dial or Listen functions in this
// package, with associated local and remote Multiaddrs.
type Conn interface {
	net.Conn

	// LocalMultiaddr returns the local Multiaddr associated
	// with this connection
	LocalMultiaddr() ma.Multiaddr

	// RemoteMultiaddr returns the remote Multiaddr associated
	// with this connection
	RemoteMultiaddr() ma.Multiaddr
}

// WrapNetConn wraps a net.Conn object with a Multiaddr
// friendly Conn.
func WrapNetConn(nconn net.Conn) (Conn, error) {
	if nconn == nil {
		return nil, fmt.Errorf("failed to convert nconn.LocalAddr: nil")
	}

	laddr, err := FromNetAddr(nconn.LocalAddr())
	if err != nil {
		return nil, fmt.Errorf("failed to convert nconn.LocalAddr: %s", err)
	}

	raddr, err := FromNetAddr(nconn.RemoteAddr())
	if err != nil {
		return nil, fmt.Errorf("failed to convert nconn.RemoteAddr: %s", err)
	}

	return &maConn{
		Conn:  nconn,
		laddr: laddr,
		raddr: raddr,
	}, nil
}

// maConn implements the Conn interface. It's a thin wrapper
// around a net.Conn
type maConn struct {
	net.Conn
	laddr ma.Multiaddr
	raddr ma.Multiaddr
}

// LocalMultiaddr returns the local address associated with
// this connection
func (c *maConn) LocalMultiaddr() ma.Multiaddr {
	return c.laddr
}

// RemoteMultiaddr returns the remote address associated with
// this connection
func (c *maConn) RemoteMultiaddr() ma.Multiaddr {
	return c.raddr
}

// Dialer contains options for connecting to an address. It
// is effectively the same as net.Dialer, but its LocalAddr
// and RemoteAddr options are Multiaddrs, instead of net.Addrs.
type Dialer struct {

	// Dialer is just an embedded net.Dialer, with all its options.
	net.Dialer

	// LocalAddr is the local address to use when dialing an
	// address. The address must be of a compatible type for the
	// network being dialed.
	// If nil, a local address is automatically chosen.
	LocalAddr ma.Multiaddr
}

// Dial connects to a remote address, using the options of the
// Dialer. Dialer uses an underlying net.Dialer to Dial a
// net.Conn, then wraps that in a Conn object (with local and
// remote Multiaddrs).
func (d *Dialer) Dial(remote ma.Multiaddr) (Conn, error) {
	return d.DialContext(context.Background(), remote)
}

// DialContext allows to provide a custom context to Dial().
func (d *Dialer) DialContext(ctx context.Context, remote ma.Multiaddr) (Conn, error) {
	// if a LocalAddr is specified, use it on the embedded dialer.
	if d.LocalAddr != nil {
		// convert our multiaddr to net.Addr friendly
		naddr, err := ToNetAddr(d.LocalAddr)
		if err != nil {
			return nil, err
		}

		// set the dialer's LocalAddr as naddr
		d.Dialer.LocalAddr = naddr
	}

	// get the net.Dial friendly arguments from the remote addr
	rnet, rnaddr, err := DialArgs(remote)
	if err != nil {
		return nil, err
	}

	// ok, Dial!
	var nconn net.Conn
	switch rnet {
	case "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6":
		nconn, err = d.Dialer.DialContext(ctx, rnet, rnaddr)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unrecognized network: %s", rnet)
	}

	// get local address (pre-specified or assigned within net.Conn)
	local := d.LocalAddr
	if local == nil {
		local, err = FromNetAddr(nconn.LocalAddr())
		if err != nil {
			return nil, err
		}
	}

	return &maConn{
		Conn:  nconn,
		laddr: local,
		raddr: remote,
	}, nil
}

// Dial connects to a remote address. It uses an underlying net.Conn,
// then wraps it in a Conn object (with local and remote Multiaddrs).
func Dial(remote ma.Multiaddr) (Conn, error) {
	return (&Dialer{}).Dial(remote)
}

// A Listener is a generic network listener for stream-oriented protocols.
// it uses an embedded net.Listener, overriding net.Listener.Accept to
// return a Conn and providing Multiaddr.
type Listener interface {

	// NetListener returns the embedded net.Listener. Use with caution.
	NetListener() net.Listener

	// Accept waits for and returns the next connection to the listener.
	// Returns a Multiaddr friendly Conn
	Accept() (Conn, error)

	// Close closes the listener.
	// Any blocked Accept operations will be unblocked and return errors.
	Close() error

	// Multiaddr returns the listener's (local) Multiaddr.
	Multiaddr() ma.Multiaddr

	// Addr returns the net.Listener's network address.
	Addr() net.Addr
}

// maListener implements Listener
type maListener struct {
	net.Listener
	laddr ma.Multiaddr
}

// NetListener returns the embedded net.Listener. Use with caution.
func (l *maListener) NetListener() net.Listener {
	return l.Listener
}

// Accept waits for and returns the next connection to the listener.
// Returns a Multiaddr friendly Conn
func (l *maListener) Accept() (Conn, error) {
	nconn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	raddr, err := FromNetAddr(nconn.RemoteAddr())
	if err != nil {
		return nil, fmt.Errorf("failed to convert connn.RemoteAddr: %s", err)
	}

	return &maConn{
		Conn:  nconn,
		laddr: l.laddr,
		raddr: raddr,
	}, nil
}

// Multiaddr returns the listener's (local) Multiaddr.
func (l *maListener) Multiaddr() ma.Multiaddr {
	return l.laddr
}

// Addr returns the listener's network address.
func (l *maListener) Addr() net.Addr {
	return l.Listener.Addr()
}

// Listen announces on the local network address laddr.
// The Multiaddr must be a "ThinWaist" stream-oriented network:
// ip4/tcp, ip6/tcp, (TODO: unix, unixpacket)
// See Dial for the syntax of laddr.
func Listen(laddr ma.Multiaddr) (Listener, error) {

	// get the net.Listen friendly arguments from the remote addr
	lnet, lnaddr, err := DialArgs(laddr)
	if err != nil {
		return nil, err
	}

	nl, err := net.Listen(lnet, lnaddr)
	if err != nil {
		return nil, err
	}

	// we want to fetch the new multiaddr from the listener, as it may
	// have resolved to some other value. WrapNetListener does it for us.
	return WrapNetListener(nl)
}

// WrapNetListener wraps a net.Listener with a manet.Listener.
func WrapNetListener(nl net.Listener) (Listener, error) {
	laddr, err := FromNetAddr(nl.Addr())
	if err != nil {
		return nil, err
	}

	return &maListener{
		Listener: nl,
		laddr:    laddr,
	}, nil
}

// A PacketConn is a generic packet oriented network connection which uses an
// underlying net.PacketConn, wrapped with the locally bound Multiaddr.
type PacketConn interface {
	Connection() net.PacketConn

	Multiaddr() ma.Multiaddr

	ReadFrom(b []byte) (int, ma.Multiaddr, error)
	WriteTo(b []byte, maddr ma.Multiaddr) (int, error)

	Close() error
}

// maPacketConn implements PacketConn
type maPacketConn struct {
	net.PacketConn
	laddr ma.Multiaddr
}

// Connection returns the embedded net.PacketConn.
func (l *maPacketConn) Connection() net.PacketConn {
	return l.PacketConn
}

// Multiaddr returns the bound local Multiaddr.
func (l *maPacketConn) Multiaddr() ma.Multiaddr {
	return l.laddr
}

func (l *maPacketConn) ReadFrom(b []byte) (int, ma.Multiaddr, error) {
	n, addr, err := l.PacketConn.ReadFrom(b)
	maddr, _ := FromNetAddr(addr)
	return n, maddr, err
}

func (l *maPacketConn) WriteTo(b []byte, maddr ma.Multiaddr) (int, error) {
	addr, err := ToNetAddr(maddr)
	if err != nil {
		return 0, err
	}
	return l.PacketConn.WriteTo(b, addr)
}

// ListenPacket announces on the local network address laddr.
// The Multiaddr must be a packet driven network, like udp4 or udp6.
// See Dial for the syntax of laddr.
func ListenPacket(laddr ma.Multiaddr) (PacketConn, error) {
	lnet, lnaddr, err := DialArgs(laddr)
	if err != nil {
		return nil, err
	}

	pc, err := net.ListenPacket(lnet, lnaddr)
	if err != nil {
		return nil, err
	}

	// We want to fetch the new multiaddr from the listener, as it may
	// have resolved to some other value. WrapPacketConn does this.
	return WrapPacketConn(pc)
}

// WrapPacketConn wraps a net.PacketConn with a manet.PacketConn.
func WrapPacketConn(pc net.PacketConn) (PacketConn, error) {
	laddr, err := FromNetAddr(pc.LocalAddr())
	if err != nil {
		return nil, err
	}

	return &maPacketConn{
		PacketConn: pc,
		laddr:      laddr,
	}, nil
}

// InterfaceMultiaddrs will return the addresses matching net.InterfaceAddrs
func InterfaceMultiaddrs() ([]ma.Multiaddr, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	maddrs := make([]ma.Multiaddr, len(addrs))
	for i, a := range addrs {
		maddrs[i], err = FromNetAddr(a)
		if err != nil {
			return nil, err
		}
	}
	return maddrs, nil
}

// AddrMatch returns the Multiaddrs that match the protocol stack on addr
func AddrMatch(match ma.Multiaddr, addrs []ma.Multiaddr) []ma.Multiaddr {

	// we should match transports entirely.
	p1s := match.Protocols()

	out := make([]ma.Multiaddr, 0, len(addrs))
	for _, a := range addrs {
		p2s := a.Protocols()
		if len(p1s) != len(p2s) {
			continue
		}

		match := true
		for i, p2 := range p2s {
			if p1s[i].Code != p2.Code {
				match = false
				break
			}
		}
		if match {
			out = append(out, a)
		}
	}
	return out
}
