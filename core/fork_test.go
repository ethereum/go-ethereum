package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
)

type fakeChain struct {
	db *ethdb.LDBDatabase
}

func newFakeChain() fakeChain {
	db, _ := ethdb.NewMemDatabase()
	return fakeChain{
		db: db,
	}
}

func (c fakeChain) GetTd(common.Hash) *big.Int {
	return new(big.Int)
}

func (c fakeChain) GetBlocksFromHash(common.Hash, int) []*types.Block {
	return nil
}

func (c fakeChain) GetBlock(hash common.Hash) *types.Block {
	return GetBlock(c.db, hash)
}

func TestGetNumHash(t *testing.T) {
	chain := newFakeChain()
	genesis := WriteGenesisBlockForTesting(chain.db)
	config := &ChainConfig{HomesteadBlock: new(big.Int)}

	fork, err := Fork(chain.db, config, chain, genesis.Hash(), 0)
	if err != nil {
		t.Fatal(err)
	}

	genesisHash := fork.GetNumHash(0)
	if genesisHash != genesis.Hash() {
		t.Error("genesis hash failed")
	}

	if fork.GetNumHash(1) != (common.Hash{}) {
		t.Error("expected exmpty hash to be returned")
	}

	const chainLength = 5
	blocks := make([]*UnsealedBlock, chainLength)
	for i := 0; i < chainLength; i++ {
		unsealedBlock := fork.NewUnsealedBlock(common.Address{}, nil)
		fork.CommitBlock(new(big.Int), unsealedBlock.Block, unsealedBlock.receipts)
		blocks[i] = unsealedBlock
	}

	for i := uint64(0); i < chainLength; i++ {
		if fork.GetNumHash(fork.originN+i+1) != blocks[i].Block.Hash() {
			t.Errorf("%d failed: expected %x got %x", i, fork.GetNumHash(fork.originN+i+1), blocks[i].Block.Hash())
		}

		idx := fork.hashToIdx[blocks[i].Block.Hash()]
		if uint64(idx) != i {
			t.Errorf("%d failed: expected hash %x to map to %d got %d", i, blocks[i].Block.Hash(), i, idx)
		}
	}
}
