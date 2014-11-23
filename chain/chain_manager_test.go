package chain

import (
	"container/list"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/chain/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/wire"
)

// In these tests, TD = block.Number
var TD *big.Int

func init() {
	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "")
	ethutil.Config.Db, _ = ethdb.NewMemDatabase()
}

type fakeproc struct {
}

func (self fakeproc) ProcessWithParent(a, b *types.Block) (*big.Int, state.Messages, error) {
	TD = new(big.Int).Add(TD, big.NewInt(1))
	return TD, nil, nil
}

// So we can generate blocks easily
type fakePow struct{}

func (f fakePow) Search(block *types.Block, stop <-chan struct{}) []byte { return nil }
func (f fakePow) Verify(hash []byte, diff *big.Int, nonce []byte) bool   { return true }
func (f fakePow) GetHashrate() int64                                     { return 0 }
func (f fakePow) Turbo(bool)                                             {}

// We need this guy because ProcessWithParent clears txs from the pool
type fakeEth struct{}

func (e *fakeEth) BlockManager() *BlockManager                        { return nil }
func (e *fakeEth) ChainManager() *ChainManager                        { return nil }
func (e *fakeEth) TxPool() *TxPool                                    { return &TxPool{} }
func (e *fakeEth) Broadcast(msgType wire.MsgType, data []interface{}) {}
func (e *fakeEth) PeerCount() int                                     { return 0 }
func (e *fakeEth) IsMining() bool                                     { return false }
func (e *fakeEth) IsListening() bool                                  { return false }
func (e *fakeEth) Peers() *list.List                                  { return nil }
func (e *fakeEth) KeyManager() *crypto.KeyManager                     { return nil }
func (e *fakeEth) ClientIdentity() wire.ClientIdentity                { return nil }
func (e *fakeEth) Db() ethutil.Database                               { return nil }
func (e *fakeEth) EventMux() *event.TypeMux                           { return nil }

func makechain(cman *ChainManager, max int) *BlockChain {
	blocks := make(types.Blocks, max)
	for i := 0; i < max; i++ {
		addr := ethutil.LeftPadBytes([]byte{byte(i)}, 20)
		block := cman.NewBlock(addr)
		if i != 0 {
			cman.CurrentBlock = blocks[i-1]
		}
		blocks[i] = block
	}
	return NewChain(blocks)
}

func makechain2(bman *BlockManager, max int) *BlockChain {
	parent := bman.bc.CurrentBlock
	blocks := make(types.Blocks, max)
	for i := 0; i < max; i++ {
		addr := ethutil.LeftPadBytes([]byte{byte(i)}, 20)
		block := bman.bc.NewBlock(addr)
		cbase := block.State().GetOrNewStateObject(addr)
		cbase.SetGasPool(block.CalcGasLimit(parent))
		receipts, txs, _, _, _ := bman.ProcessTransactions(cbase, block.State(), block, block, types.Transactions{})
		block.SetTransactions(txs)
		block.SetReceipts(receipts)

		bman.AccumelateRewards(block.State(), block, parent)

		block.State().Update()
		lchain := NewChain(types.Blocks{block})
		_, err := bman.bc.TestChain(lchain)
		if err != nil {
			fmt.Println("failed to run test chain!:", err)
		}
		bman.bc.InsertChain(lchain, func(block *types.Block, _ state.Messages) {})

		blocks[i] = block
		parent = block
	}
	return NewChain(blocks)
}

func TestShorterFork(t *testing.T) {
	cman := NewChainManager()
	bman := &BlockManager{bc: cman, Pow: fakePow{}, eth: &fakeEth{}}
	bman.bc.SetProcessor(bman)

	makechain2(bman, 5)

	cman2 := NewChainManager()
	cman2.Reset() // so we don't end up with last block of cman1
	bman2 := &BlockManager{bc: cman2, Pow: fakePow{}, eth: &fakeEth{}}
	bman2.bc.SetProcessor(bman2)

	chainB := makechain2(bman2, 3)

	td2, err := bman.bc.TestChain(chainB)
	if err != nil && !IsTDError(err) {
		t.Error("expected chainB not to give errors:", err)
	}

	if td2.Cmp(bman.bc.TD) >= 0 {
		t.Error("expected chainB to have lower difficulty. Got", td2, "expected less than", bman.bc.TD)
	}
}

func TestLongerFork(t *testing.T) {
	cman := NewChainManager()
	cman.SetProcessor(fakeproc{})

	TD = big.NewInt(1)
	chainA := makechain(cman, 5)

	TD = big.NewInt(1)
	chainB := makechain(cman, 10)

	td, err := cman.TestChain(chainA)
	if err != nil {
		t.Error("unable to create new TD from chainA:", err)
	}
	cman.TD = td

	_, err = cman.TestChain(chainB)
	if err != nil {
		t.Error("expected chainB not to give errors:", err)
	}
}

func TestEqualFork(t *testing.T) {
	cman := NewChainManager()
	cman.SetProcessor(fakeproc{})

	TD = big.NewInt(1)
	chainA := makechain(cman, 5)

	TD = big.NewInt(2)
	chainB := makechain(cman, 5)

	td, err := cman.TestChain(chainA)
	if err != nil {
		t.Error("unable to create new TD from chainA:", err)
	}
	cman.TD = td

	_, err = cman.TestChain(chainB)
	if err != nil {
		t.Error("expected chainB not to give errors:", err)
	}
}

func TestBrokenChain(t *testing.T) {
	cman := NewChainManager()
	cman.SetProcessor(fakeproc{})

	TD = big.NewInt(1)
	chain := makechain(cman, 5)
	chain.Remove(chain.Front())

	_, err := cman.TestChain(chain)
	if err == nil {
		t.Error("expected broken chain to return error")
	}
}

func BenchmarkChainTesting(b *testing.B) {
	const chainlen = 1000

	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "")
	ethutil.Config.Db, _ = ethdb.NewMemDatabase()

	cman := NewChainManager()
	cman.SetProcessor(fakeproc{})

	TD = big.NewInt(1)
	chain := makechain(cman, chainlen)

	stime := time.Now()
	cman.TestChain(chain)
	fmt.Println(chainlen, "took", time.Since(stime))
}
