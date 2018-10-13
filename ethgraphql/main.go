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

type Transaction struct {
	node *node.Node
	hash common.Hash
	tx   *types.Transaction
}

func (tx *Transaction) Hash(ctx context.Context) HexBytes {
	return HexBytes{tx.hash.Bytes()}
}

type Block struct {
	node  *node.Node
	num   *rpc.BlockNumber
	hash  common.Hash
	block *types.Block
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

func (b *Block) Coinbase(ctx context.Context, args BlockNumberArgs) (*Account, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return nil, err
	}

	blockNumber := rpc.LatestBlockNumber
	if args.Block != nil {
		blockNumber = rpc.BlockNumber(*args.Block)
	}

	return &Account{
		node:        b.node,
		address:     block.Coinbase(),
		blockNumber: blockNumber,
	}, nil
}

func (b *Block) Transactions(ctx context.Context) ([]*Transaction, error) {
	block, err := b.resolve(ctx)
	if err != nil {
		return nil, err
	}

	ret := make([]*Transaction, 0, len(block.Transactions()))
	for _, tx := range block.Transactions() {
		ret = append(ret, &Transaction{
			node: b.node,
			hash: tx.Hash(),
			tx:   tx,
		})
	}
	return ret, nil
}

type Query struct {
	node *node.Node
}

type BlockArgs struct {
	Number *int32
	Hash   *string
}

func (q *Query) Block(ctx context.Context, args BlockArgs) *Block {
	if args.Number != nil {
		num := rpc.BlockNumber(uint64(*args.Number))
		return &Block{
			node: q.node,
			num:  &num,
		}
	} else if args.Hash != nil {
		return &Block{
			node: q.node,
			hash: common.HexToHash(*args.Hash),
		}
	} else {
		num := rpc.LatestBlockNumber
		return &Block{
			node: q.node,
			num:  &num,
		}
	}
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

        type Transaction {
            hash: HexBytes!
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
            block(number: Int, hash: String): Block
            account(address: HexBytes!, blockNumber: Int): Account
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
