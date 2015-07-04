package core

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/pow/ezp"
)

func proc() (*BlockProcessor, *ChainManager) {
	db, _ := ethdb.NewMemDatabase()
	var mux event.TypeMux

	genesis := GenesisBlock(0, db)
	chainMan, err := NewChainManager(genesis, db, db, db, thePow(), &mux)
	if err != nil {
		fmt.Println(err)
	}
	return NewBlockProcessor(db, db, ezp.New(), chainMan, &mux), chainMan
}

func TestNumber(t *testing.T) {
	pow := ezp.New()
	_, chain := proc()

	statedb := state.New(chain.Genesis().Root(), chain.stateDb)
	header := makeHeader(chain.Genesis(), statedb)
	header.Number = big.NewInt(3)
	err := ValidateHeader(pow, header, chain.Genesis(), false)
	if err != BlockNumberErr {
		t.Errorf("expected block number error, got %q", err)
	}

	header = makeHeader(chain.Genesis(), statedb)
	err = ValidateHeader(pow, header, chain.Genesis(), false)
	if err == BlockNumberErr {
		t.Errorf("didn't expect block number error")
	}
}

func TestPutReceipt(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()

	var addr common.Address
	addr[0] = 1
	var hash common.Hash
	hash[0] = 2

	receipt := new(types.Receipt)
	receipt.SetLogs(state.Logs{&state.Log{
		Address:   addr,
		Topics:    []common.Hash{hash},
		Data:      []byte("hi"),
		Number:    42,
		TxHash:    hash,
		TxIndex:   0,
		BlockHash: hash,
		Index:     0,
	}})

	PutReceipts(db, types.Receipts{receipt})
	receipt = GetReceipt(db, common.Hash{})
	if receipt == nil {
		t.Error("expected to get 1 receipt, got none.")
	}
}
