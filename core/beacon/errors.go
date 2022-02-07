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
	VALID              = GenericStringResponse{"VALID"}
	SUCCESS            = GenericStringResponse{"SUCCESS"}
	INVALID            = ForkChoiceResponse{Status: "INVALID", PayloadID: nil}
	SYNCING            = ForkChoiceResponse{Status: "SYNCING", PayloadID: nil}
	GenericServerError = rpc.CustomError{Code: -32000, ValidationError: "Server error"}
	UnknownPayload     = rpc.CustomError{Code: -32001, ValidationError: "Unknown payload"}
	InvalidTB          = rpc.CustomError{Code: -32002, ValidationError: "Invalid terminal block"}
)
