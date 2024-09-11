//go:build !ccc

package ccc

import (
	"bytes"
	"math/rand"
	"unsafe"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

type Checker struct {
	ID        uint64
	countdown int
	nextError *error

	skipHash  string
	skipError error
}

// NewChecker creates a new Checker
func NewChecker(lightMode bool) *Checker {
	ccc := &Checker{ID: rand.Uint64()}
	ccc.SetLightMode(lightMode)
	return ccc
}

// Reset resets a ccc, but need to do nothing in mock_ccc.
func (ccc *Checker) Reset() {
}

// ApplyTransaction appends a tx's wrapped BlockTrace into the ccc, and return the accumulated RowConsumption.
// Will only return a dummy value in mock_ccc.
func (ccc *Checker) ApplyTransaction(traces *types.BlockTrace) (*types.RowConsumption, error) {
	if ccc.nextError != nil {
		ccc.countdown--
		if ccc.countdown == 0 {
			err := *ccc.nextError
			ccc.nextError = nil
			return nil, err
		}
	}
	if ccc.skipError != nil {
		if traces.Transactions[0].TxHash == ccc.skipHash {
			return &types.RowConsumption{types.SubCircuitRowUsage{
				Name:      "mock",
				RowNumber: 1_000_001,
			}}, ccc.skipError
		}
	}
	return &types.RowConsumption{types.SubCircuitRowUsage{
		Name:      "mock",
		RowNumber: 1,
	}}, nil
}

func (ccc *Checker) ApplyTransactionRustTrace(rustTrace unsafe.Pointer) (*types.RowConsumption, error) {
	return ccc.ApplyTransaction(goTraces[rustTrace])
}

// ApplyBlock gets a block's RowConsumption.
// Will only return a dummy value in mock_ccc.
func (ccc *Checker) ApplyBlock(traces *types.BlockTrace) (*types.RowConsumption, error) {
	return &types.RowConsumption{types.SubCircuitRowUsage{
		Name:      "mock",
		RowNumber: 2,
	}}, nil
}

// CheckTxNum compares whether the tx_count in ccc match the expected.
// Will alway return true in mock_ccc.
func (ccc *Checker) CheckTxNum(expected int) (bool, uint64, error) {
	return true, uint64(expected), nil
}

// SetLightMode sets to ccc light mode
func (ccc *Checker) SetLightMode(lightMode bool) error {
	return nil
}

// ScheduleError schedules an error for a tx (see `ApplyTransaction`), only used in tests.
func (ccc *Checker) ScheduleError(cnt int, err error) {
	ccc.countdown = cnt
	ccc.nextError = &err
}

// Skip forced CCC to return always an error for a given txn
func (ccc *Checker) Skip(txnHash common.Hash, err error) {
	ccc.skipHash = txnHash.String()
	ccc.skipError = err
}

var goTraces = make(map[unsafe.Pointer]*types.BlockTrace)

func MakeRustTrace(trace *types.BlockTrace, buffer *bytes.Buffer) unsafe.Pointer {
	rustTrace := new(struct{})
	goTraces[unsafe.Pointer(rustTrace)] = trace
	return unsafe.Pointer(rustTrace)
}

func FreeRustTrace(ptr unsafe.Pointer) {
}
