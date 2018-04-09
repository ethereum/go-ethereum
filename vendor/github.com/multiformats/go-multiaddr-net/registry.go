package manet

import (
	"fmt"
	"net"
	"sync"

	ma "github.com/multiformats/go-multiaddr"
)

// FromNetAddrFunc is a generic function which converts a net.Addr to Multiaddress
type FromNetAddrFunc func(a net.Addr) (ma.Multiaddr, error)

// ToNetAddrFunc is a generic function which converts a Multiaddress to net.Addr
type ToNetAddrFunc func(ma ma.Multiaddr) (net.Addr, error)

var defaultCodecs *CodecMap

func init() {
	defaultCodecs = NewCodecMap()
	defaultCodecs.RegisterNetCodec(tcpAddrSpec)
	defaultCodecs.RegisterNetCodec(udpAddrSpec)
	defaultCodecs.RegisterNetCodec(ip4AddrSpec)
	defaultCodecs.RegisterNetCodec(ip6AddrSpec)
	defaultCodecs.RegisterNetCodec(ipnetAddrSpec)
}

// CodecMap holds a map of NetCodecs indexed by their Protocol ID
// along with parsers for the addresses they use.
// It is used to keep a list of supported network address codecs (protocols
// which addresses can be converted to and from multiaddresses).
type CodecMap struct {
	codecs       map[string]*NetCodec
	addrParsers  map[string]FromNetAddrFunc
	maddrParsers map[string]ToNetAddrFunc
	lk           sync.Mutex
}

// NewCodecMap initializes and returns a CodecMap object.
func NewCodecMap() *CodecMap {
	return &CodecMap{
		codecs:       make(map[string]*NetCodec),
		addrParsers:  make(map[string]FromNetAddrFunc),
		maddrParsers: make(map[string]ToNetAddrFunc),
	}
}

// NetCodec is used to identify a network codec, that is, a network type for
// which we are able to translate multiaddresses into standard Go net.Addr
// and back.
type NetCodec struct {
	// NetAddrNetworks is an array of strings that may be returned
	// by net.Addr.Network() calls on addresses belonging to this type
	NetAddrNetworks []string

	// ProtocolName is the string value for Multiaddr address keys
	ProtocolName string

	// ParseNetAddr parses a net.Addr belonging to this type into a multiaddr
	ParseNetAddr FromNetAddrFunc

	// ConvertMultiaddr converts a multiaddr of this type back into a net.Addr
	ConvertMultiaddr ToNetAddrFunc

	// Protocol returns the multiaddr protocol struct for this type
	Protocol ma.Protocol
}

// RegisterNetCodec adds a new NetCodec to the default codecs.
func RegisterNetCodec(a *NetCodec) {
	defaultCodecs.RegisterNetCodec(a)
}

// RegisterNetCodec adds a new NetCodec to the CodecMap. This function is
// thread safe.
func (cm *CodecMap) RegisterNetCodec(a *NetCodec) {
	cm.lk.Lock()
	defer cm.lk.Unlock()
	cm.codecs[a.ProtocolName] = a
	for _, n := range a.NetAddrNetworks {
		cm.addrParsers[n] = a.ParseNetAddr
	}

	cm.maddrParsers[a.ProtocolName] = a.ConvertMultiaddr
}

var tcpAddrSpec = &NetCodec{
	ProtocolName:     "tcp",
	NetAddrNetworks:  []string{"tcp", "tcp4", "tcp6"},
	ParseNetAddr:     parseTCPNetAddr,
	ConvertMultiaddr: parseBasicNetMaddr,
}

var udpAddrSpec = &NetCodec{
	ProtocolName:     "udp",
	NetAddrNetworks:  []string{"udp", "udp4", "udp6"},
	ParseNetAddr:     parseUDPNetAddr,
	ConvertMultiaddr: parseBasicNetMaddr,
}

var ip4AddrSpec = &NetCodec{
	ProtocolName:     "ip4",
	NetAddrNetworks:  []string{"ip4"},
	ParseNetAddr:     parseIPNetAddr,
	ConvertMultiaddr: parseBasicNetMaddr,
}

var ip6AddrSpec = &NetCodec{
	ProtocolName:     "ip6",
	NetAddrNetworks:  []string{"ip6"},
	ParseNetAddr:     parseIPNetAddr,
	ConvertMultiaddr: parseBasicNetMaddr,
}

var ipnetAddrSpec = &NetCodec{
	ProtocolName:    "ip+net",
	NetAddrNetworks: []string{"ip+net"},
	ParseNetAddr:    parseIPPlusNetAddr,
	ConvertMultiaddr: func(ma.Multiaddr) (net.Addr, error) {
		return nil, fmt.Errorf("converting ip+net multiaddr not supported")
	},
}

func (cm *CodecMap) getAddrParser(net string) (FromNetAddrFunc, error) {
	cm.lk.Lock()
	defer cm.lk.Unlock()

	parser, ok := cm.addrParsers[net]
	if !ok {
		return nil, fmt.Errorf("unknown network %v", net)
	}
	return parser, nil
}

func (cm *CodecMap) getMaddrParser(name string) (ToNetAddrFunc, error) {
	cm.lk.Lock()
	defer cm.lk.Unlock()
	p, ok := cm.maddrParsers[name]
	if !ok {
		return nil, fmt.Errorf("network not supported: %s", name)
	}

	return p, nil
}
