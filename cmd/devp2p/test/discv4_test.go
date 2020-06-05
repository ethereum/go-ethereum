package test

import (
	"crypto/ecdsa"
	"flag"
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover/v4wire"
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
	n, err := conn.Write(packet)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 2048)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err = conn.Read(buf)
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
		t.Error("Encoding", err)
	}

	var reply v4wire.Packet
	reply, err = sendPacket(packet)
	if err != nil {
		t.Error("Sending", err)
	}
	return reply
}

func SimplePing(t *testing.T) {
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

func SourceUnknownPingKnownEnode(t *testing.T) {
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

func SourceUnknownPingWrongTo(t *testing.T)             {}
func SourceUnknownPingWrongFrom(t *testing.T)           {}
func SourceUnknownPingExtraData(t *testing.T)           {}
func SourceUnknownPingExtraDataWrongFrom(t *testing.T)  {}
func SourceUnknownWrongPacketType(t *testing.T)         {}
func SourceUnknownFindNeighbours(t *testing.T)          {}
func SourceKnownPingFromSignatureMismatch(t *testing.T) {}
func PingPastExpiration(t *testing.T)                   {}

func SpoofSanityCheck(t *testing.T)              {}
func SpoofAmplificationAttackCheck(t *testing.T) {}

func FindNeighboursOnRecentlyBondedTarget(t *testing.T) {}
func FindNeighboursPastExpiration(t *testing.T)         {}

func TestPing(t *testing.T) {
	t.Run("Ping-Simple", SimplePing)
	t.Run("Ping-BasicTest(v4001)", SourceUnknownPingKnownEnode)
	t.Run("Ping-SourceUnknownrongTo(v4002)", SourceUnknownPingWrongTo)
	t.Run("Ping-SourceUnknownWrongFrom(v4003)", SourceUnknownPingWrongFrom)
	t.Run("Ping-SourceUnknownExtraData(v4004)", SourceUnknownPingExtraData)
	t.Run("Ping-SourceUnknownExtraDataWrongFrom(v4005)", SourceUnknownPingExtraDataWrongFrom)
	t.Run("Ping-SourceUnknownWrongPacketType(v4006)", SourceUnknownWrongPacketType)
	t.Run("Ping-BondedFromSignatureMismatch(v4009)", SourceKnownPingFromSignatureMismatch)
	t.Run("Ping-PastExpiration(v4011)", PingPastExpiration)
}

func TestSpoofing(t *testing.T) {
	t.Run("SpoofSanityCheck(v4013)", SpoofSanityCheck)
	t.Run("SpoofAmplification(v4014)", SpoofAmplificationAttackCheck)
}

func TestFindNode(t *testing.T) {
	t.Run("Findnode-UnbondedFindNeighbours(v4007)", SourceUnknownFindNeighbours)
	t.Run("FindNode-UnsolicitedPollution(v4010)", FindNeighboursOnRecentlyBondedTarget)
	t.Run("FindNode-PastExpiration(v4012)", FindNeighboursPastExpiration)
}
