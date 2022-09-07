// Copyright 2022 The go-ethereum Authors
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
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"math/big"
	"reflect"
	"sort"
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
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

type testBackend struct {
	db    ethdb.Database
	chain *core.BlockChain
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

func (b testBackend) SyncProgress() ethereum.SyncProgress { return ethereum.SyncProgress{} }
func (b testBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (b testBackend) FeeHistory(ctx context.Context, blockCount int, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*big.Int, [][]*big.Int, []*big.Int, []float64, error) {
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
		return b.chain.CurrentBlock().Header(), nil
	}
	return b.chain.GetHeaderByNumber(uint64(number)), nil
}
func (b testBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	panic("implement me")
}
func (b testBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	panic("implement me")
}
func (b testBackend) CurrentHeader() *types.Header { panic("implement me") }
func (b testBackend) CurrentBlock() *types.Block   { panic("implement me") }
func (b testBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	if number == rpc.LatestBlockNumber {
		return b.chain.CurrentBlock(), nil
	}
	return b.chain.GetBlockByNumber(uint64(number)), nil
}
func (b testBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	panic("implement me")
}
func (b testBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.BlockByNumber(ctx, blockNr)
	}
	panic("implement me")
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
func (b testBackend) GetTd(ctx context.Context, hash common.Hash) *big.Int { panic("implement me") }
func (b testBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmConfig *vm.Config, blockContext *vm.BlockContext) (*vm.EVM, func() error, error) {
	vmError := func() error { return nil }
	if vmConfig == nil {
		vmConfig = b.chain.GetVMConfig()
	}
	txContext := core.NewEVMTxContext(msg)
	context := core.NewEVMBlockContext(header, b.chain, nil)
	if blockContext != nil {
		context = *blockContext
	}
	return vm.NewEVM(context, txContext, state, b.chain.Config(), *vmConfig), vmError, nil
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
func (b testBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	panic("implement me")
}
func (b testBackend) TxPoolContentFrom(addr common.Address) (types.Transactions, types.Transactions) {
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
		blockNumber rpc.BlockNumber
		overrides   StateOverride
		call        TransactionArgs
		expectErr   error
		want        string
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
					StateDiff: &map[common.Hash]common.Hash{common.Hash{}: common.BigToHash(big.NewInt(123))},
				},
			},
			want: "0x000000000000000000000000000000000000000000000000000000000000007b",
		},
		// TODO add testcase exhausting gas cap
	}
	for i, tc := range testSuite {
		result, err := api.Call(context.Background(), tc.call, rpc.BlockNumberOrHash{BlockNumber: &tc.blockNumber}, &tc.overrides)
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

func TestBatchCall(t *testing.T) {
	t.Parallel()
	// Initialize test accounts
	var (
		accounts  = newAccounts(3)
		genBlocks = 10
		signer    = types.HomesteadSigner{}
		genesis   = &core.Genesis{
			Config: params.TestChainConfig,
			Alloc: core.GenesisAlloc{
				accounts[0].addr: {Balance: big.NewInt(params.Ether)},
				accounts[1].addr: {Balance: big.NewInt(params.Ether)},
				accounts[2].addr: {Balance: big.NewInt(params.Ether)},
			},
		}
	)
	api := NewBlockChainAPI(newTestBackend(t, genBlocks, genesis, func(i int, b *core.BlockGen) {
		// Transfer from account[0] to account[1]
		//    value: 1000 wei
		//    fee:   0 wei
		tx, _ := types.SignTx(types.NewTx(&types.LegacyTx{
			Nonce:    uint64(i),
			To:       &accounts[1].addr,
			Value:    big.NewInt(1000),
			Gas:      params.TxGas,
			GasPrice: b.BaseFee(),
			Data:     nil,
		}), signer, accounts[0].key)
		b.AddTx(tx)
	}))
	var (
		randomAccounts = newAccounts(3)
		latestBlock    = rpc.LatestBlockNumber
	)
	type res struct {
		Return string
		Error  string
	}
	var testSuite = []struct {
		config    BatchCallConfig
		expectErr error
		want      []res
	}{
		// First value transfer OK after state override, second one should succeed
		// because of first transfer.
		{
			config: BatchCallConfig{
				Block: rpc.BlockNumberOrHash{BlockNumber: &latestBlock},
				StateOverrides: &StateOverride{
					randomAccounts[0].addr: OverrideAccount{Balance: newRPCBalance(big.NewInt(1000))},
				},
				Calls: []BatchCallArgs{{
					TransactionArgs: TransactionArgs{
						From:  &randomAccounts[0].addr,
						To:    &randomAccounts[1].addr,
						Value: (*hexutil.Big)(big.NewInt(1000)),
					},
				}, {
					TransactionArgs: TransactionArgs{
						From:  &randomAccounts[1].addr,
						To:    &randomAccounts[2].addr,
						Value: (*hexutil.Big)(big.NewInt(1000)),
					},
				}},
			},
			want: []res{{
				Return: "0x",
			}, {
				Return: "0x",
			}},
		},
		// Block overrides should work, each call is simulated on a different block number
		{
			config: BatchCallConfig{
				Block: rpc.BlockNumberOrHash{BlockNumber: &latestBlock},
				Calls: []BatchCallArgs{{
					TransactionArgs: TransactionArgs{
						From: &accounts[0].addr,
						Input: &hexutil.Bytes{
							0x43,             // NUMBER
							0x60, 0x00, 0x52, // MSTORE offset 0
							0x60, 0x20, 0x60, 0x00, 0xf3,
						},
					},
					BlockOverrides: &BlockOverrides{
						Number: (*hexutil.Big)(big.NewInt(10)),
					},
				}, {
					TransactionArgs: TransactionArgs{
						From: &accounts[1].addr,
						Input: &hexutil.Bytes{
							0x43,             // NUMBER
							0x60, 0x00, 0x52, // MSTORE offset 0
							0x60, 0x20, 0x60, 0x00, 0xf3,
						},
					},
					BlockOverrides: &BlockOverrides{
						Number: (*hexutil.Big)(big.NewInt(11)),
					},
				}},
			},
			want: []res{{
				Return: "0x000000000000000000000000000000000000000000000000000000000000000a",
			}, {
				Return: "0x000000000000000000000000000000000000000000000000000000000000000b",
			}},
		},
		// Test on solidity storage example. Set value in one call, read in next.
		{
			config: BatchCallConfig{
				Block: rpc.BlockNumberOrHash{BlockNumber: &latestBlock},
				StateOverrides: &StateOverride{
					randomAccounts[2].addr: OverrideAccount{
						Code: hex2Bytes("608060405234801561001057600080fd5b50600436106100365760003560e01c80632e64cec11461003b5780636057361d14610059575b600080fd5b610043610075565b60405161005091906100d9565b60405180910390f35b610073600480360381019061006e919061009d565b61007e565b005b60008054905090565b8060008190555050565b60008135905061009781610103565b92915050565b6000602082840312156100b3576100b26100fe565b5b60006100c184828501610088565b91505092915050565b6100d3816100f4565b82525050565b60006020820190506100ee60008301846100ca565b92915050565b6000819050919050565b600080fd5b61010c816100f4565b811461011757600080fd5b5056fea2646970667358221220404e37f487a89a932dca5e77faaf6ca2de3b991f93d230604b1b8daaef64766264736f6c63430008070033"),
					},
				},
				Calls: []BatchCallArgs{{
					// Set value to 5
					TransactionArgs: TransactionArgs{
						From:  &randomAccounts[0].addr,
						To:    &randomAccounts[2].addr,
						Input: hex2Bytes("6057361d0000000000000000000000000000000000000000000000000000000000000005"),
					},
				}, {
					// Read value
					TransactionArgs: TransactionArgs{
						From:  &randomAccounts[0].addr,
						To:    &randomAccounts[2].addr,
						Input: hex2Bytes("2e64cec1"),
					},
				}},
			},
			want: []res{{
				Return: "0x",
			}, {
				Return: "0x0000000000000000000000000000000000000000000000000000000000000005",
			}},
		},
	}

	for i, tc := range testSuite {
		result, err := api.BatchCall(context.Background(), tc.config)
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
		// Turn result into res-struct
		var have []res
		resBytes, _ := json.Marshal(result)
		if err := json.Unmarshal(resBytes, &have); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}
		if !reflect.DeepEqual(have, tc.want) {
			t.Errorf("test %d, result mismatch, have\n%v\n, want\n%v\n", i, have, tc.want)
		}
	}
}

type Account struct {
	key  *ecdsa.PrivateKey
	addr common.Address
}

type Accounts []Account

func (a Accounts) Len() int           { return len(a) }
func (a Accounts) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Accounts) Less(i, j int) bool { return bytes.Compare(a[i].addr.Bytes(), a[j].addr.Bytes()) < 0 }

func newAccounts(n int) (accounts Accounts) {
	for i := 0; i < n; i++ {
		key, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(key.PublicKey)
		accounts = append(accounts, Account{key: key, addr: addr})
	}
	sort.Sort(accounts)
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
