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

// Package ssz contains the SSZ wire types for the REST-SSZ Engine API
// (execution-apis PR #793). The containers here are intentionally separate
// from beacon/engine.ExecutableData & friends; they map onto the JSON-RPC
// types via convert.go.
package ssz

// MAX_* constants from refactor-ssz.md.
const (
	MaxBytesPerTx              = 1 << 30 // 1 GiB, EIP-4844
	MaxTxsPerPayload           = 1 << 20 // 1,048,576, Bellatrix
	MaxWithdrawalsPerPayload   = 16      // Capella
	MaxExtraDataBytes          = 32      // Bellatrix
	MaxBlobCommitmentsPerBlock = 1 << 12 // 4,096, Deneb
	FieldElementsPerBlob       = 4096    // EIP-4844
	BytesPerFieldElement       = 32      // EIP-4844
	BytesPerBlob               = FieldElementsPerBlob * BytesPerFieldElement
	CellsPerExtBlob            = 128 // EIP-7594
	FieldElementsPerCell       = 64  // EIP-7594
	// A cell spans FIELD_ELEMENTS_PER_CELL field elements of the *extended*
	// blob, so BytesPerCell = 64 * 32 = 2048. (Not BytesPerBlob/CellsPerExtBlob,
	// which divides the original-blob byte count over the extended-blob cell
	// count and halves the true size — see execution-apis refactor-ssz.md.)
	BytesPerCell                   = FieldElementsPerCell * BytesPerFieldElement
	MaxBalBytes                    = MaxBytesPerTx // EIP-7928 placeholder
	MaxExecutionRequestsPerPayload = 1 << 8        // 256, EIP-7685
	MaxBytesPerExecutionRequest    = MaxBytesPerTx // placeholder
	MaxVersionedHashesPerRequest   = 128           // Osaka
	MaxBlobsRequest                = MaxVersionedHashesPerRequest
	MaxBodiesRequest               = 1 << 5 // 32, Shanghai
	MaxErrorBytes                  = 1024
)

// Status enum values for PayloadStatus.
const (
	StatusValid    uint8 = 0
	StatusInvalid  uint8 = 1
	StatusSyncing  uint8 = 2
	StatusAccepted uint8 = 3
)
