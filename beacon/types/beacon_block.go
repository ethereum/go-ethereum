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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	zrntcommon "github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/tree"

	// beacon forks
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/beacon/deneb"
	"github.com/protolambda/zrnt/eth2/beacon/electra"
)

type blockObject interface {
	HashTreeRoot(spec *zrntcommon.Spec, hFn tree.HashFn) zrntcommon.Root
	Header(spec *zrntcommon.Spec) *zrntcommon.BeaconBlockHeader
}

// BeaconBlock represents a full block in the beacon chain.
type BeaconBlock struct {
	jsonBeaconBlock
	blockObj blockObject
}

type jsonBeaconBlock struct {
	Version             string `json:"version"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
	Finalized           bool   `json:"finalized"`
	Data                struct {
		Message   json.RawMessage `json:"message"`
		Signature hexutil.Bytes   `json:"signature"`
	} `json:"data"`
}

// UnmarshalJSON implements json.Marshaler.
func (b *BeaconBlock) UnmarshalJSON(input []byte) error {
	if err := json.Unmarshal(input, &b.jsonBeaconBlock); err != nil {
		return err
	}
	switch b.Version {
	case "capella":
		b.blockObj = new(capella.BeaconBlock)
	case "deneb":
		b.blockObj = new(deneb.BeaconBlock)
	case "electra":
		b.blockObj = new(electra.BeaconBlock)
	default:
		return fmt.Errorf("unsupported fork: %s", b.Version)
	}
	return json.Unmarshal(b.Data.Message, b.blockObj)
}

// MarshalJSON implements json.Marshaler.
/*func (b *BeaconBlock) MarshalJSON() ([]byte, error) {
	var bb jsonBeaconBlock
	bb.Version = b.version
	bb.Data.Message = b.json
	return json.Marshal(&bb)
}*/

// NewBeaconBlock wraps a ZRNT block (only used for testing).
func NewBeaconBlock(obj blockObject) *BeaconBlock {
	switch obj := obj.(type) {
	case *capella.BeaconBlock:
		return &BeaconBlock{blockObj: obj}
	case *deneb.BeaconBlock:
		return &BeaconBlock{blockObj: obj}
	case *electra.BeaconBlock:
		return &BeaconBlock{blockObj: obj}
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
	case *electra.BeaconBlock:
		return uint64(obj.Slot)
	default:
		panic(fmt.Errorf("unsupported block type %T", b.blockObj))
	}
}

// ExecutionPayload parses and returns the execution payload of the block.
func (b *BeaconBlock) ExecutionPayload() (*types.Block, error) {
	switch obj := b.blockObj.(type) {
	case *capella.BeaconBlock:
		return convertPayload(&obj.Body.ExecutionPayload, &obj.ParentRoot, nil)
	case *deneb.BeaconBlock:
		return convertPayload(&obj.Body.ExecutionPayload, &obj.ParentRoot, nil)
	case *electra.BeaconBlock:
		requests := b.ExecutionRequestsList()
		return convertPayload(&obj.Body.ExecutionPayload, &obj.ParentRoot, requests)
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
	case *electra.BeaconBlock:
		return headerFromZRNT(obj.Header(configs.Mainnet))
	default:
		panic(fmt.Errorf("unsupported block type %T", b.blockObj))
	}
}

// Root computes the SSZ root hash of the block.
func (b *BeaconBlock) Root() common.Hash {
	return common.Hash(b.blockObj.HashTreeRoot(configs.Mainnet, tree.GetHashFn()))
}

// ExecutionRequestsList returns the execution layer requests of the block.
func (b *BeaconBlock) ExecutionRequestsList() [][]byte {
	switch obj := b.blockObj.(type) {
	case *capella.BeaconBlock, *deneb.BeaconBlock:
		return nil
	case *electra.BeaconBlock:
		r := obj.Body.ExecutionRequests
		return marshalRequests(configs.Mainnet,
			&r.Deposits,
			&r.Withdrawals,
			&r.Consolidations,
		)
	default:
		panic(fmt.Errorf("unsupported block type %T", b.blockObj))
	}
}

func marshalRequests(spec *zrntcommon.Spec, items ...zrntcommon.SpecObj) (list [][]byte) {
	var buf bytes.Buffer
	list = [][]byte{}
	for typ, data := range items {
		buf.Reset()
		buf.WriteByte(byte(typ))
		w := codec.NewEncodingWriter(&buf)
		if err := data.Serialize(spec, w); err != nil {
			panic(err)
		}
		if buf.Len() == 1 {
			continue // skip empty requests
		}
		list = append(list, bytes.Clone(buf.Bytes()))
	}
	return list
}
