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
	jt[SELFBALANCE] = &operation{
		execute:     opSelfBalance,
		constantGas: GasFastStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
}

func opSelfBalance(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	balance := interpreter.evm.StateDB.GetBalance(scope.Contract.Address())
	scope.Stack.push(balance)
	return nil, nil
}

// enable1344 applies EIP-1344 (ChainID Opcode)
// - Adds an opcode that returns the current chain’s EIP-155 unique identifier
func enable1344(jt *JumpTable) {
	// New opcode
	jt[CHAINID] = &operation{
		execute:     opChainID,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
}

// opChainID implements CHAINID opcode
func opChainID(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	chainId, _ := uint256.FromBig(interpreter.evm.chainConfig.ChainID)
	scope.Stack.push(chainId)
	return nil, nil
}

// enable2200 applies EIP-2200 (Rebalance net-metered SSTORE)
func enable2200(jt *JumpTable) {
	jt[SLOAD].constantGas = params.SloadGasEIP2200
	jt[SSTORE].dynamicGas = gasSStoreEIP2200
}

// enable2929 enables "EIP-2929: Gas cost increases for state access opcodes"
// https://eips.ethereum.org/EIPS/eip-2929
func enable2929(jt *JumpTable) {
	jt[SSTORE].dynamicGas = gasSStoreEIP2929

	jt[SLOAD].constantGas = 0
	jt[SLOAD].dynamicGas = gasSLoadEIP2929

	jt[EXTCODECOPY].constantGas = params.WarmStorageReadCostEIP2929
	jt[EXTCODECOPY].dynamicGas = gasExtCodeCopyEIP2929

	jt[EXTCODESIZE].constantGas = params.WarmStorageReadCostEIP2929
	jt[EXTCODESIZE].dynamicGas = gasEip2929AccountCheck

	jt[EXTCODEHASH].constantGas = params.WarmStorageReadCostEIP2929
	jt[EXTCODEHASH].dynamicGas = gasEip2929AccountCheck

	jt[BALANCE].constantGas = params.WarmStorageReadCostEIP2929
	jt[BALANCE].dynamicGas = gasEip2929AccountCheck

	jt[CALL].constantGas = params.WarmStorageReadCostEIP2929
	jt[CALL].dynamicGas = gasCallEIP2929

	jt[CALLCODE].constantGas = params.WarmStorageReadCostEIP2929
	jt[CALLCODE].dynamicGas = gasCallCodeEIP2929

	jt[STATICCALL].constantGas = params.WarmStorageReadCostEIP2929
	jt[STATICCALL].dynamicGas = gasStaticCallEIP2929

	jt[DELEGATECALL].constantGas = params.WarmStorageReadCostEIP2929
	jt[DELEGATECALL].dynamicGas = gasDelegateCallEIP2929

	// This was previously part of the dynamic cost, but we're using it as a constantGas
	// factor here
	jt[SELFDESTRUCT].constantGas = params.SelfdestructGasEIP150
	jt[SELFDESTRUCT].dynamicGas = gasSelfdestructEIP2929
}

// enable3529 enabled "EIP-3529: Reduction in refunds":
// - Removes refunds for selfdestructs
// - Reduces refunds for SSTORE
// - Reduces max refunds to 20% gas
func enable3529(jt *JumpTable) {
	jt[SSTORE].dynamicGas = gasSStoreEIP3529
	jt[SELFDESTRUCT].dynamicGas = gasSelfdestructEIP3529
}

// enable3198 applies EIP-3198 (BASEFEE Opcode)
// - Adds an opcode that returns the current block's base fee.
func enable3198(jt *JumpTable) {
	// New opcode
	jt[BASEFEE] = &operation{
		execute:     opBaseFee,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
}

// enable1153 applies EIP-1153 "Transient Storage"
// - Adds TLOAD that reads from transient storage
// - Adds TSTORE that writes to transient storage
func enable1153(jt *JumpTable) {
	jt[TLOAD] = &operation{
		execute:     opTload,
		constantGas: params.WarmStorageReadCostEIP2929,
		minStack:    minStack(1, 1),
		maxStack:    maxStack(1, 1),
	}

	jt[TSTORE] = &operation{
		execute:     opTstore,
		constantGas: params.WarmStorageReadCostEIP2929,
		minStack:    minStack(2, 0),
		maxStack:    maxStack(2, 0),
	}
}

// opTload implements TLOAD opcode
func opTload(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	loc := scope.Stack.peek()
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
	loc := scope.Stack.pop()
	val := scope.Stack.pop()
	interpreter.evm.StateDB.SetTransientState(scope.Contract.Address(), loc.Bytes32(), val.Bytes32())
	return nil, nil
}

// opBaseFee implements BASEFEE opcode
func opBaseFee(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	baseFee, _ := uint256.FromBig(interpreter.evm.Context.BaseFee)
	scope.Stack.push(baseFee)
	return nil, nil
}

// enable3855 applies EIP-3855 (PUSH0 opcode)
func enable3855(jt *JumpTable) {
	// New opcode
	jt[PUSH0] = &operation{
		execute:     opPush0,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
}

// opPush0 implements the PUSH0 opcode
func opPush0(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	scope.Stack.push(new(uint256.Int))
	return nil, nil
}

// enable3860 enables "EIP-3860: Limit and meter initcode"
// https://eips.ethereum.org/EIPS/eip-3860
func enable3860(jt *JumpTable) {
	jt[CREATE].dynamicGas = gasCreateEip3860
	jt[CREATE2].dynamicGas = gasCreate2Eip3860
}

// enable5656 enables EIP-5656 (MCOPY opcode)
// https://eips.ethereum.org/EIPS/eip-5656
func enable5656(jt *JumpTable) {
	jt[MCOPY] = &operation{
		execute:     opMcopy,
		constantGas: GasFastestStep,
		dynamicGas:  gasMcopy,
		minStack:    minStack(3, 0),
		maxStack:    maxStack(3, 0),
		memorySize:  memoryMcopy,
	}
}

// opMcopy implements the MCOPY opcode (https://eips.ethereum.org/EIPS/eip-5656)
func opMcopy(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		dst    = scope.Stack.pop()
		src    = scope.Stack.pop()
		length = scope.Stack.pop()
	)
	// These values are checked for overflow during memory expansion calculation
	// (the memorySize function on the opcode).
	scope.Memory.Copy(dst.Uint64(), src.Uint64(), length.Uint64())
	return nil, nil
}

// opBlobHash implements the BLOBHASH opcode
func opBlobHash(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	index := scope.Stack.peek()
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
	scope.Stack.push(blobBaseFee)
	return nil, nil
}

// enable4844 applies EIP-4844 (BLOBHASH opcode)
func enable4844(jt *JumpTable) {
	jt[BLOBHASH] = &operation{
		execute:     opBlobHash,
		constantGas: GasFastestStep,
		minStack:    minStack(1, 1),
		maxStack:    maxStack(1, 1),
	}
}

// enable7516 applies EIP-7516 (BLOBBASEFEE opcode)
func enable7516(jt *JumpTable) {
	jt[BLOBBASEFEE] = &operation{
		execute:     opBlobBaseFee,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
}

// enable6780 applies EIP-6780 (deactivate SELFDESTRUCT)
func enable6780(jt *JumpTable) {
	jt[SELFDESTRUCT] = &operation{
		execute:     opSelfdestruct6780,
		dynamicGas:  gasSelfdestructEIP3529,
		constantGas: params.SelfdestructGasEIP150,
		minStack:    minStack(1, 0),
		maxStack:    maxStack(1, 0),
	}
}

func opExtCodeCopyEIP4762(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
	var (
		stack      = scope.Stack
		a          = stack.pop()
		memOffset  = stack.pop()
		codeOffset = stack.pop()
		length     = stack.pop()
	)
	uint64CodeOffset, overflow := codeOffset.Uint64WithOverflow()
	if overflow {
		uint64CodeOffset = math.MaxUint64
	}
	addr := common.Address(a.Bytes20())
	code := interpreter.evm.StateDB.GetCode(addr)
	contract := &Contract{
		Code: code,
		self: AccountRef(addr),
	}
	paddedCodeCopy, copyOffset, nonPaddedCopyLength := getDataAndAdjustedBounds(code, uint64CodeOffset, length.Uint64())
	statelessGas := interpreter.evm.AccessEvents.CodeChunksRangeGas(addr, copyOffset, nonPaddedCopyLength, uint64(len(contract.Code)), false)
	if !scope.Contract.UseGas(statelessGas, interpreter.evm.Config.Tracer, tracing.GasChangeUnspecified) {
		scope.Contract.Gas = 0
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
		scope.Stack.push(integer.SetUint64(uint64(scope.Contract.Code[*pc])))

		if !scope.Contract.IsDeployment && *pc%31 == 0 {
			// touch next chunk if PUSH1 is at the boundary. if so, *pc has
			// advanced past this boundary.
			contractAddr := scope.Contract.Address()
			statelessGas := interpreter.evm.AccessEvents.CodeChunksRangeGas(contractAddr, *pc+1, uint64(1), uint64(len(scope.Contract.Code)), false)
			if !scope.Contract.UseGas(statelessGas, interpreter.evm.Config.Tracer, tracing.GasChangeUnspecified) {
				scope.Contract.Gas = 0
				return nil, ErrOutOfGas
			}
		}
	} else {
		scope.Stack.push(integer.Clear())
	}
	return nil, nil
}

func makePushEIP4762(size uint64, pushByteSize int) executionFunc {
	return func(pc *uint64, interpreter *EVMInterpreter, scope *ScopeContext) ([]byte, error) {
		var (
			codeLen = len(scope.Contract.Code)
			start   = min(codeLen, int(*pc+1))
			end     = min(codeLen, start+pushByteSize)
		)
		scope.Stack.push(new(uint256.Int).SetBytes(
			common.RightPadBytes(
				scope.Contract.Code[start:end],
				pushByteSize,
			)),
		)

		if !scope.Contract.IsDeployment {
			contractAddr := scope.Contract.Address()
			statelessGas := interpreter.evm.AccessEvents.CodeChunksRangeGas(contractAddr, uint64(start), uint64(pushByteSize), uint64(len(scope.Contract.Code)), false)
			if !scope.Contract.UseGas(statelessGas, interpreter.evm.Config.Tracer, tracing.GasChangeUnspecified) {
				scope.Contract.Gas = 0
				return nil, ErrOutOfGas
			}
		}

		*pc += size
		return nil, nil
	}
}

func enable4762(jt *JumpTable) {
	jt[SSTORE] = &operation{
		dynamicGas: gasSStore4762,
		execute:    opSstore,
		minStack:   minStack(2, 0),
		maxStack:   maxStack(2, 0),
	}
	jt[SLOAD] = &operation{
		dynamicGas: gasSLoad4762,
		execute:    opSload,
		minStack:   minStack(1, 1),
		maxStack:   maxStack(1, 1),
	}

	jt[BALANCE] = &operation{
		execute:    opBalance,
		dynamicGas: gasBalance4762,
		minStack:   minStack(1, 1),
		maxStack:   maxStack(1, 1),
	}

	jt[EXTCODESIZE] = &operation{
		execute:    opExtCodeSize,
		dynamicGas: gasExtCodeSize4762,
		minStack:   minStack(1, 1),
		maxStack:   maxStack(1, 1),
	}

	jt[EXTCODEHASH] = &operation{
		execute:    opExtCodeHash,
		dynamicGas: gasExtCodeHash4762,
		minStack:   minStack(1, 1),
		maxStack:   maxStack(1, 1),
	}

	jt[EXTCODECOPY] = &operation{
		execute:    opExtCodeCopyEIP4762,
		dynamicGas: gasExtCodeCopyEIP4762,
		minStack:   minStack(4, 0),
		maxStack:   maxStack(4, 0),
		memorySize: memoryExtCodeCopy,
	}

	jt[CODECOPY] = &operation{
		execute:     opCodeCopy,
		constantGas: GasFastestStep,
		dynamicGas:  gasCodeCopyEip4762,
		minStack:    minStack(3, 0),
		maxStack:    maxStack(3, 0),
		memorySize:  memoryCodeCopy,
	}

	jt[SELFDESTRUCT] = &operation{
		execute:     opSelfdestruct6780,
		dynamicGas:  gasSelfdestructEIP4762,
		constantGas: params.SelfdestructGasEIP150,
		minStack:    minStack(1, 0),
		maxStack:    maxStack(1, 0),
	}

	jt[CREATE] = &operation{
		execute:     opCreate,
		constantGas: params.CreateNGasEip4762,
		dynamicGas:  gasCreateEip3860,
		minStack:    minStack(3, 1),
		maxStack:    maxStack(3, 1),
		memorySize:  memoryCreate,
	}

	jt[CREATE2] = &operation{
		execute:     opCreate2,
		constantGas: params.CreateNGasEip4762,
		dynamicGas:  gasCreate2Eip3860,
		minStack:    minStack(4, 1),
		maxStack:    maxStack(4, 1),
		memorySize:  memoryCreate2,
	}

	jt[CALL] = &operation{
		execute:    opCall,
		dynamicGas: gasCallEIP4762,
		minStack:   minStack(7, 1),
		maxStack:   maxStack(7, 1),
		memorySize: memoryCall,
	}

	jt[CALLCODE] = &operation{
		execute:    opCallCode,
		dynamicGas: gasCallCodeEIP4762,
		minStack:   minStack(7, 1),
		maxStack:   maxStack(7, 1),
		memorySize: memoryCall,
	}

	jt[STATICCALL] = &operation{
		execute:    opStaticCall,
		dynamicGas: gasStaticCallEIP4762,
		minStack:   minStack(6, 1),
		maxStack:   maxStack(6, 1),
		memorySize: memoryStaticCall,
	}

	jt[DELEGATECALL] = &operation{
		execute:    opDelegateCall,
		dynamicGas: gasDelegateCallEIP4762,
		minStack:   minStack(6, 1),
		maxStack:   maxStack(6, 1),
		memorySize: memoryDelegateCall,
	}

	jt[PUSH1] = &operation{
		execute:     opPush1EIP4762,
		constantGas: GasFastestStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
	for i := 1; i < 32; i++ {
		jt[PUSH1+OpCode(i)] = &operation{
			execute:     makePushEIP4762(uint64(i+1), i+1),
			constantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		}
	}
}

// enableEOF applies the EOF changes.
// OBS! For EOF, there are two changes:
//  1. Two separate jumptables are required. One, EOF-jumptable, is used by
//     eof contracts. This one contains things like RJUMP.
//  2. The regular non-eof jumptable also needs to be modified, specifically to
//     modify how EXTCODECOPY works under the hood.
//
// This method _only_ deals with case 1.
func enableEOF(jt *JumpTable) {
	// Deprecate opcodes
	undefined := &operation{
		execute:     opUndefined,
		constantGas: 0,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
		undefined:   true,
	}
	jt[CALL] = undefined
	jt[CALLCODE] = undefined
	jt[DELEGATECALL] = undefined
	jt[STATICCALL] = undefined
	jt[SELFDESTRUCT] = undefined
	jt[JUMP] = undefined
	jt[JUMPI] = undefined
	jt[PC] = undefined
	jt[CREATE] = undefined
	jt[CREATE2] = undefined
	jt[CODESIZE] = undefined
	jt[CODECOPY] = undefined
	jt[EXTCODESIZE] = undefined
	jt[EXTCODECOPY] = undefined
	jt[EXTCODEHASH] = undefined
	jt[GAS] = undefined
	// Allow 0xFE to terminate sections
	jt[INVALID] = &operation{
		execute:     opUndefined,
		constantGas: 0,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
	}

	// New opcodes
	jt[RJUMP] = &operation{
		execute:     opRjump,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
	}
	jt[RJUMPI] = &operation{
		execute:     opRjumpi,
		constantGas: GasFastishStep,
		minStack:    minStack(1, 0),
		maxStack:    maxStack(1, 0),
	}
	jt[RJUMPV] = &operation{
		execute:     opRjumpv,
		constantGas: GasFastishStep,
		minStack:    minStack(1, 0),
		maxStack:    maxStack(1, 0),
	}
	jt[CALLF] = &operation{
		execute:     opCallf,
		constantGas: GasFastStep,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
	}
	jt[RETF] = &operation{
		execute:     opRetf,
		constantGas: GasFastestStep,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
	}
	jt[JUMPF] = &operation{
		execute:     opJumpf,
		constantGas: GasFastStep,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
	}
	jt[EOFCREATE] = &operation{
		execute:     opEOFCreate,
		constantGas: params.Create2Gas,
		dynamicGas:  gasEOFCreate,
		minStack:    minStack(4, 1),
		maxStack:    maxStack(4, 1),
		memorySize:  memoryEOFCreate,
	}
	jt[RETURNCONTRACT] = &operation{
		execute: opReturnContract,
		// returncontract has zero constant gas cost
		dynamicGas: pureMemoryGascost,
		minStack:   minStack(2, 0),
		maxStack:   maxStack(2, 0),
		memorySize: memoryReturnContract,
	}
	jt[DATALOAD] = &operation{
		execute:     opDataLoad,
		constantGas: GasFastishStep,
		minStack:    minStack(1, 1),
		maxStack:    maxStack(1, 1),
	}
	jt[DATALOADN] = &operation{
		execute:     opDataLoadN,
		constantGas: GasFastestStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
	jt[DATASIZE] = &operation{
		execute:     opDataSize,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
	jt[DATACOPY] = &operation{
		execute:     opDataCopy,
		constantGas: GasFastestStep,
		dynamicGas:  memoryCopierGas(2),
		minStack:    minStack(3, 0),
		maxStack:    maxStack(3, 0),
		memorySize:  memoryDataCopy,
	}
	jt[DUPN] = &operation{
		execute:     opDupN,
		constantGas: GasFastestStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
	jt[SWAPN] = &operation{
		execute:     opSwapN,
		constantGas: GasFastestStep,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
	}
	jt[EXCHANGE] = &operation{
		execute:     opExchange,
		constantGas: GasFastestStep,
		minStack:    minStack(0, 0),
		maxStack:    maxStack(0, 0),
	}
	jt[RETURNDATALOAD] = &operation{
		execute:     opReturnDataLoad,
		constantGas: GasFastestStep,
		minStack:    minStack(1, 1),
		maxStack:    maxStack(1, 1),
	}
	jt[EXTCALL] = &operation{
		execute:     opExtCall,
		constantGas: params.WarmStorageReadCostEIP2929,
		dynamicGas:  makeCallVariantGasCallEIP2929(gasExtCall, 0),
		minStack:    minStack(4, 1),
		maxStack:    maxStack(4, 1),
		memorySize:  memoryExtCall,
	}
	jt[EXTDELEGATECALL] = &operation{
		execute:     opExtDelegateCall,
		dynamicGas:  makeCallVariantGasCallEIP2929(gasExtDelegateCall, 0),
		constantGas: params.WarmStorageReadCostEIP2929,
		minStack:    minStack(3, 1),
		maxStack:    maxStack(3, 1),
		memorySize:  memoryExtCall,
	}
	jt[EXTSTATICCALL] = &operation{
		execute:     opExtStaticCall,
		constantGas: params.WarmStorageReadCostEIP2929,
		dynamicGas:  makeCallVariantGasCallEIP2929(gasExtStaticCall, 0),
		minStack:    minStack(3, 1),
		maxStack:    maxStack(3, 1),
		memorySize:  memoryExtCall,
	}
}
