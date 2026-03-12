// Copyright 2025 The go-ethereum Authors
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

// pushtx submits a raw signed transaction to an Ethereum JSON-RPC endpoint.
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

const defaultRPCURL = "http://127.0.0.1:8545"

func main() {
	if err := run(os.Args[1:], os.Stdin); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader) error {
	var (
		rpcURL string
		txHex  string
	)
	// Parse flags manually so the tool stays minimal.
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--rpc" || args[i] == "-rpc":
			i++
			if i >= len(args) {
				return fmt.Errorf("missing value for %s", args[i-1])
			}
			rpcURL = args[i]
		case strings.HasPrefix(args[i], "--rpc="):
			rpcURL = strings.TrimPrefix(args[i], "--rpc=")
		case strings.HasPrefix(args[i], "-rpc="):
			rpcURL = strings.TrimPrefix(args[i], "-rpc=")
		case args[i] == "-h" || args[i] == "--help":
			printUsage()
			return nil
		case strings.HasPrefix(args[i], "-"):
			return fmt.Errorf("unknown flag: %s", args[i])
		default:
			if txHex != "" {
				return fmt.Errorf("unexpected argument: %s", args[i])
			}
			txHex = args[i]
		}
	}

	if rpcURL == "" {
		rpcURL = defaultRPCURL
	}

	// Read transaction hex from stdin when no positional argument is given.
	if txHex == "" {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		txHex = strings.TrimSpace(string(data))
	}
	if txHex == "" {
		return fmt.Errorf("no transaction data provided\nUsage: pushtx [--rpc URL] <tx-hex>")
	}

	rawTx, err := hex.DecodeString(strings.TrimPrefix(txHex, "0x"))
	if err != nil {
		return fmt.Errorf("invalid hex data: %w", err)
	}

	// Decode the transaction so we can display a summary.
	var tx types.Transaction
	if err := tx.UnmarshalBinary(rawTx); err != nil {
		return fmt.Errorf("decoding transaction: %w", err)
	}
	printTxSummary(&tx)

	// Send to the RPC endpoint.
	hash, err := sendRawTransaction(rpcURL, rawTx)
	if err != nil {
		return fmt.Errorf("sending transaction: %w", err)
	}
	fmt.Println("Transaction submitted successfully")
	fmt.Println("Hash:", hash.Hex())
	return nil
}

// sendRawTransaction dials the given RPC endpoint and calls
// eth_sendRawTransaction with the provided raw bytes.
func sendRawTransaction(rpcURL string, rawTx []byte) (common.Hash, error) {
	client, err := rpc.Dial(rpcURL)
	if err != nil {
		return common.Hash{}, fmt.Errorf("connecting to %s: %w", rpcURL, err)
	}
	defer client.Close()

	var hash common.Hash
	err = client.CallContext(context.Background(), &hash, "eth_sendRawTransaction", hexutil.Encode(rawTx))
	if err != nil {
		return common.Hash{}, err
	}
	return hash, nil
}

// printTxSummary displays the decoded transaction details to stdout.
func printTxSummary(tx *types.Transaction) {
	signer := types.LatestSignerForChainID(tx.ChainId())
	from, err := types.Sender(signer, tx)
	if err != nil {
		from = common.Address{}
	}

	fmt.Println("Transaction details:")
	fmt.Println("  Type:     ", tx.Type())
	fmt.Println("  From:     ", from.Hex())
	if tx.To() != nil {
		fmt.Println("  To:       ", tx.To().Hex())
	} else {
		fmt.Println("  To:        (contract creation)")
	}
	fmt.Println("  Nonce:    ", tx.Nonce())
	fmt.Println("  Value:    ", formatWei(tx.Value()))
	fmt.Println("  Gas limit:", tx.Gas())
	fmt.Println("  Chain ID: ", tx.ChainId())
}

// formatWei converts a wei amount to a human-readable string showing
// both the wei value and the ETH equivalent.
func formatWei(wei *big.Int) string {
	if wei == nil || wei.Sign() == 0 {
		return "0 wei (0 ETH)"
	}
	ether := new(big.Float).Quo(new(big.Float).SetInt(wei), new(big.Float).SetFloat64(1e18))
	return fmt.Sprintf("%s wei (%s ETH)", wei.String(), ether.Text('f', 18))
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: pushtx [--rpc URL] <tx-hex>

Submit a raw signed Ethereum transaction to a JSON-RPC endpoint.

The transaction data can be provided as a positional argument or via stdin.

Options:
  --rpc URL   JSON-RPC endpoint (default: %s)
  -h, --help  Show this help message

Examples:
  pushtx --rpc http://localhost:8545 0xf86c...
  echo 0xf86c... | pushtx --rpc http://localhost:8545
`, defaultRPCURL)
}
