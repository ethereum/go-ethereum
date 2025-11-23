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

package miner

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// API exposes miner-related methods for the RPC interface.
type API struct {
	miner *Miner
}

// NewAPI creates a new API instance.
func NewAPI(miner *Miner) *API {
	return &API{miner: miner}
}

// BundleArgs represents arguments for bundle submission.
type BundleArgs struct {
	Txs          []hexutil.Bytes `json:"txs"`
	MinTimestamp *hexutil.Uint64 `json:"minTimestamp,omitempty"`
	MaxTimestamp *hexutil.Uint64 `json:"maxTimestamp,omitempty"`
	RevertingTxs []int           `json:"revertingTxs,omitempty"`
	TargetBlock  *hexutil.Uint64 `json:"targetBlock,omitempty"`
}

// BundleSimulationResponse represents the result of a bundle simulation.
type BundleSimulationResponse struct {
	Success         bool                   `json:"success"`
	GasUsed         hexutil.Uint64         `json:"gasUsed"`
	Profit          *hexutil.Big           `json:"profit"`
	CoinbaseBalance *hexutil.Big           `json:"coinbaseBalance"`
	FailedTxIndex   int                    `json:"failedTxIndex,omitempty"`
	FailedTxError   string                 `json:"failedTxError,omitempty"`
	TxResults       []TxSimulationResponse `json:"txResults"`
}

// TxSimulationResponse represents the result of a transaction simulation.
type TxSimulationResponse struct {
	Success     bool           `json:"success"`
	GasUsed     hexutil.Uint64 `json:"gasUsed"`
	Error       string         `json:"error,omitempty"`
	ReturnValue hexutil.Bytes  `json:"returnValue,omitempty"`
}

// SubmitBundle submits a bundle for inclusion in future blocks.
func (api *API) SubmitBundle(ctx context.Context, args BundleArgs) (common.Hash, error) {
	if len(args.Txs) == 0 {
		return common.Hash{}, errors.New("bundle must contain at least one transaction")
	}

	// Decode transactions
	txs := make([]*types.Transaction, len(args.Txs))
	for i, encodedTx := range args.Txs {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(encodedTx); err != nil {
			return common.Hash{}, err
		}
		txs[i] = &tx
	}

	// Create bundle
	bundle := &Bundle{
		Txs:          txs,
		RevertingTxs: args.RevertingTxs,
	}

	if args.MinTimestamp != nil {
		bundle.MinTimestamp = uint64(*args.MinTimestamp)
	}
	if args.MaxTimestamp != nil {
		bundle.MaxTimestamp = uint64(*args.MaxTimestamp)
	}
	if args.TargetBlock != nil {
		bundle.TargetBlock = uint64(*args.TargetBlock)
	}

	// Add bundle
	if err := api.miner.AddBundle(bundle); err != nil {
		return common.Hash{}, err
	}

	// Return hash of first transaction as bundle ID
	return txs[0].Hash(), nil
}

// SimulateBundle simulates a bundle and returns the result.
func (api *API) SimulateBundle(ctx context.Context, args BundleArgs) (*BundleSimulationResponse, error) {
	if len(args.Txs) == 0 {
		return nil, errors.New("bundle must contain at least one transaction")
	}

	// Decode transactions
	txs := make([]*types.Transaction, len(args.Txs))
	for i, encodedTx := range args.Txs {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(encodedTx); err != nil {
			return nil, err
		}
		txs[i] = &tx
	}

	// Create bundle
	bundle := &Bundle{
		Txs:          txs,
		RevertingTxs: args.RevertingTxs,
	}

	if args.MinTimestamp != nil {
		bundle.MinTimestamp = uint64(*args.MinTimestamp)
	}
	if args.MaxTimestamp != nil {
		bundle.MaxTimestamp = uint64(*args.MaxTimestamp)
	}

	// Get current header for simulation
	header := api.miner.chain.CurrentHeader()
	
	// Create simulation header based on current + 1
	simHeader := &types.Header{
		ParentHash: header.Hash(),
		Number:     new(big.Int).Add(header.Number, big.NewInt(1)),
		GasLimit:   header.GasLimit,
		Time:       header.Time + 12, // Assume 12 second block time
		BaseFee:    header.BaseFee,
	}

	// Simulate bundle
	result, err := api.miner.SimulateBundle(bundle, simHeader)
	if err != nil {
		return nil, err
	}

	// Convert result to response
	response := &BundleSimulationResponse{
		Success:         result.Success,
		GasUsed:         hexutil.Uint64(result.GasUsed),
		Profit:          (*hexutil.Big)(result.Profit),
		CoinbaseBalance: (*hexutil.Big)(result.CoinbaseBalance),
		FailedTxIndex:   result.FailedTxIndex,
		TxResults:       make([]TxSimulationResponse, len(result.TxResults)),
	}

	if result.FailedTxError != nil {
		response.FailedTxError = result.FailedTxError.Error()
	}

	for i, txResult := range result.TxResults {
		txResp := TxSimulationResponse{
			Success: txResult.Success,
			GasUsed: hexutil.Uint64(txResult.GasUsed),
		}
		if txResult.Error != nil {
			txResp.Error = txResult.Error.Error()
		}
		if txResult.ReturnValue != nil {
			txResp.ReturnValue = txResult.ReturnValue
		}
		response.TxResults[i] = txResp
	}

	return response, nil
}

// GetBundles returns currently pending bundles for a specific block number.
func (api *API) GetBundles(ctx context.Context, blockNumber hexutil.Uint64) (int, error) {
	bundles := api.miner.GetBundles(uint64(blockNumber))
	return len(bundles), nil
}

// ClearExpiredBundles removes bundles that are expired for the given block number.
func (api *API) ClearExpiredBundles(ctx context.Context, blockNumber hexutil.Uint64) error {
	api.miner.ClearExpiredBundles(uint64(blockNumber))
	return nil
}

