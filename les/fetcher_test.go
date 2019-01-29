package les

import (
	"math/big"
	"testing"

	"net"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

func TestFetcherULCPeerSelector(t *testing.T) {

	var (
		id1 enode.ID = newNodeID(t).ID()
		id2 enode.ID = newNodeID(t).ID()
		id3 enode.ID = newNodeID(t).ID()
		id4 enode.ID = newNodeID(t).ID()
	)

	ftn1 := &fetcherTreeNode{
		hash: common.HexToHash("1"),
		td:   big.NewInt(1),
	}
	ftn2 := &fetcherTreeNode{
		hash:   common.HexToHash("2"),
		td:     big.NewInt(2),
		parent: ftn1,
	}
	ftn3 := &fetcherTreeNode{
		hash:   common.HexToHash("3"),
		td:     big.NewInt(3),
		parent: ftn2,
	}
	lf := lightFetcher{
		pm: &ProtocolManager{
			ulc: &ulc{
				trustedKeys: map[string]struct{}{
					id1.String(): {},
					id2.String(): {},
					id3.String(): {},
					id4.String(): {},
				},
				minTrustedFraction: 70,
			},
		},
		maxConfirmedTd: ftn1.td,

		peers: map[*peer]*fetcherPeerInfo{
			{
				id:        "peer1",
				Peer:      p2p.NewPeer(id1, "peer1", []p2p.Cap{}),
				isTrusted: true,
			}: {
				nodeByHash: map[common.Hash]*fetcherTreeNode{
					ftn1.hash: ftn1,
					ftn2.hash: ftn2,
				},
			},
			{
				Peer:      p2p.NewPeer(id2, "peer2", []p2p.Cap{}),
				id:        "peer2",
				isTrusted: true,
			}: {
				nodeByHash: map[common.Hash]*fetcherTreeNode{
					ftn1.hash: ftn1,
					ftn2.hash: ftn2,
				},
			},
			{
				id:        "peer3",
				Peer:      p2p.NewPeer(id3, "peer3", []p2p.Cap{}),
				isTrusted: true,
			}: {
				nodeByHash: map[common.Hash]*fetcherTreeNode{
					ftn1.hash: ftn1,
					ftn2.hash: ftn2,
					ftn3.hash: ftn3,
				},
			},
			{
				id:        "peer4",
				Peer:      p2p.NewPeer(id4, "peer4", []p2p.Cap{}),
				isTrusted: true,
			}: {
				nodeByHash: map[common.Hash]*fetcherTreeNode{
					ftn1.hash: ftn1,
				},
			},
		},
		chain: &lightChainStub{
			tds: map[common.Hash]*big.Int{},
			headers: map[common.Hash]*types.Header{
				ftn1.hash: {},
				ftn2.hash: {},
				ftn3.hash: {},
			},
		},
	}
	bestHash, bestAmount, bestTD, sync := lf.findBestRequest()

	if bestTD == nil {
		t.Fatal("Empty result")
	}

	if bestTD.Cmp(ftn2.td) != 0 {
		t.Fatal("bad td", bestTD)
	}
	if bestHash != ftn2.hash {
		t.Fatal("bad hash", bestTD)
	}

	_, _ = bestAmount, sync
}

type lightChainStub struct {
	BlockChain
	tds                         map[common.Hash]*big.Int
	headers                     map[common.Hash]*types.Header
	insertHeaderChainAssertFunc func(chain []*types.Header, checkFreq int) (int, error)
}

func (l *lightChainStub) GetHeader(hash common.Hash, number uint64) *types.Header {
	if h, ok := l.headers[hash]; ok {
		return h
	}

	return nil
}

func (l *lightChainStub) LockChain()   {}
func (l *lightChainStub) UnlockChain() {}

func (l *lightChainStub) GetTd(hash common.Hash, number uint64) *big.Int {
	if td, ok := l.tds[hash]; ok {
		return td
	}
	return nil
}

func (l *lightChainStub) InsertHeaderChain(chain []*types.Header, checkFreq int) (int, error) {
	return l.insertHeaderChainAssertFunc(chain, checkFreq)
}

func newNodeID(t *testing.T) *enode.Node {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal("generate key err:", err)
	}
	return enode.NewV4(&key.PublicKey, net.IP{}, 35000, 35000)
}
