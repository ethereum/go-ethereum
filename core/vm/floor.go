// Copyright 2026 The go-ethereum Authors
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

import "github.com/ethereum/go-ethereum/params"

// FloorGasAccumulator implements the per-transaction floor accumulator defined
// by EIP-8279 (Block Access List Byte Floor). It is an internal counter on the
// execution environment, seeded with the EIP-8131 static floor and extended at
// runtime by FloorGasPerByte for every byte an opcode adds to the EIP-7928
// Block Access List.
//
// The accumulator is not part of the signed transaction, is not RLP-encoded,
// gossiped, or persisted; no gas is reserved or deducted from the execution
// budget. It is checked against the transaction's gas limit only to ensure the
// sender can pay the floor if it ends up binding, and at transaction end the
// receipt gas is settled as max(execution_gas_used, floor_gas_used).
type FloorGasAccumulator struct {
	floorGasUsed uint64 // accumulated floor gas (static seed + runtime extensions)
	gasLimit     uint64 // tx.gas; the accumulator must never climb past this
}

// NewFloorGasAccumulator returns an accumulator seeded with the static floor
// and bounded by the transaction gas limit.
func NewFloorGasAccumulator(staticFloor, gasLimit uint64) *FloorGasAccumulator {
	return &FloorGasAccumulator{floorGasUsed: staticFloor, gasLimit: gasLimit}
}

// FloorGasUsed returns the current value of the floor accumulator.
func (f *FloorGasAccumulator) FloorGasUsed() uint64 {
	if f == nil {
		return 0
	}
	return f.floorGasUsed
}

// extendFloor extends the floor accumulator by numBytes BAL bytes, each priced
// at params.FloorGasPerByte. It MUST be called BEFORE the matching BAL
// insertion or state mutation: if the new floor would exceed the transaction
// gas limit it returns ErrOutOfGas, which aborts the operation before any
// unpaid BAL byte exists. A nil accumulator (pre-EIP-8279, or contexts without
// BAL construction) is a no-op.
func (f *FloorGasAccumulator) extendFloor(numBytes uint64) error {
	if f == nil {
		return nil
	}
	// numBytes is bounded by deployed-code length in the worst case; guard the
	// multiplication against overflow before checking against the gas limit.
	if numBytes > (^uint64(0))/params.FloorGasPerByte {
		return ErrOutOfGas
	}
	extension := numBytes * params.FloorGasPerByte
	if f.floorGasUsed > f.gasLimit-min(f.gasLimit, extension) {
		return ErrOutOfGas
	}
	newFloor := f.floorGasUsed + extension
	if newFloor > f.gasLimit {
		return ErrOutOfGas
	}
	f.floorGasUsed = newFloor
	return nil
}
