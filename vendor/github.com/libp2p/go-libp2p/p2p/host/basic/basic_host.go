package basichost

import (
	"context"
	"io"
	"time"

	identify "github.com/libp2p/go-libp2p/p2p/protocol/identify"

	logging "github.com/ipfs/go-log"
	goprocess "github.com/jbenet/goprocess"
	circuit "github.com/libp2p/go-libp2p-circuit"
	ifconnmgr "github.com/libp2p/go-libp2p-interface-connmgr"
	metrics "github.com/libp2p/go-libp2p-metrics"
	mstream "github.com/libp2p/go-libp2p-metrics/stream"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	protocol "github.com/libp2p/go-libp2p-protocol"
	ma "github.com/multiformats/go-multiaddr"
	madns "github.com/multiformats/go-multiaddr-dns"
	msmux "github.com/multiformats/go-multistream"
)

var log = logging.Logger("basichost")

var (
	// DefaultNegotiationTimeout is the default value for HostOpts.NegotiationTimeout.
	DefaultNegotiationTimeout = time.Second * 60

	// DefaultAddrsFactory is the default value for HostOpts.AddrsFactory.
	DefaultAddrsFactory = func(addrs []ma.Multiaddr) []ma.Multiaddr { return addrs }
)

// AddrsFactory functions can be passed to New in order to override
// addresses returned by Addrs.
type AddrsFactory func([]ma.Multiaddr) []ma.Multiaddr

// Option is a type used to pass in options to the host.
//
// Deprecated in favor of HostOpts and NewHost.
type Option int

// NATPortMap makes the host attempt to open port-mapping in NAT devices
// for all its listeners. Pass in this option in the constructor to
// asynchronously a) find a gateway, b) open port mappings, c) republish
// port mappings periodically. The NATed addresses are included in the
// Host's Addrs() list.
//
// This option is deprecated in favor of HostOpts and NewHost.
const NATPortMap Option = iota

// BasicHost is the basic implementation of the host.Host interface. This
// particular host implementation:
//  * uses a protocol muxer to mux per-protocol streams
//  * uses an identity service to send + receive node information
//  * uses a nat service to establish NAT port mappings
type BasicHost struct {
	network    inet.Network
	mux        *msmux.MultistreamMuxer
	ids        *identify.IDService
	natmgr     NATManager
	addrs      AddrsFactory
	maResolver *madns.Resolver
	cmgr       ifconnmgr.ConnManager

	negtimeout time.Duration

	proc goprocess.Process

	bwc metrics.Reporter
}

// HostOpts holds options that can be passed to NewHost in order to
// customize construction of the *BasicHost.
type HostOpts struct {

	// MultistreamMuxer is essential for the *BasicHost and will use a sensible default value if omitted.
	MultistreamMuxer *msmux.MultistreamMuxer

	// NegotiationTimeout determines the read and write timeouts on streams.
	// If 0 or omitted, it will use DefaultNegotiationTimeout.
	// If below 0, timeouts on streams will be deactivated.
	NegotiationTimeout time.Duration

	// IdentifyService holds an implementation of the /ipfs/id/ protocol.
	// If omitted, a new *identify.IDService will be used.
	IdentifyService *identify.IDService

	// AddrsFactory holds a function which can be used to override or filter the result of Addrs.
	// If omitted, there's no override or filtering, and the results of Addrs and AllAddrs are the same.
	AddrsFactory AddrsFactory

	// MultiaddrResolves holds the go-multiaddr-dns.Resolver used for resolving
	// /dns4, /dns6, and /dnsaddr addresses before trying to connect to a peer.
	MultiaddrResolver *madns.Resolver

	// NATManager takes care of setting NAT port mappings, and discovering external addresses.
	// If omitted, this will simply be disabled.
	NATManager NATManager

	// BandwidthReporter is used for collecting aggregate metrics of the
	// bandwidth used by various protocols.
	BandwidthReporter metrics.Reporter

	// ConnManager is a libp2p connection manager
	ConnManager ifconnmgr.ConnManager

	// Relay indicates whether the host should use circuit relay transport
	EnableRelay bool

	// RelayOpts are options for the relay transport; only meaningful when Relay=true
	RelayOpts []circuit.RelayOpt
}

// NewHost constructs a new *BasicHost and activates it by attaching its stream and connection handlers to the given inet.Network.
func NewHost(ctx context.Context, net inet.Network, opts *HostOpts) (*BasicHost, error) {
	ctx, cancel := context.WithCancel(ctx)
	h := &BasicHost{
		network:    net,
		mux:        msmux.NewMultistreamMuxer(),
		negtimeout: DefaultNegotiationTimeout,
		addrs:      DefaultAddrsFactory,
		maResolver: madns.DefaultResolver,
	}

	h.proc = goprocess.WithTeardown(func() error {
		if h.natmgr != nil {
			h.natmgr.Close()
		}
		cancel()
		return h.Network().Close()
	})

	if opts.MultistreamMuxer != nil {
		h.mux = opts.MultistreamMuxer
	}

	if opts.IdentifyService != nil {
		h.ids = opts.IdentifyService
	} else {
		// we can't set this as a default above because it depends on the *BasicHost.
		h.ids = identify.NewIDService(h)
	}

	if uint64(opts.NegotiationTimeout) != 0 {
		h.negtimeout = opts.NegotiationTimeout
	}

	if opts.AddrsFactory != nil {
		h.addrs = opts.AddrsFactory
	}

	if opts.NATManager != nil {
		h.natmgr = opts.NATManager
	}

	if opts.MultiaddrResolver != nil {
		h.maResolver = opts.MultiaddrResolver
	}

	if opts.BandwidthReporter != nil {
		h.bwc = opts.BandwidthReporter
		h.ids.Reporter = opts.BandwidthReporter
	}

	if opts.ConnManager == nil {
		h.cmgr = &ifconnmgr.NullConnMgr{}
	} else {
		h.cmgr = opts.ConnManager
		net.Notify(h.cmgr.Notifee())
	}

	net.SetConnHandler(h.newConnHandler)
	net.SetStreamHandler(h.newStreamHandler)

	if opts.EnableRelay {
		err := circuit.AddRelayTransport(ctx, h, opts.RelayOpts...)
		if err != nil {
			h.Close()
			return nil, err
		}
	}

	return h, nil
}

// New constructs and sets up a new *BasicHost with given Network and options.
// Three options can be passed: NATPortMap, AddrsFactory, and metrics.Reporter.
// This function is deprecated in favor of NewHost and HostOpts.
func New(net inet.Network, opts ...interface{}) *BasicHost {
	hostopts := &HostOpts{}

	for _, o := range opts {
		switch o := o.(type) {
		case Option:
			switch o {
			case NATPortMap:
				hostopts.NATManager = newNatManager(net)
			}
		case metrics.Reporter:
			hostopts.BandwidthReporter = o
		case AddrsFactory:
			hostopts.AddrsFactory = AddrsFactory(o)
		case ifconnmgr.ConnManager:
			hostopts.ConnManager = o
		case *madns.Resolver:
			hostopts.MultiaddrResolver = o
		}
	}

	h, err := NewHost(context.Background(), net, hostopts)
	if err != nil {
		// this cannot happen with legacy options
		// plus we want to keep the (deprecated) legacy interface unchanged
		panic(err)
	}

	return h
}

// newConnHandler is the remote-opened conn handler for inet.Network
func (h *BasicHost) newConnHandler(c inet.Conn) {
	// Clear protocols on connecting to new peer to avoid issues caused
	// by misremembering protocols between reconnects
	h.Peerstore().SetProtocols(c.RemotePeer())
	h.ids.IdentifyConn(c)
}

// newStreamHandler is the remote-opened stream handler for inet.Network
// TODO: this feels a bit wonky
func (h *BasicHost) newStreamHandler(s inet.Stream) {
	before := time.Now()

	if h.negtimeout > 0 {
		if err := s.SetDeadline(time.Now().Add(h.negtimeout)); err != nil {
			log.Error("setting stream deadline: ", err)
			s.Reset()
			return
		}
	}

	lzc, protoID, handle, err := h.Mux().NegotiateLazy(s)
	took := time.Now().Sub(before)
	if err != nil {
		if err == io.EOF {
			logf := log.Debugf
			if took > time.Second*10 {
				logf = log.Warningf
			}
			logf("protocol EOF: %s (took %s)", s.Conn().RemotePeer(), took)
		} else {
			log.Infof("protocol mux failed: %s (took %s)", err, took)
		}
		s.Reset()
		return
	}

	s = &streamWrapper{
		Stream: s,
		rw:     lzc,
	}

	if h.negtimeout > 0 {
		if err := s.SetDeadline(time.Time{}); err != nil {
			log.Error("resetting stream deadline: ", err)
			s.Reset()
			return
		}
	}

	s.SetProtocol(protocol.ID(protoID))

	if h.bwc != nil {
		s = mstream.WrapStream(s, h.bwc)
	}
	log.Debugf("protocol negotiation took %s", took)

	go handle(protoID, s)
}

// ID returns the (local) peer.ID associated with this Host
func (h *BasicHost) ID() peer.ID {
	return h.Network().LocalPeer()
}

// Peerstore returns the Host's repository of Peer Addresses and Keys.
func (h *BasicHost) Peerstore() pstore.Peerstore {
	return h.Network().Peerstore()
}

// Network returns the Network interface of the Host
func (h *BasicHost) Network() inet.Network {
	return h.network
}

// Mux returns the Mux multiplexing incoming streams to protocol handlers
func (h *BasicHost) Mux() *msmux.MultistreamMuxer {
	return h.mux
}

// IDService returns
func (h *BasicHost) IDService() *identify.IDService {
	return h.ids
}

// SetStreamHandler sets the protocol handler on the Host's Mux.
// This is equivalent to:
//   host.Mux().SetHandler(proto, handler)
// (Threadsafe)
func (h *BasicHost) SetStreamHandler(pid protocol.ID, handler inet.StreamHandler) {
	h.Mux().AddHandler(string(pid), func(p string, rwc io.ReadWriteCloser) error {
		is := rwc.(inet.Stream)
		is.SetProtocol(protocol.ID(p))
		handler(is)
		return nil
	})
}

// SetStreamHandlerMatch sets the protocol handler on the Host's Mux
// using a matching function to do protocol comparisons
func (h *BasicHost) SetStreamHandlerMatch(pid protocol.ID, m func(string) bool, handler inet.StreamHandler) {
	h.Mux().AddHandlerWithFunc(string(pid), m, func(p string, rwc io.ReadWriteCloser) error {
		is := rwc.(inet.Stream)
		is.SetProtocol(protocol.ID(p))
		handler(is)
		return nil
	})
}

// RemoveStreamHandler returns ..
func (h *BasicHost) RemoveStreamHandler(pid protocol.ID) {
	h.Mux().RemoveHandler(string(pid))
}

// NewStream opens a new stream to given peer p, and writes a p2p/protocol
// header with given protocol.ID. If there is no connection to p, attempts
// to create one. If ProtocolID is "", writes no header.
// (Threadsafe)
func (h *BasicHost) NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (inet.Stream, error) {
	pref, err := h.preferredProtocol(p, pids)
	if err != nil {
		return nil, err
	}

	if pref != "" {
		return h.newStream(ctx, p, pref)
	}

	var protoStrs []string
	for _, pid := range pids {
		protoStrs = append(protoStrs, string(pid))
	}

	s, err := h.Network().NewStream(ctx, p)
	if err != nil {
		return nil, err
	}

	selected, err := msmux.SelectOneOf(protoStrs, s)
	if err != nil {
		s.Reset()
		return nil, err
	}
	selpid := protocol.ID(selected)
	s.SetProtocol(selpid)
	h.Peerstore().AddProtocols(p, selected)

	if h.bwc != nil {
		s = mstream.WrapStream(s, h.bwc)
	}

	return s, nil
}

func pidsToStrings(pids []protocol.ID) []string {
	out := make([]string, len(pids))
	for i, p := range pids {
		out[i] = string(p)
	}
	return out
}

func (h *BasicHost) preferredProtocol(p peer.ID, pids []protocol.ID) (protocol.ID, error) {
	pidstrs := pidsToStrings(pids)
	supported, err := h.Peerstore().SupportsProtocols(p, pidstrs...)
	if err != nil {
		return "", err
	}

	var out protocol.ID
	if len(supported) > 0 {
		out = protocol.ID(supported[0])
	}
	return out, nil
}

func (h *BasicHost) newStream(ctx context.Context, p peer.ID, pid protocol.ID) (inet.Stream, error) {
	s, err := h.Network().NewStream(ctx, p)
	if err != nil {
		return nil, err
	}

	s.SetProtocol(pid)

	if h.bwc != nil {
		s = mstream.WrapStream(s, h.bwc)
	}

	lzcon := msmux.NewMSSelect(s, string(pid))
	return &streamWrapper{
		Stream: s,
		rw:     lzcon,
	}, nil
}

// Connect ensures there is a connection between this host and the peer with
// given peer.ID. If there is not an active connection, Connect will issue a
// h.Network.Dial, and block until a connection is open, or an error is returned.
// Connect will absorb the addresses in pi into its internal peerstore.
// It will also resolve any /dns4, /dns6, and /dnsaddr addresses.
func (h *BasicHost) Connect(ctx context.Context, pi pstore.PeerInfo) error {
	// absorb addresses into peerstore
	h.Peerstore().AddAddrs(pi.ID, pi.Addrs, pstore.TempAddrTTL)

	cs := h.Network().ConnsToPeer(pi.ID)
	if len(cs) > 0 {
		return nil
	}

	resolved, err := h.resolveAddrs(ctx, h.Peerstore().PeerInfo(pi.ID))
	if err != nil {
		return err
	}
	h.Peerstore().AddAddrs(pi.ID, resolved, pstore.TempAddrTTL)

	return h.dialPeer(ctx, pi.ID)
}

func (h *BasicHost) resolveAddrs(ctx context.Context, pi pstore.PeerInfo) ([]ma.Multiaddr, error) {
	proto := ma.ProtocolWithCode(ma.P_IPFS).Name
	p2paddr, err := ma.NewMultiaddr("/" + proto + "/" + pi.ID.Pretty())
	if err != nil {
		return nil, err
	}

	var addrs []ma.Multiaddr
	for _, addr := range pi.Addrs {
		addrs = append(addrs, addr)
		if !madns.Matches(addr) {
			continue
		}

		reqaddr := addr.Encapsulate(p2paddr)
		resaddrs, err := h.maResolver.Resolve(ctx, reqaddr)
		if err != nil {
			log.Infof("error resolving %s: %s", reqaddr, err)
		}
		for _, res := range resaddrs {
			pi, err := pstore.InfoFromP2pAddr(res)
			if err != nil {
				log.Infof("error parsing %s: %s", res, err)
			}
			addrs = append(addrs, pi.Addrs...)
		}
	}

	return addrs, nil
}

// dialPeer opens a connection to peer, and makes sure to identify
// the connection once it has been opened.
func (h *BasicHost) dialPeer(ctx context.Context, p peer.ID) error {
	log.Debugf("host %s dialing %s", h.ID, p)
	c, err := h.Network().DialPeer(ctx, p)
	if err != nil {
		return err
	}

	// Clear protocols on connecting to new peer to avoid issues caused
	// by misremembering protocols between reconnects
	h.Peerstore().SetProtocols(p)

	// identify the connection before returning.
	done := make(chan struct{})
	go func() {
		h.ids.IdentifyConn(c)
		close(done)
	}()

	// respect don contexteone
	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	log.Debugf("host %s finished dialing %s", h.ID(), p)
	return nil
}

func (h *BasicHost) ConnManager() ifconnmgr.ConnManager {
	return h.cmgr
}

// Addrs returns listening addresses that are safe to announce to the network.
// The output is the same as AllAddrs, but processed by AddrsFactory.
func (h *BasicHost) Addrs() []ma.Multiaddr {
	return h.addrs(h.AllAddrs())
}

// AllAddrs returns all the addresses of BasicHost at this moment in time.
// It's ok to not include addresses if they're not available to be used now.
func (h *BasicHost) AllAddrs() []ma.Multiaddr {
	addrs, err := h.Network().InterfaceListenAddresses()
	if err != nil {
		log.Debug("error retrieving network interface addrs")
	}

	if h.ids != nil { // add external observed addresses
		addrs = append(addrs, h.ids.OwnObservedAddrs()...)
	}

	if h.natmgr != nil { // natmgr is nil if we do not use nat option.
		nat := h.natmgr.NAT()
		if nat != nil { // nat is nil if not ready, or no nat is available.
			addrs = append(addrs, nat.ExternalAddrs()...)
		}
	}

	return addrs
}

// Close shuts down the Host's services (network, etc).
func (h *BasicHost) Close() error {
	return h.proc.Close()
}

// GetBandwidthReporter exposes the Host's bandiwth metrics reporter
func (h *BasicHost) GetBandwidthReporter() metrics.Reporter {
	return h.bwc
}

type streamWrapper struct {
	inet.Stream
	rw io.ReadWriter
}

func (s *streamWrapper) Read(b []byte) (int, error) {
	return s.rw.Read(b)
}

func (s *streamWrapper) Write(b []byte) (int, error) {
	return s.rw.Write(b)
}
