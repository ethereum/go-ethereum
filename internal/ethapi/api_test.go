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
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"math/big"
	"reflect"
	"sort"
	"strings"
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
	"golang.org/x/crypto/sha3"
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
		tx, err := types.SignNewTx(key, signer, tt)
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

		//  rpcTransaction
		rpcTx := newRPCTransaction(tx, common.Hash{}, 0, 0, 0, nil, config)
		if data, err := json.Marshal(rpcTx); err != nil {
			t.Fatalf("test %d: marshalling failed; %v", i, err)
		} else if err = tx2.UnmarshalJSON(data); err != nil {
			t.Fatalf("test %d: unmarshal failed: %v", i, err)
		} else if want, have := tx.Hash(), tx2.Hash(); want != have {
			t.Fatalf("test %d: tx changed, want %x have %x", i, want, have)
		}
	}
}

func allTransactionTypes(addr common.Address, config *params.ChainConfig) []types.TxData {
	return []types.TxData{
		&types.LegacyTx{
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
		&types.LegacyTx{
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
		&types.AccessListTx{
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
		&types.AccessListTx{
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
		&types.DynamicFeeTx{
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
		&types.DynamicFeeTx{
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
	}
}

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
	return b.chain.GetHeaderByNumber(uint64(number)), nil
}
func (b testBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	panic("implement me")
}
func (b testBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	panic("implement me")
}
func (b testBackend) CurrentHeader() *types.Header { panic("implement me") }
func (b testBackend) CurrentBlock() *types.Header  { panic("implement me") }
func (b testBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	if number == rpc.LatestBlockNumber {
		head := b.chain.CurrentBlock()
		return b.chain.GetBlock(head.Hash(), head.Number.Uint64()), nil
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
func (b testBackend) GetTd(ctx context.Context, hash common.Hash) *big.Int { panic("implement me") }
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
		dad      = common.HexToAddress("0x0000000000000000000000000000000000000dad")
		genesis  = &core.Genesis{
			Config: params.TestChainConfig,
			Alloc: core.GenesisAlloc{
				accounts[0].addr: {Balance: big.NewInt(params.Ether)},
				accounts[1].addr: {Balance: big.NewInt(params.Ether)},
				accounts[2].addr: {Balance: big.NewInt(params.Ether)},
				dad: {
					Balance: big.NewInt(params.Ether),
					Nonce:   1,
					Storage: map[common.Hash]common.Hash{
						common.Hash{}: common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
					},
				},
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
		// Clear storage trie
		{
			blockNumber: rpc.LatestBlockNumber,
			call: TransactionArgs{
				From: &accounts[1].addr,
				// Yul:
				// object "Test" {
				//    code {
				//        let dad := 0x0000000000000000000000000000000000000dad
				//        if eq(balance(dad), 0) {
				//            revert(0, 0)
				//        }
				//        let slot := sload(0)
				//        mstore(0, slot)
				//        return(0, 32)
				//    }
				// }
				Input: hex2Bytes("610dad6000813103600f57600080fd5b6000548060005260206000f3"),
			},
			overrides: StateOverride{
				dad: OverrideAccount{
					State: &map[common.Hash]common.Hash{},
				},
			},
			want: "0x0000000000000000000000000000000000000000000000000000000000000000",
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

func TestMulticallV1(t *testing.T) {
	t.Parallel()
	// Initialize test accounts
	var (
		accounts  = newAccounts(3)
		genBlocks = 10
		signer    = types.HomesteadSigner{}
		cac       = common.HexToAddress("0x0000000000000000000000000000000000000cac")
		genesis   = &core.Genesis{
			Config: params.TestChainConfig,
			Alloc: core.GenesisAlloc{
				accounts[0].addr: {Balance: big.NewInt(params.Ether)},
				accounts[1].addr: {Balance: big.NewInt(params.Ether)},
				accounts[2].addr: {Balance: big.NewInt(params.Ether)},
				// Yul:
				// object "Test" {
				//     code {
				//         let dad := 0x0000000000000000000000000000000000000dad
				//         selfdestruct(dad)
				//     }
				// }
				cac: {Balance: big.NewInt(params.Ether), Code: common.Hex2Bytes("610dad80ff")},
			},
		}
		n10hash       = crypto.Keccak256Hash([]byte{0xa}).Hex()
		sha256Address = common.BytesToAddress([]byte{0x02})
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
		randomAccounts   = newAccounts(4)
		latest           = rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
		includeTransfers = true
	)
	type callRes struct {
		ReturnValue string `json:"return"`
		Error       string
		Logs        []types.Log
		GasUsed     string
		Status      string
		Transfers   []transfer
	}
	type blockRes struct {
		Number string
		Hash   string
		// Ignore timestamp
		GasLimit     string
		GasUsed      string
		FeeRecipient string
		BaseFee      string
		Calls        []callRes
	}
	var testSuite = []struct {
		name             string
		blocks           []CallBatch
		tag              rpc.BlockNumberOrHash
		includeTransfers *bool
		expectErr        error
		want             []blockRes
	}{
		// State build-up over calls:
		// First value transfer OK after state override.
		// Second one should succeed because of first transfer.
		{
			name: "simple",
			tag:  latest,
			blocks: []CallBatch{{
				StateOverrides: &StateOverride{
					randomAccounts[0].addr: OverrideAccount{Balance: newRPCBalance(big.NewInt(1000))},
				},
				Calls: []TransactionArgs{{
					From:  &randomAccounts[0].addr,
					To:    &randomAccounts[1].addr,
					Value: (*hexutil.Big)(big.NewInt(1000)),
				}, {
					From:  &randomAccounts[1].addr,
					To:    &randomAccounts[2].addr,
					Value: (*hexutil.Big)(big.NewInt(1000)),
				}},
			}},
			want: []blockRes{{
				Number:       "0xa",
				Hash:         n10hash,
				GasLimit:     "0x47e7c4",
				GasUsed:      "0xa410",
				FeeRecipient: "0x0000000000000000000000000000000000000000",
				Calls: []callRes{{
					ReturnValue: "0x",
					GasUsed:     "0x5208",
					Logs:        []types.Log{},
					Status:      "0x1",
				}, {
					ReturnValue: "0x",
					GasUsed:     "0x5208",
					Logs:        []types.Log{},
					Status:      "0x1",
				}},
			}},
		}, {
			// State build-up over blocks.
			name: "simple-multi-block",
			tag:  latest,
			blocks: []CallBatch{{
				StateOverrides: &StateOverride{
					randomAccounts[0].addr: OverrideAccount{Balance: newRPCBalance(big.NewInt(2000))},
				},
				Calls: []TransactionArgs{
					{
						From:  &randomAccounts[0].addr,
						To:    &randomAccounts[1].addr,
						Value: (*hexutil.Big)(big.NewInt(1000)),
					}, {
						From:  &randomAccounts[0].addr,
						To:    &randomAccounts[3].addr,
						Value: (*hexutil.Big)(big.NewInt(1000)),
					},
				},
			}, {
				StateOverrides: &StateOverride{
					randomAccounts[3].addr: OverrideAccount{Balance: newRPCBalance(big.NewInt(0))},
				},
				Calls: []TransactionArgs{
					{
						From:  &randomAccounts[1].addr,
						To:    &randomAccounts[2].addr,
						Value: (*hexutil.Big)(big.NewInt(1000)),
					}, {
						From:  &randomAccounts[3].addr,
						To:    &randomAccounts[2].addr,
						Value: (*hexutil.Big)(big.NewInt(1000)),
					},
				},
			}},
			want: []blockRes{{
				Number:       "0xa",
				Hash:         n10hash,
				GasLimit:     "0x47e7c4",
				GasUsed:      "0xa410",
				FeeRecipient: "0x0000000000000000000000000000000000000000",
				Calls: []callRes{{
					ReturnValue: "0x",
					GasUsed:     "0x5208",
					Logs:        []types.Log{},
					Status:      "0x1",
				}, {
					ReturnValue: "0x",
					GasUsed:     "0x5208",
					Logs:        []types.Log{},
					Status:      "0x1",
				}},
			}, {
				Number:       "0xa",
				Hash:         n10hash,
				GasLimit:     "0x47e7c4",
				GasUsed:      "0x5208",
				FeeRecipient: "0x0000000000000000000000000000000000000000",
				Calls: []callRes{{
					ReturnValue: "0x",
					GasUsed:     "0x5208",
					Logs:        []types.Log{},
					Status:      "0x1",
				}, {
					ReturnValue: "0x",
					GasUsed:     "0x0",
					Logs:        []types.Log{},
					Status:      "0x0",
					Error:       fmt.Sprintf("err: insufficient funds for gas * price + value: address %s have 0 want 1000 (supplied gas 9937000)", randomAccounts[3].addr.String()),
				}},
			}},
		}, {
			// Block overrides should work, each call is simulated on a different block number
			name: "block-overrides",
			tag:  latest,
			blocks: []CallBatch{{
				BlockOverrides: &BlockOverrides{
					Number: (*hexutil.Big)(big.NewInt(11)),
				},
				Calls: []TransactionArgs{
					{
						From: &accounts[0].addr,
						Input: &hexutil.Bytes{
							0x43,             // NUMBER
							0x60, 0x00, 0x52, // MSTORE offset 0
							0x60, 0x20, 0x60, 0x00, 0xf3, // RETURN
						},
					},
				},
			}, {
				BlockOverrides: &BlockOverrides{
					Number: (*hexutil.Big)(big.NewInt(12)),
				},
				Calls: []TransactionArgs{{
					From: &accounts[1].addr,
					Input: &hexutil.Bytes{
						0x43,             // NUMBER
						0x60, 0x00, 0x52, // MSTORE offset 0
						0x60, 0x20, 0x60, 0x00, 0xf3,
					},
				}},
			}},
			want: []blockRes{{
				Number:       "0xb",
				Hash:         crypto.Keccak256Hash([]byte{0xb}).Hex(),
				GasLimit:     "0x47e7c4",
				GasUsed:      "0xe891",
				FeeRecipient: "0x0000000000000000000000000000000000000000",
				Calls: []callRes{{
					ReturnValue: "0x000000000000000000000000000000000000000000000000000000000000000b",
					GasUsed:     "0xe891",
					Logs:        []types.Log{},
					Status:      "0x1",
				}},
			}, {
				Number:       "0xc",
				Hash:         crypto.Keccak256Hash([]byte{0xc}).Hex(),
				GasLimit:     "0x47e7c4",
				GasUsed:      "0xe891",
				FeeRecipient: "0x0000000000000000000000000000000000000000",
				Calls: []callRes{{
					ReturnValue: "0x000000000000000000000000000000000000000000000000000000000000000c",
					GasUsed:     "0xe891",
					Logs:        []types.Log{},
					Status:      "0x1",
				}},
			}},
		},
		// Block numbers must be in order.
		{
			name: "block-number-order",
			tag:  latest,
			blocks: []CallBatch{{
				BlockOverrides: &BlockOverrides{
					Number: (*hexutil.Big)(big.NewInt(12)),
				},
				Calls: []TransactionArgs{{
					From: &accounts[1].addr,
					Input: &hexutil.Bytes{
						0x43,             // NUMBER
						0x60, 0x00, 0x52, // MSTORE offset 0
						0x60, 0x20, 0x60, 0x00, 0xf3, // RETURN
					},
				}},
			}, {
				BlockOverrides: &BlockOverrides{
					Number: (*hexutil.Big)(big.NewInt(11)),
				},
				Calls: []TransactionArgs{{
					From: &accounts[0].addr,
					Input: &hexutil.Bytes{
						0x43,             // NUMBER
						0x60, 0x00, 0x52, // MSTORE offset 0
						0x60, 0x20, 0x60, 0x00, 0xf3, // RETURN
					},
				}},
			}},
			want:      []blockRes{},
			expectErr: errors.New("block numbers must be in order"),
		},
		// Test on solidity storage example. Set value in one call, read in next.
		{
			name: "storage-contract",
			tag:  latest,
			blocks: []CallBatch{{
				StateOverrides: &StateOverride{
					randomAccounts[2].addr: OverrideAccount{
						Code: hex2Bytes("608060405234801561001057600080fd5b50600436106100365760003560e01c80632e64cec11461003b5780636057361d14610059575b600080fd5b610043610075565b60405161005091906100d9565b60405180910390f35b610073600480360381019061006e919061009d565b61007e565b005b60008054905090565b8060008190555050565b60008135905061009781610103565b92915050565b6000602082840312156100b3576100b26100fe565b5b60006100c184828501610088565b91505092915050565b6100d3816100f4565b82525050565b60006020820190506100ee60008301846100ca565b92915050565b6000819050919050565b600080fd5b61010c816100f4565b811461011757600080fd5b5056fea2646970667358221220404e37f487a89a932dca5e77faaf6ca2de3b991f93d230604b1b8daaef64766264736f6c63430008070033"),
					},
				},
				Calls: []TransactionArgs{{
					// Set value to 5
					From:  &randomAccounts[0].addr,
					To:    &randomAccounts[2].addr,
					Input: hex2Bytes("6057361d0000000000000000000000000000000000000000000000000000000000000005"),
				}, {
					// Read value
					From:  &randomAccounts[0].addr,
					To:    &randomAccounts[2].addr,
					Input: hex2Bytes("2e64cec1"),
				},
				},
			}},
			want: []blockRes{{
				Number:       "0xa",
				Hash:         n10hash,
				GasLimit:     "0x47e7c4",
				GasUsed:      "0x10683",
				FeeRecipient: "0x0000000000000000000000000000000000000000",
				Calls: []callRes{{
					ReturnValue: "0x",
					GasUsed:     "0xaacc",
					Logs:        []types.Log{},
					Status:      "0x1",
				}, {
					ReturnValue: "0x0000000000000000000000000000000000000000000000000000000000000005",
					GasUsed:     "0x5bb7",
					Logs:        []types.Log{},
					Status:      "0x1",
				}},
			}},
		},
		// Test logs output.
		{
			name: "logs",
			tag:  latest,
			blocks: []CallBatch{{
				StateOverrides: &StateOverride{
					randomAccounts[2].addr: OverrideAccount{
						// Yul code:
						// object "Test" {
						//    code {
						//        let hash:u256 := 0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff
						//        log1(0, 0, hash)
						//        return (0, 0)
						//    }
						// }
						Code: hex2Bytes("7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80600080a1600080f3"),
					},
				},
				Calls: []TransactionArgs{{
					From: &randomAccounts[0].addr,
					To:   &randomAccounts[2].addr,
				}},
			}},
			want: []blockRes{{
				Number:       "0xa",
				Hash:         n10hash,
				GasLimit:     "0x47e7c4",
				GasUsed:      "0x5508",
				FeeRecipient: "0x0000000000000000000000000000000000000000",
				Calls: []callRes{{
					ReturnValue: "0x",
					Logs: []types.Log{{
						Address:     randomAccounts[2].addr,
						Topics:      []common.Hash{common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")},
						BlockNumber: 10,
						Data:        []byte{},
					}},
					GasUsed: "0x5508",
					Status:  "0x1",
				}},
			}},
		},
		// Test ecrecover override
		{
			name: "ecrecover-override",
			tag:  latest,
			blocks: []CallBatch{{
				StateOverrides: &StateOverride{
					randomAccounts[2].addr: OverrideAccount{
						// Yul code that returns ecrecover(0, 0, 0, 0).
						// object "Test" {
						//    code {
						//        // Free memory pointer
						//        let free_ptr := mload(0x40)
						//
						//        // Initialize inputs with zeros
						//        mstore(free_ptr, 0)  // Hash
						//        mstore(add(free_ptr, 0x20), 0)  // v
						//        mstore(add(free_ptr, 0x40), 0)  // r
						//        mstore(add(free_ptr, 0x60), 0)  // s
						//
						//        // Call ecrecover precompile (at address 1) with all 0 inputs
						//        let success := staticcall(gas(), 1, free_ptr, 0x80, free_ptr, 0x20)
						//
						//        // Check if the call was successful
						//        if eq(success, 0) {
						//            revert(0, 0)
						//        }
						//
						//        // Return the recovered address
						//        return(free_ptr, 0x14)
						//    }
						// }
						Code: hex2Bytes("6040516000815260006020820152600060408201526000606082015260208160808360015afa60008103603157600080fd5b601482f3"),
					},
					common.BytesToAddress([]byte{0x01}): OverrideAccount{
						// Yul code that returns the address of the caller.
						// object "Test" {
						//    code {
						//        let c := caller()
						//        mstore(0, c)
						//        return(0xc, 0x14)
						//    }
						// }
						Code: hex2Bytes("33806000526014600cf3"),
					},
				},
				BlockOverrides: &BlockOverrides{},
				Calls: []TransactionArgs{{
					From: &randomAccounts[0].addr,
					To:   &randomAccounts[2].addr,
				}},
			}},
			want: []blockRes{{
				Number:       "0xa",
				Hash:         n10hash,
				GasLimit:     "0x47e7c4",
				GasUsed:      "0x52f6",
				FeeRecipient: "0x0000000000000000000000000000000000000000",
				Calls: []callRes{{
					// Caller is in this case the contract that invokes ecrecover.
					ReturnValue: strings.ToLower(randomAccounts[2].addr.String()),
					GasUsed:     "0x52f6",
					Logs:        []types.Log{},
					Status:      "0x1",
				}},
			}},
		},
		// Test moving the sha256 precompile.
		{
			name: "precompile-move",
			tag:  latest,
			blocks: []CallBatch{{
				StateOverrides: &StateOverride{
					sha256Address: OverrideAccount{
						// Yul code that returns the calldata.
						// object "Test" {
						//    code {
						//        let size := calldatasize() // Get the size of the calldata
						//
						//        // Allocate memory to store the calldata
						//        let memPtr := msize()
						//
						//        // Copy calldata to memory
						//        calldatacopy(memPtr, 0, size)
						//
						//        // Return the calldata from memory
						//        return(memPtr, size)
						//    }
						// }
						Code:   hex2Bytes("365981600082378181f3"),
						MoveTo: &randomAccounts[2].addr,
					},
				},
				Calls: []TransactionArgs{{
					From:  &randomAccounts[0].addr,
					To:    &randomAccounts[2].addr,
					Input: hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
				}, {
					From:  &randomAccounts[0].addr,
					To:    &sha256Address,
					Input: hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
				}},
			}},
			want: []blockRes{{
				Number:       "0xa",
				Hash:         n10hash,
				GasLimit:     "0x47e7c4",
				GasUsed:      "0xa58c",
				FeeRecipient: "0x0000000000000000000000000000000000000000",
				Calls: []callRes{{
					ReturnValue: "0xec4916dd28fc4c10d78e287ca5d9cc51ee1ae73cbfde08c6b37324cbfaac8bc5",
					GasUsed:     "0x52dc",
					Logs:        []types.Log{},
					Status:      "0x1",
				}, {
					ReturnValue: "0x0000000000000000000000000000000000000000000000000000000000000001",
					GasUsed:     "0x52b0",
					Logs:        []types.Log{},
					Status:      "0x1",
				}},
			}},
		},
		// Test ether transfers.
		{
			name: "transfer-logs",
			tag:  latest,
			blocks: []CallBatch{{
				StateOverrides: &StateOverride{
					randomAccounts[0].addr: OverrideAccount{
						Balance: newRPCBalance(big.NewInt(100)),
						// Yul code that transfers 100 wei to address passed in calldata:
						// object "Test" {
						//    code {
						//        let recipient := shr(96, calldataload(0))
						//        let value := 100
						//        let success := call(gas(), recipient, value, 0, 0, 0, 0)
						//        if eq(success, 0) {
						//            revert(0, 0)
						//        }
						//    }
						// }
						Code: hex2Bytes("60003560601c606460008060008084865af160008103601d57600080fd5b505050"),
					},
				},
				Calls: []TransactionArgs{{
					From:  &accounts[0].addr,
					To:    &randomAccounts[0].addr,
					Value: (*hexutil.Big)(big.NewInt(50)),
					Input: hex2Bytes(strings.TrimPrefix(randomAccounts[1].addr.String(), "0x")),
				}},
			}},
			includeTransfers: &includeTransfers,
			want: []blockRes{{
				Number:       "0xa",
				Hash:         n10hash,
				GasLimit:     "0x47e7c4",
				GasUsed:      "0xd984",
				FeeRecipient: "0x0000000000000000000000000000000000000000",
				Calls: []callRes{{
					ReturnValue: "0x",
					GasUsed:     "0xd984",
					Transfers: []transfer{
						{
							From:  accounts[0].addr,
							To:    randomAccounts[0].addr,
							Value: big.NewInt(50),
						}, {
							From:  randomAccounts[0].addr,
							To:    randomAccounts[1].addr,
							Value: big.NewInt(100),
						},
					},
					Logs:   []types.Log{},
					Status: "0x1",
				}},
			}},
		},
		// Tests selfdestructed contract.
		{
			name: "selfdestruct",
			tag:  latest,
			blocks: []CallBatch{{
				Calls: []TransactionArgs{{
					From: &accounts[0].addr,
					To:   &cac,
				}, {
					From: &accounts[0].addr,
					// Check that cac is selfdestructed and balance transferred to dad.
					// object "Test" {
					//    code {
					//        let cac := 0x0000000000000000000000000000000000000cac
					//        let dad := 0x0000000000000000000000000000000000000dad
					//        if gt(balance(cac), 0) {
					//            revert(0, 0)
					//        }
					//        if gt(extcodesize(cac), 0) {
					//            revert(0, 0)
					//        }
					//        if eq(balance(dad), 0) {
					//            revert(0, 0)
					//        }
					//    }
					// }
					Input: hex2Bytes("610cac610dad600082311115601357600080fd5b6000823b1115602157600080fd5b6000813103602e57600080fd5b5050"),
				}},
			}},
			want: []blockRes{{
				Number:       "0xa",
				Hash:         n10hash,
				GasLimit:     "0x47e7c4",
				GasUsed:      "0x1b83f",
				FeeRecipient: "0x0000000000000000000000000000000000000000",
				Calls: []callRes{{
					ReturnValue: "0x",
					GasUsed:     "0xd166",
					Logs:        []types.Log{},
					Status:      "0x1",
				}, {
					ReturnValue: "0x",
					GasUsed:     "0xe6d9",
					Logs:        []types.Log{},
					Status:      "0x1",
				}},
			}},
		},
	}

	for i, tc := range testSuite {
		t.Run(tc.name, func(t *testing.T) {
			opts := multicallOpts{BlockStateCalls: tc.blocks}
			if tc.includeTransfers != nil && *tc.includeTransfers {
				opts.TraceTransfers = true
			}
			result, err := api.MulticallV1(context.Background(), opts, tc.tag)
			if tc.expectErr != nil {
				if err == nil {
					t.Fatalf("test %d: want error %v, have nothing", i, tc.expectErr)
				}
				if !errors.Is(err, tc.expectErr) {
					// Second try
					if !reflect.DeepEqual(err, tc.expectErr) {
						t.Errorf("test %d: error mismatch, want %v, have %v", i, tc.expectErr, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("test %d: want no error, have %v", i, err)
			}
			// Turn result into res-struct
			var have []blockRes
			resBytes, _ := json.Marshal(result)
			if err := json.Unmarshal(resBytes, &have); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}
			if !reflect.DeepEqual(have, tc.want) {
				t.Errorf("test %d, result mismatch, have\n%v\n, want\n%v\n", i, have, tc.want)
			}
		})
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
	rpcBytes := hexutil.Bytes(common.FromHex(str))
	return &rpcBytes
}

// testHasher is the helper tool for transaction/receipt list hashing.
// The original hasher is trie, in order to get rid of import cycle,
// use the testing hasher instead.
type testHasher struct {
	hasher hash.Hash
}

func newHasher() *testHasher {
	return &testHasher{hasher: sha3.NewLegacyKeccak256()}
}

func (h *testHasher) Reset() {
	h.hasher.Reset()
}

func (h *testHasher) Update(key, val []byte) error {
	h.hasher.Write(key)
	h.hasher.Write(val)
	return nil
}

func (h *testHasher) Hash() common.Hash {
	return common.BytesToHash(h.hasher.Sum(nil))
}

func TestRPCMarshalBlock(t *testing.T) {
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
	block := types.NewBlock(&types.Header{Number: big.NewInt(100)}, txs, nil, nil, newHasher())

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
		resp, err := RPCMarshalBlock(block, tc.inclTx, tc.fullTx, params.MainnetChainConfig)
		if err != nil {
			t.Errorf("test %d: got error %v", i, err)
			continue
		}
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
