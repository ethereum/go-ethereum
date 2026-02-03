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

package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rlp"
)

// Payload represents the input data for stateless execution containing
// a block and its associated witness data for verification.
type Payload struct {
	ChainID uint64
	Block   *types.Block
	Witness *stateless.Witness
}

func init() {
	debug.SetGCPercent(-1) // Disable garbage collection
}

func main() {
	input := getInput()
	var payload Payload
	rlp.DecodeBytes(input, &payload)

	chainConfig, err := getChainConfig(payload.ChainID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get chain config: %v\n", err)
		os.Exit(13)
	}
	vmConfig := vm.Config{}

	crossStateRoot, crossReceiptRoot, err := core.ExecuteStateless(chainConfig, vmConfig, payload.Block, payload.Witness)
	if err != nil {
		fmt.Fprintf(os.Stderr, "stateless self-validation failed: %v\n", err)
		os.Exit(10)
	}
	if crossStateRoot != payload.Block.Root() {
		fmt.Fprintf(os.Stderr, "stateless self-validation root mismatch (cross: %x local: %x)\n", crossStateRoot, payload.Block.Root())
		os.Exit(11)
	}
	if crossReceiptRoot != payload.Block.ReceiptHash() {
		fmt.Fprintf(os.Stderr, "stateless self-validation receipt root mismatch (cross: %x local: %x)\n", crossReceiptRoot, payload.Block.ReceiptHash())
		os.Exit(12)
	}
}
