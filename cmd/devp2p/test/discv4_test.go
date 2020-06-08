package test

import (
	"crypto/ecdsa"
	"flag"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover/v4wire"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

const (
	expiration  = 20 * time.Second
	wrongPacket = 66
)

var (
	enodeID    = flag.String("enode", "", "enode:... as per `admin.nodeInfo.enode`")
	remoteAddr = flag.String("remoteAddr", "127.0.0.1:30303", "")
	waitTime   = flag.Int("waitTime", 500, "ms to wait for response")

	priv              *ecdsa.PrivateKey
	localhost         = net.ParseIP("127.0.0.1")
	localhostEndpoint = v4wire.Endpoint{IP: localhost}
	remoteEndpoint    = v4wire.Endpoint{IP: net.ParseIP(*remoteAddr)}
	wrongEndpoint     = v4wire.Endpoint{IP: net.ParseIP("192.0.2.0")}
)

type pingWithJunk struct {
	Version    uint
	From, To   v4wire.Endpoint
	Expiration uint64
	JunkData1  uint
	JunkData2  []byte
}

func (req *pingWithJunk) Name() string { return "PING/v4" }
func (req *pingWithJunk) Kind() byte   { return v4wire.PingPacket }

type pingWrongType struct {
	Version    uint
	From, To   v4wire.Endpoint
	Expiration uint64
}

func (req *pingWrongType) Name() string { return "WRONG/v4" }
func (req *pingWrongType) Kind() byte   { return wrongPacket }

func TestMain(m *testing.M) {
	flag.Parse()

	fmt.Println(*enodeID, *remoteAddr, *waitTime)

	var err error
	priv, err = crypto.GenerateKey()
	if err != nil {
		panic(err)
	}

	raddr, err := net.ResolveUDPAddr("udp", *remoteAddr)
	if err != nil {
		panic(err)
	}
	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	os.Exit(m.Run())
}

func futureExpiration() uint64 {
	return uint64(time.Now().Add(expiration).Unix())
}

func sendPacket(packet []byte, read bool) (v4wire.Packet, error) {
	raddr, err := net.ResolveUDPAddr("udp", *remoteAddr)
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

	if !read {
		return nil, nil
	}
	buf := make([]byte, 2048)
	if err = conn.SetReadDeadline(time.Now().Add(time.Duration(*waitTime) * time.Millisecond)); err != nil {
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

func sendRequest(req v4wire.Packet, read bool) (v4wire.Packet, error) {
	packet, _, err := v4wire.Encode(priv, req)
	if err != nil {
		return nil, err
	}

	var reply v4wire.Packet
	reply, err = sendPacket(packet, read)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

func PingKnownEnode(t *testing.T) {
	req := v4wire.Ping{
		Version:    4,
		From:       localhostEndpoint,
		To:         remoteEndpoint,
		Expiration: futureExpiration(),
	}
	reply, err := sendRequest(&req, true)
	if err != nil {
		t.Fatal(err)
	}
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
	reply, err := sendRequest(&req, true)
	if err != nil {
		t.Fatal(err)
	}
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
	reply, err := sendRequest(&req, true)
	if err != nil {
		t.Fatal(err)
	}
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
	reply, err := sendRequest(&req, true)
	if err != nil {
		t.Fatal(err)
	}
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
	reply, err := sendRequest(&req, true)
	if err != nil {
		t.Fatal(err)
	}
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
	reply, _ := sendRequest(&req, true)
	if reply != nil {
		t.Fatal("Expected no reply, got", reply)
	}
}

func WrongPacketType(t *testing.T) {
	req := pingWrongType{
		Version:    4,
		From:       localhostEndpoint,
		To:         remoteEndpoint,
		Expiration: futureExpiration(),
	}
	reply, _ := sendRequest(&req, true)
	if reply != nil {
		t.Fatal("Expected no reply, got", reply)
	}
}

func SourceKnownPingFromSignatureMismatch(t *testing.T) {
	var reply v4wire.Packet
	var err error
	req := v4wire.Ping{
		Version:    4,
		From:       localhostEndpoint,
		To:         remoteEndpoint,
		Expiration: futureExpiration(),
	}
	reply, err = sendRequest(&req, true)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Kind() != v4wire.PongPacket {
		t.Error("Reply is not a Pong", reply.Name())
	}

	//hang around for a bit (we don't know if the target was already bonded or not)
	time.Sleep(2 * time.Second)

	req2 := v4wire.Ping{
		Version:    4,
		From:       wrongEndpoint,
		To:         remoteEndpoint,
		Expiration: futureExpiration(),
	}
	reply, err = sendRequest(&req2, true)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Kind() != v4wire.PongPacket {
		t.Error("Reply is not a Pong after bonding", reply.Name())
	}
}

func FindNeighbours(t *testing.T) {}

func SpoofSanityCheck(t *testing.T)              {}
func SpoofAmplificationAttackCheck(t *testing.T) {}

func FindNeighboursOnRecentlyBondedTarget(t *testing.T) {
	//try to bond with the target
	pingReq := v4wire.Ping{
		Version:    4,
		From:       localhostEndpoint,
		To:         remoteEndpoint,
		Expiration: futureExpiration(),
	}
	_, err := sendRequest(&pingReq, true)
	if err != nil {
		t.Fatal("First ping failed", err)
	}

	//hang around for a bit (we don't know if the target was already bonded or not)
	time.Sleep(2 * time.Second)

	//send an unsolicited neighbours packet
	var fakeKey *ecdsa.PrivateKey
	fakeKey, err = crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	fakePub := fakeKey.PublicKey
	encFakeKey := v4wire.EncodePubkey(&fakePub)
	fakeNeighbor := v4wire.Node{ID: encFakeKey, IP: net.IP{1, 2, 3, 4}, UDP: 123, TCP: 123}
	neighborsReq := v4wire.Neighbors{
		Nodes:      []v4wire.Node{fakeNeighbor},
		Expiration: futureExpiration(),
	}
	reply, err := sendRequest(&neighborsReq, true)
	if err != nil {
		t.Fatal("NeighborsReq", err)
	}

	//now call find neighbours
	targetNode := enode.MustParseV4(*enodeID)
	targetEncKey := v4wire.EncodePubkey(targetNode.Pubkey())
	findReq := v4wire.Findnode{
		Target:     targetEncKey,
		Expiration: futureExpiration(),
	}
	reply, err = sendRequest(&findReq, true)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Kind() != v4wire.PongPacket {
		t.Fatal("Expected pong, got", reply.Name())
	}
}

func FindNeighboursPastExpiration(t *testing.T) {}

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
