// Copyright 2019 The go-ethereum Authors
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
	"fmt"
	"math"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var activators = map[int]func(*JumpTable){
	5656: enable5656,
	6780: enable6780,
	3855: enable3855,
	3860: enable3860,
	3529: enable3529,
	3198: enable3198,
	2929: enable2929,
	2200: enable2200,
	1884: enable1884,
	1344: enable1344,
	1153: enable1153,
	4762: enable4762,
	7702: enable7702,
	7939: enable7939,
}

// EnableEIP enables the given EIP on the config.
// This operation writes in-place, and callers need to ensure that the globally
// defined jump tables are not polluted.
func EnableEIP(eipNum int, jt *JumpTable) error {
	enablerFn, ok := activators[eipNum]
	if !ok {
		return fmt.Errorf("undefined eip %d", eipNum)
	}
	enablerFn(jt)
	return nil
}

func ValidEip(eipNum int) bool {
	_, ok := activators[eipNum]
	return ok
}
func ActivateableEips() []string {
	var nums []string
	for k := range activators {
		nums = append(nums, fmt.Sprintf("%d", k))
	}
	sort.Strings(nums)
	return nums
}

// enable1884 applies EIP-1884 to the given jump table:
// - Increase cost of BALANCE to 700
// - Increase cost of EXTCODEHASH to 700
// - Increase cost of SLOAD to 800
// - Define SELFBALANCE, with cost GasFastStep (5)
func enable1884(jt *JumpTable) {
	// Gas cost changes
	jt[SLOAD].constantGas = params.SloadGasEIP1884
	jt[BALANCE].constantGas = params.BalanceGasEIP1884
	jt[EXTCODEHASH].constantGas = params.ExtcodeHashGasEIP1884

	// New opcode
	jt[SELFBALANCE] = operation{
		execute:     opSelfBalance,
		constantGas: GasFastStep,
	}
}

func opSelfBalance(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	balance := interpreter.evm.StateDB.GetBalance(scope.Contract.Address())
	return nil, scope.Stack.push(balance)
}

// enable1344 applies EIP-1344 (ChainID Opcode)
// - Adds an opcode that returns the current chain’s EIP-155 unique identifier
func enable1344(jt *JumpTable) {
	// New opcode
	jt[CHAINID] = operation{
		execute:     opChainID,
		constantGas: GasQuickStep,
	}
}

// opChainID implements CHAINID opcode
func opChainID(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	chainId, _ := uint256.FromBig(interpreter.evm.chainConfig.ChainID)
	return nil, scope.Stack.push(chainId)
}

// enable2200 applies EIP-2200 (Rebalance net-metered SSTORE)
func enable2200(jt *JumpTable) {
	jt[SLOAD].constantGas = params.SloadGasEIP2200
	jt[SSTORE].execute = opSstoreEIP2200
}

// enable2929 enables "EIP-2929: Gas cost increases for state access opcodes"
// https://eips.ethereum.org/EIPS/eip-2929
func enable2929(jt *JumpTable) {
	jt[SSTORE].execute = opSstoreEIP2929

	jt[SLOAD].constantGas = 0
	jt[SLOAD].execute = opSLoadEIP2929

	jt[EXTCODECOPY].constantGas = params.WarmStorageReadCostEIP2929
	jt[EXTCODECOPY].execute = opExtCodeCopyEIP2929

	jt[EXTCODESIZE].constantGas = params.WarmStorageReadCostEIP2929
	jt[EXTCODESIZE].execute = opExtCodeSizeEIP2929

	jt[EXTCODEHASH].constantGas = params.WarmStorageReadCostEIP2929
	jt[EXTCODEHASH].execute = opExtCodeHashEIP2929

	jt[BALANCE].constantGas = params.WarmStorageReadCostEIP2929
	jt[BALANCE].execute = opBalanceEIP2929

	jt[CALL].constantGas = params.WarmStorageReadCostEIP2929
	jt[CALL].execute = opCallEIP2929

	jt[CALLCODE].constantGas = params.WarmStorageReadCostEIP2929
	jt[CALLCODE].execute = opCallCodeEIP2929

	jt[STATICCALL].constantGas = params.WarmStorageReadCostEIP2929
	jt[STATICCALL].execute = opStaticCallEIP2929

	jt[DELEGATECALL].constantGas = params.WarmStorageReadCostEIP2929
	jt[DELEGATECALL].execute = opDelegateCallEIP2929

	// This was previously part of the dynamic cost, but we're using it as a constantGas
	// factor here
	jt[SELFDESTRUCT].constantGas = params.SelfdestructGasEIP150
	jt[SELFDESTRUCT].execute = opSelfdestructEIP2929
}

// enable3529 enabled "EIP-3529: Reduction in refunds":
// - Removes refunds for selfdestructs
// - Reduces refunds for SSTORE
// - Reduces max refunds to 20% gas
func enable3529(jt *JumpTable) {
	jt[SSTORE].execute = opSstoreEIP3529
	jt[SELFDESTRUCT].execute = opSelfdestructEIP3529
}

// enable3198 applies EIP-3198 (BASEFEE Opcode)
// - Adds an opcode that returns the current block's base fee.
func enable3198(jt *JumpTable) {
	// New opcode
	jt[BASEFEE] = operation{
		execute:     opBaseFee,
		constantGas: GasQuickStep,
	}
}

// enable1153 applies EIP-1153 "Transient Storage"
// - Adds TLOAD that reads from transient storage
// - Adds TSTORE that writes to transient storage
func enable1153(jt *JumpTable) {
	jt[TLOAD] = operation{
		execute:     opTload,
		constantGas: params.WarmStorageReadCostEIP2929,
	}

	jt[TSTORE] = operation{
		execute:     opTstore,
		constantGas: params.WarmStorageReadCostEIP2929,
	}
}

// opTload implements TLOAD opcode
func opTload(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	loc, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}

	hash := common.Hash(loc.Bytes32())
	val := interpreter.evm.StateDB.GetTransientState(scope.Contract.Address(), hash)
	loc.SetBytes(val.Bytes())
	return nil, nil
}

// opTstore implements TSTORE opcode
func opTstore(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	if interpreter.readOnly {
		return nil, ErrWriteProtection
	}
	loc, val, err := scope.Stack.pop2(0)
	if err != nil {
		return nil, err
	}

	interpreter.evm.StateDB.SetTransientState(scope.Contract.Address(), loc.Bytes32(), val.Bytes32())
	return nil, nil
}

// opBaseFee implements BASEFEE opcode
func opBaseFee(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	baseFee, _ := uint256.FromBig(interpreter.evm.Context.BaseFee)
	return nil, scope.Stack.push(baseFee)
}

// enable3855 applies EIP-3855 (PUSH0 opcode)
func enable3855(jt *JumpTable) {
	// New opcode
	jt[PUSH0] = operation{
		execute:     opPush0,
		constantGas: GasQuickStep,
	}
}

// opPush0 implements the PUSH0 opcode
func opPush0(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	return nil, scope.Stack.push(new(uint256.Int))
}

// enable3860 enables "EIP-3860: Limit and meter initcode"
// https://eips.ethereum.org/EIPS/eip-3860
func enable3860(jt *JumpTable) {
	jt[CREATE].execute = opCreateEIP3860
	jt[CREATE2].execute = opCreate2EIP3860
}

// enable5656 enables EIP-5656 (MCOPY opcode)
// https://eips.ethereum.org/EIPS/eip-5656
func enable5656(jt *JumpTable) {
	jt[MCOPY] = operation{
		execute:     opMcopy,
		constantGas: GasFastestStep,
	}
}

// opMcopy implements the MCOPY opcode (https://eips.ethereum.org/EIPS/eip-5656)
func opMcopy(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	dst, src, length, err := scope.Stack.pop3(0)
	if err != nil {
		return nil, err
	}

	mStart := dst
	if src.Gt(mStart) {
		mStart = src
	}
	memorySize, err := calculateMemorySize(mStart, length)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := memoryCopierGas(scope.Memory, memorySize, length)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	// These values are checked for overflow during memory expansion calculation
	// (the memorySize function on the opcode).
	scope.Memory.Copy(dst.Uint64(), src.Uint64(), length.Uint64())
	return nil, nil
}

// opBlobHash implements the BLOBHASH opcode
func opBlobHash(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	index, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}

	if index.LtUint64(uint64(len(interpreter.evm.TxContext.BlobHashes))) {
		blobHash := interpreter.evm.TxContext.BlobHashes[index.Uint64()]
		index.SetBytes32(blobHash[:])
	} else {
		index.Clear()
	}
	return nil, nil
}

// opBlobBaseFee implements BLOBBASEFEE opcode
func opBlobBaseFee(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	blobBaseFee, _ := uint256.FromBig(interpreter.evm.Context.BlobBaseFee)
	return nil, scope.Stack.push(blobBaseFee)
}

// opCLZ implements the CLZ opcode (count leading zero bytes)
func opCLZ(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	x, err := scope.Stack.pop(1)
	if err != nil {
		return nil, err
	}

	x.SetUint64(256 - uint64(x.BitLen()))
	return nil, nil
}

// enable4844 applies EIP-4844 (BLOBHASH opcode)
func enable4844(jt *JumpTable) {
	jt[BLOBHASH] = operation{
		execute:     opBlobHash,
		constantGas: GasFastestStep,
	}
}

// enable7939 enables EIP-7939 (CLZ opcode)
func enable7939(jt *JumpTable) {
	jt[CLZ] = operation{
		execute:     opCLZ,
		constantGas: GasFastStep,
	}
}

// enable7516 applies EIP-7516 (BLOBBASEFEE opcode)
func enable7516(jt *JumpTable) {
	jt[BLOBBASEFEE] = operation{
		execute:     opBlobBaseFee,
		constantGas: GasQuickStep,
	}
}

// enable6780 applies EIP-6780 (deactivate SELFDESTRUCT)
func enable6780(jt *JumpTable) {
	jt[SELFDESTRUCT] = operation{
		execute:     opSelfdestructEIP6780,
		constantGas: params.SelfdestructGasEIP150,
	}
}

func opExtCodeCopyEIP4762(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	a, memOffset, codeOffset, length, err := scope.Stack.pop4(0)
	if err != nil {
		return nil, err
	}

	memorySize, err := calculateMemorySize(memOffset, length)
	if err != nil {
		return nil, err
	}
	dynamicCost, err := gasExtCodeCopyEIP4762(interpreter.evm, scope.Contract, scope.Memory, memorySize, a, length)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOutOfGas, err)
	}
	if !scope.Contract.UseGas(dynamicCost, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeCallOpCodeDynamic) {
		return nil, ErrOutOfGas
	}
	scope.Memory.Resize(memorySize)

	uint64CodeOffset, overflow := codeOffset.Uint64WithOverflow()
	if overflow {
		uint64CodeOffset = math.MaxUint64
	}
	addr := common.Address(a.Bytes20())
	code := interpreter.evm.StateDB.GetCode(addr)
	paddedCodeCopy, copyOffset, nonPaddedCopyLength := getDataAndAdjustedBounds(code, uint64CodeOffset, length.Uint64())
	consumed, wanted := interpreter.evm.AccessEvents.CodeChunksRangeGas(addr, copyOffset, nonPaddedCopyLength, uint64(len(code)), false, scope.Contract.Gas)
	scope.Contract.UseGas(consumed, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeUnspecified)
	if consumed < wanted {
		return nil, ErrOutOfGas
	}
	scope.Memory.Set(memOffset.Uint64(), length.Uint64(), paddedCodeCopy)

	return nil, nil
}

// opPush1EIP4762 handles the special case of PUSH1 opcode for EIP-4762, which
// need not worry about the adjusted bound logic when adding the PUSHDATA to
// the list of access events.
func opPush1EIP4762(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		codeLen = uint64(len(scope.Contract.Code))
		integer = new(uint256.Int)
	)
	*pc += 1
	if *pc < codeLen {
		if err := scope.Stack.push(integer.SetUint64(uint64(scope.Contract.Code[*pc]))); err != nil {
			return nil, err
		}

		if !scope.Contract.IsDeployment && !scope.Contract.IsSystemCall && *pc%31 == 0 {
			// touch next chunk if PUSH1 is at the boundary. if so, *pc has
			// advanced past this boundary.
			contractAddr := scope.Contract.Address()
			consumed, wanted := interpreter.evm.AccessEvents.CodeChunksRangeGas(contractAddr, *pc+1, uint64(1), uint64(len(scope.Contract.Code)), false, scope.Contract.Gas)
			scope.Contract.UseGas(wanted, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeUnspecified)
			if consumed < wanted {
				return nil, ErrOutOfGas
			}
		}
		return nil, nil
	}
	return nil, scope.Stack.push(integer.Clear())
}

func makePushEIP4762(size uint64, pushByteSize int) executionFunc {
	return func(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
		var (
			codeLen = len(scope.Contract.Code)
			start   = min(codeLen, int(*pc+1))
			end     = min(codeLen, start+pushByteSize)
		)
		if err := scope.Stack.push(new(uint256.Int).SetBytes(
			common.RightPadBytes(
				scope.Contract.Code[start:end],
				pushByteSize,
			)),
		); err != nil {
			return nil, err
		}

		if !scope.Contract.IsDeployment && !scope.Contract.IsSystemCall {
			contractAddr := scope.Contract.Address()
			consumed, wanted := interpreter.evm.AccessEvents.CodeChunksRangeGas(contractAddr, uint64(start), uint64(pushByteSize), uint64(len(scope.Contract.Code)), false, scope.Contract.Gas)
			scope.Contract.UseGas(consumed, interpreter.evm.Config.Tracer.OnGasChange, tracing.GasChangeUnspecified)
			if consumed < wanted {
				return nil, ErrOutOfGas
			}
		}

		*pc += size
		return nil, nil
	}
}

func enable4762(jt *JumpTable) {
	jt[SSTORE] = operation{
		execute: opSstoreEIP4762,
	}
	jt[SLOAD] = operation{
		execute: opSLoadEIP4762,
	}

	jt[BALANCE] = operation{
		execute: opBalanceEIP4762,
	}

	jt[EXTCODESIZE] = operation{
		execute: opExtCodeSizeEIP4762,
	}

	jt[EXTCODEHASH] = operation{
		execute: opExtCodeHashEIP4762,
	}

	jt[EXTCODECOPY] = operation{
		execute: opExtCodeCopyEIP4762,
	}

	jt[CODECOPY] = operation{
		execute:     opCodeCopyEIP4762,
		constantGas: GasFastestStep,
	}

	jt[SELFDESTRUCT] = operation{
		execute:     opSelfdestructEIP4762,
		constantGas: params.SelfdestructGasEIP150,
	}

	jt[CREATE] = operation{
		execute:     opCreateEIP3860,
		constantGas: params.CreateNGasEip4762,
	}

	jt[CREATE2] = operation{
		execute:     opCreate2EIP3860,
		constantGas: params.CreateNGasEip4762,
	}

	jt[CALL] = operation{
		execute: opCallEIP4762,
	}

	jt[CALLCODE] = operation{
		execute: opCallCodeEIP4762,
	}

	jt[STATICCALL] = operation{
		execute: opStaticCallEIP4762,
	}

	jt[DELEGATECALL] = operation{
		execute: opDelegateCallEIP4762,
	}

	jt[PUSH1] = operation{
		execute:     opPush1EIP4762,
		constantGas: GasFastestStep,
	}
	for i := 1; i < 32; i++ {
		jt[PUSH1+OpCode(i)] = operation{
			execute:     makePushEIP4762(uint64(i+1), i+1),
			constantGas: GasFastestStep,
		}
	}
}

// enable7702 the EIP-7702 changes to support delegation designators.
func enable7702(jt *JumpTable) {
	jt[CALL].execute = opCallEIP7702
	jt[CALLCODE].execute = opCallCodeEIP7702
	jt[STATICCALL].execute = opStaticCallEIP7702
	jt[DELEGATECALL].execute = opDelegateCallEIP7702
}
