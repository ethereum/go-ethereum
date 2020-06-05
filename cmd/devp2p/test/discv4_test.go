package test

import (
	"crypto/ecdsa"
	"flag"
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover/v4wire"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	expiration = 20 * time.Second
)

var (
	enodeID           string
	remoteAddr        string
	priv              *ecdsa.PrivateKey
	localhost         = net.ParseIP("127.0.0.1")
	localhostEndpoint = v4wire.Endpoint{IP: localhost}
	remoteEndpoint    = v4wire.Endpoint{IP: net.ParseIP(remoteAddr)}
	wrongEndpoint     = v4wire.Endpoint{IP: net.ParseIP("192.0.2.0")}
)

type pingWithJunk struct {
	Version    uint
	From, To   v4wire.Endpoint
	Expiration uint64
	JunkData1  uint
	JunkData2  []byte
	// Ignore additional fields (for forward compatibility).
	Rest []rlp.RawValue `rlp:"tail"`
}

func (req *pingWithJunk) Name() string { return "PING/v4" }
func (req *pingWithJunk) Kind() byte   { return v4wire.PingPacket }

func init() {
	flag.StringVar(&enodeID, "enode", "", "enode:... as per `admin.nodeInfo.enode`")
	flag.StringVar(&remoteAddr, "remoteAddr", "127.0.0.1:30303", "")

	var err error
	priv, err = crypto.GenerateKey()
	if err != nil {
		panic(err)
	}

	raddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		panic(err)
	}
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
}

func futureExpiration() uint64 {
	return uint64(time.Now().Add(expiration).Unix())
}

func sendPacket(packet []byte) (v4wire.Packet, error) {
	raddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_, err = conn.Write(packet)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 2048)
	if err = conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return nil, err
	}
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	p, _, _, err := v4wire.Decode(buf[:n])
	if err != nil {
		return nil, err
	}
	return p, nil
}

func sendRequest(t *testing.T, req v4wire.Packet) v4wire.Packet {
	packet, _, err := v4wire.Encode(priv, req)
	if err != nil {
		t.Fatal("Encoding", err)
	}

	var reply v4wire.Packet
	reply, err = sendPacket(packet)
	if err != nil {
		t.Fatal("Sending", err)
	}
	return reply
}

func PingKnownEnode(t *testing.T) {
	req := v4wire.Ping{
		Version:    4,
		From:       localhostEndpoint,
		To:         remoteEndpoint,
		Expiration: futureExpiration(),
	}
	reply := sendRequest(t, &req)
	if reply.Kind() != v4wire.PongPacket {
		t.Error("Reply is not a Pong", reply.Name())
	}
}

func PingWrongTo(t *testing.T) {
	req := v4wire.Ping{
		Version:    4,
		From:       localhostEndpoint,
		To:         wrongEndpoint,
		Expiration: futureExpiration(),
	}
	reply := sendRequest(t, &req)
	if reply.Kind() != v4wire.PongPacket {
		t.Error("Reply is not a Pong", reply.Name())
	}
}

func PingWrongFrom(t *testing.T) {
	req := v4wire.Ping{
		Version:    4,
		From:       wrongEndpoint,
		To:         remoteEndpoint,
		Expiration: futureExpiration(),
	}
	reply := sendRequest(t, &req)
	if reply.Kind() != v4wire.PongPacket {
		t.Error("Reply is not a Pong", reply.Name())
	}
}

func PingExtraData(t *testing.T) {
	req := pingWithJunk{
		Version:    4,
		From:       localhostEndpoint,
		To:         remoteEndpoint,
		Expiration: futureExpiration(),
		JunkData1:  42,
		JunkData2:  []byte{9, 8, 7, 6, 5, 4, 3, 2, 1},
	}
	reply := sendRequest(t, &req)
	if reply.Kind() != v4wire.PongPacket {
		t.Error("Reply is not a Pong", reply.Name())
	}
}

func PingExtraDataWrongFrom(t *testing.T) {
	req := pingWithJunk{
		Version:    4,
		From:       wrongEndpoint,
		To:         remoteEndpoint,
		Expiration: futureExpiration(),
		JunkData1:  42,
		JunkData2:  []byte{9, 8, 7, 6, 5, 4, 3, 2, 1},
	}
	reply := sendRequest(t, &req)
	if reply.Kind() != v4wire.PongPacket {
		t.Error("Reply is not a Pong", reply.Name())
	}
}

func PingPastExpiration(t *testing.T) {
	req := v4wire.Ping{
		Version:    4,
		From:       localhostEndpoint,
		To:         remoteEndpoint,
		Expiration: -futureExpiration(),
	}
	reply := sendRequest(t, &req)
	if reply.Kind() != v4wire.PongPacket {
		t.Error("Reply is not a Pong", reply.Name())
	}

}

func WrongPacketType(t *testing.T)                      {}
func FindNeighbours(t *testing.T)                       {}
func SourceKnownPingFromSignatureMismatch(t *testing.T) {}

func SpoofSanityCheck(t *testing.T)              {}
func SpoofAmplificationAttackCheck(t *testing.T) {}

func FindNeighboursOnRecentlyBondedTarget(t *testing.T) {}
func FindNeighboursPastExpiration(t *testing.T)         {}

func TestPing(t *testing.T) {
	t.Run("Ping-BasicTest(v4001)", PingKnownEnode)
	t.Run("Ping-WrongTo(v4002)", PingWrongTo)
	t.Run("Ping-WrongFrom(v4003)", PingWrongFrom)
	t.Run("Ping-ExtraData(v4004)", PingExtraData)
	t.Run("Ping-ExtraDataWrongFrom(v4005)", PingExtraDataWrongFrom)
	t.Run("Ping-PastExpiration(v4011)", PingPastExpiration)
	t.Run("Ping-WrongPacketType(v4006)", WrongPacketType)
	t.Run("Ping-BondedFromSignatureMismatch(v4009)", SourceKnownPingFromSignatureMismatch)
}

func TestSpoofing(t *testing.T) {
	t.Run("SpoofSanityCheck(v4013)", SpoofSanityCheck)
	t.Run("SpoofAmplification(v4014)", SpoofAmplificationAttackCheck)
}

func TestFindNode(t *testing.T) {
	t.Run("Findnode-UnbondedFindNeighbours(v4007)", FindNeighbours)
	t.Run("FindNode-UnsolicitedPollution(v4010)", FindNeighboursOnRecentlyBondedTarget)
	t.Run("FindNode-PastExpiration(v4012)", FindNeighboursPastExpiration)
}
