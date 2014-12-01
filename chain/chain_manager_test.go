package chain

import (
	"container/list"
	"fmt"
	"log"
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

// Called from each Test to re-init the DB
// Since DBs are global, we need to use two
//  to separate chains, so they don't know about
//  eachother when they shouldn't
var DB = []*ethdb.MemDatabase{}

func initDB() {
	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "")
	// we need two databases, since we need two chain managers
	for i := 0; i < 2; i++ {
		db, _ := ethdb.NewMemDatabase()
		DB = append(DB, db)
	}
	ethutil.Config.Db = DB[0]
}

// swap currently active global DB
func setDB(i int) {
	ethutil.Config.Db = DB[i]
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

// Create new block from coinbase and parent
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
func makeBlock(bman *BlockManager, parent *types.Block, i int) *types.Block {
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
func makeChain(bman *BlockManager, parent *types.Block, max int) *BlockChain {
	bman.bc.CurrentBlock = parent
	bman.bc.LastBlockHash = parent.Hash()
	blocks := make(types.Blocks, max)
	var td *big.Int
	var err error
	for i := 0; i < max; i++ {
		block := makeBlock(bman, parent, i)
		// add the parent and its difficulty to the working chain
		// so ProcessWithParent can access it
		bman.bc.workingChain = NewChain(types.Blocks{parent})
		bman.bc.workingChain.Back().Value.(*link).Td = td
		td, _, err = bman.bc.processor.ProcessWithParent(block, parent)
		if err != nil {
			fmt.Println("process with parent failed", err)
			log.Fatal(err)
		}
		blocks[i] = block
		parent = block
	}
	lchain := NewChain(blocks)
	return lchain
}

// Make a new canonical chain n block long
// by running TestChain and InsertChain
// on result of makeChain
func newCanonical(n int) (*BlockManager, error) {
	bman := &BlockManager{bc: NewChainManager(), Pow: fakePow{}, eth: &fakeEth{}}
	bman.bc.SetProcessor(bman)
	parent := bman.bc.CurrentBlock
	lchain := makeChain(bman, parent, n)

	_, err := bman.bc.TestChain(lchain)
	if err != nil {
		return nil, err
	}
	bman.bc.InsertChain(lchain, func(block *types.Block, _ state.Messages) {})
	return bman, nil
}

// Create a new chain manager starting from given block
// Effectively a fork factory
func newChainManager(block *types.Block) *ChainManager {
	bc := &ChainManager{}
	bc.genesisBlock = types.NewBlockFromBytes(ethutil.Encode(Genesis))
	if block == nil {
		bc.Reset()
	} else {
		bc.CurrentBlock = block
		bc.SetTotalDifficulty(ethutil.Big("0"))
		bc.TD = block.BlockInfo().TD
	}
	return bc
}

// Flush the blocks so their states point to the current DB,
// and not the db they were created against (simulate receiving
// blocks from a peer).
// Encode to rlp, decode back, forcing an empty trie
// with block's state root, pointing to current global DB
// (ie. setDB should be called before using this, or it has no
//	effect...)
func flushChain(chain *BlockChain) *BlockChain {
	for e := chain.Front(); e != nil; e = e.Next() {
		l := e.Value.(*link)
		b := l.Block
		encode := b.RlpEncode()
		b.RlpDecode(encode)
	}
	return chain
}

// Test fork of length N starting from block i
// Since we are simulating two peers with different chains
// and states, we must be careful to maintain two separate
// databases that know nothing about eachother
func testFork(t *testing.T, bman *BlockManager, i, N int, f func(td1, td2 *big.Int)) {
	var b *types.Block = nil
	if i > 0 {
		b = bman.bc.GetBlockByNumber(uint64(i))
	}
	// switch database to create the new chain
	setDB(1)
	bman2 := &BlockManager{bc: newChainManager(b), Pow: fakePow{}, eth: &fakeEth{}}
	bman2.bc.SetProcessor(bman2)
	parent := bman2.bc.CurrentBlock
	chainB := makeChain(bman2, parent, N)
	bman2.bc.TestChain(chainB)
	bman2.bc.InsertChain(chainB, func(block *types.Block, _ state.Messages) {})

	// Now to test second chain against first
	// we switch back to first chain's db
	setDB(0)
	// but chainB's blocks still have states that point to DB 1
	// we need to flush the chain with some fresh rlp decode/encode
	// to point to the new db
	// This simulates receiving the chain from a peer
	// (through the blockpool)
	chainB = flushChain(chainB)
	// now we try and reconstruct the states
	// evaluating the fork
	td2, err := bman.bc.TestChain(chainB)
	if err != nil && !IsTDError(err) {
		t.Error("expected chainB not to give errors:", err)
	}
	// Compare difficulties
	f(bman.bc.TD, td2)
}

// Test basic extension of canonical chain with new blocks
func TestExtendCanonical(t *testing.T) {
	initDB()
	// make first chain starting from genesis
	bman, err := newCanonical(5)
	if err != nil {
		t.Fatal("Could not make new canonical chain:", err)
	}

	f := func(td1, td2 *big.Int) {
		if td2.Cmp(td1) <= 0 {
			t.Error("expected chainB to have higher difficulty. Got", td2, "expected more than", td1)
		}
	}

	// Start fork from current height (5)
	testFork(t, bman, 5, 1, f)
	testFork(t, bman, 5, 2, f)
	testFork(t, bman, 5, 5, f)
	testFork(t, bman, 5, 10, f)
}

// Test a fork with less TD than the canonical chain
func TestShorterFork(t *testing.T) {
	initDB()
	// make first chain starting from genesis
	bman, err := newCanonical(10)
	if err != nil {
		t.Fatal("Could not make new canonical chain:", err)
	}

	f := func(td1, td2 *big.Int) {
		if td2.Cmp(td1) >= 0 {
			t.Error("expected chainB to have lower difficulty. Got", td2, "expected less than", td1)
		}
	}

	// Sum of numbers must be less than 10
	// for this to be a shorter fork
	testFork(t, bman, 0, 3, f)
	testFork(t, bman, 0, 7, f)
	testFork(t, bman, 1, 3, f)
	testFork(t, bman, 1, 7, f)
	testFork(t, bman, 5, 3, f)
	testFork(t, bman, 5, 4, f)
}

// Test a fork with more TD than canonical chain
func TestLongerFork(t *testing.T) {
	initDB()
	// make first chain starting from genesis
	bman, err := newCanonical(10)
	if err != nil {
		t.Fatal("Could not make new canonical chain:", err)
	}

	f := func(td1, td2 *big.Int) {
		if td2.Cmp(td1) <= 0 {
			t.Error("expected chainB to have higher difficulty. Got", td2, "expected more than", td1)
		}
	}

	// Sum of numbers must be greater than 10
	// for this to be a longer fork
	testFork(t, bman, 0, 11, f)
	testFork(t, bman, 0, 15, f)
	testFork(t, bman, 1, 10, f)
	testFork(t, bman, 1, 12, f)
	testFork(t, bman, 5, 6, f)
	testFork(t, bman, 5, 8, f)
}

// Test a fork with equal TD to canonical chain
func TestEqualFork(t *testing.T) {
	initDB()
	bman, err := newCanonical(10)
	if err != nil {
		t.Fatal("Could not make new canonical chain:", err)
	}

	f := func(td1, td2 *big.Int) {
		if td2.Cmp(td1) != 0 {
			t.Error("expected chainB to have equal difficulty. Got", td2, "expected less than", td1)
		}
	}

	// Sum of numbers must be equal to 10
	// for this to be an equal fork
	testFork(t, bman, 0, 10, f)
	testFork(t, bman, 1, 9, f)
	testFork(t, bman, 2, 8, f)
	testFork(t, bman, 5, 5, f)
	testFork(t, bman, 6, 4, f)
	testFork(t, bman, 9, 1, f)
}

// Test a broken chain (no common ancestor)
func TestBrokenChain(t *testing.T) {
	initDB()
	bman, err := newCanonical(5)
	if err != nil {
		t.Fatal("Could not make new canonical chain:", err)
	}

	bman2 := &BlockManager{bc: NewChainManager(), Pow: fakePow{}, eth: &fakeEth{}}
	bman2.bc.SetProcessor(bman2)
	parent := bman2.bc.CurrentBlock

	chainB := makeChain(bman2, parent, 5)
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

	chain := makeChain(bman2, parent, chainlen)

	stime := time.Now()
	bman.bc.TestChain(chain)
	fmt.Println(chainlen, "took", time.Since(stime))
}
