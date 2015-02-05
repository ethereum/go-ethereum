package discover

import (
	logpkg "log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/logger"
)

func init() {
	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, logpkg.LstdFlags, logger.DebugLevel))
}

func TestUDP_ping(t *testing.T) {
	t.Parallel()

	n1, _ := ListenUDP(newkey(), "127.0.0.1:0")
	n2, _ := ListenUDP(newkey(), "127.0.0.1:0")
	defer n1.Close()
	defer n2.Close()

	if err := n1.net.ping(n2.self); err != nil {
		t.Fatalf("ping error: %v", err)
	}
	if find(n2, n1.self.ID) == nil {
		t.Errorf("node 2 does not contain id of node 1")
	}
	if e := find(n1, n2.self.ID); e != nil {
		t.Errorf("node 1 does contains id of node 2: %v", e)
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

func TestUDP_findnode(t *testing.T) {
	t.Parallel()

	n1, _ := ListenUDP(newkey(), "127.0.0.1:0")
	n2, _ := ListenUDP(newkey(), "127.0.0.1:0")
	defer n1.Close()
	defer n2.Close()

	entry := &Node{ID: NodeID{1}, Addr: &net.UDPAddr{IP: net.IP{1, 2, 3, 4}, Port: 15}}
	n2.add([]*Node{entry})

	target := randomID(n1.self.ID, 100)
	result, _ := n1.net.findnode(n2.self, target)
	if len(result) != 1 {
		t.Fatalf("wrong number of results: got %d, want 1", len(result))
	}
	if result[0].ID != entry.ID {
		t.Errorf("wrong result: got %v, want %v", result[0], entry)
	}
}

func TestUDP_replytimeout(t *testing.T) {
	t.Parallel()

	// reserve a port so we don't talk to an existing service by accident
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	fd, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer fd.Close()

	n1, _ := ListenUDP(newkey(), "127.0.0.1:0")
	defer n1.Close()
	n2 := n1.bumpOrAdd(randomID(n1.self.ID, 10), fd.LocalAddr().(*net.UDPAddr))

	if err := n1.net.ping(n2); err != errTimeout {
		t.Error("expected timeout error, got", err)
	}

	if result, err := n1.net.findnode(n2, n1.self.ID); err != errTimeout {
		t.Error("expected timeout error, got", err)
	} else if len(result) > 0 {
		t.Error("expected empty result, got", result)
	}
}

func TestUDP_findnodeMultiReply(t *testing.T) {
	t.Parallel()

	n1, _ := ListenUDP(newkey(), "127.0.0.1:0")
	n2, _ := ListenUDP(newkey(), "127.0.0.1:0")
	udp2 := n2.net.(*udp)
	defer n1.Close()
	defer n2.Close()

	nodes := make([]*Node, bucketSize)
	for i := range nodes {
		nodes[i] = &Node{
			Addr: &net.UDPAddr{IP: net.IP{1, 2, 3, 4}, Port: i + 1},
			ID:   randomID(n2.self.ID, i+1),
		}
	}

	// ask N2 for neighbors. it will send an empty reply back.
	// the request will wait for up to bucketSize replies.
	resultC := make(chan []*Node)
	go func() {
		ns, err := n1.net.findnode(n2.self, n1.self.ID)
		if err != nil {
			t.Error("findnode error:", err)
		}
		resultC <- ns
	}()

	// send a few more neighbors packets to N1.
	// it should collect those.
	for end := 0; end < len(nodes); {
		off := end
		if end = end + 5; end > len(nodes) {
			end = len(nodes)
		}
		udp2.send(n1.self, neighborsPacket, neighbors{
			Nodes:      nodes[off:end],
			Expiration: uint64(time.Now().Add(10 * time.Second).Unix()),
		})
	}

	// check that they are all returned. we cannot just check for
	// equality because they might not be returned in the order they
	// were sent.
	result := <-resultC
	if hasDuplicates(result) {
		t.Error("result slice contains duplicates")
	}
	if len(result) != len(nodes) {
		t.Errorf("wrong number of nodes returned: got %d, want %d", len(result), len(nodes))
	}
	matched := make(map[NodeID]bool)
	for _, n := range result {
		for _, expn := range nodes {
			if n.ID == expn.ID { // && bytes.Equal(n.Addr.IP, expn.Addr.IP) && n.Addr.Port == expn.Addr.Port {
				matched[n.ID] = true
			}
		}
	}
	if len(matched) != len(nodes) {
		t.Errorf("wrong number of matching nodes: got %d, want %d", len(matched), len(nodes))
	}
}
