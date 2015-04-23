package discover

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	logpkg "log"
	"net"
	"os"
	"path"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/logger"
)

func init() {
	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, logpkg.LstdFlags, logger.ErrorLevel))
}

type udpTest struct {
	t                   *testing.T
	pipe                *dgramPipe
	table               *Table
	udp                 *udp
	sent                [][]byte
	localkey, remotekey *ecdsa.PrivateKey
	remoteaddr          *net.UDPAddr
}

func newUDPTest(t *testing.T) *udpTest {
	test := &udpTest{
		t:          t,
		pipe:       newpipe(),
		localkey:   newkey(),
		remotekey:  newkey(),
		remoteaddr: &net.UDPAddr{IP: net.IP{1, 2, 3, 4}, Port: 30303},
	}
	test.table, test.udp = newUDP(test.localkey, test.pipe, nil, "")
	return test
}

// handles a packet as if it had been sent to the transport.
func (test *udpTest) packetIn(wantError error, ptype byte, data packet) error {
	enc, err := encodePacket(test.remotekey, ptype, data)
	if err != nil {
		return test.errorf("packet (%d) encode error: %v", err)
	}
	test.sent = append(test.sent, enc)
	err = data.handle(test.udp, test.remoteaddr, PubkeyID(&test.remotekey.PublicKey), enc[:macSize])
	if err != wantError {
		return test.errorf("error mismatch: got %q, want %q", err, wantError)
	}
	return nil
}

// waits for a packet to be sent by the transport.
// validate should have type func(*udpTest, X) error, where X is a packet type.
func (test *udpTest) waitPacketOut(validate interface{}) error {
	dgram := test.pipe.waitPacketOut()
	p, _, _, err := decodePacket(dgram)
	if err != nil {
		return test.errorf("sent packet decode error: %v", err)
	}
	fn := reflect.ValueOf(validate)
	exptype := fn.Type().In(0)
	if reflect.TypeOf(p) != exptype {
		return test.errorf("sent packet type mismatch, got: %v, want: %v", reflect.TypeOf(p), exptype)
	}
	fn.Call([]reflect.Value{reflect.ValueOf(p)})
	return nil
}

func (test *udpTest) errorf(format string, args ...interface{}) error {
	_, file, line, ok := runtime.Caller(2) // errorf + waitPacketOut
	if ok {
		file = path.Base(file)
	} else {
		file = "???"
		line = 1
	}
	err := fmt.Errorf(format, args...)
	fmt.Printf("\t%s:%d: %v\n", file, line, err)
	test.t.Fail()
	return err
}

// shared test variables
var (
	futureExp  = uint64(time.Now().Add(10 * time.Hour).Unix())
	testTarget = MustHexID("01010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101")
)

func TestUDP_packetErrors(t *testing.T) {
	test := newUDPTest(t)
	defer test.table.Close()

	test.packetIn(errExpired, pingPacket, &ping{IP: "foo", Port: 99, Version: Version})
	test.packetIn(errBadVersion, pingPacket, &ping{IP: "foo", Port: 99, Version: 99, Expiration: futureExp})
	test.packetIn(errUnsolicitedReply, pongPacket, &pong{ReplyTok: []byte{}, Expiration: futureExp})
	test.packetIn(errUnknownNode, findnodePacket, &findnode{Expiration: futureExp})
	test.packetIn(errUnsolicitedReply, neighborsPacket, &neighbors{Expiration: futureExp})
}

func TestUDP_pingTimeout(t *testing.T) {
	t.Parallel()
	test := newUDPTest(t)
	defer test.table.Close()

	toaddr := &net.UDPAddr{IP: net.ParseIP("1.2.3.4"), Port: 2222}
	toid := NodeID{1, 2, 3, 4}
	if err := test.udp.ping(toid, toaddr); err != errTimeout {
		t.Error("expected timeout error, got", err)
	}
}

func TestUDP_findnodeTimeout(t *testing.T) {
	t.Parallel()
	test := newUDPTest(t)
	defer test.table.Close()

	toaddr := &net.UDPAddr{IP: net.ParseIP("1.2.3.4"), Port: 2222}
	toid := NodeID{1, 2, 3, 4}
	target := NodeID{4, 5, 6, 7}
	result, err := test.udp.findnode(toid, toaddr, target)
	if err != errTimeout {
		t.Error("expected timeout error, got", err)
	}
	if len(result) > 0 {
		t.Error("expected empty result, got", result)
	}
}

func TestUDP_findnode(t *testing.T) {
	test := newUDPTest(t)
	defer test.table.Close()

	// put a few nodes into the table. their exact
	// distribution shouldn't matter much, altough we need to
	// take care not to overflow any bucket.
	target := testTarget
	nodes := &nodesByDistance{target: target}
	for i := 0; i < bucketSize; i++ {
		nodes.push(&Node{
			IP:       net.IP{1, 2, 3, byte(i)},
			DiscPort: i + 2,
			TCPPort:  i + 2,
			ID:       randomID(test.table.self.ID, i+2),
		}, bucketSize)
	}
	test.table.add(nodes.entries)

	// ensure there's a bond with the test node,
	// findnode won't be accepted otherwise.
	test.table.db.add(PubkeyID(&test.remotekey.PublicKey), test.remoteaddr, 99)

	// check that closest neighbors are returned.
	test.packetIn(nil, findnodePacket, &findnode{Target: testTarget, Expiration: futureExp})
	test.waitPacketOut(func(p *neighbors) {
		expected := test.table.closest(testTarget, bucketSize)
		if len(p.Nodes) != bucketSize {
			t.Errorf("wrong number of results: got %d, want %d", len(p.Nodes), bucketSize)
		}
		for i := range p.Nodes {
			if p.Nodes[i].ID != expected.entries[i].ID {
				t.Errorf("result mismatch at %d:\n  got:  %v\n  want: %v", i, p.Nodes[i], expected.entries[i])
			}
		}
	})
}

func TestUDP_findnodeMultiReply(t *testing.T) {
	test := newUDPTest(t)
	defer test.table.Close()

	// queue a pending findnode request
	resultc, errc := make(chan []*Node), make(chan error)
	go func() {
		rid := PubkeyID(&test.remotekey.PublicKey)
		ns, err := test.udp.findnode(rid, test.remoteaddr, testTarget)
		if err != nil && len(ns) == 0 {
			errc <- err
		} else {
			resultc <- ns
		}
	}()

	// wait for the findnode to be sent.
	// after it is sent, the transport is waiting for a reply
	test.waitPacketOut(func(p *findnode) {
		if p.Target != testTarget {
			t.Errorf("wrong target: got %v, want %v", p.Target, testTarget)
		}
	})

	// send the reply as two packets.
	list := []*Node{
		MustParseNode("enode://ba85011c70bcc5c04d8607d3a0ed29aa6179c092cbdda10d5d32684fb33ed01bd94f588ca8f91ac48318087dcb02eaf36773a7a453f0eedd6742af668097b29c@10.0.1.16:30303"),
		MustParseNode("enode://81fa361d25f157cd421c60dcc28d8dac5ef6a89476633339c5df30287474520caca09627da18543d9079b5b288698b542d56167aa5c09111e55acdbbdf2ef799@10.0.1.16:30303"),
		MustParseNode("enode://9bffefd833d53fac8e652415f4973bee289e8b1a5c6c4cbe70abf817ce8a64cee11b823b66a987f51aaa9fba0d6a91b3e6bf0d5a5d1042de8e9eeea057b217f8@10.0.1.36:30301"),
		MustParseNode("enode://1b5b4aa662d7cb44a7221bfba67302590b643028197a7d5214790f3bac7aaa4a3241be9e83c09cf1f6c69d007c634faae3dc1b1221793e8446c0b3a09de65960@10.0.1.16:30303"),
	}
	test.packetIn(nil, neighborsPacket, &neighbors{Expiration: futureExp, Nodes: list[:2]})
	test.packetIn(nil, neighborsPacket, &neighbors{Expiration: futureExp, Nodes: list[2:]})

	// check that the sent neighbors are all returned by findnode
	select {
	case result := <-resultc:
		if !reflect.DeepEqual(result, list) {
			t.Errorf("neighbors mismatch:\n  got:  %v\n  want: %v", result, list)
		}
	case err := <-errc:
		t.Errorf("findnode error: %v", err)
	case <-time.After(5 * time.Second):
		t.Error("findnode did not return within 5 seconds")
	}
}

func TestUDP_successfulPing(t *testing.T) {
	test := newUDPTest(t)
	defer test.table.Close()

	done := make(chan struct{})
	go func() {
		test.packetIn(nil, pingPacket, &ping{IP: "foo", Port: 99, Version: Version, Expiration: futureExp})
		close(done)
	}()

	// the ping is replied to.
	test.waitPacketOut(func(p *pong) {
		pinghash := test.sent[0][:macSize]
		if !bytes.Equal(p.ReplyTok, pinghash) {
			t.Errorf("got ReplyTok %x, want %x", p.ReplyTok, pinghash)
		}
	})

	// remote is unknown, the table pings back.
	test.waitPacketOut(func(p *ping) error { return nil })
	test.packetIn(nil, pongPacket, &pong{Expiration: futureExp})

	// ping should return shortly after getting the pong packet.
	<-done

	// check that the node was added.
	rid := PubkeyID(&test.remotekey.PublicKey)
	rnode := find(test.table, rid)
	if rnode == nil {
		t.Fatalf("node %v not found in table", rid)
	}
	if !bytes.Equal(rnode.IP, test.remoteaddr.IP) {
		t.Errorf("node has wrong IP: got %v, want: %v", rnode.IP, test.remoteaddr.IP)
	}
	if rnode.DiscPort != test.remoteaddr.Port {
		t.Errorf("node has wrong Port: got %v, want: %v", rnode.DiscPort, test.remoteaddr.Port)
	}
	if rnode.TCPPort != 99 {
		t.Errorf("node has wrong Port: got %v, want: %v", rnode.TCPPort, 99)
	}
}

func find(tab *Table, id NodeID) *Node {
	for _, b := range tab.buckets {
		for _, e := range b.entries {
			if e.ID == id {
				return e
			}
		}
	}
	return nil
}

// dgramPipe is a fake UDP socket. It queues all sent datagrams.
type dgramPipe struct {
	mu      *sync.Mutex
	cond    *sync.Cond
	closing chan struct{}
	closed  bool
	queue   [][]byte
}

func newpipe() *dgramPipe {
	mu := new(sync.Mutex)
	return &dgramPipe{
		closing: make(chan struct{}),
		cond:    &sync.Cond{L: mu},
		mu:      mu,
	}
}

// WriteToUDP queues a datagram.
func (c *dgramPipe) WriteToUDP(b []byte, to *net.UDPAddr) (n int, err error) {
	msg := make([]byte, len(b))
	copy(msg, b)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return 0, errors.New("closed")
	}
	c.queue = append(c.queue, msg)
	c.cond.Signal()
	return len(b), nil
}

// ReadFromUDP just hangs until the pipe is closed.
func (c *dgramPipe) ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error) {
	<-c.closing
	return 0, nil, io.EOF
}

func (c *dgramPipe) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		close(c.closing)
		c.closed = true
	}
	return nil
}

func (c *dgramPipe) LocalAddr() net.Addr {
	return &net.UDPAddr{}
}

func (c *dgramPipe) waitPacketOut() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	for len(c.queue) == 0 {
		c.cond.Wait()
	}
	p := c.queue[0]
	copy(c.queue, c.queue[1:])
	c.queue = c.queue[:len(c.queue)-1]
	return p
}
