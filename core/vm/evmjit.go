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


static const struct evm_context_fn_table* get_context_fn_table()
{
	int account_exists(void*, void*);
	void get_storage(struct evm_uint256be*, void*, void*, struct evm_uint256be*);
	void set_storage(void*, void*, void*, void*);
	void get_balance(void*, void*, void*);
	size_t get_code(unsigned char**, void*, void*);
	void selfdestruct(void*, void*, void*);
	void getTxCtx(void*, void*);
	void getBlockHash(void*, void*, long long);
	void set_logs(void*, void*, void*, size_t, void*, size_t);

	static const struct evm_context_fn_table fn_table = {
		(evm_account_exists_fn) account_exists,
		(evm_get_storage_fn) get_storage,
		(evm_set_storage_fn) set_storage,
		(evm_get_balance_fn) get_balance,
		(evm_get_code_fn) get_code,
		(evm_selfdestruct_fn) selfdestruct,
		NULL,
		(evm_get_tx_context_fn) getTxCtx,
		(evm_get_block_hash_fn) getBlockHash,
		(evm_log_fn) set_logs
	};
	return &fn_table;
}


static struct evm_result evm_execute(
	struct evm_instance* instance,
	struct evm_context* context,
	enum evm_revision rev,
	const struct evm_message* msg,
	uint8_t const* code,
	size_t code_size
)
{
	return instance->execute(instance, context, rev, msg, code, code_size);
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

type ContextWrapper struct {
	c C.struct_evm_context
	index int
}

func NewJit(env *EVM, cfg Config) *EVMJIT {
	// FIXME: Destroy the jit later.
	return &EVMJIT{C.evmjit_create(), env}
}


var contextMap = make(map[int]*EVMCContext)
var contextMapMu sync.Mutex

func pinCtx(ctx *EVMCContext) int {
	contextMapMu.Lock()

	// Find empty slot in the map starting from the map length.
	id := len(contextMap)
	for contextMap[id] != nil {
		id++
	}
	contextMap[id] = ctx
	contextMapMu.Unlock()
	return id
}

func unpinCtx(id int) {
	contextMapMu.Lock()
	delete(contextMap, id)
	contextMapMu.Unlock()
}

func getCtx(idx int) *EVMCContext {
	contextMapMu.Lock()
	defer contextMapMu.Unlock()
	return contextMap[idx]
}

func getEnv(pCtx unsafe.Pointer) *EVM {
	ctxWrapper := (*ContextWrapper)(pCtx)
	return getCtx(ctxWrapper.index).env
}

func HashToEvmc(hash common.Hash) C.struct_evm_uint256be {
	return C.struct_evm_uint256be{*(*[32]C.uint8_t)(unsafe.Pointer(&hash[0]))}
}

func AddressToEvmc(addr common.Address) C.struct_evm_address {
	return C.struct_evm_address{*(*[20]C.uint8_t)(unsafe.Pointer(&addr[0]))}
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

func EvmcHashToSlice(uint256 *C.struct_evm_uint256be) []byte {
	return GoByteSlice(unsafe.Pointer(uint256), 32)
}

//export account_exists
func account_exists(pCtx unsafe.Pointer, pAddr unsafe.Pointer) C.int {
	// Get the execution context.
	env := getEnv(pCtx)

	arg := GoByteSlice(pAddr, 20)
	var addr common.Address
	copy(addr[:], arg[:])
	eip158 := env.ChainConfig().IsEIP158(env.BlockNumber)
	var exist C.int
	if eip158 {
		if !env.StateDB.Empty(addr) {
			exist = 1
		}
	} else if env.StateDB.Exist(addr) {
		exist = 1
	}
	// fmt.Printf("EXISTS? %x : %v\n", addr, exist)
	return exist
}

//export get_storage
func get_storage(pResult *C.struct_evm_uint256be, pCtx unsafe.Pointer, pAddr unsafe.Pointer, pArg *C.struct_evm_uint256be) {
	result := EvmcHashToSlice(pResult)
	env := getEnv(pCtx)

	var addr common.Address
	copy(addr[:], GoByteSlice(pAddr, 20))

	arg := *(*[32]byte)(unsafe.Pointer(pArg))
	val := env.StateDB.GetState(addr, arg)
	copy(result, val[:])
}

//export set_storage
func set_storage(pCtx unsafe.Pointer, pAddr unsafe.Pointer, pArg1 unsafe.Pointer, pArg2 unsafe.Pointer) {
	env := getEnv(pCtx)

	var addr common.Address
	copy(addr[:], GoByteSlice(pAddr, 20))

	key := *(*[32]byte)(pArg1)
	newVal := *(*[32]byte)(pArg2)
	oldVal := env.StateDB.GetState(addr, key)
	env.StateDB.SetState(addr, key, newVal)
	if !common.EmptyHash(oldVal) && common.EmptyHash(newVal) {
		env.StateDB.AddRefund(params.SstoreRefundGas)
	}
	// fmt.Printf("EVMJIT STORE %x : %x [%x, %d]\n", arg1, arg2, ctx.contract.Address(), int(uintptr(pEnv)))
}

//export get_balance
func get_balance(pResult unsafe.Pointer, pCtx unsafe.Pointer, pAddr unsafe.Pointer) {
	result := GoByteSlice(pResult, 32)
	env := getEnv(pCtx)

	var addr common.Address
	copy(addr[:], GoByteSlice(pAddr, 20))
	balance := env.StateDB.GetBalance(addr)
	val := common.BigToHash(balance)
	copy(result, val[:])
}

//export get_code
func get_code(ppCode **C.uint8_t, pCtx unsafe.Pointer, pAddr unsafe.Pointer) C.size_t {
	env := getEnv(pCtx)

	var addr common.Address
	copy(addr[:], GoByteSlice(pAddr, 20))
	if ppCode != nil {
		code := env.StateDB.GetCode(addr)
		*ppCode = ptr(code)

		// fmt.Printf("EXTCODE %x : %d\n", addr, pResAsMemRef.len)
		return C.size_t(len(code))
	} else {
		return C.size_t(env.StateDB.GetCodeSize(addr))
	}
}

//export selfdestruct
func selfdestruct(pCtx unsafe.Pointer, pAddr unsafe.Pointer, pArg unsafe.Pointer) {
	env := getEnv(pCtx)

	var addr common.Address
	copy(addr[:], GoByteSlice(pAddr, 20))

	var beneficiary common.Address
	copy(beneficiary[:], GoByteSlice(pArg, 20))

	db := env.StateDB
	if !db.HasSuicided(addr) {
		db.AddRefund(params.SuicideRefundGas)
	}
	balance := db.GetBalance(addr)
	db.AddBalance(beneficiary, balance)
	db.Suicide(addr)
}

//export getTxCtx
func getTxCtx(pResult unsafe.Pointer, pCtx unsafe.Pointer) {
	env := getEnv(pCtx)
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
func getBlockHash(pResult unsafe.Pointer, pCtx unsafe.Pointer, number int64) {
	// Represent the result memory as Go slice of 32 bytes.
	result := GoByteSlice(pResult, 32)
	env := getEnv(pCtx)
	b := env.BlockNumber.Int64()
	a := b - 256
	var hash common.Hash
	if number >= a && number < b {
		hash = env.GetHash(uint64(number))
	}
	copy(result, hash[:])
}

//export set_logs
func set_logs(pCtx unsafe.Pointer, pAddr unsafe.Pointer, pData unsafe.Pointer, dataSize C.size_t, pTopics unsafe.Pointer, topicsCount C.size_t) {
	env := getEnv(pCtx)

	var addr common.Address
	copy(addr[:], GoByteSlice(pAddr, 20))

	data := C.GoBytes(pData, C.int(dataSize))
	tData := C.GoBytes(pTopics, C.int(topicsCount * 32))

	nTopics := int(topicsCount)
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

	ctxWrapper := (*ContextWrapper)(pCtx)
	ctx := getCtx(ctxWrapper.index)
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

func getRevision(env *EVM) C.enum_evm_revision {
	n := env.BlockNumber
	if env.ChainConfig().IsEIP158(n) {
		return C.EVM_SPURIOUS_DRAGON
	}
	if env.ChainConfig().IsEIP150(n) {
		return C.EVM_TANGERINE_WHISTLE
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
	gas := C.int64_t(contract.Gas.Int64())
	rev := getRevision(evm.env)
	// fmt.Printf("EVMJIT pre Run (gas %d %d mode: %d, env: %d) %x\n", contract.Gas, gas, mode, env, evm.contract.Address())

	// Create context for this execution.
	wrapper := ContextWrapper{}
	wrapper.c.fn_table = C.get_context_fn_table()
	wrapper.index = pinCtx(&EVMCContext{contract, evm.env})

	var msg C.struct_evm_message
	msg.address = AddressToEvmc(contract.Address())
	msg.sender = AddressToEvmc(contract.Caller())
	msg.value = BigToEvmc(contract.value)
	msg.input = ptr(input)
	msg.input_size = C.size_t(len(input))
	msg.gas = gas
	msg.depth = C.int32_t(evm.env.depth - 1)
	msg.code_hash = HashToEvmc(crypto.Keccak256Hash(code))

	r := C.evm_execute(evm.jit, &wrapper.c, rev, &msg, codePtr, codeSize)

	unpinCtx(wrapper.index)

	// fmt.Printf("EVMJIT Run %d %d %x\n", r.code, r.gas_left, evm.contract.Address())
	if r.gas_left > gas {
		panic("OOPS")
	}
	contract.Gas.SetInt64(int64(r.gas_left))
	// fmt.Printf("Gas left: %d\n", contract.Gas)
	output := C.GoBytes(unsafe.Pointer(r.output_data), C.int(r.output_size))

	if r.status_code != 0 {
		// EVMJIT does not informs about the kind of the EVM expection.
		err = ErrOutOfGas
	}

	if r.release != nil {
		C.evm_release_result(&r)
	}

	return output, err
}
