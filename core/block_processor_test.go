package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/pow/ezp"
)

func proc() (*BlockProcessor, *ChainManager) {
	db, _ := ethdb.NewMemDatabase()
	var mux event.TypeMux

	chainMan := NewChainManager(db, db, &mux)
	return NewBlockProcessor(db, db, ezp.New(), nil, chainMan, &mux), chainMan
}

func TestNumber(t *testing.T) {
	bp, chain := proc()
	block1 := chain.NewBlock(common.Address{})
	block1.Header().Number = big.NewInt(3)
	block1.Header().Time--

	err := bp.ValidateHeader(block1.Header(), chain.Genesis().Header())
	if err != BlockNumberErr {
		t.Errorf("expected block number error %v", err)
	}

	block1 = chain.NewBlock(common.Address{})
	err = bp.ValidateHeader(block1.Header(), chain.Genesis().Header())
	if err == BlockNumberErr {
		t.Errorf("didn't expect block number error")
	}
}
