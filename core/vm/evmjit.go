// Copyright 2017 The go-ethereum Authors
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

/*

#include <evmjit.h>


static struct evm_instance* new_evmjit()
{
	// Declare exported Go functions. The are ABI compatible with C callbacks
	// but differ by type names and const pointers.
	void queryState(void*, size_t, int, void* addr, void*);
	void updateState(size_t env, int key, void* addr, void* arg1, void* arg2);
	long long call(void* env, int kind, long long gas, void* address,
		void* value, void* input, size_t input_size, void* output,
		size_t output_size);
	void getTxCtx(void*, size_t);
	void getBlockHash(void*, size_t, long long);

	struct evm_factory factory = evmjit_get_factory();
	return factory.create((evm_query_state_fn)queryState,
		(evm_update_state_fn)updateState, (evm_call_fn)call,
		(evm_get_tx_context_fn)getTxCtx, (evm_get_block_hash_fn)getBlockHash);
}

static struct evm_result evm_execute(struct evm_instance* instance,
	struct evm_env* env, enum evm_mode mode, struct evm_uint256be code_hash,
	uint8_t const* code, size_t code_size, struct evm_message msg)
{
	return instance->execute(instance, env, mode, code_hash, code, code_size,
		msg);
}

static void evm_release_result(struct evm_result* result)
{
	result->release(result);
}

#cgo CFLAGS:  -I/home/chfast/Projects/ethereum/evmjit/include
#cgo LDFLAGS: -levmjit-standalone -lstdc++ -lm -ldl -L/home/chfast/Projects/ethereum/evmjit/build/release-llvm/libevmjit
*/
import "C"

import (
	"fmt"
	"math/big"
	"reflect"
	"sync"
	"unsafe"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

type EVMJIT struct {
	jit  *C.struct_evm_instance
	env  *EVM
}

type EVMCContext struct {
	contract *Contract
	env      *EVM
}

func NewJit(env *EVM, cfg Config) *EVMJIT {
	// FIXME: Destroy the jit later.
	return &EVMJIT{C.new_evmjit(), env}
}


var contextMap = make(map[uintptr]*EVMCContext)
var contextMapMu sync.Mutex

func pinCtx(ctx *EVMCContext) uintptr {
	contextMapMu.Lock()

	// Find empty slot in the map starting from the map length.
	id := uintptr(len(contextMap))
	for contextMap[id] != nil {
		id++
	}
	contextMap[id] = ctx
	contextMapMu.Unlock()
	return id
}

func unpinCtx(id uintptr) {
	contextMapMu.Lock()
	delete(contextMap, id)
	contextMapMu.Unlock()
}

func getCtx(idx uintptr) *EVMCContext {
	contextMapMu.Lock()
	defer contextMapMu.Unlock()
	return contextMap[idx]
}

func getEnv(idx uintptr) *EVM {
	return getCtx(idx).env
}

func HashToEvmc(hash common.Hash) C.struct_evm_uint256be {
	return C.struct_evm_uint256be{bytes: *(*[32]C.uint8_t)(unsafe.Pointer(&hash[0]))}
}

func AddressToEvmc(addr common.Address) C.struct_evm_uint160be {
	return C.struct_evm_uint160be{bytes: *(*[20]C.uint8_t)(unsafe.Pointer(&addr[0]))}
}

func BigToEvmc(i *big.Int) C.struct_evm_uint256be {
	return HashToEvmc(common.BigToHash(i))
}

func assert(cond bool, msg string) {
	if !cond {
		panic(fmt.Sprintf("Assertion failure! %v", msg))
	}
}

type MemoryRef struct {
	ptr *C.uint8_t
	len int
}

func GoByteSlice(data unsafe.Pointer, size C.size_t) []byte {
	var sliceHeader reflect.SliceHeader
	sliceHeader.Data = uintptr(data)
	sliceHeader.Len = int(size)
	sliceHeader.Cap = int(size)
	return *(*[]byte)(unsafe.Pointer(&sliceHeader))
}

//export getTxCtx
func getTxCtx(pResult unsafe.Pointer, ctxIdx uintptr) {
	env := getEnv(ctxIdx)
	txCtx := (*C.struct_evm_tx_context)(pResult)
	txCtx.tx_gas_price = BigToEvmc(env.GasPrice)
	txCtx.tx_origin = AddressToEvmc(env.Origin)
	txCtx.block_coinbase = AddressToEvmc(env.Coinbase)
	txCtx.block_number = C.int64_t(env.BlockNumber.Int64())
	txCtx.block_timestamp = C.int64_t(env.Time.Int64())
	txCtx.block_gas_limit = C.int64_t(env.GasLimit.Int64())
	txCtx.block_difficulty = BigToEvmc(env.Difficulty)
}

//export getBlockHash
func getBlockHash(pResult unsafe.Pointer, ctxIdx uintptr, number int64) {
	// Represent the result memory as Go slice of 32 bytes.
	result := GoByteSlice(pResult, 32)
	env := getEnv(ctxIdx)
	b := env.BlockNumber.Int64()
	a := b - 256
	var hash common.Hash
	if number >= a && number < b {
		hash = env.GetHash(uint64(number))
	}
	copy(result, hash[:])
}

//export queryState
func queryState(pResult unsafe.Pointer, ctxIdx uintptr, key int32, pAddr unsafe.Pointer, pArg unsafe.Pointer) {
	// Represent the result memory as Go slice of 32 bytes.
	result := GoByteSlice(pResult, 32)
	// Or as pointer to int64.
	pInt64Result := (*int64)(pResult)

	// Get the execution context.
	env := getEnv(ctxIdx)

	var addr common.Address
	copy(addr[:], GoByteSlice(pAddr, 20))

	switch key {
	case C.EVM_SLOAD:
		arg := *(*[32]byte)(pArg)
		val := env.StateDB.GetState(addr, arg)
		copy(result, val[:])
	case C.EVM_CODE_BY_ADDRESS:
		code := env.StateDB.GetCode(addr)
		pResAsMemRef := (*MemoryRef)(pResult)
		pResAsMemRef.ptr = ptr(code)
		pResAsMemRef.len = len(code)
	// fmt.Printf("EXTCODE %x : %d\n", addr, pResAsMemRef.len)
	case C.EVM_CODE_SIZE:
		pInt64Result := (*int64)(pResult)
		*pInt64Result = int64(env.StateDB.GetCodeSize(addr))
	// fmt.Printf("EXTCODESIZE %x : %d\n", addr, *pInt64Result)
	case C.EVM_BALANCE:
		balance := env.StateDB.GetBalance(addr)
		val := common.BigToHash(balance)
		copy(result, val[:])
	case C.EVM_ACCOUNT_EXISTS:
		eip158 := env.ChainConfig().IsEIP158(env.BlockNumber)
		var exist int64
		if eip158 {
			if !env.StateDB.Empty(addr) {
				exist = 1
			}
		} else if env.StateDB.Exist(addr) {
			exist = 1
		}
		*pInt64Result = exist
	default:
		panic(fmt.Sprintf("Unhandled EVM-C query %d\n", key))
	}
}

//export updateState
func updateState(ctxIdx uintptr, key int32, pAddr unsafe.Pointer, pArg1 unsafe.Pointer, pArg2 unsafe.Pointer) {
	env := getEnv(ctxIdx)

	var addr common.Address
	copy(addr[:], GoByteSlice(pAddr, 20))

	switch key {
	case C.EVM_SSTORE:
		key := *(*[32]byte)(pArg1)
		newVal := *(*[32]byte)(pArg2)
		oldVal := env.StateDB.GetState(addr, key)
		env.StateDB.SetState(addr, key, newVal)
		if !common.EmptyHash(oldVal) && common.EmptyHash(newVal) {
			env.StateDB.AddRefund(params.SstoreRefundGas)
		}
	// fmt.Printf("EVMJIT STORE %x : %x [%x, %d]\n", arg1, arg2, ctx.contract.Address(), int(uintptr(pEnv)))
	case C.EVM_LOG:
		dataRef := (*MemoryRef)(pArg1)
		topicsRef := (*MemoryRef)(pArg2)
		data := C.GoBytes(unsafe.Pointer(dataRef.ptr), C.int(dataRef.len))
		// FIXME: Avoid double copy of topics.
		tData := C.GoBytes(unsafe.Pointer(topicsRef.ptr), C.int(topicsRef.len))
		nTopics := topicsRef.len / 32
		topics := make([]common.Hash, nTopics)
		for i := 0; i < nTopics; i++ {
			copy(topics[i][:], tData[i*32:(i+1)*32])
		}
		env.StateDB.AddLog(&types.Log{
			Address: addr,
			Topics:  topics,
			Data:    data,
			BlockNumber: env.BlockNumber.Uint64(),
		})
	case C.EVM_SELFDESTRUCT:
		arg := GoByteSlice(pArg1, 32)
		db := env.StateDB
		if !db.HasSuicided(addr) {
			db.AddRefund(params.SuicideRefundGas)
		}
		balance := db.GetBalance(addr)
		beneficiary := common.BytesToAddress(arg[12:])
		db.AddBalance(beneficiary, balance)
		db.Suicide(addr)
	}
}

//export call
func call(
pCtx unsafe.Pointer,
kind int32,
gas int64,
pAddr unsafe.Pointer,
pValue unsafe.Pointer,
pInput unsafe.Pointer,
inputSize C.size_t,
pOutput unsafe.Pointer,
outputSize C.size_t) int64 {

	ctx := getCtx(uintptr(pCtx))
	address := *(*[20]byte)(pAddr)
	value := (*(*common.Hash)(pValue)).Big()
	input := GoByteSlice(pInput, inputSize)
	output := GoByteSlice(pOutput, outputSize)
	bigGas := new(big.Int).SetInt64(gas)

	// TODO: For some reason C.EVM_CALL_FAILURE "constant" is not visible
	// by go linker. This probably should be reported as cgo bug.
	const callFailureFlag int64 = -(1 << 63);

	switch kind {
	case C.EVM_CALL:
		// fmt.Printf("CALL(gas %d, %x)\n", bigGas, address)
		ctx.contract.Gas.SetInt64(0)
		ret, err := ctx.env.Call(ctx.contract, address, input, bigGas, value)
		gasLeft := ctx.contract.Gas.Int64()
		assert(gasLeft <= gas, fmt.Sprintf("%d <= %d", gasLeft, gas))
		// fmt.Printf("Gas left %d\n", gasLeft)
		if err == nil {
			copy(output, ret)
			assert(gasLeft <= gas, fmt.Sprintf("%d <= %d", gasLeft, gas))
			return gasLeft
		}
		return gasLeft | callFailureFlag
	case C.EVM_CALLCODE:
		// fmt.Printf("CALLCODE(gas %d, %x, value %d)\n", bigGas, address, value)
		ctx.contract.Gas.SetInt64(0)
		ret, err := ctx.env.CallCode(ctx.contract, address, input, bigGas, value)
		gasLeft := ctx.contract.Gas.Int64()
		assert(gasLeft <= gas, fmt.Sprintf("%d <= %d", gasLeft, gas))
		// fmt.Printf("Gas left %d\n", gasLeft)
		if err == nil {
			copy(output, ret)
			return gasLeft
		} else {
			// fmt.Printf("Error: %v\n", err)
		}
		return gasLeft | callFailureFlag
	case C.EVM_DELEGATECALL:
		// fmt.Printf("DELEGATECALL(gas %d, %x)\n", bigGas, address)
		ctx.contract.Gas.SetInt64(0)
		ret, err := ctx.env.DelegateCall(ctx.contract, address, input, bigGas)
		gasLeft := ctx.contract.Gas.Int64()
		assert(gasLeft <= gas, fmt.Sprintf("%d <= %d", gasLeft, gas))
		// fmt.Printf("Gas left %d\n", gasLeft)
		if err == nil {
			copy(output, ret)
			return gasLeft
		}
		return gasLeft | callFailureFlag
	case C.EVM_CREATE:
		// fmt.Printf("DELEGATECALL(gas %d, %x)\n", bigGas, address)
		ctx.contract.Gas.SetInt64(0)
		_, addr, err := ctx.env.Create(ctx.contract, input, bigGas, value)
		gasLeft := ctx.contract.Gas.Int64()
		assert(gasLeft <= gas, fmt.Sprintf("%d <= %d", gasLeft, gas))
		if (ctx.env.ChainConfig().IsHomestead(ctx.env.BlockNumber) && err == ErrCodeStoreOutOfGas) ||
			(err != nil && err != ErrCodeStoreOutOfGas) {
			return callFailureFlag
		} else {
			copy(output, addr[:])
			return gasLeft
		}
	}
	return 0
}

func ptr(bytes []byte) *C.uint8_t {
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&bytes))
	return (*C.uint8_t)(unsafe.Pointer(header.Data))
}

func getMode(env *EVM) C.enum_evm_mode {
	n := env.BlockNumber
	if env.ChainConfig().IsEIP158(n) {
		return C.EVM_CLEARING
	}
	if env.ChainConfig().IsEIP150(n) {
		return C.EVM_ANTI_DOS
	}
	if env.ChainConfig().IsHomestead(n) {
		return C.EVM_HOMESTEAD
	}
	return C.EVM_FRONTIER
}

func (evm *EVMJIT) Run(contract *Contract, input []byte) (ret []byte, err error) {
	evm.env.depth++
	defer func() { evm.env.depth-- }()

	if contract.CodeAddr != nil {
		if p := PrecompiledContracts[*contract.CodeAddr]; p != nil {
			return RunPrecompiledContract(p, input, contract)
		}
	}

	// Don't bother with the execution if there's no code.
	if len(contract.Code) == 0 {
		return nil, nil
	}

	code := contract.Code
	codePtr := (*C.uint8_t)(unsafe.Pointer(&code[0]))
	codeSize := C.size_t(len(code))
	codeHash := HashToEvmc(crypto.Keccak256Hash(code))
	gas := C.int64_t(contract.Gas.Int64())
	inputPtr := ptr(input)
	inputLen := C.size_t(len(input))
	value := BigToEvmc(contract.value)
	mode := getMode(evm.env)
	// fmt.Printf("EVMJIT pre Run (gas %d %d mode: %d, env: %d) %x\n", contract.Gas, gas, mode, env, evm.contract.Address())

	// Create context for this execution.
	ctxIdx := pinCtx(&EVMCContext{contract, evm.env})

	var msg C.struct_evm_message
	msg.address = AddressToEvmc(contract.Address())
	msg.sender = AddressToEvmc(contract.Caller())
	msg.value = value
	msg.input = inputPtr
	msg.input_size = inputLen
	msg.gas = gas
	msg.depth = C.int32_t(evm.env.depth - 1)
	r := C.evm_execute(evm.jit, unsafe.Pointer(ctxIdx), mode, codeHash, codePtr, codeSize, msg)

	unpinCtx(ctxIdx)

	// fmt.Printf("EVMJIT Run %d %d %x\n", r.code, r.gas_left, evm.contract.Address())
	if r.gas_left > gas {
		panic("OOPS")
	}
	contract.Gas.SetInt64(int64(r.gas_left))
	// fmt.Printf("Gas left: %d\n", contract.Gas)
	output := C.GoBytes(unsafe.Pointer(r.output_data), C.int(r.output_size))

	if r.code != 0 {
		// EVMJIT does not informs about the kind of the EVM expection.
		err = ErrOutOfGas
	}

	C.evm_release_result(&r)
	return output, err
}

