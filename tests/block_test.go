package tests

import (
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
)

// TODO: refactor test setup & execution to better align with vm and tx tests
func TestBcValidBlockTests(t *testing.T) {
	runBlockTestsInFile("files/BlockTests/bcValidBlockTest.json", []string{}, t)
}

func TestBcUncleTests(t *testing.T) {
	runBlockTestsInFile("files/BlockTests/bcUncleTest.json", []string{}, t)
}

func TestBcUncleHeaderValidityTests(t *testing.T) {
	runBlockTestsInFile("files/BlockTests/bcUncleHeaderValiditiy.json", []string{}, t)
}

func TestBcInvalidHeaderTests(t *testing.T) {
	runBlockTestsInFile("files/BlockTests/bcInvalidHeaderTest.json", []string{}, t)
}

func TestBcInvalidRLPTests(t *testing.T) {
	runBlockTestsInFile("files/BlockTests/bcInvalidRLPTest.json", []string{}, t)
}

func TestBcJSAPITests(t *testing.T) {
	runBlockTestsInFile("files/BlockTests/bcJS_API_Test.json", []string{}, t)
}

func TestBcRPCAPITests(t *testing.T) {
	runBlockTestsInFile("files/BlockTests/bcRPC_API_Test.json", []string{}, t)
}

func TestBcForkBlockTests(t *testing.T) {
	runBlockTestsInFile("files/BlockTests/bcForkBlockTest.json", []string{}, t)
}

func TestBcTotalDifficulty(t *testing.T) {
	runBlockTestsInFile("files/BlockTests/bcTotalDifficultyTest.json", []string{}, t)
}

func runBlockTestsInFile(filepath string, snafus []string, t *testing.T) {
	bt, err := LoadBlockTests(filepath)
	if err != nil {
		t.Fatal(err)
	}

	notWorking := make(map[string]bool, 100)
	for _, name := range snafus {
		notWorking[name] = true
	}

	for name, test := range bt {
		if !notWorking[name] {
			runBlockTest(name, test, t)
		}
	}
}

func runBlockTest(name string, test *BlockTest, t *testing.T) {
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
	statedb, err := test.InsertPreState(ethereum)
	if err != nil {
		t.Fatalf("InsertPreState: %v", err)
	}

	err = test.TryBlocksInsert(ethereum.ChainManager())
	if err != nil {
		t.Fatal(err)
	}

	if err = test.ValidatePostState(statedb); err != nil {
		t.Fatal("post state validation failed: %v", err)
	}
	t.Log("Test passed: ", name)
}

func testEthConfig() *eth.Config {
	ks := crypto.NewKeyStorePassphrase(filepath.Join(common.DefaultDataDir(), "keys"))

	return &eth.Config{
		DataDir:        common.DefaultDataDir(),
		Verbosity:      5,
		Etherbase:      "primary",
		AccountManager: accounts.NewManager(ks),
		NewDB:          func(path string) (common.Database, error) { return ethdb.NewMemDatabase() },
	}
}
