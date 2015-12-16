// Copyright 2016 The go-ethereum Authors
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

package eth

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/net/context"
)

// ContractBackend implements bind.ContractBackend with direct calls to Ethereum
// internals to support operating on contracts within subprotocols like eth and
// swarm.
//
// Internally this backend uses the already exposed API endpoints of the Ethereum
// object. These should be rewritten to internal Go method calls when the Go API
// is refactored to support a clean library use.
type ContractBackend struct {
	eapi  *ethapi.PublicEthereumAPI        // Wrapper around the Ethereum object to access metadata
	bcapi *ethapi.PublicBlockChainAPI      // Wrapper around the blockchain to access chain data
	txapi *ethapi.PublicTransactionPoolAPI // Wrapper around the transaction pool to access transaction data
}

// NewContractBackend creates a new native contract backend using an existing
// Etheruem object.
func NewContractBackend(eth *FullNodeService) *ContractBackend {
	return &ContractBackend{
		eapi:  ethapi.NewPublicEthereumAPI(eth.apiBackend, nil, nil),
		bcapi: ethapi.NewPublicBlockChainAPI(eth.apiBackend),
		txapi: ethapi.NewPublicTransactionPoolAPI(eth.apiBackend),
	}
}

// HasCode implements bind.ContractVerifier.HasCode by retrieving any code associated
// with the contract from the local API, and checking its size.
func (b *ContractBackend) HasCode(ctx context.Context, contract common.Address, pending bool) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	block := rpc.LatestBlockNumber
	if pending {
		block = rpc.PendingBlockNumber
	}
	out, err := b.bcapi.GetCode(ctx, contract, block)
	return len(common.FromHex(out)) > 0, err
}

// ContractCall implements bind.ContractCaller executing an Ethereum contract
// call with the specified data as the input. The pending flag requests execution
// against the pending block, not the stable head of the chain.
func (b *ContractBackend) ContractCall(ctx context.Context, contract common.Address, data []byte, pending bool) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	// Convert the input args to the API spec
	args := ethapi.CallArgs{
		To:   &contract,
		Data: common.ToHex(data),
	}
	block := rpc.LatestBlockNumber
	if pending {
		block = rpc.PendingBlockNumber
	}
	// Execute the call and convert the output back to Go types
	out, err := b.bcapi.Call(ctx, args, block)
	return common.FromHex(out), err
}

// PendingAccountNonce implements bind.ContractTransactor retrieving the current
// pending nonce associated with an account.
func (b *ContractBackend) PendingAccountNonce(ctx context.Context, account common.Address) (uint64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	out, err := b.txapi.GetTransactionCount(ctx, account, rpc.PendingBlockNumber)
	return out.Uint64(), err
}

// SuggestGasPrice implements bind.ContractTransactor retrieving the currently
// suggested gas price to allow a timely execution of a transaction.
func (b *ContractBackend) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return b.eapi.GasPrice(ctx)
}

// EstimateGasLimit implements bind.ContractTransactor triing to estimate the gas
// needed to execute a specific transaction based on the current pending state of
// the backend blockchain. There is no guarantee that this is the true gas limit
// requirement as other transactions may be added or removed by miners, but it
// should provide a basis for setting a reasonable default.
func (b *ContractBackend) EstimateGasLimit(ctx context.Context, sender common.Address, contract *common.Address, value *big.Int, data []byte) (*big.Int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	out, err := b.bcapi.EstimateGas(ctx, ethapi.CallArgs{
		From:  sender,
		To:    contract,
		Value: *rpc.NewHexNumber(value),
		Data:  common.ToHex(data),
	})
	return out.BigInt(), err
}

// SendTransaction implements bind.ContractTransactor injects the transaction
// into the pending pool for execution.
func (b *ContractBackend) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	if ctx == nil {
		ctx = context.Background()
	}
	raw, _ := rlp.EncodeToBytes(tx)
	_, err := b.txapi.SendRawTransaction(ctx, common.ToHex(raw))
	return err
}
