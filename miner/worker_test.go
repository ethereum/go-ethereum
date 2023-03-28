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

package miner

import (
	"math/big"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"gotest.tools/assert"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/bor"
	"github.com/ethereum/go-ethereum/consensus/bor/api"
	"github.com/ethereum/go-ethereum/consensus/bor/valset"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/tests/bor/mocks"
)

func TestGenerateBlockAndImportEthash(t *testing.T) {
	t.Parallel()

	testGenerateBlockAndImport(t, false, false)
}

func TestGenerateBlockAndImportClique(t *testing.T) {
	t.Parallel()

	testGenerateBlockAndImport(t, true, false)
}

func TestGenerateBlockAndImportBor(t *testing.T) {
	t.Parallel()

	testGenerateBlockAndImport(t, false, true)
}

//nolint:thelper
func testGenerateBlockAndImport(t *testing.T, isClique bool, isBor bool) {
	var (
		engine      consensus.Engine
		chainConfig *params.ChainConfig
		db          = rawdb.NewMemoryDatabase()
		ctrl        *gomock.Controller
	)

	if isBor {
		chainConfig = params.BorUnittestChainConfig

		engine, ctrl = getFakeBorFromConfig(t, chainConfig)
		defer ctrl.Finish()
	} else {
		if isClique {
			chainConfig = params.AllCliqueProtocolChanges
			chainConfig.Clique = &params.CliqueConfig{Period: 1, Epoch: 30000}
			engine = clique.New(chainConfig.Clique, db)
		} else {
			chainConfig = params.AllEthashProtocolChanges
			engine = ethash.NewFaker()
		}
	}

	defer engine.Close()

	chainConfig.LondonBlock = big.NewInt(0)

	w, b, _ := NewTestWorker(t, chainConfig, engine, db, 0, 0, 0)
	defer w.close()

	// This test chain imports the mined blocks.
	db2 := rawdb.NewMemoryDatabase()
	b.Genesis.MustCommit(db2)

	chain, _ := core.NewBlockChain(db2, nil, b.chain.Config(), engine, vm.Config{}, nil, nil, nil)
	defer chain.Stop()

	// Ignore empty commit here for less noise.
	w.skipSealHook = func(task *task) bool {
		return len(task.receipts) == 0
	}

	// Wait for mined blocks.
	sub := w.mux.Subscribe(core.NewMinedBlockEvent{})
	defer sub.Unsubscribe()

	// Start mining!
	w.start()

	var (
		err   error
		uncle *types.Block
	)

	for i := 0; i < 5; i++ {
		err = b.txPool.AddLocal(b.newRandomTx(true))
		if err != nil {
			t.Fatal("while adding a local transaction", err)
		}

		err = b.txPool.AddLocal(b.newRandomTx(false))
		if err != nil {
			t.Fatal("while adding a remote transaction", err)
		}

		uncle, err = b.newRandomUncle()
		if err != nil {
			t.Fatal("while making an uncle block", err)
		}

		w.postSideBlock(core.ChainSideEvent{Block: uncle})

		uncle, err = b.newRandomUncle()
		if err != nil {
			t.Fatal("while making an uncle block", err)
		}

		w.postSideBlock(core.ChainSideEvent{Block: uncle})

		select {
		case ev := <-sub.Chan():
			block := ev.Data.(core.NewMinedBlockEvent).Block
			if _, err := chain.InsertChain([]*types.Block{block}); err != nil {
				t.Fatalf("failed to insert new mined block %d: %v", block.NumberU64(), err)
			}
		case <-time.After(3 * time.Second): // Worker needs 1s to include new changes.
			t.Fatalf("timeout")
		}
	}
}

func getFakeBorFromConfig(t *testing.T, chainConfig *params.ChainConfig) (consensus.Engine, *gomock.Controller) {
	t.Helper()

	ctrl := gomock.NewController(t)

	ethAPIMock := api.NewMockCaller(ctrl)
	ethAPIMock.EXPECT().Call(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	spanner := bor.NewMockSpanner(ctrl)
	spanner.EXPECT().GetCurrentValidators(gomock.Any(), gomock.Any(), gomock.Any()).Return([]*valset.Validator{
		{
			ID:               0,
			Address:          TestBankAddress,
			VotingPower:      100,
			ProposerPriority: 0,
		},
	}, nil).AnyTimes()

	heimdallClientMock := mocks.NewMockIHeimdallClient(ctrl)
	heimdallClientMock.EXPECT().Close().Times(1)

	contractMock := bor.NewMockGenesisContract(ctrl)

	db, _, _ := NewDBForFakes(t)

	engine := NewFakeBor(t, db, chainConfig, ethAPIMock, spanner, heimdallClientMock, contractMock)

	return engine, ctrl
}

func TestEmptyWorkEthash(t *testing.T) {
	t.Skip()
	testEmptyWork(t, ethashChainConfig, ethash.NewFaker())
}
func TestEmptyWorkClique(t *testing.T) {
	t.Skip()
	testEmptyWork(t, cliqueChainConfig, clique.New(cliqueChainConfig.Clique, rawdb.NewMemoryDatabase()))
}

func testEmptyWork(t *testing.T, chainConfig *params.ChainConfig, engine consensus.Engine) {
	defer engine.Close()

	w, _, _ := NewTestWorker(t, chainConfig, engine, rawdb.NewMemoryDatabase(), 0, 0, 0)
	defer w.close()

	var (
		taskIndex int
		taskCh    = make(chan struct{}, 2)
	)

	checkEqual := func(t *testing.T, task *task, index int) {
		// The first empty work without any txs included
		receiptLen, balance := 0, big.NewInt(0)

		if index == 1 {
			// The second full work with 1 tx included
			receiptLen, balance = 1, big.NewInt(1000)
		}

		if len(task.receipts) != receiptLen {
			t.Fatalf("receipt number mismatch: have %d, want %d", len(task.receipts), receiptLen)
		}

		if task.state.GetBalance(testUserAddress).Cmp(balance) != 0 {
			t.Fatalf("account balance mismatch: have %d, want %d", task.state.GetBalance(testUserAddress), balance)
		}
	}

	w.newTaskHook = func(task *task) {
		if task.block.NumberU64() == 1 {
			checkEqual(t, task, taskIndex)
			taskIndex += 1
			taskCh <- struct{}{}
		}
	}

	w.skipSealHook = func(task *task) bool { return true }
	w.fullTaskHook = func() {
		time.Sleep(100 * time.Millisecond)
	}

	w.start() // Start mining!

	for i := 0; i < 2; i += 1 {
		select {
		case <-taskCh:
		case <-time.NewTimer(3 * time.Second).C:
			t.Error("new task timeout")
		}
	}
}

func TestStreamUncleBlock(t *testing.T) {
	ethash := ethash.NewFaker()
	defer ethash.Close()

	w, b, _ := NewTestWorker(t, ethashChainConfig, ethash, rawdb.NewMemoryDatabase(), 1, 0, 0)
	defer w.close()

	var taskCh = make(chan struct{})

	taskIndex := 0
	w.newTaskHook = func(task *task) {
		if task.block.NumberU64() == 2 {
			// The first task is an empty task, the second
			// one has 1 pending tx, the third one has 1 tx
			// and 1 uncle.
			if taskIndex == 2 {
				have := task.block.Header().UncleHash
				want := types.CalcUncleHash([]*types.Header{b.uncleBlock.Header()})
				if have != want {
					t.Errorf("uncle hash mismatch: have %s, want %s", have.Hex(), want.Hex())
				}
			}

			taskCh <- struct{}{}
			taskIndex += 1
		}
	}

	w.skipSealHook = func(task *task) bool {
		return true
	}

	w.fullTaskHook = func() {
		time.Sleep(100 * time.Millisecond)
	}

	w.start()

	for i := 0; i < 2; i += 1 {
		select {
		case <-taskCh:
		case <-time.NewTimer(time.Second).C:
			t.Error("new task timeout")
		}
	}

	w.postSideBlock(core.ChainSideEvent{Block: b.uncleBlock})

	select {
	case <-taskCh:
	case <-time.NewTimer(time.Second).C:
		t.Error("new task timeout")
	}
}

func TestRegenerateMiningBlockEthash(t *testing.T) {
	testRegenerateMiningBlock(t, ethashChainConfig, ethash.NewFaker())
}

func TestRegenerateMiningBlockClique(t *testing.T) {
	testRegenerateMiningBlock(t, cliqueChainConfig, clique.New(cliqueChainConfig.Clique, rawdb.NewMemoryDatabase()))
}

func testRegenerateMiningBlock(t *testing.T, chainConfig *params.ChainConfig, engine consensus.Engine) {
	defer engine.Close()

	w, b, _ := NewTestWorker(t, chainConfig, engine, rawdb.NewMemoryDatabase(), 0, 0, 0)
	defer w.close()

	var taskCh = make(chan struct{}, 3)

	taskIndex := 0
	w.newTaskHook = func(task *task) {
		if task.block.NumberU64() == 1 {
			// The first task is an empty task, the second
			// one has 1 pending tx, the third one has 2 txs
			if taskIndex == 2 {
				receiptLen, balance := 2, big.NewInt(2000)

				if len(task.receipts) != receiptLen {
					t.Errorf("receipt number mismatch: have %d, want %d", len(task.receipts), receiptLen)
				}

				if task.state.GetBalance(testUserAddress).Cmp(balance) != 0 {
					t.Errorf("account balance mismatch: have %d, want %d", task.state.GetBalance(testUserAddress), balance)
				}
			}

			taskCh <- struct{}{}
			taskIndex += 1
		}
	}

	w.skipSealHook = func(task *task) bool {
		return true
	}

	w.fullTaskHook = func() {
		time.Sleep(100 * time.Millisecond)
	}

	w.start()
	// Ignore the first two works
	for i := 0; i < 2; i += 1 {
		select {
		case <-taskCh:
		case <-time.NewTimer(time.Second).C:
			t.Error("new task timeout")
		}
	}

	b.txPool.AddLocals(newTxs)
	time.Sleep(time.Second)

	select {
	case <-taskCh:
	case <-time.NewTimer(time.Second).C:
		t.Error("new task timeout")
	}
}

func TestAdjustIntervalEthash(t *testing.T) {
	// Skipping this test as recommit interval would remain constant
	t.Skip()
	testAdjustInterval(t, ethashChainConfig, ethash.NewFaker())
}

func TestAdjustIntervalClique(t *testing.T) {

	// Skipping this test as recommit interval would remain constant
	t.Skip()
	testAdjustInterval(t, cliqueChainConfig, clique.New(cliqueChainConfig.Clique, rawdb.NewMemoryDatabase()))
}

func testAdjustInterval(t *testing.T, chainConfig *params.ChainConfig, engine consensus.Engine) {
	defer engine.Close()

	w, _, _ := NewTestWorker(t, chainConfig, engine, rawdb.NewMemoryDatabase(), 0, 0, 0)
	defer w.close()

	w.skipSealHook = func(task *task) bool {
		return true
	}
	w.fullTaskHook = func() {
		time.Sleep(100 * time.Millisecond)
	}
	var (
		progress = make(chan struct{}, 10)
		result   = make([]float64, 0, 10)
		index    = 0
		start    uint32
	)

	w.resubmitHook = func(minInterval time.Duration, recommitInterval time.Duration) {
		// Short circuit if interval checking hasn't started.
		if atomic.LoadUint32(&start) == 0 {
			return
		}
		var wantMinInterval, wantRecommitInterval time.Duration

		switch index {
		case 0:
			wantMinInterval, wantRecommitInterval = 3*time.Second, 3*time.Second
		case 1:
			origin := float64(3 * time.Second.Nanoseconds())
			estimate := origin*(1-intervalAdjustRatio) + intervalAdjustRatio*(origin/0.8+intervalAdjustBias)
			wantMinInterval, wantRecommitInterval = 3*time.Second, time.Duration(estimate)*time.Nanosecond
		case 2:
			estimate := result[index-1]
			min := float64(3 * time.Second.Nanoseconds())
			estimate = estimate*(1-intervalAdjustRatio) + intervalAdjustRatio*(min-intervalAdjustBias)
			wantMinInterval, wantRecommitInterval = 3*time.Second, time.Duration(estimate)*time.Nanosecond
		case 3:
			wantMinInterval, wantRecommitInterval = time.Second, time.Second
		}

		// Check interval
		if minInterval != wantMinInterval {
			t.Errorf("resubmit min interval mismatch: have %v, want %v ", minInterval, wantMinInterval)
		}
		if recommitInterval != wantRecommitInterval {
			t.Errorf("resubmit interval mismatch: have %v, want %v", recommitInterval, wantRecommitInterval)
		}

		result = append(result, float64(recommitInterval.Nanoseconds()))
		index += 1
		progress <- struct{}{}
	}

	w.start()

	time.Sleep(time.Second) // Ensure two tasks have been summitted due to start opt
	atomic.StoreUint32(&start, 1)

	w.setRecommitInterval(3 * time.Second)

	select {
	case <-progress:
	case <-time.NewTimer(time.Second).C:
		t.Error("interval reset timeout")
	}

	w.resubmitAdjustCh <- &intervalAdjust{inc: true, ratio: 0.8}

	select {
	case <-progress:
	case <-time.NewTimer(time.Second).C:
		t.Error("interval reset timeout")
	}

	w.resubmitAdjustCh <- &intervalAdjust{inc: false}

	select {
	case <-progress:
	case <-time.NewTimer(time.Second).C:
		t.Error("interval reset timeout")
	}

	w.setRecommitInterval(500 * time.Millisecond)

	select {
	case <-progress:
	case <-time.NewTimer(time.Second).C:
		t.Error("interval reset timeout")
	}
}

func TestGetSealingWorkEthash(t *testing.T) {
	testGetSealingWork(t, ethashChainConfig, ethash.NewFaker(), false)
}

func TestGetSealingWorkClique(t *testing.T) {
	testGetSealingWork(t, cliqueChainConfig, clique.New(cliqueChainConfig.Clique, rawdb.NewMemoryDatabase()), false)
}

func TestGetSealingWorkPostMerge(t *testing.T) {
	local := new(params.ChainConfig)
	*local = *ethashChainConfig
	local.TerminalTotalDifficulty = big.NewInt(0)
	testGetSealingWork(t, local, ethash.NewFaker(), true)
}

func testGetSealingWork(t *testing.T, chainConfig *params.ChainConfig, engine consensus.Engine, postMerge bool) {
	defer engine.Close()

	w, b, _ := NewTestWorker(t, chainConfig, engine, rawdb.NewMemoryDatabase(), 0, 0, 0)
	defer w.close()

	w.setExtra([]byte{0x01, 0x02})
	w.postSideBlock(core.ChainSideEvent{Block: b.uncleBlock})

	w.skipSealHook = func(task *task) bool {
		return true
	}

	w.fullTaskHook = func() {
		time.Sleep(100 * time.Millisecond)
	}

	timestamp := uint64(time.Now().Unix())
	assertBlock := func(block *types.Block, number uint64, coinbase common.Address, random common.Hash) {
		if block.Time() != timestamp {
			// Sometime the timestamp will be mutated if the timestamp
			// is even smaller than parent block's. It's OK.
			t.Logf("Invalid timestamp, want %d, get %d", timestamp, block.Time())
		}

		if len(block.Uncles()) != 0 {
			t.Error("Unexpected uncle block")
		}

		_, isClique := engine.(*clique.Clique)
		if !isClique {
			if len(block.Extra()) != 0 {
				t.Error("Unexpected extra field")
			}

			if block.Coinbase() != coinbase {
				t.Errorf("Unexpected coinbase got %x want %x", block.Coinbase(), coinbase)
			}
		} else {
			if block.Coinbase() != (common.Address{}) {
				t.Error("Unexpected coinbase")
			}
		}

		if !isClique {
			if block.MixDigest() != random {
				t.Error("Unexpected mix digest")
			}
		}

		if block.Nonce() != 0 {
			t.Error("Unexpected block nonce")
		}

		if block.NumberU64() != number {
			t.Errorf("Mismatched block number, want %d got %d", number, block.NumberU64())
		}
	}

	var cases = []struct {
		parent       common.Hash
		coinbase     common.Address
		random       common.Hash
		expectNumber uint64
		expectErr    bool
	}{
		{
			b.chain.Genesis().Hash(),
			common.HexToAddress("0xdeadbeef"),
			common.HexToHash("0xcafebabe"),
			uint64(1),
			false,
		},
		{
			b.chain.CurrentBlock().Hash(),
			common.HexToAddress("0xdeadbeef"),
			common.HexToHash("0xcafebabe"),
			b.chain.CurrentBlock().NumberU64() + 1,
			false,
		},
		{
			b.chain.CurrentBlock().Hash(),
			common.Address{},
			common.HexToHash("0xcafebabe"),
			b.chain.CurrentBlock().NumberU64() + 1,
			false,
		},
		{
			b.chain.CurrentBlock().Hash(),
			common.Address{},
			common.Hash{},
			b.chain.CurrentBlock().NumberU64() + 1,
			false,
		},
		{
			common.HexToHash("0xdeadbeef"),
			common.HexToAddress("0xdeadbeef"),
			common.HexToHash("0xcafebabe"),
			0,
			true,
		},
	}

	// This API should work even when the automatic sealing is not enabled
	for _, c := range cases {
		block, err := w.getSealingBlock(c.parent, timestamp, c.coinbase, c.random)

		if c.expectErr {
			if err == nil {
				t.Error("Expect error but get nil")
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error %v", err)
			}

			assertBlock(block, c.expectNumber, c.coinbase, c.random)
		}
	}

	// This API should work even when the automatic sealing is enabled
	w.start()

	for _, c := range cases {
		block, err := w.getSealingBlock(c.parent, timestamp, c.coinbase, c.random)
		if c.expectErr {
			if err == nil {
				t.Error("Expect error but get nil")
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error %v", err)
			}

			assertBlock(block, c.expectNumber, c.coinbase, c.random)
		}
	}
}

func TestCommitInterruptExperimentBor(t *testing.T) {
	t.Parallel()
	// with 1 sec block time and 200 millisec tx delay we should get 5 txs per block
	testCommitInterruptExperimentBor(t, 200, 5)

	// with 1 sec block time and 100 millisec tx delay we should get 10 txs per block
	testCommitInterruptExperimentBor(t, 100, 10)
}

func testCommitInterruptExperimentBor(t *testing.T, delay uint, txCount int) {
	t.Helper()

	var (
		engine      consensus.Engine
		chainConfig *params.ChainConfig
		db          = rawdb.NewMemoryDatabase()
		ctrl        *gomock.Controller
	)

	chainConfig = params.BorUnittestChainConfig

	log.Root().SetHandler(log.LvlFilterHandler(4, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	engine, ctrl = getFakeBorFromConfig(t, chainConfig)
	defer func() {
		engine.Close()
		ctrl.Finish()
	}()

	w, b, _ := NewTestWorker(t, chainConfig, engine, db, 0, 1, delay)
	defer w.close()

	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func() {
		wg.Done()

		for {
			tx := b.newRandomTx(false)
			if err := b.TxPool().AddRemote(tx); err != nil {
				t.Log(err)
			}

			time.Sleep(20 * time.Millisecond)
		}
	}()

	wg.Wait()

	// Start mining!
	w.start()
	time.Sleep(5 * time.Second)
	w.stop()

	assert.Equal(t, txCount, w.chain.CurrentBlock().Transactions().Len())
}

func BenchmarkBorMining(b *testing.B) {
	chainConfig := params.BorUnittestChainConfig

	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	ethAPIMock := api.NewMockCaller(ctrl)
	ethAPIMock.EXPECT().Call(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	spanner := bor.NewMockSpanner(ctrl)
	spanner.EXPECT().GetCurrentValidators(gomock.Any(), gomock.Any(), gomock.Any()).Return([]*valset.Validator{
		{
			ID:               0,
			Address:          TestBankAddress,
			VotingPower:      100,
			ProposerPriority: 0,
		},
	}, nil).AnyTimes()

	heimdallClientMock := mocks.NewMockIHeimdallClient(ctrl)
	heimdallClientMock.EXPECT().Close().Times(1)

	contractMock := bor.NewMockGenesisContract(ctrl)

	db, _, _ := NewDBForFakes(b)

	engine := NewFakeBor(b, db, chainConfig, ethAPIMock, spanner, heimdallClientMock, contractMock)
	defer engine.Close()

	chainConfig.LondonBlock = big.NewInt(0)

	w, back, _ := NewTestWorker(b, chainConfig, engine, db, 0, 0, 0)
	defer w.close()

	// This test chain imports the mined blocks.
	db2 := rawdb.NewMemoryDatabase()
	back.Genesis.MustCommit(db2)

	chain, _ := core.NewBlockChain(db2, nil, back.chain.Config(), engine, vm.Config{}, nil, nil, nil)
	defer chain.Stop()

	// Ignore empty commit here for less noise.
	w.skipSealHook = func(task *task) bool {
		return len(task.receipts) == 0
	}

	// fulfill tx pool
	const (
		totalGas    = testGas + params.TxGas
		totalBlocks = 10
	)

	var err error

	txInBlock := int(back.Genesis.GasLimit/totalGas) + 1

	// a bit risky
	for i := 0; i < 2*totalBlocks*txInBlock; i++ {
		err = back.txPool.AddLocal(back.newRandomTx(true))
		if err != nil {
			b.Fatal("while adding a local transaction", err)
		}

		err = back.txPool.AddLocal(back.newRandomTx(false))
		if err != nil {
			b.Fatal("while adding a remote transaction", err)
		}
	}

	// Wait for mined blocks.
	sub := w.mux.Subscribe(core.NewMinedBlockEvent{})
	defer sub.Unsubscribe()

	b.ResetTimer()

	prev := uint64(time.Now().Unix())

	// Start mining!
	w.start()

	blockPeriod, ok := back.Genesis.Config.Bor.Period["0"]
	if !ok {
		blockPeriod = 1
	}

	for i := 0; i < totalBlocks; i++ {
		select {
		case ev := <-sub.Chan():
			block := ev.Data.(core.NewMinedBlockEvent).Block

			if _, err := chain.InsertChain([]*types.Block{block}); err != nil {
				b.Fatalf("failed to insert new mined block %d: %v", block.NumberU64(), err)
			}

			b.Log("block", block.NumberU64(), "time", block.Time()-prev, "txs", block.Transactions().Len(), "gasUsed", block.GasUsed(), "gasLimit", block.GasLimit())

			prev = block.Time()
		case <-time.After(time.Duration(blockPeriod) * time.Second):
			b.Fatalf("timeout")
		}
	}
}
