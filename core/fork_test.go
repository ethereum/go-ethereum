package core

import (
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

func (c fakeChain) Db() ethdb.Database {
	return c.db
}

func (c fakeChain) GetBlock(hash common.Hash) *types.Block {
	return GetBlock(c.db, hash)
}

func TestGetNumHash(t *testing.T) {
	chain := newFakeChain()
	genesis := WriteGenesisBlockForTesting(chain.db)

	fork, err := Fork(chain, genesis.Hash())
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
}
