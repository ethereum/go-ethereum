// Copyright 2023 The go-ethereum Authors
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

package params

// Verkle tree EIP: costs associated to witness accesses
var (
	WitnessBranchReadCost  uint64 = 1900
	WitnessChunkReadCost   uint64 = 200
	WitnessBranchWriteCost uint64 = 3000
	WitnessChunkWriteCost  uint64 = 500
	WitnessChunkFillCost   uint64 = 6200
)

// ClearVerkleWitnessCosts sets all witness costs to 0, which is necessary
// for historical block replay simulations.
func ClearVerkleWitnessCosts() {
	WitnessBranchReadCost = 0
	WitnessChunkReadCost = 0
	WitnessBranchWriteCost = 0
	WitnessChunkWriteCost = 0
	WitnessChunkFillCost = 0
}
