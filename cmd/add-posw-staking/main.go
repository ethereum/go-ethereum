// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

// add-posw-staking adds missing PoSW staking contract accounts to a QuarkChain
// genesis configuration's alloc. Shard 0 gets both root and shard staking
// contracts; other shards get only shard staking. Existing accounts are preserved
// and staking contract code is validated before the output file is written.
//
// Usage: add-posw-staking <input-genesis.json> <output-genesis.json>
// The first argument is this project's QuarkChain genesis JSON file: the geth
// genesis format extended with config.quarkchain. Its shardId selects the staking
// contracts to add; an omitted shardId defaults to 0. A plain geth genesis file
// without config.quarkchain is rejected.
// The second argument is the output path for the complete updated genesis JSON;
// an existing output file is overwritten after successful generation.
//
// Example (run from the repository root):
//
//	go run ./cmd/add-posw-staking ./genesis.base.json ./genesis.json
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: add-posw-staking genesis.base.json genesis.json")
		os.Exit(2)
	}
	input, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer input.Close()
	var output bytes.Buffer
	if err := generate(input, &output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(os.Args[2], output.Bytes(), 0644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(input io.Reader, output io.Writer) error {
	var genesis core.Genesis
	if err := json.NewDecoder(input).Decode(&genesis); err != nil {
		return fmt.Errorf("decode genesis: %w", err)
	}
	if genesis.Config == nil || genesis.Config.QuarkChain == nil {
		return errors.New("genesis requires a quarkchain configuration")
	}
	if err := genesis.Config.CheckConfigForkOrder(); err != nil {
		return err
	}
	if genesis.Alloc == nil {
		genesis.Alloc = make(types.GenesisAlloc)
	}
	for address, account := range core.PoSWStakingContractAllocs(genesis.Config.QuarkChain.ShardID) {
		if _, exists := genesis.Alloc[address]; !exists {
			genesis.Alloc[address] = account
		}
	}
	if err := genesis.ValidatePoSWStakingContracts(); err != nil {
		return err
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(&genesis)
}
