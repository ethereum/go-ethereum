//go:build evmone && cgo

package vm

/*
#cgo CFLAGS: -I${SRCDIR}/../../evmone/evmc/include
#cgo !mipsle LDFLAGS: -L${SRCDIR}/../../evmone/build/lib -L${SRCDIR}/../../evmone/build/lib/evmone_precompiles -L${SRCDIR}/../../evmone/build/deps/src/blst -levmone -levmone_precompiles -lblst -lstdc++ -lm
#cgo mipsle LDFLAGS: -L${SRCDIR}/../../evmone/build-mipsle/lib -L${SRCDIR}/../../evmone/build-mipsle/lib/evmone_precompiles -L${SRCDIR}/../../evmone/build-mipsle/deps/src/blst -levmone -levmone_precompiles -lblst -lstdc++ -lm

#include <evmc/evmc.h>
#include <stdlib.h>
#include <string.h>

// Forward declarations for Go-exported host callbacks.
extern bool     goAccountExists(uintptr_t handle, const evmc_address* addr);
extern evmc_bytes32 goGetStorage(uintptr_t handle, const evmc_address* addr, const evmc_bytes32* key);
extern enum evmc_storage_status goSetStorage(uintptr_t handle, const evmc_address* addr, const evmc_bytes32* key, const evmc_bytes32* value);
extern evmc_uint256be goGetBalance(uintptr_t handle, const evmc_address* addr);
extern size_t   goGetCodeSize(uintptr_t handle, const evmc_address* addr);
extern evmc_bytes32 goGetCodeHash(uintptr_t handle, const evmc_address* addr);
extern size_t   goCopyCode(uintptr_t handle, const evmc_address* addr, size_t code_offset, uint8_t* buffer_data, size_t buffer_size);
extern bool     goSelfdestruct(uintptr_t handle, const evmc_address* addr, const evmc_address* beneficiary);
extern struct evmc_result goCall(uintptr_t handle, const struct evmc_message* msg);
extern struct evmc_tx_context goGetTxContext(uintptr_t handle);
extern evmc_bytes32 goGetBlockHash(uintptr_t handle, int64_t number);
extern void     goEmitLog(uintptr_t handle, const evmc_address* addr, const uint8_t* data, size_t data_size, const evmc_bytes32 topics[], size_t topics_count);
extern enum evmc_access_status goAccessAccount(uintptr_t handle, const evmc_address* addr);
extern enum evmc_access_status goAccessStorage(uintptr_t handle, const evmc_address* addr, const evmc_bytes32* key);
extern evmc_bytes32 goGetTransientStorage(uintptr_t handle, const evmc_address* addr, const evmc_bytes32* key);
extern void     goSetTransientStorage(uintptr_t handle, const evmc_address* addr, const evmc_bytes32* key, const evmc_bytes32* value);

// C wrapper functions that bridge EVMC host interface to Go.
// These are needed because CGo cannot directly use Go function pointers as C callbacks.

// The handle is stored as the evmc_host_context* pointer value (cast to uintptr_t on Go side).

static bool c_account_exists(struct evmc_host_context* ctx, const evmc_address* addr) {
    return goAccountExists((uintptr_t)ctx, addr);
}
static evmc_bytes32 c_get_storage(struct evmc_host_context* ctx, const evmc_address* addr, const evmc_bytes32* key) {
    return goGetStorage((uintptr_t)ctx, addr, key);
}
static enum evmc_storage_status c_set_storage(struct evmc_host_context* ctx, const evmc_address* addr, const evmc_bytes32* key, const evmc_bytes32* value) {
    return goSetStorage((uintptr_t)ctx, addr, key, value);
}
static evmc_uint256be c_get_balance(struct evmc_host_context* ctx, const evmc_address* addr) {
    return goGetBalance((uintptr_t)ctx, addr);
}
static size_t c_get_code_size(struct evmc_host_context* ctx, const evmc_address* addr) {
    return goGetCodeSize((uintptr_t)ctx, addr);
}
static evmc_bytes32 c_get_code_hash(struct evmc_host_context* ctx, const evmc_address* addr) {
    return goGetCodeHash((uintptr_t)ctx, addr);
}
static size_t c_copy_code(struct evmc_host_context* ctx, const evmc_address* addr, size_t code_offset, uint8_t* buffer_data, size_t buffer_size) {
    return goCopyCode((uintptr_t)ctx, addr, code_offset, buffer_data, buffer_size);
}
static bool c_selfdestruct(struct evmc_host_context* ctx, const evmc_address* addr, const evmc_address* beneficiary) {
    return goSelfdestruct((uintptr_t)ctx, addr, beneficiary);
}
static struct evmc_result c_call(struct evmc_host_context* ctx, const struct evmc_message* msg) {
    return goCall((uintptr_t)ctx, msg);
}
static struct evmc_tx_context c_get_tx_context(struct evmc_host_context* ctx) {
    return goGetTxContext((uintptr_t)ctx);
}
static evmc_bytes32 c_get_block_hash(struct evmc_host_context* ctx, int64_t number) {
    return goGetBlockHash((uintptr_t)ctx, number);
}
static void c_emit_log(struct evmc_host_context* ctx, const evmc_address* addr, const uint8_t* data, size_t data_size, const evmc_bytes32 topics[], size_t topics_count) {
    goEmitLog((uintptr_t)ctx, addr, data, data_size, topics, topics_count);
}
static enum evmc_access_status c_access_account(struct evmc_host_context* ctx, const evmc_address* addr) {
    return goAccessAccount((uintptr_t)ctx, addr);
}
static enum evmc_access_status c_access_storage(struct evmc_host_context* ctx, const evmc_address* addr, const evmc_bytes32* key) {
    return goAccessStorage((uintptr_t)ctx, addr, key);
}
static evmc_bytes32 c_get_transient_storage(struct evmc_host_context* ctx, const evmc_address* addr, const evmc_bytes32* key) {
    return goGetTransientStorage((uintptr_t)ctx, addr, key);
}
static void c_set_transient_storage(struct evmc_host_context* ctx, const evmc_address* addr, const evmc_bytes32* key, const evmc_bytes32* value) {
    goSetTransientStorage((uintptr_t)ctx, addr, key, value);
}

// The singleton host interface.
static const struct evmc_host_interface go_host = {
    .account_exists       = c_account_exists,
    .get_storage          = c_get_storage,
    .set_storage          = c_set_storage,
    .get_balance          = c_get_balance,
    .get_code_size        = c_get_code_size,
    .get_code_hash        = c_get_code_hash,
    .copy_code            = c_copy_code,
    .selfdestruct         = c_selfdestruct,
    .call                 = c_call,
    .get_tx_context       = c_get_tx_context,
    .get_block_hash       = c_get_block_hash,
    .emit_log             = c_emit_log,
    .access_account       = c_access_account,
    .access_storage       = c_access_storage,
    .get_transient_storage = c_get_transient_storage,
    .set_transient_storage = c_set_transient_storage,
};

// evmc_create_evmone is provided by libevmone.a.
extern struct evmc_vm* evmc_create_evmone(void);

// execute_evmone calls evmc_execute on the given VM with our host interface.
static struct evmc_result execute_evmone(
    struct evmc_vm* vm,
    uintptr_t handle,
    enum evmc_revision rev,
    int64_t gas,
    const evmc_address* recipient,
    const evmc_address* sender,
    const uint8_t* input_data,
    size_t input_size,
    const evmc_uint256be* value,
    const uint8_t* code,
    size_t code_size,
    int32_t depth,
    uint32_t flags
) {
    struct evmc_message msg;
    memset(&msg, 0, sizeof(msg));
    msg.kind = EVMC_CALL;
    msg.flags = flags;
    msg.depth = depth;
    msg.gas = gas;
    msg.recipient = *recipient;
    msg.sender = *sender;
    msg.input_data = input_data;
    msg.input_size = input_size;
    msg.value = *value;

    return vm->execute(vm, &go_host, (struct evmc_host_context*)(void*)handle, rev, &msg, code, code_size);
}

// create_vm creates the evmone VM instance.
static struct evmc_vm* create_vm(void) {
    return evmc_create_evmone();
}

// release_result calls the release function pointer on an evmc_result.
static void release_result(struct evmc_result* result) {
    if (result->release) {
        result->release(result);
    }
}

// free_result_output frees the output data of an evmc_result (used as release callback).
void free_result_output(const struct evmc_result* result) {
    free((void*)result->output_data);
}
*/
import "C"

import (
	"runtime/cgo"
	"sync"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

var (
	evmoneVM   *C.struct_evmc_vm
	evmoneOnce sync.Once
)

// initEvmone creates the singleton evmone VM instance.
func initEvmone() {
	evmoneOnce.Do(func() {
		evmoneVM = C.create_vm()
		if evmoneVM == nil {
			panic("evmone: failed to create VM instance")
		}
	})
}

// evmcHostContext wraps the EVM and contract for use in EVMC host callbacks.
type evmcHostContext struct {
	evm      *EVM
	contract *Contract
}

// pinHostContext creates a cgo.Handle for the host context, returning
// the handle value. The caller must call handle.Delete() when done.
func pinHostContext(ctx *evmcHostContext) cgo.Handle {
	return cgo.NewHandle(ctx)
}

// hostContextFromHandle recovers the evmcHostContext from a cgo.Handle value.
func hostContextFromHandle(h uintptr) *evmcHostContext {
	return cgo.Handle(h).Value().(*evmcHostContext)
}

// Type conversion helpers between Go types and EVMC C types.

func goAddress(addr *C.evmc_address) common.Address {
	var a common.Address
	copy(a[:], C.GoBytes(unsafe.Pointer(&addr.bytes[0]), 20))
	return a
}

func goHash(h *C.evmc_bytes32) common.Hash {
	var hash common.Hash
	copy(hash[:], C.GoBytes(unsafe.Pointer(&h.bytes[0]), 32))
	return hash
}

func evmcAddress(addr common.Address) C.evmc_address {
	var a C.evmc_address
	for i := 0; i < 20; i++ {
		a.bytes[i] = C.uint8_t(addr[i])
	}
	return a
}

func evmcHash(h common.Hash) C.evmc_bytes32 {
	var hash C.evmc_bytes32
	for i := 0; i < 32; i++ {
		hash.bytes[i] = C.uint8_t(h[i])
	}
	return hash
}

// evmcUint256 converts a uint256.Int (little-endian limbs) to EVMC big-endian bytes32.
func evmcUint256(v *uint256.Int) C.evmc_uint256be {
	b32 := v.Bytes32() // big-endian [32]byte
	var out C.evmc_uint256be
	for i := 0; i < 32; i++ {
		out.bytes[i] = C.uint8_t(b32[i])
	}
	return out
}

// evmcExecuteResult holds the results from an evmone execution in Go-native types.
type evmcExecuteResult struct {
	statusCode int32
	gasLeft    int64
	gasRefund  int64
	output     []byte
}

// executeEvmone calls the C execute_evmone function and returns the result.
// This must be in the same file as the C preamble that defines execute_evmone.
func executeEvmone(
	handle cgo.Handle,
	rev int32,
	gas int64,
	recipient common.Address,
	sender common.Address,
	input []byte,
	code []byte,
	value *uint256.Int,
	depth int32,
	readOnly bool,
) evmcExecuteResult {
	// Allocate all parameter structs in C memory to avoid cgo pointer violations.
	// Go 1.21+ strictly checks that no Go pointers are passed to or returned from C.
	cRecipient := (*C.evmc_address)(C.malloc(C.size_t(unsafe.Sizeof(C.evmc_address{}))))
	defer C.free(unsafe.Pointer(cRecipient))
	*cRecipient = evmcAddress(recipient)

	cSender := (*C.evmc_address)(C.malloc(C.size_t(unsafe.Sizeof(C.evmc_address{}))))
	defer C.free(unsafe.Pointer(cSender))
	*cSender = evmcAddress(sender)

	cValue := (*C.evmc_uint256be)(C.malloc(C.size_t(unsafe.Sizeof(C.evmc_uint256be{}))))
	defer C.free(unsafe.Pointer(cValue))
	*cValue = evmcUint256(value)

	var inputPtr *C.uint8_t
	var inputSize C.size_t
	if len(input) > 0 {
		cInput := C.CBytes(input)
		defer C.free(cInput)
		inputPtr = (*C.uint8_t)(cInput)
		inputSize = C.size_t(len(input))
	}

	var codePtr *C.uint8_t
	var codeSize C.size_t
	if len(code) > 0 {
		cCode := C.CBytes(code)
		defer C.free(cCode)
		codePtr = (*C.uint8_t)(cCode)
		codeSize = C.size_t(len(code))
	}

	var flags C.uint32_t
	if readOnly {
		flags = C.EVMC_STATIC
	}

	result := C.execute_evmone(
		evmoneVM,
		C.uintptr_t(handle),
		C.enum_evmc_revision(rev),
		C.int64_t(gas),
		cRecipient,
		cSender,
		inputPtr,
		inputSize,
		cValue,
		codePtr,
		codeSize,
		C.int32_t(depth),
		flags,
	)

	var output []byte
	if result.output_data != nil && result.output_size > 0 {
		output = C.GoBytes(unsafe.Pointer(result.output_data), C.int(result.output_size))
	}

	// Free the result's output buffer via the release callback.
	C.release_result(&result)

	return evmcExecuteResult{
		statusCode: int32(result.status_code),
		gasLeft:    int64(result.gas_left),
		gasRefund:  int64(result.gas_refund),
		output:     output,
	}
}
