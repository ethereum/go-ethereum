// Copyright 2024 The go-ethereum Authors
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

package types

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	zrntcommon "github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/beacon/deneb"
	"github.com/protolambda/ztyp/tree"
)

// headerObject is an interface that defines the method to get the HashTreeRoot.
type headerObject interface {
	HashTreeRoot(hFn tree.HashFn) zrntcommon.Root
}

// ExecutionHeader holds a reference to an object that implements the headerObject interface.
type ExecutionHeader struct {
	obj headerObject
}

// ExecutionHeaderFromJSON decodes an execution header from JSON data provided by the beacon chain API.
// It selects the appropriate fork (capella, deneb) and unmarshals the data accordingly.
func ExecutionHeaderFromJSON(forkName string, data []byte) (*ExecutionHeader, error) {
	var obj headerObject

	switch forkName {
	case "capella":
		obj = new(capella.ExecutionPayloadHeader)
	case "deneb":
		obj = new(deneb.ExecutionPayloadHeader)
	default:
		return nil, fmt.Errorf("unsupported fork: %s", forkName)
	}

	// Use a streaming decoder for efficiency, especially with large JSON payloads
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal execution header for fork %s: %w", forkName, err)
	}
	return &ExecutionHeader{obj: obj}, nil
}

// NewExecutionHeader initializes a new ExecutionHeader object with the given headerObject.
// It ensures the object type is one of the supported ExecutionPayloadHeaders (capella, deneb).
func NewExecutionHeader(obj headerObject) *ExecutionHeader {
	switch obj.(type) {
	case *capella.ExecutionPayloadHeader, *deneb.ExecutionPayloadHeader:
		// Supported types
	default:
		panic(fmt.Errorf("unsupported ExecutionPayloadHeader type %T", obj))
	}
	return &ExecutionHeader{obj: obj}
}

// PayloadRoot returns the Merkle root of the execution payload header.
func (eh *ExecutionHeader) PayloadRoot() merkle.Value {
	return merkle.Value(eh.obj.HashTreeRoot(tree.GetHashFn()))
}

// BlockHash extracts the block hash from the underlying execution payload header.
// It checks the type of the object and returns the correct block hash based on the type.
func (eh *ExecutionHeader) BlockHash() common.Hash {
	if obj, ok := eh.obj.(*capella.ExecutionPayloadHeader); ok {
		return common.Hash(obj.BlockHash)
	}
	if obj, ok := eh.obj.(*deneb.ExecutionPayloadHeader); ok {
		return common.Hash(obj.BlockHash)
	}
	panic(fmt.Errorf("unsupported ExecutionPayloadHeader type %T", eh.obj))
}
