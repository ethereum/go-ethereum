//go:build circuit_capacity_checker

package circuitcapacitychecker

/*
#cgo LDFLAGS: -lm -ldl -lzkp
#include <stdlib.h>
#include "./libzkp/libzkp.h"
*/
import "C" //nolint:typecheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

// mutex for concurrent CircuitCapacityChecker creations
var (
	creationMu  sync.Mutex
	encodeTimer = metrics.NewRegisteredTimer("ccc/encode", nil)
)

func init() {
	C.init()
}

type CircuitCapacityChecker struct {
	// mutex for each CircuitCapacityChecker itself
	sync.Mutex
	ID         uint64
	jsonBuffer bytes.Buffer
}

// NewCircuitCapacityChecker creates a new CircuitCapacityChecker
func NewCircuitCapacityChecker(lightMode bool) *CircuitCapacityChecker {
	creationMu.Lock()
	defer creationMu.Unlock()

	id := C.new_circuit_capacity_checker()
	ccc := &CircuitCapacityChecker{ID: uint64(id)}
	ccc.SetLightMode(lightMode)
	return ccc
}

// Reset resets a CircuitCapacityChecker
func (ccc *CircuitCapacityChecker) Reset() {
	ccc.Lock()
	defer ccc.Unlock()

	C.reset_circuit_capacity_checker(C.uint64_t(ccc.ID))
}

// ApplyTransaction appends a tx's wrapped BlockTrace into the ccc, and return the accumulated RowConsumption
func (ccc *CircuitCapacityChecker) ApplyTransaction(traces *types.BlockTrace) (*types.RowConsumption, error) {
	ccc.Lock()
	defer ccc.Unlock()

	if len(traces.Transactions) != 1 || len(traces.ExecutionResults) != 1 || len(traces.TxStorageTraces) != 1 {
		log.Error("malformatted BlockTrace in ApplyTransaction", "id", ccc.ID,
			"len(traces.Transactions)", len(traces.Transactions),
			"len(traces.ExecutionResults)", len(traces.ExecutionResults),
			"len(traces.TxStorageTraces)", len(traces.TxStorageTraces),
			"err", "length of Transactions, or ExecutionResults, or TxStorageTraces, is not equal to 1")
		return nil, ErrUnknown
	}

	encodeStart := time.Now()
	rustTrace := MakeRustTrace(traces, &ccc.jsonBuffer)
	if rustTrace == nil {
		log.Error("fail to parse json in to rust trace", "id", ccc.ID, "TxHash", traces.Transactions[0].TxHash)
		return nil, ErrUnknown
	}
	encodeTimer.UpdateSince(encodeStart)

	log.Debug("start to check circuit capacity for tx", "id", ccc.ID, "TxHash", traces.Transactions[0].TxHash)
	return ccc.applyTransactionRustTrace(rustTrace)
}

func (ccc *CircuitCapacityChecker) ApplyTransactionRustTrace(rustTrace unsafe.Pointer) (*types.RowConsumption, error) {
	ccc.Lock()
	defer ccc.Unlock()
	return ccc.applyTransactionRustTrace(rustTrace)
}

func (ccc *CircuitCapacityChecker) applyTransactionRustTrace(rustTrace unsafe.Pointer) (*types.RowConsumption, error) {
	rawResult := C.apply_tx(C.uint64_t(ccc.ID), rustTrace)
	defer func() {
		C.free_c_chars(rawResult)
	}()

	result := &WrappedRowUsage{}
	if err := json.Unmarshal([]byte(C.GoString(rawResult)), result); err != nil {
		log.Error("fail to json unmarshal apply_tx result", "id", ccc.ID, "err", err)
		return nil, ErrUnknown
	}

	if result.Error != "" {
		log.Error("fail to apply_tx in CircuitCapacityChecker", "id", ccc.ID, "err", result.Error)
		return nil, ErrUnknown
	}
	if result.AccRowUsage == nil {
		log.Error("fail to apply_tx in CircuitCapacityChecker",
			"id", ccc.ID, "result.AccRowUsage == nil", result.AccRowUsage == nil,
			"err", "AccRowUsage is empty unexpectedly")
		return nil, ErrUnknown
	}
	if !result.AccRowUsage.IsOk {
		return nil, ErrBlockRowConsumptionOverflow
	}
	return (*types.RowConsumption)(&result.AccRowUsage.RowUsageDetails), nil
}

// ApplyBlock gets a block's RowConsumption
func (ccc *CircuitCapacityChecker) ApplyBlock(traces *types.BlockTrace) (*types.RowConsumption, error) {
	ccc.Lock()
	defer ccc.Unlock()

	encodeStart := time.Now()
	rustTrace := MakeRustTrace(traces, &ccc.jsonBuffer)
	if rustTrace == nil {
		log.Error("fail to parse json in to rust trace", "id", ccc.ID, "TxHash", traces.Transactions[0].TxHash)
		return nil, ErrUnknown
	}
	encodeTimer.UpdateSince(encodeStart)

	log.Debug("start to check circuit capacity for block", "id", ccc.ID, "blockNumber", traces.Header.Number, "blockHash", traces.Header.Hash())
	rawResult := C.apply_block(C.uint64_t(ccc.ID), rustTrace)
	defer func() {
		C.free_c_chars(rawResult)
	}()
	log.Debug("check circuit capacity for block done", "id", ccc.ID, "blockNumber", traces.Header.Number, "blockHash", traces.Header.Hash())

	result := &WrappedRowUsage{}
	if err := json.Unmarshal([]byte(C.GoString(rawResult)), result); err != nil {
		log.Error("fail to json unmarshal apply_block result", "id", ccc.ID, "blockNumber", traces.Header.Number, "blockHash", traces.Header.Hash(), "err", err)
		return nil, ErrUnknown
	}

	if result.Error != "" {
		log.Error("fail to apply_block in CircuitCapacityChecker", "id", ccc.ID, "blockNumber", traces.Header.Number, "blockHash", traces.Header.Hash(), "err", result.Error)
		return nil, ErrUnknown
	}
	if result.AccRowUsage == nil {
		log.Error("fail to apply_block in CircuitCapacityChecker", "id", ccc.ID, "blockNumber", traces.Header.Number, "blockHash", traces.Header.Hash(), "err", "AccRowUsage is empty unexpectedly")
		return nil, ErrUnknown
	}
	if !result.AccRowUsage.IsOk {
		return nil, ErrBlockRowConsumptionOverflow
	}
	return (*types.RowConsumption)(&result.AccRowUsage.RowUsageDetails), nil
}

// CheckTxNum compares whether the tx_count in ccc match the expected
func (ccc *CircuitCapacityChecker) CheckTxNum(expected int) (bool, uint64, error) {
	ccc.Lock()
	defer ccc.Unlock()

	log.Debug("ccc get_tx_num start", "id", ccc.ID)
	rawResult := C.get_tx_num(C.uint64_t(ccc.ID))
	defer func() {
		C.free_c_chars(rawResult)
	}()
	log.Debug("ccc get_tx_num end", "id", ccc.ID)

	result := &WrappedTxNum{}
	if err := json.Unmarshal([]byte(C.GoString(rawResult)), result); err != nil {
		return false, 0, fmt.Errorf("fail to json unmarshal get_tx_num result, id: %d, err: %w", ccc.ID, err)
	}
	if result.Error != "" {
		return false, 0, fmt.Errorf("fail to get_tx_num in CircuitCapacityChecker, id: %d, err: %w", ccc.ID, result.Error)
	}

	return result.TxNum == uint64(expected), result.TxNum, nil
}

// SetLightMode sets to ccc light mode
func (ccc *CircuitCapacityChecker) SetLightMode(lightMode bool) error {
	ccc.Lock()
	defer ccc.Unlock()

	log.Debug("ccc set_light_mode start", "id", ccc.ID)
	rawResult := C.set_light_mode(C.uint64_t(ccc.ID), C.bool(lightMode))
	defer func() {
		C.free_c_chars(rawResult)
	}()
	log.Debug("ccc set_light_mode end", "id", ccc.ID)

	result := &WrappedCommonResult{}
	if err := json.Unmarshal([]byte(C.GoString(rawResult)), result); err != nil {
		return fmt.Errorf("fail to json unmarshal set_light_mode result, id: %d, err: %w", ccc.ID, err)
	}
	if result.Error != "" {
		return fmt.Errorf("fail to set_light_mode in CircuitCapacityChecker, id: %d, err: %w", ccc.ID, result.Error)
	}

	return nil
}

func MakeRustTrace(trace *types.BlockTrace, buffer *bytes.Buffer) unsafe.Pointer {
	if buffer == nil {
		buffer = new(bytes.Buffer)
	}
	buffer.Reset()

	err := json.NewEncoder(buffer).Encode(trace)
	if err != nil {
		log.Error("fail to json marshal traces in MakeRustTrace", "err", err)
		return nil
	}

	tracesStr := C.CString(string(buffer.Bytes()))
	defer func() {
		C.free(unsafe.Pointer(tracesStr))
	}()

	return C.parse_json_to_rust_trace(tracesStr)
}

func FreeRustTrace(ptr unsafe.Pointer) {
	C.free_rust_trace(ptr)
}
