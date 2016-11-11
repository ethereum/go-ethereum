// Copyright 2015 The go-ethereum Authors
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

// Package ethapi implements the general Ethereum API functions.
package ethapi

import (
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/net/context"
)

// Backend defines most methods required for the RPC API.
type Backend interface {
	ethereum.ChainReader
	ethereum.ChainStateReader
	ethereum.ChainSyncReader
	ethereum.TransactionSender
	ethereum.GasPricer
	ethereum.GasEstimator
	ethereum.ContractCaller

	ProtocolVersion() int
	AccountManager() *accounts.Manager // TODO(fjl): this should be a constructor argb
	BlockTD(common.Hash) *big.Int
	RemoveTransaction(txhash common.Hash)
	PendingTransactions() []*types.Transaction
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	ResetHeadBlock(number uint64) // for admin API
}

// PendingState is implemented by the eth.Ethereum backend and provides access to optional
// features that can only be provided by a full pending state.
type PendingState interface {
	PendingBlock() (*types.Block, error)
	ethereum.PendingStateReader
	ethereum.PendingContractCaller
}

type TransactionInclusionBlock interface {
	// returns the block at which the given transaction was included in the blockchain
	TransactionInclusionBlock(txhash common.Hash) (blockhash common.Hash, blocknum uint64, index int, err error)
}

func GetAPIs(apiBackend Backend, solcPath string) []rpc.API {
	compiler := makeCompilerAPIs(solcPath)
	all := []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicEthereumAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicBlockChainAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicTransactionPoolAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(apiBackend),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicAccountAPI(apiBackend.AccountManager()),
			Public:    true,
		}, {
			Namespace: "personal",
			Version:   "1.0",
			Service:   NewPrivateAccountAPI(apiBackend),
			Public:    false,
		},
	}
	return append(compiler, all...)
}
