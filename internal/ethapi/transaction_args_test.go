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
	"fmt"
	"math/big"
	"reflect"
	"testing"
	"time"

	ethereum "github.com/XinFinOrg/XDPoSChain"
	"github.com/XinFinOrg/XDPoSChain/XDCx"
	"github.com/XinFinOrg/XDPoSChain/XDCx/tradingstate"
	"github.com/XinFinOrg/XDPoSChain/XDCxlending"
	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/hexutil"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/bloombits"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/eth/downloader"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/event"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rpc"
)

// TestSetFeeDefaults tests the logic for filling in default fee values works as expected.
func TestSetFeeDefaults(t *testing.T) {
	type test struct {
		name     string
		isLondon bool
		in       *TransactionArgs
		want     *TransactionArgs
		err      error
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
			false,
			&TransactionArgs{},
			&TransactionArgs{GasPrice: fortytwo},
			nil,
		},
		{
			"legacy tx pre-London with zero price",
			false,
			&TransactionArgs{GasPrice: zero},
			&TransactionArgs{GasPrice: zero},
			nil,
		},
		{
			"legacy tx post-London, explicit gas price",
			true,
			&TransactionArgs{GasPrice: fortytwo},
			&TransactionArgs{GasPrice: fortytwo},
			nil,
		},
		{
			"legacy tx post-London with zero price",
			true,
			&TransactionArgs{GasPrice: zero},
			nil,
			errors.New("gasPrice must be non-zero after EIP-1559 fork"),
		},

		// Access list txs
		{
			"access list tx pre-London",
			false,
			&TransactionArgs{AccessList: al},
			&TransactionArgs{AccessList: al, GasPrice: fortytwo},
			nil,
		},
		{
			"access list tx post-London, explicit gas price",
			false,
			&TransactionArgs{AccessList: al, GasPrice: fortytwo},
			&TransactionArgs{AccessList: al, GasPrice: fortytwo},
			nil,
		},
		{
			"access list tx post-London",
			true,
			&TransactionArgs{AccessList: al},
			&TransactionArgs{AccessList: al, MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
		},
		{
			"access list tx post-London, only max fee",
			true,
			&TransactionArgs{AccessList: al, MaxFeePerGas: maxFee},
			&TransactionArgs{AccessList: al, MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
		},
		{
			"access list tx post-London, only priority fee",
			true,
			&TransactionArgs{AccessList: al, MaxFeePerGas: maxFee},
			&TransactionArgs{AccessList: al, MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
		},

		// Dynamic fee txs
		{
			"dynamic tx post-London",
			true,
			&TransactionArgs{},
			&TransactionArgs{MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
		},
		{
			"dynamic tx post-London, only max fee",
			true,
			&TransactionArgs{MaxFeePerGas: maxFee},
			&TransactionArgs{MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
		},
		{
			"dynamic tx post-London, only priority fee",
			true,
			&TransactionArgs{MaxFeePerGas: maxFee},
			&TransactionArgs{MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
		},
		{
			"dynamic fee tx pre-London, maxFee set",
			false,
			&TransactionArgs{MaxFeePerGas: maxFee},
			nil,
			fmt.Errorf("maxFeePerGas and maxPriorityFeePerGas are not valid before EIP-1559 is active"),
		},
		{
			"dynamic fee tx pre-London, priorityFee set",
			false,
			&TransactionArgs{MaxPriorityFeePerGas: fortytwo},
			nil,
			fmt.Errorf("maxFeePerGas and maxPriorityFeePerGas are not valid before EIP-1559 is active"),
		},
		{
			"dynamic fee tx, maxFee < priorityFee",
			true,
			&TransactionArgs{MaxFeePerGas: maxFee, MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1000))},
			nil,
			fmt.Errorf("maxFeePerGas (0x3e) < maxPriorityFeePerGas (0x3e8)"),
		},
		{
			"dynamic fee tx, maxFee < priorityFee while setting default",
			true,
			&TransactionArgs{MaxFeePerGas: (*hexutil.Big)(big.NewInt(7))},
			nil,
			fmt.Errorf("maxFeePerGas (0x7) < maxPriorityFeePerGas (0x2a)"),
		},
		{
			"dynamic fee tx post-London, explicit gas price",
			true,
			&TransactionArgs{MaxFeePerGas: zero, MaxPriorityFeePerGas: zero},
			nil,
			errors.New("maxFeePerGas must be non-zero"),
		},

		// Misc
		{
			"set all fee parameters",
			false,
			&TransactionArgs{GasPrice: fortytwo, MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
			fmt.Errorf("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified"),
		},
		{
			"set gas price and maxPriorityFee",
			false,
			&TransactionArgs{GasPrice: fortytwo, MaxPriorityFeePerGas: fortytwo},
			nil,
			fmt.Errorf("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified"),
		},
		{
			"set gas price and maxFee",
			true,
			&TransactionArgs{GasPrice: fortytwo, MaxFeePerGas: maxFee},
			nil,
			fmt.Errorf("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified"),
		},
	}

	ctx := context.Background()
	for i, test := range tests {
		if test.isLondon {
			b.activateLondon()
		} else {
			b.deactivateLondon()
		}
		got := test.in
		err := got.setFeeDefaults(ctx, b)
		if err != nil && err.Error() == test.err.Error() {
			// Test threw expected error.
			continue
		} else if err != nil {
			t.Fatalf("test %d (%s): unexpected error: %s", i, test.name, err)
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
	config := &params.ChainConfig{
		ChainId:             big.NewInt(42),
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
		BerlinBlock:         big.NewInt(0),
		Eip1559Block:        big.NewInt(1000),
	}
	return &backendMock{
		current: &types.Header{
			Difficulty: big.NewInt(10000000000),
			Number:     big.NewInt(1100),
			GasLimit:   8_000_000,
			GasUsed:    8_000_000,
			Time:       big.NewInt(555),
			Extra:      make([]byte, 32),
			BaseFee:    big.NewInt(10),
		},
		config: config,
	}
}

func (b *backendMock) activateLondon() {
	b.current.Number = big.NewInt(1100)
}

func (b *backendMock) deactivateLondon() {
	b.current.Number = big.NewInt(900)
}

func (b *backendMock) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return big.NewInt(42), nil
}

func (b *backendMock) CurrentHeader() *types.Header { return b.current }

func (b *backendMock) ChainConfig() *params.ChainConfig { return b.config }

// Other methods needed to implement Backend interface.
func (b *backendMock) SyncProgress() ethereum.SyncProgress { return ethereum.SyncProgress{} }
func (b *backendMock) FeeHistory(context.Context, uint64, rpc.BlockNumber, []float64) (*big.Int, [][]*big.Int, []*big.Int, []float64, error) {
	return nil, nil, nil, nil, nil
}

func (b *backendMock) BlobBaseFee(ctx context.Context) *big.Int {
	return new(big.Int)
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

func (b *backendMock) CurrentBlock() *types.Block { return nil }

func (b *backendMock) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	return nil, nil
}

func (b *backendMock) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return nil, nil
}

func (b *backendMock) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	return nil, nil
}

func (b *backendMock) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	return nil, nil, nil
}

func (b *backendMock) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	return nil, nil, nil
}

func (b *backendMock) PendingBlockAndReceipts() (*types.Block, types.Receipts) { return nil, nil }

func (b *backendMock) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return nil, nil
}

func (b *backendMock) GetTd(common.Hash) *big.Int {
	return nil
}

func (b *backendMock) GetEVM(context.Context, core.Message, *state.StateDB, *tradingstate.TradingStateDB, *types.Header, *vm.Config) (*vm.EVM, func() error, error) {
	return nil, nil, nil
}

func (b *backendMock) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription { return nil }
func (b *backendMock) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return nil
}
func (b *backendMock) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return nil
}
func (b *backendMock) SendTx(ctx context.Context, signedTx *types.Transaction) error { return nil }
func (b *backendMock) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	return nil, [32]byte{}, 0, 0, nil
}
func (b *backendMock) GetPoolTransactions() (types.Transactions, error)         { return nil, nil }
func (b *backendMock) GetPoolTransaction(txHash common.Hash) *types.Transaction { return nil }
func (b *backendMock) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return 0, nil
}
func (b *backendMock) Stats() (pending int, queued int) { return 0, 0 }
func (b *backendMock) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return nil, nil
}
func (b *backendMock) TxPoolContentFrom(addr common.Address) (types.Transactions, types.Transactions) {
	return nil, nil
}
func (b *backendMock) SubscribeNewTxsEvent(chan<- core.NewTxsEvent) event.Subscription { return nil }
func (b *backendMock) BloomStatus() (uint64, uint64)                                   { return 0, 0 }
func (b *backendMock) GetLogs(ctx context.Context, blockHash common.Hash, number uint64) ([][]*types.Log, error) {
	return nil, nil
}
func (b *backendMock) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {}
func (b *backendMock) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription         { return nil }
func (b *backendMock) SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return nil
}
func (b *backendMock) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return nil
}

func (b *backendMock) Engine() consensus.Engine { return nil }

func (b *backendMock) AreTwoBlockSamePath(bh1 common.Hash, bh2 common.Hash) bool {
	return true
}

func (b *backendMock) Downloader() *downloader.Downloader {
	return nil
}

func (b *backendMock) EventMux() *event.TypeMux {
	return nil
}

func (b *backendMock) GetBlock(context.Context, common.Hash) (*types.Block, error) {
	return nil, nil
}

func (b *backendMock) GetBlocksHashCache(blockNr uint64) []common.Hash {
	return []common.Hash{}
}

func (b *backendMock) GetEngine() consensus.Engine {
	return nil
}

func (b *backendMock) GetEpochDuration() *big.Int {
	return nil
}

func (b *backendMock) GetIPCClient() (bind.ContractBackend, error) {
	return nil, nil
}

func (b *backendMock) GetMasternodesCap(uint64) map[common.Address]*big.Int {
	return nil
}

func (b *backendMock) GetOrderNonce(common.Hash) (uint64, error) {
	return 0, nil
}

func (b *backendMock) GetBody(ctx context.Context, hash common.Hash, number rpc.BlockNumber) (*types.Body, error) {
	return nil, nil
}

func (b *backendMock) GetRewardByHash(common.Hash) map[string]map[string]map[string]*big.Int {
	return nil
}

func (b *backendMock) GetVotersCap(*big.Int, common.Address, []common.Address) map[common.Address]*big.Int {
	return nil
}

func (b *backendMock) GetVotersRewards(common.Address) map[common.Address]*big.Int {
	return nil
}

func (b *backendMock) LendingService() *XDCxlending.Lending {
	return nil
}

func (b *backendMock) OrderStats() (int, int) {
	return 0, 0
}

func (b *backendMock) OrderTxPoolContent() (map[common.Address]types.OrderTransactions, map[common.Address]types.OrderTransactions) {
	return nil, nil
}

func (b *backendMock) ProtocolVersion() int {
	return 0
}

func (b *backendMock) SendLendingTx(context.Context, *types.LendingTransaction) error {
	return nil
}

func (b *backendMock) SendOrderTx(context.Context, *types.OrderTransaction) error {
	return nil
}

func (b *backendMock) XDCxService() *XDCx.XDCX {
	return nil
}
