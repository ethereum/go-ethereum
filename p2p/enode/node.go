// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package enode

import (
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/bits"
	"net"
	"net/netip"
	"strings"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
)

var errMissingPrefix = errors.New("missing 'enr:' prefix for base64-encoded record")

// Node represents a host on the network.
type Node struct {
	r  enr.Record
	id ID

	// hostname tracks the DNS name of the node.
	hostname string

	// endpoint information
	ip  netip.Addr
	udp uint16
	tcp uint16
}

// New wraps a node record. The record must be valid according to the given
// identity scheme.
func New(validSchemes enr.IdentityScheme, r *enr.Record) (*Node, error) {
	if err := r.VerifySignature(validSchemes); err != nil {
		return nil, err
	}
	var id ID
	if n := copy(id[:], validSchemes.NodeAddr(r)); n != len(id) {
		return nil, fmt.Errorf("invalid node ID length %d, need %d", n, len(id))
	}
	return newNodeWithID(r, id), nil
}

func newNodeWithID(r *enr.Record, id ID) *Node {
	n := &Node{r: *r, id: id}
	// Set the preferred endpoint.
	// Here we decide between IPv4 and IPv6, choosing the 'most global' address.
	var ip4 netip.Addr
	var ip6 netip.Addr
	n.Load((*enr.IPv4Addr)(&ip4))
	n.Load((*enr.IPv6Addr)(&ip6))
	valid4 := validIP(ip4)
	valid6 := validIP(ip6)
	switch {
	case valid4 && valid6:
		if localityScore(ip4) >= localityScore(ip6) {
			n.setIP4(ip4)
		} else {
			n.setIP6(ip6)
		}
	case valid4:
		n.setIP4(ip4)
	case valid6:
		n.setIP6(ip6)
	default:
		n.setIPv4Ports()
	}
	return n
}

// validIP reports whether 'ip' is a valid node endpoint IP address.
func validIP(ip netip.Addr) bool {
	return ip.IsValid() && !ip.IsMulticast()
}

func localityScore(ip netip.Addr) int {
	switch {
	case ip.IsUnspecified():
		return 0
	case ip.IsLoopback():
		return 1
	case ip.IsLinkLocalUnicast():
		return 2
	case ip.IsPrivate():
		return 3
	default:
		return 4
	}
}

func (n *Node) setIP4(ip netip.Addr) {
	n.ip = ip
	n.setIPv4Ports()
}

func (n *Node) setIPv4Ports() {
	n.Load((*enr.UDP)(&n.udp))
	n.Load((*enr.TCP)(&n.tcp))
}

func (n *Node) setIP6(ip netip.Addr) {
	if ip.Is4In6() {
		n.setIP4(ip)
		return
	}
	n.ip = ip
	if err := n.Load((*enr.UDP6)(&n.udp)); err != nil {
		n.Load((*enr.UDP)(&n.udp))
	}
	if err := n.Load((*enr.TCP6)(&n.tcp)); err != nil {
		n.Load((*enr.TCP)(&n.tcp))
	}
}

// MustParse parses a node record or enode:// URL. It panics if the input is invalid.
func MustParse(rawurl string) *Node {
	n, err := Parse(ValidSchemes, rawurl)
	if err != nil {
		panic("invalid node: " + err.Error())
	}
	return n
}

// Parse decodes and verifies a base64-encoded node record.
func Parse(validSchemes enr.IdentityScheme, input string) (*Node, error) {
	if strings.HasPrefix(input, "enode://") {
		return ParseV4(input)
	}
	if !strings.HasPrefix(input, "enr:") {
		return nil, errMissingPrefix
	}
	bin, err := base64.RawURLEncoding.DecodeString(input[4:])
	if err != nil {
		return nil, err
	}
	var r enr.Record
	if err := rlp.DecodeBytes(bin, &r); err != nil {
		return nil, err
	}
	return New(validSchemes, &r)
}

// ID returns the node identifier.
func (n *Node) ID() ID {
	return n.id
}

// Seq returns the sequence number of the underlying record.
func (n *Node) Seq() uint64 {
	return n.r.Seq()
}

// Load retrieves an entry from the underlying record.
func (n *Node) Load(k enr.Entry) error {
	return n.r.Load(k)
}

// IP returns the IP address of the node.
func (n *Node) IP() net.IP {
	return net.IP(n.ip.AsSlice())
}

// IPAddr returns the IP address of the node.
func (n *Node) IPAddr() netip.Addr {
	return n.ip
}

// UDP returns the UDP port of the node.
func (n *Node) UDP() int {
	return int(n.udp)
}

// TCP returns the TCP port of the node.
func (n *Node) TCP() int {
	return int(n.tcp)
}

// WithHostname adds a DNS hostname to the node.
func (n *Node) WithHostname(hostname string) *Node {
	cpy := *n
	cpy.hostname = hostname
	return &cpy
}

// Hostname returns the DNS name assigned by WithHostname.
func (n *Node) Hostname() string {
	return n.hostname
}

// UDPEndpoint returns the announced UDP endpoint.
func (n *Node) UDPEndpoint() (netip.AddrPort, bool) {
	if !n.ip.IsValid() || n.ip.IsUnspecified() || n.udp == 0 {
		return netip.AddrPort{}, false
	}
	return netip.AddrPortFrom(n.ip, n.udp), true
}

// TCPEndpoint returns the announced TCP endpoint.
func (n *Node) TCPEndpoint() (netip.AddrPort, bool) {
	if !n.ip.IsValid() || n.ip.IsUnspecified() || n.tcp == 0 {
		return netip.AddrPort{}, false
	}
	return netip.AddrPortFrom(n.ip, n.tcp), true
}

// QUICEndpoint returns the announced QUIC endpoint.
func (n *Node) QUICEndpoint() (netip.AddrPort, bool) {
	var quic uint16
	if n.ip.Is4() || n.ip.Is4In6() {
		n.Load((*enr.QUIC)(&quic))
	} else if n.ip.Is6() {
		n.Load((*enr.QUIC6)(&quic))
	}
	if !n.ip.IsValid() || n.ip.IsUnspecified() || quic == 0 {
		return netip.AddrPort{}, false
	}
	return netip.AddrPortFrom(n.ip, quic), true
}

// Pubkey returns the secp256k1 public key of the node, if present.
func (n *Node) Pubkey() *ecdsa.PublicKey {
	var key ecdsa.PublicKey
	if n.Load((*Secp256k1)(&key)) != nil {
		return nil
	}
	return &key
}

// Record returns the node's record. The return value is a copy and may
// be modified by the caller.
func (n *Node) Record() *enr.Record {
	cpy := n.r
	return &cpy
}

// ValidateComplete checks whether n has a valid IP and UDP port.
// Deprecated: don't use this method.
func (n *Node) ValidateComplete() error {
	if !n.ip.IsValid() {
		return errors.New("missing IP address")
	}
	if n.ip.IsMulticast() || n.ip.IsUnspecified() {
		return errors.New("invalid IP (multicast/unspecified)")
	}
	if n.udp == 0 {
		return errors.New("missing UDP port")
	}
	// Validate the node key (on curve, etc.).
	var key Secp256k1
	return n.Load(&key)
}

// String returns the text representation of the record.
func (n *Node) String() string {
	if isNewV4(n) {
		return n.URLv4() // backwards-compatibility glue for NewV4 nodes
	}
	enc, _ := rlp.EncodeToBytes(&n.r) // always succeeds because record is valid
	b64 := base64.RawURLEncoding.EncodeToString(enc)
	return "enr:" + b64
}

// MarshalText implements encoding.TextMarshaler.
func (n *Node) MarshalText() ([]byte, error) {
	return []byte(n.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (n *Node) UnmarshalText(text []byte) error {
	dec, err := Parse(ValidSchemes, string(text))
	if err == nil {
		*n = *dec
	}
	return err
}

// ID is a unique identifier for each node.
type ID [32]byte

// Bytes returns a byte slice representation of the ID
func (n ID) Bytes() []byte {
	return n[:]
}

// ID prints as a long hexadecimal number.
func (n ID) String() string {
	return fmt.Sprintf("%x", n[:])
}

// GoString returns the Go syntax representation of a ID is a call to HexID.
func (n ID) GoString() string {
	return fmt.Sprintf("enode.HexID(\"%x\")", n[:])
}

// TerminalString returns a shortened hex string for terminal logging.
func (n ID) TerminalString() string {
	return hex.EncodeToString(n[:8])
}

// MarshalText implements the encoding.TextMarshaler interface.
func (n ID) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(n[:])), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (n *ID) UnmarshalText(text []byte) error {
	id, err := ParseID(string(text))
	if err != nil {
		return err
	}
	*n = id
	return nil
}

// HexID converts a hex string to an ID.
// The string may be prefixed with 0x.
// It panics if the string is not a valid ID.
func HexID(in string) ID {
	id, err := ParseID(in)
	if err != nil {
		panic(err)
	}
	return id
}

func ParseID(in string) (ID, error) {
	var id ID
	b, err := hex.DecodeString(strings.TrimPrefix(in, "0x"))
	if err != nil {
		return id, err
	} else if len(b) != len(id) {
		return id, fmt.Errorf("wrong length, want %d hex chars", len(id)*2)
	}
	copy(id[:], b)
	return id, nil
}

// DistCmp compares the distances a->target and b->target.
// Returns -1 if a is closer to target, 1 if b is closer to target
// and 0 if they are equal.
func DistCmp(target, a, b ID) int {
	for i := range target {
		da := a[i] ^ target[i]
		db := b[i] ^ target[i]
		if da > db {
			return 1
		} else if da < db {
			return -1
		}
	}
	return 0
}

// LogDist returns the logarithmic distance between a and b, log2(a ^ b).
func LogDist(a, b ID) int {
	lz := 0
	for i := range a {
		x := a[i] ^ b[i]
		if x == 0 {
			lz += 8
		} else {
			lz += bits.LeadingZeros8(x)
			break
		}
	}
	return len(a)*8 - lz
}
