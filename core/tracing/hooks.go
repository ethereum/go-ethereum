// Copyright 2024 The go-ethereum Authors
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

package tracing

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

type ScopeContext interface {
	GetMemoryData() []byte
	GetStackData() []uint256.Int
	GetCaller() common.Address
	GetAddress() common.Address
	GetCallValue() *uint256.Int
	GetCallInput() []byte
}

type StateDB interface {
	GetBalance(common.Address) *uint256.Int
	GetNonce(common.Address) uint64
	GetCode(common.Address) []byte
	GetState(common.Address, common.Hash) common.Hash
	Exist(common.Address) bool
	GetRefund() uint64
}

// Canceler is an interface that wraps the Cancel method.
// It allows loggers to cancel EVM processing.
type Canceler interface {
	Cancel()
}

type VMContext struct {
	Coinbase    common.Address
	BlockNumber *big.Int
	Time        uint64
	Random      *common.Hash
	// Effective tx gas price
	GasPrice    *big.Int
	ChainConfig *params.ChainConfig
	StateDB     StateDB
	VM          Canceler
}

// BlockEvent is emitted upon tracing an incoming block.
// It contains the block as well as consensus related information.
type BlockEvent struct {
	Block     *types.Block
	TD        *big.Int
	Finalized *types.Header
	Safe      *types.Header
}

// OpCode is an EVM opcode
// TODO: provide utils for consumers
type OpCode byte

type LiveLogger struct {
	/*
		- VM events -
	*/
	// Transaction level
	// Call simulations don't come with a valid signature. `from` field
	// to be used for address of the caller.
	CaptureTxStart func(vm *VMContext, tx *types.Transaction, from common.Address)
	CaptureTxEnd   func(receipt *types.Receipt, err error)
	// Top call frame
	CaptureStart func(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int)
	// CaptureEnd is invoked when the processing of the top call ends.
	// See docs for `CaptureExit` for info on the `reverted` parameter.
	CaptureEnd func(output []byte, gasUsed uint64, err error, reverted bool)
	// Rest of call frames
	CaptureEnter func(typ OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int)
	// CaptureExit is invoked when the processing of a message ends.
	// `revert` is true when there was an error during the execution.
	// Exceptionally, before the homestead hardfork a contract creation that
	// ran out of gas when attempting to persist the code to database did not
	// count as a call failure and did not cause a revert of the call. This will
	// be indicated by `reverted == false` and `err == ErrCodeStoreOutOfGas`.
	CaptureExit func(output []byte, gasUsed uint64, err error, reverted bool)
	// Opcode level
	CaptureState          func(pc uint64, op OpCode, gas, cost uint64, scope ScopeContext, rData []byte, depth int, err error)
	CaptureFault          func(pc uint64, op OpCode, gas, cost uint64, scope ScopeContext, depth int, err error)
	CaptureKeccakPreimage func(hash common.Hash, data []byte)
	// Misc
	OnGasChange func(old, new uint64, reason GasChangeReason)

	/*
		- Chain events -
	*/
	OnBlockchainInit func(chainConfig *params.ChainConfig)
	// OnBlockStart is called before executing `block`.
	// `td` is the total difficulty prior to `block`.
	OnBlockStart func(event BlockEvent)
	OnBlockEnd   func(err error)
	// OnSkippedBlock indicates a block was skipped during processing
	// due to it being known previously. This can happen e.g. when recovering
	// from a crash.
	OnSkippedBlock func(event BlockEvent)
	OnGenesisBlock func(genesis *types.Block, alloc types.GenesisAlloc)

	/*
		- State events -
	*/
	OnBalanceChange func(addr common.Address, prev, new *big.Int, reason BalanceChangeReason)
	OnNonceChange   func(addr common.Address, prev, new uint64)
	OnCodeChange    func(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte)
	OnStorageChange func(addr common.Address, slot common.Hash, prev, new common.Hash)
	OnLog           func(log *types.Log)
}

// BalanceChangeReason is used to indicate the reason for a balance change, useful
// for tracing and reporting.
type BalanceChangeReason byte

const (
	BalanceChangeUnspecified BalanceChangeReason = 0

	// Issuance
	// BalanceIncreaseRewardMineUncle is a reward for mining an uncle block.
	BalanceIncreaseRewardMineUncle BalanceChangeReason = 1
	// BalanceIncreaseRewardMineBlock is a reward for mining a block.
	BalanceIncreaseRewardMineBlock BalanceChangeReason = 2
	// BalanceIncreaseWithdrawal is ether withdrawn from the beacon chain.
	BalanceIncreaseWithdrawal BalanceChangeReason = 3
	// BalanceIncreaseGenesisBalance is ether allocated at the genesis block.
	BalanceIncreaseGenesisBalance BalanceChangeReason = 4

	// Transaction fees
	// BalanceIncreaseRewardTransactionFee is the transaction tip increasing block builder's balance.
	BalanceIncreaseRewardTransactionFee BalanceChangeReason = 5
	// BalanceDecreaseGasBuy is spent to purchase gas for execution a transaction.
	// Part of this gas will be burnt as per EIP-1559 rules.
	BalanceDecreaseGasBuy BalanceChangeReason = 6
	// BalanceIncreaseGasReturn is ether returned for unused gas at the end of execution.
	BalanceIncreaseGasReturn BalanceChangeReason = 7

	// DAO fork
	// BalanceIncreaseDaoContract is ether sent to the DAO refund contract.
	BalanceIncreaseDaoContract BalanceChangeReason = 8
	// BalanceDecreaseDaoAccount is ether taken from a DAO account to be moved to the refund contract.
	BalanceDecreaseDaoAccount BalanceChangeReason = 9

	// BalanceChangeTransfer is ether transferred via a call.
	// it is a decrease for the sender and an increase for the recipient.
	BalanceChangeTransfer BalanceChangeReason = 10
	// BalanceChangeTouchAccount is a transfer of zero value. It is only there to
	// touch-create an account.
	BalanceChangeTouchAccount BalanceChangeReason = 11

	// BalanceIncreaseSelfdestruct is added to the recipient as indicated by a selfdestructing account.
	BalanceIncreaseSelfdestruct BalanceChangeReason = 12
	// BalanceDecreaseSelfdestruct is deducted from a contract due to self-destruct.
	BalanceDecreaseSelfdestruct BalanceChangeReason = 13
	// BalanceDecreaseSelfdestructBurn is ether that is sent to an already self-destructed
	// account within the same tx (captured at end of tx).
	// Note it doesn't account for a self-destruct which appoints itself as recipient.
	BalanceDecreaseSelfdestructBurn BalanceChangeReason = 14
)

// GasChangeReason is used to indicate the reason for a gas change, useful
// for tracing and reporting.
//
// There is essentially two types of gas changes, those that can be emitted once per transaction
// and those that can be emitted on a call basis, so possibly multiple times per transaction.
//
// They can be recognized easily by their name, those that start with `GasChangeTx` are emitted
// once per transaction, while those that start with `GasChangeCall` are emitted on a call basis.
type GasChangeReason byte

const (
	GasChangeUnspecified GasChangeReason = iota

	// GasChangeTxInitialBalance is the initial balance for the call which will be equal to the gasLimit of the call. There is only
	// one such gas change per transaction.
	GasChangeTxInitialBalance
	// GasChangeTxIntrinsicGas is the amount of gas that will be charged for the intrinsic cost of the transaction, there is
	// always exactly one of those per transaction.
	GasChangeTxIntrinsicGas
	// GasChangeTxRefunds is the sum of all refunds which happened during the tx execution (e.g. storage slot being cleared)
	// this generates an increase in gas. There is at most one of such gas change per transaction.
	GasChangeTxRefunds
	// GasChangeTxLeftOverReturned is the amount of gas left over at the end of transaction's execution that will be returned
	// to the chain. This change will always be a negative change as we "drain" left over gas towards 0. If there was no gas
	// left at the end of execution, no such even will be emitted. The returned gas's value in Wei is returned to caller.
	// There is at most one of such gas change per transaction.
	GasChangeTxLeftOverReturned

	// GasChangeCallInitialBalance is the initial balance for the call which will be equal to the gasLimit of the call. There is only
	// one such gas change per call.
	GasChangeCallInitialBalance
	// GasChangeCallLeftOverReturned is the amount of gas left over that will be returned to the caller, this change will always
	// be a negative change as we "drain" left over gas towards 0. If there was no gas left at the end of execution, no such even
	// will be emitted.
	GasChangeCallLeftOverReturned
	// GasChangeCallLeftOverRefunded is the amount of gas that will be refunded to the call after the child call execution it
	// executed completed. This value is always positive as we are giving gas back to the you, the left over gas of the child.
	// If there was no gas left to be refunded, no such even will be emitted.
	GasChangeCallLeftOverRefunded
	// GasChangeCallContractCreation is the amount of gas that will be burned for a CREATE.
	GasChangeCallContractCreation
	// GasChangeContractCreation is the amount of gas that will be burned for a CREATE2.
	GasChangeCallContractCreation2
	// GasChangeCallCodeStorage is the amount of gas that will be charged for code storage.
	GasChangeCallCodeStorage
	// GasChangeCallOpCode is the amount of gas that will be charged for an opcode executed by the EVM, exact opcode that was
	// performed can be check by `CaptureState` handling.
	GasChangeCallOpCode
	// GasChangeCallPrecompiledContract is the amount of gas that will be charged for a precompiled contract execution.
	GasChangeCallPrecompiledContract
	// GasChangeCallStorageColdAccess is the amount of gas that will be charged for a cold storage access as controlled by EIP2929 rules.
	GasChangeCallStorageColdAccess
	// GasChangeCallFailedExecution is the burning of the remaining gas when the execution failed without a revert.
	GasChangeCallFailedExecution

	// GasChangeIgnored is a special value that can be used to indicate that the gas change should be ignored as
	// it will be "manually" tracked by a direct emit of the gas change event.
	GasChangeIgnored GasChangeReason = 0xFF
)
