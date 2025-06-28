package miner

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// testMinerBackend implements miner's Backend interface.
type testMinerBackend struct {
	txPool *txpool.TxPool
	chain  *core.BlockChain
}

func (b *testMinerBackend) BlockChain() *core.BlockChain { return b.chain }
func (b *testMinerBackend) TxPool() *txpool.TxPool       { return b.txPool }

func newTestMinerBackend(t *testing.T, genesis *core.Genesis, engine consensus.Engine) *testMinerBackend {
	chain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), &core.CacheConfig{TrieDirtyDisabled: true}, genesis, nil, engine, vm.Config{}, nil)
	if err != nil {
		t.Fatalf("core.NewBlockChain failed: %v", err)
	}

	testTxPoolConfig := legacypool.DefaultConfig
	pool := legacypool.New(testTxPoolConfig, chain)
	txpool, err := txpool.New(testTxPoolConfig.PriceLimit, chain, []txpool.SubPool{pool})
	if err != nil {
		t.Fatalf("txpool.New failed: %v", err)
	}

	return &testMinerBackend{
		chain:  chain,
		txPool: txpool,
	}
}

func newTestMiner(eth *testMinerBackend) *Miner {
	return New(eth, DefaultConfig, eth.chain.Engine())
}

func newPendingTransactions(fromPrivateKey *ecdsa.PrivateKey, toAddress common.Address) []*types.Transaction {
	signer := types.LatestSigner(params.TestChainConfig)

	tx1 := types.MustSignNewTx(fromPrivateKey, signer, &types.AccessListTx{
		ChainID:  params.TestChainConfig.ChainID,
		Nonce:    0,
		To:       &toAddress,
		Value:    big.NewInt(1000),
		Gas:      params.TxGas,
		GasPrice: big.NewInt(params.InitialBaseFee),
	})
	pendingTxs = append(pendingTxs, tx1)

	tx2 := types.MustSignNewTx(fromPrivateKey, signer, &types.LegacyTx{
		Nonce:    1,
		To:       &toAddress,
		Value:    big.NewInt(1000),
		Gas:      params.TxGas,
		GasPrice: big.NewInt(params.InitialBaseFee),
	})
	pendingTxs = append(pendingTxs, tx2)

	return pendingTxs
}

func TestBuildInclusionList(t *testing.T) {
	var (
		// Test accounts
		testBankKey, _  = crypto.GenerateKey()
		testBankAddress = crypto.PubkeyToAddress(testBankKey.PublicKey)
		testBankFunds   = big.NewInt(1000000000000000000)

		testUserKey, _  = crypto.GenerateKey()
		testUserAddress = crypto.PubkeyToAddress(testUserKey.PublicKey)

		// Test genesis and consensus engine
		testGenesis = &core.Genesis{
			Config: params.TestChainConfig,
			Alloc:  types.GenesisAlloc{testBankAddress: {Balance: testBankFunds}},
		}
		testEngine = ethash.NewFaker()
	)

	// Initialize miner and backend.
	eth := newTestMinerBackend(t, testGenesis, testEngine)
	miner := newTestMiner(eth)

	// Add pending transactions to the pool.
	pendingTxs := newPendingTransactions(testBankKey, testUserAddress)
	eth.txPool.Add(pendingTxs, true)

	// Build inclusion list.
	args := &BuildInclusionListArgs{
		Parent: eth.chain.CurrentBlock().Hash(),
	}
	inclusionList, err := miner.BuildInclusionList(args)
	if err != nil {
		t.Fatalf("Failed to build inclusion list %v", err)
	}

	// Verify inclusion list size.
	inclusionListSize := uint64(0)
	for _, tx := range inclusionList {
		inclusionListSize += uint64(len(tx))
	}
	require.LessOrEqual(t, inclusionListSize, params.MaxBytesPerInclusionList)
}
