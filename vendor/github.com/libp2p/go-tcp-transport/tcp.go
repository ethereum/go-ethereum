package tcp

import (
	"context"
	"fmt"
	"net"
	"sync"

	logging "github.com/ipfs/go-log"
	tpt "github.com/libp2p/go-libp2p-transport"
	reuseport "github.com/libp2p/go-reuseport"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	mafmt "github.com/whyrusleeping/mafmt"
)

var log = logging.Logger("tcp-tpt")

type TcpTransport struct {
	dlock   sync.Mutex
	dialers map[string]tpt.Dialer

	llock     sync.Mutex
	listeners map[string]tpt.Listener
}

var _ tpt.Transport = &TcpTransport{}

// NewTCPTransport creates a tcp transport object that tracks dialers and listeners
// created. It represents an entire tcp stack (though it might not necessarily be)
func NewTCPTransport() *TcpTransport {
	return &TcpTransport{
		dialers:   make(map[string]tpt.Dialer),
		listeners: make(map[string]tpt.Listener),
	}
}

func (t *TcpTransport) Dialer(laddr ma.Multiaddr, opts ...tpt.DialOpt) (tpt.Dialer, error) {
	if laddr == nil {
		zaddr, err := ma.NewMultiaddr("/ip4/0.0.0.0/tcp/0")
		if err != nil {
			return nil, err
		}
		laddr = zaddr
	}
	t.dlock.Lock()
	defer t.dlock.Unlock()
	s := laddr.String()
	d, found := t.dialers[s]
	if found {
		return d, nil
	}
	var doReuse bool
	for _, o := range opts {
		switch o := o.(type) {
		case tpt.ReuseportOpt:
			doReuse = bool(o)
		default:
			return nil, fmt.Errorf("unrecognized option: %#v", o)
		}
	}

	tcpd, err := t.newTcpDialer(laddr, doReuse)
	if err != nil {
		return nil, err
	}

	t.dialers[s] = tcpd
	return tcpd, nil
}

func (t *TcpTransport) Listen(laddr ma.Multiaddr) (tpt.Listener, error) {
	if !t.Matches(laddr) {
		return nil, fmt.Errorf("tcp transport cannot listen on %q", laddr)
	}

	t.llock.Lock()
	defer t.llock.Unlock()
	s := laddr.String()
	l, found := t.listeners[s]
	if found {
		return l, nil
	}

	list, err := manetListen(laddr)
	if err != nil {
		return nil, err
	}

	tlist := &tcpListener{
		list:      list,
		transport: t,
	}

	t.listeners[s] = tlist
	return tlist, nil
}

func manetListen(addr ma.Multiaddr) (manet.Listener, error) {
	network, naddr, err := manet.DialArgs(addr)
	if err != nil {
		return nil, err
	}

	if ReuseportIsAvailable() {
		nl, err := reuseport.Listen(network, naddr)
		if err == nil {
			// hey, it worked!
			return manet.WrapNetListener(nl)
		}
		// reuseport is available, but we failed to listen. log debug, and retry normally.
		log.Debugf("reuseport available, but failed to listen: %s %s, %s", network, naddr, err)
	}

	// either reuseport not available, or it failed. try normally.
	return manet.Listen(addr)
}

func (t *TcpTransport) Matches(a ma.Multiaddr) bool {
	return mafmt.TCP.Matches(a)
}

type tcpDialer struct {
	laddr ma.Multiaddr

	doReuse bool

	rd       reuseport.Dialer
	madialer manet.Dialer
	pattern  mafmt.Pattern

	transport tpt.Transport
}

var _ tpt.Dialer = &tcpDialer{}

func maddrToTcp(addr ma.Multiaddr) (*net.TCPAddr, error) {
	la, err := manet.ToNetAddr(addr)
	if err != nil {
		return nil, err // something wrong with addr.
	}
	latcp, ok := la.(*net.TCPAddr)
	if !ok {
		return nil, fmt.Errorf("not a tcp multiaddr: %s", addr)
	}
	return latcp, nil
}

func (t *TcpTransport) newTcpDialer(laddr ma.Multiaddr, doReuse bool) (*tcpDialer, error) {
	// get the local net.Addr manually
	la, err := maddrToTcp(laddr)
	if err != nil {
		return nil, err
	}

	var pattern mafmt.Pattern
	if TCP4.Matches(laddr) {
		pattern = TCP4
	} else if TCP6.Matches(laddr) {
		pattern = TCP6
	} else {
		return nil, fmt.Errorf("local addr did not match TCP4 or TCP6: %s", laddr)
	}

	// Ignore the port when constructing the default (non-reuseport) dialer.
	labase := *la
	labase.Port = 0

	dialer := &tcpDialer{
		laddr:   laddr,
		pattern: pattern,
		madialer: manet.Dialer{
			Dialer: net.Dialer{
				LocalAddr: &labase,
			},
		},
		transport: t,
	}

	if doReuse && ReuseportIsAvailable() {
		dialer.doReuse = true
		dialer.rd = reuseport.Dialer{
			D: net.Dialer{
				LocalAddr: la,
			},
		}
	}
	return dialer, nil
}

func (d *tcpDialer) Dial(raddr ma.Multiaddr) (tpt.Conn, error) {
	return d.DialContext(context.Background(), raddr)
}

func (d *tcpDialer) DialContext(ctx context.Context, raddr ma.Multiaddr) (tpt.Conn, error) {
	var c manet.Conn
	var err error
	if d.doReuse {
		c, err = d.reuseDial(ctx, raddr)
	} else {
		c, err = d.madialer.DialContext(ctx, raddr)
	}

	if err != nil {
		return nil, err
	}

	return &tcpConn{
		Conn: c,
		t:    d.transport,
	}, nil
}

func (d *tcpDialer) reuseDial(ctx context.Context, raddr ma.Multiaddr) (manet.Conn, error) {
	network, netraddr, err := manet.DialArgs(raddr)
	if err != nil {
		return nil, err
	}

	rpev := log.EventBegin(ctx, "tptDialReusePort", logging.LoggableMap{
		"raddr": raddr,
	})

	con, err := d.rd.DialContext(ctx, network, netraddr)
	if err == nil {
		rpev.Done()
		return manet.WrapNetConn(con)
	}
	rpev.SetError(err)
	rpev.Done()

	if !ReuseErrShouldRetry(err) {
		return nil, err
	}

	return d.madialer.DialContext(ctx, raddr)
}

var TCP4 = mafmt.And(mafmt.Base(ma.P_IP4), mafmt.Base(ma.P_TCP))
var TCP6 = mafmt.And(mafmt.Base(ma.P_IP6), mafmt.Base(ma.P_TCP))

func (d *tcpDialer) Matches(a ma.Multiaddr) bool {
	return d.pattern.Matches(a)
}

type tcpListener struct {
	list      manet.Listener
	transport tpt.Transport
}

var _ tpt.Listener = &tcpListener{}

func (d *tcpListener) Accept() (tpt.Conn, error) {
	c, err := d.list.Accept()
	if err != nil {
		return nil, err
	}

	return &tcpConn{
		Conn: c,
		t:    d.transport,
	}, nil
}

func (d *tcpListener) Addr() net.Addr {
	return d.list.Addr()
}

func (t *tcpListener) Multiaddr() ma.Multiaddr {
	return t.list.Multiaddr()
}

func (t *tcpListener) NetListener() net.Listener {
	return t.list.NetListener()
}

func (d *tcpListener) Close() error {
	return d.list.Close()
}

type tcpConn struct {
	manet.Conn
	t tpt.Transport
}

var _ tpt.Conn = &tcpConn{}

func (c *tcpConn) Transport() tpt.Transport {
	return c.t
}
