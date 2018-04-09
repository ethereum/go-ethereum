package libp2p

import (
	"context"
	"crypto/rand"
	"fmt"

	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	pnet "github.com/libp2p/go-libp2p-interface-pnet"
	metrics "github.com/libp2p/go-libp2p-metrics"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	swarm "github.com/libp2p/go-libp2p-swarm"
	transport "github.com/libp2p/go-libp2p-transport"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	mux "github.com/libp2p/go-stream-muxer"
	ma "github.com/multiformats/go-multiaddr"
	mplex "github.com/whyrusleeping/go-smux-multiplex"
	msmux "github.com/whyrusleeping/go-smux-multistream"
	yamux "github.com/whyrusleeping/go-smux-yamux"
)

// Config describes a set of settings for a libp2p node
type Config struct {
	Transports   []transport.Transport
	Muxer        mux.Transport
	ListenAddrs  []ma.Multiaddr
	PeerKey      crypto.PrivKey
	Peerstore    pstore.Peerstore
	Protector    pnet.Protector
	Reporter     metrics.Reporter
	DisableSecio bool
	EnableNAT    bool
}

type Option func(cfg *Config) error

func Transports(tpts ...transport.Transport) Option {
	return func(cfg *Config) error {
		cfg.Transports = append(cfg.Transports, tpts...)
		return nil
	}
}

func ListenAddrStrings(s ...string) Option {
	return func(cfg *Config) error {
		for _, addrstr := range s {
			a, err := ma.NewMultiaddr(addrstr)
			if err != nil {
				return err
			}
			cfg.ListenAddrs = append(cfg.ListenAddrs, a)
		}
		return nil
	}
}

func ListenAddrs(addrs ...ma.Multiaddr) Option {
	return func(cfg *Config) error {
		cfg.ListenAddrs = append(cfg.ListenAddrs, addrs...)
		return nil
	}
}

type transportEncOpt int

const (
	EncPlaintext = transportEncOpt(0)
	EncSecio     = transportEncOpt(1)
)

func TransportEncryption(tenc ...transportEncOpt) Option {
	return func(cfg *Config) error {
		if len(tenc) != 1 {
			return fmt.Errorf("can only specify a single transport encryption option right now")
		}

		// TODO: actually make this pluggable, otherwise tls will get tricky
		switch tenc[0] {
		case EncPlaintext:
			cfg.DisableSecio = true
		case EncSecio:
			// noop
		default:
			return fmt.Errorf("unrecognized transport encryption option: %d", tenc[0])
		}
		return nil
	}
}

func NoEncryption() Option {
	return TransportEncryption(EncPlaintext)
}

func NATPortMap() Option {
	return func(cfg *Config) error {
		cfg.EnableNAT = true
		return nil
	}
}

func Muxer(m mux.Transport) Option {
	return func(cfg *Config) error {
		if cfg.Muxer != nil {
			return fmt.Errorf("cannot specify multiple muxer options")
		}

		cfg.Muxer = m
		return nil
	}
}

func Peerstore(ps pstore.Peerstore) Option {
	return func(cfg *Config) error {
		if cfg.Peerstore != nil {
			return fmt.Errorf("cannot specify multiple peerstore options")
		}

		cfg.Peerstore = ps
		return nil
	}
}

func PrivateNetwork(prot pnet.Protector) Option {
	return func(cfg *Config) error {
		if cfg.Protector != nil {
			return fmt.Errorf("cannot specify multiple private network options")
		}

		cfg.Protector = prot
		return nil
	}
}

func BandwidthReporter(rep metrics.Reporter) Option {
	return func(cfg *Config) error {
		if cfg.Reporter != nil {
			return fmt.Errorf("cannot specify multiple bandwidth reporter options")
		}

		cfg.Reporter = rep
		return nil
	}
}

func Identity(sk crypto.PrivKey) Option {
	return func(cfg *Config) error {
		if cfg.PeerKey != nil {
			return fmt.Errorf("cannot specify multiple identities")
		}

		cfg.PeerKey = sk
		return nil
	}
}

func New(ctx context.Context, opts ...Option) (host.Host, error) {
	var cfg Config
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	return newWithCfg(ctx, &cfg)
}

func newWithCfg(ctx context.Context, cfg *Config) (host.Host, error) {
	// If no key was given, generate a random 2048 bit RSA key
	if cfg.PeerKey == nil {
		priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
		if err != nil {
			return nil, err
		}
		cfg.PeerKey = priv
	}

	// Obtain Peer ID from public key
	pid, err := peer.IDFromPublicKey(cfg.PeerKey.GetPublic())
	if err != nil {
		return nil, err
	}

	// Create a new blank peerstore if none was passed in
	ps := cfg.Peerstore
	if ps == nil {
		ps = pstore.NewPeerstore()
	}

	// Set default muxer if none was passed in
	muxer := cfg.Muxer
	if muxer == nil {
		muxer = DefaultMuxer()
	}

	// If secio is disabled, don't add our private key to the peerstore
	if !cfg.DisableSecio {
		ps.AddPrivKey(pid, cfg.PeerKey)
		ps.AddPubKey(pid, cfg.PeerKey.GetPublic())
	}

	swrm, err := swarm.NewSwarmWithProtector(ctx, cfg.ListenAddrs, pid, ps, cfg.Protector, muxer, cfg.Reporter)
	if err != nil {
		return nil, err
	}

	netw := (*swarm.Network)(swrm)

	hostOpts := &bhost.HostOpts{}

	if cfg.EnableNAT {
		hostOpts.NATManager = bhost.NewNATManager(netw)
	}

	return bhost.NewHost(ctx, netw, hostOpts)
}

func DefaultMuxer() mux.Transport {
	// Set up stream multiplexer
	tpt := msmux.NewBlankTransport()

	// By default, support yamux and multiplex
	tpt.AddTransport("/yamux/1.0.0", yamux.DefaultTransport)
	tpt.AddTransport("/mplex/6.3.0", mplex.DefaultTransport)

	return tpt
}

func Defaults(cfg *Config) error {
	// Create a multiaddress that listens on a random port on all interfaces
	addr, err := ma.NewMultiaddr("/ip4/0.0.0.0/tcp/0")
	if err != nil {
		return err
	}

	cfg.ListenAddrs = []ma.Multiaddr{addr}
	cfg.Peerstore = pstore.NewPeerstore()
	cfg.Muxer = DefaultMuxer()
	return nil
}
