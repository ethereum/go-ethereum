package discover

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

const nodeIDBits = 512

// Node represents a host on the network.
type Node struct {
	ID NodeID
	IP net.IP

	DiscPort int // UDP listening port for discovery protocol
	TCPPort  int // TCP listening port for RLPx
}

func newNode(id NodeID, addr *net.UDPAddr) *Node {
	return &Node{
		ID:       id,
		IP:       addr.IP,
		DiscPort: addr.Port,
		TCPPort:  addr.Port,
	}
}

func (n *Node) isValid() bool {
	// TODO: don't accept localhost, LAN addresses from internet hosts
	return !n.IP.IsMulticast() && !n.IP.IsUnspecified() && n.TCPPort != 0 && n.DiscPort != 0
}

func (n *Node) addr() *net.UDPAddr {
	return &net.UDPAddr{IP: n.IP, Port: n.DiscPort}
}

// The string representation of a Node is a URL.
// Please see ParseNode for a description of the format.
func (n *Node) String() string {
	addr := net.TCPAddr{IP: n.IP, Port: n.TCPPort}
	u := url.URL{
		Scheme: "enode",
		User:   url.User(fmt.Sprintf("%x", n.ID[:])),
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
// The hexadecimal node ID is encoded in the username portion of the
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
		return nil, errors.New("does not contain node ID")
	}
	if n.ID, err = HexID(u.User.String()); err != nil {
		return nil, fmt.Errorf("invalid node ID (%v)", err)
	}
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
	return rlp.Encode(w, rpcNode{IP: n.IP.String(), Port: uint16(n.TCPPort), ID: n.ID})
}
func (n *Node) DecodeRLP(s *rlp.Stream) (err error) {
	var ext rpcNode
	if err = s.Decode(&ext); err == nil {
		n.TCPPort = int(ext.Port)
		n.DiscPort = int(ext.Port)
		n.ID = ext.ID
		if n.IP = net.ParseIP(ext.IP); n.IP == nil {
			return errors.New("invalid IP string")
		}
	}
	return err
}

// NodeID is a unique identifier for each node.
// The node identifier is a marshaled elliptic curve public key.
type NodeID [nodeIDBits / 8]byte

// NodeID prints as a long hexadecimal number.
func (n NodeID) String() string {
	return fmt.Sprintf("%x", n[:])
}

// The Go syntax representation of a NodeID is a call to HexID.
func (n NodeID) GoString() string {
	return fmt.Sprintf("discover.HexID(\"%x\")", n[:])
}

// HexID converts a hex string to a NodeID.
// The string may be prefixed with 0x.
func HexID(in string) (NodeID, error) {
	if strings.HasPrefix(in, "0x") {
		in = in[2:]
	}
	var id NodeID
	b, err := hex.DecodeString(in)
	if err != nil {
		return id, err
	} else if len(b) != len(id) {
		return id, fmt.Errorf("wrong length, need %d hex bytes", len(id))
	}
	copy(id[:], b)
	return id, nil
}

// MustHexID converts a hex string to a NodeID.
// It panics if the string is not a valid NodeID.
func MustHexID(in string) NodeID {
	id, err := HexID(in)
	if err != nil {
		panic(err)
	}
	return id
}

// PubkeyID returns a marshaled representation of the given public key.
func PubkeyID(pub *ecdsa.PublicKey) NodeID {
	var id NodeID
	pbytes := elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	if len(pbytes)-1 != len(id) {
		panic(fmt.Errorf("need %d bit pubkey, got %d bits", (len(id)+1)*8, len(pbytes)))
	}
	copy(id[:], pbytes[1:])
	return id
}

// Pubkey returns the public key represented by the node ID.
// It returns an error if the ID is not a point on the curve.
func (id NodeID) Pubkey() (*ecdsa.PublicKey, error) {
	p := &ecdsa.PublicKey{Curve: crypto.S256(), X: new(big.Int), Y: new(big.Int)}
	half := len(id) / 2
	p.X.SetBytes(id[:half])
	p.Y.SetBytes(id[half:])
	if !p.Curve.IsOnCurve(p.X, p.Y) {
		return nil, errors.New("not a point on the S256 curve")
	}
	return p, nil
}

// recoverNodeID computes the public key used to sign the
// given hash from the signature.
func recoverNodeID(hash, sig []byte) (id NodeID, err error) {
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

// randomID returns a random NodeID such that logdist(a, b) == n
func randomID(a NodeID, n int) (b NodeID) {
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

// nodeDB stores all nodes we know about.
type nodeDB struct {
	ldb *leveldb.DB
}

var dbVersionKey = []byte("pv")

// Opens the backing LevelDB. If path is "", we use an in-memory database.
func newNodeDB(path string, version int64) (db *nodeDB, err error) {
	db = new(nodeDB)
	opts := new(opt.Options)
	if path == "" {
		db.ldb, err = leveldb.Open(storage.NewMemStorage(), opts)
	} else {
		db.ldb, err = openLDB(path, opts, version)
	}
	return db, err
}

func openLDB(path string, opts *opt.Options, version int64) (*leveldb.DB, error) {
	ldb, err := leveldb.OpenFile(path, opts)
	if _, iscorrupted := err.(leveldb.ErrCorrupted); iscorrupted {
		ldb, err = leveldb.RecoverFile(path, opts)
	}
	if err != nil {
		return nil, err
	}
	// The nodes contained in the database correspond to a certain
	// protocol version. Flush all nodes if the DB version doesn't match.
	// There is no need to do this for memory databases because they
	// won't ever be used with a different protocol version.
	shouldVal := make([]byte, binary.MaxVarintLen64)
	shouldVal = shouldVal[:binary.PutVarint(shouldVal, version)]
	val, err := ldb.Get(dbVersionKey, nil)
	if err == leveldb.ErrNotFound {
		err = ldb.Put(dbVersionKey, shouldVal, nil)
	} else if err == nil && !bytes.Equal(val, shouldVal) {
		// Delete and start over.
		ldb.Close()
		if err = os.RemoveAll(path); err != nil {
			return nil, err
		}
		return openLDB(path, opts, version)
	}
	if err != nil {
		ldb.Close()
		ldb = nil
	}
	return ldb, err
}

func (db *nodeDB) get(id NodeID) *Node {
	v, err := db.ldb.Get(id[:], nil)
	if err != nil {
		return nil
	}
	n := new(Node)
	if err := rlp.DecodeBytes(v, n); err != nil {
		return nil
	}
	return n
}

func (db *nodeDB) update(n *Node) error {
	v, err := rlp.EncodeToBytes(n)
	if err != nil {
		return err
	}
	return db.ldb.Put(n.ID[:], v, nil)
}

func (db *nodeDB) add(id NodeID, addr *net.UDPAddr, tcpPort uint16) *Node {
	n := &Node{ID: id, IP: addr.IP, DiscPort: addr.Port, TCPPort: int(tcpPort)}
	db.update(n)
	return n
}

func (db *nodeDB) close() {
	db.ldb.Close()
}
