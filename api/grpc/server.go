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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
)

// Backend defines the interface for accessing blockchain data.
type Backend interface {
	BlockChain() *core.BlockChain
	TxPool() *core.TxPool
	Miner() *miner.Miner
	ChainConfig() *params.ChainConfig
	CurrentHeader() *types.Header
	StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error)
	RPCGasCap() uint64
}

// TraderServer implements the gRPC trader service.
type TraderServer struct {
	backend Backend
	config  *params.ChainConfig
}

// NewTraderServer creates a new gRPC trader server.
func NewTraderServer(eth *eth.Ethereum) *TraderServer {
	return &TraderServer{
		backend: eth,
		config:  eth.BlockChain().Config(),
	}
}

// SimulateBundle simulates a bundle and returns detailed results.
func (s *TraderServer) SimulateBundle(ctx context.Context, req *SimulateBundleRequest) (*SimulateBundleResponse, error) {
	if len(req.Transactions) == 0 {
		return nil, errors.New("bundle must contain at least one transaction")
	}

	// Decode transactions
	txs := make([]*types.Transaction, len(req.Transactions))
	for i, encodedTx := range req.Transactions {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(encodedTx); err != nil {
			return nil, err
		}
		txs[i] = &tx
	}

	// Create bundle
	bundle := &miner.Bundle{
		Txs:          txs,
		RevertingTxs: make([]int, len(req.RevertingTxs)),
	}

	for i, idx := range req.RevertingTxs {
		bundle.RevertingTxs[i] = int(idx)
	}

	if req.MinTimestamp != nil {
		bundle.MinTimestamp = *req.MinTimestamp
	}
	if req.MaxTimestamp != nil {
		bundle.MaxTimestamp = *req.MaxTimestamp
	}

	// Get simulation header
	currentHeader := s.backend.CurrentHeader()
	simHeader := &types.Header{
		ParentHash: currentHeader.Hash(),
		Number:     new(big.Int).Add(currentHeader.Number, big.NewInt(1)),
		GasLimit:   currentHeader.GasLimit,
		Time:       currentHeader.Time + 12,
		BaseFee:    currentHeader.BaseFee,
	}

	// Simulate
	result, err := s.backend.Miner().SimulateBundle(bundle, simHeader)
	if err != nil {
		return nil, err
	}

	// Convert to protobuf response
	response := &SimulateBundleResponse{
		Success:         result.Success,
		GasUsed:         result.GasUsed,
		Profit:          result.Profit.Bytes(),
		CoinbaseBalance: result.CoinbaseBalance.Bytes(),
		FailedTxIndex:   int32(result.FailedTxIndex),
		TxResults:       make([]*TxSimulationResult, len(result.TxResults)),
	}

	if result.FailedTxError != nil {
		response.FailedTxError = result.FailedTxError.Error()
	}

	for i, txResult := range result.TxResults {
		pbResult := &TxSimulationResult{
			Success: txResult.Success,
			GasUsed: txResult.GasUsed,
		}
		if txResult.Error != nil {
			pbResult.Error = txResult.Error.Error()
		}
		if txResult.ReturnValue != nil {
			pbResult.ReturnValue = txResult.ReturnValue
		}
		response.TxResults[i] = pbResult
	}

	return response, nil
}

// SubmitBundle submits a bundle for inclusion in future blocks.
func (s *TraderServer) SubmitBundle(ctx context.Context, req *SubmitBundleRequest) (*SubmitBundleResponse, error) {
	if len(req.Transactions) == 0 {
		return nil, errors.New("bundle must contain at least one transaction")
	}

	// Decode transactions
	txs := make([]*types.Transaction, len(req.Transactions))
	for i, encodedTx := range req.Transactions {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(encodedTx); err != nil {
			return nil, err
		}
		txs[i] = &tx
	}

	// Create bundle
	bundle := &miner.Bundle{
		Txs:          txs,
		RevertingTxs: make([]int, len(req.RevertingTxs)),
	}

	for i, idx := range req.RevertingTxs {
		bundle.RevertingTxs[i] = int(idx)
	}

	if req.MinTimestamp != nil {
		bundle.MinTimestamp = *req.MinTimestamp
	}
	if req.MaxTimestamp != nil {
		bundle.MaxTimestamp = *req.MaxTimestamp
	}
	if req.TargetBlock != nil {
		bundle.TargetBlock = *req.TargetBlock
	}

	// Add bundle
	if err := s.backend.Miner().AddBundle(bundle); err != nil {
		return nil, err
	}

	return &SubmitBundleResponse{
		BundleHash: txs[0].Hash().Bytes(),
	}, nil
}

// GetStorageBatch retrieves multiple storage slots efficiently.
func (s *TraderServer) GetStorageBatch(ctx context.Context, req *GetStorageBatchRequest) (*GetStorageBatchResponse, error) {
	if len(req.Contract) != 20 {
		return nil, errors.New("invalid contract address")
	}

	contract := common.BytesToAddress(req.Contract)
	blockNr := rpc.LatestBlockNumber
	if req.BlockNumber != nil {
		blockNr = rpc.BlockNumber(*req.BlockNumber)
	}

	// Get state
	stateDB, _, err := s.backend.StateAndHeaderByNumber(ctx, blockNr)
	if err != nil {
		return nil, err
	}

	// Batch read storage
	values := make([][]byte, len(req.Slots))
	for i, slotBytes := range req.Slots {
		if len(slotBytes) != 32 {
			return nil, errors.New("invalid slot size")
		}
		slot := common.BytesToHash(slotBytes)
		value := stateDB.GetState(contract, slot)
		values[i] = value.Bytes()
	}

	return &GetStorageBatchResponse{
		Values: values,
	}, nil
}

// GetPendingTransactions returns pending transactions.
func (s *TraderServer) GetPendingTransactions(ctx context.Context, req *GetPendingTransactionsRequest) (*GetPendingTransactionsResponse, error) {
	// Get pending from txpool
	pending := s.backend.TxPool().Pending(core.PendingFilter{})

	var txs [][]byte
	for _, accountTxs := range pending {
		for _, ltx := range accountTxs {
			tx := ltx.Resolve()
			if tx == nil {
				continue
			}
			
			// Filter by min gas price if specified
			if req.MinGasPrice != nil {
				gasPrice := tx.GasPrice()
				if tx.Type() == types.DynamicFeeTxType {
					gasPrice = tx.GasFeeCap()
				}
				if gasPrice.Cmp(new(big.Int).SetUint64(*req.MinGasPrice)) < 0 {
					continue
				}
			}
			
			encoded, err := tx.MarshalBinary()
			if err != nil {
				log.Warn("Failed to encode transaction", "hash", tx.Hash(), "err", err)
				continue
			}
			txs = append(txs, encoded)
		}
	}

	return &GetPendingTransactionsResponse{
		Transactions: txs,
	}, nil
}

// CallContract executes a contract call.
func (s *TraderServer) CallContract(ctx context.Context, req *CallContractRequest) (*CallContractResponse, error) {
	if len(req.To) != 20 {
		return nil, errors.New("invalid contract address")
	}

	blockNr := rpc.LatestBlockNumber
	if req.BlockNumber != nil {
		blockNr = rpc.BlockNumber(*req.BlockNumber)
	}

	stateDB, header, err := s.backend.StateAndHeaderByNumber(ctx, blockNr)
	if err != nil {
		return nil, err
	}

	// Prepare message
	from := common.Address{}
	if len(req.From) == 20 {
		from = common.BytesToAddress(req.From)
	}
	to := common.BytesToAddress(req.To)

	gas := s.backend.RPCGasCap()
	if req.Gas != nil && *req.Gas > 0 {
		gas = *req.Gas
	}

	gasPrice := new(big.Int)
	if req.GasPrice != nil {
		gasPrice = new(big.Int).SetUint64(*req.GasPrice)
	} else if header.BaseFee != nil {
		gasPrice = header.BaseFee
	}

	value := new(big.Int)
	if req.Value != nil {
		value = new(big.Int).SetBytes(req.Value)
	}

	msg := &core.Message{
		From:              from,
		To:                &to,
		Value:             value,
		GasLimit:          gas,
		GasPrice:          gasPrice,
		GasFeeCap:         gasPrice,
		GasTipCap:         gasPrice,
		Data:              req.Data,
		SkipAccountChecks: true,
	}

	// Create EVM
	blockContext := core.NewEVMBlockContext(header, s.backend.BlockChain(), nil)
	txContext := core.NewEVMTxContext(msg)
	evm := vm.NewEVM(blockContext, txContext, stateDB, s.config, vm.Config{})

	// Execute
	result, err := core.ApplyMessage(evm, msg, new(core.GasPool).AddGas(gas))
	if err != nil {
		return &CallContractResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	response := &CallContractResponse{
		ReturnData: result.ReturnData,
		GasUsed:    result.UsedGas,
		Success:    !result.Failed(),
	}

	if result.Failed() {
		response.Error = result.Err.Error()
	}

	return response, nil
}

