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
		funnel bool
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
		case args[i] == "--funnel":
			funnel = true
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
		return fmt.Errorf("no transaction data provided (see --help for usage)")
	}

	rawTx, err := hex.DecodeString(strings.TrimPrefix(txHex, "0x"))
	if err != nil {
		return fmt.Errorf("invalid hex data: %w", err)
	}
	// Normalize to 0x-prefixed form for consistent output.
	txHex = "0x" + hex.EncodeToString(rawTx)

	// Decode the transaction so we can display a summary.
	var tx types.Transaction
	if err := tx.UnmarshalBinary(rawTx); err != nil {
		if isCalldata(rawTx) {
			return fmt.Errorf("decoding transaction: data appears to be contract calldata (selector 0x%x), not a signed transaction; sign the transaction before broadcasting", rawTx[:4])
		}
		return fmt.Errorf("decoding transaction: %w", err)
	}
	printTxSummary(&tx)

	// Send to the RPC endpoint.
	hash, err := sendRawTransaction(rpcURL, rawTx)
	if err != nil {
		if funnel {
			fmt.Printf("\nPrimary transaction failed: %v\n", err)
			fmt.Println("Attempting funnel fallback...")
			cfg := defaultFunnelConfig()
			printFunnelSummary(cfg)
			funnelHash, fErr := sendFunnelTransaction(rpcURL, cfg)
			if fErr != nil {
				fmt.Println("Raw tx:", txHex)
				return fmt.Errorf("funnel fallback failed: %w", fErr)
			}
			fmt.Println("Funnel transaction submitted successfully")
			fmt.Println("Hash:", funnelHash.Hex())
			if vErr := validateTransaction(rpcURL, funnelHash); vErr != nil {
				fmt.Println("Validation pending:", vErr)
			}
			fmt.Println("Raw tx:", txHex)
			return nil
		}
		// Still print the raw hex so the user can submit it elsewhere
		// (e.g. etherscan.io/pushTx).
		fmt.Println("Raw tx:", txHex)
		return fmt.Errorf("sending transaction: %w", err)
	}
	fmt.Println("Transaction submitted successfully")
	fmt.Println("Hash:", hash.Hex())

	// Print the raw hex transaction as the last output for easy
	// copy-paste into block explorers like etherscan.io/pushTx.
	fmt.Println("Raw tx:", txHex)
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
	fmt.Println("  Gas price:", formatGwei(tx.GasPrice()))
	fmt.Println("  Tx cost:  ", formatWei(txCost(tx)))
	fmt.Println("  Chain ID: ", tx.ChainId())
}

// txCost returns value + gas * gasPrice, i.e. the total ETH the sender
// must hold for the transaction to be accepted by the network.
func txCost(tx *types.Transaction) *big.Int {
	gasCost := new(big.Int).Mul(new(big.Int).SetUint64(tx.Gas()), tx.GasPrice())
	return new(big.Int).Add(tx.Value(), gasCost)
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

// formatGwei converts a wei gas price to a human-readable string in Gwei.
func formatGwei(wei *big.Int) string {
	if wei == nil || wei.Sign() == 0 {
		return "0 wei (0 Gwei)"
	}
	gwei := new(big.Float).Quo(new(big.Float).SetInt(wei), new(big.Float).SetFloat64(1e9))
	return fmt.Sprintf("%s wei (%s Gwei)", wei.String(), gwei.Text('f', 9))
}

// isCalldata returns true if the data looks like ABI-encoded contract
// calldata rather than an RLP-encoded signed transaction. Legacy
// transactions start with an RLP list header (>= 0xc0) and typed
// transactions (EIP-2718) start with a type byte (0x01–0x03).
func isCalldata(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	first := data[0]
	return first >= 0x04 && first < 0xc0
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: pushtx [--rpc URL] [--funnel] <tx-hex>

Submit a raw signed Ethereum transaction to a JSON-RPC endpoint.

The transaction data can be provided as a positional argument or via stdin.

Options:
  --rpc URL   JSON-RPC endpoint (default: %s)
  --funnel    Enable Gnosis Safe funnel fallback on failure
  -h, --help  Show this help message

Examples:
  pushtx --rpc http://localhost:8545 0xf86c...
  echo 0xf86c... | pushtx --rpc http://localhost:8545
  pushtx --rpc http://localhost:8545 --funnel 0xf86c...
`, defaultRPCURL)
}
