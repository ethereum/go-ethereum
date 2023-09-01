// Copyright 2015 The go-ethereum Authors
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

package vm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// EVMLogger is used to collect execution traces from an EVM transaction
// execution. CaptureState is called for each step of the VM with the
// current VM state.
// Note that reference types are actual VM data structures; make copies
// if you need to retain them beyond the current call.
type EVMLogger interface {
	// Transaction level
	CaptureTxStart(evm *EVM, tx *types.Transaction)
	CaptureTxEnd(receipt *types.Receipt, err error)
	// Top call frame
	CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int)
	CaptureEnd(output []byte, gasUsed uint64, err error)
	// Rest of call frames
	CaptureEnter(typ OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int)
	CaptureExit(output []byte, gasUsed uint64, err error)
	// Opcode level
	CaptureState(pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, rData []byte, depth int, err error)
	CaptureFault(pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, depth int, err error)
	CaptureKeccakPreimage(hash common.Hash, data []byte)
	// Misc
	OnGasChange(old, new uint64, reason GasChangeReason)
}

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
	// always exactly one of those per transaction
	GasChangeTxIntrinsicGas
	// GasChangeTxRefunds is the sum of all refunds which happened during the tx execution (e.g. storage slot being cleared)
	// this generates an increase in gas. There is only one such gas change per transaction.
	GasChangeTxRefunds
	// GasChangeTxBuyBack is the amount of gas that will be bought back by the chain and returned in Wei to the caller at the very
	// end of the transaction's execution. There is only one such gas change per transaction.
	GasChangeTxBuyBack

	// GasChangeCallContractCreation is the amount of gas that will be burned for a CREATE, today controlled by EIP150 rules
	GasChangeCallContractCreation
	// GasChangeContractCreation is the amount of gas that will be burned for a CREATE2, today controlled by EIP150 rules
	GasChangeCallContractCreation2
	// GasChangeCallCodeStorage is the amount of gas that will be charged for code storage
	GasChangeCallCodeStorage
	// GasChangeCallOpCode is the amount of gas that will be charged for an opcode executed by the EVM, exact opcode that was
	// performed can be check by `CaptureState` handling
	GasChangeCallOpCode
	// GasChangeCallPrecompiledContract is the amount of gas that will be charged for a precompiled contract execution
	GasChangeCallPrecompiledContract
	// GasChangeCallStorageColdAccess is the amount of gas that will be charged for a cold storage access as controlled by EIP2929 rules
	GasChangeCallStorageColdAccess
	// GasChangeCallLeftOverRefunded is the amount of gas that will be refunded to the caller after the execution of the call, if
	// there is left over at the end of call's execution. This can change can happen multiple times within a single transaction as
	// each call is independent of each other.
	GasChangeCallLeftOverRefunded
	// GasChangeCallFailedExecution is the burning of the remaining gas when the execution failed without a revert
	GasChangeCallFailedExecution
)
