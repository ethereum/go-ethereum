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
#include <stdlib.h>


static const struct evm_context_fn_table* get_context_fn_table()
{
	int account_exists(void*, void*);
	void get_storage(struct evm_uint256be*, void*, void*, struct evm_uint256be*);
	void set_storage(void*, void*, void*, void*);
	void get_balance(void*, void*, void*);
	size_t get_code(unsigned char**, void*, void*);
	void selfdestruct(void*, void*, void*);
	void call(struct evm_result*, void*, struct evm_message*);
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
		(evm_call_fn) call,
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

static void free_result_output(const struct evm_result* result)
{
	free((void*)result->output_data);
}

static void add_result_releaser(struct evm_result* result)
{
	result->release = free_result_output;
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
	intPool  *intPool
	readOnly   bool
	returnData []byte // Last CALL's return data for subsequent reuse
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
	return &EVMJIT{C.evmjit_create(), env, nil, false, nil}
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

func getContract(pCtx unsafe.Pointer) *Contract {
	ctxWrapper := (*ContextWrapper)(pCtx)
	return getCtx(ctxWrapper.index).contract
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
	// fmt.Printf("SLOAD %x: %x\n", addr, val)
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
		env.StateDB.AddRefund(new(big.Int).SetUint64(params.SstoreRefundGas))
	}
	// fmt.Printf("EVMJIT STORE %x: %x := %x\n", addr, key, newVal)
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
	fmt.Printf("BALANCE %x : %v\n", addr, balance)
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
		db.AddRefund(new(big.Int).SetUint64(params.SuicideRefundGas))
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
func call(result *C.struct_evm_result, pCtx unsafe.Pointer, msg *C.struct_evm_message) {
	env := getEnv(pCtx)
	contract := getContract(pCtx)

	addr := *(*common.Address)(unsafe.Pointer(&msg.address))
	value := (*(*common.Hash)(unsafe.Pointer(&msg.value))).Big()
	input := GoByteSlice(unsafe.Pointer(msg.input), msg.input_size)
	gas := uint64(msg.gas)

	var output []byte
	var gasLeft uint64
	var err error

	switch msg.kind {
	case C.EVM_CALL:
		staticCall := (msg.flags & C.EVM_STATIC) != 0
		if staticCall {
			fmt.Printf("STATICCALL(gas %d, %x)\n", gas, addr)
			output, gasLeft, err = env.StaticCall(contract, addr, input, gas)
		} else {
			fmt.Printf("CALL(gas %d, %x)\n", gas, addr)
			output, gasLeft, err = env.Call(contract, addr, input, gas, value)
		}


	case C.EVM_CALLCODE:
		fmt.Printf("CALLCODE(gas %d, %x, value %d)\n", gas, addr, value)
		output, gasLeft, err = env.CallCode(contract, addr, input, gas, value)

	case C.EVM_DELEGATECALL:
		fmt.Printf("DELEGATECALL(gas %d, %x)\n", gas, addr)
		output, gasLeft, err = env.DelegateCall(contract, addr, input, gas)

	case C.EVM_CREATE:
		fmt.Printf("CREATE(gas %d, %x)\n", gas, addr)
		var createAddr common.Address
		_, createAddr, gasLeft, err = env.Create(contract, input, gas, value)
		isHomestead := env.ChainConfig().IsHomestead(env.BlockNumber)
		if !isHomestead && err == ErrCodeStoreOutOfGas {
			err = nil
		}
		if err == nil {
			// Copy create address to result.
			ca := GoByteSlice(unsafe.Pointer(&result.create_address.bytes), 20)
			copy(ca, createAddr[:])
		}
	}

	assert(gasLeft <= gas, fmt.Sprintf("%d <= %d", gasLeft, gas))
	fmt.Printf("Gas left %d, err: %s\n", gasLeft, err)
	result.gas_left = C.int64_t(gasLeft)

	// Map error to status code.
	if err == nil {
		result.status_code = C.EVM_SUCCESS
	} else {
		result.status_code = C.EVM_FAILURE
	}

	if len(output) > 0 {
		cOutput := C.CBytes(output)
		result.output_data = (*C.uint8_t)(cOutput)
		result.output_size = C.size_t(len(output))
		// We can use result.release = C.release_result_output, but there are
		// some problems with linking. Probably definition of
		// release_result_output() must be in different C file.
		C.add_result_releaser(result)
	} else {
		result.output_data = nil
		result.output_size = 0
		result.release = nil
	}
	fmt.Printf("CALL end\n")
}

func ptr(bytes []byte) *C.uint8_t {
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&bytes))
	return (*C.uint8_t)(unsafe.Pointer(header.Data))
}

func getRevision(env *EVM) C.enum_evm_revision {
	n := env.BlockNumber
	if env.ChainConfig().IsByzantium(n) {
		return C.EVM_BYZANTIUM
	}
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

func (evm *EVMJIT) Run(snapshot int, contract *Contract, input []byte) (ret []byte, err error) {
	evm.env.depth++
	defer func() { evm.env.depth-- }()

	// Don't bother with the execution if there's no code.
	if len(contract.Code) == 0 {
		return nil, nil
	}

	code := contract.Code
	codePtr := (*C.uint8_t)(unsafe.Pointer(&code[0]))
	codeSize := C.size_t(len(code))
	gas := C.int64_t(contract.Gas)
	rev := getRevision(evm.env)

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
	codeHash := crypto.Keccak256Hash(code)
	msg.code_hash = HashToEvmc(codeHash)
	if evm.readOnly {
		msg.flags = C.EVM_STATIC
	} else {
		msg.flags = 0
	}

	// fmt.Printf("EVMJIT pre Run (gas %d %d mode: %d, env: %d) %x %x\n", contract.Gas, gas, rev, wrapper.index, codeHash, contract.Address())

	r := C.evm_execute(evm.jit, &wrapper.c, rev, &msg, codePtr, codeSize)

	unpinCtx(wrapper.index)

	fmt.Printf("EVMJIT Run [%d]: %d %d %x\n", evm.env.depth - 1, r.status_code, r.gas_left, contract.Address())
	if r.gas_left > gas {
		panic("OOPS")
	}
	contract.Gas = uint64(r.gas_left)
	// fmt.Printf("Gas left: %d\n", contract.Gas)
	output := C.GoBytes(unsafe.Pointer(r.output_data), C.int(r.output_size))

	if r.status_code == C.EVM_REVERT {
		evm.env.StateDB.RevertToSnapshot(snapshot)
	} else if r.status_code != C.EVM_SUCCESS {
		// EVMJIT does not informs about the kind of the EVM expection.
		err = ErrOutOfGas
	}

	if r.release != nil {
		// fmt.Printf("Releasing result with %p\n", r.release)
		C.evm_release_result(&r)
	}

	return output, err
}
