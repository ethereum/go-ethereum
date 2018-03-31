package multiaddr

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// Protocol is a Multiaddr protocol description structure.
type Protocol struct {
	Code       int
	Size       int // a size of -1 indicates a length-prefixed variable size
	Name       string
	VCode      []byte
	Path       bool // indicates a path protocol (eg unix, http)
	Transcoder Transcoder
}

// You **MUST** register your multicodecs with
// https://github.com/multiformats/multicodec before adding them here.
//
// TODO: Use a single source of truth for all multicodecs instead of
// distributing them like this...
const (
	P_IP4   = 0x0004
	P_TCP   = 0x0006
	P_UDP   = 0x0111
	P_DCCP  = 0x0021
	P_IP6   = 0x0029
	P_QUIC  = 0x01CC
	P_SCTP  = 0x0084
	P_UDT   = 0x012D
	P_UTP   = 0x012E
	P_UNIX  = 0x0190
	P_IPFS  = 0x01A5
	P_HTTP  = 0x01E0
	P_HTTPS = 0x01BB
	P_ONION = 0x01BC
)

// These are special sizes
const (
	LengthPrefixedVarSize = -1
)

// Protocols is the list of multiaddr protocols supported by this module.
var Protocols = []Protocol{
	Protocol{P_IP4, 32, "ip4", CodeToVarint(P_IP4), false, TranscoderIP4},
	Protocol{P_TCP, 16, "tcp", CodeToVarint(P_TCP), false, TranscoderPort},
	Protocol{P_UDP, 16, "udp", CodeToVarint(P_UDP), false, TranscoderPort},
	Protocol{P_DCCP, 16, "dccp", CodeToVarint(P_DCCP), false, TranscoderPort},
	Protocol{P_IP6, 128, "ip6", CodeToVarint(P_IP6), false, TranscoderIP6},
	// these require varint:
	Protocol{P_SCTP, 16, "sctp", CodeToVarint(P_SCTP), false, TranscoderPort},
	Protocol{P_ONION, 96, "onion", CodeToVarint(P_ONION), false, TranscoderOnion},
	Protocol{P_UTP, 0, "utp", CodeToVarint(P_UTP), false, nil},
	Protocol{P_UDT, 0, "udt", CodeToVarint(P_UDT), false, nil},
	Protocol{P_QUIC, 0, "quic", CodeToVarint(P_QUIC), false, nil},
	Protocol{P_HTTP, 0, "http", CodeToVarint(P_HTTP), false, nil},
	Protocol{P_HTTPS, 0, "https", CodeToVarint(P_HTTPS), false, nil},
	Protocol{P_IPFS, LengthPrefixedVarSize, "ipfs", CodeToVarint(P_IPFS), false, TranscoderIPFS},
	Protocol{P_UNIX, LengthPrefixedVarSize, "unix", CodeToVarint(P_UNIX), true, TranscoderUnix},
}

func AddProtocol(p Protocol) error {
	for _, pt := range Protocols {
		if pt.Code == p.Code {
			return fmt.Errorf("protocol code %d already taken by %q", p.Code, pt.Name)
		}
		if pt.Name == p.Name {
			return fmt.Errorf("protocol by the name %q already exists", p.Name)
		}
	}

	Protocols = append(Protocols, p)
	return nil
}

// ProtocolWithName returns the Protocol description with given string name.
func ProtocolWithName(s string) Protocol {
	for _, p := range Protocols {
		if p.Name == s {
			return p
		}
	}
	return Protocol{}
}

// ProtocolWithCode returns the Protocol description with given protocol code.
func ProtocolWithCode(c int) Protocol {
	for _, p := range Protocols {
		if p.Code == c {
			return p
		}
	}
	return Protocol{}
}

// ProtocolsWithString returns a slice of protocols matching given string.
func ProtocolsWithString(s string) ([]Protocol, error) {
	s = strings.Trim(s, "/")
	sp := strings.Split(s, "/")
	if len(sp) == 0 {
		return nil, nil
	}

	t := make([]Protocol, len(sp))
	for i, name := range sp {
		p := ProtocolWithName(name)
		if p.Code == 0 {
			return nil, fmt.Errorf("no protocol with name: %s", name)
		}
		t[i] = p
	}
	return t, nil
}

// CodeToVarint converts an integer to a varint-encoded []byte
func CodeToVarint(num int) []byte {
	buf := make([]byte, (num/7)+1) // varint package is uint64
	n := binary.PutUvarint(buf, uint64(num))
	return buf[:n]
}

// VarintToCode converts a varint-encoded []byte to an integer protocol code
func VarintToCode(buf []byte) int {
	num, _, err := ReadVarintCode(buf)
	if err != nil {
		panic(err)
	}
	return num
}

// ReadVarintCode reads a varint code from the beginning of buf.
// returns the code, and the number of bytes read.
func ReadVarintCode(buf []byte) (int, int, error) {
	num, n := binary.Uvarint(buf)
	if n < 0 {
		return 0, 0, fmt.Errorf("varints larger than uint64 not yet supported")
	}
	return int(num), n, nil
}
