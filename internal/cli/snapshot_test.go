package cli

import (
	"bytes"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state/pruner"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/cli/server"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/stretchr/testify/require"
)

var (
	canonicalSeed               = 1
	blockPruneBackUpBlockNumber = 128
	key, _                      = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	address                     = crypto.PubkeyToAddress(key.PublicKey)
	balance                     = big.NewInt(1_000000000000000000)
	gspec                       = &core.Genesis{Config: params.TestChainConfig, Alloc: core.GenesisAlloc{address: {Balance: balance}}}
	signer                      = types.LatestSigner(gspec.Config)
	config                      = &core.CacheConfig{
		TrieCleanLimit: 256,
		TrieDirtyLimit: 256,
		TrieTimeLimit:  5 * time.Minute,
		SnapshotLimit:  0, // Disable snapshot
		TriesInMemory:  128,
	}
	engine = ethash.NewFullFaker()
)

func TestOfflineBlockPrune(t *testing.T) {
	t.Parallel()

	// Corner case for 0 remain in ancinetStore.
	testOfflineBlockPruneWithAmountReserved(t, 0)

	// General case.
	testOfflineBlockPruneWithAmountReserved(t, 100)
}

func testOfflineBlockPruneWithAmountReserved(t *testing.T, amountReserved uint64) {
	t.Helper()

	datadir, err := os.MkdirTemp("", "")
	require.NoError(t, err, "failed to create temporary datadir")

	os.RemoveAll(datadir)

	chaindbPath := filepath.Join(datadir, "chaindata")
	oldAncientPath := filepath.Join(chaindbPath, "ancient")
	newAncientPath := filepath.Join(chaindbPath, "ancient_back")

	_, _, blockList, receiptsList, externTdList, startBlockNumber, _ := BlockchainCreator(t, chaindbPath, oldAncientPath, amountReserved)

	node := startEthService(t, chaindbPath)
	defer node.Close()

	// Initialize a block pruner for pruning, only remain amountReserved blocks backward.
	testBlockPruner := pruner.NewBlockPruner(node, oldAncientPath, newAncientPath, amountReserved)
	dbHandles, err := server.MakeDatabaseHandles(0)
	require.NoError(t, err, "failed to create database handles")

	err = testBlockPruner.BlockPruneBackup(chaindbPath, 512, dbHandles, "", false, false)
	require.NoError(t, err, "failed to backup block")

	dbBack, err := rawdb.Open(rawdb.OpenOptions{
		Type:              node.Config().DBEngine,
		Directory:         chaindbPath,
		AncientsDirectory: newAncientPath,
		Namespace:         "",
		Cache:             0,
		Handles:           0,
		ReadOnly:          false,
		DisableFreeze:     true,
		IsLastOffset:      false,
	})
	require.NoError(t, err, "failed to create db with ancient backend")

	defer dbBack.Close()

	// Check the absence of genesis
	genesis, err := dbBack.Ancient("hashes", 0)
	require.Equal(t, []byte(nil), genesis, "got genesis but should be absent")
	require.NotNil(t, err, "not-nill error expected")

	// Check against if the backup data matched original one
	for blockNumber := startBlockNumber; blockNumber < startBlockNumber+amountReserved; blockNumber++ {
		// Fetch the data explicitly from ancient db instead of `ReadCanonicalHash` because it
		// will pull data from leveldb if not found in ancient.
		blockHash, err := dbBack.Ancient("hashes", blockNumber)
		require.NoError(t, err, "error fetching block hash from ancient db")

		// We can proceed with fetching other things via generic functions because if
		// the block wouldn't have been there in ancient db, the function above to get
		// block hash itself would've thrown error.
		hash := common.BytesToHash(blockHash)
		block := rawdb.ReadBlock(dbBack, hash, blockNumber)

		require.Equal(t, block.Hash(), hash, "block data mismatch between oldDb and backupDb")
		require.Equal(t, blockList[blockNumber-startBlockNumber].Hash(), hash, "block data mismatch between oldDb and backupDb")

		receipts := rawdb.ReadRawReceipts(dbBack, hash, blockNumber)
		checkReceiptsRLP(t, receipts, receiptsList[blockNumber-startBlockNumber])

		// Calculate the total difficulty of the block
		td := rawdb.ReadTd(dbBack, hash, blockNumber)
		require.NotNil(t, td, "failed to read td", consensus.ErrUnknownAncestor)

		require.Equal(t, td.Cmp(externTdList[blockNumber-startBlockNumber]), 0, "Td mismatch between oldDb and backupDb")
	}

	// Check if ancientDb freezer replaced successfully
	err = testBlockPruner.AncientDbReplacer()
	require.NoError(t, err, "error replacing ancient db")

	if _, err := os.Stat(newAncientPath); err != nil {
		if !os.IsNotExist(err) {
			t.Fatalf("ancientDb replaced unsuccessfully")
		}
	}

	_, err = os.Stat(oldAncientPath)
	require.NoError(t, err, "failed to replace ancientDb")
}

func BlockchainCreator(t *testing.T, chaindbPath, AncientPath string, blockRemain uint64) (ethdb.Database, []*types.Block, []*types.Block, []types.Receipts, []*big.Int, uint64, *core.BlockChain) {
	t.Helper()

	// Create a database with ancient freezer
	db, err := rawdb.Open(rawdb.OpenOptions{
		Directory:         chaindbPath,
		AncientsDirectory: AncientPath,
		Namespace:         "",
		Cache:             0,
		Handles:           0,
		ReadOnly:          false,
		DisableFreeze:     false,
		IsLastOffset:      false,
	})
	require.NoError(t, err, "failed to create db with ancient backend")

	defer db.Close()

	genesis := gspec.MustCommit(db, trie.NewDatabase(db, trie.HashDefaults))
	// Initialize a fresh chain with only a genesis block
	blockchain, err := core.NewBlockChain(db, config, gspec, nil, engine, vm.Config{}, nil, nil, nil)
	require.NoError(t, err, "failed to create chain")

	// Make chain starting from genesis
	blocks, _ := core.GenerateChain(gspec.Config, genesis, ethash.NewFaker(), db, 500, func(i int, block *core.BlockGen) {
		block.SetCoinbase(common.Address{0: byte(canonicalSeed), 19: byte(i)})
		tx, err := types.SignTx(types.NewTransaction(block.TxNonce(address), common.Address{0x00}, big.NewInt(1), params.TxGas, big.NewInt(8750000000), nil), signer, key)
		if err != nil {
			require.NoError(t, err, "failed to sign tx")
		}
		block.AddTx(tx)
		block.SetDifficulty(big.NewInt(1000000))
	})

	_, err = blockchain.InsertChain(blocks)
	require.NoError(t, err, "failed to insert chain")

	// Force run a freeze cycle
	type freezer interface {
		Freeze(threshold uint64) error
		Ancients() (uint64, error)
	}

	err = db.(freezer).Freeze(10)
	require.NoError(t, err, "failed to perform freeze operation")

	// make sure there're frozen items
	frozen, err := db.Ancients()
	require.NoError(t, err, "failed to fetch ancients items from db")
	require.NotEqual(t, frozen, 0, "no elements in freezer db")
	require.GreaterOrEqual(t, frozen, blockRemain, "block amount is not enough for pruning")

	oldOffSet := rawdb.ReadOffsetOfCurrentAncientFreezer(db)
	// Get the actual start block number.
	startBlockNumber := frozen - blockRemain + oldOffSet
	// Initialize the slice to buffer the block data left.
	blockList := make([]*types.Block, 0, blockPruneBackUpBlockNumber)
	receiptsList := make([]types.Receipts, 0, blockPruneBackUpBlockNumber)
	externTdList := make([]*big.Int, 0, blockPruneBackUpBlockNumber)
	// All ancient data within the most recent 128 blocks write into memory buffer for future new ancient_back directory usage.
	for blockNumber := startBlockNumber; blockNumber < frozen+oldOffSet; blockNumber++ {
		blockHash := rawdb.ReadCanonicalHash(db, blockNumber)
		block := rawdb.ReadBlock(db, blockHash, blockNumber)
		blockList = append(blockList, block)
		receipts := rawdb.ReadRawReceipts(db, blockHash, blockNumber)
		receiptsList = append(receiptsList, receipts)
		// Calculate the total difficulty of the block
		td := rawdb.ReadTd(db, blockHash, blockNumber)
		require.NotNil(t, td, "failed to read td", consensus.ErrUnknownAncestor)

		externTdList = append(externTdList, td)
	}

	return db, blocks, blockList, receiptsList, externTdList, startBlockNumber, blockchain
}

func checkReceiptsRLP(t *testing.T, have, want types.Receipts) {
	t.Helper()

	require.Equal(t, len(want), len(have), "receipts sizes mismatch")

	for i := 0; i < len(want); i++ {
		rlpHave, err := rlp.EncodeToBytes(have[i])
		require.NoError(t, err, "error in rlp encoding")

		rlpWant, err := rlp.EncodeToBytes(want[i])
		require.NoError(t, err, "error in rlp encoding")

		require.Equal(t, true, bytes.Equal(rlpHave, rlpWant), "receipt rlp mismatch")
	}
}

// startEthService creates a full node instance for testing.
func startEthService(t *testing.T, chaindbPath string) *node.Node {
	t.Helper()

	n, err := node.New(&node.Config{DataDir: chaindbPath})
	require.NoError(t, err, "failed to create node")

	err = n.Start()
	require.NoError(t, err, "failed to start node")

	return n
}
