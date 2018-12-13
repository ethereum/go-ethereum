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

func TestForwardBasic(t *testing.T) {
	base := newBaseAddress() // 0xFFFFFF.......
	var peerAddresses []pot.Address
	var dst pot.Address
	const depth = 9
	for i := 0; i <= depth; i++ {
		a := pot.RandomAddressAt(base, i)
		peerAddresses = append(peerAddresses, a)
		a = pot.RandomAddressAt(base, i)
		peerAddresses = append(peerAddresses, a)
	}

	// skip one level, add one peer at one level below
	a := pot.RandomAddressAt(base, depth+2)
	peerAddresses = append(peerAddresses, a)

	kad := network.NewKademlia(base[:], network.NewKadParams())
	ps := createPss(t, kad)
	addPeers(kad, peerAddresses)

	const firstNearest = depth * 2 // first peer in the nearest neighbours' bin
	nearestNeighbours := []int{firstNearest, firstNearest + 1, firstNearest + 2}
	//fmt.Println(kad.String()) // print kademlia map for debugging, before any test starts

	for i := 0; i < len(peerAddresses); i++ {
		// send msg directly to the known peers (recipient address == peer address)
		testForwardMsg(100+i, t, ps, peerAddresses[i][:], peerAddresses, []int{i})
	}

	for i := 0; i < firstNearest; i++ {
		// send random messages with different proximity orders
		po := i / 2
		dst := pot.RandomAddressAt(base, po)
		testForwardMsg(200+i, t, ps, dst[:], peerAddresses, []int{po * 2, po*2 + 1})
	}

	for i := firstNearest; i < len(peerAddresses); i++ {
		// recipient address falls into the nearest neighbours' bin
		dst := pot.RandomAddressAt(base, i)
		testForwardMsg(300+i, t, ps, dst[:], peerAddresses, nearestNeighbours)
	}

	// send msg with proximity order higher than the last nearest neighbour
	dst = pot.RandomAddressAt(base, 29)
	testForwardMsg(400, t, ps, dst[:], peerAddresses, nearestNeighbours)

	// test with partial addresses
	const part = 12

	for i := 0; i < firstNearest; i++ {
		// send messages with partial address falling into different proximity orders
		po := i / 2
		if po%8 != 0 {
			testForwardMsg(500+i, t, ps, peerAddresses[i][:po], peerAddresses, []int{po * 2, po*2 + 1})
		}
		testForwardMsg(550+i, t, ps, peerAddresses[i][:part], peerAddresses, []int{po * 2, po*2 + 1})
	}

	for i := firstNearest; i < len(peerAddresses); i++ {
		// partial address falls into the nearest neighbours' bin
		testForwardMsg(600+i, t, ps, peerAddresses[i][:part], peerAddresses, nearestNeighbours)
	}

	// partial address with proximity order higher than the last nearest neighbour
	dst = pot.RandomAddressAt(base, part)
	testForwardMsg(700, t, ps, dst[:part], peerAddresses, nearestNeighbours)

	// special cases where partial address matches a large group of peers
	all := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	testForwardMsg(800, t, ps, []byte{}, peerAddresses, all)
	testForwardMsg(900, t, ps, peerAddresses[19][:1], peerAddresses, all[16:])
}

func testForwardMsg(num int, t *testing.T, ps *Pss, addr []byte, addresses []pot.Address, expected []int) {
	testResMap := make(map[pot.Address]int)
	msg := newTestMsg(addr)
	ps.forward(msg, func(p *Pss, sp *network.Peer, msg *PssMsg) bool {
		a := pot.NewAddressFromBytes(sp.Address())
		testResMap[a]++
		return true
	})

	// check test results
	var fail bool
	s := fmt.Sprintf("test id: %d, msg address: %x..., radius: %d", num, addr[:len(addr)%4], 8*len(addr))

	// false negatives
	for _, i := range expected {
		a := addresses[i]
		received := testResMap[a]
		if received != 1 {
			s += fmt.Sprintf("\npeer number %d [%x...] received %d messages", i, a[:4], received)
			fail = true
		}
		testResMap[a] = 0
	}

	// false positives
	for k, v := range testResMap {
		if v != 0 {
			// find the index of the false positive peer
			var j int
			for j = 0; j < len(addresses); j++ {
				if addresses[j] == k {
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

func newBaseAddress() pot.Address {
	//base := network.RandomAddr().OAddr
	base := make([]byte, 32)
	for i := 0; i < len(base); i++ {
		base[i] = 0xFF
	}
	return pot.NewAddressFromBytes(base)
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
