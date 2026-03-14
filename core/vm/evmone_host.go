//go:build evmone && cgo

package vm

/*
#include <evmc/evmc.h>
#include <stdlib.h>
#include <string.h>

// free_result_output is defined in evmone.go's C preamble.
extern void free_result_output(const struct evmc_result* result);
*/
import "C"

import (
	"math/big"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

//export goAccountExists
func goAccountExists(handle C.uintptr_t, addr *C.evmc_address) C.bool {
	ctx := hostContextFromHandle(uintptr(handle))
	return C.bool(ctx.evm.StateDB.Exist(goAddress(addr)))
}

//export goGetStorage
func goGetStorage(handle C.uintptr_t, addr *C.evmc_address, key *C.evmc_bytes32) C.evmc_bytes32 {
	ctx := hostContextFromHandle(uintptr(handle))
	val := ctx.evm.StateDB.GetState(goAddress(addr), goHash(key))
	return evmcHash(val)
}

//export goSetStorage
func goSetStorage(handle C.uintptr_t, addr *C.evmc_address, key *C.evmc_bytes32, value *C.evmc_bytes32) C.enum_evmc_storage_status {
	ctx := hostContextFromHandle(uintptr(handle))
	address := goAddress(addr)
	slot := goHash(key)
	newVal := goHash(value)

	current, original := ctx.evm.StateDB.GetStateAndCommittedState(address, slot)

	ctx.evm.StateDB.SetState(address, slot, newVal)

	// Determine the EVMC storage status based on original, current, and new values.
	// This follows the EIP-2200 / EIP-1283 net gas metering logic.
	// evmone uses this status to calculate the appropriate gas cost internally.
	zeroHash := common.Hash{}

	if current == newVal {
		// No-op or dirty re-assignment: minimal cost (SLOAD_GAS)
		return C.EVMC_STORAGE_ASSIGNED
	}
	if original == current {
		if original == zeroHash {
			// 0 -> 0 -> Z: creating a new slot
			return C.EVMC_STORAGE_ADDED
		}
		if newVal == zeroHash {
			// X -> X -> 0: deleting
			return C.EVMC_STORAGE_DELETED
		}
		// X -> X -> Z: modifying
		return C.EVMC_STORAGE_MODIFIED
	}
	// original != current (dirty slot)
	if original != zeroHash {
		if current == zeroHash {
			if newVal == original {
				// X -> 0 -> X: restoring deleted
				return C.EVMC_STORAGE_DELETED_RESTORED
			}
			// X -> 0 -> Z: re-adding after delete
			return C.EVMC_STORAGE_DELETED_ADDED
		}
		if newVal == zeroHash {
			// X -> Y -> 0: deleting modified
			return C.EVMC_STORAGE_MODIFIED_DELETED
		}
		if newVal == original {
			// X -> Y -> X: restoring modified
			return C.EVMC_STORAGE_MODIFIED_RESTORED
		}
	} else {
		// original == zero, current != zero, newVal != current
		if newVal == zeroHash {
			// 0 -> Y -> 0: deleting added
			return C.EVMC_STORAGE_ADDED_DELETED
		}
	}

	// Catch-all: dirty update
	return C.EVMC_STORAGE_ASSIGNED
}

//export goGetBalance
func goGetBalance(handle C.uintptr_t, addr *C.evmc_address) C.evmc_uint256be {
	ctx := hostContextFromHandle(uintptr(handle))
	balance := ctx.evm.StateDB.GetBalance(goAddress(addr))
	return evmcUint256(balance)
}

//export goGetCodeSize
func goGetCodeSize(handle C.uintptr_t, addr *C.evmc_address) C.size_t {
	ctx := hostContextFromHandle(uintptr(handle))
	return C.size_t(ctx.evm.StateDB.GetCodeSize(goAddress(addr)))
}

//export goGetCodeHash
func goGetCodeHash(handle C.uintptr_t, addr *C.evmc_address) C.evmc_bytes32 {
	ctx := hostContextFromHandle(uintptr(handle))
	hash := ctx.evm.StateDB.GetCodeHash(goAddress(addr))
	return evmcHash(hash)
}

//export goCopyCode
func goCopyCode(handle C.uintptr_t, addr *C.evmc_address, codeOffset C.size_t, bufferData *C.uint8_t, bufferSize C.size_t) C.size_t {
	ctx := hostContextFromHandle(uintptr(handle))
	code := ctx.evm.StateDB.GetCode(goAddress(addr))

	offset := int(codeOffset)
	if offset >= len(code) {
		return 0
	}
	toCopy := len(code) - offset
	if toCopy > int(bufferSize) {
		toCopy = int(bufferSize)
	}
	if toCopy > 0 {
		dst := unsafe.Slice((*byte)(unsafe.Pointer(bufferData)), int(bufferSize))
		copy(dst[:toCopy], code[offset:offset+toCopy])
	}
	return C.size_t(toCopy)
}

//export goSelfdestruct
func goSelfdestruct(handle C.uintptr_t, addr *C.evmc_address, beneficiary *C.evmc_address) C.bool {
	ctx := hostContextFromHandle(uintptr(handle))
	address := goAddress(addr)
	benefAddr := goAddress(beneficiary)

	// Transfer balance to beneficiary
	balance := ctx.evm.StateDB.GetBalance(address)
	if balance.Sign() > 0 {
		ctx.evm.StateDB.SubBalance(address, balance, 0)
		ctx.evm.StateDB.AddBalance(benefAddr, balance, 0)
	}

	// Post-Cancun: use EIP-6780 semantics
	if ctx.evm.chainRules.IsCancun {
		_, destructed := ctx.evm.StateDB.SelfDestruct6780(address)
		return C.bool(destructed)
	}

	hasPreviouslyDestructed := ctx.evm.StateDB.HasSelfDestructed(address)
	ctx.evm.StateDB.SelfDestruct(address)
	return C.bool(!hasPreviouslyDestructed)
}

//export goCall
func goCall(handle C.uintptr_t, msg *C.struct_evmc_message) C.struct_evmc_result {
	ctx := hostContextFromHandle(uintptr(handle))

	kind := msg.kind
	sender := goAddress(&msg.sender)
	recipient := goAddress(&msg.recipient)
	input := C.GoBytes(unsafe.Pointer(msg.input_data), C.int(msg.input_size))
	gas := uint64(msg.gas)
	value := new(uint256.Int)
	value.SetBytes(C.GoBytes(unsafe.Pointer(&msg.value.bytes[0]), 32))

	var (
		ret         []byte
		leftOverGas uint64
		err         error
	)

	switch kind {
	case C.EVMC_CALL:
		if msg.flags&C.EVMC_STATIC != 0 {
			ret, leftOverGas, err = ctx.evm.StaticCall(sender, recipient, input, gas)
		} else {
			ret, leftOverGas, err = ctx.evm.Call(sender, recipient, input, gas, value)
		}

	case C.EVMC_CALLCODE:
		ret, leftOverGas, err = ctx.evm.CallCode(sender, recipient, input, gas, value)

	case C.EVMC_DELEGATECALL:
		// For DELEGATECALL, sender is the original caller (contract.Caller()),
		// recipient is the current contract, and code_address has the code.
		codeAddr := goAddress(&msg.code_address)
		ret, leftOverGas, err = ctx.evm.DelegateCall(sender, recipient, codeAddr, input, gas, value)

	case C.EVMC_CREATE:
		var createAddr common.Address
		ret, createAddr, leftOverGas, err = ctx.evm.Create(sender, input, gas, value)
		return makeEvmcResult(ret, leftOverGas, err, createAddr)

	case C.EVMC_CREATE2:
		salt := new(uint256.Int)
		salt.SetBytes(C.GoBytes(unsafe.Pointer(&msg.create2_salt.bytes[0]), 32))
		var createAddr common.Address
		ret, createAddr, leftOverGas, err = ctx.evm.Create2(sender, input, gas, value, salt)
		return makeEvmcResult(ret, leftOverGas, err, createAddr)
	}

	return makeEvmcResult(ret, leftOverGas, err, common.Address{})
}

// makeEvmcResult constructs an evmc_result from Go execution results.
func makeEvmcResult(output []byte, gasLeft uint64, err error, createAddr common.Address) C.struct_evmc_result {
	var result C.struct_evmc_result

	result.gas_left = C.int64_t(gasLeft)

	if err == nil {
		result.status_code = C.EVMC_SUCCESS
	} else if err == ErrExecutionReverted {
		result.status_code = C.EVMC_REVERT
	} else if err == ErrOutOfGas {
		result.status_code = C.EVMC_OUT_OF_GAS
	} else if err == ErrDepth {
		result.status_code = C.EVMC_CALL_DEPTH_EXCEEDED
	} else if err == ErrInsufficientBalance {
		result.status_code = C.EVMC_INSUFFICIENT_BALANCE
	} else {
		result.status_code = C.EVMC_FAILURE
	}

	if len(output) > 0 {
		// Allocate C memory for output data and set release callback.
		cData := C.malloc(C.size_t(len(output)))
		C.memcpy(cData, unsafe.Pointer(&output[0]), C.size_t(len(output)))
		result.output_data = (*C.uint8_t)(cData)
		result.output_size = C.size_t(len(output))
		result.release = C.evmc_release_result_fn(C.free_result_output)
	}

	if createAddr != (common.Address{}) {
		result.create_address = evmcAddress(createAddr)
	}

	return result
}

//export goGetTxContext
func goGetTxContext(handle C.uintptr_t) C.struct_evmc_tx_context {
	ctx := hostContextFromHandle(uintptr(handle))
	evm := ctx.evm

	var txCtx C.struct_evmc_tx_context

	// Gas price
	if evm.GasPrice != nil {
		gasPrice := uint256.MustFromBig(evm.GasPrice)
		txCtx.tx_gas_price = evmcUint256(gasPrice)
	}

	// Origin
	txCtx.tx_origin = evmcAddress(evm.TxContext.Origin)

	// Block coinbase
	txCtx.block_coinbase = evmcAddress(evm.Context.Coinbase)

	// Block number
	if evm.Context.BlockNumber != nil {
		txCtx.block_number = C.int64_t(evm.Context.BlockNumber.Int64())
	}

	// Block timestamp
	txCtx.block_timestamp = C.int64_t(evm.Context.Time)

	// Block gas limit
	txCtx.block_gas_limit = C.int64_t(evm.Context.GasLimit)

	// PREVRANDAO (post-merge) / DIFFICULTY (pre-merge)
	if evm.Context.Random != nil {
		txCtx.block_prev_randao = evmcHash(*evm.Context.Random)
	} else if evm.Context.Difficulty != nil {
		diff := uint256.MustFromBig(evm.Context.Difficulty)
		txCtx.block_prev_randao = evmcUint256(diff)
	}

	// Chain ID
	if evm.chainConfig != nil && evm.chainConfig.ChainID != nil {
		chainID := uint256.MustFromBig(evm.chainConfig.ChainID)
		txCtx.chain_id = evmcUint256(chainID)
	}

	// Base fee
	if evm.Context.BaseFee != nil {
		baseFee := uint256.MustFromBig(evm.Context.BaseFee)
		txCtx.block_base_fee = evmcUint256(baseFee)
	}

	// Blob base fee
	if evm.Context.BlobBaseFee != nil {
		blobBaseFee := uint256.MustFromBig(evm.Context.BlobBaseFee)
		txCtx.blob_base_fee = evmcUint256(blobBaseFee)
	}

	// Blob hashes — must be allocated in C memory to avoid CGO pointer violation.
	if len(evm.TxContext.BlobHashes) > 0 {
		n := len(evm.TxContext.BlobHashes)
		size := C.size_t(n) * C.size_t(unsafe.Sizeof(C.evmc_bytes32{}))
		cBlobs := (*C.evmc_bytes32)(C.malloc(size))
		blobArr := unsafe.Slice(cBlobs, n)
		for i, h := range evm.TxContext.BlobHashes {
			blobArr[i] = evmcHash(h)
		}
		txCtx.blob_hashes = cBlobs
		txCtx.blob_hashes_count = C.size_t(n)
		// Note: this C memory is leaked per-call. evmone copies the data
		// and doesn't retain the pointer, so a future optimization could
		// pool or free it, but correctness requires C-allocated memory here.
	}

	return txCtx
}

//export goGetBlockHash
func goGetBlockHash(handle C.uintptr_t, number C.int64_t) C.evmc_bytes32 {
	ctx := hostContextFromHandle(uintptr(handle))
	hash := ctx.evm.Context.GetHash(uint64(number))
	return evmcHash(hash)
}

//export goEmitLog
func goEmitLog(handle C.uintptr_t, addr *C.evmc_address, data *C.uint8_t, dataSize C.size_t, topics *C.evmc_bytes32, topicsCount C.size_t) {
	ctx := hostContextFromHandle(uintptr(handle))
	address := goAddress(addr)

	var logData []byte
	if dataSize > 0 {
		logData = C.GoBytes(unsafe.Pointer(data), C.int(dataSize))
	}

	nTopics := int(topicsCount)
	logTopics := make([]common.Hash, nTopics)
	if nTopics > 0 {
		topicSlice := unsafe.Slice(topics, nTopics)
		for i := 0; i < nTopics; i++ {
			logTopics[i] = goHash(&topicSlice[i])
		}
	}

	ctx.evm.StateDB.AddLog(&types.Log{
		Address: address,
		Topics:  logTopics,
		Data:    logData,
		// Block number and tx hash are filled in by the receipt processing.
	})
}

//export goAccessAccount
func goAccessAccount(handle C.uintptr_t, addr *C.evmc_address) C.enum_evmc_access_status {
	ctx := hostContextFromHandle(uintptr(handle))
	address := goAddress(addr)

	warm := ctx.evm.StateDB.AddressInAccessList(address)
	if !warm {
		ctx.evm.StateDB.AddAddressToAccessList(address)
		return C.EVMC_ACCESS_COLD
	}
	return C.EVMC_ACCESS_WARM
}

//export goAccessStorage
func goAccessStorage(handle C.uintptr_t, addr *C.evmc_address, key *C.evmc_bytes32) C.enum_evmc_access_status {
	ctx := hostContextFromHandle(uintptr(handle))
	address := goAddress(addr)
	slot := goHash(key)

	_, slotWarm := ctx.evm.StateDB.SlotInAccessList(address, slot)
	if !slotWarm {
		ctx.evm.StateDB.AddSlotToAccessList(address, slot)
		return C.EVMC_ACCESS_COLD
	}
	return C.EVMC_ACCESS_WARM
}

//export goGetTransientStorage
func goGetTransientStorage(handle C.uintptr_t, addr *C.evmc_address, key *C.evmc_bytes32) C.evmc_bytes32 {
	ctx := hostContextFromHandle(uintptr(handle))
	val := ctx.evm.StateDB.GetTransientState(goAddress(addr), goHash(key))
	return evmcHash(val)
}

//export goSetTransientStorage
func goSetTransientStorage(handle C.uintptr_t, addr *C.evmc_address, key *C.evmc_bytes32, value *C.evmc_bytes32) {
	ctx := hostContextFromHandle(uintptr(handle))
	ctx.evm.StateDB.SetTransientState(goAddress(addr), goHash(key), goHash(value))
}

// uint256FromBig converts a *big.Int to *uint256.Int, returning zero for nil.
func uint256FromBig(b *big.Int) *uint256.Int {
	if b == nil {
		return new(uint256.Int)
	}
	v, _ := uint256.FromBig(b)
	if v == nil {
		return new(uint256.Int)
	}
	return v
}
