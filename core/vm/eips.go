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
	"github.com/ethereum/go-ethereum/core/vm/roles"
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
	7701: enable7701,
	7702: enable7702,
	7939: enable7939,
	8024: enable8024,
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

func opSelfBalance(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	balance := evm.StateDB.GetBalance(scope.Contract.Address())
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
func opChainID(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	chainId, _ := uint256.FromBig(evm.chainConfig.ChainID)
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
func opTload(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	loc := scope.Stack.peek()
	hash := common.Hash(loc.Bytes32())
	val := evm.StateDB.GetTransientState(scope.Contract.Address(), hash)
	loc.SetBytes(val.Bytes())
	return nil, nil
}

// opTstore implements TSTORE opcode
func opTstore(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	if evm.readOnly {
		return nil, ErrWriteProtection
	}
	loc := scope.Stack.pop()
	val := scope.Stack.pop()
	evm.StateDB.SetTransientState(scope.Contract.Address(), loc.Bytes32(), val.Bytes32())
	return nil, nil
}

// opBaseFee implements BASEFEE opcode
func opBaseFee(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	baseFee, _ := uint256.FromBig(evm.Context.BaseFee)
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
func opPush0(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
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
func opMcopy(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
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
func opBlobHash(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	index := scope.Stack.peek()
	if index.LtUint64(uint64(len(evm.TxContext.BlobHashes))) {
		blobHash := evm.TxContext.BlobHashes[index.Uint64()]
		index.SetBytes32(blobHash[:])
	} else {
		index.Clear()
	}
	return nil, nil
}

// opBlobBaseFee implements BLOBBASEFEE opcode
func opBlobBaseFee(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	blobBaseFee, _ := uint256.FromBig(evm.Context.BlobBaseFee)
	scope.Stack.push(blobBaseFee)
	return nil, nil
}

// opCLZ implements the CLZ opcode (count leading zero bits)
func opCLZ(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	x := scope.Stack.peek()
	x.SetUint64(256 - uint64(x.BitLen()))
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

// enable7939 enables EIP-7939 (CLZ opcode)
func enable7939(jt *JumpTable) {
	jt[CLZ] = &operation{
		execute:     opCLZ,
		constantGas: GasFastStep,
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

// enable8024 applies EIP-8024 (DUPN, SWAPN, EXCHANGE)
func enable8024(jt *JumpTable) {
	jt[DUPN] = &operation{
		execute:     opDupN,
		constantGas: GasFastestStep,
		minStack:    minStack(1, 0),
		maxStack:    maxStack(0, 1),
	}
	jt[SWAPN] = &operation{
		execute:     opSwapN,
		constantGas: GasFastestStep,
		minStack:    minStack(2, 0),
		maxStack:    maxStack(0, 0),
	}
	jt[EXCHANGE] = &operation{
		execute:     opExchange,
		constantGas: GasFastestStep,
		minStack:    minStack(2, 0),
		maxStack:    maxStack(0, 0),
	}
}

func opExtCodeCopyEIP4762(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
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
	code := evm.StateDB.GetCode(addr)
	paddedCodeCopy, copyOffset, nonPaddedCopyLength := getDataAndAdjustedBounds(code, uint64CodeOffset, length.Uint64())
	consumed, wanted := evm.AccessEvents.CodeChunksRangeGas(addr, copyOffset, nonPaddedCopyLength, uint64(len(code)), false, scope.Contract.Gas)
	scope.Contract.UseGas(consumed, evm.Config.Tracer, tracing.GasChangeUnspecified)
	if consumed < wanted {
		return nil, ErrOutOfGas
	}
	scope.Memory.Set(memOffset.Uint64(), length.Uint64(), paddedCodeCopy)

	return nil, nil
}

// opPush1EIP4762 handles the special case of PUSH1 opcode for EIP-4762, which
// need not worry about the adjusted bound logic when adding the PUSHDATA to
// the list of access events.
func opPush1EIP4762(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	var (
		codeLen = uint64(len(scope.Contract.Code))
		integer = new(uint256.Int)
	)
	*pc += 1
	if *pc < codeLen {
		scope.Stack.push(integer.SetUint64(uint64(scope.Contract.Code[*pc])))

		if !scope.Contract.IsDeployment && !scope.Contract.IsSystemCall && *pc%31 == 0 {
			// touch next chunk if PUSH1 is at the boundary. if so, *pc has
			// advanced past this boundary.
			contractAddr := scope.Contract.Address()
			consumed, wanted := evm.AccessEvents.CodeChunksRangeGas(contractAddr, *pc+1, uint64(1), uint64(len(scope.Contract.Code)), false, scope.Contract.Gas)
			scope.Contract.UseGas(wanted, evm.Config.Tracer, tracing.GasChangeUnspecified)
			if consumed < wanted {
				return nil, ErrOutOfGas
			}
		}
	} else {
		scope.Stack.push(integer.Clear())
	}
	return nil, nil
}

func makePushEIP4762(size uint64, pushByteSize int) executionFunc {
	return func(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
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

		if !scope.Contract.IsDeployment && !scope.Contract.IsSystemCall {
			contractAddr := scope.Contract.Address()
			consumed, wanted := evm.AccessEvents.CodeChunksRangeGas(contractAddr, uint64(start), uint64(pushByteSize), uint64(len(scope.Contract.Code)), false, scope.Contract.Gas)
			scope.Contract.UseGas(consumed, evm.Config.Tracer, tracing.GasChangeUnspecified)
			if consumed < wanted {
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

// enable7702 the EIP-7702 changes to support delegation designators.
func enable7702(jt *JumpTable) {
	jt[CALL].dynamicGas = gasCallEIP7702
	jt[CALLCODE].dynamicGas = gasCallCodeEIP7702
	jt[STATICCALL].dynamicGas = gasStaticCallEIP7702
	jt[DELEGATECALL].dynamicGas = gasDelegateCallEIP7702
}

// opCurrentRole pushes the current execution role to the stack.
func opCurrentRole(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	role := uint256.NewInt(uint64(scope.Contract.Role))
	scope.Stack.push(role)
	return nil, nil
}

// opAcceptRole accepts the provided role from the stack.
func opAcceptRole(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	// TODO: should this return data also? What will consume it?
	role := scope.Stack.pop()
	evm.accepted = roles.Role(role.Uint64())
	return nil, errStopToken
}

type paramLoader func(*EVM, uint64) []byte

// uint64ToBytes32 converts a uint64 to a 32-byte big-endian slice.
func uint64ToBytes32(v uint64) []byte {
	return common.LeftPadBytes(new(uint256.Int).SetUint64(v).Bytes(), 32)
}

func loadType(evm *EVM, id uint64) []byte {
	return uint64ToBytes32(uint64(evm.TxContext.TxType))
}

func loadNonce(evm *EVM, id uint64) []byte {
	return uint64ToBytes32(evm.TxContext.Nonce)
}

func loadSender(evm *EVM, id uint64) []byte {
	return common.LeftPadBytes(evm.TxContext.Origin.Bytes(), 32)
}

func loadSenderValidationData(evm *EVM, id uint64) []byte {
	if evm.TxContext.Sender == nil {
		return nil
	}
	return evm.TxContext.Sender.Data
}

func loadDeployer(evm *EVM, id uint64) []byte {
	// 0 or 32 bytes: return nil when not set
	if evm.TxContext.Deployer == nil {
		return nil
	}
	return common.LeftPadBytes(evm.TxContext.Deployer.Target.Bytes(), 32)
}

func loadDeployerData(evm *EVM, id uint64) []byte {
	// dynamic, default empty array
	if evm.TxContext.Deployer == nil {
		return nil
	}
	return evm.TxContext.Deployer.Data
}

func loadPaymaster(evm *EVM, id uint64) []byte {
	// 0 or 32 bytes: return nil when not set
	if evm.TxContext.Paymaster == nil {
		return nil
	}
	return common.LeftPadBytes(evm.TxContext.Paymaster.Target.Bytes(), 32)
}

func loadPaymasterData(evm *EVM, id uint64) []byte {
	// dynamic, default empty array
	if evm.TxContext.Paymaster == nil {
		return nil
	}
	return evm.TxContext.Paymaster.Data
}

func loadSenderExecutionData(evm *EVM, id uint64) []byte {
	return evm.TxContext.SenderExecutionData
}

func loadMaxPriorityFeePerGas(evm *EVM, id uint64) []byte {
	if evm.TxContext.MaxPriorityFeePerGas == nil {
		return make([]byte, 32)
	}
	return common.LeftPadBytes(evm.TxContext.MaxPriorityFeePerGas.Bytes(), 32)
}

func loadMaxFeePerGas(evm *EVM, id uint64) []byte {
	if evm.TxContext.MaxFeePerGas == nil {
		return make([]byte, 32)
	}
	return common.LeftPadBytes(evm.TxContext.MaxFeePerGas.Bytes(), 32)
}

func loadSenderValidationGas(evm *EVM, id uint64) []byte {
	if evm.TxContext.Sender == nil {
		return uint64ToBytes32(0)
	}
	return uint64ToBytes32(evm.TxContext.Sender.Gas)
}

func loadPaymasterValidationGas(evm *EVM, id uint64) []byte {
	// 32 bytes, default 0
	if evm.TxContext.Paymaster == nil {
		return uint64ToBytes32(0)
	}
	return uint64ToBytes32(evm.TxContext.Paymaster.Gas)
}

func loadSenderExecutionGas(evm *EVM, id uint64) []byte {
	return uint64ToBytes32(evm.TxContext.SenderExecutionGas)
}

func loadPaymasterPostOpGas(evm *EVM, id uint64) []byte {
	// 32 bytes, default 0
	if evm.TxContext.Paymaster == nil {
		return uint64ToBytes32(0)
	}
	return uint64ToBytes32(evm.TxContext.Paymaster.PostOpGas)
}

func loadAccessListHash(evm *EVM, id uint64) []byte {
	return evm.TxContext.AccessListHash[:]
}

func loadAuthorizationListHash(evm *EVM, id uint64) []byte {
	return evm.TxContext.AuthorizationListHash[:]
}

func loadExecutionStatus(evm *EVM, id uint64) []byte {
	return uint64ToBytes32(evm.TxContext.ExecutionStatus)
}

func loadExecutionGasUsed(evm *EVM, id uint64) []byte {
	return uint64ToBytes32(evm.TxContext.ExecutionGasUsed)
}

func loadTxHashForSignature(evm *EVM, id uint64) []byte {
	if evm.TxContext.TxHashForSignature == nil {
		return nil
	}
	return evm.TxContext.TxHashForSignature[:]
}

// idToParam is a map of identifier to transaction parameter loader.
var idToParam = map[uint64]paramLoader{
	0x00: loadType,
	0x01: loadNonce,
	0x02: loadSender,
	0x03: loadSenderValidationData,
	0x04: loadDeployer,
	0x05: loadDeployerData,
	0x06: loadPaymaster,
	0x07: loadPaymasterData,
	0x08: loadSenderExecutionData,
	0x0B: loadMaxPriorityFeePerGas,
	0x0C: loadMaxFeePerGas,
	0x0D: loadSenderValidationGas,
	0x0E: loadPaymasterValidationGas,
	0x0F: loadSenderExecutionGas,
	0x10: loadPaymasterPostOpGas,
	0x11: loadAccessListHash,
	0x12: loadAuthorizationListHash,
	0xf1: loadExecutionStatus,
	0xf2: loadExecutionGasUsed,
	0xff: loadTxHashForSignature,
}

// opTxParamLoad will load the 32-byte word of a transaction parameter at the
// specified index.
func opTxParamLoad(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	var (
		stack = scope.Stack
		idx   = stack.pop()
		id    = stack.pop()
		get   paramLoader
		ok    bool
	)
	id64, overflow := id.Uint64WithOverflow()
	if get, ok = idToParam[id64]; !ok || overflow {
		return nil, ErrTxParamIdInvalid
	}

	param := get(evm, id.Uint64())
	value := new(uint256.Int)
	idx64, overflow := idx.Uint64WithOverflow()
	if !overflow {
		value.SetBytes(getData(param, idx64, 32))
	}
	stack.push(value)
	return nil, nil
}

// opTxParamSize loads the specified transaction parameter then computes its
// size and pushes the result to the stack.
func opTxParamSize(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	var (
		stack = scope.Stack
		id    = stack.peek()
		get   paramLoader
		ok    bool
	)
	id64, overflow := id.Uint64WithOverflow()
	if get, ok = idToParam[id64]; !ok || overflow {
		return nil, ErrTxParamIdInvalid
	}

	param := get(evm, id.Uint64())
	stack.push(id.SetUint64(uint64(len(param))))

	return nil, nil
}

// opTxParamCopy copies the specified transaction parameter into memory.
func opTxParamCopy(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	var (
		stack      = scope.Stack
		memOffset  = scope.Stack.pop()
		dataOffset = scope.Stack.pop()
		length     = scope.Stack.pop()
		id         = stack.pop()
		get        paramLoader
		ok         bool
	)

	id64, overflow := id.Uint64WithOverflow()
	if get, ok = idToParam[id64]; !ok || overflow {
		return nil, ErrTxParamIdInvalid
	}
	param := get(evm, id.Uint64())

	dataOffset64, overflow := dataOffset.Uint64WithOverflow()
	if overflow {
		dataOffset64 = math.MaxUint64
	}

	// These values are checked for overflow during gas cost calculation
	memOffset64 := memOffset.Uint64()
	length64 := length.Uint64()
	scope.Memory.Set(memOffset64, length64, getData(param, dataOffset64, length64))
	return nil, nil
}

// enable7701 enables the EIP-7701 changes to support native account
// abstraction.
func enable7701(jt *JumpTable) {
	jt[CURRENTROLE] = &operation{
		execute:     opCurrentRole,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
	jt[ACCEPTROLE] = &operation{
		execute:     opAcceptRole,
		constantGas: GasQuickStep,
		minStack:    minStack(1, 0),
		maxStack:    maxStack(1, 0),
	}
	jt[TXPARAMLOAD] = &operation{
		execute:     opTxParamLoad,
		constantGas: GasFastestStep,
		minStack:    minStack(2, 1),
		maxStack:    maxStack(2, 1),
	}
	jt[TXPARAMSIZE] = &operation{
		execute:     opTxParamSize,
		constantGas: GasQuickStep,
		minStack:    minStack(1, 1),
		maxStack:    maxStack(1, 1),
	}
	jt[TXPARAMCOPY] = &operation{
		execute:     opTxParamCopy,
		constantGas: GasFastestStep,
		dynamicGas:  gasCallDataCopy,
		minStack:    minStack(4, 0),
		maxStack:    maxStack(4, 0),
		memorySize:  memoryCallDataCopy,
	}
}
