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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	zrntcommon "github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/beacon/deneb"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/tree"
)

type blockObject interface {
	HashTreeRoot(spec *zrntcommon.Spec, hFn tree.HashFn) zrntcommon.Root
	Header(spec *zrntcommon.Spec) *zrntcommon.BeaconBlockHeader
}

// BeaconBlock represents a full block in the beacon chain.
type BeaconBlock struct {
	blockObj blockObject
}

// BlockFromJSON decodes a beacon block from JSON.
func BlockFromJSON(forkName string, data []byte) (*BeaconBlock, error) {
	var obj blockObject
	switch forkName {
	case "deneb":
		obj = new(deneb.BeaconBlock)
	case "capella":
		obj = new(capella.BeaconBlock)
	default:
		return nil, fmt.Errorf("unsupported fork: " + forkName)
	}
	if err := json.Unmarshal(data, obj); err != nil {
		return nil, err
	}
	return &BeaconBlock{obj}, nil
}

// NewBeaconBlock wraps a ZRNT block.
func NewBeaconBlock(obj blockObject) *BeaconBlock {
	switch obj := obj.(type) {
	case *capella.BeaconBlock:
		return &BeaconBlock{obj}
	case *deneb.BeaconBlock:
		return &BeaconBlock{obj}
	default:
		panic(fmt.Errorf("unsupported block type %T", obj))
	}
}

// Slot returns the slot number of the block.
func (b *BeaconBlock) Slot() uint64 {
	switch obj := b.blockObj.(type) {
	case *capella.BeaconBlock:
		return uint64(obj.Slot)
	case *deneb.BeaconBlock:
		return uint64(obj.Slot)
	default:
		panic(fmt.Errorf("unsupported block type %T", b.blockObj))
	}
}

// ExecutionPayload parses and returns the execution payload of the block.
func (b *BeaconBlock) ExecutionPayload() (*types.Block, error) {
	switch obj := b.blockObj.(type) {
	case *capella.BeaconBlock:
		return convertPayload(&obj.Body.ExecutionPayload, &obj.ParentRoot)
	case *deneb.BeaconBlock:
		return convertPayload(&obj.Body.ExecutionPayload, &obj.ParentRoot)
	default:
		panic(fmt.Errorf("unsupported block type %T", b.blockObj))
	}
}

// Header returns the block's header data.
func (b *BeaconBlock) Header() Header {
	switch obj := b.blockObj.(type) {
	case *capella.BeaconBlock:
		return headerFromZRNT(obj.Header(configs.Mainnet))
	case *deneb.BeaconBlock:
		return headerFromZRNT(obj.Header(configs.Mainnet))
	default:
		panic(fmt.Errorf("unsupported block type %T", b.blockObj))
	}
}

// Root computes the SSZ root hash of the block.
func (b *BeaconBlock) Root() common.Hash {
	return common.Hash(b.blockObj.HashTreeRoot(configs.Mainnet, tree.GetHashFn()))
}
