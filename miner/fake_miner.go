package miner

import (
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/bor"
	"github.com/ethereum/go-ethereum/consensus/bor/api"
	"github.com/ethereum/go-ethereum/consensus/bor/valset"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/tests/bor/mocks"
	"github.com/ethereum/go-ethereum/trie"
)

type DefaultBorMiner struct {
	Miner   *Miner
	Mux     *event.TypeMux //nolint:staticcheck
	Cleanup func(skipMiner bool)

	Ctrl               *gomock.Controller
	EthAPIMock         api.Caller
	HeimdallClientMock bor.IHeimdallClient
	ContractMock       bor.GenesisContract
}

func NewBorDefaultMiner(t *testing.T) *DefaultBorMiner {
	t.Helper()

	ctrl := gomock.NewController(t)

	ethAPI := api.NewMockCaller(ctrl)
	ethAPI.EXPECT().Call(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	spanner := bor.NewMockSpanner(ctrl)
	spanner.EXPECT().GetCurrentValidators(gomock.Any(), gomock.Any(), gomock.Any()).Return([]*valset.Validator{
		{
			ID:               0,
			Address:          common.Address{0x1},
			VotingPower:      100,
			ProposerPriority: 0,
		},
	}, nil).AnyTimes()

	heimdallClient := mocks.NewMockIHeimdallClient(ctrl)
	heimdallClient.EXPECT().Close().Times(1)

	genesisContracts := bor.NewMockGenesisContract(ctrl)

	miner, mux, cleanup := createBorMiner(t, ethAPI, spanner, heimdallClient, genesisContracts)

	return &DefaultBorMiner{
		Miner:              miner,
		Mux:                mux,
		Cleanup:            cleanup,
		Ctrl:               ctrl,
		EthAPIMock:         ethAPI,
		HeimdallClientMock: heimdallClient,
		ContractMock:       genesisContracts,
	}
}

//nolint:staticcheck
func createBorMiner(t *testing.T, ethAPIMock api.Caller, spanner bor.Spanner, heimdallClientMock bor.IHeimdallClient, contractMock bor.GenesisContract) (*Miner, *event.TypeMux, func(skipMiner bool)) {
	t.Helper()

	// Create Ethash config
	chainDB, _, chainConfig := NewDBForFakes(t)

	engine := NewFakeBor(t, chainDB, chainConfig, ethAPIMock, spanner, heimdallClientMock, contractMock)

	// Create Ethereum backend
	bc, err := core.NewBlockChain(chainDB, nil, chainConfig, engine, vm.Config{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("can't create new chain %v", err)
	}

	statedb, _ := state.New(common.Hash{}, state.NewDatabase(chainDB), nil)
	blockchain := &testBlockChain{statedb, 10000000, new(event.Feed)}

	pool := core.NewTxPool(testTxPoolConfig, chainConfig, blockchain)
	backend := NewMockBackend(bc, pool)

	// Create event Mux
	mux := new(event.TypeMux)

	config := Config{
		Etherbase: common.HexToAddress("123456789"),
	}

	// Create Miner
	miner := New(backend, &config, chainConfig, mux, engine, nil)

	cleanup := func(skipMiner bool) {
		bc.Stop()
		engine.Close()
		pool.Stop()

		if !skipMiner {
			miner.Close()
		}
	}

	return miner, mux, cleanup
}

type TensingObject interface {
	Helper()
	Fatalf(format string, args ...any)
}

func NewDBForFakes(t TensingObject) (ethdb.Database, *core.Genesis, *params.ChainConfig) {
	t.Helper()

	memdb := memorydb.New()
	chainDB := rawdb.NewDatabase(memdb)
	genesis := core.DeveloperGenesisBlock(2, 11_500_000, common.HexToAddress("12345"))

	chainConfig, _, err := core.SetupGenesisBlock(chainDB, genesis)
	if err != nil {
		t.Fatalf("can't create new chain config: %v", err)
	}

	chainConfig.Bor.Period = map[string]uint64{
		"0": 1,
	}
	chainConfig.Bor.Sprint = map[string]uint64{
		"0": 64,
	}

	return chainDB, genesis, chainConfig
}

func NewFakeBor(t TensingObject, chainDB ethdb.Database, chainConfig *params.ChainConfig, ethAPIMock api.Caller, spanner bor.Spanner, heimdallClientMock bor.IHeimdallClient, contractMock bor.GenesisContract) consensus.Engine {
	t.Helper()

	if chainConfig.Bor == nil {
		chainConfig.Bor = params.BorUnittestChainConfig.Bor
	}

	return bor.New(chainConfig, chainDB, ethAPIMock, spanner, heimdallClientMock, contractMock, false)
}

type mockBackend struct {
	bc     *core.BlockChain
	txPool *core.TxPool
}

func NewMockBackend(bc *core.BlockChain, txPool *core.TxPool) *mockBackend {
	return &mockBackend{
		bc:     bc,
		txPool: txPool,
	}
}

func (m *mockBackend) BlockChain() *core.BlockChain {
	return m.bc
}

func (m *mockBackend) TxPool() *core.TxPool {
	return m.txPool
}

func (m *mockBackend) StateAtBlock(block *types.Block, reexec uint64, base *state.StateDB, checkLive bool, preferDisk bool) (statedb *state.StateDB, err error) {
	return nil, errors.New("not supported")
}

type testBlockChain struct {
	statedb       *state.StateDB
	gasLimit      uint64
	chainHeadFeed *event.Feed
}

func (bc *testBlockChain) CurrentBlock() *types.Block {
	return types.NewBlock(&types.Header{
		GasLimit: bc.gasLimit,
	}, nil, nil, nil, trie.NewStackTrie(nil))
}

func (bc *testBlockChain) GetBlock(hash common.Hash, number uint64) *types.Block {
	return bc.CurrentBlock()
}

func (bc *testBlockChain) StateAt(common.Hash) (*state.StateDB, error) {
	return bc.statedb, nil
}

func (bc *testBlockChain) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return bc.chainHeadFeed.Subscribe(ch)
}

var (
	// Test chain configurations
	testTxPoolConfig  core.TxPoolConfig
	ethashChainConfig *params.ChainConfig
	cliqueChainConfig *params.ChainConfig

	// Test accounts
	testBankKey, _  = crypto.GenerateKey()
	TestBankAddress = crypto.PubkeyToAddress(testBankKey.PublicKey)
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
	testTxPoolConfig = core.DefaultTxPoolConfig
	testTxPoolConfig.Journal = ""
	ethashChainConfig = new(params.ChainConfig)
	*ethashChainConfig = *params.TestChainConfig
	cliqueChainConfig = new(params.ChainConfig)
	*cliqueChainConfig = *params.TestChainConfig
	cliqueChainConfig.Clique = &params.CliqueConfig{
		Period: 10,
		Epoch:  30000,
	}
}
