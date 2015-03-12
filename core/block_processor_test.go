package core

import (
	"math/big"
	"testing"

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
	block1 := chain.NewBlock(nil)
	block1.Header().Number = big.NewInt(3)

	err := bp.ValidateHeader(block1.Header(), chain.Genesis().Header())
	if err != BlockNumberErr {
		t.Errorf("expected block number error")
	}

	block1 = chain.NewBlock(nil)
	err = bp.ValidateHeader(block1.Header(), chain.Genesis().Header())
	if err == BlockNumberErr {
		t.Errorf("didn't expect block number error")
	}
}
