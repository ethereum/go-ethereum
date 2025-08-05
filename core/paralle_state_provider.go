package core

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

type PreStateType int

const (
	BALPreState PreStateType = iota
	SeqPreState              // Only for debug
)

const preStateType = BALPreState

type PreStateProvider interface {
	PrestateAtIndex(i int) (*state.StateDB, error)
}

type SequentialPrestateProvider struct {
	statedb *state.StateDB
	block   *types.Block
	gp      *GasPool
	signer  types.Signer
	// contex
	usedGas *uint64
	evm     *vm.EVM
}

func (s *SequentialPrestateProvider) PrestateAtIndex(i int) (*state.StateDB, error) {
	if i < 0 || i > len(s.block.Transactions()) {
		return nil, fmt.Errorf("tx index %d out of range [0, %d)", i, len(s.block.Transactions()))
	}
	if i == 0 {
		return s.statedb.Copy(), nil // first transaction uses the original state
	}
	i = i - 1
	tx := s.block.Transactions()[i]
	signer := s.signer
	header := s.block.Header()
	statedb := s.statedb
	blockHash := s.block.Hash()
	blockNumber := s.block.Number()
	// execute the transaction again to simulate the state changes
	msg, err := TransactionToMessage(tx, signer, header.BaseFee)
	if err != nil {
		return nil, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
	}
	statedb.SetTxContext(tx.Hash(), i)

	_, err = ApplyTransactionWithEVM(msg, s.gp, statedb, blockNumber, blockHash, s.block.Time(), tx, s.usedGas, s.evm)
	if err != nil {
		return nil, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
	}
	return statedb.Copy(), nil
}
