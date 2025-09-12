// Copyright 2025 The go-ethereum Authors
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

//go:build example

package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

// ExtWitness is a witness RLP encoding for transferring across clients.
// This is taken from PR #32216 until it's merged.
// It contains block headers, contract codes, state nodes, and storage keys
// required for stateless execution verification.
type ExtWitness struct {
	Headers []*types.Header `json:"headers"`
	Codes   []hexutil.Bytes `json:"codes"`
	State   []hexutil.Bytes `json:"state"`
	Keys    []hexutil.Bytes `json:"keys"`
}

// This is taken from PR #32216 until it's merged
// fromExtWitness converts the consensus witness format into our internal one.
func fromExtWitness(ext *ExtWitness) (*stateless.Witness, error) {
	w := &stateless.Witness{}
	w.Headers = ext.Headers

	w.Codes = make(map[string]struct{}, len(ext.Codes))
	for _, code := range ext.Codes {
		w.Codes[string(code)] = struct{}{}
	}
	w.State = make(map[string]struct{}, len(ext.State))
	for _, node := range ext.State {
		w.State[string(node)] = struct{}{}
	}
	return w, nil
}

//go:embed 1192c3_witness.rlp
var witnessRlp []byte

//go:embed 1192c3_block.rlp
var blockRlp []byte

// getInput is a platform-specific function that will recover the input payload
// and returns it as a slice. It is expected to be an RLP-encoded Payload structure
// that contains the witness and the block.
// This is a demo version, that is intended to run on a regular computer, so what
// it does is embed a small Hoodi block, encodes the Payload structure containing
// the block and its witness as RLP, and returns the encoding.
func getInput() []byte {
	var block types.Block
	err := rlp.DecodeBytes(blockRlp, &block)
	if err != nil {
		panic(err)
	}

	var extwitness ExtWitness
	err = rlp.DecodeBytes(witnessRlp, &extwitness)
	if err != nil {
		panic(err)
	}
	witness, err := fromExtWitness(&extwitness)
	if err != nil {
		panic(err)
	}

	payload := Payload{
		ChainID: params.HoodiChainConfig.ChainID.Uint64(),
		Block:   &block,
		Witness: witness,
	}

	encoded, err := rlp.EncodeToBytes(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode payload: %v\n", err)
		os.Exit(20)
	}
	return encoded
}
