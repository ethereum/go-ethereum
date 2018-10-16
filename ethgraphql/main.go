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
	"math/big"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

func getBackend(n *node.Node) (ethapi.Backend, error) {
	var ethereum *eth.Ethereum
	if err := n.Service(&ethereum); err != nil {
		return nil, err
	}
	return ethereum.APIBackend, nil
}

type Bytes32 struct {
	common.Hash
}

func (_ Bytes32) ImplementsGraphQLType(name string) bool { return name == "Bytes32" }

func (b *Bytes32) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		*b = Bytes32{common.HexToHash(input)}
	default:
		err = fmt.Errorf("Unexpected type for Hash: %v", input)
	}
	return err
}

type Address struct {
	common.Address
}

func (h Address) ImplementsGraphQLType(name string) bool { return name == "Address" }

func (h *Address) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		*h = Address{common.HexToAddress(input)}
	default:
		err = fmt.Errorf("Unexpected type for Hash: %v", input)
	}
	return err
}

type BigNum struct {
	*big.Int
}

func (bn BigNum) ImplementsGraphQLType(name string) bool { return name == "BigNum" }

func (bn *BigNum) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		i := big.NewInt(0)
		i.SetString(input, 10)
		*bn = BigNum{i}
	case int32:
		*bn = BigNum{big.NewInt(int64(input))}
	default:
		err = fmt.Errorf("Unexpected type for Hash: %v", input)
	}
	return err
}

func (bn BigNum) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, bn.Text(10)), nil
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

func (a *Account) Address(ctx context.Context) (Address, error) {
	return Address{a.address}, nil
}

func (a *Account) Balance(ctx context.Context) (BigNum, error) {
	state, err := a.getState(ctx)
	if err != nil {
		return BigNum{}, err
	}

	return BigNum{state.GetBalance(a.address)}, nil
}

func (a *Account) TransactionCount(ctx context.Context) (int32, error) {
	state, err := a.getState(ctx)
	if err != nil {
		return 0, err
	}

	return int32(state.GetNonce(a.address)), nil
}

func (a *Account) Code(ctx context.Context) (hexutil.Bytes, error) {
	state, err := a.getState(ctx)
	if err != nil {
		return hexutil.Bytes{}, err
	}

	return hexutil.Bytes(state.GetCode(a.address)), nil
}

type StorageSlotArgs struct {
	Slot Bytes32
}

func (a *Account) Storage(ctx context.Context, args StorageSlotArgs) (Bytes32, error) {
	state, err := a.getState(ctx)
	if err != nil {
		return Bytes32{}, err
	}

	return Bytes32{state.GetState(a.address, args.Slot.Hash)}, nil
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

func (l *Log) Topics(ctx context.Context) []Bytes32 {
	ret := make([]Bytes32, 0, len(l.log.Topics))
	for _, topic := range l.log.Topics {
		ret = append(ret, Bytes32{topic})
	}
	return ret
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

func (tx *Transaction) Hash(ctx context.Context) Bytes32 {
	return Bytes32{tx.hash}
}

func (t *Transaction) InputData(ctx context.Context) (hexutil.Bytes, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return hexutil.Bytes{}, err
	}
	return hexutil.Bytes(tx.Data()), nil
}

func (t *Transaction) Gas(ctx context.Context) (int32, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return 0, err
	}
	return int32(tx.Gas()), nil
}

func (t *Transaction) GasPrice(ctx context.Context) (BigNum, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return BigNum{}, err
	}
	return BigNum{tx.GasPrice()}, nil
}

func (t *Transaction) Value(ctx context.Context) (BigNum, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return BigNum{}, err
	}
	return BigNum{tx.Value()}, nil
}

func (t *Transaction) Nonce(ctx context.Context) (int32, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return 0, err
	}
	return int32(tx.Nonce()), nil
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

func (t *Transaction) Status(ctx context.Context) (*int32, error) {
	receipt, err := t.getReceipt(ctx)
	if err != nil || receipt == nil {
		return nil, err
	}

	ret := int32(receipt.Status)
	return &ret, nil
}

func (t *Transaction) GasUsed(ctx context.Context) (*int32, error) {
	receipt, err := t.getReceipt(ctx)
	if err != nil || receipt == nil {
		return nil, err
	}

	ret := int32(receipt.GasUsed)
	return &ret, nil
}

func (t *Transaction) CumulativeGasUsed(ctx context.Context) (*int32, error) {
	receipt, err := t.getReceipt(ctx)
	if err != nil || receipt == nil {
		return nil, err
	}

	ret := int32(receipt.CumulativeGasUsed)
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

	if b.num != nil {
		b.block, err = be.BlockByNumber(ctx, *b.num)
	} else {
		b.block, err = be.GetBlock(ctx, b.hash)
	}
	return b.block, err
}

func (b *Block) resolveReceipts(ctx context.Context) ([]*types.Receipt, error) {
	if b.receipts == nil {
		be, err := getBackend(b.node)
		if err != nil {
			return nil, err
		}

		hash := b.hash
		if hash == (common.Hash{}) {
			block, err := b.resolve(ctx)
			if err != nil {
				return nil, err
			}
			hash = block.Hash()
		}

		receipts, err := be.GetReceipts(ctx, hash)
		if err != nil {
			return nil, err
		}
		b.receipts = []*types.Receipt(receipts)
	}
	return b.receipts, nil
}

func (b *Block) Number(ctx context.Context) (int32, error) {
	if b.num == nil || *b.num == rpc.LatestBlockNumber {
		block, err := b.resolve(ctx)
		if err != nil {
			return 0, err
		}
		num := rpc.BlockNumber(block.Number().Uint64())
		b.num = &num
	}
	return int32(*b.num), nil
}

func (b *Block) Hash(ctx context.Context) (Bytes32, error) {
	if b.hash == (common.Hash{}) {
		block, err := b.resolve(ctx)
		if err != nil {
			return Bytes32{}, err
		}
		b.hash = block.Hash()
	}
	return Bytes32{b.hash}, nil
}

func (b *Block) GasLimit(ctx context.Context) (int32, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return 0, err
	}
	return int32(block.GasLimit()), nil
}

func (b *Block) GasUsed(ctx context.Context) (int32, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return 0, err
	}
	return int32(block.GasUsed()), nil
}

func (b *Block) Parent(ctx context.Context) (*Block, error) {
	// If the block hasn't been fetched, and we'll need it, fetch it.
	if b.num == nil && b.hash != (common.Hash{}) && b.block == nil {
		if _, err := b.resolve(ctx); err != nil {
			return nil, err
		}
	}

	if b.block != nil && b.block.NumberU64() > 0 {
		num := rpc.BlockNumber(b.block.NumberU64() - 1)
		return &Block{
			node: b.node,
			num:  &num,
			hash: b.block.ParentHash(),
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

func (b *Block) Difficulty(ctx context.Context) (BigNum, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return BigNum{}, err
	}
	return BigNum{block.Difficulty()}, nil
}

func (b *Block) Timestamp(ctx context.Context) (BigNum, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return BigNum{}, err
	}
	return BigNum{block.Time()}, nil
}

func (b *Block) Nonce(ctx context.Context) (BigNum, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return BigNum{}, err
	}
	i := new(big.Int)
	i.SetUint64(block.Nonce())
	return BigNum{i}, nil
}

func (b *Block) MixHash(ctx context.Context) (Bytes32, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return Bytes32{}, err
	}
	return Bytes32{block.MixDigest()}, nil
}

func (b *Block) TransactionsRoot(ctx context.Context) (Bytes32, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return Bytes32{}, err
	}
	return Bytes32{block.TxHash()}, nil
}

func (b *Block) StateRoot(ctx context.Context) (Bytes32, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return Bytes32{}, err
	}
	return Bytes32{block.Root()}, nil
}

func (b *Block) ReceiptsRoot(ctx context.Context) (Bytes32, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return Bytes32{}, err
	}
	return Bytes32{block.ReceiptHash()}, nil
}

func (b *Block) OmmerHash(ctx context.Context) (Bytes32, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return Bytes32{}, err
	}
	return Bytes32{block.UncleHash()}, nil
}

func (b *Block) OmmerCount(ctx context.Context) (int32, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return 0, err
	}
	return int32(len(block.Uncles())), nil
}

func (b *Block) Ommers(ctx context.Context) ([]*Block, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return nil, err
	}

	ret := make([]*Block, 0, len(block.Uncles()))
	for _, uncle := range block.Uncles() {
		blockNumber := rpc.BlockNumber(uncle.Number.Uint64())
		ret = append(ret, &Block{
			node: b.node,
			num:  &blockNumber,
			hash: uncle.Hash(),
		})
	}
	return ret, nil
}

func (b *Block) ExtraData(ctx context.Context) (hexutil.Bytes, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return hexutil.Bytes{}, err
	}
	return hexutil.Bytes(block.Extra()), nil
}

func (b *Block) LogsBloom(ctx context.Context) (hexutil.Bytes, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return hexutil.Bytes{}, err
	}
	return hexutil.Bytes(block.Bloom().Bytes()), nil
}

func (b *Block) TotalDifficulty(ctx context.Context) (BigNum, error) {
	h := b.hash
	if h == (common.Hash{}) {
		block, err := b.resolve(ctx)
		if err != nil {
			return BigNum{}, err
		}
		h = block.Hash()
	}

	be, err := getBackend(b.node)
	if err != nil {
		return BigNum{}, err
	}

	return BigNum{be.GetTd(h)}, nil
}

type BlockNumberArgs struct {
	Block *int32
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

func (b *Block) TransactionCount(ctx context.Context) (int32, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return 0, err
	}
	return int32(len(block.Transactions())), nil
}

func (b *Block) Transactions(ctx context.Context) ([]*Transaction, error) {
	block, err := b.resolve(ctx)
	if err != nil {
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
	return ret, nil
}

type ArrayIndexArgs struct {
	Index int32
}

func (b *Block) TransactionAt(ctx context.Context, args ArrayIndexArgs) (*Transaction, error) {
	block, err := b.resolve(ctx)
	if err != nil {
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
	if err != nil {
		return nil, err
	}

	uncles := block.Uncles()
	if args.Index < 0 || int(args.Index) >= len(uncles) {
		return nil, nil
	}

	uncle := uncles[args.Index]
	blockNumber := rpc.BlockNumber(uncle.Number.Uint64())
	return &Block{
		node: b.node,
		num:  &blockNumber,
		hash: uncle.Hash(),
	}, nil
}

type Resolver struct {
	node *node.Node
}

type BlockArgs struct {
	Number *int32
	Hash   *Bytes32
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
			hash: args.Hash.Hash,
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
	From int32
	To   *int32
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
	Address     Address
	BlockNumber *int32
}

func (r *Resolver) Account(ctx context.Context, args AccountArgs) *Account {
	blockNumber := rpc.LatestBlockNumber
	if args.BlockNumber != nil {
		blockNumber = rpc.BlockNumber(*args.BlockNumber)
	}

	return &Account{
		node:        r.node,
		address:     args.Address.Address,
		blockNumber: blockNumber,
	}
}

type TransactionArgs struct {
	Hash Bytes32
}

func (r *Resolver) Transaction(ctx context.Context, args TransactionArgs) (*Transaction, error) {
	tx := &Transaction{
		node: r.node,
		hash: args.Hash.Hash,
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

func (r *Resolver) SendRawTransaction(ctx context.Context, args struct{ Data hexutil.Bytes }) (Bytes32, error) {
	be, err := getBackend(r.node)
	if err != nil {
		return Bytes32{}, err
	}

	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(args.Data, tx); err != nil {
		return Bytes32{}, err
	}
	hash, err := ethapi.SubmitTransaction(ctx, be, tx)
	return Bytes32{hash}, err
}

type CallData struct {
	From        *Address
	To          *Address
	Gas         *int32
	GasPrice    *BigNum
	Value       *BigNum
	Data        *hexutil.Bytes
	BlockNumber *int32
}

type CallResult struct {
	data    hexutil.Bytes
	gasUsed int32
	status  int32
}

func (c *CallResult) Data() hexutil.Bytes {
	return c.data
}

func (c *CallResult) GasUsed() int32 {
	return c.gasUsed
}

func (c *CallResult) Status() int32 {
	return c.status
}

func convertCallData(data CallData) (ethapi.CallArgs, rpc.BlockNumber) {
	callArgs := ethapi.CallArgs{}
	if data.From != nil {
		callArgs.From = data.From.Address
	}
	if data.To != nil {
		addr := data.To.Address
		callArgs.To = &addr
	}
	if data.Gas != nil {
		callArgs.Gas = hexutil.Uint64(*data.Gas)
	}
	if data.GasPrice != nil {
		callArgs.GasPrice = hexutil.Big(*data.GasPrice.Int)
	}
	if data.Value != nil {
		callArgs.Value = hexutil.Big(*data.Value.Int)
	}
	if data.Data != nil {
		callArgs.Data = *data.Data
	}

	blockNumber := rpc.LatestBlockNumber
	if data.BlockNumber != nil {
		blockNumber = rpc.BlockNumber(*data.BlockNumber)
	}

	return callArgs, blockNumber
}

func (r *Resolver) Call(ctx context.Context, args struct{ Data CallData }) (*CallResult, error) {
	be, err := getBackend(r.node)
	if err != nil {
		return nil, err
	}

	callArgs, blockNumber := convertCallData(args.Data)

	result, gas, failed, err := ethapi.DoCall(ctx, be, callArgs, blockNumber, vm.Config{}, 5*time.Second)
	status := int32(1)
	if failed {
		status = 0
	}
	return &CallResult{
		data:    hexutil.Bytes(result),
		gasUsed: int32(gas),
		status:  status,
	}, err
}

func (r *Resolver) EstimateGas(ctx context.Context, args struct{ Data CallData }) (int32, error) {
	be, err := getBackend(r.node)
	if err != nil {
		return 0, err
	}

	callArgs, blockNumber := convertCallData(args.Data)

	gas, err := ethapi.DoEstimateGas(ctx, be, callArgs, blockNumber)
	return int32(gas), err
}

func NewHandler(n *node.Node) (http.Handler, error) {
	q := Resolver{n}

	s := `
        scalar Bytes32
        scalar Address
        scalar Bytes
        scalar BigNum

        schema {
            query: Query
            mutation: Mutation
        }

        type Account {
            address: Address!
            balance: BigNum!
            transactionCount: Int!
            code: Bytes!
            storage(slot: Bytes32!): Bytes32!
        }

        type Log {
            index: Int!
            account(block: Int): Account!
            topics: [Bytes32!]!
            data: Bytes!
            transaction: Transaction!
        }

        type Transaction {
            hash: Bytes32!
            nonce: Int!
            index: Int
            from(block: Int): Account!
            to(block: Int): Account
            value: BigNum!
            gasPrice: BigNum!
            gas: Int!
            inputData: Bytes!
            block: Block

            status: Int
            gasUsed: Int
            cumulativeGasUsed: Int
            createdContract(block: Int): Account
            logs: [Log!]
        }

        type Block {
            number: Int!
            hash: Bytes32!
            parent: Block
            nonce: BigNum!
            transactionsRoot: Bytes32!
            transactionCount: Int!
            stateRoot: Bytes32!
            receiptsRoot: Bytes32!
            miner(block: Int): Account!
            extraData: Bytes!
            gasLimit: Int!
            gasUsed: Int!
            timestamp: BigNum!
            logsBloom: Bytes!
            mixHash: Bytes32!
            difficulty: BigNum!
            totalDifficulty: BigNum!
            ommerCount: Int!
            ommers: [Block]!
            ommerAt(index: Int!): Block
            ommerHash: Bytes32!
            transactions: [Transaction!]!
            transactionAt(index: Int!): Transaction
        }

        input CallData {
            from: Address
            to: Address
            gas: Int
            gasPrice: BigNum
            value: BigNum
            data: Bytes
            blockNumber: Int
        }

        type CallResult {
            data: Bytes!
            gasUsed: Int!
            status: Int!
        }

        type Query {
            account(address: Address!, blockNumber: Int): Account!
            block(number: Int, hash: Bytes32): Block
            blocks(from: Int!, to: Int): [Block!]!
            transaction(hash: Bytes32!): Transaction
            call(data: CallData!): CallResult
            estimateGas(data: CallData!): Int!
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
