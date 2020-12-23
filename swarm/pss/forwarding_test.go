package pss

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/pot"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
)

type testCase struct {
	name      string
	recipient []byte
	peers     []pot.Address
	expected  []int
	exclusive bool
	nFails    int
	success   bool
	errors    string
}

var testCases []testCase

// the purpose of this test is to see that pss.forward() function correctly
// selects the peers for message forwarding, depending on the message address
// and kademlia constellation.
func TestForwardBasic(t *testing.T) {
	baseAddrBytes := make([]byte, 32)
	for i := 0; i < len(baseAddrBytes); i++ {
		baseAddrBytes[i] = 0xFF
	}
	var c testCase
	base := pot.NewAddressFromBytes(baseAddrBytes)
	var peerAddresses []pot.Address
	const depth = 10
	for i := 0; i <= depth; i++ {
		// add two peers for each proximity order
		a := pot.RandomAddressAt(base, i)
		peerAddresses = append(peerAddresses, a)
		a = pot.RandomAddressAt(base, i)
		peerAddresses = append(peerAddresses, a)
	}

	// skip one level, add one peer at one level deeper.
	// as a result, we will have an edge case of three peers in nearest neighbours' bin.
	peerAddresses = append(peerAddresses, pot.RandomAddressAt(base, depth+2))

	kad := network.NewKademlia(base[:], network.NewKadParams())
	ps := createPss(t, kad)
	defer ps.Stop()
	addPeers(kad, peerAddresses)

	const firstNearest = depth * 2 // shallowest peer in the nearest neighbours' bin
	nearestNeighbours := []int{firstNearest, firstNearest + 1, firstNearest + 2}
	var all []int // indices of all the peers
	for i := 0; i < len(peerAddresses); i++ {
		all = append(all, i)
	}

	for i := 0; i < len(peerAddresses); i++ {
		// send msg directly to the known peers (recipient address == peer address)
		c = testCase{
			name:      fmt.Sprintf("Send direct to known, id: [%d]", i),
			recipient: peerAddresses[i][:],
			peers:     peerAddresses,
			expected:  []int{i},
			exclusive: false,
		}
		testCases = append(testCases, c)
	}

	for i := 0; i < firstNearest; i++ {
		// send random messages with proximity orders, corresponding to PO of each bin,
		// with one peer being closer to the recipient address
		a := pot.RandomAddressAt(peerAddresses[i], 64)
		c = testCase{
			name:      fmt.Sprintf("Send random to each PO, id: [%d]", i),
			recipient: a[:],
			peers:     peerAddresses,
			expected:  []int{i},
			exclusive: false,
		}
		testCases = append(testCases, c)
	}

	for i := 0; i < firstNearest; i++ {
		// send random messages with proximity orders, corresponding to PO of each bin,
		// with random proximity relative to the recipient address
		po := i / 2
		a := pot.RandomAddressAt(base, po)
		c = testCase{
			name:      fmt.Sprintf("Send direct to known, id: [%d]", i),
			recipient: a[:],
			peers:     peerAddresses,
			expected:  []int{po * 2, po*2 + 1},
			exclusive: true,
		}
		testCases = append(testCases, c)
	}

	for i := firstNearest; i < len(peerAddresses); i++ {
		// recipient address falls into the nearest neighbours' bin
		a := pot.RandomAddressAt(base, i)
		c = testCase{
			name:      fmt.Sprintf("recipient address falls into the nearest neighbours' bin, id: [%d]", i),
			recipient: a[:],
			peers:     peerAddresses,
			expected:  nearestNeighbours,
			exclusive: false,
		}
		testCases = append(testCases, c)
	}

	// send msg with proximity order much deeper than the deepest nearest neighbour
	a2 := pot.RandomAddressAt(base, 77)
	c = testCase{
		name:      "proximity order much deeper than the deepest nearest neighbour",
		recipient: a2[:],
		peers:     peerAddresses,
		expected:  nearestNeighbours,
		exclusive: false,
	}
	testCases = append(testCases, c)

	// test with partial addresses
	const part = 12

	for i := 0; i < firstNearest; i++ {
		// send messages with partial address falling into different proximity orders
		po := i / 2
		if i%8 != 0 {
			c = testCase{
				name:      fmt.Sprintf("partial address falling into different proximity orders, id: [%d]", i),
				recipient: peerAddresses[i][:i],
				peers:     peerAddresses,
				expected:  []int{po * 2, po*2 + 1},
				exclusive: true,
			}
			testCases = append(testCases, c)
		}
		c = testCase{
			name:      fmt.Sprintf("extended partial address falling into different proximity orders, id: [%d]", i),
			recipient: peerAddresses[i][:part],
			peers:     peerAddresses,
			expected:  []int{po * 2, po*2 + 1},
			exclusive: true,
		}
		testCases = append(testCases, c)
	}

	for i := firstNearest; i < len(peerAddresses); i++ {
		// partial address falls into the nearest neighbours' bin
		c = testCase{
			name:      fmt.Sprintf("partial address falls into the nearest neighbours' bin, id: [%d]", i),
			recipient: peerAddresses[i][:part],
			peers:     peerAddresses,
			expected:  nearestNeighbours,
			exclusive: false,
		}
		testCases = append(testCases, c)
	}

	// partial address with proximity order deeper than any of the nearest neighbour
	a3 := pot.RandomAddressAt(base, part)
	c = testCase{
		name:      "partial address with proximity order deeper than any of the nearest neighbour",
		recipient: a3[:part],
		peers:     peerAddresses,
		expected:  nearestNeighbours,
		exclusive: false,
	}
	testCases = append(testCases, c)

	// special cases where partial address matches a large group of peers

	// zero bytes of address is given, msg should be delivered to all the peers
	c = testCase{
		name:      "zero bytes of address is given",
		recipient: []byte{},
		peers:     peerAddresses,
		expected:  all,
		exclusive: false,
	}
	testCases = append(testCases, c)

	// luminous radius of 8 bits, proximity order 8
	indexAtPo8 := 16
	c = testCase{
		name:      "luminous radius of 8 bits",
		recipient: []byte{0xFF},
		peers:     peerAddresses,
		expected:  all[indexAtPo8:],
		exclusive: false,
	}
	testCases = append(testCases, c)

	// luminous radius of 256 bits, proximity order 8
	a4 := pot.Address{}
	a4[0] = 0xFF
	c = testCase{
		name:      "luminous radius of 256 bits",
		recipient: a4[:],
		peers:     peerAddresses,
		expected:  []int{indexAtPo8, indexAtPo8 + 1},
		exclusive: true,
	}
	testCases = append(testCases, c)

	// check correct behaviour in case send fails
	for i := 2; i < firstNearest-3; i += 2 {
		po := i / 2
		// send random messages with proximity orders, corresponding to PO of each bin,
		// with different numbers of failed attempts.
		// msg should be received by only one of the deeper peers.
		a := pot.RandomAddressAt(base, po)
		c = testCase{
			name:      fmt.Sprintf("Send direct to known, id: [%d]", i),
			recipient: a[:],
			peers:     peerAddresses,
			expected:  all[i+1:],
			exclusive: true,
			nFails:    rand.Int()%3 + 2,
		}
		testCases = append(testCases, c)
	}

	for _, c := range testCases {
		testForwardMsg(t, ps, &c)
	}
}

// this function tests the forwarding of a single message. the recipient address is passed as param,
// along with addresses of all peers, and indices of those peers which are expected to receive the message.
func testForwardMsg(t *testing.T, ps *Pss, c *testCase) {
	recipientAddr := c.recipient
	peers := c.peers
	expected := c.expected
	exclusive := c.exclusive
	nFails := c.nFails
	tries := 0 // number of previous failed tries

	resultMap := make(map[pot.Address]int)

	defer func() { sendFunc = sendMsg }()
	sendFunc = func(_ *Pss, sp *network.Peer, _ *PssMsg) bool {
		if tries < nFails {
			tries++
			return false
		}
		a := pot.NewAddressFromBytes(sp.Address())
		resultMap[a]++
		return true
	}

	msg := newTestMsg(recipientAddr)
	ps.forward(msg)

	// check test results
	var fail bool
	precision := len(recipientAddr)
	if precision > 4 {
		precision = 4
	}
	s := fmt.Sprintf("test [%s]\nmsg address: %x..., radius: %d", c.name, recipientAddr[:precision], 8*len(recipientAddr))

	// false negatives (expected message didn't reach peer)
	if exclusive {
		var cnt int
		for _, i := range expected {
			a := peers[i]
			cnt += resultMap[a]
			resultMap[a] = 0
		}
		if cnt != 1 {
			s += fmt.Sprintf("\n%d messages received by %d peers with indices: [%v]", cnt, len(expected), expected)
			fail = true
		}
	} else {
		for _, i := range expected {
			a := peers[i]
			received := resultMap[a]
			if received != 1 {
				s += fmt.Sprintf("\npeer number %d [%x...] received %d messages", i, a[:4], received)
				fail = true
			}
			resultMap[a] = 0
		}
	}

	// false positives (unexpected message reached peer)
	for k, v := range resultMap {
		if v != 0 {
			// find the index of the false positive peer
			var j int
			for j = 0; j < len(peers); j++ {
				if peers[j] == k {
					break
				}
			}
			s += fmt.Sprintf("\npeer number %d [%x...] received %d messages", j, k[:4], v)
			fail = true
		}
	}

	if fail {
		t.Fatal(s)
	}
}

func addPeers(kad *network.Kademlia, addresses []pot.Address) {
	for _, a := range addresses {
		p := newTestDiscoveryPeer(a, kad)
		kad.On(p)
	}
}

func createPss(t *testing.T, kad *network.Kademlia) *Pss {
	privKey, err := crypto.GenerateKey()
	pssp := NewPssParams().WithPrivateKey(privKey)
	ps, err := NewPss(kad, pssp)
	if err != nil {
		t.Fatal(err.Error())
	}
	return ps
}

func newTestDiscoveryPeer(addr pot.Address, kad *network.Kademlia) *network.Peer {
	rw := &p2p.MsgPipeRW{}
	p := p2p.NewPeer(enode.ID{}, "test", []p2p.Cap{})
	pp := protocols.NewPeer(p, rw, &protocols.Spec{})
	bp := &network.BzzPeer{
		Peer: pp,
		BzzAddr: &network.BzzAddr{
			OAddr: addr.Bytes(),
			UAddr: []byte(fmt.Sprintf("%x", addr[:])),
		},
	}
	return network.NewPeer(bp, kad)
}

func newTestMsg(addr []byte) *PssMsg {
	msg := newPssMsg(&msgParams{})
	msg.To = addr[:]
	msg.Expire = uint32(time.Now().Add(time.Second * 60).Unix())
	msg.Payload = &whisper.Envelope{
		Topic: [4]byte{},
		Data:  []byte("i have nothing to hide"),
	}
	return msg
}
