package core

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
)

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
