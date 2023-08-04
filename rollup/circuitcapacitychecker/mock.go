//go:build !circuit_capacity_checker

package circuitcapacitychecker

import (
	"math/rand"

	"github.com/scroll-tech/go-ethereum/core/types"
)

type CircuitCapacityChecker struct {
	ID uint64
}

func NewCircuitCapacityChecker() *CircuitCapacityChecker {
	return &CircuitCapacityChecker{ID: rand.Uint64()}
}

func (ccc *CircuitCapacityChecker) Reset() {
}

func (ccc *CircuitCapacityChecker) ApplyTransaction(traces *types.BlockTrace) (*types.RowConsumption, error) {
	return &types.RowConsumption{types.SubCircuitRowUsage{
		Name:      "mock",
		RowNumber: 1,
	}}, nil
}

func (ccc *CircuitCapacityChecker) ApplyBlock(traces *types.BlockTrace) (*types.RowConsumption, error) {
	return &types.RowConsumption{types.SubCircuitRowUsage{
		Name:      "mock",
		RowNumber: 2,
	}}, nil
}
