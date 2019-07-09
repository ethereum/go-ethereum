// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package les

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

func TestULCSyncWithOnePeer(t *testing.T) {
	f := newFullPeerPair(t, 1, 4)
	l := newLightPeer(t, []string{f.Node.String()}, 100)

	if reflect.DeepEqual(f.PM.blockchain.CurrentHeader().Hash(), l.PM.blockchain.CurrentHeader().Hash()) {
		t.Fatal("blocks are equal")
	}
	_, _, err := connectPeers(f, l, 2)
	if err != nil {
		t.Fatal(err)
	}
	l.PM.fetcher.lock.Lock()
	l.PM.fetcher.nextRequest()
	l.PM.fetcher.lock.Unlock()

	if !reflect.DeepEqual(f.PM.blockchain.CurrentHeader().Hash(), l.PM.blockchain.CurrentHeader().Hash()) {
		t.Fatal("sync doesn't work")
	}
}

func TestULCReceiveAnnounce(t *testing.T) {
	f := newFullPeerPair(t, 1, 4)
	l := newLightPeer(t, []string{f.Node.String()}, 100)
	fPeer, lPeer, err := connectPeers(f, l, 2)
	if err != nil {
		t.Fatal(err)
	}
	l.PM.synchronise(fPeer)

	//check that the sync is finished correctly
	if !reflect.DeepEqual(f.PM.blockchain.CurrentHeader().Hash(), l.PM.blockchain.CurrentHeader().Hash()) {
		t.Fatal("sync doesn't work")
	}
	l.PM.peers.lock.Lock()
	if len(l.PM.peers.peers) == 0 {
		t.Fatal("peer list should not be empty")
	}
	l.PM.peers.lock.Unlock()

	time.Sleep(time.Second)
	//send a signed announce message(payload doesn't matter)
	td := f.PM.blockchain.GetTd(l.PM.blockchain.CurrentHeader().Hash(), l.PM.blockchain.CurrentHeader().Number.Uint64())
	announce := announceData{
		Number: l.PM.blockchain.CurrentHeader().Number.Uint64() + 1,
		Td:     td.Add(td, big.NewInt(1)),
	}
	announce.sign(f.Key)
	lPeer.SendAnnounce(announce)
}

func TestULCShouldNotSyncWithTwoPeersOneHaveEmptyChain(t *testing.T) {
	f1 := newFullPeerPair(t, 1, 4)
	f2 := newFullPeerPair(t, 2, 0)
	l := newLightPeer(t, []string{f1.Node.String(), f2.Node.String()}, 100)
	_, _, err := connectPeers(f1, l, 2)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = connectPeers(f2, l, 2)
	if err != nil {
		t.Fatal(err)
	}
	l.PM.fetcher.lock.Lock()
	l.PM.fetcher.nextRequest()
	l.PM.fetcher.lock.Unlock()

	if reflect.DeepEqual(f2.PM.blockchain.CurrentHeader().Hash(), l.PM.blockchain.CurrentHeader().Hash()) {
		t.Fatal("Incorrect hash: second peer has empty chain")
	}
}

func TestULCShouldNotSyncWithThreePeersOneHaveEmptyChain(t *testing.T) {
	f1 := newFullPeerPair(t, 1, 3)
	f2 := newFullPeerPair(t, 2, 4)
	f3 := newFullPeerPair(t, 3, 0)

	l := newLightPeer(t, []string{f1.Node.String(), f2.Node.String(), f3.Node.String()}, 60)
	_, _, err := connectPeers(f1, l, 2)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = connectPeers(f2, l, 2)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = connectPeers(f3, l, 2)
	if err != nil {
		t.Fatal(err)
	}
	l.PM.fetcher.lock.Lock()
	l.PM.fetcher.nextRequest()
	l.PM.fetcher.lock.Unlock()

	if !reflect.DeepEqual(f1.PM.blockchain.CurrentHeader().Hash(), l.PM.blockchain.CurrentHeader().Hash()) {
		t.Fatal("Incorrect hash")
	}
}

type pairPeer struct {
	Name string
	Node *enode.Node
	PM   *ProtocolManager
	Key  *ecdsa.PrivateKey
}

func connectPeers(full, light pairPeer, version int) (*peer, *peer, error) {
	// Create a message pipe to communicate through
	app, net := p2p.MsgPipe()

	peerLight := full.PM.newPeer(version, NetworkId, p2p.NewPeer(light.Node.ID(), light.Name, nil), net)
	peerFull := light.PM.newPeer(version, NetworkId, p2p.NewPeer(full.Node.ID(), full.Name, nil), app)

	// Start the peerLight on a new thread
	errc1 := make(chan error, 1)
	errc2 := make(chan error, 1)
	go func() {
		select {
		case light.PM.newPeerCh <- peerFull:
			errc1 <- light.PM.handle(peerFull)
		case <-light.PM.quitSync:
			errc1 <- p2p.DiscQuitting
		}
	}()
	go func() {
		select {
		case full.PM.newPeerCh <- peerLight:
			errc2 <- full.PM.handle(peerLight)
		case <-full.PM.quitSync:
			errc2 <- p2p.DiscQuitting
		}
	}()

	select {
	case <-time.After(time.Millisecond * 100):
	case err := <-errc1:
		return nil, nil, fmt.Errorf("peerLight handshake error: %v", err)
	case err := <-errc2:
		return nil, nil, fmt.Errorf("peerFull handshake error: %v", err)
	}

	return peerFull, peerLight, nil
}

// newFullPeerPair creates node with full sync mode
func newFullPeerPair(t *testing.T, index int, numberOfblocks int) pairPeer {
	db := rawdb.NewMemoryDatabase()

	pmFull, _ := newTestProtocolManagerMust(t, false, numberOfblocks, nil, nil, nil, db, nil, 0)

	peerPairFull := pairPeer{
		Name: "full node",
		PM:   pmFull,
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal("generate key err:", err)
	}
	peerPairFull.Key = key
	peerPairFull.Node = enode.NewV4(&key.PublicKey, net.ParseIP("127.0.0.1"), 35000, 35000)
	return peerPairFull
}

// newLightPeer creates node with light sync mode
func newLightPeer(t *testing.T, ulcServers []string, ulcFraction int) pairPeer {
	peers := newPeerSet()
	dist := newRequestDistributor(peers, make(chan struct{}), &mclock.System{})
	rm := newRetrieveManager(peers, dist, nil)
	ldb := rawdb.NewMemoryDatabase()

	odr := NewLesOdr(ldb, light.DefaultClientIndexerConfig, rm)

	pmLight, _ := newTestProtocolManagerMust(t, true, 0, odr, nil, peers, ldb, ulcServers, ulcFraction)
	peerPairLight := pairPeer{
		Name: "ulc node",
		PM:   pmLight,
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal("generate key err:", err)
	}
	peerPairLight.Key = key
	peerPairLight.Node = enode.NewV4(&key.PublicKey, net.IP{}, 35000, 35000)
	return peerPairLight
}
