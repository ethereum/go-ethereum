package core

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
)

// Define testKey for signing transactions
var testKey, _ = crypto.HexToECDSA("45a915e4d060149eb4365960e6a7a45f334393093061116b197e3240065ff2d8")

// TestVerkleTransitionWithReorg tests the mapping during a reorg at the verkle transition
func TestVerkleTransitionWithReorg(t *testing.T) {
	// Get address from the test key for funding
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)

	// Configure Verkle transition at block 10
	var (
		db     = rawdb.NewMemoryDatabase()
		trieDb = triedb.NewDatabase(db, nil)
		gspec  = &Genesis{
			Config: &params.ChainConfig{
				ChainID:        big.NewInt(1337),
				VerkleBlock:    big.NewInt(10),
				HomesteadBlock: big.NewInt(0),
				EIP150Block:    big.NewInt(0),
				EIP155Block:    big.NewInt(0),
				EIP158Block:    big.NewInt(0),
			},
			Alloc: GenesisAlloc{
				common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7"): {Balance: big.NewInt(1000000000000000000)},
				testAddr: {Balance: big.NewInt(1000000000000000000)}, // Add funds to test account
			},
		}
		genesis = gspec.MustCommit(db, trieDb)
		engine  = ethash.NewFaker()
	)

	// Create our blockchain instance using the correct parameter order
	blockchain, err := NewBlockChain(
		db,          // ethdb.Database
		nil,         // *CacheConfig
		gspec,       // *Genesis
		nil,         // *ChainOverrides
		engine,      // consensus.Engine
		vm.Config{}, // vm.Config (not pointer)
		nil,         // *uint64 (lastAcceptedHash)
	)
	if err != nil {
		t.Fatalf("Failed to create blockchain: %v", err)
	}
	defer blockchain.Stop()

	// Create a chain up to the fork block (block 9)
	blocks, _ := GenerateChain(gspec.Config, genesis, engine, db, 9, func(i int, gen *BlockGen) {
		gen.SetCoinbase(common.HexToAddress(fmt.Sprintf("0x%d", i)))
	})

	// Insert blocks up to block 9
	if _, err := blockchain.InsertChain(blocks); err != nil {
		t.Fatalf("Failed to insert initial chain: %v", err)
	}

	// Create two competing forks from block 9 (both are at the Verkle transition height)
	// Fork 1 with account A1
	fork1Block10, _ := GenerateChain(gspec.Config, blocks[len(blocks)-1], engine, db, 1, func(i int, gen *BlockGen) {
		gen.SetCoinbase(common.HexToAddress("0xA1"))
	})

	// Fork 2 with account A2
	fork2Block10, _ := GenerateChain(gspec.Config, blocks[len(blocks)-1], engine, db, 1, func(i int, gen *BlockGen) {
		gen.SetCoinbase(common.HexToAddress("0xA2"))
	})

	// Insert fork 1 (blocks 10-A1)
	if _, err := blockchain.InsertChain(fork1Block10); err != nil {
		t.Fatalf("Failed to insert fork 1: %v", err)
	}

	// Both forks add a new account A3
	addr3 := common.HexToAddress("0xA3")

	// Create block 11 on fork 1 (adding account A3)
	fork1Block11, _ := GenerateChain(gspec.Config, fork1Block10[0], engine, db, 1, func(i int, gen *BlockGen) {
		// Add account A3
		tx, _ := types.SignTx(
			types.NewTransaction(0, addr3, big.NewInt(100), 21000, big.NewInt(1), nil),
			types.LatestSigner(gspec.Config),
			testKey,
		)
		gen.AddTx(tx)
	})

	// Insert block 11 on fork 1
	if _, err := blockchain.InsertChain(fork1Block11); err != nil {
		t.Fatalf("Failed to insert fork 1 block 11: %v", err)
	}

	// Verify the mapping for fork 1's block 10
	baseRoot1, exists := blockchain.blockToBaseStateRoot.Get(fork1Block10[0].Hash())
	if !exists {
		t.Fatalf("Expected mapping to exist for fork 1 block 10")
	}

	// Now simulate a reorg by inserting fork 2 (with higher total difficulty)
	for i := range fork2Block10 {
		fork2Block10[i].Header().Difficulty = new(big.Int).Add(fork2Block10[i].Header().Difficulty, big.NewInt(1000))
	}

	// Insert fork 2 (blocks 10-A2)
	if _, err := blockchain.InsertChain(fork2Block10); err != nil {
		t.Fatalf("Failed to insert fork 2: %v", err)
	}

	// Create block 11 on fork 2 (also adding account A3)
	fork2Block11, _ := GenerateChain(gspec.Config, fork2Block10[0], engine, db, 1, func(i int, gen *BlockGen) {
		// Add same account A3
		tx, _ := types.SignTx(
			types.NewTransaction(0, addr3, big.NewInt(100), 21000, big.NewInt(1), nil),
			types.LatestSigner(gspec.Config),
			testKey,
		)
		gen.AddTx(tx)
	})

	// Ensure fork 2 has higher total difficulty
	for i := range fork2Block11 {
		fork2Block11[i].Header().Difficulty = new(big.Int).Add(fork2Block11[i].Header().Difficulty, big.NewInt(1000))
	}

	// Insert block 11 on fork 2
	if _, err := blockchain.InsertChain(fork2Block11); err != nil {
		t.Fatalf("Failed to insert fork 2 block 11: %v", err)
	}

	// Verify the mapping for fork 2's block 10
	baseRoot2, exists := blockchain.blockToBaseStateRoot.Get(fork2Block10[0].Hash())
	if !exists {
		t.Fatalf("Expected mapping to exist for fork 2 block 10")
	}

	// The base roots should be different
	if baseRoot1 != baseRoot2 {
		t.Fatalf("Expected same base roots since both forks build on the same parent (block 9)")
	}

	// Current chain head should be fork 2 block 11
	if blockchain.CurrentBlock().Hash() != fork2Block11[0].Hash() {
		t.Fatalf("Expected chain head to be fork 2 block 11")
	}

	// Verify we can correctly get state for fork 2 block 11
	state, err := blockchain.StateAt(fork2Block11[0].Root())
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}

	// Check A3 is present in the state (handle uint256.Int vs big.Int)
	balance := state.GetBalance(addr3)
	expectedBalance := big.NewInt(100)

	// Compare as strings to handle both *big.Int and *uint256.Int
	if balance.String() != expectedBalance.String() {
		t.Fatalf("Expected A3 balance of 100, got %v", balance)
	}
}
