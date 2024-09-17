package ccc

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/consensus/ethash"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/trie"
)

func TestAsyncChecker(t *testing.T) {
	// testKey is a private key to use for funding a tester account.
	testKey, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	// testAddr is the Ethereum address of the tester account.
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)

	// Create a database pre-initialize with a genesis block
	db := rawdb.NewMemoryDatabase()
	gspec := &core.Genesis{
		Config: params.TestChainConfig,
		Alloc:  core.GenesisAlloc{testAddr: {Balance: new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether))}},
	}
	gspec.MustCommit(db, trie.NewDatabase(db, trie.HashDefaultsWithZktrie))

	chain, _ := core.NewBlockChain(db, nil, gspec, nil, ethash.NewFaker(), vm.Config{}, nil, nil)
	asyncChecker := NewAsyncChecker(chain, 1, false)
	chain.Validator().WithAsyncValidator(asyncChecker.Check)

	bs, _ := core.GenerateChain(params.TestChainConfig, chain.Genesis(), ethash.NewFaker(), db, 100, func(i int, block *core.BlockGen) {
		for i := 0; i < 10; i++ {
			signer := types.MakeSigner(params.TestChainConfig, block.Number(), block.Timestamp())
			tx, err := types.SignTx(types.NewTransaction(block.TxNonce(testAddr), testAddr, big.NewInt(1000), params.TxGas, block.BaseFee(), nil), signer, testKey)
			if err != nil {
				panic(err)
			}
			block.AddTx(tx)
		}
	})

	noReorgBlocks := bs[:len(bs)/2]
	if _, err := chain.InsertChain(noReorgBlocks); err != nil {
		panic(err)
	}
	time.Sleep(time.Second)
	for _, block := range noReorgBlocks {
		require.NotNil(t, rawdb.ReadBlockRowConsumption(db, block.Hash()))
	}

	reorgBlocks := bs[len(bs)/2:]
	skippedTxn := reorgBlocks[3].Transactions()[3]
	checker := <-asyncChecker.freeCheckers
	checker.Skip(skippedTxn.Hash(), ErrBlockRowConsumptionOverflow)
	// trigger an error on some later height, we shouldn't get a notification for this
	checker.ScheduleError(50, ErrBlockRowConsumptionOverflow)

	asyncChecker.freeCheckers <- checker

	var failingBlockHash common.Hash
	var errWithIdx *ErrorWithTxnIdx
	asyncChecker.WithOnFailingBlock(func(b *types.Block, err error) {
		failingBlockHash = b.Hash()
		require.ErrorAs(t, err, &errWithIdx)
	})

	if _, err := chain.InsertChain(reorgBlocks); err != nil {
		panic(err)
	}

	time.Sleep(3 * time.Second)
	require.Equal(t, reorgBlocks[3].Hash(), failingBlockHash)
	require.Equal(t, uint(3), errWithIdx.TxIdx)
	require.True(t, errWithIdx.ShouldSkip)
}
