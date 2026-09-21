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

	// beacon forks
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/electra"
)

// blockObject is implemented by the fork-specific beacon block types. Note
// that forks which did not change the block structure reuse the type of the
// previous fork (e.g. fulu blocks are represented by electra.BeaconBlock).
type blockObject interface {
	HashTreeRoot() ([32]byte, error)
}

// beaconBlockData is the fork-specific representation of a beacon block's contents.
type beaconBlockData interface {
	Slot() uint64
	Header() Header
	Root() common.Hash
	ExecutionPayload() (*types.Block, error)
	ExecutionRequestsList() [][]byte
}

// BeaconBlock represents a full block in the beacon chain.
type BeaconBlock struct {
	data beaconBlockData
}

// Slot returns the slot number of the block.
func (b *BeaconBlock) Slot() uint64 { return b.data.Slot() }

// Header returns the block's header data.
func (b *BeaconBlock) Header() Header { return b.data.Header() }

// Root returns the SSZ root hash of the block.
func (b *BeaconBlock) Root() common.Hash { return b.data.Root() }

// ExecutionPayload returns the execution payload of the block.
func (b *BeaconBlock) ExecutionPayload() (*types.Block, error) { return b.data.ExecutionPayload() }

// ExecutionRequestsList returns the execution layer requests of the block.
func (b *BeaconBlock) ExecutionRequestsList() [][]byte { return b.data.ExecutionRequestsList() }

// BlockFromJSON decodes a beacon block from JSON.
func BlockFromJSON(forkName string, data []byte) (*BeaconBlock, error) {
	var obj blockObject
	switch forkName {
	case "capella":
		obj = new(capella.BeaconBlock)
	case "deneb":
		obj = new(deneb.BeaconBlock)
	case "electra", "fulu":
		obj = new(electra.BeaconBlock)
	case "gloas":
		return decodeGloasBeaconBlock(data)
	default:
		return nil, fmt.Errorf("unsupported fork: %s", forkName)
	}
	if err := json.Unmarshal(data, obj); err != nil {
		return nil, err
	}
	root, err := obj.HashTreeRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to compute block root: %v", err)
	}
	return &BeaconBlock{data: &legacyBeaconBlock{blockObj: obj, root: root}}, nil
}

// NewBeaconBlock wraps a consensus layer block.
func NewBeaconBlock(obj blockObject) *BeaconBlock {
	switch obj.(type) {
	case *capella.BeaconBlock:
	case *deneb.BeaconBlock:
	case *electra.BeaconBlock: // includes fulu blocks
	default:
		panic(fmt.Errorf("unsupported block type %T", obj))
	}
	root, err := obj.HashTreeRoot()
	if err != nil {
		panic(fmt.Errorf("failed to compute block root: %v", err))
	}
	return &BeaconBlock{data: &legacyBeaconBlock{blockObj: obj, root: root}}
}

// legacyBeaconBlock implements beaconBlockData for the forks whose full block
// format is available (capella up to fulu).
type legacyBeaconBlock struct {
	blockObj blockObject
	root     common.Hash
}

// Slot returns the slot number of the block.
func (b *legacyBeaconBlock) Slot() uint64 {
	switch obj := b.blockObj.(type) {
	case *capella.BeaconBlock:
		return uint64(obj.Slot)
	case *deneb.BeaconBlock:
		return uint64(obj.Slot)
	case *electra.BeaconBlock: // includes fulu blocks
		return uint64(obj.Slot)
	default:
		panic(fmt.Errorf("unsupported block type %T", b.blockObj))
	}
}

// ExecutionPayload parses and returns the execution payload of the block.
func (b *legacyBeaconBlock) ExecutionPayload() (*types.Block, error) {
	switch obj := b.blockObj.(type) {
	case *capella.BeaconBlock:
		return convertPayload(obj.Body.ExecutionPayload, obj.ParentRoot, nil)
	case *deneb.BeaconBlock:
		return convertPayload(obj.Body.ExecutionPayload, obj.ParentRoot, nil)
	case *electra.BeaconBlock: // includes fulu blocks
		requests := b.ExecutionRequestsList()
		return convertPayload(obj.Body.ExecutionPayload, obj.ParentRoot, requests)
	default:
		panic(fmt.Errorf("unsupported block type %T", b.blockObj))
	}
}

// Header returns the block's header data.
func (b *legacyBeaconBlock) Header() Header {
	switch obj := b.blockObj.(type) {
	case *capella.BeaconBlock:
		return makeHeader(uint64(obj.Slot), uint64(obj.ProposerIndex), obj.ParentRoot, obj.StateRoot, obj.Body)
	case *deneb.BeaconBlock:
		return makeHeader(uint64(obj.Slot), uint64(obj.ProposerIndex), obj.ParentRoot, obj.StateRoot, obj.Body)
	case *electra.BeaconBlock: // includes fulu blocks
		return makeHeader(uint64(obj.Slot), uint64(obj.ProposerIndex), obj.ParentRoot, obj.StateRoot, obj.Body)
	default:
		panic(fmt.Errorf("unsupported block type %T", b.blockObj))
	}
}

func makeHeader(slot, proposerIndex uint64, parentRoot, stateRoot [32]byte, body blockObject) Header {
	bodyRoot, err := body.HashTreeRoot()
	if err != nil {
		panic(fmt.Errorf("failed to compute body root: %v", err))
	}
	return Header{
		Slot:          slot,
		ProposerIndex: proposerIndex,
		ParentRoot:    parentRoot,
		StateRoot:     stateRoot,
		BodyRoot:      bodyRoot,
	}
}

// Root returns the SSZ root hash of the block.
func (b *legacyBeaconBlock) Root() common.Hash {
	return b.root
}

// ExecutionRequestsList returns the execution layer requests of the block.
func (b *legacyBeaconBlock) ExecutionRequestsList() [][]byte {
	switch obj := b.blockObj.(type) {
	case *capella.BeaconBlock, *deneb.BeaconBlock:
		return nil
	case *electra.BeaconBlock: // includes fulu blocks
		return marshalRequests(obj.Body.ExecutionRequests)
	default:
		panic(fmt.Errorf("unsupported block type %T", b.blockObj))
	}
}

// marshalRequests encodes the execution layer requests into the flat
// type-prefixed representation of EIP-7685, with empty requests omitted.
func marshalRequests(r *electra.ExecutionRequests) [][]byte {
	list := [][]byte{}
	if r == nil {
		return list
	}
	for typ, data := range [][]byte{
		marshalItems(r.Deposits),
		marshalItems(r.Withdrawals),
		marshalItems(r.Consolidations),
	} {
		if len(data) == 0 {
			continue // skip empty requests
		}
		list = append(list, append([]byte{byte(typ)}, data...))
	}
	return list
}

// marshalItems SSZ-encodes a list of fixed-size request objects by simple
// concatenation of the encoded items.
func marshalItems[T interface{ MarshalSSZ() ([]byte, error) }](items []T) []byte {
	var buf []byte
	for _, item := range items {
		enc, err := item.MarshalSSZ()
		if err != nil {
			panic(err)
		}
		buf = append(buf, enc...)
	}
	return buf
}
