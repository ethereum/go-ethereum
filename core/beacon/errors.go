// Copyright 2022 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package beacon

import "github.com/ethereum/go-ethereum/rpc"

var (
	// VALID is returned by the engine API in the following calls:
	//   - newPayloadV1:       if the payload was already known or was just validated and executed
	//   - forkchoiceUpdateV1: if the chain accepted the reorg (might ignore if it's stale)
	VALID = "VALID"

	// INVALID is returned by the engine API in the following calls:
	//   - newPayloadV1:       if the payload failed to execute on top of the local chain
	//   - forkchoiceUpdateV1: if the new head is unknown, pre-merge, or reorg to it fails
	INVALID = "INVALID"

	// SYNCING is returned by the engine API in the following calls:
	//   - newPayloadV1:       if the payload was accepted on top of an active sync
	//   - forkchoiceUpdateV1: if the new head was seen before, but not part of the chain
	SYNCING = "SYNCING"

	// ACCEPTED is returned by the engine API in the following calls:
	//   - newPayloadV1: if the payload was accepted, but not processed (side chain)
	ACCEPTED = "ACCEPTED"

	INVALIDBLOCKHASH     = "INVALID_BLOCK_HASH"
	INVALIDTERMINALBLOCK = "INVALID_TERMINAL_BLOCK"

	GenericServerError = rpc.CustomError{Code: -32000, ValidationError: "Server error"}
	UnknownPayload     = rpc.CustomError{Code: -32001, ValidationError: "Unknown payload"}
	InvalidTB          = rpc.CustomError{Code: -32002, ValidationError: "Invalid terminal block"}

	STATUS_INVALID = ForkChoiceResponse{PayloadStatus: PayloadStatusV1{Status: INVALID}, PayloadID: nil}
	STATUS_SYNCING = ForkChoiceResponse{PayloadStatus: PayloadStatusV1{Status: SYNCING}, PayloadID: nil}
)
