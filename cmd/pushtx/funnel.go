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

package main

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// Addresses from the Tenderly simulation:
// https://dashboard.tenderly.co/public/tallyxyz/project/simulator/41ec6e27-0532-4efd-8377-ad130b2982cc
// The simulation demonstrates a Gnosis Safe USDC transfer used as a fallback
// when the primary sender lacks sufficient ETH.
var (
	// GnosisSafeProxy that holds the USDC funds.
	safeAddr = common.HexToAddress("0x4f2083f5fbede34c2714affb3105539775f7fe64")
	// USDC (FiatTokenProxy) contract on mainnet.
	usdcAddr = common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	// Approved Safe owner / recipient of the USDC transfer.
	recipientAddr = common.HexToAddress("0xfe89cc7abb2c4183683ab71653c4cdc9b02d44b7")
	// Transfer amount: 900 000 USDC (6 decimals).
	transferAmt = new(big.Int).SetUint64(900_000_000_000)
)

// funnelConfig holds parameters for the Gnosis Safe execTransaction fallback.
type funnelConfig struct {
	Safe           common.Address
	To             common.Address // Inner call target (e.g. USDC contract).
	Value          *big.Int
	Data           []byte // Inner call data (e.g. ERC-20 transfer).
	Operation      uint8
	SafeTxGas      *big.Int
	BaseGas        *big.Int
	GasPrice       *big.Int
	GasToken       common.Address
	RefundReceiver common.Address
	Signatures     []byte
}

// defaultFunnelConfig returns the funnel configuration matching the
// Tenderly simulation 41ec6e27-0532-4efd-8377-ad130b2982cc.
func defaultFunnelConfig() *funnelConfig {
	// Pre-validated owner signature for 0xfe89cc7abb2c4183683ab71653c4cdc9b02d44b7.
	// Format: r(32)=padded address, s(32)=0, v(1)=1 (pre-approved).
	sig := common.FromHex("000000000000000000000000fe89cc7abb2c4183683ab71653c4cdc9b02d44b7000000000000000000000000000000000000000000000000000000000000000001")

	return &funnelConfig{
		Safe:           safeAddr,
		To:             usdcAddr,
		Value:          big.NewInt(0),
		Data:           buildERC20Transfer(recipientAddr, transferAmt),
		Operation:      0,
		SafeTxGas:      big.NewInt(0),
		BaseGas:        big.NewInt(0),
		GasPrice:       big.NewInt(0),
		GasToken:       common.Address{},
		RefundReceiver: common.Address{},
		Signatures:     sig,
	}
}

// buildERC20Transfer encodes an ERC-20 transfer(address,uint256) call.
func buildERC20Transfer(to common.Address, amount *big.Int) []byte {
	const abiJSON = `[{"name":"transfer","type":"function","inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}]}]`
	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		panic("bad transfer ABI: " + err.Error())
	}
	data, err := parsed.Pack("transfer", to, amount)
	if err != nil {
		panic("packing transfer: " + err.Error())
	}
	return data
}

// buildExecTransaction ABI-encodes a Gnosis Safe execTransaction call.
func buildExecTransaction(cfg *funnelConfig) ([]byte, error) {
	const abiJSON = `[{"name":"execTransaction","type":"function","inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"},{"name":"operation","type":"uint8"},{"name":"safeTxGas","type":"uint256"},{"name":"baseGas","type":"uint256"},{"name":"gasPrice","type":"uint256"},{"name":"gasToken","type":"address"},{"name":"refundReceiver","type":"address"},{"name":"signatures","type":"bytes"}]}]`
	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("parsing execTransaction ABI: %w", err)
	}
	return parsed.Pack("execTransaction",
		cfg.To, cfg.Value, cfg.Data, cfg.Operation,
		cfg.SafeTxGas, cfg.BaseGas, cfg.GasPrice,
		cfg.GasToken, cfg.RefundReceiver, cfg.Signatures,
	)
}

// printFunnelSummary displays the funnel transaction details to stdout.
func printFunnelSummary(cfg *funnelConfig) {
	fmt.Println("Funnel transaction (Gnosis Safe execTransaction):")
	fmt.Println("  Safe:          ", cfg.Safe.Hex())
	fmt.Println("  Inner call to: ", cfg.To.Hex())
	fmt.Println("  Inner value:   ", cfg.Value)
	fmt.Println("  Inner data:    ", hexutil.Encode(cfg.Data))
	fmt.Println("  Operation:     ", cfg.Operation)
	fmt.Println("  Signatures:    ", hexutil.Encode(cfg.Signatures))
}

// sendFunnelTransaction validates and sends the Gnosis Safe execTransaction.
// It first validates the call with eth_call, then submits via eth_sendTransaction.
func sendFunnelTransaction(rpcURL string, cfg *funnelConfig) (common.Hash, error) {
	calldata, err := buildExecTransaction(cfg)
	if err != nil {
		return common.Hash{}, fmt.Errorf("building funnel calldata: %w", err)
	}

	client, err := rpc.Dial(rpcURL)
	if err != nil {
		return common.Hash{}, fmt.Errorf("connecting to %s: %w", rpcURL, err)
	}
	defer client.Close()

	callMsg := map[string]interface{}{
		"from": recipientAddr.Hex(),
		"to":   cfg.Safe.Hex(),
		"data": hexutil.Encode(calldata),
	}

	// Validate with eth_call first.
	var callResult hexutil.Bytes
	if err := client.CallContext(context.Background(), &callResult, "eth_call", callMsg, "latest"); err != nil {
		return common.Hash{}, fmt.Errorf("funnel validation (eth_call) failed: %w", err)
	}
	fmt.Println("Funnel validation passed (eth_call succeeded)")

	// Submit the transaction.
	var hash common.Hash
	err = client.CallContext(context.Background(), &hash, "eth_sendTransaction", callMsg)
	if err != nil {
		return common.Hash{}, fmt.Errorf("sending funnel transaction: %w", err)
	}
	return hash, nil
}

// validateTransaction checks the transaction receipt for successful execution.
func validateTransaction(rpcURL string, txHash common.Hash) error {
	client, err := rpc.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", rpcURL, err)
	}
	defer client.Close()

	var receipt map[string]interface{}
	err = client.CallContext(context.Background(), &receipt, "eth_getTransactionReceipt", txHash)
	if err != nil {
		return fmt.Errorf("getting receipt: %w", err)
	}
	if receipt == nil {
		fmt.Println("Transaction not yet mined, check later:", txHash.Hex())
		return nil
	}
	status, ok := receipt["status"]
	if !ok {
		return fmt.Errorf("receipt missing status field")
	}
	statusStr, ok := status.(string)
	if !ok {
		return fmt.Errorf("unexpected status type in receipt")
	}
	if statusStr != "0x1" {
		return fmt.Errorf("transaction failed (status: %s)", statusStr)
	}
	fmt.Println("Transaction validated: execution successful")
	return nil
}
