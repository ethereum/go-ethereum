package les

import (
	"bytes"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/access"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

type odrTestFn func(ca *access.ChainAccess, bc *core.BlockChain, ctx *access.OdrContext, bhash common.Hash) []byte

func TestOdrGetBlockLes1(t *testing.T) { testOdr(t, 1, odrGetBlock) }

func odrGetBlock(ca *access.ChainAccess, bc *core.BlockChain, ctx *access.OdrContext, bhash common.Hash) []byte {
	block := bc.GetBlock(bhash, ctx)
	if block == nil {
		return nil
	}
	rlp, _ := rlp.EncodeToBytes(block)
	return rlp
}

func TestOdrGetReceiptsLes1(t *testing.T) { testOdr(t, 1, odrGetReceipts) }

func odrGetReceipts(ca *access.ChainAccess, bc *core.BlockChain, ctx *access.OdrContext, bhash common.Hash) []byte {
	receipts := core.GetBlockReceipts(ca, bhash, ctx)
	if receipts == nil {
		return nil
	}
	rlp, _ := rlp.EncodeToBytes(receipts)
	return rlp
}

func TestOdrAccountsLes1(t *testing.T) { testOdr(t, 1, odrAccounts) }

func odrAccounts(ca *access.ChainAccess, bc *core.BlockChain, ctx *access.OdrContext, bhash common.Hash) []byte {
	acc1Key, _ := crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	acc2Key, _ := crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	acc1Addr := crypto.PubkeyToAddress(acc1Key.PublicKey)
	acc2Addr := crypto.PubkeyToAddress(acc2Key.PublicKey)
	dummyAddr := common.HexToAddress("1234567812345678123456781234567812345678")
	acc := []common.Address{ testBankAddress, acc1Addr, acc2Addr, dummyAddr}

	trie.ClearGlobalCache()
	
	var res []byte
	for _, addr := range acc {
		header := bc.GetHeader(bhash)
		st, err := state.New(header.Root, ca, ctx)
		if err == nil {
			bal := st.GetBalance(addr)
			rlp, _ := rlp.EncodeToBytes(bal)
			res = append(res, rlp...)
		}
	}
	
	return res
}

func testOdr(t *testing.T, protocol int, fn odrTestFn) {
	// Define accounts to simulate transactions with
	acc1Key, _ := crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	acc2Key, _ := crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	acc1Addr := crypto.PubkeyToAddress(acc1Key.PublicKey)
	acc2Addr := crypto.PubkeyToAddress(acc2Key.PublicKey)

	// Create a chain generator with some simple transactions (blatantly stolen from @fjl/chain_makerts_test)
	generator := func(i int, block *core.BlockGen) {
		switch i {
		case 0:
			// In block 1, the test bank sends account #1 some ether.
			tx, _ := types.NewTransaction(block.TxNonce(testBankAddress), acc1Addr, big.NewInt(10000), params.TxGas, nil, nil).SignECDSA(testBankKey)
			block.AddTx(tx)
		case 1:
			// In block 2, the test bank sends some more ether to account #1.
			// acc1Addr passes it on to account #2.
			tx1, _ := types.NewTransaction(block.TxNonce(testBankAddress), acc1Addr, big.NewInt(1000), params.TxGas, nil, nil).SignECDSA(testBankKey)
			tx2, _ := types.NewTransaction(block.TxNonce(acc1Addr), acc2Addr, big.NewInt(1000), params.TxGas, nil, nil).SignECDSA(acc1Key)
			block.AddTx(tx1)
			block.AddTx(tx2)
		case 2:
			// Block 3 is empty but was mined by account #2.
			block.SetCoinbase(acc2Addr)
			block.SetExtra([]byte("yeehaw"))
		case 3:
			// Block 4 includes blocks 2 and 3 as uncle headers (with modified extra data).
			b2 := block.PrevBlock(1).Header()
			b2.Extra = []byte("foo")
			block.AddUncle(b2)
			b3 := block.PrevBlock(2).Header()
			b3.Extra = []byte("foo")
			block.AddUncle(b3)
		}
	}
	// Assemble the test environment
	pm, ca := newTestProtocolManagerMust(t, false, 4, generator)
	lpm, lca := newTestProtocolManagerMust(t, true, 0, nil)
	_, _, lpeer, _ := newTestPeerPair("peer", protocol, pm, lpm)
	time.Sleep(time.Millisecond * 100)
	lpm.synchronise(lpeer)

	cid := access.NewChannelID(time.Millisecond * 200)

	test := func(expFail uint64) {
		for i := uint64(0); i <= pm.blockchain.CurrentBlock().NumberU64(); i++ {
			bhash := core.GetCanonicalHash(ca.Db(), i)
			b1 := fn(ca, pm.blockchain, access.NoOdr, bhash)
			b2 := fn(lca, lpm.blockchain, access.NewContext(cid), bhash)
			eq := bytes.Equal(b1, b2)
			exp := i < expFail
			if exp && !eq {		
				t.Errorf("odr mismatch")
			}
			if !exp && eq {		
				t.Errorf("unexpected odr match")
			}
		}
	}

	// temporarily remove peer to test odr fails
	lca.UnregisterPeer(lpeer.id)
	// expect retrievals to fail (except genesis block) without a les peer
	test(1)
	lca.RegisterPeer(lpeer.id, lpeer.version, lpeer.Head(), lpeer.RequestBodies, lpeer.RequestNodeData, lpeer.RequestReceipts, lpeer.RequestProofs)
	// expect all retrievals to pass
	test(5)
	lca.UnregisterPeer(lpeer.id)
	// still expect all retrievals to pass, now data should be cached locally
	test(5)
}