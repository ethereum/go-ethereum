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
	"errors"
	"math/big"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
)

const (
	// testCode is the testing contract binary code which will initialises some
	// variables in constructor
	testCode = "0x60806040527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0060005534801561003457600080fd5b5060fc806100436000396000f3fe6080604052348015600f57600080fd5b506004361060325760003560e01c80630c4dae8814603757806398a213cf146053575b600080fd5b603d607e565b6040518082815260200191505060405180910390f35b607c60048036036020811015606757600080fd5b81019080803590602001909291905050506084565b005b60005481565b806000819055507fe9e44f9f7da8c559de847a3232b57364adc0354f15a2cd8dc636d54396f9587a6000546040518082815260200191505060405180910390a15056fea265627a7a723058208ae31d9424f2d0bc2a3da1a5dd659db2d71ec322a17db8f87e19e209e3a1ff4a64736f6c634300050a0032"

	// testGas is the gas required for contract deployment.
	testGas = 144109
)

var (
	// Test chain configurations
	testTxPoolConfig  txpool.Config
	ethashChainConfig *params.ChainConfig
	cliqueChainConfig *params.ChainConfig

	// Test accounts
	testBankKey, _  = crypto.GenerateKey()
	testBankAddress = crypto.PubkeyToAddress(testBankKey.PublicKey)
	testBankFunds   = big.NewInt(1000000000000000000)

	testUserKey, _  = crypto.GenerateKey()
	testUserAddress = crypto.PubkeyToAddress(testUserKey.PublicKey)

	// Test transactions
	pendingTxs []*types.Transaction
	newTxs     []*types.Transaction

	testConfig = &Config{
		Recommit: time.Second,
		GasCeil:  params.GenesisGasLimit,
	}
)

func init() {
	testTxPoolConfig = txpool.DefaultConfig
	testTxPoolConfig.Journal = ""
	ethashChainConfig = new(params.ChainConfig)
	*ethashChainConfig = *params.TestChainConfig
	cliqueChainConfig = new(params.ChainConfig)
	*cliqueChainConfig = *params.TestChainConfig
	cliqueChainConfig.Clique = &params.CliqueConfig{
		Period: 10,
		Epoch:  30000,
	}

	signer := types.LatestSigner(params.TestChainConfig)
	tx1 := types.MustSignNewTx(testBankKey, signer, &types.AccessListTx{
		ChainID:  params.TestChainConfig.ChainID,
		Nonce:    0,
		To:       &testUserAddress,
		Value:    big.NewInt(1000),
		Gas:      params.TxGas,
		GasPrice: big.NewInt(params.InitialBaseFee),
	})
	pendingTxs = append(pendingTxs, tx1)

	tx2 := types.MustSignNewTx(testBankKey, signer, &types.LegacyTx{
		Nonce:    1,
		To:       &testUserAddress,
		Value:    big.NewInt(1000),
		Gas:      params.TxGas,
		GasPrice: big.NewInt(params.InitialBaseFee),
	})
	newTxs = append(newTxs, tx2)

	rand.Seed(time.Now().UnixNano())
}

// testWorkerBackend implements worker.Backend interfaces and wraps all information needed during the testing.
type testWorkerBackend struct {
	db         ethdb.Database
	txPool     *txpool.TxPool
	chain      *core.BlockChain
	genesis    *core.Genesis
	uncleBlock *types.Block
}

func newTestWorkerBackend(t *testing.T, chainConfig *params.ChainConfig, engine consensus.Engine, db ethdb.Database, n int) *testWorkerBackend {
	var gspec = &core.Genesis{
		Config: chainConfig,
		Alloc:  core.GenesisAlloc{testBankAddress: {Balance: testBankFunds}},
	}
	switch e := engine.(type) {
	case *clique.Clique:
		gspec.ExtraData = make([]byte, 32+common.AddressLength+crypto.SignatureLength)
		copy(gspec.ExtraData[32:32+common.AddressLength], testBankAddress.Bytes())
		e.Authorize(testBankAddress, func(account accounts.Account, s string, data []byte) ([]byte, error) {
			return crypto.Sign(crypto.Keccak256(data), testBankKey)
		})
	case *ethash.Ethash:
	default:
		t.Fatalf("unexpected consensus engine type: %T", engine)
	}
	chain, err := core.NewBlockChain(db, &core.CacheConfig{TrieDirtyDisabled: true}, gspec, nil, engine, vm.Config{}, nil, nil)
	if err != nil {
		t.Fatalf("core.NewBlockChain failed: %v", err)
	}
	txpool := txpool.NewTxPool(testTxPoolConfig, chainConfig, chain)

	// Generate a small n-block chain and an uncle block for it
	var uncle *types.Block
	if n > 0 {
		genDb, blocks, _ := core.GenerateChainWithGenesis(gspec, engine, n, func(i int, gen *core.BlockGen) {
			gen.SetCoinbase(testBankAddress)
		})
		if _, err := chain.InsertChain(blocks); err != nil {
			t.Fatalf("failed to insert origin chain: %v", err)
		}
		parent := chain.GetBlockByHash(chain.CurrentBlock().ParentHash())
		blocks, _ = core.GenerateChain(chainConfig, parent, engine, genDb, 1, func(i int, gen *core.BlockGen) {
			gen.SetCoinbase(testUserAddress)
		})
		uncle = blocks[0]
	} else {
		_, blocks, _ := core.GenerateChainWithGenesis(gspec, engine, 1, func(i int, gen *core.BlockGen) {
			gen.SetCoinbase(testUserAddress)
		})
		uncle = blocks[0]
	}
	return &testWorkerBackend{
		db:         db,
		chain:      chain,
		txPool:     txpool,
		genesis:    gspec,
		uncleBlock: uncle,
	}
}

func (b *testWorkerBackend) BlockChain() *core.BlockChain { return b.chain }
func (b *testWorkerBackend) TxPool() *txpool.TxPool       { return b.txPool }
func (b *testWorkerBackend) StateAtBlock(block *types.Block, reexec uint64, base *state.StateDB, checkLive bool, preferDisk bool) (statedb *state.StateDB, err error) {
	return nil, errors.New("not supported")
}

func (b *testWorkerBackend) newRandomUncle() *types.Block {
	var parent *types.Block
	cur := b.chain.CurrentBlock()
	if cur.NumberU64() == 0 {
		parent = b.chain.Genesis()
	} else {
		parent = b.chain.GetBlockByHash(b.chain.CurrentBlock().ParentHash())
	}
	blocks, _ := core.GenerateChain(b.chain.Config(), parent, b.chain.Engine(), b.db, 1, func(i int, gen *core.BlockGen) {
		var addr = make([]byte, common.AddressLength)
		rand.Read(addr)
		gen.SetCoinbase(common.BytesToAddress(addr))
	})
	return blocks[0]
}

func (b *testWorkerBackend) newRandomTx(creation bool) *types.Transaction {
	var tx *types.Transaction
	gasPrice := big.NewInt(10 * params.InitialBaseFee)
	if creation {
		tx, _ = types.SignTx(types.NewContractCreation(b.txPool.Nonce(testBankAddress), big.NewInt(0), testGas, gasPrice, common.FromHex(testCode)), types.HomesteadSigner{}, testBankKey)
	} else {
		tx, _ = types.SignTx(types.NewTransaction(b.txPool.Nonce(testBankAddress), testUserAddress, big.NewInt(1000), params.TxGas, gasPrice, nil), types.HomesteadSigner{}, testBankKey)
	}
	return tx
}

func newTestWorker(t *testing.T, chainConfig *params.ChainConfig, engine consensus.Engine, db ethdb.Database, blocks int) (*worker, *testWorkerBackend) {
	backend := newTestWorkerBackend(t, chainConfig, engine, db, blocks)
	backend.txPool.AddLocals(pendingTxs)
	w := newWorker(testConfig, chainConfig, engine, backend, new(event.TypeMux), nil, false)
	w.setEtherbase(testBankAddress)
	return w, backend
}

func TestGenerateBlockAndImportEthash(t *testing.T) {
	testGenerateBlockAndImport(t, false)
}

func TestGenerateBlockAndImportClique(t *testing.T) {
	testGenerateBlockAndImport(t, true)
}

func testGenerateBlockAndImport(t *testing.T, isClique bool) {
	var (
		engine      consensus.Engine
		chainConfig params.ChainConfig
		db          = rawdb.NewMemoryDatabase()
	)
	if isClique {
		chainConfig = *params.AllCliqueProtocolChanges
		chainConfig.Clique = &params.CliqueConfig{Period: 1, Epoch: 30000}
		engine = clique.New(chainConfig.Clique, db)
	} else {
		chainConfig = *params.AllEthashProtocolChanges
		engine = ethash.NewFaker()
	}
	w, b := newTestWorker(t, &chainConfig, engine, db, 0)
	defer w.close()

	// This test chain imports the mined blocks.
	chain, _ := core.NewBlockChain(rawdb.NewMemoryDatabase(), nil, b.genesis, nil, engine, vm.Config{}, nil, nil)
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

	for i := 0; i < 5; i++ {
		b.txPool.AddLocal(b.newRandomTx(true))
		b.txPool.AddLocal(b.newRandomTx(false))
		w.postSideBlock(core.ChainSideEvent{Block: b.newRandomUncle()})
		w.postSideBlock(core.ChainSideEvent{Block: b.newRandomUncle()})

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

func TestEmptyWorkEthash(t *testing.T) {
	testEmptyWork(t, ethashChainConfig, ethash.NewFaker())
}
func TestEmptyWorkClique(t *testing.T) {
	testEmptyWork(t, cliqueChainConfig, clique.New(cliqueChainConfig.Clique, rawdb.NewMemoryDatabase()))
}

func testEmptyWork(t *testing.T, chainConfig *params.ChainConfig, engine consensus.Engine) {
	defer engine.Close()

	w, _ := newTestWorker(t, chainConfig, engine, rawdb.NewMemoryDatabase(), 0)
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

	w, b := newTestWorker(t, ethashChainConfig, ethash, rawdb.NewMemoryDatabase(), 1)
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

	w, b := newTestWorker(t, chainConfig, engine, rawdb.NewMemoryDatabase(), 0)
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
	testAdjustInterval(t, ethashChainConfig, ethash.NewFaker())
}

func TestAdjustIntervalClique(t *testing.T) {
	testAdjustInterval(t, cliqueChainConfig, clique.New(cliqueChainConfig.Clique, rawdb.NewMemoryDatabase()))
}

func testAdjustInterval(t *testing.T, chainConfig *params.ChainConfig, engine consensus.Engine) {
	defer engine.Close()

	w, _ := newTestWorker(t, chainConfig, engine, rawdb.NewMemoryDatabase(), 0)
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

	time.Sleep(time.Second) // Ensure two tasks have been submitted due to start opt
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
	testGetSealingWork(t, ethashChainConfig, ethash.NewFaker())
}

func TestGetSealingWorkClique(t *testing.T) {
	testGetSealingWork(t, cliqueChainConfig, clique.New(cliqueChainConfig.Clique, rawdb.NewMemoryDatabase()))
}

func TestGetSealingWorkPostMerge(t *testing.T) {
	local := new(params.ChainConfig)
	*local = *ethashChainConfig
	local.TerminalTotalDifficulty = big.NewInt(0)
	testGetSealingWork(t, local, ethash.NewFaker())
}

func testGetSealingWork(t *testing.T, chainConfig *params.ChainConfig, engine consensus.Engine) {
	defer engine.Close()

	w, b := newTestWorker(t, chainConfig, engine, rawdb.NewMemoryDatabase(), 0)
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
		block, _, err := w.getSealingBlock(c.parent, timestamp, c.coinbase, c.random, false)
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
		block, _, err := w.getSealingBlock(c.parent, timestamp, c.coinbase, c.random, false)
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
