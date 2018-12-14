package pss

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/pot"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

var testResMap map[pot.Address]int

// this function substitutes the real send function, since
// we only want to test the peer selection functionality
func dummySendMsg(_ *Pss, sp *network.Peer, _ *PssMsg) bool {
	a := pot.NewAddressFromBytes(sp.Address())
	testResMap[a]++
	return true
}

// setDummySendMsg replaces sendMessage function for testing purposes
func setDummySendMsg() {
	sendMessage = dummySendMsg
}

// resetSendMsgProduction resets sendMessage function to production version
func resetSendMsgProduction() {
	sendMessage = sendMessageProd
}

// the purpose of this test is to see that pss.forward() function correctly
// selects the peers for message forwarding, depending on the message address
// and kademlia constellation.
func TestForwardBasic(t *testing.T) {
	setDummySendMsg()
	defer resetSendMsgProduction()

	baseAddrBytes := make([]byte, 32)
	for i := 0; i < len(baseAddrBytes); i++ {
		baseAddrBytes[i] = 0xFF
	}
	base := pot.NewAddressFromBytes(baseAddrBytes)
	var peerAddresses []pot.Address
	var a pot.Address
	const depth = 9
	for i := 0; i <= depth; i++ {
		// add one peer for each proximity order
		a = pot.RandomAddressAt(base, i)
		peerAddresses = append(peerAddresses, a)
	}

	// add one peer to the "depth" level, then skip one level, add one peer at one level below.
	// as a result, we will have an edge case of three peers in nearest neighbours' bin.
	peerAddresses = append(peerAddresses, pot.RandomAddressAt(base, depth))
	peerAddresses = append(peerAddresses, pot.RandomAddressAt(base, depth+2))

	kad := network.NewKademlia(base[:], network.NewKadParams())
	ps := createPss(t, kad)
	addPeers(kad, peerAddresses)

	const firstNearest = depth // first peer in the nearest neighbours' bin
	nearestNeighbours := []int{firstNearest, firstNearest + 1, firstNearest + 2}

	for i := 0; i < len(peerAddresses); i++ {
		// send msg directly to the known peers (recipient address == peer address)
		testForwardMsg(100+i, t, ps, peerAddresses[i][:], peerAddresses, []int{i})
	}

	for i := 0; i < firstNearest; i++ {
		// send random messages with proximity orders, corresponding to PO of each bin
		a = pot.RandomAddressAt(base, i)
		testForwardMsg(200+i, t, ps, a[:], peerAddresses, []int{i})
	}

	for i := firstNearest; i < len(peerAddresses); i++ {
		// recipient address falls into the nearest neighbours' bin
		a = pot.RandomAddressAt(base, i)
		testForwardMsg(300+i, t, ps, a[:], peerAddresses, nearestNeighbours)
	}

	// send msg with proximity order much deeper than the deepest nearest neighbour
	a = pot.RandomAddressAt(base, 77)
	testForwardMsg(400, t, ps, a[:], peerAddresses, nearestNeighbours)

	// test with partial addresses
	const part = 12

	for i := 0; i < firstNearest; i++ {
		// send messages with partial address falling into different proximity orders
		if i%8 != 0 {
			testForwardMsg(500+i, t, ps, peerAddresses[i][:i], peerAddresses, []int{i})
		}
		testForwardMsg(550+i, t, ps, peerAddresses[i][:part], peerAddresses, []int{i})
	}

	for i := firstNearest; i < len(peerAddresses); i++ {
		// partial address falls into the nearest neighbours' bin
		testForwardMsg(600+i, t, ps, peerAddresses[i][:part], peerAddresses, nearestNeighbours)
	}

	// partial address with proximity order higher than the last nearest neighbour
	a = pot.RandomAddressAt(base, part)
	testForwardMsg(700, t, ps, a[:part], peerAddresses, nearestNeighbours)

	// special cases where partial address matches a large group of peers
	all := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	testForwardMsg(800, t, ps, []byte{}, peerAddresses, all)

	// luminous radius of one byte (8 bits)
	testForwardMsg(900, t, ps, baseAddrBytes[:1], peerAddresses, all[8:])
}

// this function tests the forwarding of a single message. the recipient address is passed as param,
// along with addresses of all peers, and indices of those peers which are expected to receive the message.
func testForwardMsg(testID int, t *testing.T, ps *Pss, recipientAddr []byte, peers []pot.Address, expected []int) {
	testResMap = make(map[pot.Address]int)
	msg := newTestMsg(recipientAddr)
	ps.forward(msg)

	// check test results
	var fail bool
	s := fmt.Sprintf("test id: %d, msg address: %x..., radius: %d", testID, recipientAddr[:len(recipientAddr)%4], 8*len(recipientAddr))

	// false negatives (expected message didn't reach peer)
	for _, i := range expected {
		a := peers[i]
		received := testResMap[a]
		if received != 1 {
			s += fmt.Sprintf("\npeer number %d [%x...] received %d messages", i, a[:4], received)
			fail = true
		}
		testResMap[a] = 0
	}

	// false positives (unexpected message reached peer)
	for k, v := range testResMap {
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
