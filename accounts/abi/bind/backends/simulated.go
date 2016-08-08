// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package backends

import (
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"golang.org/x/net/context"
)

// Default chain configuration which sets homestead phase at block 0 (i.e. no frontier)
var chainConfig = &core.ChainConfig{HomesteadBlock: big.NewInt(0)}

// This nil assignment ensures compile time that SimulatedBackend implements bind.ContractBackend.
var _ bind.ContractBackend = (*SimulatedBackend)(nil)

// SimulatedBackend implements bind.ContractBackend, simulating a blockchain in
// the background. Its main purpose is to allow easily testing contract bindings.
type SimulatedBackend struct {
	database   ethdb.Database   // In memory database to store our testing data
	blockchain *core.BlockChain // Ethereum blockchain to handle the consensus

	pendingBlock *types.Block   // Currently pending block that will be imported on request
	pendingState *state.StateDB // Currently pending state that will be the active on on request
}

// NewSimulatedBackend creates a new binding backend using a simulated blockchain
// for testing purposes.
func NewSimulatedBackend(accounts ...core.GenesisAccount) *SimulatedBackend {
	database, _ := ethdb.NewMemDatabase()
	core.WriteGenesisBlockForTesting(database, accounts...)
	blockchain, _ := core.NewBlockChain(database, chainConfig, new(core.FakePow), new(event.TypeMux))

	backend := &SimulatedBackend{
		database:   database,
		blockchain: blockchain,
	}
	backend.Rollback()

	return backend
}

// Commit imports all the pending transactions as a single block and starts a
// fresh new state.
func (b *SimulatedBackend) Commit() {
	if _, err := b.blockchain.InsertChain([]*types.Block{b.pendingBlock}); err != nil {
		panic(err) // This cannot happen unless the simulator is wrong, fail in that case
	}
	b.Rollback()
}

// Rollback aborts all pending transactions, reverting to the last committed state.
func (b *SimulatedBackend) Rollback() {
	blocks, _ := core.GenerateChain(nil, b.blockchain.CurrentBlock(), b.database, 1, func(int, *core.BlockGen) {})

	b.pendingBlock = blocks[0]
	b.pendingState, _ = state.New(b.pendingBlock.Root(), b.database)
}

// CodeAt implements ChainStateReader.CodeAt, returning the code associated with
// a certain account at a given block number in the blockchain.
func (b *SimulatedBackend) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	// TODO: implement block number
	statedb, _ := b.blockchain.State()
	return statedb.GetCode(contract), nil
}

// PendingCodeAt implements PendingStateReader.PendingCodeAt, returning the
// code associated with a certain account in the pending state of the blockchain.
func (b *SimulatedBackend) PendingCodeAt(ctx context.Context, contract common.Address) ([]byte, error) {
	return b.pendingState.GetCode(contract), nil
}

// CallContract implements Contractcaller.CallContract, executing the specified
// contract with the given call message at the given block number.
func (b *SimulatedBackend) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	// Create a copy of the current state db to screw around with
	block := b.blockchain.CurrentBlock()
	statedb, _ := b.blockchain.State()
	return b.callContract(ctx, call, block, statedb)
}

// PendingCallContract implements PendingContractCaller.PendingCallContract, executing
// the specified contract with the given call message against the pending state.
func (b *SimulatedBackend) PendingCallContract(ctx context.Context, call ethereum.CallMsg) ([]byte, error) {
	// Create a copy of the current state db to screw around with
	block := b.pendingBlock
	statedb := b.pendingState.Copy()
	return b.callContract(ctx, call, block, statedb)
}

// callContract implemens common code between normal and pending contract calls.
func (b *SimulatedBackend) callContract(ctx context.Context, call ethereum.CallMsg, block *types.Block, statedb *state.StateDB) ([]byte, error) {
	// If there's no code to interact with, respond with an appropriate error
	if code := statedb.GetCode(call.To); len(code) == 0 {
		return nil, bind.ErrNoCode
	}
	// Set infinite balance to the a fake caller account
	from := statedb.GetOrNewStateObject(call.From)
	from.SetBalance(common.MaxBig)

	// Wrap the call invocation to fulfil the core.Message interface
	msg := callmsg{call}

	// Execute the call and return
	vmenv := core.NewEnv(statedb, chainConfig, b.blockchain, msg, block.Header(), vm.Config{})
	gaspool := new(core.GasPool).AddGas(common.MaxBig)

	out, _, err := core.ApplyMessage(vmenv, msg, gaspool)
	return out, err
}

// PendingNonceAt implements PendingStateReader.PendingNonceAt, retrieving
// the nonce currently pending for the account.
func (b *SimulatedBackend) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return b.pendingState.GetOrNewStateObject(account).Nonce(), nil
}

// SuggestGasPrice implements ContractTransactor.SuggestGasPrice. Since the simulated
// chain doens't have miners, we just return a gas price of 1 for any call.
func (b *SimulatedBackend) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return big.NewInt(1), nil
}

// EstimateGas implements GasEstimator.EstimateGas, executing the
// requested code against the currently pending block/state and returning the used
// gas.
func (b *SimulatedBackend) EstimateGas(ctx context.Context, call ethereum.CallMsg) (*big.Int, error) {
	// Create a copy of the currently pending state db to screw around with
	block := b.pendingBlock
	statedb := b.pendingState.Copy()
	// If there's no code to interact with, respond with an appropriate error
	empty := common.Address{}
	if !reflect.DeepEqual(call.To, empty) {
		if code := statedb.GetCode(call.To); len(code) == 0 {
			return nil, bind.ErrNoCode
		}
	}
	// Set infinite balance to the a fake caller account
	from := statedb.GetOrNewStateObject(call.From)
	from.SetBalance(common.MaxBig)

	// Wrap the call invocation to fulfil the core.Message interface
	msg := callmsg{call}

	// Execute the call and return
	vmenv := core.NewEnv(statedb, chainConfig, b.blockchain, msg, block.Header(), vm.Config{})
	gaspool := new(core.GasPool).AddGas(common.MaxBig)

	_, gas, _, err := core.NewStateTransition(vmenv, msg, gaspool).TransitionDb()
	return gas, err
}

// SendTransaction implements TransactionSender.SendTransaction, delegating the raw
// transaction injection to the remote node.
func (b *SimulatedBackend) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	blocks, _ := core.GenerateChain(nil, b.blockchain.CurrentBlock(), b.database, 1, func(number int, block *core.BlockGen) {
		for _, tx := range b.pendingBlock.Transactions() {
			block.AddTx(tx)
		}
		block.AddTx(tx)
	})
	b.pendingBlock = blocks[0]
	b.pendingState, _ = state.New(b.pendingBlock.Root(), b.database)

	return nil
}

// callmsg implements core.Message to allow passing it as a transaction simulator.
type callmsg struct {
	ethereum.CallMsg
}

func (m callmsg) From() (common.Address, error)         { return m.CallMsg.From, nil }
func (m callmsg) FromFrontier() (common.Address, error) { return m.CallMsg.From, nil }
func (m callmsg) Nonce() uint64                         { return 0 }
func (m callmsg) CheckNonce() bool                      { return false }
func (m callmsg) To() *common.Address                   { return &m.CallMsg.To }
func (m callmsg) GasPrice() *big.Int                    { return m.CallMsg.GasPrice }
func (m callmsg) Gas() *big.Int                         { return m.CallMsg.Gas }
func (m callmsg) Value() *big.Int                       { return m.CallMsg.Value }
func (m callmsg) Data() []byte                          { return m.CallMsg.Data }
