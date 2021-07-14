package core

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"testing"
)

func TestMockContract(t *testing.T) {
	// Generate a canonical chain to act as the main dataset
	engine := ethash.NewFaker()
	db := rawdb.NewMemoryDatabase()
	genesis := (&Genesis{BaseFee: big.NewInt(params.InitialBaseFee)}).MustCommit(db)

	// We must use a pretty long chain to ensure that the fork doesn't overtake us
	// until after at least 128 blocks post tip
	blocks, _ := GenerateChain(params.TestChainConfig, genesis, engine, db, 2, func(i int, b *BlockGen) {
		b.SetCoinbase(common.Address{1})
		b.OffsetTime(-9)
	})

	// Import the canonical chain
	diskdb := rawdb.NewMemoryDatabase()
	(&Genesis{BaseFee: big.NewInt(params.InitialBaseFee)}).MustCommit(diskdb)

	chain, err := NewBlockChain(diskdb, nil, params.TestChainConfig, engine, vm.Config{}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	//if n, err := chain.InsertChain(blocks); err != nil {
	//	t.Fatalf("block %d: failed to insert into chain: %v", n, err)
	//}

	statedb, err := state.New(genesis.Root(), chain.stateCache, chain.snaps)
	_, _, _, err2 := chain.processor.Process(blocks[0], statedb, chain.vmConfig)
	if err != nil {
		fmt.Println(err2)
	}

	fmt.Print("test")
	fmt.Println(blocks)
	fmt.Println(chain)
}
