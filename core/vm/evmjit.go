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

#include <evmc/evmc.h>
#include <stdlib.h>
#include <dlfcn.h>


static const struct evmc_context_fn_table* get_context_fn_table()
{
	int account_exists(void*, void*);
	void get_storage(struct evmc_uint256be*, void*, void*, struct evmc_uint256be*);
	void set_storage(void*, void*, void*, void*);
	void get_balance(void*, void*, void*);
	size_t get_code_size(void*, void*);
	size_t copy_code(void*, void*, size_t, uint8_t*, size_t);
	void selfdestruct(void*, void*, void*);
	void call(struct evmc_result*, void*, struct evmc_message*);
	void getTxCtx(void*, void*);
	void getBlockHash(void*, void*, long long);
	void emit_log(void*, void*, void*, size_t, void*, size_t);

	static const struct evmc_context_fn_table fn_table = {
		(evmc_account_exists_fn) account_exists,
		(evmc_get_storage_fn) get_storage,
		(evmc_set_storage_fn) set_storage,
		(evmc_get_balance_fn) get_balance,
		(evmc_get_code_size_fn) get_code_size,
		(evmc_copy_code_fn) copy_code,
		(evmc_selfdestruct_fn) selfdestruct,
		(evmc_call_fn) call,
		(evmc_get_tx_context_fn) getTxCtx,
		(evmc_get_block_hash_fn) getBlockHash,
		(evmc_emit_log_fn) emit_log
	};
	return &fn_table;
}


static struct evmc_result evmc_execute(
	struct evmc_instance* instance,
	struct evmc_context* context,
	enum evmc_revision rev,
	const struct evmc_message* msg,
	uint8_t const* code,
	size_t code_size
)
{
	return instance->execute(instance, context, rev, msg, code, code_size);
}

static void evmc_release_result(struct evmc_result* result)
{
	result->release(result);
}

static void free_result_output(const struct evmc_result* result)
{
	free((void*)result->output_data);
}

static void add_result_releaser(struct evmc_result* result)
{
	result->release = free_result_output;
}

typedef struct evmc_instance* (*evmc_create_fn)();

static struct evmc_instance* create(void* create_symbol)
{
	evmc_create_fn fn = (evmc_create_fn)create_symbol;
	return fn();
}

#cgo CFLAGS:  -I/home/chfast/Projects/ethereum/cpp-ethereum/evmc/include
#cgo LDFLAGS: -ldl
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
	"github.com/ethereum/go-ethereum/log"
)

type EVMJIT struct {
	jit  *C.struct_evmc_instance
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
	c C.struct_evmc_context
	index int
}

var loadMu sync.Mutex
var createSymbol unsafe.Pointer
var loaded = false

func loadVM(path string) {
	if len(path) == 0 {
		panic("EVMC path not provided, use --vm flag")
	}

	loadMu.Lock()
	defer loadMu.Unlock()

	if loaded {
		return
	}

	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	handle := C.dlopen(cpath, C.RTLD_LAZY)
	if handle == nil {
		panic(fmt.Sprintf("cannot open %s", path))
	}

	centrypoint := C.CString("evmc_create")
	defer C.free(unsafe.Pointer((centrypoint)))

	C.dlerror()
	createSymbol = C.dlsym(handle, centrypoint)
	err := C.dlerror()
	if err != nil {
		panic(fmt.Sprintf("cannot find evmc_create in %s", path))
	}
	loaded = true
	log.Info("EVMC VM loaded", "path", path)
}

func NewJit(env *EVM, cfg Config) *EVMJIT {
	loadVM(cfg.EVMCPath)
	// FIXME: Destroy the instance later.
	// FIXME: Create the instance once.
	return &EVMJIT{C.create(createSymbol), env, nil, false, nil}
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

func HashToEvmc(hash common.Hash) C.struct_evmc_uint256be {
	return C.struct_evmc_uint256be{*(*[32]C.uint8_t)(unsafe.Pointer(&hash[0]))}
}

func AddressToEvmc(addr common.Address) C.struct_evmc_address {
	return C.struct_evmc_address{*(*[20]C.uint8_t)(unsafe.Pointer(&addr[0]))}
}

func BigToEvmc(i *big.Int) C.struct_evmc_uint256be {
	return HashToEvmc(common.BigToHash(i))
}

func assert(cond bool, msg string) {
	if !cond {
		panic(fmt.Sprintf("Assertion failure! %v", msg))
	}
}

func GoByteSlice(data unsafe.Pointer, size C.size_t) []byte {
	var sliceHeader reflect.SliceHeader
	sliceHeader.Data = uintptr(data)
	sliceHeader.Len = int(size)
	sliceHeader.Cap = int(size)
	return *(*[]byte)(unsafe.Pointer(&sliceHeader))
}

func EvmcHashToSlice(uint256 *C.struct_evmc_uint256be) []byte {
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
func get_storage(pResult *C.struct_evmc_uint256be, pCtx unsafe.Pointer, pAddr unsafe.Pointer, pArg *C.struct_evmc_uint256be) {
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
	if oldVal != (common.Hash{}) && newVal == (common.Hash{}) {
		env.StateDB.AddRefund(params.SstoreRefundGas)
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
	// fmt.Printf("BALANCE %x : %v\n", addr, balance)
}

//export get_code_size
func get_code_size(pCtx unsafe.Pointer, pAddr unsafe.Pointer) C.size_t {
	env := getEnv(pCtx)

	var addr common.Address
	copy(addr[:], GoByteSlice(pAddr, 20))
	return C.size_t(env.StateDB.GetCodeSize(addr));
}

//export copy_code
func copy_code(pCtx unsafe.Pointer, pAddr unsafe.Pointer, offset C.size_t, p *C.uint8_t, size C.size_t) C.size_t {
	env := getEnv(pCtx)

	var addr common.Address
	copy(addr[:], GoByteSlice(pAddr, 20))
	code := env.StateDB.GetCode(addr)
	length := C.size_t(len(code))

	if offset >= length {
		return 0
	}

	toCopy := length - offset;
	if toCopy > size {
		toCopy = size
	}

	out := GoByteSlice(unsafe.Pointer(p), size)

	copy(out, code[offset:])
	return toCopy
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
	txCtx := (*C.struct_evmc_tx_context)(pResult)
	txCtx.tx_gas_price = BigToEvmc(env.GasPrice)
	txCtx.tx_origin = AddressToEvmc(env.Origin)
	txCtx.block_coinbase = AddressToEvmc(env.Coinbase)
	txCtx.block_number = C.int64_t(env.BlockNumber.Int64())
	txCtx.block_timestamp = C.int64_t(env.Time.Int64())
	txCtx.block_gas_limit = C.int64_t(env.GasLimit)
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

//export emit_log
func emit_log(pCtx unsafe.Pointer, pAddr unsafe.Pointer, pData unsafe.Pointer, dataSize C.size_t, pTopics unsafe.Pointer, topicsCount C.size_t) {
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
func call(result *C.struct_evmc_result, pCtx unsafe.Pointer, msg *C.struct_evmc_message) {
	env := getEnv(pCtx)
	contract := getContract(pCtx)

	addr := *(*common.Address)(unsafe.Pointer(&msg.destination))
	value := (*(*common.Hash)(unsafe.Pointer(&msg.value))).Big()
	input := GoByteSlice(unsafe.Pointer(msg.input_data), msg.input_size)
	gas := uint64(msg.gas)

	var output []byte
	var gasLeft uint64
	var err error

	switch msg.kind {
	case C.EVMC_CALL:
		staticCall := (msg.flags & C.EVMC_STATIC) != 0
		if staticCall {
			// fmt.Printf("STATICCALL(gas %d, %x)\n", gas, addr)
			output, gasLeft, err = env.StaticCall(contract, addr, input, gas)
		} else {
			// fmt.Printf("CALL(gas %d, %x)\n", gas, addr)
			output, gasLeft, err = env.Call(contract, addr, input, gas, value)
		}


	case C.EVMC_CALLCODE:
		// fmt.Printf("CALLCODE(gas %d, %x, value %d)\n", gas, addr, value)
		output, gasLeft, err = env.CallCode(contract, addr, input, gas, value)

	case C.EVMC_DELEGATECALL:
		// fmt.Printf("DELEGATECALL(gas %d, %x)\n", gas, addr)
		output, gasLeft, err = env.DelegateCall(contract, addr, input, gas)

	case C.EVMC_CREATE:
		// fmt.Printf("CREATE(gas %d, %x)\n", gas, addr)
		var createAddr common.Address
		var createOutput []byte
		createOutput, createAddr, gasLeft, err = env.Create(contract, input, gas, value)
		isHomestead := env.ChainConfig().IsHomestead(env.BlockNumber)
		if !isHomestead && err == ErrCodeStoreOutOfGas {
			err = nil
		}
		if err == nil {
			// Copy create address to result.
			ca := GoByteSlice(unsafe.Pointer(&result.create_address.bytes), 20)
			copy(ca, createAddr[:])
		} else if err == errExecutionReverted {
			// Assign return buffer from REVERT.
			// TODO: Bad API design: return data buffer and the code is returned in the same place. In worst case
			//       the code is returned also when there is not enough funds to deploy the code.
			output = createOutput
		}
	}

	assert(gasLeft <= gas, fmt.Sprintf("%d <= %d", gasLeft, gas))
	//fmt.Printf("Gas left %d, err: %s, output: %x\n", gasLeft, err, output)
	result.gas_left = C.int64_t(gasLeft)

	// Map error to status code.
	if err == nil {
		result.status_code = C.EVMC_SUCCESS
	} else if err == errExecutionReverted {
		result.status_code = C.EVMC_REVERT
	} else {
		result.status_code = C.EVMC_FAILURE
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
}

func ptr(bytes []byte) *C.uint8_t {
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&bytes))
	return (*C.uint8_t)(unsafe.Pointer(header.Data))
}

func getRevision(env *EVM) C.enum_evmc_revision {
	n := env.BlockNumber
	if env.ChainConfig().IsByzantium(n) {
		return C.EVMC_BYZANTIUM
	}
	if env.ChainConfig().IsEIP158(n) {
		return C.EVMC_SPURIOUS_DRAGON
	}
	if env.ChainConfig().IsEIP150(n) {
		return C.EVMC_TANGERINE_WHISTLE
	}
	if env.ChainConfig().IsHomestead(n) {
		return C.EVMC_HOMESTEAD
	}
	return C.EVMC_FRONTIER
}

func (evm *EVMJIT) Run(contract *Contract, input []byte) (ret []byte, err error) {
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

	var msg C.struct_evmc_message
	msg.destination = AddressToEvmc(contract.Address())
	msg.sender = AddressToEvmc(contract.Caller())
	msg.value = BigToEvmc(contract.value)
	msg.gas = gas
	msg.depth = C.int32_t(evm.env.depth - 1)
	codeHash := crypto.Keccak256Hash(code)
	msg.code_hash = HashToEvmc(codeHash)
	if evm.readOnly {
		msg.flags = C.EVMC_STATIC
	} else {
		msg.flags = 0
	}

	if len(input) > 0 {
		cInput := C.CBytes(input)
		msg.input_data = (*C.uint8_t)(cInput)
		msg.input_size = C.size_t(len(input))
		defer C.free(cInput)
	} else {
		msg.input_data = nil
		msg.input_size = 0
	}

	// fmt.Printf("EVMJIT pre Run (gas %d %d mode: %d, env: %d) %x %x\n", contract.Gas, gas, rev, wrapper.index, codeHash, contract.Address())

	r := C.evmc_execute(evm.jit, &wrapper.c, rev, &msg, codePtr, codeSize)

	unpinCtx(wrapper.index)

	// fmt.Printf("EVMJIT Run [%d]: %d %d %x\n", evm.env.depth - 1, r.status_code, r.gas_left, contract.Address())
	if r.gas_left > gas {
		panic(fmt.Sprintf("gas left: %d, gas: %d, status: %d", r.gas_left, gas, r.status_code))
	}
	contract.Gas = uint64(r.gas_left)
	// fmt.Printf("Gas left: %d\n", contract.Gas)
	output := C.GoBytes(unsafe.Pointer(r.output_data), C.int(r.output_size))

	if r.status_code == C.EVMC_REVERT {
		err = errExecutionReverted
	} else if r.status_code != C.EVMC_SUCCESS {
		// EVMJIT does not informs about the kind of the EVM expection.
		err = ErrOutOfGas
	}

	if r.release != nil {
		// fmt.Printf("Releasing result with %p\n", r.release)
		C.evmc_release_result(&r)
	}

	return output, err
}
