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

func init() {
	initDB()
}

func initDB() {
	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "")
	ethutil.Config.Db, _ = ethdb.NewMemDatabase()
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

func newBlockFromParent(addr []byte, parent *types.Block) *types.Block {
	block := types.CreateBlock(
		parent.Root(),
		parent.Hash(),
		addr,
		ethutil.BigPow(2, 32),
		nil,
		"")
	block.MinGasPrice = big.NewInt(10000000000000)
	block.Difficulty = CalcDifficulty(block, parent)
	block.Number = new(big.Int).Add(parent.Number, ethutil.Big1)
	block.GasLimit = block.CalcGasLimit(parent)
	return block
}

// Actually make a block by simulating what miner would do
func makeblock(bman *BlockManager, parent *types.Block, i int) *types.Block {
	addr := ethutil.LeftPadBytes([]byte{byte(i)}, 20)
	block := newBlockFromParent(addr, parent)
	cbase := block.State().GetOrNewStateObject(addr)
	cbase.SetGasPool(block.CalcGasLimit(parent))
	receipts, txs, _, _, _ := bman.ProcessTransactions(cbase, block.State(), block, block, types.Transactions{})
	block.SetTransactions(txs)
	block.SetReceipts(receipts)
	bman.AccumelateRewards(block.State(), block, parent)
	block.State().Update()
	return block
}

// Make a chain with real blocks
// Runs ProcessWithParent to get proper state roots
func makechain(bman *BlockManager, parent *types.Block, max int) *BlockChain {
	bman.bc.CurrentBlock = parent
	bman.bc.LastBlockHash = parent.Hash()
	blocks := make(types.Blocks, max)
	var td *big.Int
	for i := 0; i < max; i++ {
		block := makeblock(bman, parent, i)
		// add the parent and its difficulty to the working chain
		// so ProcessWithParent can access it
		bman.bc.workingChain = NewChain(types.Blocks{parent})
		bman.bc.workingChain.Back().Value.(*link).Td = td
		td, _, _ = bman.bc.processor.ProcessWithParent(block, parent)
		blocks[i] = block
		parent = block
	}
	lchain := NewChain(blocks)
	return lchain
}

// Make a new canonical chain by running TestChain and InsertChain
// on result of makechain
func newCanonical(n int) (*BlockManager, error) {
	bman := &BlockManager{bc: NewChainManager(), Pow: fakePow{}, eth: &fakeEth{}}
	bman.bc.SetProcessor(bman)
	parent := bman.bc.CurrentBlock
	lchain := makechain(bman, parent, 5)

	_, err := bman.bc.TestChain(lchain)
	if err != nil {
		return nil, err
	}
	bman.bc.InsertChain(lchain, func(block *types.Block, _ state.Messages) {})
	return bman, nil
}

// new chain manager without setLastBlock
func newChainManager() *ChainManager {
	bc := &ChainManager{}
	bc.genesisBlock = types.NewBlockFromBytes(ethutil.Encode(Genesis))
	bc.Reset()
	return bc
}

func TestExtendCanonical(t *testing.T) {
	initDB()
	// make first chain starting from genesis
	bman, err := newCanonical(5)
	if err != nil {
		t.Fatal("Could not make new canonical chain:", err)
	}

	// make second chain starting from end of first chain
	bman2 := &BlockManager{bc: NewChainManager(), Pow: fakePow{}, eth: &fakeEth{}}
	bman2.bc.SetProcessor(bman2)
	parent := bman.bc.CurrentBlock
	chainB := makechain(bman2, parent, 3)

	// test second chain against first
	td2, err := bman.bc.TestChain(chainB)
	if err != nil && !IsTDError(err) {
		t.Error("expected chainB not to give errors:", err)
	}

	if td2.Cmp(bman.bc.TD) <= 0 {
		t.Error("expected chainB to have higher difficulty. Got", td2, "expected more than", bman.bc.TD)
	}
}

func TestShorterFork(t *testing.T) {
	initDB()
	// make first chain starting from genesis
	bman, err := newCanonical(5)
	if err != nil {
		t.Fatal("Could not make new canonical chain:", err)
	}

	// make second, shorter chain, starting from genesis
	bman2 := &BlockManager{bc: newChainManager(), Pow: fakePow{}, eth: &fakeEth{}}
	bman2.bc.SetProcessor(bman2)
	parent := bman2.bc.CurrentBlock
	chainB := makechain(bman2, parent, 3)

	// test second chain against first
	td2, err := bman.bc.TestChain(chainB)
	if err != nil && !IsTDError(err) {
		t.Error("expected chainB not to give errors:", err)
	}

	if td2.Cmp(bman.bc.TD) >= 0 {
		t.Error("expected chainB to have lower difficulty. Got", td2, "expected less than", bman.bc.TD)
	}
}

func TestLongerFork(t *testing.T) {
	initDB()
	// make first chain starting from genesis
	bman, err := newCanonical(5)
	if err != nil {
		t.Fatal("Could not make new canonical chain:", err)
	}

	// make second, longer chain, starting from genesis
	bman2 := &BlockManager{bc: newChainManager(), Pow: fakePow{}, eth: &fakeEth{}}
	bman2.bc.SetProcessor(bman2)
	parent := bman2.bc.CurrentBlock
	chainB := makechain(bman2, parent, 10)

	td, err := bman.bc.TestChain(chainB)
	if err != nil {
		t.Error("expected chainB not to give errors:", err)
	}

	if td.Cmp(bman.bc.TD) <= 0 {
		t.Error("expected chainB to have higher difficulty. Got", td, "expected more than", bman.bc.TD)
	}
}

func TestEqualFork(t *testing.T) {
	initDB()
	bman, err := newCanonical(5)
	if err != nil {
		t.Fatal("Could not make new canonical chain:", err)
	}

	bman2 := &BlockManager{bc: newChainManager(), Pow: fakePow{}, eth: &fakeEth{}}
	bman2.bc.SetProcessor(bman2)
	parent := bman2.bc.CurrentBlock

	chainB := makechain(bman2, parent, 5)

	td, err := bman.bc.TestChain(chainB)
	if err != nil && !IsTDError(err) {
		t.Error("expected chainB not to give errors:", err)
	}

	if td.Cmp(bman.bc.TD) != 0 {
		t.Error("expected chainB to have equal difficulty. Got", td, "expected less than", bman.bc.TD)
	}
}

func TestBrokenChain(t *testing.T) {
	initDB()
	bman, err := newCanonical(5)
	if err != nil {
		t.Fatal("Could not make new canonical chain:", err)
	}

	bman2 := &BlockManager{bc: NewChainManager(), Pow: fakePow{}, eth: &fakeEth{}}
	bman2.bc.SetProcessor(bman2)
	parent := bman2.bc.CurrentBlock

	chainB := makechain(bman2, parent, 5)
	chainB.Remove(chainB.Front())

	_, err = bman.bc.TestChain(chainB)
	if err == nil {
		t.Error("expected broken chain to return error")
	}
}

func BenchmarkChainTesting(b *testing.B) {
	initDB()
	const chainlen = 1000

	bman, err := newCanonical(5)
	if err != nil {
		b.Fatal("Could not make new canonical chain:", err)
	}

	bman2 := &BlockManager{bc: NewChainManager(), Pow: fakePow{}, eth: &fakeEth{}}
	bman2.bc.SetProcessor(bman2)
	parent := bman2.bc.CurrentBlock

	chain := makechain(bman2, parent, chainlen)

	stime := time.Now()
	bman.bc.TestChain(chain)
	fmt.Println(chainlen, "took", time.Since(stime))
}
