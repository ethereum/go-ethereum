// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// fetchpayload queries an Ethereum node over RPC, fetches a block and its
// execution witness, and writes the combined Payload (ChainID + Block +
// Witness) to disk in the format consumed by cmd/keeper.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

// Payload is duplicated from cmd/keeper/main.go (package main, not importable).
type Payload struct {
	ChainID uint64
	Block   *types.Block
	Witness *stateless.Witness
}

func main() {
	var (
		rpcURL   = flag.String("rpc", "http://localhost:8545", "RPC endpoint URL")
		blockArg = flag.String("block", "latest", `Block number: decimal, 0x-hex, or "latest"`)
		format   = flag.String("format", "rlp", "Comma-separated output formats: rlp, hex, json")
		outDir   = flag.String("out", "", "Output directory (default: current directory)")
	)
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Parse block number (nil means "latest" in ethclient).
	blockNum, err := parseBlockNumber(*blockArg)
	if err != nil {
		fatal("invalid block number %q: %v", *blockArg, err)
	}

	// Connect to the node.
	client, err := ethclient.DialContext(ctx, *rpcURL)
	if err != nil {
		fatal("failed to connect to %s: %v", *rpcURL, err)
	}
	defer client.Close()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		fatal("failed to get chain ID: %v", err)
	}

	// Fetch the block first so we have a concrete number for the witness call,
	// avoiding a race where "latest" advances between the two RPCs.
	block, err := client.BlockByNumber(ctx, blockNum)
	if err != nil {
		fatal("failed to fetch block: %v", err)
	}
	fmt.Printf("Fetched block %d (%#x)\n", block.NumberU64(), block.Hash())

	// Fetch the execution witness via the debug namespace.
	var extWitness stateless.ExtWitness
	err = client.Client().CallContext(ctx, &extWitness, "debug_executionWitness", rpc.BlockNumber(block.NumberU64()))
	if err != nil {
		fatal("failed to fetch execution witness: %v", err)
	}

	witness, err := fromExtWitness(&extWitness)
	if err != nil {
		fatal("failed to convert witness: %v", err)
	}

	payload := Payload{
		ChainID: chainID.Uint64(),
		Block:   block,
		Witness: witness,
	}

	// Encode payload as RLP (shared by "rlp" and "hex" formats).
	rlpBytes, err := rlp.EncodeToBytes(payload)
	if err != nil {
		fatal("failed to RLP-encode payload: %v", err)
	}

	// Write one output file per requested format.
	blockHex := fmt.Sprintf("%x", block.NumberU64())
	for f := range strings.SplitSeq(*format, ",") {
		f = strings.TrimSpace(f)
		outPath := filepath.Join(*outDir, fmt.Sprintf("%s_payload.%s", blockHex, f))

		var data []byte
		switch f {
		case "rlp":
			data = rlpBytes
		case "hex":
			data = []byte(hexutil.Encode(rlpBytes))
		case "json":
			data, err = marshalJSONPayload(chainID, block, &extWitness)
			if err != nil {
				fatal("failed to JSON-encode payload: %v", err)
			}
		default:
			fatal("unknown format %q (valid: rlp, hex, json)", f)
		}

		if err := os.WriteFile(outPath, data, 0644); err != nil {
			fatal("failed to write %s: %v", outPath, err)
		}
		fmt.Printf("Wrote %s (%d bytes)\n", outPath, len(data))
	}
}

// parseBlockNumber converts a CLI string to *big.Int.
// Returns nil for "latest" (ethclient convention for the head block).
func parseBlockNumber(s string) (*big.Int, error) {
	if strings.EqualFold(s, "latest") {
		return nil, nil
	}
	n := new(big.Int)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		if _, ok := n.SetString(s[2:], 16); !ok {
			return nil, fmt.Errorf("invalid hex number")
		}
		return n, nil
	}
	if _, ok := n.SetString(s, 10); !ok {
		return nil, fmt.Errorf("invalid decimal number")
	}
	return n, nil
}

// fromExtWitness converts the consensus ExtWitness into the internal Witness.
// Duplicated from core/stateless (unexported method) and cmd/keeper/getpayload_example.go.
func fromExtWitness(ext *stateless.ExtWitness) (*stateless.Witness, error) {
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

// jsonPayload is a JSON-friendly representation of Payload. It uses ExtWitness
// instead of the internal Witness (which has no JSON marshaling).
type jsonPayload struct {
	ChainID uint64                `json:"chainId"`
	Block   *types.Block          `json:"block"`
	Witness *stateless.ExtWitness `json:"witness"`
}

func marshalJSONPayload(chainID *big.Int, block *types.Block, ext *stateless.ExtWitness) ([]byte, error) {
	return json.MarshalIndent(jsonPayload{
		ChainID: chainID.Uint64(),
		Block:   block,
		Witness: ext,
	}, "", "  ")
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
