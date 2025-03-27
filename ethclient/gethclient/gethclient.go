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

// Package gethclient provides an RPC client for geth-specific APIs.
package gethclient

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"runtime/debug"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

// Client is a wrapper around rpc.Client that implements geth-specific functionality.
//
// If you want to use the standardized Ethereum RPC functionality, use ethclient.Client instead.
type Client struct {
	c *rpc.Client
}

// New creates a client that uses the given RPC client.
func New(c *rpc.Client) *Client {
	return &Client{c}
}

// CreateAccessList tries to create an access list for a specific transaction based on the
// current pending state of the blockchain.
func (ec *Client) CreateAccessList(ctx context.Context, msg ethereum.CallMsg) (*types.AccessList, uint64, string, error) {
	type accessListResult struct {
		Accesslist *types.AccessList `json:"accessList"`
		Error      string            `json:"error,omitempty"`
		GasUsed    hexutil.Uint64    `json:"gasUsed"`
	}
	var result accessListResult
	if err := ec.c.CallContext(ctx, &result, "eth_createAccessList", toCallArg(msg)); err != nil {
		return nil, 0, "", err
	}
	return result.Accesslist, uint64(result.GasUsed), result.Error, nil
}

// AccountResult is the result of a GetProof operation.
type AccountResult struct {
	Address      common.Address  `json:"address"`
	AccountProof []string        `json:"accountProof"`
	Balance      *big.Int        `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        uint64          `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	StorageProof []StorageResult `json:"storageProof"`
}

// StorageResult provides a proof for a key-value pair.
type StorageResult struct {
	Key   string   `json:"key"`
	Value *big.Int `json:"value"`
	Proof []string `json:"proof"`
}

// GetProof returns the account and storage values of the specified account including the Merkle-proof.
// The block number can be nil, in which case the value is taken from the latest known block.
func (ec *Client) GetProof(ctx context.Context, account common.Address, keys []string, blockNumber *big.Int) (*AccountResult, error) {
	type storageResult struct {
		Key   string       `json:"key"`
		Value *hexutil.Big `json:"value"`
		Proof []string     `json:"proof"`
	}

	type accountResult struct {
		Address      common.Address  `json:"address"`
		AccountProof []string        `json:"accountProof"`
		Balance      *hexutil.Big    `json:"balance"`
		CodeHash     common.Hash     `json:"codeHash"`
		Nonce        hexutil.Uint64  `json:"nonce"`
		StorageHash  common.Hash     `json:"storageHash"`
		StorageProof []storageResult `json:"storageProof"`
	}

	// Avoid keys being 'null'.
	if keys == nil {
		keys = []string{}
	}

	var res accountResult
	err := ec.c.CallContext(ctx, &res, "eth_getProof", account, keys, toBlockNumArg(blockNumber))
	if err != nil {
		return nil, err
	}
	// Turn hexutils back to normal data types
	storageResults := make([]StorageResult, 0, len(res.StorageProof))
	for _, st := range res.StorageProof {
		storageResults = append(storageResults, StorageResult{
			Key:   st.Key,
			Value: st.Value.ToInt(),
			Proof: st.Proof,
		})
	}
	result := AccountResult{
		Address:      res.Address,
		AccountProof: res.AccountProof,
		Balance:      res.Balance.ToInt(),
		Nonce:        uint64(res.Nonce),
		CodeHash:     res.CodeHash,
		StorageHash:  res.StorageHash,
		StorageProof: storageResults,
	}
	return &result, nil
}

// CallContract executes a message call transaction, which is directly executed in the VM
// of the node, but never mined into the blockchain.
//
// blockNumber selects the block height at which the call runs. It can be nil, in which
// case the code is taken from the latest known block. Note that state from very old
// blocks might not be available.
//
// overrides specifies a map of contract states that should be overwritten before executing
// the message call.
// Please use ethclient.CallContract instead if you don't need the override functionality.
func (ec *Client) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int, overrides *map[common.Address]OverrideAccount) ([]byte, error) {
	var hex hexutil.Bytes
	err := ec.c.CallContext(
		ctx, &hex, "eth_call", toCallArg(msg),
		toBlockNumArg(blockNumber), overrides,
	)
	return hex, err
}

// CallContractWithBlockOverrides executes a message call transaction, which is directly executed
// in the VM  of the node, but never mined into the blockchain.
//
// blockNumber selects the block height at which the call runs. It can be nil, in which
// case the code is taken from the latest known block. Note that state from very old
// blocks might not be available.
//
// overrides specifies a map of contract states that should be overwritten before executing
// the message call.
//
// blockOverrides specifies block fields exposed to the EVM that can be overridden for the call.
//
// Please use ethclient.CallContract instead if you don't need the override functionality.
func (ec *Client) CallContractWithBlockOverrides(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int, overrides *map[common.Address]OverrideAccount, blockOverrides BlockOverrides) ([]byte, error) {
	var hex hexutil.Bytes
	err := ec.c.CallContext(
		ctx, &hex, "eth_call", toCallArg(msg),
		toBlockNumArg(blockNumber), overrides, blockOverrides,
	)
	return hex, err
}

// GCStats retrieves the current garbage collection stats from a geth node.
func (ec *Client) GCStats(ctx context.Context) (*debug.GCStats, error) {
	var result debug.GCStats
	err := ec.c.CallContext(ctx, &result, "debug_gcStats")
	return &result, err
}

// MemStats retrieves the current memory stats from a geth node.
func (ec *Client) MemStats(ctx context.Context) (*runtime.MemStats, error) {
	var result runtime.MemStats
	err := ec.c.CallContext(ctx, &result, "debug_memStats")
	return &result, err
}

// SetHead sets the current head of the local chain by block number.
// Note, this is a destructive action and may severely damage your chain.
// Use with extreme caution.
func (ec *Client) SetHead(ctx context.Context, number *big.Int) error {
	return ec.c.CallContext(ctx, nil, "debug_setHead", toBlockNumArg(number))
}

// GetNodeInfo retrieves the node info of a geth node.
func (ec *Client) GetNodeInfo(ctx context.Context) (*p2p.NodeInfo, error) {
	var result p2p.NodeInfo
	err := ec.c.CallContext(ctx, &result, "admin_nodeInfo")
	return &result, err
}

// SubscribeFullPendingTransactions subscribes to new pending transactions.
func (ec *Client) SubscribeFullPendingTransactions(ctx context.Context, ch chan<- *types.Transaction) (*rpc.ClientSubscription, error) {
	return ec.c.EthSubscribe(ctx, ch, "newPendingTransactions", true)
}

// SubscribePendingTransactions subscribes to new pending transaction hashes.
func (ec *Client) SubscribePendingTransactions(ctx context.Context, ch chan<- common.Hash) (*rpc.ClientSubscription, error) {
	return ec.c.EthSubscribe(ctx, ch, "newPendingTransactions")
}

// TxTraceResult is the result of a single transaction trace.
type TxTraceResult struct {
	TxHash common.Hash `json:"txHash"`           // Transaction hash
	Result interface{} `json:"result,omitempty"` // Trace results produced by the tracer
	Error  string      `json:"error,omitempty"`  // Trace failure produced by the tracer
}

// BlockTraceResult represents the results of tracing a single block.
type BlockTraceResult struct {
	Block  hexutil.Uint64   `json:"block"`  // Block number corresponding to this trace
	Hash   common.Hash      `json:"hash"`   // Block hash corresponding to this trace
	Traces []*TxTraceResult `json:"traces"` // Trace results produced by the task
}

// TraceTransaction returns the structured logs created during the execution of EVM
// and returns them as a JSON object.
func (ec *Client) TraceTransaction(ctx context.Context, txHash common.Hash, config *tracers.TraceConfig) (interface{}, error) {
	var result interface{}
	err := ec.c.CallContext(ctx, &result, "debug_traceTransaction", txHash, config)
	return result, err
}

// TraceCall lets you trace a given eth_call. It collects the structured logs
// created during the execution of EVM if the given transaction was added on
// top of the provided block and returns them as a JSON object.
func (ec *Client) TraceCall(ctx context.Context, msg ethereum.CallMsg, blockNrOrHash rpc.BlockNumberOrHash, config *tracers.TraceCallConfig) (interface{}, error) {
	var result interface{}
	err := ec.c.CallContext(ctx, &result, "debug_traceCall", toCallArg(msg), blockNrOrHash, config)
	return result, err
}

// TraceBlock returns the structured logs created during the execution of EVM
// for a specific block. This can be by block number, hash, or the latest/pending block.
func (ec *Client) TraceBlock(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, config *tracers.TraceConfig) ([]*TxTraceResult, error) {
	var result []*TxTraceResult

	// Determine the method to call based on the type of blockNrOrHash
	var method string
	var arg interface{}

	if hash, ok := blockNrOrHash.Hash(); ok {
		method = "debug_traceBlockByHash"
		arg = hash
	} else if number, ok := blockNrOrHash.Number(); ok {
		method = "debug_traceBlockByNumber"
		arg = number
	} else {
		return nil, errors.New("invalid arguments; neither block nor hash specified")
	}

	err := ec.c.CallContext(ctx, &result, method, arg, config)
	return result, err
}

// TraceChain returns the structured logs created during the execution of EVM between
// two blocks (inclusive) and returns them as an array of JSON objects.
func (ec *Client) TraceChain(ctx context.Context, start, end rpc.BlockNumber, config *tracers.TraceConfig) (*rpc.ClientSubscription, error) {
	return ec.c.EthSubscribe(ctx, make(chan *BlockTraceResult), "debug_traceChain", start, end, config)
}

// CallTracerResult contains the results of a call tracer.
type CallTracerResult struct {
	Type    string             `json:"type"`
	From    common.Address     `json:"from"`
	To      common.Address     `json:"to"`
	Value   *hexutil.Big       `json:"value"`
	Gas     hexutil.Uint64     `json:"gas"`
	GasUsed hexutil.Uint64     `json:"gasUsed"`
	Input   hexutil.Bytes      `json:"input"`
	Output  hexutil.Bytes      `json:"output"`
	Error   string             `json:"error,omitempty"`
	Calls   []CallTracerResult `json:"calls,omitempty"`
}

// TraceCallWithCallTracer traces a call with the call tracer, which provides
// detailed information about nested calls.
func (ec *Client) TraceCallWithCallTracer(ctx context.Context, msg ethereum.CallMsg, blockNrOrHash rpc.BlockNumberOrHash) (*CallTracerResult, error) {
	callTracer := "callTracer"
	config := &tracers.TraceCallConfig{
		TraceConfig: tracers.TraceConfig{
			Tracer: &callTracer,
		},
	}

	var result CallTracerResult
	err := ec.c.CallContext(ctx, &result, "debug_traceCall", toCallArg(msg), blockNrOrHash, config)
	return &result, err
}

// TraceTransactionWithCallTracer traces a transaction with the call tracer,
// which provides detailed information about nested calls.
func (ec *Client) TraceTransactionWithCallTracer(ctx context.Context, txHash common.Hash) (*CallTracerResult, error) {
	callTracer := "callTracer"
	config := &tracers.TraceConfig{
		Tracer: &callTracer,
	}

	var result CallTracerResult
	err := ec.c.CallContext(ctx, &result, "debug_traceTransaction", txHash, config)
	return &result, err
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	if number.Sign() >= 0 {
		return hexutil.EncodeBig(number)
	}
	// It's negative.
	if number.IsInt64() {
		return rpc.BlockNumber(number.Int64()).String()
	}
	// It's negative and large, which is invalid.
	return fmt.Sprintf("<invalid %d>", number)
}

func toCallArg(msg ethereum.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["input"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	if msg.GasFeeCap != nil {
		arg["maxFeePerGas"] = (*hexutil.Big)(msg.GasFeeCap)
	}
	if msg.GasTipCap != nil {
		arg["maxPriorityFeePerGas"] = (*hexutil.Big)(msg.GasTipCap)
	}
	if msg.AccessList != nil {
		arg["accessList"] = msg.AccessList
	}
	if msg.BlobGasFeeCap != nil {
		arg["maxFeePerBlobGas"] = (*hexutil.Big)(msg.BlobGasFeeCap)
	}
	if msg.BlobHashes != nil {
		arg["blobVersionedHashes"] = msg.BlobHashes
	}
	if msg.AuthorizationList != nil {
		arg["authorizationList"] = msg.AuthorizationList
	}
	return arg
}

// OverrideAccount is an alias for ethereum.OverrideAccount.
type OverrideAccount = ethereum.OverrideAccount

// BlockOverrides is an alias for ethereum.BlockOverrides.
type BlockOverrides = ethereum.BlockOverrides
