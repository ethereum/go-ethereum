// EVMC: Ethereum Client-VM Connector API.
// Copyright 2018 The EVMC Authors.
// Licensed under the Apache License, Version 2.0. See the LICENSE file.

package evmc

/*
#cgo CFLAGS:  -I${SRCDIR}/.. -Wall -Wextra
#cgo !windows LDFLAGS: -ldl

#include <evmc/evmc.h>
#include <evmc/helpers.h>
#include <evmc/loader.h>

#include <stdlib.h>
#include <string.h>

static inline int set_option(struct evmc_instance* instance, char* name, char* value)
{
	int ret = evmc_set_option(instance, name, value);
	free(name);
	free(value);
	return ret;
}

struct extended_context
{
	struct evmc_context context;
	int64_t index;
};

extern const struct evmc_context_fn_table evmc_go_fn_table;

static struct evmc_result execute_wrapper(struct evmc_instance* instance, int64_t context_index, enum evmc_revision rev,
	const struct evmc_address* destination, const struct evmc_address* sender, const struct evmc_uint256be* value,
	const uint8_t* input_data, size_t input_size, const struct evmc_uint256be* code_hash, int64_t gas,
	int32_t depth, enum evmc_call_kind kind, uint32_t flags, const uint8_t* code, size_t code_size)
{
	struct evmc_uint256be create2_salt = {};
	struct evmc_message msg = {
		*destination,
		*sender,
		*value,
		input_data,
		input_size,
		*code_hash,
		create2_salt,
		gas,
		depth,
		kind,
		flags,
	};

	struct extended_context ctx = {{&evmc_go_fn_table}, context_index};
	return evmc_execute(instance, &ctx.context, rev, &msg, code, code_size);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
)

// Static asserts.
const (
	_ = uint(common.HashLength - C.sizeof_struct_evmc_uint256be) // The size of evmc_uint256be equals the size of Hash.
	_ = uint(C.sizeof_struct_evmc_uint256be - common.HashLength)
	_ = uint(common.AddressLength - C.sizeof_struct_evmc_address) // The size of evmc_address equals the size of Address.
	_ = uint(C.sizeof_struct_evmc_address - common.AddressLength)
)

type Error int32

func (err Error) IsInternalError() bool {
	return err < 0
}

func (err Error) Error() string {
	code := C.enum_evmc_status_code(err)

	switch code {
	case C.EVMC_FAILURE:
		return "evmc: failure"
	case C.EVMC_REVERT:
		return "evmc: revert"
	case C.EVMC_OUT_OF_GAS:
		return "evmc: out of gas"
	case C.EVMC_INVALID_INSTRUCTION:
		return "evmc: invalid instruction"
	case C.EVMC_UNDEFINED_INSTRUCTION:
		return "evmc: undefined instruction"
	case C.EVMC_STACK_OVERFLOW:
		return "evmc: stack overflow"
	case C.EVMC_STACK_UNDERFLOW:
		return "evmc: stack underflow"
	case C.EVMC_BAD_JUMP_DESTINATION:
		return "evmc: bad jump destination"
	case C.EVMC_INVALID_MEMORY_ACCESS:
		return "evmc: invalid memory access"
	case C.EVMC_CALL_DEPTH_EXCEEDED:
		return "evmc: call depth exceeded"
	case C.EVMC_STATIC_MODE_VIOLATION:
		return "evmc: static mode violation"
	case C.EVMC_PRECOMPILE_FAILURE:
		return "evmc: precompile failure"
	case C.EVMC_CONTRACT_VALIDATION_FAILURE:
		return "evmc: contract validation failure"
	case C.EVMC_ARGUMENT_OUT_OF_RANGE:
		return "evmc: argument out of range"
	case C.EVMC_WASM_UNREACHABLE_INSTRUCTION:
		return "evmc: the WebAssembly unreachable instruction has been hit during execution"
	case C.EVMC_WASM_TRAP:
		return "evmc: a WebAssembly trap has been hit during execution"
	case C.EVMC_REJECTED:
		return "evmc: rejected"
	}

	if code < 0 {
		return fmt.Sprintf("evmc: internal error (%d)", int32(code))
	}

	return fmt.Sprintf("evmc: unknown non-fatal status code %d", int32(code))
}

const (
	Failure = Error(C.EVMC_FAILURE)
	Revert  = Error(C.EVMC_REVERT)
)

type Revision int32

const (
	Frontier         Revision = C.EVMC_FRONTIER
	Homestead        Revision = C.EVMC_HOMESTEAD
	TangerineWhistle Revision = C.EVMC_TANGERINE_WHISTLE
	SpuriousDragon   Revision = C.EVMC_SPURIOUS_DRAGON
	Byzantium        Revision = C.EVMC_BYZANTIUM
	Constantinople   Revision = C.EVMC_CONSTANTINOPLE
)

type Instance struct {
	handle *C.struct_evmc_instance
}

func Load(filename string) (instance *Instance, err error) {
	cfilename := C.CString(filename)
	var loaderErr C.enum_evmc_loader_error_code
	handle := C.evmc_load_and_create(cfilename, &loaderErr)
	C.free(unsafe.Pointer(cfilename))
	switch loaderErr {
	case C.EVMC_LOADER_SUCCESS:
		instance = &Instance{handle}
	case C.EVMC_LOADER_CANNOT_OPEN:
		err = fmt.Errorf("evmc loader: cannot open %s", filename)
	case C.EVMC_LOADER_SYMBOL_NOT_FOUND:
		err = fmt.Errorf("evmc loader: the EVMC create function not found in %s", filename)
	case C.EVMC_LOADER_INVALID_ARGUMENT:
		panic("evmc loader: filename argument is invalid")
	case C.EVMC_LOADER_INSTANCE_CREATION_FAILURE:
		err = errors.New("evmc loader: VM instance creation failure")
	case C.EVMC_LOADER_ABI_VERSION_MISMATCH:
		err = errors.New("evmc loader: ABI version mismatch")
	default:
		panic(fmt.Sprintf("evmc loader: unexpected error (%d)", int(loaderErr)))
	}
	return instance, err
}

func (instance *Instance) Destroy() {
	C.evmc_destroy(instance.handle)
}

func (instance *Instance) Name() string {
	// TODO: consider using C.evmc_vm_name(instance.handle)
	return C.GoString(instance.handle.name)
}

func (instance *Instance) Version() string {
	// TODO: consider using C.evmc_vm_version(instance.handle)
	return C.GoString(instance.handle.version)
}

func (instance *Instance) SetOption(name string, value string) (err error) {

	r := C.set_option(instance.handle, C.CString(name), C.CString(value))
	if r != 1 {
		err = fmt.Errorf("evmc: option '%s' not accepted", name)
	}
	return err
}

func (instance *Instance) Execute(ctx HostContext, rev Revision,
	destination common.Address, sender common.Address, value common.Hash, input []byte, codeHash common.Hash, gas int64,
	depth int, kind CallKind, static bool, code []byte) (output []byte, gasLeft int64, err error) {

	flags := C.uint32_t(0)
	if static {
		flags |= C.EVMC_STATIC
	}

	ctxId := addHostContext(ctx)
	// FIXME: Clarify passing by pointer vs passing by value.
	evmcDestination := evmcAddress(destination)
	evmcSender := evmcAddress(sender)
	evmcValue := evmcUint256be(value)
	evmcCodeHash := evmcUint256be(codeHash)
	result := C.execute_wrapper(instance.handle, C.int64_t(ctxId), uint32(rev), &evmcDestination, &evmcSender, &evmcValue,
		bytesPtr(input), C.size_t(len(input)), &evmcCodeHash, C.int64_t(gas), C.int32_t(depth), C.enum_evmc_call_kind(kind),
		flags, bytesPtr(code), C.size_t(len(code)))
	removeHostContext(ctxId)

	output = C.GoBytes(unsafe.Pointer(result.output_data), C.int(result.output_size))
	gasLeft = int64(result.gas_left)
	if result.status_code != C.EVMC_SUCCESS {
		err = Error(result.status_code)
	}

	if result.release != nil {
		C.evmc_release_result(&result)
	}

	return output, gasLeft, err
}

var (
	hostContextCounter int
	hostContextMap     = map[int]HostContext{}
	hostContextMapMu   sync.Mutex
)

func addHostContext(ctx HostContext) int {
	hostContextMapMu.Lock()
	id := hostContextCounter
	hostContextCounter++
	hostContextMap[id] = ctx
	hostContextMapMu.Unlock()
	return id
}

func removeHostContext(id int) {
	hostContextMapMu.Lock()
	delete(hostContextMap, id)
	hostContextMapMu.Unlock()
}

func getHostContext(idx int) HostContext {
	hostContextMapMu.Lock()
	ctx := hostContextMap[idx]
	hostContextMapMu.Unlock()
	return ctx
}

func evmcUint256be(in common.Hash) C.struct_evmc_uint256be {
	out := C.struct_evmc_uint256be{}
	for i := 0; i < len(in); i++ {
		out.bytes[i] = C.uint8_t(in[i])
	}
	return out
}

func evmcAddress(address common.Address) C.struct_evmc_address {
	r := C.struct_evmc_address{}
	for i := 0; i < len(address); i++ {
		r.bytes[i] = C.uint8_t(address[i])
	}
	return r
}

func bytesPtr(bytes []byte) *C.uint8_t {
	if len(bytes) == 0 {
		return nil
	}
	return (*C.uint8_t)(unsafe.Pointer(&bytes[0]))
}
