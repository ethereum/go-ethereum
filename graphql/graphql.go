// Copyright 2019 The go-ethereum Authors
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

// Package graphql provides a GraphQL interface to Ethereum node data.
package graphql

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	errOnlyOnMainChain = errors.New("this operation is only available for blocks on the canonical chain")
	errBlockInvariant  = errors.New("block objects must be instantiated with at least one of num or hash")
)

// Account represents an Ethereum account at a particular block.
type Account struct {
	backend     ethapi.Backend
	address     common.Address
	blockNumber rpc.BlockNumber
}

// getState fetches the StateDB object for an account.
func (a *Account) getState(ctx context.Context) (*state.StateDB, error) {
	state, _, err := a.backend.StateAndHeaderByNumber(ctx, a.blockNumber)
	return state, err
}

func (a *Account) Address(ctx context.Context) (common.Address, error) {
	return a.address, nil
}

func (a *Account) Balance(ctx context.Context) (hexutil.Big, error) {
	state, err := a.getState(ctx)
	if err != nil {
		return hexutil.Big{}, err
	}
	return hexutil.Big(*state.GetBalance(a.address)), nil
}

func (a *Account) TransactionCount(ctx context.Context) (hexutil.Uint64, error) {
	state, err := a.getState(ctx)
	if err != nil {
		return 0, err
	}
	return hexutil.Uint64(state.GetNonce(a.address)), nil
}

func (a *Account) Code(ctx context.Context) (hexutil.Bytes, error) {
	state, err := a.getState(ctx)
	if err != nil {
		return hexutil.Bytes{}, err
	}
	return hexutil.Bytes(state.GetCode(a.address)), nil
}

func (a *Account) Storage(ctx context.Context, args struct{ Slot common.Hash }) (common.Hash, error) {
	state, err := a.getState(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	return state.GetState(a.address, args.Slot), nil
}

// Log represents an individual log message. All arguments are mandatory.
type Log struct {
	backend     ethapi.Backend
	transaction *Transaction
	log         *types.Log
}

func (l *Log) Transaction(ctx context.Context) *Transaction {
	return l.transaction
}

func (l *Log) Account(ctx context.Context, args BlockNumberArgs) *Account {
	return &Account{
		backend:     l.backend,
		address:     l.log.Address,
		blockNumber: args.Number(),
	}
}

func (l *Log) Index(ctx context.Context) int32 {
	return int32(l.log.Index)
}

func (l *Log) Topics(ctx context.Context) []common.Hash {
	return l.log.Topics
}

func (l *Log) Data(ctx context.Context) hexutil.Bytes {
	return hexutil.Bytes(l.log.Data)
}

// Transaction represents an Ethereum transaction.
// backend and hash are mandatory; all others will be fetched when required.
type Transaction struct {
	backend ethapi.Backend
	hash    common.Hash
	tx      *types.Transaction
	block   *Block
	index   uint64
}

// resolve returns the internal transaction object, fetching it if needed.
func (t *Transaction) resolve(ctx context.Context) (*types.Transaction, error) {
	if t.tx == nil {
		tx, blockHash, _, index := rawdb.ReadTransaction(t.backend.ChainDb(), t.hash)
		if tx != nil {
			t.tx = tx
			t.block = &Block{
				backend:   t.backend,
				hash:      blockHash,
				canonical: unknown,
			}
			t.index = index
		} else {
			t.tx = t.backend.GetPoolTransaction(t.hash)
		}
	}
	return t.tx, nil
}

func (t *Transaction) Hash(ctx context.Context) common.Hash {
	return t.hash
}

func (t *Transaction) InputData(ctx context.Context) (hexutil.Bytes, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return hexutil.Bytes{}, err
	}
	return hexutil.Bytes(tx.Data()), nil
}

func (t *Transaction) Gas(ctx context.Context) (hexutil.Uint64, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return 0, err
	}
	return hexutil.Uint64(tx.Gas()), nil
}

func (t *Transaction) GasPrice(ctx context.Context) (hexutil.Big, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return hexutil.Big{}, err
	}
	return hexutil.Big(*tx.GasPrice()), nil
}

func (t *Transaction) Value(ctx context.Context) (hexutil.Big, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return hexutil.Big{}, err
	}
	return hexutil.Big(*tx.Value()), nil
}

func (t *Transaction) Nonce(ctx context.Context) (hexutil.Uint64, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return 0, err
	}
	return hexutil.Uint64(tx.Nonce()), nil
}

func (t *Transaction) To(ctx context.Context, args BlockNumberArgs) (*Account, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return nil, err
	}
	to := tx.To()
	if to == nil {
		return nil, nil
	}
	return &Account{
		backend:     t.backend,
		address:     *to,
		blockNumber: args.Number(),
	}, nil
}

func (t *Transaction) From(ctx context.Context, args BlockNumberArgs) (*Account, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return nil, err
	}
	var signer types.Signer = types.HomesteadSigner{}
	if tx.Protected() {
		signer = types.NewEIP155Signer(tx.ChainId())
	}
	from, _ := types.Sender(signer, tx)

	return &Account{
		backend:     t.backend,
		address:     from,
		blockNumber: args.Number(),
	}, nil
}

func (t *Transaction) Block(ctx context.Context) (*Block, error) {
	if _, err := t.resolve(ctx); err != nil {
		return nil, err
	}
	return t.block, nil
}

func (t *Transaction) Index(ctx context.Context) (*int32, error) {
	if _, err := t.resolve(ctx); err != nil {
		return nil, err
	}
	if t.block == nil {
		return nil, nil
	}
	index := int32(t.index)
	return &index, nil
}

// getReceipt returns the receipt associated with this transaction, if any.
func (t *Transaction) getReceipt(ctx context.Context) (*types.Receipt, error) {
	if _, err := t.resolve(ctx); err != nil {
		return nil, err
	}
	if t.block == nil {
		return nil, nil
	}
	receipts, err := t.block.resolveReceipts(ctx)
	if err != nil {
		return nil, err
	}
	return receipts[t.index], nil
}

func (t *Transaction) Status(ctx context.Context) (*hexutil.Uint64, error) {
	receipt, err := t.getReceipt(ctx)
	if err != nil || receipt == nil {
		return nil, err
	}
	ret := hexutil.Uint64(receipt.Status)
	return &ret, nil
}

func (t *Transaction) GasUsed(ctx context.Context) (*hexutil.Uint64, error) {
	receipt, err := t.getReceipt(ctx)
	if err != nil || receipt == nil {
		return nil, err
	}
	ret := hexutil.Uint64(receipt.GasUsed)
	return &ret, nil
}

func (t *Transaction) CumulativeGasUsed(ctx context.Context) (*hexutil.Uint64, error) {
	receipt, err := t.getReceipt(ctx)
	if err != nil || receipt == nil {
		return nil, err
	}
	ret := hexutil.Uint64(receipt.CumulativeGasUsed)
	return &ret, nil
}

func (t *Transaction) CreatedContract(ctx context.Context, args BlockNumberArgs) (*Account, error) {
	receipt, err := t.getReceipt(ctx)
	if err != nil || receipt == nil || receipt.ContractAddress == (common.Address{}) {
		return nil, err
	}
	return &Account{
		backend:     t.backend,
		address:     receipt.ContractAddress,
		blockNumber: args.Number(),
	}, nil
}

func (t *Transaction) Logs(ctx context.Context) (*[]*Log, error) {
	receipt, err := t.getReceipt(ctx)
	if err != nil || receipt == nil {
		return nil, err
	}
	ret := make([]*Log, 0, len(receipt.Logs))
	for _, log := range receipt.Logs {
		ret = append(ret, &Log{
			backend:     t.backend,
			transaction: t,
			log:         log,
		})
	}
	return &ret, nil
}

type BlockType int

const (
	unknown BlockType = iota
	isCanonical
	notCanonical
)

// Block represents an Ethereum block.
// backend, and either num or hash are mandatory. All other fields are lazily fetched
// when required.
type Block struct {
	backend   ethapi.Backend
	num       *rpc.BlockNumber
	hash      common.Hash
	header    *types.Header
	block     *types.Block
	receipts  []*types.Receipt
	canonical BlockType // Indicates if this block is on the main chain or not.
}

func (b *Block) onMainChain(ctx context.Context) error {
	if b.canonical == unknown {
		header, err := b.resolveHeader(ctx)
		if err != nil {
			return err
		}
		canonHeader, err := b.backend.HeaderByNumber(ctx, rpc.BlockNumber(header.Number.Uint64()))
		if err != nil {
			return err
		}
		if header.Hash() == canonHeader.Hash() {
			b.canonical = isCanonical
		} else {
			b.canonical = notCanonical
		}
	}
	if b.canonical != isCanonical {
		return errOnlyOnMainChain
	}
	return nil
}

// resolve returns the internal Block object representing this block, fetching
// it if necessary.
func (b *Block) resolve(ctx context.Context) (*types.Block, error) {
	if b.block != nil {
		return b.block, nil
	}
	var err error
	if b.hash != (common.Hash{}) {
		b.block, err = b.backend.BlockByHash(ctx, b.hash)
	} else {
		b.block, err = b.backend.BlockByNumber(ctx, *b.num)
	}
	if b.block != nil && b.header == nil {
		b.header = b.block.Header()
	}
	return b.block, err
}

// resolveHeader returns the internal Header object for this block, fetching it
// if necessary. Call this function instead of `resolve` unless you need the
// additional data (transactions and uncles).
func (b *Block) resolveHeader(ctx context.Context) (*types.Header, error) {
	if b.num == nil && b.hash == (common.Hash{}) {
		return nil, errBlockInvariant
	}
	var err error
	if b.header == nil {
		if b.hash != (common.Hash{}) {
			b.header, err = b.backend.HeaderByHash(ctx, b.hash)
		} else {
			b.header, err = b.backend.HeaderByNumber(ctx, *b.num)
		}
	}
	return b.header, err
}

// resolveReceipts returns the list of receipts for this block, fetching them
// if necessary.
func (b *Block) resolveReceipts(ctx context.Context) ([]*types.Receipt, error) {
	if b.receipts == nil {
		hash := b.hash
		if hash == (common.Hash{}) {
			header, err := b.resolveHeader(ctx)
			if err != nil {
				return nil, err
			}
			hash = header.Hash()
		}
		receipts, err := b.backend.GetReceipts(ctx, hash)
		if err != nil {
			return nil, err
		}
		b.receipts = []*types.Receipt(receipts)
	}
	return b.receipts, nil
}

func (b *Block) Number(ctx context.Context) (hexutil.Uint64, error) {
	if b.num == nil || *b.num == rpc.LatestBlockNumber {
		header, err := b.resolveHeader(ctx)
		if err != nil {
			return 0, err
		}
		num := rpc.BlockNumber(header.Number.Uint64())
		b.num = &num
	}
	return hexutil.Uint64(*b.num), nil
}

func (b *Block) Hash(ctx context.Context) (common.Hash, error) {
	if b.hash == (common.Hash{}) {
		header, err := b.resolveHeader(ctx)
		if err != nil {
			return common.Hash{}, err
		}
		b.hash = header.Hash()
	}
	return b.hash, nil
}

func (b *Block) GasLimit(ctx context.Context) (hexutil.Uint64, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return 0, err
	}
	return hexutil.Uint64(header.GasLimit), nil
}

func (b *Block) GasUsed(ctx context.Context) (hexutil.Uint64, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return 0, err
	}
	return hexutil.Uint64(header.GasUsed), nil
}

func (b *Block) Parent(ctx context.Context) (*Block, error) {
	// If the block header hasn't been fetched, and we'll need it, fetch it.
	if b.num == nil && b.hash != (common.Hash{}) && b.header == nil {
		if _, err := b.resolveHeader(ctx); err != nil {
			return nil, err
		}
	}
	if b.header != nil && b.header.Number.Uint64() > 0 {
		num := rpc.BlockNumber(b.header.Number.Uint64() - 1)
		return &Block{
			backend:   b.backend,
			num:       &num,
			hash:      b.header.ParentHash,
			canonical: unknown,
		}, nil
	}
	if b.num != nil && *b.num != 0 {
		num := *b.num - 1
		return &Block{
			backend:   b.backend,
			num:       &num,
			canonical: isCanonical,
		}, nil
	}
	return nil, nil
}

func (b *Block) Difficulty(ctx context.Context) (hexutil.Big, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return hexutil.Big{}, err
	}
	return hexutil.Big(*header.Difficulty), nil
}

func (b *Block) Timestamp(ctx context.Context) (hexutil.Uint64, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return 0, err
	}
	return hexutil.Uint64(header.Time), nil
}

func (b *Block) Nonce(ctx context.Context) (hexutil.Bytes, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return hexutil.Bytes{}, err
	}
	return hexutil.Bytes(header.Nonce[:]), nil
}

func (b *Block) MixHash(ctx context.Context) (common.Hash, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	return header.MixDigest, nil
}

func (b *Block) TransactionsRoot(ctx context.Context) (common.Hash, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	return header.TxHash, nil
}

func (b *Block) StateRoot(ctx context.Context) (common.Hash, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	return header.Root, nil
}

func (b *Block) ReceiptsRoot(ctx context.Context) (common.Hash, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	return header.ReceiptHash, nil
}

func (b *Block) OmmerHash(ctx context.Context) (common.Hash, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return common.Hash{}, err
	}
	return header.UncleHash, nil
}

func (b *Block) OmmerCount(ctx context.Context) (*int32, error) {
	block, err := b.resolve(ctx)
	if err != nil || block == nil {
		return nil, err
	}
	count := int32(len(block.Uncles()))
	return &count, err
}

func (b *Block) Ommers(ctx context.Context) (*[]*Block, error) {
	block, err := b.resolve(ctx)
	if err != nil || block == nil {
		return nil, err
	}
	ret := make([]*Block, 0, len(block.Uncles()))
	for _, uncle := range block.Uncles() {
		blockNumber := rpc.BlockNumber(uncle.Number.Uint64())
		ret = append(ret, &Block{
			backend:   b.backend,
			num:       &blockNumber,
			hash:      uncle.Hash(),
			header:    uncle,
			canonical: notCanonical,
		})
	}
	return &ret, nil
}

func (b *Block) ExtraData(ctx context.Context) (hexutil.Bytes, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return hexutil.Bytes{}, err
	}
	return hexutil.Bytes(header.Extra), nil
}

func (b *Block) LogsBloom(ctx context.Context) (hexutil.Bytes, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return hexutil.Bytes{}, err
	}
	return hexutil.Bytes(header.Bloom.Bytes()), nil
}

func (b *Block) TotalDifficulty(ctx context.Context) (hexutil.Big, error) {
	h := b.hash
	if h == (common.Hash{}) {
		header, err := b.resolveHeader(ctx)
		if err != nil {
			return hexutil.Big{}, err
		}
		h = header.Hash()
	}
	return hexutil.Big(*b.backend.GetTd(h)), nil
}

// BlockNumberArgs encapsulates arguments to accessors that specify a block number.
type BlockNumberArgs struct {
	Block *hexutil.Uint64
}

// Number returns the provided block number, or rpc.LatestBlockNumber if none
// was provided.
func (a BlockNumberArgs) Number() rpc.BlockNumber {
	if a.Block != nil {
		return rpc.BlockNumber(*a.Block)
	}
	return rpc.LatestBlockNumber
}

func (b *Block) Miner(ctx context.Context, args BlockNumberArgs) (*Account, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return nil, err
	}
	return &Account{
		backend:     b.backend,
		address:     header.Coinbase,
		blockNumber: args.Number(),
	}, nil
}

func (b *Block) TransactionCount(ctx context.Context) (*int32, error) {
	block, err := b.resolve(ctx)
	if err != nil || block == nil {
		return nil, err
	}
	count := int32(len(block.Transactions()))
	return &count, err
}

func (b *Block) Transactions(ctx context.Context) (*[]*Transaction, error) {
	block, err := b.resolve(ctx)
	if err != nil || block == nil {
		return nil, err
	}
	ret := make([]*Transaction, 0, len(block.Transactions()))
	for i, tx := range block.Transactions() {
		ret = append(ret, &Transaction{
			backend: b.backend,
			hash:    tx.Hash(),
			tx:      tx,
			block:   b,
			index:   uint64(i),
		})
	}
	return &ret, nil
}

func (b *Block) TransactionAt(ctx context.Context, args struct{ Index int32 }) (*Transaction, error) {
	block, err := b.resolve(ctx)
	if err != nil || block == nil {
		return nil, err
	}
	txs := block.Transactions()
	if args.Index < 0 || int(args.Index) >= len(txs) {
		return nil, nil
	}
	tx := txs[args.Index]
	return &Transaction{
		backend: b.backend,
		hash:    tx.Hash(),
		tx:      tx,
		block:   b,
		index:   uint64(args.Index),
	}, nil
}

func (b *Block) OmmerAt(ctx context.Context, args struct{ Index int32 }) (*Block, error) {
	block, err := b.resolve(ctx)
	if err != nil || block == nil {
		return nil, err
	}
	uncles := block.Uncles()
	if args.Index < 0 || int(args.Index) >= len(uncles) {
		return nil, nil
	}
	uncle := uncles[args.Index]
	blockNumber := rpc.BlockNumber(uncle.Number.Uint64())
	return &Block{
		backend:   b.backend,
		num:       &blockNumber,
		hash:      uncle.Hash(),
		header:    uncle,
		canonical: notCanonical,
	}, nil
}

// BlockFilterCriteria encapsulates criteria passed to a `logs` accessor inside
// a block.
type BlockFilterCriteria struct {
	Addresses *[]common.Address // restricts matches to events created by specific contracts

	// The Topic list restricts matches to particular event topics. Each event has a list
	// of topics. Topics matches a prefix of that list. An empty element slice matches any
	// topic. Non-empty elements represent an alternative that matches any of the
	// contained topics.
	//
	// Examples:
	// {} or nil          matches any topic list
	// {{A}}              matches topic A in first position
	// {{}, {B}}          matches any topic in first position, B in second position
	// {{A}, {B}}         matches topic A in first position, B in second position
	// {{A, B}}, {C, D}}  matches topic (A OR B) in first position, (C OR D) in second position
	Topics *[][]common.Hash
}

// runFilter accepts a filter and executes it, returning all its results as
// `Log` objects.
func runFilter(ctx context.Context, be ethapi.Backend, filter *filters.Filter) ([]*Log, error) {
	logs, err := filter.Logs(ctx)
	if err != nil || logs == nil {
		return nil, err
	}
	ret := make([]*Log, 0, len(logs))
	for _, log := range logs {
		ret = append(ret, &Log{
			backend:     be,
			transaction: &Transaction{backend: be, hash: log.TxHash},
			log:         log,
		})
	}
	return ret, nil
}

func (b *Block) Logs(ctx context.Context, args struct{ Filter BlockFilterCriteria }) ([]*Log, error) {
	var addresses []common.Address
	if args.Filter.Addresses != nil {
		addresses = *args.Filter.Addresses
	}
	var topics [][]common.Hash
	if args.Filter.Topics != nil {
		topics = *args.Filter.Topics
	}
	hash := b.hash
	if hash == (common.Hash{}) {
		header, err := b.resolveHeader(ctx)
		if err != nil {
			return nil, err
		}
		hash = header.Hash()
	}
	// Construct the range filter
	filter := filters.NewBlockFilter(b.backend, hash, addresses, topics)

	// Run the filter and return all the logs
	return runFilter(ctx, b.backend, filter)
}

func (b *Block) Account(ctx context.Context, args struct {
	Address common.Address
}) (*Account, error) {
	err := b.onMainChain(ctx)
	if err != nil {
		return nil, err
	}
	if b.num == nil {
		_, err := b.resolveHeader(ctx)
		if err != nil {
			return nil, err
		}
	}
	return &Account{
		backend:     b.backend,
		address:     args.Address,
		blockNumber: *b.num,
	}, nil
}

// CallData encapsulates arguments to `call` or `estimateGas`.
// All arguments are optional.
type CallData struct {
	From     *common.Address // The Ethereum address the call is from.
	To       *common.Address // The Ethereum address the call is to.
	Gas      *hexutil.Uint64 // The amount of gas provided for the call.
	GasPrice *hexutil.Big    // The price of each unit of gas, in wei.
	Value    *hexutil.Big    // The value sent along with the call.
	Data     *hexutil.Bytes  // Any data sent with the call.
}

// CallResult encapsulates the result of an invocation of the `call` accessor.
type CallResult struct {
	data    hexutil.Bytes  // The return data from the call
	gasUsed hexutil.Uint64 // The amount of gas used
	status  hexutil.Uint64 // The return status of the call - 0 for failure or 1 for success.
}

func (c *CallResult) Data() hexutil.Bytes {
	return c.data
}

func (c *CallResult) GasUsed() hexutil.Uint64 {
	return c.gasUsed
}

func (c *CallResult) Status() hexutil.Uint64 {
	return c.status
}

func (b *Block) Call(ctx context.Context, args struct {
	Data ethapi.CallArgs
}) (*CallResult, error) {
	err := b.onMainChain(ctx)
	if err != nil {
		return nil, err
	}
	if b.num == nil {
		_, err := b.resolveHeader(ctx)
		if err != nil {
			return nil, err
		}
	}
	result, gas, failed, err := ethapi.DoCall(ctx, b.backend, args.Data, *b.num, nil, vm.Config{}, 5*time.Second, b.backend.RPCGasCap())
	status := hexutil.Uint64(1)
	if failed {
		status = 0
	}
	return &CallResult{
		data:    hexutil.Bytes(result),
		gasUsed: hexutil.Uint64(gas),
		status:  status,
	}, err
}

func (b *Block) EstimateGas(ctx context.Context, args struct {
	Data ethapi.CallArgs
}) (hexutil.Uint64, error) {
	err := b.onMainChain(ctx)
	if err != nil {
		return hexutil.Uint64(0), err
	}
	if b.num == nil {
		_, err := b.resolveHeader(ctx)
		if err != nil {
			return hexutil.Uint64(0), err
		}
	}
	gas, err := ethapi.DoEstimateGas(ctx, b.backend, args.Data, *b.num, b.backend.RPCGasCap())
	return gas, err
}

type Pending struct {
	backend ethapi.Backend
}

func (p *Pending) TransactionCount(ctx context.Context) (int32, error) {
	txs, err := p.backend.GetPoolTransactions()
	return int32(len(txs)), err
}

func (p *Pending) Transactions(ctx context.Context) (*[]*Transaction, error) {
	txs, err := p.backend.GetPoolTransactions()
	if err != nil {
		return nil, err
	}
	ret := make([]*Transaction, 0, len(txs))
	for i, tx := range txs {
		ret = append(ret, &Transaction{
			backend: p.backend,
			hash:    tx.Hash(),
			tx:      tx,
			index:   uint64(i),
		})
	}
	return &ret, nil
}

func (p *Pending) Account(ctx context.Context, args struct {
	Address common.Address
}) *Account {
	return &Account{
		backend:     p.backend,
		address:     args.Address,
		blockNumber: rpc.PendingBlockNumber,
	}
}

func (p *Pending) Call(ctx context.Context, args struct {
	Data ethapi.CallArgs
}) (*CallResult, error) {
	result, gas, failed, err := ethapi.DoCall(ctx, p.backend, args.Data, rpc.PendingBlockNumber, nil, vm.Config{}, 5*time.Second, p.backend.RPCGasCap())
	status := hexutil.Uint64(1)
	if failed {
		status = 0
	}
	return &CallResult{
		data:    hexutil.Bytes(result),
		gasUsed: hexutil.Uint64(gas),
		status:  status,
	}, err
}

func (p *Pending) EstimateGas(ctx context.Context, args struct {
	Data ethapi.CallArgs
}) (hexutil.Uint64, error) {
	return ethapi.DoEstimateGas(ctx, p.backend, args.Data, rpc.PendingBlockNumber, p.backend.RPCGasCap())
}

// Resolver is the top-level object in the GraphQL hierarchy.
type Resolver struct {
	backend ethapi.Backend
}

func (r *Resolver) Block(ctx context.Context, args struct {
	Number *hexutil.Uint64
	Hash   *common.Hash
}) (*Block, error) {
	var block *Block
	if args.Number != nil {
		num := rpc.BlockNumber(uint64(*args.Number))
		block = &Block{
			backend:   r.backend,
			num:       &num,
			canonical: isCanonical,
		}
	} else if args.Hash != nil {
		block = &Block{
			backend:   r.backend,
			hash:      *args.Hash,
			canonical: unknown,
		}
	} else {
		num := rpc.LatestBlockNumber
		block = &Block{
			backend:   r.backend,
			num:       &num,
			canonical: isCanonical,
		}
	}
	// Resolve the header, return nil if it doesn't exist.
	// Note we don't resolve block directly here since it will require an
	// additional network request for light client.
	h, err := block.resolveHeader(ctx)
	if err != nil {
		return nil, err
	} else if h == nil {
		return nil, nil
	}
	return block, nil
}

func (r *Resolver) Blocks(ctx context.Context, args struct {
	From hexutil.Uint64
	To   *hexutil.Uint64
}) ([]*Block, error) {
	from := rpc.BlockNumber(args.From)

	var to rpc.BlockNumber
	if args.To != nil {
		to = rpc.BlockNumber(*args.To)
	} else {
		to = rpc.BlockNumber(r.backend.CurrentBlock().Number().Int64())
	}
	if to < from {
		return []*Block{}, nil
	}
	ret := make([]*Block, 0, to-from+1)
	for i := from; i <= to; i++ {
		num := i
		ret = append(ret, &Block{
			backend:   r.backend,
			num:       &num,
			canonical: isCanonical,
		})
	}
	return ret, nil
}

func (r *Resolver) Pending(ctx context.Context) *Pending {
	return &Pending{r.backend}
}

func (r *Resolver) Transaction(ctx context.Context, args struct{ Hash common.Hash }) (*Transaction, error) {
	tx := &Transaction{
		backend: r.backend,
		hash:    args.Hash,
	}
	// Resolve the transaction; if it doesn't exist, return nil.
	t, err := tx.resolve(ctx)
	if err != nil {
		return nil, err
	} else if t == nil {
		return nil, nil
	}
	return tx, nil
}

func (r *Resolver) SendRawTransaction(ctx context.Context, args struct{ Data hexutil.Bytes }) (common.Hash, error) {
	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(args.Data, tx); err != nil {
		return common.Hash{}, err
	}
	hash, err := ethapi.SubmitTransaction(ctx, r.backend, tx)
	return hash, err
}

// FilterCriteria encapsulates the arguments to `logs` on the root resolver object.
type FilterCriteria struct {
	FromBlock *hexutil.Uint64   // beginning of the queried range, nil means genesis block
	ToBlock   *hexutil.Uint64   // end of the range, nil means latest block
	Addresses *[]common.Address // restricts matches to events created by specific contracts

	// The Topic list restricts matches to particular event topics. Each event has a list
	// of topics. Topics matches a prefix of that list. An empty element slice matches any
	// topic. Non-empty elements represent an alternative that matches any of the
	// contained topics.
	//
	// Examples:
	// {} or nil          matches any topic list
	// {{A}}              matches topic A in first position
	// {{}, {B}}          matches any topic in first position, B in second position
	// {{A}, {B}}         matches topic A in first position, B in second position
	// {{A, B}}, {C, D}}  matches topic (A OR B) in first position, (C OR D) in second position
	Topics *[][]common.Hash
}

func (r *Resolver) Logs(ctx context.Context, args struct{ Filter FilterCriteria }) ([]*Log, error) {
	// Convert the RPC block numbers into internal representations
	begin := rpc.LatestBlockNumber.Int64()
	if args.Filter.FromBlock != nil {
		begin = int64(*args.Filter.FromBlock)
	}
	end := rpc.LatestBlockNumber.Int64()
	if args.Filter.ToBlock != nil {
		end = int64(*args.Filter.ToBlock)
	}
	var addresses []common.Address
	if args.Filter.Addresses != nil {
		addresses = *args.Filter.Addresses
	}
	var topics [][]common.Hash
	if args.Filter.Topics != nil {
		topics = *args.Filter.Topics
	}
	// Construct the range filter
	filter := filters.NewRangeFilter(filters.Backend(r.backend), begin, end, addresses, topics)
	return runFilter(ctx, r.backend, filter)
}

func (r *Resolver) GasPrice(ctx context.Context) (hexutil.Big, error) {
	price, err := r.backend.SuggestPrice(ctx)
	return hexutil.Big(*price), err
}

func (r *Resolver) ProtocolVersion(ctx context.Context) (int32, error) {
	return int32(r.backend.ProtocolVersion()), nil
}

// SyncState represents the synchronisation status returned from the `syncing` accessor.
type SyncState struct {
	progress ethereum.SyncProgress
}

func (s *SyncState) StartingBlock() hexutil.Uint64 {
	return hexutil.Uint64(s.progress.StartingBlock)
}

func (s *SyncState) CurrentBlock() hexutil.Uint64 {
	return hexutil.Uint64(s.progress.CurrentBlock)
}

func (s *SyncState) HighestBlock() hexutil.Uint64 {
	return hexutil.Uint64(s.progress.HighestBlock)
}

func (s *SyncState) PulledStates() *hexutil.Uint64 {
	ret := hexutil.Uint64(s.progress.PulledStates)
	return &ret
}

func (s *SyncState) KnownStates() *hexutil.Uint64 {
	ret := hexutil.Uint64(s.progress.KnownStates)
	return &ret
}

// Syncing returns false in case the node is currently not syncing with the network. It can be up to date or has not
// yet received the latest block headers from its pears. In case it is synchronizing:
// - startingBlock: block number this node started to synchronise from
// - currentBlock:  block number this node is currently importing
// - highestBlock:  block number of the highest block header this node has received from peers
// - pulledStates:  number of state entries processed until now
// - knownStates:   number of known state entries that still need to be pulled
func (r *Resolver) Syncing() (*SyncState, error) {
	progress := r.backend.Downloader().Progress()

	// Return not syncing if the synchronisation already completed
	if progress.CurrentBlock >= progress.HighestBlock {
		return nil, nil
	}
	// Otherwise gather the block sync stats
	return &SyncState{progress}, nil
}
