package native

// Copyright 2021 The go-ethereum Authors
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

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/internal"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

func init() {
	tracers.DefaultDirectory.Register("keccak256PreimageTracer", newKeccak256PreimageTracer, false)
}

// keccak256PreimageTracer is a native tracer that collects preimages of all KECCAK256 operations.
// This tracer is particularly useful for analyzing smart contract execution patterns,
// especially when debugging storage access in Solidity mappings and dynamic arrays.
type keccak256PreimageTracer struct {
	computedHashes map[common.Hash]hexutil.Bytes
}

// newKeccak256PreimageTracer returns a new keccak256PreimageTracer instance.
func newKeccak256PreimageTracer(ctx *tracers.Context, cfg json.RawMessage, chainConfig *params.ChainConfig) (*tracers.Tracer, error) {
	t := &keccak256PreimageTracer{
		computedHashes: make(map[common.Hash]hexutil.Bytes),
	}
	return &tracers.Tracer{
		Hooks: &tracing.Hooks{
			OnOpcode: t.OnOpcode,
		},
		GetResult: t.GetResult,
	}, nil
}

func (t *keccak256PreimageTracer) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	if op == byte(vm.KECCAK256) {
		sd := scope.StackData()
		// it turns out that sometimes the stack is empty, evm will fail in this case, but we should not panic here
		if len(sd) < 2 {
			return
		}

		dataOffset := internal.StackBack(sd, 0).Uint64()
		dataLength := internal.StackBack(sd, 1).Uint64()
		preimage, err := internal.GetMemoryCopyPadded(scope.MemoryData(), int64(dataOffset), int64(dataLength))
		if err != nil {
			log.Warn("keccak256PreimageTracer: failed to copy keccak preimage from memory", "err", err)
			return
		}

		hash := crypto.Keccak256(preimage)

		t.computedHashes[common.Hash(hash)] = hexutil.Bytes(preimage)
	}
}

// GetResult returns the collected keccak256 preimages as a JSON object mapping hashes to preimages.
func (t *keccak256PreimageTracer) GetResult() (json.RawMessage, error) {
	msg, err := json.Marshal(t.computedHashes)
	if err != nil {
		return nil, err
	}
	return msg, nil
}
