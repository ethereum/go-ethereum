package tests

// TODO: figure out how to move this file to tests package and get imports working there
import (
	//	"os"
	"path"

	// TODO: refactor to avoid depending on CLI stuff
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"testing"
)

// TODO: refactor test setup & execution to better align with vm and tx tests
// TODO: refactor to avoid duplication with cmd/geth/blocktest.go
func TestBcValidBlockTests(t *testing.T) {
	//dir, _ := os.Getwd()
	//t.Logf("CWD: ", dir)
	runBlockTestsInFile("files/BlockTests/bcValidBlockTest.json", t)
}

func runBlockTestsInFile(filepath string, t *testing.T) {
	bt, err := LoadBlockTests(filepath)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range bt {
		runBlockTest(test, t)
	}
}

func runBlockTest(test *BlockTest, t *testing.T) {
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
	t.Log("chain loaded")

	if err := test.ValidatePostState(statedb); err != nil {
		t.Fatal("post state validation failed: %v", err)
	}
	t.Log("Block Test post state validated.")
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
