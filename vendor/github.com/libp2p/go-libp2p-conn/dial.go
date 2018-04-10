package conn

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	addrutil "github.com/libp2p/go-addr-util"
	ci "github.com/libp2p/go-libp2p-crypto"
	iconn "github.com/libp2p/go-libp2p-interface-conn"
	ipnet "github.com/libp2p/go-libp2p-interface-pnet"
	lgbl "github.com/libp2p/go-libp2p-loggables"
	peer "github.com/libp2p/go-libp2p-peer"
	transport "github.com/libp2p/go-libp2p-transport"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	msmux "github.com/multiformats/go-multistream"
)

// DialTimeout is the maximum duration a Dial is allowed to take.
// This includes the time between dialing the raw network connection,
// protocol selection as well the handshake, if applicable.
var DialTimeout = 60 * time.Second

// Dialer is an object with a peer identity that can open connections.
//
// NewDialer must be used to instantiate new Dialer objects.
type Dialer struct {
	// LocalPeer is the identity of the local Peer.
	LocalPeer peer.ID

	// LocalAddrs is a set of local addresses to use.
	//LocalAddrs []ma.Multiaddr

	// Dialers are the sub-dialers usable by this dialer,
	// selected in order based on the address being dialed.
	Dialers []transport.Dialer

	// PrivateKey used to initialize a secure connection.
	// Warning: if PrivateKey is nil, connection will not be secured.
	PrivateKey ci.PrivKey

	// Protector makes dialer part of a private network.
	// It includes implementation details how connection are protected.
	// Can be nil, then dialer is in public network.
	Protector ipnet.Protector

	// Wrapper to wrap the raw connection. Can be nil.
	Wrapper ConnWrapper

	fallback transport.Dialer
}

// NewDialer creates a new Dialer object.
//
// Before any calls to Dial are made, underlying dialers must be added
// with AddDialer, and Protector (if any) must be set.
func NewDialer(p peer.ID, pk ci.PrivKey, wrap ConnWrapper) *Dialer {
	return &Dialer{
		LocalPeer:  p,
		PrivateKey: pk,
		Wrapper:    wrap,
		fallback:   new(transport.FallbackDialer),
	}
}

// String returns the string representation of this Dialer.
func (d *Dialer) String() string {
	return fmt.Sprintf("<Dialer %s ...>", d.LocalPeer)
}

// Dial connects to a peer over a particular address.
// The remote peer ID is only verified if secure connections are in use.
// It returns once the connection is established, the protocol negotiated,
// and the handshake complete (if applicable).
func (d *Dialer) Dial(ctx context.Context, raddr ma.Multiaddr, remote peer.ID) (c iconn.Conn, err error) {
	deadline := time.Now().Add(DialTimeout)
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	logdial := lgbl.Dial("conn", d.LocalPeer, remote, nil, raddr)
	logdial["encrypted"] = (d.PrivateKey != nil) // log wether this will be an encrypted dial or not.
	logdial["inPrivNet"] = (d.Protector != nil)

	defer log.EventBegin(ctx, "connDial", logdial).Done()

	if d.Protector == nil && ipnet.ForcePrivateNetwork {
		log.Error("tried to dial with no Private Network Protector but usage" +
			" of Private Networks is forced by the enviroment")
		return nil, ipnet.ErrNotInPrivateNetwork
	}

	defer func() {
		if err != nil {
			logdial["error"] = err.Error()
			logdial["dial"] = "failure"
		}
	}()

	maconn, err := d.rawConnDial(ctx, raddr, remote)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			maconn.Close()
		}
	}()

	if d.Protector != nil {
		maconn, err = d.Protector.Protect(maconn)
		if err != nil {
			return nil, err
		}
	}

	if d.Wrapper != nil {
		maconn = d.Wrapper(maconn)
	}

	cryptoProtoChoice := SecioTag
	if !iconn.EncryptConnections || d.PrivateKey == nil {
		cryptoProtoChoice = NoEncryptionTag
	}

	selectResult := make(chan error, 1)
	go func() {
		selectResult <- msmux.SelectProtoOrFail(cryptoProtoChoice, maconn)
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err = <-selectResult:
		if err != nil {
			return nil, err
		}
	}

	c = newSingleConn(ctx, d.LocalPeer, remote, maconn)
	if d.PrivateKey == nil || !iconn.EncryptConnections {
		log.Warning("dialer %s dialing INSECURELY %s at %s!", d, remote, raddr)
		return c, nil
	}

	c2, err := newSecureConn(ctx, d.PrivateKey, c)
	if err != nil {
		c.Close()
		return nil, err
	}

	// if the connection is not to whom we thought it would be...
	connRemote := c2.RemotePeer()
	if connRemote != remote {
		c2.Close()
		return nil, fmt.Errorf("misdial to %s through %s (got %s): %s", remote, raddr, connRemote, err)
	}

	logdial["dial"] = "success"
	return c2, nil
}

// AddDialer adds a sub-dialer usable by this dialer.
// Dialers added first will be selected first, based on the address.
func (d *Dialer) AddDialer(pd transport.Dialer) {
	d.Dialers = append(d.Dialers, pd)
}

// returns dialer that can dial the given address
func (d *Dialer) subDialerForAddr(raddr ma.Multiaddr) transport.Dialer {
	for _, pd := range d.Dialers {
		if pd.Matches(raddr) {
			return pd
		}
	}

	if d.fallback.Matches(raddr) {
		return d.fallback
	}

	return nil
}

// rawConnDial dials the underlying net.Conn + manet.Conns
func (d *Dialer) rawConnDial(ctx context.Context, raddr ma.Multiaddr, remote peer.ID) (transport.Conn, error) {
	if strings.HasPrefix(raddr.String(), "/ip4/0.0.0.0") {
		log.Event(ctx, "connDialZeroAddr", lgbl.Dial("conn", d.LocalPeer, remote, nil, raddr))
		return nil, fmt.Errorf("Attempted to connect to zero address: %s", raddr)
	}

	sd := d.subDialerForAddr(raddr)
	if sd == nil {
		return nil, fmt.Errorf("no dialer for %s", raddr)
	}

	return sd.DialContext(ctx, raddr)
}

func pickLocalAddr(laddrs []ma.Multiaddr, raddr ma.Multiaddr) (laddr ma.Multiaddr) {
	if len(laddrs) < 1 {
		return nil
	}

	// make sure that we ONLY use local addrs that match the remote addr.
	laddrs = manet.AddrMatch(raddr, laddrs)
	if len(laddrs) < 1 {
		return nil
	}

	// make sure that we ONLY use local addrs that CAN dial the remote addr.
	// filter out all the local addrs that aren't capable
	raddrIPLayer := ma.Split(raddr)[0]
	raddrIsLoopback := manet.IsIPLoopback(raddrIPLayer)
	raddrIsLinkLocal := manet.IsIP6LinkLocal(raddrIPLayer)
	laddrs = addrutil.FilterAddrs(laddrs, func(a ma.Multiaddr) bool {
		laddrIPLayer := ma.Split(a)[0]
		laddrIsLoopback := manet.IsIPLoopback(laddrIPLayer)
		laddrIsLinkLocal := manet.IsIP6LinkLocal(laddrIPLayer)
		if laddrIsLoopback { // our loopback addrs can only dial loopbacks.
			return raddrIsLoopback
		}
		if laddrIsLinkLocal {
			return raddrIsLinkLocal // out linklocal addrs can only dial link locals.
		}
		return true
	})

	// TODO pick with a good heuristic
	// we use a random one for now to prevent bad addresses from making nodes unreachable
	// with a random selection, multiple tries may work.
	return laddrs[rand.Intn(len(laddrs))]
}

// MultiaddrProtocolsMatch returns whether two multiaddrs match in protocol stacks.
func MultiaddrProtocolsMatch(a, b ma.Multiaddr) bool {
	ap := a.Protocols()
	bp := b.Protocols()

	if len(ap) != len(bp) {
		return false
	}

	for i, api := range ap {
		if api.Code != bp[i].Code {
			return false
		}
	}

	return true
}

// MultiaddrNetMatch returns the first Multiaddr found to match  network.
func MultiaddrNetMatch(tgt ma.Multiaddr, srcs []ma.Multiaddr) ma.Multiaddr {
	for _, a := range srcs {
		if MultiaddrProtocolsMatch(tgt, a) {
			return a
		}
	}
	return nil
}
