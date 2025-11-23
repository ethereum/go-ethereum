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

package grpc

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
)

// Backend defines the necessary methods from the Ethereum backend for the gRPC server.
type Backend interface {
	ChainConfig() *params.ChainConfig
	BlockChain() *core.BlockChain
	StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error)
	Miner() *miner.Miner
}

// TraderServer implements the TraderServiceServer interface.
type TraderServer struct {
	UnimplementedTraderServiceServer
	backend Backend
}

// NewTraderServer creates a new TraderServer instance.
func NewTraderServer(backend Backend) *TraderServer {
	return &TraderServer{backend: backend}
}

// SimulateBundle simulates bundle execution and returns results.
func (s *TraderServer) SimulateBundle(ctx context.Context, req *SimulateBundleRequest) (*SimulateBundleResponse, error) {
	if len(req.Transactions) == 0 {
		return nil, errors.New("bundle cannot be empty")
	}

	// Convert protobuf transactions to types.Transaction
	txs := make([]*types.Transaction, len(req.Transactions))
	for i, rawTx := range req.Transactions {
		tx := new(types.Transaction)
		if err := tx.UnmarshalBinary(rawTx); err != nil {
			return nil, fmt.Errorf("failed to decode transaction %d: %w", i, err)
		}
		txs[i] = tx
	}

	// Create bundle
	var targetBlock uint64 = 0
	if req.TargetBlock != nil {
		targetBlock = *req.TargetBlock
	}
	
	var minTs, maxTs uint64
	if req.MinTimestamp != nil {
		minTs = *req.MinTimestamp
	}
	if req.MaxTimestamp != nil {
		maxTs = *req.MaxTimestamp
	}
	
	revertingIndices := make([]int, len(req.RevertingTxs))
	for i, idx := range req.RevertingTxs {
		revertingIndices[i] = int(idx)
	}
	
	bundle := &miner.Bundle{
		Txs:          txs,
		MinTimestamp: minTs,
		MaxTimestamp: maxTs,
		RevertingTxs: revertingIndices,
		TargetBlock:  targetBlock,
	}

	// Get current block header for simulation
	header := s.backend.BlockChain().CurrentBlock()
	if header == nil {
		return nil, errors.New("current block not found")
	}

	// Simulate bundle  
	result, err := s.backend.Miner().SimulateBundle(bundle, header)
	if err != nil {
		return nil, fmt.Errorf("bundle simulation failed: %w", err)
	}

	// Convert result to protobuf
	pbResult := &SimulateBundleResponse{
		Success:         result.Success,
		GasUsed:         result.GasUsed,
		Profit:          result.Profit.Bytes(),
		CoinbaseBalance: result.CoinbaseBalance.Bytes(),
		TxResults:       make([]*TxSimulationResult, len(result.TxResults)),
	}

	for i, txRes := range result.TxResults {
		errStr := ""
		if txRes.Error != nil {
			errStr = txRes.Error.Error()
		}
		
		pbResult.TxResults[i] = &TxSimulationResult{
			Success:     txRes.Success,
			GasUsed:     txRes.GasUsed,
			Error:       errStr,
			ReturnValue: txRes.ReturnValue,
		}
		
		if !txRes.Success {
			pbResult.FailedTxIndex = int32(i)
			pbResult.FailedTxError = errStr
		}
	}

	return pbResult, nil
}

// SubmitBundle submits a bundle for inclusion in future blocks.
func (s *TraderServer) SubmitBundle(ctx context.Context, req *SubmitBundleRequest) (*SubmitBundleResponse, error) {
	if len(req.Transactions) == 0 {
		return nil, errors.New("bundle cannot be empty")
	}

	// Convert protobuf transactions to types.Transaction
	txs := make([]*types.Transaction, len(req.Transactions))
	for i, rawTx := range req.Transactions {
		tx := new(types.Transaction)
		if err := tx.UnmarshalBinary(rawTx); err != nil {
			return nil, fmt.Errorf("failed to decode transaction %d: %w", i, err)
		}
		txs[i] = tx
	}

	// Create bundle
	var targetBlock uint64 = 0
	if req.TargetBlock != nil {
		targetBlock = *req.TargetBlock
	}
	
	var minTs, maxTs uint64
	if req.MinTimestamp != nil {
		minTs = *req.MinTimestamp
	}
	if req.MaxTimestamp != nil {
		maxTs = *req.MaxTimestamp
	}
	
	revertingIndices := make([]int, len(req.RevertingTxs))
	for i, idx := range req.RevertingTxs {
		revertingIndices[i] = int(idx)
	}
	
	bundle := &miner.Bundle{
		Txs:          txs,
		MinTimestamp: minTs,
		MaxTimestamp: maxTs,
		RevertingTxs: revertingIndices,
		TargetBlock:  targetBlock,
	}

	// Add bundle to miner
	if err := s.backend.Miner().AddBundle(bundle); err != nil {
		return nil, fmt.Errorf("failed to add bundle to miner: %w", err)
	}

	// Create hash from bundle transactions
	bundleHash := types.DeriveSha(types.Transactions(bundle.Txs), trie.NewStackTrie(nil))
	
	return &SubmitBundleResponse{
		BundleHash: bundleHash.Bytes(),
	}, nil
}

// GetStorageBatch retrieves multiple storage slots in a single call.
func (s *TraderServer) GetStorageBatch(ctx context.Context, req *GetStorageBatchRequest) (*GetStorageBatchResponse, error) {
	if len(req.Contract) != common.AddressLength {
		return nil, errors.New("invalid contract address")
	}
	if len(req.Slots) == 0 {
		return nil, errors.New("no storage slots provided")
	}

	addr := common.BytesToAddress(req.Contract)

	// Get state at specified block
	var blockNrOrHash rpc.BlockNumberOrHash
	if req.BlockNumber != nil {
		blockNrOrHash = rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(*req.BlockNumber))
	} else {
		blockNrOrHash = rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
	}

	stateDB, _, err := s.backend.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get state for block: %w", err)
	}
	if stateDB == nil {
		return nil, errors.New("state not found for block")
	}

	// Batch read storage slots
	values := make([][]byte, len(req.Slots))
	for i, keyBytes := range req.Slots {
		if len(keyBytes) != common.HashLength {
			return nil, fmt.Errorf("invalid storage key length at index %d: %d", i, len(keyBytes))
		}
		key := common.BytesToHash(keyBytes)
		value := stateDB.GetState(addr, key)
		values[i] = value.Bytes()
	}

	return &GetStorageBatchResponse{Values: values}, nil
}

// GetPendingTransactions returns currently pending transactions.
func (s *TraderServer) GetPendingTransactions(ctx context.Context, req *GetPendingTransactionsRequest) (*GetPendingTransactionsResponse, error) {
	// Get pending transactions from miner
	pending, _, _ := s.backend.Miner().Pending()
	if pending == nil {
		return &GetPendingTransactionsResponse{Transactions: [][]byte{}}, nil
	}

	// Filter by gas price if requested
	var minGasPrice *big.Int
	if req.MinGasPrice != nil {
		minGasPrice = new(big.Int).SetUint64(*req.MinGasPrice)
	}

	// Collect and encode transactions
	var encodedTxs [][]byte
	for _, tx := range pending.Transactions() {
		if minGasPrice != nil && tx.GasPrice().Cmp(minGasPrice) < 0 {
			continue
		}
		
		encoded, err := tx.MarshalBinary()
		if err != nil {
			log.Warn("Failed to encode pending transaction", "hash", tx.Hash(), "err", err)
			continue
		}
		encodedTxs = append(encodedTxs, encoded)
	}

	return &GetPendingTransactionsResponse{Transactions: encodedTxs}, nil
}

// CallContract executes a contract call.
func (s *TraderServer) CallContract(ctx context.Context, req *CallContractRequest) (*CallContractResponse, error) {
	var (
		from common.Address
		to   *common.Address
	)
	if len(req.From) > 0 {
		from = common.BytesToAddress(req.From)
	}
	if len(req.To) > 0 {
		t := common.BytesToAddress(req.To)
		to = &t
	}

	value := new(big.Int)
	if len(req.Value) > 0 {
		value.SetBytes(req.Value)
	}

	gasPrice := new(big.Int)
	if req.GasPrice != nil {
		gasPrice.SetUint64(*req.GasPrice)
	}

	gas := uint64(100000000) // Default gas limit
	if req.Gas != nil {
		gas = *req.Gas
	}

	msg := &core.Message{
		From:     from,
		To:       to,
		Value:    value,
		GasLimit: gas,
		GasPrice: gasPrice,
		Data:     req.Data,
	}

	// Get state at specified block
	var blockNrOrHash rpc.BlockNumberOrHash
	if req.BlockNumber != nil {
		blockNrOrHash = rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(*req.BlockNumber))
	} else {
		blockNrOrHash = rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
	}

	stateDB, header, err := s.backend.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get state and header: %w", err)
	}
	if stateDB == nil || header == nil {
		return nil, errors.New("state or header not found")
	}

	// Create EVM and execute call
	blockContext := core.NewEVMBlockContext(header, s.backend.BlockChain(), nil)
	vmConfig := vm.Config{}

	evm := vm.NewEVM(blockContext, stateDB, s.backend.ChainConfig(), vmConfig)

	gasPool := new(core.GasPool).AddGas(gas)
	execResult, err := core.ApplyMessage(evm, msg, gasPool)

	resp := &CallContractResponse{
		ReturnData: execResult.ReturnData,
		GasUsed:    execResult.UsedGas,
		Success:    !execResult.Failed(),
	}
	if err != nil {
		resp.Error = err.Error()
	} else if execResult.Failed() {
		resp.Error = execResult.Err.Error()
	}

	return resp, nil
}

