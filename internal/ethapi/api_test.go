// Copyright 2023 The go-ethereum Authors
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

package ethapi

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/bloombits"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/internal/blocktest"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

func TestTransaction_RoundTripRpcJSON(t *testing.T) {
	var (
		config = params.AllEthashProtocolChanges
		signer = types.LatestSigner(config)
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		tests  = allTransactionTypes(common.Address{0xde, 0xad}, config)
	)
	t.Parallel()
	for i, tt := range tests {
		var tx2 types.Transaction
		tx, err := types.SignNewTx(key, signer, tt.Tx)
		if err != nil {
			t.Fatalf("test %d: signing failed: %v", i, err)
		}
		// Regular transaction
		if data, err := json.Marshal(tx); err != nil {
			t.Fatalf("test %d: marshalling failed; %v", i, err)
		} else if err = tx2.UnmarshalJSON(data); err != nil {
			t.Fatalf("test %d: sunmarshal failed: %v", i, err)
		} else if want, have := tx.Hash(), tx2.Hash(); want != have {
			t.Fatalf("test %d: stx changed, want %x have %x", i, want, have)
		}

		// rpcTransaction
		rpcTx := newRPCTransaction(tx, common.Hash{}, 0, 0, 0, nil, config)
		if data, err := json.Marshal(rpcTx); err != nil {
			t.Fatalf("test %d: marshalling failed; %v", i, err)
		} else if err = tx2.UnmarshalJSON(data); err != nil {
			t.Fatalf("test %d: unmarshal failed: %v", i, err)
		} else if want, have := tx.Hash(), tx2.Hash(); want != have {
			t.Fatalf("test %d: tx changed, want %x have %x", i, want, have)
		} else {
			want, have := tt.Want, string(data)
			require.JSONEqf(t, want, have, "test %d: rpc json not match, want %s have %s", i, want, have)
		}
	}
}

type txData struct {
	Tx   types.TxData
	Want string
}

func allTransactionTypes(addr common.Address, config *params.ChainConfig) []txData {
	return []txData{
		{
			Tx: &types.LegacyTx{
				Nonce:    5,
				GasPrice: big.NewInt(6),
				Gas:      7,
				To:       &addr,
				Value:    big.NewInt(8),
				Data:     []byte{0, 1, 2, 3, 4},
				V:        big.NewInt(9),
				R:        big.NewInt(10),
				S:        big.NewInt(11),
			},
			Want: `{"blockHash":null,"blockNumber":null,"from":"0x71562b71999873db5b286df957af199ec94617f7","gas":"0x7","gasPrice":"0x6","hash":"0x5f3240454cd09a5d8b1c5d651eefae7a339262875bcd2d0e6676f3d989967008","input":"0x0001020304","nonce":"0x5","to":"0xdead000000000000000000000000000000000000","transactionIndex":null,"value":"0x8","type":"0x0","chainId":"0x539","v":"0xa96","r":"0xbc85e96592b95f7160825d837abb407f009df9ebe8f1b9158a4b8dd093377f75","s":"0x1b55ea3af5574c536967b039ba6999ef6c89cf22fc04bcb296e0e8b0b9b576f5"}`,
		}, {
			Tx: &types.LegacyTx{
				Nonce:    5,
				GasPrice: big.NewInt(6),
				Gas:      7,
				To:       nil,
				Value:    big.NewInt(8),
				Data:     []byte{0, 1, 2, 3, 4},
				V:        big.NewInt(32),
				R:        big.NewInt(10),
				S:        big.NewInt(11),
			},
			Want: `{"blockHash":null,"blockNumber":null,"from":"0x71562b71999873db5b286df957af199ec94617f7","gas":"0x7","gasPrice":"0x6","hash":"0x806e97f9d712b6cb7e781122001380a2837531b0fc1e5f5d78174ad4cb699873","input":"0x0001020304","nonce":"0x5","to":null,"transactionIndex":null,"value":"0x8","type":"0x0","chainId":"0x539","v":"0xa96","r":"0x9dc28b267b6ad4e4af6fe9289668f9305c2eb7a3241567860699e478af06835a","s":"0xa0b51a071aa9bed2cd70aedea859779dff039e3630ea38497d95202e9b1fec7"}`,
		},
		{
			Tx: &types.AccessListTx{
				ChainID:  config.ChainID,
				Nonce:    5,
				GasPrice: big.NewInt(6),
				Gas:      7,
				To:       &addr,
				Value:    big.NewInt(8),
				Data:     []byte{0, 1, 2, 3, 4},
				AccessList: types.AccessList{
					types.AccessTuple{
						Address:     common.Address{0x2},
						StorageKeys: []common.Hash{types.EmptyRootHash},
					},
				},
				V: big.NewInt(32),
				R: big.NewInt(10),
				S: big.NewInt(11),
			},
			Want: `{"blockHash":null,"blockNumber":null,"from":"0x71562b71999873db5b286df957af199ec94617f7","gas":"0x7","gasPrice":"0x6","hash":"0x121347468ee5fe0a29f02b49b4ffd1c8342bc4255146bb686cd07117f79e7129","input":"0x0001020304","nonce":"0x5","to":"0xdead000000000000000000000000000000000000","transactionIndex":null,"value":"0x8","type":"0x1","accessList":[{"address":"0x0200000000000000000000000000000000000000","storageKeys":["0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"]}],"chainId":"0x539","v":"0x0","r":"0xf372ad499239ae11d91d34c559ffc5dab4daffc0069e03afcabdcdf231a0c16b","s":"0x28573161d1f9472fa0fd4752533609e72f06414f7ab5588699a7141f65d2abf"}`,
		}, {
			Tx: &types.AccessListTx{
				ChainID:  config.ChainID,
				Nonce:    5,
				GasPrice: big.NewInt(6),
				Gas:      7,
				To:       nil,
				Value:    big.NewInt(8),
				Data:     []byte{0, 1, 2, 3, 4},
				AccessList: types.AccessList{
					types.AccessTuple{
						Address:     common.Address{0x2},
						StorageKeys: []common.Hash{types.EmptyRootHash},
					},
				},
				V: big.NewInt(32),
				R: big.NewInt(10),
				S: big.NewInt(11),
			},
			Want: `{"blockHash":null,"blockNumber":null,"from":"0x71562b71999873db5b286df957af199ec94617f7","gas":"0x7","gasPrice":"0x6","hash":"0x067c3baebede8027b0f828a9d933be545f7caaec623b00684ac0659726e2055b","input":"0x0001020304","nonce":"0x5","to":null,"transactionIndex":null,"value":"0x8","type":"0x1","accessList":[{"address":"0x0200000000000000000000000000000000000000","storageKeys":["0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"]}],"chainId":"0x539","v":"0x1","r":"0x542981b5130d4613897fbab144796cb36d3cb3d7807d47d9c7f89ca7745b085c","s":"0x7425b9dd6c5deaa42e4ede35d0c4570c4624f68c28d812c10d806ffdf86ce63"}`,
		}, {
			Tx: &types.DynamicFeeTx{
				ChainID:   config.ChainID,
				Nonce:     5,
				GasTipCap: big.NewInt(6),
				GasFeeCap: big.NewInt(9),
				Gas:       7,
				To:        &addr,
				Value:     big.NewInt(8),
				Data:      []byte{0, 1, 2, 3, 4},
				AccessList: types.AccessList{
					types.AccessTuple{
						Address:     common.Address{0x2},
						StorageKeys: []common.Hash{types.EmptyRootHash},
					},
				},
				V: big.NewInt(32),
				R: big.NewInt(10),
				S: big.NewInt(11),
			},
			Want: `{"blockHash":null,"blockNumber":null,"from":"0x71562b71999873db5b286df957af199ec94617f7","gas":"0x7","gasPrice":"0x9","maxFeePerGas":"0x9","maxPriorityFeePerGas":"0x6","hash":"0xb63e0b146b34c3e9cb7fbabb5b3c081254a7ded6f1b65324b5898cc0545d79ff","input":"0x0001020304","nonce":"0x5","to":"0xdead000000000000000000000000000000000000","transactionIndex":null,"value":"0x8","type":"0x2","accessList":[{"address":"0x0200000000000000000000000000000000000000","storageKeys":["0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"]}],"chainId":"0x539","v":"0x1","r":"0x3b167e05418a8932cd53d7578711fe1a76b9b96c48642402bb94978b7a107e80","s":"0x22f98a332d15ea2cc80386c1ebaa31b0afebfa79ebc7d039a1e0074418301fef"}`,
		}, {
			Tx: &types.DynamicFeeTx{
				ChainID:    config.ChainID,
				Nonce:      5,
				GasTipCap:  big.NewInt(6),
				GasFeeCap:  big.NewInt(9),
				Gas:        7,
				To:         nil,
				Value:      big.NewInt(8),
				Data:       []byte{0, 1, 2, 3, 4},
				AccessList: types.AccessList{},
				V:          big.NewInt(32),
				R:          big.NewInt(10),
				S:          big.NewInt(11),
			},
			Want: `{"blockHash":null,"blockNumber":null,"from":"0x71562b71999873db5b286df957af199ec94617f7","gas":"0x7","gasPrice":"0x9","maxFeePerGas":"0x9","maxPriorityFeePerGas":"0x6","hash":"0xcbab17ee031a9d5b5a09dff909f0a28aedb9b295ac0635d8710d11c7b806ec68","input":"0x0001020304","nonce":"0x5","to":null,"transactionIndex":null,"value":"0x8","type":"0x2","accessList":[],"chainId":"0x539","v":"0x0","r":"0x6446b8a682db7e619fc6b4f6d1f708f6a17351a41c7fbd63665f469bc78b41b9","s":"0x7626abc15834f391a117c63450047309dbf84c5ce3e8e609b607062641e2de43"}`,
		},
	}
}

type testBackend struct {
	db      ethdb.Database
	chain   *core.BlockChain
	pending *types.Block
}

func newTestBackend(t *testing.T, n int, gspec *core.Genesis, generator func(i int, b *core.BlockGen)) *testBackend {
	var (
		engine  = ethash.NewFaker()
		backend = &testBackend{
			db: rawdb.NewMemoryDatabase(),
		}
		cacheConfig = &core.CacheConfig{
			TrieCleanLimit:    256,
			TrieDirtyLimit:    256,
			TrieTimeLimit:     5 * time.Minute,
			SnapshotLimit:     0,
			TrieDirtyDisabled: true, // Archive mode
		}
	)
	// Generate blocks for testing
	_, blocks, _ := core.GenerateChainWithGenesis(gspec, engine, n, generator)
	chain, err := core.NewBlockChain(backend.db, cacheConfig, gspec, nil, engine, vm.Config{}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create tester chain: %v", err)
	}
	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d: failed to insert into chain: %v", n, err)
	}
	backend.chain = chain
	return backend
}

func (b *testBackend) setPendingBlock(block *types.Block) {
	b.pending = block
}

func (b testBackend) SyncProgress() ethereum.SyncProgress { return ethereum.SyncProgress{} }
func (b testBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (b testBackend) FeeHistory(ctx context.Context, blockCount uint64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*big.Int, [][]*big.Int, []*big.Int, []float64, error) {
	return nil, nil, nil, nil, nil
}
func (b testBackend) ChainDb() ethdb.Database           { return b.db }
func (b testBackend) AccountManager() *accounts.Manager { return nil }
func (b testBackend) ExtRPCEnabled() bool               { return false }
func (b testBackend) RPCGasCap() uint64                 { return 10000000 }
func (b testBackend) RPCEVMTimeout() time.Duration      { return time.Second }
func (b testBackend) RPCTxFeeCap() float64              { return 0 }
func (b testBackend) UnprotectedAllowed() bool          { return false }
func (b testBackend) SetHead(number uint64)             {}
func (b testBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	if number == rpc.LatestBlockNumber {
		return b.chain.CurrentBlock(), nil
	}
	if number == rpc.PendingBlockNumber && b.pending != nil {
		return b.pending.Header(), nil
	}
	return b.chain.GetHeaderByNumber(uint64(number)), nil
}
func (b testBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.chain.GetHeaderByHash(hash), nil
}
func (b testBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.HeaderByNumber(ctx, blockNr)
	}
	if blockHash, ok := blockNrOrHash.Hash(); ok {
		return b.HeaderByHash(ctx, blockHash)
	}
	panic("unknown type rpc.BlockNumberOrHash")
}
func (b testBackend) CurrentHeader() *types.Header { return b.chain.CurrentBlock() }
func (b testBackend) CurrentBlock() *types.Header  { return b.chain.CurrentBlock() }
func (b testBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	if number == rpc.LatestBlockNumber {
		head := b.chain.CurrentBlock()
		return b.chain.GetBlock(head.Hash(), head.Number.Uint64()), nil
	}
	if number == rpc.PendingBlockNumber {
		return b.pending, nil
	}
	return b.chain.GetBlockByNumber(uint64(number)), nil
}
func (b testBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.chain.GetBlockByHash(hash), nil
}
func (b testBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.BlockByNumber(ctx, blockNr)
	}
	if blockHash, ok := blockNrOrHash.Hash(); ok {
		return b.BlockByHash(ctx, blockHash)
	}
	panic("unknown type rpc.BlockNumberOrHash")
}
func (b testBackend) GetBody(ctx context.Context, hash common.Hash, number rpc.BlockNumber) (*types.Body, error) {
	return b.chain.GetBlock(hash, uint64(number.Int64())).Body(), nil
}
func (b testBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	if number == rpc.PendingBlockNumber {
		panic("pending state not implemented")
	}
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	stateDb, err := b.chain.StateAt(header.Root)
	return stateDb, header, err
}
func (b testBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.StateAndHeaderByNumber(ctx, blockNr)
	}
	panic("only implemented for number")
}
func (b testBackend) PendingBlockAndReceipts() (*types.Block, types.Receipts) { panic("implement me") }
func (b testBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	panic("implement me")
}
func (b testBackend) GetTd(ctx context.Context, hash common.Hash) *big.Int {
	if b.pending != nil && hash == b.pending.Hash() {
		return nil
	}
	return big.NewInt(1)
}
func (b testBackend) GetEVM(ctx context.Context, msg *core.Message, state *state.StateDB, header *types.Header, vmConfig *vm.Config, blockContext *vm.BlockContext) (*vm.EVM, func() error) {
	vmError := func() error { return nil }
	if vmConfig == nil {
		vmConfig = b.chain.GetVMConfig()
	}
	txContext := core.NewEVMTxContext(msg)
	context := core.NewEVMBlockContext(header, b.chain, nil)
	if blockContext != nil {
		context = *blockContext
	}
	return vm.NewEVM(context, txContext, state, b.chain.Config(), *vmConfig), vmError
}
func (b testBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	panic("implement me")
}
func (b testBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	panic("implement me")
}
func (b testBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	panic("implement me")
}
func (b testBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	panic("implement me")
}
func (b testBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	panic("implement me")
}
func (b testBackend) GetPoolTransactions() (types.Transactions, error)         { panic("implement me") }
func (b testBackend) GetPoolTransaction(txHash common.Hash) *types.Transaction { panic("implement me") }
func (b testBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	panic("implement me")
}
func (b testBackend) Stats() (pending int, queued int) { panic("implement me") }
func (b testBackend) TxPoolContent() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction) {
	panic("implement me")
}
func (b testBackend) TxPoolContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction) {
	panic("implement me")
}
func (b testBackend) SubscribeNewTxsEvent(events chan<- core.NewTxsEvent) event.Subscription {
	panic("implement me")
}
func (b testBackend) ChainConfig() *params.ChainConfig { return b.chain.Config() }
func (b testBackend) Engine() consensus.Engine         { return b.chain.Engine() }
func (b testBackend) GetLogs(ctx context.Context, blockHash common.Hash, number uint64) ([][]*types.Log, error) {
	panic("implement me")
}
func (b testBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	panic("implement me")
}
func (b testBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	panic("implement me")
}
func (b testBackend) SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription {
	panic("implement me")
}
func (b testBackend) BloomStatus() (uint64, uint64) { panic("implement me") }
func (b testBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	panic("implement me")
}

func TestEstimateGas(t *testing.T) {
	t.Parallel()
	// Initialize test accounts
	var (
		accounts = newAccounts(2)
		genesis  = &core.Genesis{
			Config: params.TestChainConfig,
			Alloc: core.GenesisAlloc{
				accounts[0].addr: {Balance: big.NewInt(params.Ether)},
				accounts[1].addr: {Balance: big.NewInt(params.Ether)},
			},
		}
		genBlocks      = 10
		signer         = types.HomesteadSigner{}
		randomAccounts = newAccounts(2)
	)
	api := NewBlockChainAPI(newTestBackend(t, genBlocks, genesis, func(i int, b *core.BlockGen) {
		// Transfer from account[0] to account[1]
		//    value: 1000 wei
		//    fee:   0 wei
		tx, _ := types.SignTx(types.NewTx(&types.LegacyTx{Nonce: uint64(i), To: &accounts[1].addr, Value: big.NewInt(1000), Gas: params.TxGas, GasPrice: b.BaseFee(), Data: nil}), signer, accounts[0].key)
		b.AddTx(tx)
	}))
	var testSuite = []struct {
		blockNumber rpc.BlockNumber
		call        TransactionArgs
		expectErr   error
		want        uint64
	}{
		// simple transfer on latest block
		{
			blockNumber: rpc.LatestBlockNumber,
			call: TransactionArgs{
				From:  &accounts[0].addr,
				To:    &accounts[1].addr,
				Value: (*hexutil.Big)(big.NewInt(1000)),
			},
			expectErr: nil,
			want:      21000,
		},
		// simple transfer with insufficient funds on latest block
		{
			blockNumber: rpc.LatestBlockNumber,
			call: TransactionArgs{
				From:  &randomAccounts[0].addr,
				To:    &accounts[1].addr,
				Value: (*hexutil.Big)(big.NewInt(1000)),
			},
			expectErr: core.ErrInsufficientFunds,
			want:      21000,
		},
		// empty create
		{
			blockNumber: rpc.LatestBlockNumber,
			call:        TransactionArgs{},
			expectErr:   nil,
			want:        53000,
		},
	}
	for i, tc := range testSuite {
		result, err := api.EstimateGas(context.Background(), tc.call, &rpc.BlockNumberOrHash{BlockNumber: &tc.blockNumber})
		if tc.expectErr != nil {
			if err == nil {
				t.Errorf("test %d: want error %v, have nothing", i, tc.expectErr)
				continue
			}
			if !errors.Is(err, tc.expectErr) {
				t.Errorf("test %d: error mismatch, want %v, have %v", i, tc.expectErr, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("test %d: want no error, have %v", i, err)
			continue
		}
		if uint64(result) != tc.want {
			t.Errorf("test %d, result mismatch, have\n%v\n, want\n%v\n", i, uint64(result), tc.want)
		}
	}
}

func TestCall(t *testing.T) {
	t.Parallel()
	// Initialize test accounts
	var (
		accounts = newAccounts(3)
		genesis  = &core.Genesis{
			Config: params.TestChainConfig,
			Alloc: core.GenesisAlloc{
				accounts[0].addr: {Balance: big.NewInt(params.Ether)},
				accounts[1].addr: {Balance: big.NewInt(params.Ether)},
				accounts[2].addr: {Balance: big.NewInt(params.Ether)},
			},
		}
		genBlocks = 10
		signer    = types.HomesteadSigner{}
	)
	api := NewBlockChainAPI(newTestBackend(t, genBlocks, genesis, func(i int, b *core.BlockGen) {
		// Transfer from account[0] to account[1]
		//    value: 1000 wei
		//    fee:   0 wei
		tx, _ := types.SignTx(types.NewTx(&types.LegacyTx{Nonce: uint64(i), To: &accounts[1].addr, Value: big.NewInt(1000), Gas: params.TxGas, GasPrice: b.BaseFee(), Data: nil}), signer, accounts[0].key)
		b.AddTx(tx)
	}))
	randomAccounts := newAccounts(3)
	var testSuite = []struct {
		blockNumber    rpc.BlockNumber
		overrides      StateOverride
		call           TransactionArgs
		blockOverrides BlockOverrides
		expectErr      error
		want           string
	}{
		// transfer on genesis
		{
			blockNumber: rpc.BlockNumber(0),
			call: TransactionArgs{
				From:  &accounts[0].addr,
				To:    &accounts[1].addr,
				Value: (*hexutil.Big)(big.NewInt(1000)),
			},
			expectErr: nil,
			want:      "0x",
		},
		// transfer on the head
		{
			blockNumber: rpc.BlockNumber(genBlocks),
			call: TransactionArgs{
				From:  &accounts[0].addr,
				To:    &accounts[1].addr,
				Value: (*hexutil.Big)(big.NewInt(1000)),
			},
			expectErr: nil,
			want:      "0x",
		},
		// transfer on a non-existent block, error expects
		{
			blockNumber: rpc.BlockNumber(genBlocks + 1),
			call: TransactionArgs{
				From:  &accounts[0].addr,
				To:    &accounts[1].addr,
				Value: (*hexutil.Big)(big.NewInt(1000)),
			},
			expectErr: errors.New("header not found"),
		},
		// transfer on the latest block
		{
			blockNumber: rpc.LatestBlockNumber,
			call: TransactionArgs{
				From:  &accounts[0].addr,
				To:    &accounts[1].addr,
				Value: (*hexutil.Big)(big.NewInt(1000)),
			},
			expectErr: nil,
			want:      "0x",
		},
		// Call which can only succeed if state is state overridden
		{
			blockNumber: rpc.LatestBlockNumber,
			call: TransactionArgs{
				From:  &randomAccounts[0].addr,
				To:    &randomAccounts[1].addr,
				Value: (*hexutil.Big)(big.NewInt(1000)),
			},
			overrides: StateOverride{
				randomAccounts[0].addr: OverrideAccount{Balance: newRPCBalance(new(big.Int).Mul(big.NewInt(1), big.NewInt(params.Ether)))},
			},
			want: "0x",
		},
		// Invalid call without state overriding
		{
			blockNumber: rpc.LatestBlockNumber,
			call: TransactionArgs{
				From:  &randomAccounts[0].addr,
				To:    &randomAccounts[1].addr,
				Value: (*hexutil.Big)(big.NewInt(1000)),
			},
			expectErr: core.ErrInsufficientFunds,
		},
		// Successful simple contract call
		//
		// // SPDX-License-Identifier: GPL-3.0
		//
		//  pragma solidity >=0.7.0 <0.8.0;
		//
		//  /**
		//   * @title Storage
		//   * @dev Store & retrieve value in a variable
		//   */
		//  contract Storage {
		//      uint256 public number;
		//      constructor() {
		//          number = block.number;
		//      }
		//  }
		{
			blockNumber: rpc.LatestBlockNumber,
			call: TransactionArgs{
				From: &randomAccounts[0].addr,
				To:   &randomAccounts[2].addr,
				Data: hex2Bytes("8381f58a"), // call number()
			},
			overrides: StateOverride{
				randomAccounts[2].addr: OverrideAccount{
					Code:      hex2Bytes("6080604052348015600f57600080fd5b506004361060285760003560e01c80638381f58a14602d575b600080fd5b60336049565b6040518082815260200191505060405180910390f35b6000548156fea2646970667358221220eab35ffa6ab2adfe380772a48b8ba78e82a1b820a18fcb6f59aa4efb20a5f60064736f6c63430007040033"),
					StateDiff: &map[common.Hash]common.Hash{{}: common.BigToHash(big.NewInt(123))},
				},
			},
			want: "0x000000000000000000000000000000000000000000000000000000000000007b",
		},
		// Block overrides should work
		{
			blockNumber: rpc.LatestBlockNumber,
			call: TransactionArgs{
				From: &accounts[1].addr,
				Input: &hexutil.Bytes{
					0x43,             // NUMBER
					0x60, 0x00, 0x52, // MSTORE offset 0
					0x60, 0x20, 0x60, 0x00, 0xf3,
				},
			},
			blockOverrides: BlockOverrides{Number: (*hexutil.Big)(big.NewInt(11))},
			want:           "0x000000000000000000000000000000000000000000000000000000000000000b",
		},
	}
	for i, tc := range testSuite {
		result, err := api.Call(context.Background(), tc.call, rpc.BlockNumberOrHash{BlockNumber: &tc.blockNumber}, &tc.overrides, &tc.blockOverrides)
		if tc.expectErr != nil {
			if err == nil {
				t.Errorf("test %d: want error %v, have nothing", i, tc.expectErr)
				continue
			}
			if !errors.Is(err, tc.expectErr) {
				// Second try
				if !reflect.DeepEqual(err, tc.expectErr) {
					t.Errorf("test %d: error mismatch, want %v, have %v", i, tc.expectErr, err)
				}
			}
			continue
		}
		if err != nil {
			t.Errorf("test %d: want no error, have %v", i, err)
			continue
		}
		if !reflect.DeepEqual(result.String(), tc.want) {
			t.Errorf("test %d, result mismatch, have\n%v\n, want\n%v\n", i, result.String(), tc.want)
		}
	}
}

type Account struct {
	key  *ecdsa.PrivateKey
	addr common.Address
}

func newAccounts(n int) (accounts []Account) {
	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(key.PublicKey)
		accounts = append(accounts, Account{key: key, addr: addr})
	}
	slices.SortFunc(accounts, func(a, b Account) bool { return a.addr.Less(b.addr) })
	return accounts
}

func newRPCBalance(balance *big.Int) **hexutil.Big {
	rpcBalance := (*hexutil.Big)(balance)
	return &rpcBalance
}

func hex2Bytes(str string) *hexutil.Bytes {
	rpcBytes := hexutil.Bytes(common.Hex2Bytes(str))
	return &rpcBytes
}

func TestRPCMarshalBlock(t *testing.T) {
	t.Parallel()
	var (
		txs []*types.Transaction
		to  = common.BytesToAddress([]byte{0x11})
	)
	for i := uint64(1); i <= 4; i++ {
		var tx *types.Transaction
		if i%2 == 0 {
			tx = types.NewTx(&types.LegacyTx{
				Nonce:    i,
				GasPrice: big.NewInt(11111),
				Gas:      1111,
				To:       &to,
				Value:    big.NewInt(111),
				Data:     []byte{0x11, 0x11, 0x11},
			})
		} else {
			tx = types.NewTx(&types.AccessListTx{
				ChainID:  big.NewInt(1337),
				Nonce:    i,
				GasPrice: big.NewInt(11111),
				Gas:      1111,
				To:       &to,
				Value:    big.NewInt(111),
				Data:     []byte{0x11, 0x11, 0x11},
			})
		}
		txs = append(txs, tx)
	}
	block := types.NewBlock(&types.Header{Number: big.NewInt(100)}, txs, nil, nil, blocktest.NewHasher())

	var testSuite = []struct {
		inclTx bool
		fullTx bool
		want   string
	}{
		// without txs
		{
			inclTx: false,
			fullTx: false,
			want:   `{"difficulty":"0x0","extraData":"0x","gasLimit":"0x0","gasUsed":"0x0","hash":"0x9b73c83b25d0faf7eab854e3684c7e394336d6e135625aafa5c183f27baa8fee","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":"0x0000000000000000000000000000000000000000","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","number":"0x64","parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","size":"0x296","stateRoot":"0x0000000000000000000000000000000000000000000000000000000000000000","timestamp":"0x0","transactionsRoot":"0x661a9febcfa8f1890af549b874faf9fa274aede26ef489d9db0b25daa569450e","uncles":[]}`,
		},
		// only tx hashes
		{
			inclTx: true,
			fullTx: false,
			want:   `{"difficulty":"0x0","extraData":"0x","gasLimit":"0x0","gasUsed":"0x0","hash":"0x9b73c83b25d0faf7eab854e3684c7e394336d6e135625aafa5c183f27baa8fee","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":"0x0000000000000000000000000000000000000000","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","number":"0x64","parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","size":"0x296","stateRoot":"0x0000000000000000000000000000000000000000000000000000000000000000","timestamp":"0x0","transactions":["0x7d39df979e34172322c64983a9ad48302c2b889e55bda35324afecf043a77605","0x9bba4c34e57c875ff57ac8d172805a26ae912006985395dc1bdf8f44140a7bf4","0x98909ea1ff040da6be56bc4231d484de1414b3c1dac372d69293a4beb9032cb5","0x12e1f81207b40c3bdcc13c0ee18f5f86af6d31754d57a0ea1b0d4cfef21abef1"],"transactionsRoot":"0x661a9febcfa8f1890af549b874faf9fa274aede26ef489d9db0b25daa569450e","uncles":[]}`,
		},
		// full tx details
		{
			inclTx: true,
			fullTx: true,
			want:   `{"difficulty":"0x0","extraData":"0x","gasLimit":"0x0","gasUsed":"0x0","hash":"0x9b73c83b25d0faf7eab854e3684c7e394336d6e135625aafa5c183f27baa8fee","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":"0x0000000000000000000000000000000000000000","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","number":"0x64","parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","size":"0x296","stateRoot":"0x0000000000000000000000000000000000000000000000000000000000000000","timestamp":"0x0","transactions":[{"blockHash":"0x9b73c83b25d0faf7eab854e3684c7e394336d6e135625aafa5c183f27baa8fee","blockNumber":"0x64","from":"0x0000000000000000000000000000000000000000","gas":"0x457","gasPrice":"0x2b67","hash":"0x7d39df979e34172322c64983a9ad48302c2b889e55bda35324afecf043a77605","input":"0x111111","nonce":"0x1","to":"0x0000000000000000000000000000000000000011","transactionIndex":"0x0","value":"0x6f","type":"0x1","accessList":[],"chainId":"0x539","v":"0x0","r":"0x0","s":"0x0"},{"blockHash":"0x9b73c83b25d0faf7eab854e3684c7e394336d6e135625aafa5c183f27baa8fee","blockNumber":"0x64","from":"0x0000000000000000000000000000000000000000","gas":"0x457","gasPrice":"0x2b67","hash":"0x9bba4c34e57c875ff57ac8d172805a26ae912006985395dc1bdf8f44140a7bf4","input":"0x111111","nonce":"0x2","to":"0x0000000000000000000000000000000000000011","transactionIndex":"0x1","value":"0x6f","type":"0x0","chainId":"0x7fffffffffffffee","v":"0x0","r":"0x0","s":"0x0"},{"blockHash":"0x9b73c83b25d0faf7eab854e3684c7e394336d6e135625aafa5c183f27baa8fee","blockNumber":"0x64","from":"0x0000000000000000000000000000000000000000","gas":"0x457","gasPrice":"0x2b67","hash":"0x98909ea1ff040da6be56bc4231d484de1414b3c1dac372d69293a4beb9032cb5","input":"0x111111","nonce":"0x3","to":"0x0000000000000000000000000000000000000011","transactionIndex":"0x2","value":"0x6f","type":"0x1","accessList":[],"chainId":"0x539","v":"0x0","r":"0x0","s":"0x0"},{"blockHash":"0x9b73c83b25d0faf7eab854e3684c7e394336d6e135625aafa5c183f27baa8fee","blockNumber":"0x64","from":"0x0000000000000000000000000000000000000000","gas":"0x457","gasPrice":"0x2b67","hash":"0x12e1f81207b40c3bdcc13c0ee18f5f86af6d31754d57a0ea1b0d4cfef21abef1","input":"0x111111","nonce":"0x4","to":"0x0000000000000000000000000000000000000011","transactionIndex":"0x3","value":"0x6f","type":"0x0","chainId":"0x7fffffffffffffee","v":"0x0","r":"0x0","s":"0x0"}],"transactionsRoot":"0x661a9febcfa8f1890af549b874faf9fa274aede26ef489d9db0b25daa569450e","uncles":[]}`,
		},
	}

	for i, tc := range testSuite {
		resp := RPCMarshalBlock(block, tc.inclTx, tc.fullTx, params.MainnetChainConfig)
		out, err := json.Marshal(resp)
		if err != nil {
			t.Errorf("test %d: json marshal error: %v", i, err)
			continue
		}
		if have := string(out); have != tc.want {
			t.Errorf("test %d: want: %s have: %s", i, tc.want, have)
		}
	}
}

func TestRPCGetBlockOrHeader(t *testing.T) {
	t.Parallel()

	// Initialize test accounts
	var (
		acc1Key, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
		acc2Key, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
		acc1Addr   = crypto.PubkeyToAddress(acc1Key.PublicKey)
		acc2Addr   = crypto.PubkeyToAddress(acc2Key.PublicKey)
		genesis    = &core.Genesis{
			Config: params.TestChainConfig,
			Alloc: core.GenesisAlloc{
				acc1Addr: {Balance: big.NewInt(params.Ether)},
				acc2Addr: {Balance: big.NewInt(params.Ether)},
			},
		}
		genBlocks = 10
		signer    = types.HomesteadSigner{}
		tx        = types.NewTx(&types.LegacyTx{
			Nonce:    11,
			GasPrice: big.NewInt(11111),
			Gas:      1111,
			To:       &acc2Addr,
			Value:    big.NewInt(111),
			Data:     []byte{0x11, 0x11, 0x11},
		})
		withdrawal = &types.Withdrawal{
			Index:     0,
			Validator: 1,
			Address:   common.Address{0x12, 0x34},
			Amount:    10,
		}
		pending = types.NewBlockWithWithdrawals(&types.Header{Number: big.NewInt(11), Time: 42}, []*types.Transaction{tx}, nil, nil, []*types.Withdrawal{withdrawal}, blocktest.NewHasher())
	)
	backend := newTestBackend(t, genBlocks, genesis, func(i int, b *core.BlockGen) {
		// Transfer from account[0] to account[1]
		//    value: 1000 wei
		//    fee:   0 wei
		tx, _ := types.SignTx(types.NewTx(&types.LegacyTx{Nonce: uint64(i), To: &acc2Addr, Value: big.NewInt(1000), Gas: params.TxGas, GasPrice: b.BaseFee(), Data: nil}), signer, acc1Key)
		b.AddTx(tx)
	})
	backend.setPendingBlock(pending)
	api := NewBlockChainAPI(backend)
	blockHashes := make([]common.Hash, genBlocks+1)
	ctx := context.Background()
	for i := 0; i <= genBlocks; i++ {
		header, err := backend.HeaderByNumber(ctx, rpc.BlockNumber(i))
		if err != nil {
			t.Errorf("failed to get block: %d err: %v", i, err)
		}
		blockHashes[i] = header.Hash()
	}
	pendingHash := pending.Hash()

	var testSuite = []struct {
		blockNumber rpc.BlockNumber
		blockHash   *common.Hash
		fullTx      bool
		reqHeader   bool
		want        string
		expectErr   error
	}{
		// 0. latest header
		{
			blockNumber: rpc.LatestBlockNumber,
			reqHeader:   true,
			want:        `{"baseFeePerGas":"0xfdc7303", "difficulty":"0x20000", "extraData":"0x", "gasLimit":"0x47e7c4", "gasUsed":"0x5208", "hash":"0x97f540a3577c0f645c5dada5da86f38350e8f847e71f21124f917835003e2607", "logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", "miner":"0x0000000000000000000000000000000000000000", "mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000", "nonce":"0x0000000000000000", "number":"0xa", "parentHash":"0xda97ed946e0d502fb898b0ac881bd44da3c7fee5eaf184431e1ec3d361dad17e", "receiptsRoot":"0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2", "sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347", "stateRoot":"0xbb62872e4023fa8a8b17b9cc37031f4817d9595779748d01cba408b495707a91", "timestamp":"0x64", "totalDifficulty":"0x1", "transactionsRoot":"0xb0893d21a4a44dc26a962a6e91abae66df87fb61ac9c60e936aee89c76331445"}`,
		},
		// 1. genesis header
		{
			blockNumber: rpc.BlockNumber(0),
			reqHeader:   true,
			want:        `{"baseFeePerGas":"0x3b9aca00", "difficulty":"0x20000", "extraData":"0x", "gasLimit":"0x47e7c4", "gasUsed":"0x0", "hash":"0xbdc7d83b8f876938810462fe8d053263a482e44201e3883d4ae204ff4de7eff5", "logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", "miner":"0x0000000000000000000000000000000000000000", "mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000", "nonce":"0x0000000000000000", "number":"0x0", "parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000", "receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421", "sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347", "stateRoot":"0xfe168c5e9584a85927212e5bea5304bb7d0d8a893453b4b2c52176a72f585ae2", "timestamp":"0x0", "totalDifficulty":"0x1", "transactionsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"}`,
		},
		// 2. #1 header
		{
			blockNumber: rpc.BlockNumber(1),
			reqHeader:   true,
			want:        `{"baseFeePerGas":"0x342770c0", "difficulty":"0x20000", "extraData":"0x", "gasLimit":"0x47e7c4", "gasUsed":"0x5208", "hash":"0x0da274b315de8e4d5bf8717218ec43540464ef36378cb896469bb731e1d3f3cb", "logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", "miner":"0x0000000000000000000000000000000000000000", "mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000", "nonce":"0x0000000000000000", "number":"0x1", "parentHash":"0xbdc7d83b8f876938810462fe8d053263a482e44201e3883d4ae204ff4de7eff5", "receiptsRoot":"0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2", "sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347", "stateRoot":"0x92c5c55a698963f5b06e3aee415630f5c48b0760e537af94917ce9c4f42a2e22", "timestamp":"0xa", "totalDifficulty":"0x1", "transactionsRoot":"0xca0ebcce920d2cdfbf9e1dbe90ed3441a1a576f344bd80e60508da814916f4e7"}`,
		},
		// 3. latest-1 header
		{
			blockNumber: rpc.BlockNumber(9),
			reqHeader:   true,
			want:        `{"baseFeePerGas":"0x121a9cca", "difficulty":"0x20000", "extraData":"0x", "gasLimit":"0x47e7c4", "gasUsed":"0x5208", "hash":"0xda97ed946e0d502fb898b0ac881bd44da3c7fee5eaf184431e1ec3d361dad17e", "logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", "miner":"0x0000000000000000000000000000000000000000", "mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000", "nonce":"0x0000000000000000", "number":"0x9", "parentHash":"0x5abd19c39d9f1c6e52998e135ea14e1fbc5db3fa2a108f4538e238ca5c2e68d7", "receiptsRoot":"0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2", "sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347", "stateRoot":"0xbd4aa2c2873df709151075250a8c01c9a14d2b0e2f715dbdd16e0ef8030c2cf0", "timestamp":"0x5a", "totalDifficulty":"0x1", "transactionsRoot":"0x0767ed8359337dc6a8fdc77fe52db611bed1be87aac73c4556b1bf1dd3d190a5"}`,
		},
		// 4. latest+1 header
		{
			blockNumber: rpc.BlockNumber(11),
			reqHeader:   true,
			want:        "null",
		},
		// 5. pending header
		{
			blockNumber: rpc.PendingBlockNumber,
			reqHeader:   true,
			want:        `{"difficulty":"0x0","extraData":"0x","gasLimit":"0x0","gasUsed":"0x0","hash":null,"logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":null,"mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":null,"number":"0xb","parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","stateRoot":"0x0000000000000000000000000000000000000000000000000000000000000000","timestamp":"0x2a","totalDifficulty":null,"transactionsRoot":"0x98d9f6dd0aa479c0fb448f2627e9f1964aca699fccab8f6e95861547a4699e37","withdrawalsRoot":"0x73d756269cdfc22e7e17a3548e36f42f750ca06d7e3cd98d1b6d0eb5add9dc84"}`,
		},
		// 6. latest block
		{
			blockNumber: rpc.LatestBlockNumber,
			want:        `{"baseFeePerGas":"0xfdc7303","difficulty":"0x20000","extraData":"0x","gasLimit":"0x47e7c4","gasUsed":"0x5208","hash":"0x97f540a3577c0f645c5dada5da86f38350e8f847e71f21124f917835003e2607","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":"0x0000000000000000000000000000000000000000","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","number":"0xa","parentHash":"0xda97ed946e0d502fb898b0ac881bd44da3c7fee5eaf184431e1ec3d361dad17e","receiptsRoot":"0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","size":"0x26a","stateRoot":"0xbb62872e4023fa8a8b17b9cc37031f4817d9595779748d01cba408b495707a91","timestamp":"0x64","totalDifficulty":"0x1","transactions":["0x3ee4094ca1e0b07a66dd616a057e081e53144ca7e9685a126fd4dda9ca042644"],"transactionsRoot":"0xb0893d21a4a44dc26a962a6e91abae66df87fb61ac9c60e936aee89c76331445","uncles":[]}`,
		},
		// 7. genesis block
		{
			blockNumber: rpc.BlockNumber(0),
			want:        `{"baseFeePerGas":"0x3b9aca00","difficulty":"0x20000","extraData":"0x","gasLimit":"0x47e7c4","gasUsed":"0x0","hash":"0xbdc7d83b8f876938810462fe8d053263a482e44201e3883d4ae204ff4de7eff5","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":"0x0000000000000000000000000000000000000000","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","number":"0x0","parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","size":"0x200","stateRoot":"0xfe168c5e9584a85927212e5bea5304bb7d0d8a893453b4b2c52176a72f585ae2","timestamp":"0x0","totalDifficulty":"0x1","transactions":[],"transactionsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","uncles":[]}`,
		},
		// 8. #1 block
		{
			blockNumber: rpc.BlockNumber(1),
			want:        `{"baseFeePerGas":"0x342770c0","difficulty":"0x20000","extraData":"0x","gasLimit":"0x47e7c4","gasUsed":"0x5208","hash":"0x0da274b315de8e4d5bf8717218ec43540464ef36378cb896469bb731e1d3f3cb","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":"0x0000000000000000000000000000000000000000","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","number":"0x1","parentHash":"0xbdc7d83b8f876938810462fe8d053263a482e44201e3883d4ae204ff4de7eff5","receiptsRoot":"0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","size":"0x26a","stateRoot":"0x92c5c55a698963f5b06e3aee415630f5c48b0760e537af94917ce9c4f42a2e22","timestamp":"0xa","totalDifficulty":"0x1","transactions":["0x644a31c354391520d00e95b9affbbb010fc79ac268144ab8e28207f4cf51097e"],"transactionsRoot":"0xca0ebcce920d2cdfbf9e1dbe90ed3441a1a576f344bd80e60508da814916f4e7","uncles":[]}`,
		},
		// 9. latest-1 block
		{
			blockNumber: rpc.BlockNumber(9),
			fullTx:      true,
			want:        `{"baseFeePerGas":"0x121a9cca","difficulty":"0x20000","extraData":"0x","gasLimit":"0x47e7c4","gasUsed":"0x5208","hash":"0xda97ed946e0d502fb898b0ac881bd44da3c7fee5eaf184431e1ec3d361dad17e","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":"0x0000000000000000000000000000000000000000","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","number":"0x9","parentHash":"0x5abd19c39d9f1c6e52998e135ea14e1fbc5db3fa2a108f4538e238ca5c2e68d7","receiptsRoot":"0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","size":"0x26a","stateRoot":"0xbd4aa2c2873df709151075250a8c01c9a14d2b0e2f715dbdd16e0ef8030c2cf0","timestamp":"0x5a","totalDifficulty":"0x1","transactions":[{"blockHash":"0xda97ed946e0d502fb898b0ac881bd44da3c7fee5eaf184431e1ec3d361dad17e","blockNumber":"0x9","from":"0x703c4b2bd70c169f5717101caee543299fc946c7","gas":"0x5208","gasPrice":"0x121a9cca","hash":"0xecd155a61a5734b3efab75924e3ae34026c7c4133d8c2a46122bd03d7d199725","input":"0x","nonce":"0x8","to":"0x0d3ab14bbad3d99f4203bd7a11acb94882050e7e","transactionIndex":"0x0","value":"0x3e8","type":"0x0","v":"0x1b","r":"0xc6028b8e983d62fa8542f8a7633fb23cc941be2c897134352d95a7d9b19feafd","s":"0xeb6adcaaae3bed489c6cce4435f9db05d23a52820c78bd350e31eec65ed809d"}],"transactionsRoot":"0x0767ed8359337dc6a8fdc77fe52db611bed1be87aac73c4556b1bf1dd3d190a5","uncles":[]}`,
		},
		// 10. latest+1 block
		{
			blockNumber: rpc.BlockNumber(11),
			fullTx:      true,
			want:        "null",
		},
		// 11. pending block
		{
			blockNumber: rpc.PendingBlockNumber,
			want:        `{"difficulty":"0x0","extraData":"0x","gasLimit":"0x0","gasUsed":"0x0","hash":null,"logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":null,"mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":null,"number":"0xb","parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","size":"0x256","stateRoot":"0x0000000000000000000000000000000000000000000000000000000000000000","timestamp":"0x2a","totalDifficulty":null,"transactions":["0x4afee081df5dff7a025964032871f7d4ba4d21baf5f6376a2f4a9f79fc506298"],"transactionsRoot":"0x98d9f6dd0aa479c0fb448f2627e9f1964aca699fccab8f6e95861547a4699e37","withdrawals":[{"index":"0x0","validatorIndex":"0x1","address":"0x1234000000000000000000000000000000000000","amount":"0xa"}],"withdrawalsRoot":"0x73d756269cdfc22e7e17a3548e36f42f750ca06d7e3cd98d1b6d0eb5add9dc84","uncles":[]}`,
		},
		// 12. pending block + fullTx
		{
			blockNumber: rpc.PendingBlockNumber,
			fullTx:      true,
			want:        `{"difficulty":"0x0","extraData":"0x","gasLimit":"0x0","gasUsed":"0x0","hash":null,"logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":null,"mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":null,"number":"0xb","parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","size":"0x256","stateRoot":"0x0000000000000000000000000000000000000000000000000000000000000000","timestamp":"0x2a","totalDifficulty":null,"transactions":[{"blockHash":"0x6cebd9f966ea686f44b981685e3f0eacea28591a7a86d7fbbe521a86e9f81165","blockNumber":"0xb","from":"0x0000000000000000000000000000000000000000","gas":"0x457","gasPrice":"0x2b67","hash":"0x4afee081df5dff7a025964032871f7d4ba4d21baf5f6376a2f4a9f79fc506298","input":"0x111111","nonce":"0xb","to":"0x0d3ab14bbad3d99f4203bd7a11acb94882050e7e","transactionIndex":"0x0","value":"0x6f","type":"0x0","chainId":"0x7fffffffffffffee","v":"0x0","r":"0x0","s":"0x0"}],"transactionsRoot":"0x98d9f6dd0aa479c0fb448f2627e9f1964aca699fccab8f6e95861547a4699e37","uncles":[],"withdrawals":[{"index":"0x0","validatorIndex":"0x1","address":"0x1234000000000000000000000000000000000000","amount":"0xa"}],"withdrawalsRoot":"0x73d756269cdfc22e7e17a3548e36f42f750ca06d7e3cd98d1b6d0eb5add9dc84"}`,
		},
		// 13. latest header by hash
		{
			blockHash: &blockHashes[len(blockHashes)-1],
			reqHeader: true,
			want:      `{"baseFeePerGas":"0xfdc7303", "difficulty":"0x20000", "extraData":"0x", "gasLimit":"0x47e7c4", "gasUsed":"0x5208", "hash":"0x97f540a3577c0f645c5dada5da86f38350e8f847e71f21124f917835003e2607", "logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", "miner":"0x0000000000000000000000000000000000000000", "mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000", "nonce":"0x0000000000000000", "number":"0xa", "parentHash":"0xda97ed946e0d502fb898b0ac881bd44da3c7fee5eaf184431e1ec3d361dad17e", "receiptsRoot":"0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2", "sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347", "stateRoot":"0xbb62872e4023fa8a8b17b9cc37031f4817d9595779748d01cba408b495707a91", "timestamp":"0x64", "totalDifficulty":"0x1", "transactionsRoot":"0xb0893d21a4a44dc26a962a6e91abae66df87fb61ac9c60e936aee89c76331445"}`,
		},
		// 14. genesis header by hash
		{
			blockHash: &blockHashes[0],
			reqHeader: true,
			want:      `{"baseFeePerGas":"0x3b9aca00", "difficulty":"0x20000", "extraData":"0x", "gasLimit":"0x47e7c4", "gasUsed":"0x0", "hash":"0xbdc7d83b8f876938810462fe8d053263a482e44201e3883d4ae204ff4de7eff5", "logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", "miner":"0x0000000000000000000000000000000000000000", "mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000", "nonce":"0x0000000000000000", "number":"0x0", "parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000", "receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421", "sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347", "stateRoot":"0xfe168c5e9584a85927212e5bea5304bb7d0d8a893453b4b2c52176a72f585ae2", "timestamp":"0x0", "totalDifficulty":"0x1", "transactionsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"}`,
		},
		// 15. #1 header
		{
			blockHash: &blockHashes[1],
			reqHeader: true,
			want:      `{"baseFeePerGas":"0x342770c0", "difficulty":"0x20000", "extraData":"0x", "gasLimit":"0x47e7c4", "gasUsed":"0x5208", "hash":"0x0da274b315de8e4d5bf8717218ec43540464ef36378cb896469bb731e1d3f3cb", "logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", "miner":"0x0000000000000000000000000000000000000000", "mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000", "nonce":"0x0000000000000000", "number":"0x1", "parentHash":"0xbdc7d83b8f876938810462fe8d053263a482e44201e3883d4ae204ff4de7eff5", "receiptsRoot":"0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2", "sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347", "stateRoot":"0x92c5c55a698963f5b06e3aee415630f5c48b0760e537af94917ce9c4f42a2e22", "timestamp":"0xa", "totalDifficulty":"0x1", "transactionsRoot":"0xca0ebcce920d2cdfbf9e1dbe90ed3441a1a576f344bd80e60508da814916f4e7"}`,
		},
		// 16. latest-1 header
		{
			blockHash: &blockHashes[len(blockHashes)-2],
			reqHeader: true,
			want:      `{"baseFeePerGas":"0x121a9cca", "difficulty":"0x20000", "extraData":"0x", "gasLimit":"0x47e7c4", "gasUsed":"0x5208", "hash":"0xda97ed946e0d502fb898b0ac881bd44da3c7fee5eaf184431e1ec3d361dad17e", "logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", "miner":"0x0000000000000000000000000000000000000000", "mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000", "nonce":"0x0000000000000000", "number":"0x9", "parentHash":"0x5abd19c39d9f1c6e52998e135ea14e1fbc5db3fa2a108f4538e238ca5c2e68d7", "receiptsRoot":"0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2", "sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347", "stateRoot":"0xbd4aa2c2873df709151075250a8c01c9a14d2b0e2f715dbdd16e0ef8030c2cf0", "timestamp":"0x5a", "totalDifficulty":"0x1", "transactionsRoot":"0x0767ed8359337dc6a8fdc77fe52db611bed1be87aac73c4556b1bf1dd3d190a5"}`,
		},
		// 17. empty hash
		{
			blockHash: &common.Hash{},
			reqHeader: true,
			want:      "null",
		},
		// 18. pending hash
		{
			blockHash: &pendingHash,
			reqHeader: true,
			want:      `null`,
		},
		// 19. latest block
		{
			blockHash: &blockHashes[len(blockHashes)-1],
			want:      `{"baseFeePerGas":"0xfdc7303","difficulty":"0x20000","extraData":"0x","gasLimit":"0x47e7c4","gasUsed":"0x5208","hash":"0x97f540a3577c0f645c5dada5da86f38350e8f847e71f21124f917835003e2607","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":"0x0000000000000000000000000000000000000000","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","number":"0xa","parentHash":"0xda97ed946e0d502fb898b0ac881bd44da3c7fee5eaf184431e1ec3d361dad17e","receiptsRoot":"0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","size":"0x26a","stateRoot":"0xbb62872e4023fa8a8b17b9cc37031f4817d9595779748d01cba408b495707a91","timestamp":"0x64","totalDifficulty":"0x1","transactions":["0x3ee4094ca1e0b07a66dd616a057e081e53144ca7e9685a126fd4dda9ca042644"],"transactionsRoot":"0xb0893d21a4a44dc26a962a6e91abae66df87fb61ac9c60e936aee89c76331445","uncles":[]}`,
		},
		// 20. genesis block
		{
			blockHash: &blockHashes[0],
			want:      `{"baseFeePerGas":"0x3b9aca00","difficulty":"0x20000","extraData":"0x","gasLimit":"0x47e7c4","gasUsed":"0x0","hash":"0xbdc7d83b8f876938810462fe8d053263a482e44201e3883d4ae204ff4de7eff5","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":"0x0000000000000000000000000000000000000000","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","number":"0x0","parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","size":"0x200","stateRoot":"0xfe168c5e9584a85927212e5bea5304bb7d0d8a893453b4b2c52176a72f585ae2","timestamp":"0x0","totalDifficulty":"0x1","transactions":[],"transactionsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","uncles":[]}`,
		},
		// 21. #1 block
		{
			blockHash: &blockHashes[1],
			want:      `{"baseFeePerGas":"0x342770c0","difficulty":"0x20000","extraData":"0x","gasLimit":"0x47e7c4","gasUsed":"0x5208","hash":"0x0da274b315de8e4d5bf8717218ec43540464ef36378cb896469bb731e1d3f3cb","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":"0x0000000000000000000000000000000000000000","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","number":"0x1","parentHash":"0xbdc7d83b8f876938810462fe8d053263a482e44201e3883d4ae204ff4de7eff5","receiptsRoot":"0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","size":"0x26a","stateRoot":"0x92c5c55a698963f5b06e3aee415630f5c48b0760e537af94917ce9c4f42a2e22","timestamp":"0xa","totalDifficulty":"0x1","transactions":["0x644a31c354391520d00e95b9affbbb010fc79ac268144ab8e28207f4cf51097e"],"transactionsRoot":"0xca0ebcce920d2cdfbf9e1dbe90ed3441a1a576f344bd80e60508da814916f4e7","uncles":[]}`,
		},
		// 22. latest-1 block
		{
			blockHash: &blockHashes[len(blockHashes)-2],
			fullTx:    true,
			want:      `{"baseFeePerGas":"0x121a9cca","difficulty":"0x20000","extraData":"0x","gasLimit":"0x47e7c4","gasUsed":"0x5208","hash":"0xda97ed946e0d502fb898b0ac881bd44da3c7fee5eaf184431e1ec3d361dad17e","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":"0x0000000000000000000000000000000000000000","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","number":"0x9","parentHash":"0x5abd19c39d9f1c6e52998e135ea14e1fbc5db3fa2a108f4538e238ca5c2e68d7","receiptsRoot":"0x056b23fbba480696b65fe5a59b8f2148a1299103c4f57df839233af2cf4ca2d2","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","size":"0x26a","stateRoot":"0xbd4aa2c2873df709151075250a8c01c9a14d2b0e2f715dbdd16e0ef8030c2cf0","timestamp":"0x5a","totalDifficulty":"0x1","transactions":[{"blockHash":"0xda97ed946e0d502fb898b0ac881bd44da3c7fee5eaf184431e1ec3d361dad17e","blockNumber":"0x9","from":"0x703c4b2bd70c169f5717101caee543299fc946c7","gas":"0x5208","gasPrice":"0x121a9cca","hash":"0xecd155a61a5734b3efab75924e3ae34026c7c4133d8c2a46122bd03d7d199725","input":"0x","nonce":"0x8","to":"0x0d3ab14bbad3d99f4203bd7a11acb94882050e7e","transactionIndex":"0x0","value":"0x3e8","type":"0x0","v":"0x1b","r":"0xc6028b8e983d62fa8542f8a7633fb23cc941be2c897134352d95a7d9b19feafd","s":"0xeb6adcaaae3bed489c6cce4435f9db05d23a52820c78bd350e31eec65ed809d"}],"transactionsRoot":"0x0767ed8359337dc6a8fdc77fe52db611bed1be87aac73c4556b1bf1dd3d190a5","uncles":[]}`,
		},
		// 23. empty hash + body
		{
			blockHash: &common.Hash{},
			fullTx:    true,
			want:      "null",
		},
		// 24. pending block
		{
			blockHash: &pendingHash,
			want:      `null`,
		},
		// 25. pending block + fullTx
		{
			blockHash: &pendingHash,
			fullTx:    true,
			want:      `null`,
		},
	}

	for i, tt := range testSuite {
		var (
			result map[string]interface{}
			err    error
		)
		if tt.blockHash != nil {
			if tt.reqHeader {
				result = api.GetHeaderByHash(context.Background(), *tt.blockHash)
			} else {
				result, err = api.GetBlockByHash(context.Background(), *tt.blockHash, tt.fullTx)
			}
		} else {
			if tt.reqHeader {
				result, err = api.GetHeaderByNumber(context.Background(), tt.blockNumber)
			} else {
				result, err = api.GetBlockByNumber(context.Background(), tt.blockNumber, tt.fullTx)
			}
		}
		if tt.expectErr != nil {
			if err == nil {
				t.Errorf("test %d: want error %v, have nothing", i, tt.expectErr)
				continue
			}
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("test %d: error mismatch, want %v, have %v", i, tt.expectErr, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("test %d: want no error, have %v", i, err)
			continue
		}
		data, err := json.Marshal(result)
		if err != nil {
			t.Errorf("test %d: json marshal error", i)
			continue
		}
		want, have := tt.want, string(data)
		require.JSONEqf(t, want, have, "test %d: json not match, want: %s, have: %s", i, want, have)
	}
}
