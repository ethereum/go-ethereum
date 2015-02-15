package discover

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	PublicKeyBits = 512
	idBits        = 256
)

var (
	hashfunc = crypto.SHA256
	// idSize = hashfunc.Size()
)

type NodeID [idBits / 8]byte

// Node represents a host on the network.
type Node struct {
	PublicKey PublicKey
	IP        net.IP
	NodeID    NodeID

	DiscPort int // UDP listening port for discovery protocol
	TCPPort  int // TCP listening port for RLPx

	active time.Time
}

func newNode(pubkey PublicKey, id NodeID, addr *net.UDPAddr) *Node {
	return &Node{
		PublicKey: pubkey,
		IP:        addr.IP,
		DiscPort:  addr.Port,
		TCPPort:   addr.Port,
		active:    time.Now(),
		NodeID:    id,
	}
}

func Hash(id PublicKey) (a NodeID) {
	copy(a[:], ethcrypto.Sha256(id[:]))
	return
}

func (n *Node) isValid() bool {
	// TODO: don't accept localhost, LAN addresses from internet hosts
	return !n.IP.IsMulticast() && !n.IP.IsUnspecified() && n.TCPPort != 0 && n.DiscPort != 0
}

// The string representation of a Node is a URL.
// Please see ParseNode for a description of the format.
func (n *Node) String() string {
	addr := net.TCPAddr{IP: n.IP, Port: n.TCPPort}
	u := url.URL{
		Scheme: "enode",
		User:   url.User(fmt.Sprintf("%x", n.PublicKey[:])),
		Host:   addr.String(),
	}
	if n.DiscPort != n.TCPPort {
		u.RawQuery = "discport=" + strconv.Itoa(n.DiscPort)
	}
	return u.String()
}

// ParseNode parses a node URL.
//
// A node URL has scheme "enode".
//
// The hexadecimal node PublicKey is encoded in the username portion of the
// URL, separated from the host by an @ sign. The hostname can only be
// given as an IP address, DNS domain names are not allowed. The port
// in the host name section is the TCP listening port. If the TCP and
// UDP (discovery) ports differ, the UDP port is specified as query
// parameter "discport".
//
// In the following example, the node URL describes
// a node with IP address 10.3.58.6, TCP listening port 30303
// and UDP discovery port 30301.
//
//    enode://<hex node id>@10.3.58.6:30303?discport=30301
func ParseNode(rawurl string) (*Node, error) {
	var n Node
	u, err := url.Parse(rawurl)
	if u.Scheme != "enode" {
		return nil, errors.New("invalid URL scheme, want \"enode\"")
	}
	if u.User == nil {
		return nil, errors.New("does not contain node PublicKey")
	}
	if n.PublicKey, err = HexPublicKey(u.User.String()); err != nil {
		return nil, fmt.Errorf("invalid node PublicKey (%v)", err)
	}
	n.NodeID = Hash(n.PublicKey)
	ip, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, fmt.Errorf("invalid host: %v", err)
	}
	if n.IP = net.ParseIP(ip); n.IP == nil {
		return nil, errors.New("invalid IP address")
	}
	if n.TCPPort, err = strconv.Atoi(port); err != nil {
		return nil, errors.New("invalid port")
	}
	qv := u.Query()
	if qv.Get("discport") == "" {
		n.DiscPort = n.TCPPort
	} else {
		if n.DiscPort, err = strconv.Atoi(qv.Get("discport")); err != nil {
			return nil, errors.New("invalid discport in query")
		}
	}
	return &n, nil
}

// MustParseNode parses a node URL. It panics if the URL is not valid.
func MustParseNode(rawurl string) *Node {
	n, err := ParseNode(rawurl)
	if err != nil {
		panic("invalid node URL: " + err.Error())
	}
	return n
}

func (n Node) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, rpcNode{IP: n.IP.String(), Port: uint16(n.TCPPort), PublicKey: n.PublicKey})
}
func (n *Node) DecodeRLP(s *rlp.Stream) (err error) {
	var ext rpcNode
	if err = s.Decode(&ext); err == nil {
		n.TCPPort = int(ext.Port)
		n.DiscPort = int(ext.Port)
		n.PublicKey = ext.PublicKey
		if n.IP = net.ParseIP(ext.IP); n.IP == nil {
			return errors.New("invalid IP string")
		}
	}
	return err
}

// PublicKey is a unique identifier for each node.
// The node identifier is a marshaled elliptic curve public key.
type PublicKey [PublicKeyBits / 8]byte

// PublicKey prints as a long hexadecimal number.
func (n PublicKey) String() string {
	return fmt.Sprintf("%#x", n[:])
}

// The Go syntax representation of a PublicKey is a call to HexPublicKey.
func (n PublicKey) GoString() string {
	return fmt.Sprintf("discover.HexPublicKey(\"%#x\")", n[:])
}

// HexPublicKey converts a hex string to a PublicKey.
// The string may be prefixed with 0x.
func HexPublicKey(in string) (PublicKey, error) {
	if strings.HasPrefix(in, "0x") {
		in = in[2:]
	}
	var id PublicKey
	b, err := hex.DecodeString(in)
	if err != nil {
		return id, err
	} else if len(b) != len(id) {
		return id, fmt.Errorf("wrong length, need %d hex bytes", len(id))
	}
	copy(id[:], b)
	return id, nil
}

// MustHexPublicKey converts a hex string to a PublicKey.
// It panics if the string is not a valid PublicKey.
func MustHexPublicKey(in string) PublicKey {
	id, err := HexPublicKey(in)
	if err != nil {
		panic(err)
	}
	return id
}

// PublicKey returns a marshaled representation of the given public key.
func exportPublicKey(pub *ecdsa.PublicKey) PublicKey {
	var id PublicKey
	pbytes := elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	if len(pbytes)-1 != len(id) {
		panic(fmt.Errorf("need %d bit pubkey, got %d bits", (len(id)+1)*8, len(pbytes)))
	}
	copy(id[:], pbytes[1:])
	return id
}

// recoverPublicKey computes the public key used to sign the
// given hash from the signature.
func recoverPublicKey(hash, sig []byte) (id PublicKey, err error) {
	pubkey, err := secp256k1.RecoverPubkey(hash, sig)
	if err != nil {
		return id, err
	}
	if len(pubkey)-1 != len(id) {
		return id, fmt.Errorf("recovered pubkey has %d bits, want %d bits", len(pubkey)*8, (len(id)+1)*8)
	}
	for i := range id {
		id[i] = pubkey[i+1]
	}
	return id, nil
}

// distcmp compares the distances a->target and b->target.
// Returns -1 if a is closer to target, 1 if b is closer to target
// and 0 if they are equal.
func distcmp(target, a, b NodeID) int {
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

// table of leading zero counts for bytes [0..255]
var lzcount = [256]int{
	8, 7, 6, 6, 5, 5, 5, 5,
	4, 4, 4, 4, 4, 4, 4, 4,
	3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
}

// logdist returns the logarithmic distance between a and b, log2(a ^ b).
func logdist(a, b NodeID) int {
	lz := 0
	for i := range a {
		x := a[i] ^ b[i]
		if x == 0 {
			lz += 8
		} else {
			lz += lzcount[x]
			break
		}
	}
	return len(a)*8 - lz
}

// randomNodeID returns a random NodeID such that logdist(a, b) == n
func randomNodeID(a NodeID, n int) (b NodeID) {
	if n == 0 {
		return a
	}
	// flip bit at position n, fill the rest with random bits
	b = a
	pos := len(a) - n/8 - 1
	bit := byte(0x01) << (byte(n%8) - 1)
	if bit == 0 {
		pos++
		bit = 0x80
	}
	b[pos] = a[pos]&^bit | ^a[pos]&bit // TODO: randomize end bits
	for i := pos + 1; i < len(a); i++ {
		b[i] = byte(rand.Intn(255))
	}
	return b
}
