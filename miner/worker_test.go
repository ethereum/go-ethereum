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
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/bor"
	"github.com/ethereum/go-ethereum/consensus/bor/api"
	"github.com/ethereum/go-ethereum/consensus/bor/valset"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/tests/bor/mocks"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/golang/mock/gomock"
	"github.com/holiman/uint256"
	"gotest.tools/assert"
)

// nolint : paralleltest
func TestGenerateBlockAndImportClique(t *testing.T) {
	testGenerateBlockAndImport(t, true, false)
}

// nolint : paralleltest
func TestGenerateBlockAndImportBor(t *testing.T) {
	testGenerateBlockAndImport(t, false, true)
}

//nolint:thelper
func testGenerateBlockAndImport(t *testing.T, isClique bool, isBor bool) {
	var (
		engine      consensus.Engine
		chainConfig params.ChainConfig
		db          = rawdb.NewMemoryDatabase()
		ctrl        *gomock.Controller
	)

	if isBor {
		chainConfig = *params.BorUnittestChainConfig

		engine, ctrl = getFakeBorFromConfig(t, &chainConfig)
		defer ctrl.Finish()
	} else {
		if isClique {
			chainConfig = *params.AllCliqueProtocolChanges
			chainConfig.Clique = &params.CliqueConfig{Period: 1, Epoch: 30000}
			engine = clique.New(chainConfig.Clique, db)
		} else {
			chainConfig = *params.AllEthashProtocolChanges
			engine = ethash.NewFaker()
		}
	}

	defer engine.Close()

	w, b, _ := newTestWorker(t, &chainConfig, engine, db, false, 0, 0)
	defer w.close()

	// This test chain imports the mined blocks.
	chain, _ := core.NewBlockChain(rawdb.NewMemoryDatabase(), nil, b.genesis, nil, engine, vm.Config{}, nil, nil, nil)
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
		err error
	)
	// []*types.Transaction{tx}
	var i uint64
	for i = 0; i < 5; i++ {
		err = b.txPool.Add([]*types.Transaction{b.newRandomTxWithNonce(true, i)}, false)[0]
		if err != nil {
			t.Fatal("while adding a local transaction", err)
		}

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

	for i = 5; i < 10; i++ {
		err = b.txPool.Add([]*types.Transaction{b.newRandomTxWithNonce(false, i)}, false)[0]
		if err != nil {
			t.Fatal("while adding a remote transaction", err)
		}

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

const (
	// testCode is the testing contract binary code which will initialises some
	// variables in constructor
	testCode = "0x60806040527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0060005534801561003457600080fd5b5060fc806100436000396000f3fe6080604052348015600f57600080fd5b506004361060325760003560e01c80630c4dae8814603757806398a213cf146053575b600080fd5b603d607e565b6040518082815260200191505060405180910390f35b607c60048036036020811015606757600080fd5b81019080803590602001909291905050506084565b005b60005481565b806000819055507fe9e44f9f7da8c559de847a3232b57364adc0354f15a2cd8dc636d54396f9587a6000546040518082815260200191505060405180910390a15056fea265627a7a723058208ae31d9424f2d0bc2a3da1a5dd659db2d71ec322a17db8f87e19e209e3a1ff4a64736f6c634300050a0032"

	// testGas is the gas required for contract deployment.
	testGas                   = 144109
	storageContractByteCode   = "608060405234801561001057600080fd5b50610150806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80632e64cec11461003b5780636057361d14610059575b600080fd5b610043610075565b60405161005091906100a1565b60405180910390f35b610073600480360381019061006e91906100ed565b61007e565b005b60008054905090565b8060008190555050565b6000819050919050565b61009b81610088565b82525050565b60006020820190506100b66000830184610092565b92915050565b600080fd5b6100ca81610088565b81146100d557600080fd5b50565b6000813590506100e7816100c1565b92915050565b600060208284031215610103576101026100bc565b5b6000610111848285016100d8565b9150509291505056fea2646970667358221220322c78243e61b783558509c9cc22cb8493dde6925aa5e89a08cdf6e22f279ef164736f6c63430008120033"
	storageContractTxCallData = "0x6057361d0000000000000000000000000000000000000000000000000000000000000001"
	storageCallTxGas          = 100000
)

var (
	// Test chain configurations
	testTxPoolConfig  legacypool.Config
	ethashChainConfig *params.ChainConfig
	cliqueChainConfig *params.ChainConfig

	// Test accounts
	testBankKey, _  = crypto.GenerateKey()
	testBankAddress = crypto.PubkeyToAddress(testBankKey.PublicKey)
	testBankFunds   = big.NewInt(1000000000000000000)
	TestBankAddress = crypto.PubkeyToAddress(testBankKey.PublicKey)

	testUserKey, _  = crypto.GenerateKey()
	testUserAddress = crypto.PubkeyToAddress(testUserKey.PublicKey)

	// Test transactions
	pendingTxs []*types.Transaction
	newTxs     []*types.Transaction

	testConfig = &Config{
		Recommit:            time.Second,
		GasCeil:             params.GenesisGasLimit,
		CommitInterruptFlag: true,
	}
)

func init() {
	testTxPoolConfig = legacypool.DefaultConfig
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
}

// testWorkerBackend implements worker.Backend interfaces and wraps all information needed during the testing.
type testWorkerBackend struct {
	db      ethdb.Database
	txPool  *txpool.TxPool
	chain   *core.BlockChain
	genesis *core.Genesis
}

func newTestWorkerBackend(t TensingObject, chainConfig *params.ChainConfig, engine consensus.Engine, db ethdb.Database) *testWorkerBackend {
	var gspec = &core.Genesis{
		Config: chainConfig,
		Alloc:  types.GenesisAlloc{testBankAddress: {Balance: testBankFunds}},
	}
	switch e := engine.(type) {
	case *bor.Bor:
		gspec.ExtraData = make([]byte, 32+common.AddressLength+crypto.SignatureLength)
		copy(gspec.ExtraData[32:32+common.AddressLength], TestBankAddress.Bytes())
		e.Authorize(TestBankAddress, func(account accounts.Account, s string, data []byte) ([]byte, error) {
			return crypto.Sign(crypto.Keccak256(data), testBankKey)
		})
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
	// genesis := gspec.MustCommit(db)
	chain, err := core.NewBlockChain(db, &core.CacheConfig{TrieDirtyDisabled: true}, gspec, nil, engine, vm.Config{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("core.NewBlockChain failed: %v", err)
	}
	pool := legacypool.New(testTxPoolConfig, chain)
	txpool, _ := txpool.New(testTxPoolConfig.PriceLimit, chain, []txpool.SubPool{pool})

	return &testWorkerBackend{
		db:      db,
		chain:   chain,
		txPool:  txpool,
		genesis: gspec,
	}
}

func (b *testWorkerBackend) BlockChain() *core.BlockChain { return b.chain }
func (b *testWorkerBackend) TxPool() *txpool.TxPool       { return b.txPool }
func (b *testWorkerBackend) PeerCount() int {
	panic("unimplemented")
}

func (b *testWorkerBackend) newRandomTx(creation bool) *types.Transaction {
	var tx *types.Transaction
	gasPrice := big.NewInt(26 * params.InitialBaseFee)
	if creation {
		tx, _ = types.SignTx(types.NewContractCreation(b.txPool.Nonce(testBankAddress), big.NewInt(0), testGas, gasPrice, common.FromHex(testCode)), types.HomesteadSigner{}, testBankKey)
	} else {
		tx, _ = types.SignTx(types.NewTransaction(b.txPool.Nonce(testBankAddress), testUserAddress, big.NewInt(1000), params.TxGas, gasPrice, nil), types.HomesteadSigner{}, testBankKey)
	}
	return tx
}

// newRandomTxWithNonce creates a new transaction with the given nonce.
func (b *testWorkerBackend) newRandomTxWithNonce(creation bool, nonce uint64) *types.Transaction {
	var tx *types.Transaction

	gasPrice := big.NewInt(100 * params.InitialBaseFee)

	if creation {
		tx, _ = types.SignTx(types.NewContractCreation(b.txPool.Nonce(TestBankAddress), big.NewInt(0), testGas, gasPrice, common.FromHex(testCode)), types.HomesteadSigner{}, testBankKey)
	} else {
		tx, _ = types.SignTx(types.NewTransaction(nonce, testUserAddress, big.NewInt(1000), params.TxGas, gasPrice, nil), types.HomesteadSigner{}, testBankKey)
	}

	return tx
}

// newStorageCreateContractTx creates a new transaction to deploy a storage smart contract.
func (b *testWorkerBackend) newStorageCreateContractTx() (*types.Transaction, common.Address) {
	var tx *types.Transaction

	gasPrice := big.NewInt(26 * params.InitialBaseFee)

	tx, _ = types.SignTx(types.NewContractCreation(b.txPool.Nonce(TestBankAddress), big.NewInt(0), testGas, gasPrice, common.FromHex(storageContractByteCode)), types.HomesteadSigner{}, testBankKey)
	contractAddr := crypto.CreateAddress(TestBankAddress, b.txPool.Nonce(TestBankAddress))

	return tx, contractAddr
}

// newStorageContractCallTx creates a new transaction to call a storage smart contract.
func (b *testWorkerBackend) newStorageContractCallTx(to common.Address, nonce uint64) *types.Transaction {
	var tx *types.Transaction

	gasPrice := big.NewInt(26 * params.InitialBaseFee)

	tx, _ = types.SignTx(types.NewTransaction(nonce, to, nil, storageCallTxGas, gasPrice, common.FromHex(storageContractTxCallData)), types.HomesteadSigner{}, testBankKey)

	return tx
}

func newTestWorker(t TensingObject, chainConfig *params.ChainConfig, engine consensus.Engine, db ethdb.Database, noempty bool, delay uint, opcodeDelay uint) (*worker, *testWorkerBackend, func()) {
	backend := newTestWorkerBackend(t, chainConfig, engine, db)
	backend.txPool.Add(pendingTxs, false)
	w := newWorker(testConfig, chainConfig, engine, backend, new(event.TypeMux), nil, false)
	if delay != 0 || opcodeDelay != 0 {
		w.setInterruptCtx(vm.InterruptCtxDelayKey, delay)
		w.setInterruptCtx(vm.InterruptCtxOpcodeDelayKey, opcodeDelay)
	}
	w.setEtherbase(testBankAddress)
	// enable empty blocks
	w.noempty.Store(noempty)
	return w, backend, w.close
}

func TestGenerateAndImportBlock(t *testing.T) {
	t.Parallel()
	var (
		db     = rawdb.NewMemoryDatabase()
		config = *params.AllCliqueProtocolChanges
	)
	config.Clique = &params.CliqueConfig{Period: 1, Epoch: 30000}
	engine := clique.New(config.Clique, db)

	w, b, _ := newTestWorker(t, &config, engine, db, false, 0, 0)
	defer w.close()

	// This test chain imports the mined blocks.
	chain, _ := core.NewBlockChain(rawdb.NewMemoryDatabase(), nil, b.genesis, nil, engine, vm.Config{}, nil, nil, nil)
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
		b.txPool.Add([]*types.Transaction{b.newRandomTx(true)}, false)
		b.txPool.Add([]*types.Transaction{b.newRandomTx(false)}, false)

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
	ethAPIMock.EXPECT().Call(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	// Mock span 0 for heimdall
	span0 := createMockSpanForTest(TestBankAddress, chainConfig.ChainID.String())

	spanner := bor.NewMockSpanner(ctrl)
	spanner.EXPECT().GetCurrentValidatorsByHash(gomock.Any(), gomock.Any(), gomock.Any()).Return(span0.ValidatorSet.Validators, nil).AnyTimes()

	heimdallClientMock := mocks.NewMockIHeimdallClient(ctrl)
	heimdallClientMock.EXPECT().Span(gomock.Any(), uint64(0)).Return(&span0, nil).AnyTimes()
	heimdallClientMock.EXPECT().Close().AnyTimes()

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
	t.Helper()
	defer engine.Close()

	w, _, _ := newTestWorker(t, chainConfig, engine, rawdb.NewMemoryDatabase(), false, 0, 0)
	defer w.close()

	taskCh := make(chan struct{}, 2)
	checkEqual := func(t *testing.T, task *task) {
		// The work should contain 1 tx
		receiptLen, balance := 1, big.NewInt(1000)
		if len(task.receipts) != receiptLen {
			t.Fatalf("receipt number mismatch: have %d, want %d", len(task.receipts), receiptLen)
		}
		if task.state.GetBalance(testUserAddress).Cmp(uint256.NewInt(balance.Uint64())) != 0 {
			t.Fatalf("account balance mismatch: have %d, want %d", task.state.GetBalance(testUserAddress), balance)
		}
	}
	w.newTaskHook = func(task *task) {
		if task.block.NumberU64() == 1 {
			checkEqual(t, task)
			taskCh <- struct{}{}
		}
	}
	w.skipSealHook = func(task *task) bool { return true }
	w.fullTaskHook = func() {
		time.Sleep(100 * time.Millisecond)
	}
	w.start() // Start mining!
	select {
	case <-taskCh:
	case <-time.NewTimer(3 * time.Second).C:
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

	w, _, _ := newTestWorker(t, chainConfig, engine, rawdb.NewMemoryDatabase(), false, 0, 0)
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
		start    atomic.Bool
	)
	w.resubmitHook = func(minInterval time.Duration, recommitInterval time.Duration) {
		// Short circuit if interval checking hasn't started.
		if !start.Load() {
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
	start.Store(true)

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
	t.Parallel()
	testGetSealingWork(t, ethashChainConfig, ethash.NewFaker())
}

func TestGetSealingWorkClique(t *testing.T) {
	t.Parallel()
	testGetSealingWork(t, cliqueChainConfig, clique.New(cliqueChainConfig.Clique, rawdb.NewMemoryDatabase()))
}

func TestGetSealingWorkPostMerge(t *testing.T) {
	t.Parallel()
	local := new(params.ChainConfig)
	*local = *ethashChainConfig
	local.TerminalTotalDifficulty = big.NewInt(0)
	testGetSealingWork(t, local, ethash.NewFaker())
}

// nolint:gocognit
func testGetSealingWork(t *testing.T, chainConfig *params.ChainConfig, engine consensus.Engine) {
	defer engine.Close()

	w, b, _ := newTestWorker(t, chainConfig, engine, rawdb.NewMemoryDatabase(), false, 0, 0)
	defer w.close()

	w.setExtra([]byte{0x01, 0x02})

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
		_, isClique := engine.(*clique.Clique)
		if !isClique {
			if len(block.Extra()) != 2 {
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
			b.chain.CurrentBlock().Number.Uint64() + 1,
			false,
		},
		{
			b.chain.CurrentBlock().Hash(),
			common.Address{},
			common.HexToHash("0xcafebabe"),
			b.chain.CurrentBlock().Number.Uint64() + 1,
			false,
		},
		{
			b.chain.CurrentBlock().Hash(),
			common.Address{},
			common.Hash{},
			b.chain.CurrentBlock().Number.Uint64() + 1,
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
		r := w.getSealingBlock(&generateParams{
			parentHash:  c.parent,
			timestamp:   timestamp,
			coinbase:    c.coinbase,
			random:      c.random,
			withdrawals: nil,
			beaconRoot:  nil,
			noTxs:       false,
			forceTime:   true,
		})
		if c.expectErr {
			if r.err == nil {
				t.Error("Expect error but get nil")
			}
		} else {
			if r.err != nil {
				t.Errorf("Unexpected error %v", r.err)
			}
			assertBlock(r.block, c.expectNumber, c.coinbase, c.random)
		}
	}

	// This API should work even when the automatic sealing is enabled
	w.start()
	for _, c := range cases {
		r := w.getSealingBlock(&generateParams{
			parentHash:  c.parent,
			timestamp:   timestamp,
			coinbase:    c.coinbase,
			random:      c.random,
			withdrawals: nil,
			beaconRoot:  nil,
			noTxs:       false,
			forceTime:   true,
		})
		if c.expectErr {
			if r.err == nil {
				t.Error("Expect error but get nil")
			}
		} else {
			if r.err != nil {
				t.Errorf("Unexpected error %v", r.err)
			}
			assertBlock(r.block, c.expectNumber, c.coinbase, c.random)
		}
	}
}

// nolint: paralleltest
// TestCommitInterruptExperimentBor_NormalFlow tests the commit interrupt experiment for bor consensus by inducing
// an artificial delay at transaction level. It runs the normal mining flow triggered via new head.
func TestCommitInterruptExperimentBor_NormalFlow(t *testing.T) {
	// with 1 sec block time and 200 millisec tx delay we should get 5 txs per block
	testCommitInterruptExperimentBor(t, 200, 5, 0)

	time.Sleep(2 * time.Second)

	// with 1 sec block time and 100 millisec tx delay we should get 10 txs per block
	testCommitInterruptExperimentBor(t, 100, 10, 0)
}

// nolint : paralleltest
// TestCommitInterruptExperimentBorContract tests the commit interrupt experiment for bor consensus by inducing an artificial delay at OPCODE level.
func TestCommitInterruptExperimentBorContract(t *testing.T) {
	// pre-calculated number of OPCODES = 123. 7*123=861 < 1000, 1 tx is possible but 2 tx per block will not be possible.
	testCommitInterruptExperimentBorContract(t, 0, 1, 7)
	time.Sleep(2 * time.Second)
	// pre-calculated number of OPCODES = 123. 2*123=246 < 1000, 4 tx is possible but 5 tx per block will not be possible. But 3 happen due to other overheads.
	testCommitInterruptExperimentBorContract(t, 0, 3, 2)
	time.Sleep(2 * time.Second)
	// pre-calculated number of OPCODES = 123. 3*123=369 < 1000, 2 tx is possible but 3 tx per block will not be possible.
	testCommitInterruptExperimentBorContract(t, 0, 2, 3)
}

// nolint : thelper
// testCommitInterruptExperimentBorContract is a helper function for testing the commit interrupt experiment for bor consensus.
func testCommitInterruptExperimentBorContract(t *testing.T, delay uint, txCount int, opcodeDelay uint) {
	var (
		engine      consensus.Engine
		chainConfig *params.ChainConfig
		txInTxpool  = 100
		txs         = make([]*types.Transaction, 0, txInTxpool)
	)

	chainConfig = params.BorUnittestChainConfig

	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, true)))

	engine, _ = getFakeBorFromConfig(t, chainConfig)

	w, b, _ := newTestWorker(t, chainConfig, engine, rawdb.NewMemoryDatabase(), true, delay, opcodeDelay)
	defer w.close()

	// nonce 0 tx
	tx, addr := b.newStorageCreateContractTx()
	if err := b.txPool.Add([]*types.Transaction{tx}, false)[0]; err != nil {
		t.Fatal(err)
	}

	time.Sleep(4 * time.Second)

	// nonce starts from 1 because we already have one tx
	initNonce := uint64(1)

	for i := 0; i < txInTxpool; i++ {
		tx := b.newStorageContractCallTx(addr, initNonce+uint64(i))
		txs = append(txs, tx)
	}

	wrapped := make([]*types.Transaction, len(txs))
	copy(wrapped, txs)

	b.TxPool().Add(wrapped, false)

	// Start mining!
	w.start()
	time.Sleep(5 * time.Second)
	w.stop()

	currentBlockNumber := w.current.header.Number.Uint64()
	prevBlockTxCount := w.chain.GetBlockByNumber(currentBlockNumber - 1).Transactions().Len()
	assert.Check(t, prevBlockTxCount > 0)
	assert.Check(t, prevBlockTxCount <= txCount)
}

// // nolint : thelper
// testCommitInterruptExperimentBor is a helper function for testing the commit interrupt experiment for bor consensus.
func testCommitInterruptExperimentBor(t *testing.T, delay uint, txCount int, opcodeDelay uint) {
	var (
		engine      consensus.Engine
		chainConfig *params.ChainConfig
		db          = rawdb.NewMemoryDatabase()
		ctrl        *gomock.Controller
		txInTxpool  = 100
		txs         = make([]*types.Transaction, 0, txInTxpool)
	)

	chainConfig = params.BorUnittestChainConfig

	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, true)))

	engine, ctrl = getFakeBorFromConfig(t, chainConfig)

	w, b, _ := newTestWorker(t, chainConfig, engine, rawdb.NewMemoryDatabase(), true, delay, opcodeDelay)
	defer func() {
		w.close()
		engine.Close()
		db.Close()
		ctrl.Finish()
	}()

	// nonce starts from 0 because have no txs yet
	initNonce := uint64(0)

	for i := 0; i < txInTxpool; i++ {
		tx := b.newRandomTxWithNonce(false, initNonce+uint64(i))
		txs = append(txs, tx)
	}

	wrapped := make([]*types.Transaction, len(txs))
	for i, tx := range txs {
		wrapped[i] = tx
	}

	b.TxPool().Add(wrapped, false)

	// Start mining!
	w.start()
	time.Sleep(5 * time.Second)
	w.stop()

	currentBlockNumber := w.current.header.Number.Uint64()
	assert.Check(t, txCount >= w.chain.GetBlockByNumber(currentBlockNumber-1).Transactions().Len())
	assert.Check(t, 0 < w.chain.GetBlockByNumber(currentBlockNumber-1).Transactions().Len())
}

// TestCommitInterruptExperimentBor_NewTxFlow tests the commit interrupt experiment for bor consensus by inducing
// an artificial delay at transaction level. It runs the mining flow triggered via new transactions channel. The tests
// are a bit unconventional compared to normal flow as the situations are only possible in non-validator mode.
func TestCommitInterruptExperimentBor_NewTxFlow(t *testing.T) {
	var (
		engine      consensus.Engine
		chainConfig *params.ChainConfig
		db          = rawdb.NewMemoryDatabase()
		ctrl        *gomock.Controller
	)

	chainConfig = params.BorUnittestChainConfig

	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, true)))

	engine, ctrl = getFakeBorFromConfig(t, chainConfig)

	w, b, _ := newTestWorker(t, chainConfig, engine, rawdb.NewMemoryDatabase(), true, uint(0), uint(0))
	defer func() {
		w.close()
		engine.Close()
		db.Close()
		ctrl.Finish()
	}()

	// Create random transactions (contract interaction)
	tx1, addr := b.newStorageCreateContractTx()
	tx2 := b.newStorageContractCallTx(addr, 1)
	tx3 := b.newStorageContractCallTx(addr, 2)

	// Create a chain head subscription for tests
	chainHeadCh := make(chan core.ChainHeadEvent, 10)
	w.chain.SubscribeChainHeadEvent(chainHeadCh)

	// Start mining!
	w.start()
	go func() {
		for {
			head := <-chainHeadCh
			// We skip the initial 2 blocks as the mining timings are a bit skewed up
			if head.Header.Number.Uint64() == 2 {
				// Wait until `w.current` is updated for next block (3)
				time.Sleep(100 * time.Millisecond)

				// Stop the miner so that worker assumes it's a sentry and not a validator
				w.stop()

				// Add the first transaction to be mined normally via `txsCh`
				b.TxPool().Add([]*types.Transaction{tx1}, false)

				// Set it to syncing mode so that it doesn't mine via the `commitWork` flow
				w.syncing.Store(true)

				// Wait until the mining window (2s) is almost about to reach leaving
				// a very small time (~10ms) to try to commit transaction before timing out.
				delay := time.Until(time.Unix(int64(w.current.header.Time), 0))
				delay -= 10 * time.Millisecond

				// Case 1: This transaction should not be included due to commit interrupt
				// at opcode level. It will start the EVM execution but will end in between.
				<-time.After(delay)

				// Set an artificial delay at opcode level
				w.setInterruptCtx(vm.InterruptCtxOpcodeDelayKey, uint(500))

				// Send the second transaction
				b.TxPool().Add([]*types.Transaction{tx2}, false)

				// Reset the delay again. By this time, we're sure that it has timed out.
				delay = time.Until(time.Unix(int64(w.current.header.Time), 0))

				// Case 2: This transaction should not be included because the miner loop
				// won't accept any transactions post the deadline (i.e. header.Timestamp).
				<-time.After(delay)

				// Reset the artificial opcode delay just to be sure of the exclusion of tx
				w.setInterruptCtx(vm.InterruptCtxOpcodeDelayKey, uint(0))

				// Send the third transaction
				b.TxPool().Add([]*types.Transaction{tx3}, false)
			}
		}
	}()

	// Wait for enough time to mine 3 blocks
	time.Sleep(6 * time.Second)

	// Ensure that the last block was 3 and only 1 transactions out of 3 were included
	assert.Equal(t, w.current.header.Number.Uint64(), uint64(3))
	assert.Equal(t, w.current.tcount, 1)
	assert.Equal(t, len(w.current.txs), 1)
}

func BenchmarkBorMining(b *testing.B) {
	chainConfig := params.BorUnittestChainConfig

	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	ethAPIMock := api.NewMockCaller(ctrl)
	ethAPIMock.EXPECT().Call(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	spanner := bor.NewMockSpanner(ctrl)
	spanner.EXPECT().GetCurrentValidatorsByHash(gomock.Any(), gomock.Any(), gomock.Any()).Return([]*valset.Validator{
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

	w, back, _ := newTestWorker(b, chainConfig, engine, rawdb.NewMemoryDatabase(), false, 0, 0)
	defer w.close()

	chain, _ := core.NewBlockChain(rawdb.NewMemoryDatabase(), nil, back.genesis, nil, engine, vm.Config{}, nil, nil, nil)
	defer chain.Stop()

	// fulfill tx pool
	const (
		totalGas    = testGas + params.TxGas
		totalBlocks = 10
	)

	var err error

	txInBlock := int(back.genesis.GasLimit/totalGas) + 1

	// a bit risky
	for i := 0; i < 2*totalBlocks*txInBlock; i++ {
		err = back.txPool.Add([]*types.Transaction{back.newRandomTx(true)}, false)[0]
		if err != nil {
			b.Fatal("while adding a local transaction", err)
		}

		err = back.txPool.Add([]*types.Transaction{back.newRandomTx(false)}, false)[0]
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

	blockPeriod, ok := back.genesis.Config.Bor.Period["0"]
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

// uses core.NewParallelBlockChain to use the dependencies present in the block header
// params.BorUnittestChainConfig contains the NapoliBlock as big.NewInt(5), so the first 4 blocks will not have metadata.
// nolint: gocognit
func BenchmarkBorMiningBlockSTMMetadata(b *testing.B) {
	chainConfig := params.BorUnittestChainConfig

	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	ethAPIMock := api.NewMockCaller(ctrl)
	ethAPIMock.EXPECT().Call(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	spanner := bor.NewMockSpanner(ctrl)
	spanner.EXPECT().GetCurrentValidatorsByHash(gomock.Any(), gomock.Any(), gomock.Any()).Return([]*valset.Validator{
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

	w, back, _ := newTestWorker(b, chainConfig, engine, rawdb.NewMemoryDatabase(), false, 0, 0)
	defer w.close()

	// This test chain imports the mined blocks.
	db2 := rawdb.NewMemoryDatabase()
	back.genesis.MustCommit(db2, triedb.NewDatabase(db2, triedb.HashDefaults))

	chain, _ := core.NewParallelBlockChain(db2, nil, back.genesis, nil, engine, vm.Config{}, nil, nil, nil, 8, false)
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

	txInBlock := int(back.genesis.GasLimit/totalGas) + 1

	// a bit risky
	for i := 0; i < 2*totalBlocks*txInBlock; i++ {
		err = back.txPool.Add([]*types.Transaction{back.newRandomTx(true)}, false)[0]
		if err != nil {
			b.Fatal("while adding a local transaction", err)
		}

		err = back.txPool.Add([]*types.Transaction{back.newRandomTx(false)}, false)[0]
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

	blockPeriod, ok := back.genesis.Config.Bor.Period["0"]
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

			// check for dependencies for block number > 4
			if block.NumberU64() <= 4 {
				if block.GetTxDependency() != nil {
					b.Fatalf("dependency not nil")
				}
			} else {
				deps := block.GetTxDependency()
				if len(deps[0]) != 0 {
					b.Fatalf("wrong dependency")
				}

				for i := 1; i < block.Transactions().Len(); i++ {
					if deps[i][0] != uint64(i-1) || len(deps[i]) != 1 {
						b.Fatalf("wrong dependency")
					}
				}
			}

			b.Log("block", block.NumberU64(), "time", block.Time()-prev, "txs", block.Transactions().Len(), "gasUsed", block.GasUsed(), "gasLimit", block.GasLimit())

			prev = block.Time()
		case <-time.After(time.Duration(blockPeriod) * time.Second):
			b.Fatalf("timeout")
		}
	}
}
