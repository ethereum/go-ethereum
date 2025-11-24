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

// Package grpc provides an example client for the low-latency gRPC trading API.
// This example demonstrates how to:
// - Connect to the gRPC server
// - Submit transaction bundles
// - Simulate bundle execution
// - Perform batch storage reads
// - Subscribe to pending transactions
package grpc

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps the gRPC TraderService client with convenient methods.
type Client struct {
	conn   *grpc.ClientConn
	client TraderServiceClient
}

// NewClient creates a new gRPC client connected to the specified host and port.
func NewClient(host string, port int) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(100*1024*1024), // 100MB
			grpc.MaxCallSendMsgSize(100*1024*1024), // 100MB
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	return &Client{
		conn:   conn,
		client: NewTraderServiceClient(conn),
	}, nil
}

// Close closes the gRPC connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// SimulateBundle simulates a bundle of transactions and returns the results.
// This is useful for testing bundle profitability before submission.
func (c *Client) SimulateBundle(ctx context.Context, txs []*types.Transaction, opts *BundleOptions) (*SimulateBundleResponse, error) {
	// Encode transactions
	encodedTxs := make([][]byte, len(txs))
	for i, tx := range txs {
		encoded, err := tx.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("failed to encode transaction %d: %w", i, err)
		}
		encodedTxs[i] = encoded
	}

	req := &SimulateBundleRequest{
		Transactions: encodedTxs,
	}

	if opts != nil {
		req.MinTimestamp = opts.MinTimestamp
		req.MaxTimestamp = opts.MaxTimestamp
		req.TargetBlock = opts.TargetBlock
		req.RevertingTxs = opts.RevertingTxIndices
	}

	return c.client.SimulateBundle(ctx, req)
}

// SubmitBundle submits a bundle for inclusion in future blocks.
func (c *Client) SubmitBundle(ctx context.Context, txs []*types.Transaction, opts *BundleOptions) (common.Hash, error) {
	// Encode transactions
	encodedTxs := make([][]byte, len(txs))
	for i, tx := range txs {
		encoded, err := tx.MarshalBinary()
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to encode transaction %d: %w", i, err)
		}
		encodedTxs[i] = encoded
	}

	req := &SubmitBundleRequest{
		Transactions: encodedTxs,
	}

	if opts != nil {
		req.MinTimestamp = opts.MinTimestamp
		req.MaxTimestamp = opts.MaxTimestamp
		req.TargetBlock = opts.TargetBlock
		req.RevertingTxs = opts.RevertingTxIndices
	}

	resp, err := c.client.SubmitBundle(ctx, req)
	if err != nil {
		return common.Hash{}, err
	}

	return common.BytesToHash(resp.BundleHash), nil
}

// GetStorageBatch retrieves multiple storage slots in a single call.
// This is significantly faster than multiple eth_getStorageAt JSON-RPC calls.
func (c *Client) GetStorageBatch(ctx context.Context, contract common.Address, slots []common.Hash, blockNum *uint64) ([]common.Hash, error) {
	encodedSlots := make([][]byte, len(slots))
	for i, slot := range slots {
		encodedSlots[i] = slot.Bytes()
	}

	req := &GetStorageBatchRequest{
		Contract:    contract.Bytes(),
		Slots:       encodedSlots,
		BlockNumber: blockNum,
	}

	resp, err := c.client.GetStorageBatch(ctx, req)
	if err != nil {
		return nil, err
	}

	values := make([]common.Hash, len(resp.Values))
	for i, val := range resp.Values {
		values[i] = common.BytesToHash(val)
	}

	return values, nil
}

// GetPendingTransactions retrieves currently pending transactions.
func (c *Client) GetPendingTransactions(ctx context.Context, minGasPrice *uint64) ([]*types.Transaction, error) {
	req := &GetPendingTransactionsRequest{
		MinGasPrice: minGasPrice,
	}

	resp, err := c.client.GetPendingTransactions(ctx, req)
	if err != nil {
		return nil, err
	}

	txs := make([]*types.Transaction, 0, len(resp.Transactions))
	for i, encoded := range resp.Transactions {
		tx := new(types.Transaction)
		if err := tx.UnmarshalBinary(encoded); err != nil {
			return nil, fmt.Errorf("failed to decode transaction %d: %w", i, err)
		}
		txs = append(txs, tx)
	}

	return txs, nil
}

// CallContract executes a contract call.
func (c *Client) CallContract(ctx context.Context, msg *CallMessage, blockNum *uint64) (*CallContractResponse, error) {
	req := &CallContractRequest{
		From:        msg.From.Bytes(),
		To:          msg.To.Bytes(),
		Data:        msg.Data,
		BlockNumber: blockNum,
	}

	if msg.Gas != nil {
		req.Gas = msg.Gas
	}
	if msg.GasPrice != nil {
		req.GasPrice = msg.GasPrice
	}
	if msg.Value != nil {
		req.Value = msg.Value.Bytes()
	}

	return c.client.CallContract(ctx, req)
}

// BundleOptions contains optional parameters for bundle submission and simulation.
type BundleOptions struct {
	MinTimestamp        *uint64
	MaxTimestamp        *uint64
	TargetBlock         *uint64
	RevertingTxIndices  []int32
}

// CallMessage contains parameters for contract calls.
type CallMessage struct {
	From     common.Address
	To       common.Address
	Data     []byte
	Gas      *uint64
	GasPrice *uint64
	Value    *big.Int
}

// ExampleUsage demonstrates typical usage patterns for high-frequency trading.
func ExampleUsage() error {
	// Connect to gRPC server
	client, err := NewClient("localhost", 9090)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Example 1: Batch storage read (e.g., reading Uniswap pool reserves)
	poolAddress := common.HexToAddress("0x...")
	slot0 := common.HexToHash("0x0") // slot0 contains sqrtPriceX96 and tick
	slot1 := common.HexToHash("0x1") // Other pool data

	values, err := client.GetStorageBatch(ctx, poolAddress, []common.Hash{slot0, slot1}, nil)
	if err != nil {
		return fmt.Errorf("failed to get storage: %w", err)
	}
	fmt.Printf("Pool state: %v\n", values)

	// Example 2: Simulate a bundle before submission
	// (assuming you have prepared transactions)
	var txs []*types.Transaction // ... your transactions
	
	simResult, err := client.SimulateBundle(ctx, txs, &BundleOptions{
		TargetBlock: func() *uint64 { b := uint64(12345678); return &b }(),
	})
	if err != nil {
		return fmt.Errorf("failed to simulate bundle: %w", err)
	}

	if !simResult.Success {
		fmt.Printf("Bundle simulation failed at tx %d: %s\n", simResult.FailedTxIndex, simResult.FailedTxError)
		return nil
	}

	profit := new(big.Int).SetBytes(simResult.Profit)
	fmt.Printf("Bundle profit: %s wei, gas used: %d\n", profit.String(), simResult.GasUsed)

	// Example 3: Submit bundle if profitable
	if profit.Sign() > 0 {
		bundleHash, err := client.SubmitBundle(ctx, txs, &BundleOptions{
			TargetBlock: func() *uint64 { b := uint64(12345678); return &b }(),
		})
		if err != nil {
			return fmt.Errorf("failed to submit bundle: %w", err)
		}
		fmt.Printf("Bundle submitted: %s\n", bundleHash.Hex())
	}

	return nil
}

