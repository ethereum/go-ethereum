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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
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

type HexBytes struct {
	hexutil.Bytes
}

func (h HexBytes) ImplementsGraphQLType(name string) bool { return name == "HexBytes" }

func (h *HexBytes) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		data, err := hexutil.Decode(input)
		if err != nil {
			return err
		}
		*h = HexBytes{data}
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

func (a *Account) Address(ctx context.Context) (HexBytes, error) {
	return HexBytes{a.address.Bytes()}, nil
}

func (a *Account) Balance(ctx context.Context) (BigNum, error) {
	state, err := a.getState(ctx)
	if err != nil {
		return BigNum{}, err
	}

	return BigNum{state.GetBalance(a.address)}, nil
}

func (a *Account) Nonce(ctx context.Context) (int32, error) {
	state, err := a.getState(ctx)
	if err != nil {
		return 0, err
	}

	return int32(state.GetNonce(a.address)), nil
}

func (a *Account) Code(ctx context.Context) (HexBytes, error) {
	state, err := a.getState(ctx)
	if err != nil {
		return HexBytes{}, err
	}

	return HexBytes{state.GetCode(a.address)}, nil
}

type StorageSlotArgs struct {
	Slot HexBytes
}

func (a *Account) Storage(ctx context.Context, args StorageSlotArgs) (HexBytes, error) {
	state, err := a.getState(ctx)
	if err != nil {
		return HexBytes{}, err
	}

	return HexBytes{state.GetState(a.address, common.BytesToHash(args.Slot.Bytes)).Bytes()}, nil
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

func (l *Log) Topics(ctx context.Context) []*HexBytes {
	ret := make([]*HexBytes, 0, len(l.log.Topics))
	for _, topic := range l.log.Topics {
		ret = append(ret, &HexBytes{topic.Bytes()})
	}
	return ret
}

func (l *Log) Data(ctx context.Context) HexBytes {
	return HexBytes{l.log.Data}
}

type Receipt struct {
	node        *node.Node
	transaction *Transaction
	receipt     *types.Receipt
}

func (r *Receipt) Status(ctx context.Context) int32 {
	return int32(r.receipt.Status)
}

func (r *Receipt) GasUsed(ctx context.Context) int32 {
	return int32(r.receipt.GasUsed)
}

func (r *Receipt) CumulativeGasUsed(ctx context.Context) int32 {
	return int32(r.receipt.CumulativeGasUsed)
}

func (r *Receipt) Contract(ctx context.Context, args BlockNumberArgs) *Account {
	if r.receipt.ContractAddress == (common.Address{}) {
		return nil
	}

	return &Account{
		node:        r.node,
		address:     r.receipt.ContractAddress,
		blockNumber: args.Number(),
	}
}

func (r *Receipt) Logs(ctx context.Context) []*Log {
	ret := make([]*Log, 0, len(r.receipt.Logs))
	for _, log := range r.receipt.Logs {
		ret = append(ret, &Log{
			node:        r.node,
			transaction: r.transaction,
			log:         log,
		})
	}
	return ret
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

func (tx *Transaction) Hash(ctx context.Context) HexBytes {
	return HexBytes{tx.hash.Bytes()}
}

func (t *Transaction) Data(ctx context.Context) (HexBytes, error) {
	tx, err := t.resolve(ctx)
	if err != nil || tx == nil {
		return HexBytes{}, err
	}
	return HexBytes{tx.Data()}, nil
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

func (t *Transaction) Receipt(ctx context.Context) (*Receipt, error) {
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

	return &Receipt{
		node:        t.node,
		transaction: t,
		receipt:     receipts[t.index],
	}, nil
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
		b.block, err = be.BlockByNumber(ctx, rpc.BlockNumber(*b.num))
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

func (b *Block) Hash(ctx context.Context) (HexBytes, error) {
	if b.hash == (common.Hash{}) {
		block, err := b.resolve(ctx)
		if err != nil {
			return HexBytes{}, err
		}
		b.hash = block.Hash()
	}
	return HexBytes{b.hash.Bytes()}, nil
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

func (b *Block) Time(ctx context.Context) (BigNum, error) {
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

func (b *Block) MixDigest(ctx context.Context) (HexBytes, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return HexBytes{}, err
	}
	return HexBytes{block.MixDigest().Bytes()}, nil
}

func (b *Block) Root(ctx context.Context) (HexBytes, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return HexBytes{}, err
	}
	return HexBytes{block.Root().Bytes()}, nil
}

func (b *Block) TxHash(ctx context.Context) (HexBytes, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return HexBytes{}, err
	}
	return HexBytes{block.TxHash().Bytes()}, nil
}

func (b *Block) ReceiptHash(ctx context.Context) (HexBytes, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return HexBytes{}, err
	}
	return HexBytes{block.ReceiptHash().Bytes()}, nil
}

func (b *Block) UncleHash(ctx context.Context) (HexBytes, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return HexBytes{}, err
	}
	return HexBytes{block.UncleHash().Bytes()}, nil
}

func (b *Block) Extra(ctx context.Context) (HexBytes, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return HexBytes{}, err
	}
	return HexBytes{block.Extra()}, nil
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

func (b *Block) Coinbase(ctx context.Context, args BlockNumberArgs) (*Account, error) {
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

type Query struct {
	node *node.Node
}

type BlockArgs struct {
	Number *int32
	Hash   *HexBytes
}

func (q *Query) Block(ctx context.Context, args BlockArgs) (*Block, error) {
	var block *Block
	if args.Number != nil {
		num := rpc.BlockNumber(uint64(*args.Number))
		block = &Block{
			node: q.node,
			num:  &num,
		}
	} else if args.Hash != nil {
		block = &Block{
			node: q.node,
			hash: common.BytesToHash(args.Hash.Bytes),
		}
	} else {
		num := rpc.LatestBlockNumber
		block = &Block{
			node: q.node,
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

type AccountArgs struct {
	Address     HexBytes
	BlockNumber *int32
}

func (q *Query) Account(ctx context.Context, args AccountArgs) *Account {
	blockNumber := rpc.LatestBlockNumber
	if args.BlockNumber != nil {
		blockNumber = rpc.BlockNumber(*args.BlockNumber)
	}

	return &Account{
		node:        q.node,
		address:     common.BytesToAddress(args.Address.Bytes),
		blockNumber: blockNumber,
	}
}

type TransactionArgs struct {
	Hash HexBytes
}

func (q *Query) Transaction(ctx context.Context, args TransactionArgs) (*Transaction, error) {
	tx := &Transaction{
		node: q.node,
		hash: common.BytesToHash(args.Hash.Bytes),
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

func NewHandler(n *node.Node) (http.Handler, error) {
	q := Query{n}

	s := `
        scalar HexBytes
        scalar BigNum

        schema {
            query: Query
        }

        type Account {
            address: HexBytes!
            balance: BigNum!
            nonce: Int!
            code: HexBytes!
            storage(slot: HexBytes!): HexBytes!
        }

        type Log {
            transaction: Transaction!
            account(block: Int): Account!
            topics: [HexBytes]!
            data: HexBytes!
        }

        type Receipt {
            status: Int!
            gasUsed: Int!
            cumulativeGasUsed: Int!
            contract(block: Int): Account
            logs: [Log]!
        }

        type Transaction {
            hash: HexBytes!
            data: HexBytes!
            gas: Int!
            gasPrice: BigNum!
            value: BigNum!
            nonce: Int!
            to(block: Int): Account
            from(block: Int): Account!
            block: Block
            index: Int
            receipt: Receipt
        }

        type Block {
            number: Int!
            hash: HexBytes!
            gasLimit: Int!
            gasUsed: Int!
            parent: Block
            difficulty: BigNum!
            time: BigNum!
            nonce: BigNum!
            mixDigest: HexBytes!
            root: HexBytes!
            txHash: HexBytes!
            receiptHash: HexBytes!
            uncleHash: HexBytes!
            extra: HexBytes!
            totalDifficulty: BigNum!
            coinbase(block: Int): Account!
            transactions: [Transaction]!
        }

        type Query {
            block(number: Int, hash: HexBytes): Block
            account(address: HexBytes!, blockNumber: Int): Account!
            transaction(hash: HexBytes!): Transaction
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
