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

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

// ContractBackend implements bind.ContractBackend with direct calls to Ethereum
// internals to support operating on contracts within subprotocols like eth and
// swarm.
//
// Internally this backend uses the already exposed API endpoints of the Ethereum
// object. These should be rewritten to internal Go method calls when the Go API
// is refactored to support a clean library use.
type ContractBackend struct {
	eapi  *PublicEthereumAPI        // Wrapper around the Ethereum object to access metadata
	bcapi *PublicBlockChainAPI      // Wrapper around the blockchain to access chain data
	txapi *PublicTransactionPoolAPI // Wrapper around the transaction pool to access transaction data
}

// NewContractBackend creates a new native contract backend using an existing
// Etheruem object.
func NewContractBackend(eth *Ethereum) *ContractBackend {
	return &ContractBackend{
		eapi:  NewPublicEthereumAPI(eth),
		bcapi: NewPublicBlockChainAPI(eth.chainConfig, eth.blockchain, eth.miner, eth.chainDb, eth.gpo, eth.eventMux, eth.accountManager),
		txapi: NewPublicTransactionPoolAPI(eth),
	}
}

// ContractCall implements bind.ContractCaller executing an Ethereum contract
// call with the specified data as the input. The pending flag requests execution
// against the pending block, not the stable head of the chain.
func (b *ContractBackend) ContractCall(contract common.Address, data []byte, pending bool) ([]byte, error) {
	// Convert the input args to the API spec
	args := CallArgs{
		To:   &contract,
		Data: common.ToHex(data),
	}
	block := rpc.LatestBlockNumber
	if pending {
		block = rpc.PendingBlockNumber
	}
	// Execute the call and convert the output back to Go types
	out, err := b.bcapi.Call(args, block)
	if err == errNoCode {
		err = bind.ErrNoCode
	}
	return common.FromHex(out), err
}

// PendingAccountNonce implements bind.ContractTransactor retrieving the current
// pending nonce associated with an account.
func (b *ContractBackend) PendingAccountNonce(account common.Address) (uint64, error) {
	out, err := b.txapi.GetTransactionCount(account, rpc.PendingBlockNumber)
	return out.Uint64(), err
}

// SuggestGasPrice implements bind.ContractTransactor retrieving the currently
// suggested gas price to allow a timely execution of a transaction.
func (b *ContractBackend) SuggestGasPrice() (*big.Int, error) {
	return b.eapi.GasPrice(), nil
}

// EstimateGasLimit implements bind.ContractTransactor triing to estimate the gas
// needed to execute a specific transaction based on the current pending state of
// the backend blockchain. There is no guarantee that this is the true gas limit
// requirement as other transactions may be added or removed by miners, but it
// should provide a basis for setting a reasonable default.
func (b *ContractBackend) EstimateGasLimit(sender common.Address, contract *common.Address, value *big.Int, data []byte) (*big.Int, error) {
	out, err := b.bcapi.EstimateGas(CallArgs{
		From:  sender,
		To:    contract,
		Value: *rpc.NewHexNumber(value),
		Data:  common.ToHex(data),
	})
	if err == errNoCode {
		err = bind.ErrNoCode
	}
	return out.BigInt(), err
}

// SendTransaction implements bind.ContractTransactor injects the transaction
// into the pending pool for execution.
func (b *ContractBackend) SendTransaction(tx *types.Transaction) error {
	raw, _ := rlp.EncodeToBytes(tx)
	_, err := b.txapi.SendRawTransaction(common.ToHex(raw))
	return err
}
