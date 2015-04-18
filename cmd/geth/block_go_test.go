package main

import (
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/tests"
)

// TODO: refactor test setup & execution to better align with vm and tx tests
// TODO: refactor to avoid duplication with cmd/geth/blocktest.go
func TestBcValidBlockTests(t *testing.T) {
	runBlockTestsInFile("../../tests/files/BlockTests/bcValidBlockTest.json", t)
}

/*
func TestBcUncleTests(t *testing.T) {
	runBlockTestsInFile("../../tests/files/BlockTests/bcUncleTest.json", t)
}
*/

func runBlockTestsInFile(filepath string, t *testing.T) {
	bt, err := tests.LoadBlockTests(filepath)
	if err != nil {
		t.Fatal(err)
	}
	for name, test := range bt {
		runTest(name, test, t)
	}
}

func runTest(name string, test *tests.BlockTest, t *testing.T) {
	t.Log("Running test: ", name)
	cfg := testEthConfig()
	ethereum, err := eth.New(cfg)
	if err != nil {
		t.Fatalf("%v", err)
	}

	err = ethereum.Start()
	if err != nil {
		t.Fatalf("%v", err)
	}

	// import the genesis block
	ethereum.ResetWithGenesisBlock(test.Genesis)

	// import pre accounts
	statedb, err := test.InsertPreState(ethereum.StateDb())
	if err != nil {
		t.Fatalf("InsertPreState: %v", err)
	}

	// insert the test blocks, which will execute all transactions
	if err := test.InsertBlocks(ethereum.ChainManager()); err != nil {
		t.Fatalf("Block Test load error: %v %T", err, err)
	}

	if err := test.ValidatePostState(statedb); err != nil {
		t.Fatal("post state validation failed: %v", err)
	}
	t.Log("Test passed: ", name)
}

func testEthConfig() *eth.Config {
	ks := crypto.NewKeyStorePassphrase(path.Join(common.DefaultDataDir(), "keys"))

	return &eth.Config{
		DataDir:        common.DefaultDataDir(),
		LogLevel:       5,
		Etherbase:      "primary",
		AccountManager: accounts.NewManager(ks),
		NewDB:          func(path string) (common.Database, error) { return ethdb.NewMemDatabase() },
	}
}
