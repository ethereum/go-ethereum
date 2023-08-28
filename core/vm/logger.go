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
type GasChangeReason byte

const (
	GasChangeUnspecified GasChangeReason = iota

	// GasInitialBalance is the initial balance for the call which will be equal to the gasLimit of the call
	GasInitialBalance
	// GasRefunded is the amount of gas that will be refunded to the caller for data returned to the chain
	// PR Review: Is that the right description? Called in core/state_transition.go#StateTransition.refundGas
	GasRefunded
	// GasBuyBack is the amount of gas that will be bought back by the chain and returned in Wei to the caller
	GasBuyBack
	// GasChangeIntrinsicGas is the amount of gas that will be charged for the intrinsic cost of the transaction, there is
	// always exactly one of those per transaction
	GasChangeIntrinsicGas

	// PR Review: `GasChangeContractCreation/2` are actually the EIP150 burn cost of CREATE/CREATE2 respectively.
	// I think our old name `GasChangeContractCreation/2` is not really accurate. I don't
	// think that using EIP150 as a name is a good idea as rules can change in the future. So
	// maybe we can just call them `GasChangeCreateBurn/GasChangeCreate2Burn`? Burn might
	// feels like the wrong term, `Stipend` came to mind also but I'm unsure.
	//
	// I would like also to keep the distinction between CREATE and CREATE2 however, it's an
	// important thing IMO as they are different and could use different rules in the future.

	// GasChangeContractCreation is the amount of gas that will be burned for a CREATE, today controlled by EIP150 rules
	GasChangeContractCreation
	// GasChangeContractCreation is the amount of gas that will be burned for a CREATE2, today controlled by EIP150 rules
	GasChangeContractCreation2
	// GasChangeCodeStorage is the amount of gas that will be charged for code storage
	GasChangeCodeStorage
	// GasChangeOpCode is the amount of gas that will be charged for an opcode executed by the EVM, exact opcode that was
	// performed can be check by `CaptureState` handling
	GasChangeOpCode
	// GasChangePrecompiledContract is the amount of gas that will be charged for a precompiled contract execution
	GasChangePrecompiledContract
	// GasChangeStorageColdAccess is the amount of gas that will be charged for a cold storage access as controlled by EIP2929 rules
	GasChangeStorageColdAccess

	// GasChangeCallLeftOverRefunded is the amount of gas that will be refunded to the caller after the execution of the call, if there is left over at the end of execution
	GasChangeCallLeftOverRefunded
	// GasChangeFailedExecution is the burning of the remaining gas when the execution failed without a revert
	GasChangeFailedExecution
)
