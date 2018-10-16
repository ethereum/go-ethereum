// Copyright 2018 The go-ethereum Authors
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

package ethgraphql

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

func getBackend(n *node.Node) (*eth.EthAPIBackend, error) {
	var ethereum *eth.Ethereum
	if err := n.Service(&ethereum); err != nil {
		return nil, err
	}
	return ethereum.APIBackend, nil
}

type Account struct {
	node        *node.Node
	address     common.Address
	blockNumber rpc.BlockNumber
}

func (a *Account) getState(ctx context.Context) (*state.StateDB, error) {
	be, err := getBackend(a.node)
	if err != nil {
		return nil, err
	}

	state, _, err := be.StateAndHeaderByNumber(ctx, a.blockNumber)
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

type StorageSlotArgs struct {
	Slot common.Hash
}

func (a *Account) Storage(ctx context.Context, args StorageSlotArgs) (common.Hash, error) {
	state, err := a.getState(ctx)
	if err != nil {
		return common.Hash{}, err
	}

	return state.GetState(a.address, args.Slot), nil
}

type Log struct {
	node        *node.Node
	transaction *Transaction
	log         *types.Log
}

func (l *Log) Transaction(ctx context.Context) *Transaction {
	return l.transaction
}

func (l *Log) Account(ctx context.Context, args BlockNumberArgs) *Account {
	return &Account{
		node:        l.node,
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

type Transaction struct {
	node  *node.Node
	hash  common.Hash
	tx    *types.Transaction
	block *Block
	index uint64
}

func (t *Transaction) resolve(ctx context.Context) (*types.Transaction, error) {
	if t.tx == nil {
		be, err := getBackend(t.node)
		if err != nil {
			return nil, err
		}

		tx, blockHash, _, index := rawdb.ReadTransaction(be.ChainDb(), t.hash)
		if tx != nil {
			t.tx = tx
			t.block = &Block{
				node: t.node,
				hash: blockHash,
			}
			t.index = index
		} else {
			t.tx = be.GetPoolTransaction(t.hash)
		}
	}
	return t.tx, nil
}

func (tx *Transaction) Hash(ctx context.Context) common.Hash {
	return tx.hash
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
		node:        t.node,
		address:     *to,
		blockNumber: args.Number(),
	}, nil
}

func (t *Transaction) From(ctx context.Context, args BlockNumberArgs) (*Account, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return nil, err
	}

	var signer types.Signer = types.FrontierSigner{}
	if tx.Protected() {
		signer = types.NewEIP155Signer(tx.ChainId())
	}
	from, _ := types.Sender(signer, tx)

	return &Account{
		node:        t.node,
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
		node:        t.node,
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
			node:        t.node,
			transaction: t,
			log:         log,
		})
	}
	return &ret, nil
}

type Block struct {
	node     *node.Node
	num      *rpc.BlockNumber
	hash     common.Hash
	header   *types.Header
	block    *types.Block
	receipts []*types.Receipt
}

func (b *Block) resolve(ctx context.Context) (*types.Block, error) {
	if b.block != nil {
		return b.block, nil
	}

	be, err := getBackend(b.node)
	if err != nil {
		return nil, err
	}

	if b.hash != (common.Hash{}) {
		b.block, err = be.GetBlock(ctx, b.hash)
	} else {
		b.block, err = be.BlockByNumber(ctx, *b.num)
	}
	if b.block != nil {
		b.header = b.block.Header()
	}
	return b.block, err
}

func (b *Block) resolveHeader(ctx context.Context) (*types.Header, error) {
	if b.header == nil {
		if _, err := b.resolve(ctx); err != nil {
			return nil, err
		}
	}
	return b.header, nil
}

func (b *Block) resolveReceipts(ctx context.Context) ([]*types.Receipt, error) {
	if b.receipts == nil {
		be, err := getBackend(b.node)
		if err != nil {
			return nil, err
		}

		hash := b.hash
		if hash == (common.Hash{}) {
			header, err := b.resolveHeader(ctx)
			if err != nil {
				return nil, err
			}
			hash = header.Hash()
		}

		receipts, err := be.GetReceipts(ctx, hash)
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
	// If the block hasn't been fetched, and we'll need it, fetch it.
	if b.num == nil && b.hash != (common.Hash{}) && b.header == nil {
		if _, err := b.resolve(ctx); err != nil {
			return nil, err
		}
	}

	if b.header != nil && b.block.NumberU64() > 0 {
		num := rpc.BlockNumber(b.header.Number.Uint64() - 1)
		return &Block{
			node: b.node,
			num:  &num,
			hash: b.header.ParentHash,
		}, nil
	} else if b.num != nil && *b.num != 0 {
		num := *b.num - 1
		return &Block{
			node: b.node,
			num:  &num,
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

func (b *Block) Timestamp(ctx context.Context) (hexutil.Big, error) {
	header, err := b.resolveHeader(ctx)
	if err != nil {
		return hexutil.Big{}, err
	}
	return hexutil.Big(*header.Time), nil
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
			node:   b.node,
			num:    &blockNumber,
			hash:   uncle.Hash(),
			header: uncle,
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

	be, err := getBackend(b.node)
	if err != nil {
		return hexutil.Big{}, err
	}

	return hexutil.Big(*be.GetTd(h)), nil
}

type BlockNumberArgs struct {
	Block *hexutil.Uint64
}

func (a BlockNumberArgs) Number() rpc.BlockNumber {
	if a.Block != nil {
		return rpc.BlockNumber(*a.Block)
	}
	return rpc.LatestBlockNumber
}

func (b *Block) Miner(ctx context.Context, args BlockNumberArgs) (*Account, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return nil, err
	}

	return &Account{
		node:        b.node,
		address:     block.Coinbase(),
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
			node:  b.node,
			hash:  tx.Hash(),
			tx:    tx,
			block: b,
			index: uint64(i),
		})
	}
	return &ret, nil
}

type ArrayIndexArgs struct {
	Index int32
}

func (b *Block) TransactionAt(ctx context.Context, args ArrayIndexArgs) (*Transaction, error) {
	block, err := b.resolve(ctx)
	if err != nil || block == nil {
		return nil, err
	}

	txes := block.Transactions()
	if args.Index < 0 || int(args.Index) >= len(txes) {
		return nil, nil
	}

	tx := txes[args.Index]
	return &Transaction{
		node:  b.node,
		hash:  tx.Hash(),
		tx:    tx,
		block: b,
		index: uint64(args.Index),
	}, nil
}

func (b *Block) OmmerAt(ctx context.Context, args ArrayIndexArgs) (*Block, error) {
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
		node:   b.node,
		num:    &blockNumber,
		hash:   uncle.Hash(),
		header: uncle,
	}, nil
}

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

func runFilter(ctx context.Context, node *node.Node, filter *filters.Filter) ([]*Log, error) {
	logs, err := filter.Logs(ctx)
	if err != nil || logs == nil {
		return nil, err
	}

	ret := make([]*Log, 0, len(logs))
	for _, log := range logs {
		ret = append(ret, &Log{
			node:        node,
			transaction: &Transaction{node: node, hash: log.TxHash},
			log:         log,
		})
	}
	return ret, nil
}

func (b *Block) Logs(ctx context.Context, args struct{ Filter BlockFilterCriteria }) ([]*Log, error) {
	be, err := getBackend(b.node)
	if err != nil {
		return nil, err
	}

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
		block, err := b.resolve(ctx)
		if err != nil {
			return nil, err
		}
		hash = block.Hash()
	}

	// Construct the range filter
	filter := filters.NewBlockFilter(be, hash, addresses, topics)

	// Run the filter and return all the logs
	return runFilter(ctx, b.node, filter)
}

type Resolver struct {
	node *node.Node
}

type BlockArgs struct {
	Number *hexutil.Uint64
	Hash   *common.Hash
}

func (r *Resolver) Block(ctx context.Context, args BlockArgs) (*Block, error) {
	var block *Block
	if args.Number != nil {
		num := rpc.BlockNumber(uint64(*args.Number))
		block = &Block{
			node: r.node,
			num:  &num,
		}
	} else if args.Hash != nil {
		block = &Block{
			node: r.node,
			hash: *args.Hash,
		}
	} else {
		num := rpc.LatestBlockNumber
		block = &Block{
			node: r.node,
			num:  &num,
		}
	}

	// Resolve the block; if it doesn't exist, return nil.
	b, err := block.resolve(ctx)
	if err != nil {
		return nil, err
	} else if b == nil {
		return nil, nil
	}
	return block, nil
}

type BlocksArgs struct {
	From hexutil.Uint64
	To   *hexutil.Uint64
}

func (r *Resolver) Blocks(ctx context.Context, args BlocksArgs) ([]*Block, error) {
	be, err := getBackend(r.node)
	if err != nil {
		return nil, err
	}

	from := rpc.BlockNumber(args.From)

	var to rpc.BlockNumber
	if args.To != nil {
		to = rpc.BlockNumber(*args.To)
	} else {
		to = rpc.BlockNumber(be.CurrentBlock().Number().Int64())
	}

	if to < from {
		return []*Block{}, nil
	}

	ret := make([]*Block, 0, to-from+1)
	for i := from; i <= to; i++ {
		num := i
		ret = append(ret, &Block{
			node: r.node,
			num:  &num,
		})
	}
	return ret, nil
}

type AccountArgs struct {
	Address     common.Address
	BlockNumber *hexutil.Uint64
}

func (r *Resolver) Account(ctx context.Context, args AccountArgs) *Account {
	blockNumber := rpc.LatestBlockNumber
	if args.BlockNumber != nil {
		blockNumber = rpc.BlockNumber(*args.BlockNumber)
	}

	return &Account{
		node:        r.node,
		address:     args.Address,
		blockNumber: blockNumber,
	}
}

type TransactionArgs struct {
	Hash common.Hash
}

func (r *Resolver) Transaction(ctx context.Context, args TransactionArgs) (*Transaction, error) {
	tx := &Transaction{
		node: r.node,
		hash: args.Hash,
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
	be, err := getBackend(r.node)
	if err != nil {
		return common.Hash{}, err
	}

	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(args.Data, tx); err != nil {
		return common.Hash{}, err
	}
	hash, err := ethapi.SubmitTransaction(ctx, be, tx)
	return hash, err
}

type CallData struct {
	From     *common.Address
	To       *common.Address
	Gas      *hexutil.Uint64
	GasPrice *hexutil.Big
	Value    *hexutil.Big
	Data     *hexutil.Bytes
}

type CallResult struct {
	data    hexutil.Bytes
	gasUsed hexutil.Uint64
	status  hexutil.Uint64
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

func (r *Resolver) Call(ctx context.Context, args struct {
	Data        ethapi.CallArgs
	BlockNumber *hexutil.Uint64
}) (*CallResult, error) {
	be, err := getBackend(r.node)
	if err != nil {
		return nil, err
	}

	blockNumber := rpc.LatestBlockNumber
	if args.BlockNumber != nil {
		blockNumber = rpc.BlockNumber(*args.BlockNumber)
	}

	result, gas, failed, err := ethapi.DoCall(ctx, be, args.Data, blockNumber, vm.Config{}, 5*time.Second)
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

func (r *Resolver) EstimateGas(ctx context.Context, args struct {
	Data        ethapi.CallArgs
	BlockNumber *hexutil.Uint64
}) (hexutil.Uint64, error) {
	be, err := getBackend(r.node)
	if err != nil {
		return 0, err
	}

	blockNumber := rpc.LatestBlockNumber
	if args.BlockNumber != nil {
		blockNumber = rpc.BlockNumber(*args.BlockNumber)
	}

	gas, err := ethapi.DoEstimateGas(ctx, be, args.Data, blockNumber)
	return hexutil.Uint64(gas), err
}

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
	be, err := getBackend(r.node)
	if err != nil {
		return nil, err
	}

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
	filter := filters.NewRangeFilter(filters.Backend(be), begin, end, addresses, topics)

	return runFilter(ctx, r.node, filter)
}

func (r *Resolver) GasPrice(ctx context.Context) (hexutil.Big, error) {
	be, err := getBackend(r.node)
	if err != nil {
		return hexutil.Big{}, err
	}

	price, err := be.SuggestPrice(ctx)
	return hexutil.Big(*price), err
}

func (r *Resolver) ProtocolVersion(ctx context.Context) (int32, error) {
	be, err := getBackend(r.node)
	if err != nil {
		return 0, err
	}

	return int32(be.ProtocolVersion()), nil
}

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
	be, err := getBackend(r.node)
	if err != nil {
		return nil, err
	}
	progress := be.Downloader().Progress()

	// Return not syncing if the synchronisation already completed
	if progress.CurrentBlock >= progress.HighestBlock {
		return nil, nil
	}
	// Otherwise gather the block sync stats
	return &SyncState{progress}, nil
}

func NewHandler(n *node.Node) (http.Handler, error) {
	q := Resolver{n}

	s := `
        scalar Bytes32
        scalar Address
        scalar Bytes
        scalar BigInt
        scalar Long

        schema {
            query: Query
            mutation: Mutation
        }

        type Account {
            address: Address!
            balance: BigInt!
            transactionCount: Long!
            code: Bytes!
            storage(slot: Bytes32!): Bytes32!
        }

        type Log {
            index: Int!
            account(block: Long): Account!
            topics: [Bytes32!]!
            data: Bytes!
            transaction: Transaction!
        }

        type Transaction {
            hash: Bytes32!
            nonce: Long!
            index: Int
            from(block: Long): Account!
            to(block: Long): Account
            value: BigInt!
            gasPrice: BigInt!
            gas: Long!
            inputData: Bytes!
            block: Block

            status: Long
            gasUsed: Long
            cumulativeGasUsed: Long
            createdContract(block: Long): Account
            logs: [Log!]
        }

        input BlockFilterCriteria {
            addresses: [Address!]
            topics: [[Bytes32!]!]
        }

        type Block {
            number: Long!
            hash: Bytes32!
            parent: Block
            nonce: Bytes!
            transactionsRoot: Bytes32!
            transactionCount: Int
            stateRoot: Bytes32!
            receiptsRoot: Bytes32!
            miner(block: Long): Account!
            extraData: Bytes!
            gasLimit: Long!
            gasUsed: Long!
            timestamp: BigInt!
            logsBloom: Bytes!
            mixHash: Bytes32!
            difficulty: BigInt!
            totalDifficulty: BigInt!
            ommerCount: Int
            ommers: [Block]
            ommerAt(index: Int!): Block
            ommerHash: Bytes32!
            transactions: [Transaction!]
            transactionAt(index: Int!): Transaction
            logs(filter: BlockFilterCriteria!): [Log!]!
        }

        input CallData {
            from: Address
            to: Address
            gas: Long
            gasPrice: BigInt
            value: BigInt
            data: Bytes
        }

        type CallResult {
            data: Bytes!
            gasUsed: Long!
            status: Long!
        }

        input FilterCriteria {
            fromBlock: Long
            toBlock: Long
            addresses: [Address!]
            topics: [[Bytes32!]!]
        }

        type SyncState{
            startingBlock: Long!
            currentBlock: Long!
            highestBlock: Long!
            pulledStates: Long
            knownStates: Long
        }

        type Query {
            account(address: Address!, blockNumber: Long): Account!
            block(number: Long, hash: Bytes32): Block
            blocks(from: Long!, to: Long): [Block!]!
            transaction(hash: Bytes32!): Transaction
            call(data: CallData!, blockNumber: Long): CallResult
            estimateGas(data: CallData!, blockNumber: Long): Long!
            logs(filter: FilterCriteria!): [Log!]!
            gasPrice: BigInt!
            protocolVersion: Int!
            syncing: SyncState
        }

        type Mutation {
            sendRawTransaction(data: Bytes!): Bytes32!
        }
    `
	schema, err := graphql.ParseSchema(s, &q)
	if err != nil {
		return nil, err
	}
	h := &relay.Handler{Schema: schema}

	mux := http.NewServeMux()
	mux.Handle("/", GraphiQL{})
	mux.Handle("/graphql", h)
	mux.Handle("/graphql/", h)
	return mux, nil
}

type Service struct {
	endpoint string
	cors     []string
	vhosts   []string
	timeouts rpc.HTTPTimeouts
	node     *node.Node
	handler  http.Handler
	listener net.Listener
}

func (s *Service) Protocols() []p2p.Protocol { return nil }

func (s *Service) APIs() []rpc.API { return nil }

// Start is called after all services have been constructed and the networking
// layer was also initialized to spawn any goroutines required by the service.
func (s *Service) Start(server *p2p.Server) error {
	var err error
	s.handler, err = NewHandler(s.node)
	if err != nil {
		return err
	}

	if s.listener, err = net.Listen("tcp", s.endpoint); err != nil {
		return err
	}

	go rpc.NewHTTPServer(s.cors, s.vhosts, s.timeouts, s.handler).Serve(s.listener)
	log.Info("GraphQL endpoint opened", "url", fmt.Sprintf("http://%s", s.endpoint))
	return nil
}

// Stop terminates all goroutines belonging to the service, blocking until they
// are all terminated.
func (s *Service) Stop() error {
	if s.listener != nil {
		s.listener.Close()
		s.listener = nil
		log.Info("GraphQL endpoint closed", "url", fmt.Sprintf("http://%s", s.endpoint))
	}
	return nil
}

func NewService(ctx *node.ServiceContext, stack *node.Node, endpoint string, cors, vhosts []string, timeouts rpc.HTTPTimeouts) (*Service, error) {
	return &Service{
		endpoint: endpoint,
		cors:     cors,
		vhosts:   vhosts,
		timeouts: timeouts,
		node:     stack,
	}, nil
}

func RegisterGraphQLService(stack *node.Node, endpoint string, cors, vhosts []string, timeouts rpc.HTTPTimeouts) error {
	return stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return NewService(ctx, stack, endpoint, cors, vhosts, timeouts)
	})
}
