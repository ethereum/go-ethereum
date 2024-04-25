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
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	zrntcommon "github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/beacon/deneb"
	"github.com/protolambda/ztyp/tree"
)

type headerObject interface {
	HashTreeRoot(hFn tree.HashFn) zrntcommon.Root
}

type ExecutionHeader struct {
	obj headerObject
}

// ExecutionHeaderFromJSON decodes an execution header from JSON data provided by
// the beacon chain API.
func ExecutionHeaderFromJSON(forkName string, data []byte) (*ExecutionHeader, error) {
	var obj headerObject
	switch forkName {
	case "capella":
		obj = new(capella.ExecutionPayloadHeader)
	case "deneb":
		obj = new(deneb.ExecutionPayloadHeader)
	default:
		return nil, fmt.Errorf("unsupported fork: " + forkName)
	}
	if err := json.Unmarshal(data, obj); err != nil {
		return nil, err
	}
	return &ExecutionHeader{obj: obj}, nil
}

func NewExecutionHeader(obj headerObject) *ExecutionHeader {
	switch obj.(type) {
	case *capella.ExecutionPayloadHeader:
	case *deneb.ExecutionPayloadHeader:
	default:
		panic(fmt.Errorf("unsupported ExecutionPayloadHeader type %T", obj))
	}
	return &ExecutionHeader{obj: obj}
}

func (eh *ExecutionHeader) PayloadRoot() merkle.Value {
	return merkle.Value(eh.obj.HashTreeRoot(tree.GetHashFn()))
}

func (eh *ExecutionHeader) BlockHash() common.Hash {
	switch obj := eh.obj.(type) {
	case *capella.ExecutionPayloadHeader:
		return common.Hash(obj.BlockHash)
	case *deneb.ExecutionPayloadHeader:
		return common.Hash(obj.BlockHash)
	default:
		panic(fmt.Errorf("unsupported ExecutionPayloadHeader type %T", obj))
	}
}
