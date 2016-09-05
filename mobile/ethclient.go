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

// Contains a wrapper for the Ethereum client.

package geth

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethclient"
)

// EthereumClient provides access to the Ethereum APIs.
type EthereumClient struct {
	client *ethclient.Client
}

// NewEthereumClient connects a client to the given URL.
func NewEthereumClient(rawurl string) (*EthereumClient, error) {
	client, err := ethclient.Dial(rawurl)
	return &EthereumClient{client}, err
}

// GetBlockByHash returns the given full block.
func (ec *EthereumClient) GetBlockByHash(ctx *Context, hash *Hash) (*Block, error) {
	block, err := ec.client.BlockByHash(ctx.context, hash.hash)
	return &Block{block}, err
}

// GetBlockByNumber returns a block from the current canonical chain. If number is <0, the
// latest known block is returned.
func (ec *EthereumClient) GetBlockByNumber(ctx *Context, number int64) (*Block, error) {
	if number < 0 {
		block, err := ec.client.BlockByNumber(ctx.context, nil)
		return &Block{block}, err
	}
	block, err := ec.client.BlockByNumber(ctx.context, big.NewInt(number))
	return &Block{block}, err
}

// GetHeaderByHash returns the block header with the given hash.
func (ec *EthereumClient) GetHeaderByHash(ctx *Context, hash *Hash) (*Header, error) {
	header, err := ec.client.HeaderByHash(ctx.context, hash.hash)
	return &Header{header}, err
}

// GetHeaderByNumber returns a block header from the current canonical chain. If number is <0,
// the latest known header is returned.
func (ec *EthereumClient) GetHeaderByNumber(ctx *Context, number int64) (*Header, error) {
	if number < 0 {
		header, err := ec.client.HeaderByNumber(ctx.context, nil)
		return &Header{header}, err
	}
	header, err := ec.client.HeaderByNumber(ctx.context, big.NewInt(number))
	return &Header{header}, err
}

// GetTransactionByHash returns the transaction with the given hash.
func (ec *EthereumClient) GetTransactionByHash(ctx *Context, hash *Hash) (*Transaction, error) {
	tx, err := ec.client.TransactionByHash(ctx.context, hash.hash)
	return &Transaction{tx}, err
}

// GetTransactionCount returns the total number of transactions in the given block.
func (ec *EthereumClient) GetTransactionCount(ctx *Context, hash *Hash) (int, error) {
	count, err := ec.client.TransactionCount(ctx.context, hash.hash)
	return int(count), err
}

// GetTransactionInBlock returns a single transaction at index in the given block.
func (ec *EthereumClient) GetTransactionInBlock(ctx *Context, hash *Hash, index int) (*Transaction, error) {
	tx, err := ec.client.TransactionInBlock(ctx.context, hash.hash, uint(index))
	return &Transaction{tx}, err

}

// GetTransactionReceipt returns the receipt of a transaction by transaction hash.
// Note that the receipt is not available for pending transactions.
func (ec *EthereumClient) GetTransactionReceipt(ctx *Context, hash *Hash) (*Receipt, error) {
	receipt, err := ec.client.TransactionReceipt(ctx.context, hash.hash)
	return &Receipt{receipt}, err
}

// SyncProgress retrieves the current progress of the sync algorithm. If there's
// no sync currently running, it returns nil.
func (ec *EthereumClient) SyncProgress(ctx *Context) (*SyncProgress, error) {
	progress, err := ec.client.SyncProgress(ctx.context)
	if progress == nil {
		return nil, err
	}
	return &SyncProgress{*progress}, err
}

// NewHeadHandler is a client-side subscription callback to invoke on events and
// subscription failure.
type NewHeadHandler interface {
	OnNewHead(header *Header)
	OnError(failure string)
}

// SubscribeNewHead subscribes to notifications about the current blockchain head
// on the given channel.
func (ec *EthereumClient) SubscribeNewHead(ctx *Context, handler NewHeadHandler, buffer int) (*Subscription, error) {
	// Subscribe to the event internally
	ch := make(chan *types.Header, buffer)
	sub, err := ec.client.SubscribeNewHead(ctx.context, ch)
	if err != nil {
		return nil, err
	}
	// Start up a dispatcher to feed into the callback
	go func() {
		for {
			select {
			case header := <-ch:
				handler.OnNewHead(&Header{header})

			case err := <-sub.Err():
				handler.OnError(err.Error())
				return
			}
		}
	}()
	return &Subscription{sub}, nil
}

// State Access

// GetBalanceAt returns the wei balance of the given account.
// The block number can be <0, in which case the balance is taken from the latest known block.
func (ec *EthereumClient) GetBalanceAt(ctx *Context, account *Address, number int64) (*BigInt, error) {
	if number < 0 {
		balance, err := ec.client.BalanceAt(ctx.context, account.address, nil)
		return &BigInt{balance}, err
	}
	balance, err := ec.client.BalanceAt(ctx.context, account.address, big.NewInt(number))
	return &BigInt{balance}, err
}

// GetStorageAt returns the value of key in the contract storage of the given account.
// The block number can be <0, in which case the value is taken from the latest known block.
func (ec *EthereumClient) GetStorageAt(ctx *Context, account *Address, key *Hash, number int64) ([]byte, error) {
	if number < 0 {
		return ec.client.StorageAt(ctx.context, account.address, key.hash, nil)
	}
	return ec.client.StorageAt(ctx.context, account.address, key.hash, big.NewInt(number))
}

// GetCodeAt returns the contract code of the given account.
// The block number can be <0, in which case the code is taken from the latest known block.
func (ec *EthereumClient) GetCodeAt(ctx *Context, account *Address, number int64) ([]byte, error) {
	if number < 0 {
		return ec.client.CodeAt(ctx.context, account.address, nil)
	}
	return ec.client.CodeAt(ctx.context, account.address, big.NewInt(number))
}

// GetNonceAt returns the account nonce of the given account.
// The block number can be <0, in which case the nonce is taken from the latest known block.
func (ec *EthereumClient) GetNonceAt(ctx *Context, account *Address, number int64) (int64, error) {
	if number < 0 {
		nonce, err := ec.client.NonceAt(ctx.context, account.address, nil)
		return int64(nonce), err
	}
	nonce, err := ec.client.NonceAt(ctx.context, account.address, big.NewInt(number))
	return int64(nonce), err
}

// Filters

// FilterLogs executes a filter query.
func (ec *EthereumClient) FilterLogs(ctx *Context, query *FilterQuery) (*Logs, error) {
	logs, err := ec.client.FilterLogs(ctx.context, query.query)
	if err != nil {
		return nil, err
	}
	// Temp hack due to vm.Logs being []*vm.Log
	res := make(vm.Logs, len(logs))
	for i, log := range logs {
		res[i] = &log
	}
	return &Logs{res}, nil
}

// FilterLogsHandler is a client-side subscription callback to invoke on events and
// subscription failure.
type FilterLogsHandler interface {
	OnFilterLogs(log *Log)
	OnError(failure string)
}

// SubscribeFilterLogs subscribes to the results of a streaming filter query.
func (ec *EthereumClient) SubscribeFilterLogs(ctx *Context, query *FilterQuery, handler FilterLogsHandler, buffer int) (*Subscription, error) {
	// Subscribe to the event internally
	ch := make(chan vm.Log, buffer)
	sub, err := ec.client.SubscribeFilterLogs(ctx.context, query.query, ch)
	if err != nil {
		return nil, err
	}
	// Start up a dispatcher to feed into the callback
	go func() {
		for {
			select {
			case log := <-ch:
				handler.OnFilterLogs(&Log{&log})

			case err := <-sub.Err():
				handler.OnError(err.Error())
				return
			}
		}
	}()
	return &Subscription{sub}, nil
}

// Pending State

// GetPendingBalanceAt returns the wei balance of the given account in the pending state.
func (ec *EthereumClient) GetPendingBalanceAt(ctx *Context, account *Address) (*BigInt, error) {
	balance, err := ec.client.PendingBalanceAt(ctx.context, account.address)
	return &BigInt{balance}, err
}

// GetPendingStorageAt returns the value of key in the contract storage of the given account in the pending state.
func (ec *EthereumClient) GetPendingStorageAt(ctx *Context, account *Address, key *Hash) ([]byte, error) {
	return ec.client.PendingStorageAt(ctx.context, account.address, key.hash)
}

// GetPendingCodeAt returns the contract code of the given account in the pending state.
func (ec *EthereumClient) GetPendingCodeAt(ctx *Context, account *Address) ([]byte, error) {
	return ec.client.PendingCodeAt(ctx.context, account.address)
}

// GetPendingNonceAt returns the account nonce of the given account in the pending state.
// This is the nonce that should be used for the next transaction.
func (ec *EthereumClient) GetPendingNonceAt(ctx *Context, account *Address) (int64, error) {
	nonce, err := ec.client.PendingNonceAt(ctx.context, account.address)
	return int64(nonce), err
}

// GetPendingTransactionCount returns the total number of transactions in the pending state.
func (ec *EthereumClient) GetPendingTransactionCount(ctx *Context) (int, error) {
	count, err := ec.client.PendingTransactionCount(ctx.context)
	return int(count), err
}

// Contract Calling

// CallContract executes a message call transaction, which is directly executed in the VM
// of the node, but never mined into the blockchain.
//
// blockNumber selects the block height at which the call runs. It can be <0, in which
// case the code is taken from the latest known block. Note that state from very old
// blocks might not be available.
func (ec *EthereumClient) CallContract(ctx *Context, msg *CallMsg, number int64) ([]byte, error) {
	if number < 0 {
		return ec.client.CallContract(ctx.context, msg.msg, nil)
	}
	return ec.client.CallContract(ctx.context, msg.msg, big.NewInt(number))
}

// PendingCallContract executes a message call transaction using the EVM.
// The state seen by the contract call is the pending state.
func (ec *EthereumClient) PendingCallContract(ctx *Context, msg *CallMsg) ([]byte, error) {
	return ec.client.PendingCallContract(ctx.context, msg.msg)
}

// SuggestGasPrice retrieves the currently suggested gas price to allow a timely
// execution of a transaction.
func (ec *EthereumClient) SuggestGasPrice(ctx *Context) (*BigInt, error) {
	price, err := ec.client.SuggestGasPrice(ctx.context)
	return &BigInt{price}, err
}

// EstimateGas tries to estimate the gas needed to execute a specific transaction based on
// the current pending state of the backend blockchain. There is no guarantee that this is
// the true gas limit requirement as other transactions may be added or removed by miners,
// but it should provide a basis for setting a reasonable default.
func (ec *EthereumClient) EstimateGas(ctx *Context, msg *CallMsg) (*BigInt, error) {
	price, err := ec.client.EstimateGas(ctx.context, msg.msg)
	return &BigInt{price}, err
}

// SendTransaction injects a signed transaction into the pending pool for execution.
//
// If the transaction was a contract creation use the TransactionReceipt method to get the
// contract address after the transaction has been mined.
func (ec *EthereumClient) SendTransaction(ctx *Context, tx *Transaction) error {
	return ec.client.SendTransaction(ctx.context, tx.tx)
}
