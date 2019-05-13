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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

func TestULCSyncWithOnePeer(t *testing.T) {
	f := newFullPeerPair(t, 1, 4, testChainGen)
	ulcConfig := &eth.ULCConfig{
		MinTrustedFraction: 100,
		TrustedServers:     []string{f.Node.String()},
	}

	l := newLightPeer(t, ulcConfig)

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
	f := newFullPeerPair(t, 1, 4, testChainGen)
	ulcConfig := &eth.ULCConfig{
		MinTrustedFraction: 100,
		TrustedServers:     []string{f.Node.String()},
	}

	l := newLightPeer(t, ulcConfig)
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
	f1 := newFullPeerPair(t, 1, 4, testChainGen)
	f2 := newFullPeerPair(t, 2, 0, nil)
	ulcConf := &ulc{minTrustedFraction: 100, trustedKeys: make(map[string]struct{})}
	ulcConf.trustedKeys[f1.Node.ID().String()] = struct{}{}
	ulcConf.trustedKeys[f2.Node.ID().String()] = struct{}{}
	ulcConfig := &eth.ULCConfig{
		MinTrustedFraction: 100,
		TrustedServers:     []string{f1.Node.String(), f2.Node.String()},
	}
	l := newLightPeer(t, ulcConfig)
	l.PM.ulc.minTrustedFraction = 100

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
	f1 := newFullPeerPair(t, 1, 3, testChainGen)
	f2 := newFullPeerPair(t, 2, 4, testChainGen)
	f3 := newFullPeerPair(t, 3, 0, nil)

	ulcConfig := &eth.ULCConfig{
		MinTrustedFraction: 60,
		TrustedServers:     []string{f1.Node.String(), f2.Node.String(), f3.Node.String()},
	}

	l := newLightPeer(t, ulcConfig)
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
func newFullPeerPair(t *testing.T, index int, numberOfblocks int, chainGen func(int, *core.BlockGen)) pairPeer {
	db := rawdb.NewMemoryDatabase()

	pmFull := newTestProtocolManagerMust(t, false, numberOfblocks, chainGen, nil, nil, db, nil)

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
func newLightPeer(t *testing.T, ulcConfig *eth.ULCConfig) pairPeer {
	peers := newPeerSet()
	dist := newRequestDistributor(peers, make(chan struct{}), &mclock.System{})
	rm := newRetrieveManager(peers, dist, nil)
	ldb := rawdb.NewMemoryDatabase()

	odr := NewLesOdr(ldb, light.DefaultClientIndexerConfig, rm)

	pmLight := newTestProtocolManagerMust(t, true, 0, nil, odr, peers, ldb, ulcConfig)
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
