package les

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/access"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto/sha3"
)

var testBankSecureTrieKey = sha3.NewKeccak256().Sum(testBankAddress[:])

type accessTestFn func(ca *access.ChainAccess, bc *core.BlockChain, bhash common.Hash) access.ObjectAccess

func TestBlockAccessLes1(t *testing.T) { testAccess(t, 1, true, tfBlockAccess) }

func tfBlockAccess(ca *access.ChainAccess, bc *core.BlockChain, bhash common.Hash) access.ObjectAccess {
	return core.NewBlockAccess(ca.Db(), bhash)
}

func TestReceiptsAccessLes1(t *testing.T) { testAccess(t, 1, true, tfReceiptsAccess) }

func tfReceiptsAccess(ca *access.ChainAccess, bc *core.BlockChain, bhash common.Hash) access.ObjectAccess {
	return core.NewReceiptsAccess(ca.Db(), bhash)
}

func TestTrieEntryAccessLes1(t *testing.T) { testAccess(t, 1, false, tfTrieEntryAccess) }

func tfTrieEntryAccess(ca *access.ChainAccess, bc *core.BlockChain, bhash common.Hash) access.ObjectAccess {
	return state.NewTrieEntryAccess(bc.GetHeader(bhash).Root, ca.Db(), testBankSecureTrieKey)
}

func TestNodeDataAccessLes1(t *testing.T) { testAccess(t, 1, true, tfNodeDataAccess) }

func tfNodeDataAccess(ca *access.ChainAccess, bc *core.BlockChain, bhash common.Hash) access.ObjectAccess {
	return state.NewNodeDataAccess(ca.Db(), bc.GetHeader(bhash).Root)
}

func testAccess(t *testing.T, protocol int, shouldCache bool, fn accessTestFn) {
	// Assemble the test environment
	pm, ca := newTestProtocolManagerMust(t, false, 4, testChainGen)
	lpm, lca := newTestProtocolManagerMust(t, true, 0, nil)
	_, _, lpeer, _ := newTestPeerPair("peer", protocol, pm, lpm)
	time.Sleep(time.Millisecond * 100)
	lpm.synchronise(lpeer)

	cid := access.NewChannelID(time.Millisecond * 200)

	test := func(expFail uint64) {
		for i := uint64(0); i <= pm.blockchain.CurrentBlock().NumberU64(); i++ {
			bhash := core.GetCanonicalHash(ca.Db(), i)
			req := fn(lca, lpm.blockchain, bhash)
			err := lca.Retrieve(req, access.NewContext(cid))
			got := err == nil
			exp := i < expFail
			if exp && !got {
				t.Errorf("object retrieval failed")
			}
			if !exp && got {
				t.Errorf("unexpected object retrieval success")
			}
		}
	}

	// temporarily remove peer to test odr fails
	lca.UnregisterPeer(lpeer.id)
	// expect retrievals to fail (except genesis block) without a les peer
	if shouldCache {
		test(1)
	} else {
		test(0)
	}
	lca.RegisterPeer(lpeer.id, lpeer.version, lpeer.Head(), lpeer.RequestBodies, lpeer.RequestNodeData, lpeer.RequestReceipts, lpeer.RequestProofs)
	// expect all retrievals to pass
	test(5)
	lca.UnregisterPeer(lpeer.id)
	if shouldCache {
	// still expect all retrievals to pass, now data should be cached locally
		test(5)
	}
}
