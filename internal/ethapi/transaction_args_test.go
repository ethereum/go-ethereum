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
	"context"
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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/bloombits"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

// TestSetFeeDefaults tests the logic for filling in default fee values works as expected.
func TestSetFeeDefaults(t *testing.T) {
	t.Parallel()

	type test struct {
		name string
		fork string // options: legacy, london, cancun
		in   *TransactionArgs
		want *TransactionArgs
		err  error
	}

	var (
		b        = newBackendMock()
		zero     = (*hexutil.Big)(big.NewInt(0))
		fortytwo = (*hexutil.Big)(big.NewInt(42))
		maxFee   = (*hexutil.Big)(new(big.Int).Add(new(big.Int).Mul(b.current.BaseFee, big.NewInt(2)), fortytwo.ToInt()))
		al       = &types.AccessList{types.AccessTuple{Address: common.Address{0xaa}, StorageKeys: []common.Hash{{0x01}}}}
	)

	tests := []test{
		// Legacy txs
		{
			"legacy tx pre-London",
			"legacy",
			&TransactionArgs{},
			&TransactionArgs{GasPrice: fortytwo},
			nil,
		},
		{
			"legacy tx pre-London with zero price",
			"legacy",
			&TransactionArgs{GasPrice: zero},
			&TransactionArgs{GasPrice: zero},
			nil,
		},
		{
			"legacy tx post-London, explicit gas price",
			"london",
			&TransactionArgs{GasPrice: fortytwo},
			&TransactionArgs{GasPrice: fortytwo},
			nil,
		},
		{
			"legacy tx post-London with zero price",
			"london",
			&TransactionArgs{GasPrice: zero},
			nil,
			errors.New("gasPrice must be non-zero after london fork"),
		},

		// Access list txs
		{
			"access list tx pre-London",
			"legacy",
			&TransactionArgs{AccessList: al},
			&TransactionArgs{AccessList: al, GasPrice: fortytwo},
			nil,
		},
		{
			"access list tx post-London, explicit gas price",
			"legacy",
			&TransactionArgs{AccessList: al, GasPrice: fortytwo},
			&TransactionArgs{AccessList: al, GasPrice: fortytwo},
			nil,
		},
		{
			"access list tx post-London",
			"london",
			&TransactionArgs{AccessList: al},
			&TransactionArgs{AccessList: al, MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
		},
		{
			"access list tx post-London, only max fee",
			"london",
			&TransactionArgs{AccessList: al, MaxFeePerGas: maxFee},
			&TransactionArgs{AccessList: al, MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
		},
		{
			"access list tx post-London, only priority fee",
			"london",
			&TransactionArgs{AccessList: al, MaxFeePerGas: maxFee},
			&TransactionArgs{AccessList: al, MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
		},

		// Dynamic fee txs
		{
			"dynamic tx post-London",
			"london",
			&TransactionArgs{},
			&TransactionArgs{MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
		},
		{
			"dynamic tx post-London, only max fee",
			"london",
			&TransactionArgs{MaxFeePerGas: maxFee},
			&TransactionArgs{MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
		},
		{
			"dynamic tx post-London, only priority fee",
			"london",
			&TransactionArgs{MaxFeePerGas: maxFee},
			&TransactionArgs{MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
		},
		{
			"dynamic fee tx pre-London, maxFee set",
			"legacy",
			&TransactionArgs{MaxFeePerGas: maxFee},
			nil,
			errors.New("maxFeePerGas and maxPriorityFeePerGas are not valid before London is active"),
		},
		{
			"dynamic fee tx pre-London, priorityFee set",
			"legacy",
			&TransactionArgs{MaxPriorityFeePerGas: fortytwo},
			nil,
			errors.New("maxFeePerGas and maxPriorityFeePerGas are not valid before London is active"),
		},
		{
			"dynamic fee tx, maxFee < priorityFee",
			"london",
			&TransactionArgs{MaxFeePerGas: maxFee, MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1000))},
			nil,
			errors.New("maxFeePerGas (0x3e) < maxPriorityFeePerGas (0x3e8)"),
		},
		{
			"dynamic fee tx, maxFee < priorityFee while setting default",
			"london",
			&TransactionArgs{MaxFeePerGas: (*hexutil.Big)(big.NewInt(7))},
			nil,
			errors.New("maxFeePerGas (0x7) < maxPriorityFeePerGas (0x2a)"),
		},
		{
			"dynamic fee tx post-London, explicit gas price",
			"london",
			&TransactionArgs{MaxFeePerGas: zero, MaxPriorityFeePerGas: zero},
			nil,
			errors.New("maxFeePerGas must be non-zero"),
		},

		// Misc
		{
			"set all fee parameters",
			"legacy",
			&TransactionArgs{GasPrice: fortytwo, MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
			errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified"),
		},
		{
			"set gas price and maxPriorityFee",
			"legacy",
			&TransactionArgs{GasPrice: fortytwo, MaxPriorityFeePerGas: fortytwo},
			nil,
			errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified"),
		},
		{
			"set gas price and maxFee",
			"london",
			&TransactionArgs{GasPrice: fortytwo, MaxFeePerGas: maxFee},
			nil,
			errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified"),
		},
		// EIP-4844
		{
			"set gas price and maxFee for blob transaction",
			"cancun",
			&TransactionArgs{GasPrice: fortytwo, MaxFeePerGas: maxFee, BlobHashes: []common.Hash{}},
			nil,
			errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified"),
		},
		{
			"fill maxFeePerBlobGas",
			"cancun",
			&TransactionArgs{BlobHashes: []common.Hash{}},
			&TransactionArgs{BlobHashes: []common.Hash{}, BlobFeeCap: (*hexutil.Big)(big.NewInt(4)), MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
		},
		{
			"fill maxFeePerBlobGas when dynamic fees are set",
			"cancun",
			&TransactionArgs{BlobHashes: []common.Hash{}, MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			&TransactionArgs{BlobHashes: []common.Hash{}, BlobFeeCap: (*hexutil.Big)(big.NewInt(4)), MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
		},
	}

	ctx := context.Background()
	for i, test := range tests {
		if err := b.setFork(test.fork); err != nil {
			t.Fatalf("failed to set fork: %v", err)
		}
		got := test.in
		err := got.setFeeDefaults(ctx, b)
		if err != nil {
			if test.err == nil {
				t.Fatalf("test %d (%s): unexpected error: %s", i, test.name, err)
			} else if err.Error() != test.err.Error() {
				t.Fatalf("test %d (%s): unexpected error: (got: %s, want: %s)", i, test.name, err, test.err)
			}
			// Matching error.
			continue
		} else if test.err != nil {
			t.Fatalf("test %d (%s): expected error: %s", i, test.name, test.err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Fatalf("test %d (%s): did not fill defaults as expected: (got: %v, want: %v)", i, test.name, got, test.want)
		}
	}
}

type backendMock struct {
	current *types.Header
	config  *params.ChainConfig
}

func newBackendMock() *backendMock {
	var cancunTime uint64 = 600
	config := &params.ChainConfig{
		ChainID:             big.NewInt(42),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(1000),
		CancunTime:          &cancunTime,
	}
	return &backendMock{
		current: &types.Header{
			Difficulty: big.NewInt(10000000000),
			Number:     big.NewInt(1100),
			GasLimit:   8_000_000,
			GasUsed:    8_000_000,
			Time:       555,
			Extra:      make([]byte, 32),
			BaseFee:    big.NewInt(10),
		},
		config: config,
	}
}

func (b *backendMock) setFork(fork string) error {
	if fork == "legacy" {
		b.current.Number = big.NewInt(900)
		b.current.Time = 555
	} else if fork == "london" {
		b.current.Number = big.NewInt(1100)
		b.current.Time = 555
	} else if fork == "cancun" {
		b.current.Number = big.NewInt(1100)
		b.current.Time = 700
		// Blob base fee will be 2
		excess := uint64(2314058)
		b.current.ExcessBlobGas = &excess
	} else {
		return errors.New("invalid fork")
	}
	return nil
}

func (b *backendMock) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return big.NewInt(42), nil
}
func (b *backendMock) BlobBaseFee(ctx context.Context) *big.Int { return big.NewInt(42) }

func (b *backendMock) CurrentHeader() *types.Header     { return b.current }
func (b *backendMock) ChainConfig() *params.ChainConfig { return b.config }

// Other methods needed to implement Backend interface.
func (b *backendMock) SyncProgress() ethereum.SyncProgress { return ethereum.SyncProgress{} }
func (b *backendMock) FeeHistory(ctx context.Context, blockCount uint64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*big.Int, [][]*big.Int, []*big.Int, []float64, []*big.Int, []float64, error) {
	return nil, nil, nil, nil, nil, nil, nil
}
func (b *backendMock) ChainDb() ethdb.Database           { return nil }
func (b *backendMock) AccountManager() *accounts.Manager { return nil }
func (b *backendMock) ExtRPCEnabled() bool               { return false }
func (b *backendMock) RPCGasCap() uint64                 { return 0 }
func (b *backendMock) RPCEVMTimeout() time.Duration      { return time.Second }
func (b *backendMock) RPCTxFeeCap() float64              { return 0 }
func (b *backendMock) UnprotectedAllowed() bool          { return false }
func (b *backendMock) SetHead(number uint64)             {}
func (b *backendMock) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	return nil, nil
}
func (b *backendMock) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return nil, nil
}
func (b *backendMock) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	return nil, nil
}
func (b *backendMock) CurrentBlock() *types.Header { return nil }
func (b *backendMock) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	return nil, nil
}
func (b *backendMock) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return nil, nil
}
func (b *backendMock) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	return nil, nil
}
func (b *backendMock) GetBody(ctx context.Context, hash common.Hash, number rpc.BlockNumber) (*types.Body, error) {
	return nil, nil
}
func (b *backendMock) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	return nil, nil, nil
}
func (b *backendMock) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	return nil, nil, nil
}
func (b *backendMock) Pending() (*types.Block, types.Receipts, *state.StateDB) { return nil, nil, nil }
func (b *backendMock) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return nil, nil
}
func (b *backendMock) GetLogs(ctx context.Context, blockHash common.Hash, number uint64) ([][]*types.Log, error) {
	return nil, nil
}
func (b *backendMock) GetTd(ctx context.Context, hash common.Hash) *big.Int { return nil }
func (b *backendMock) GetEVM(ctx context.Context, msg *core.Message, state *state.StateDB, header *types.Header, vmConfig *vm.Config, blockCtx *vm.BlockContext) *vm.EVM {
	return nil
}
func (b *backendMock) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription { return nil }
func (b *backendMock) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return nil
}
func (b *backendMock) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return nil
}
func (b *backendMock) SendTx(ctx context.Context, signedTx *types.Transaction) error { return nil }
func (b *backendMock) GetTransaction(ctx context.Context, txHash common.Hash) (bool, *types.Transaction, common.Hash, uint64, uint64, error) {
	return false, nil, [32]byte{}, 0, 0, nil
}
func (b *backendMock) GetPoolTransactions() (types.Transactions, error)         { return nil, nil }
func (b *backendMock) GetPoolTransaction(txHash common.Hash) *types.Transaction { return nil }
func (b *backendMock) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return 0, nil
}
func (b *backendMock) Stats() (pending int, queued int) { return 0, 0 }
func (b *backendMock) TxPoolContent() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction) {
	return nil, nil
}
func (b *backendMock) TxPoolContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction) {
	return nil, nil
}
func (b *backendMock) SubscribeNewTxsEvent(chan<- core.NewTxsEvent) event.Subscription      { return nil }
func (b *backendMock) BloomStatus() (uint64, uint64)                                        { return 0, 0 }
func (b *backendMock) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {}
func (b *backendMock) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription         { return nil }
func (b *backendMock) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return nil
}

func (b *backendMock) Engine() consensus.Engine { return nil }
