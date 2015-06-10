package tests

import (
	"path/filepath"
	"testing"
)

var baseDir = filepath.Join(".", "files")
var blockTestDir = filepath.Join(baseDir, "BlockTests")

// TODO: refactor test setup & execution to better align with vm and tx tests
func TestBcValidBlockTests(t *testing.T) {
	// SimpleTx3 genesis block does not validate against calculated state root
	// as of 2015-06-09. unskip once working /Gustav
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcValidBlockTest.json"), []string{"SimpleTx3"}, t)
}

func TestBcUncleTests(t *testing.T) {
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcUncleTest.json"), []string{}, t)
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcBruncleTest.json"), []string{}, t)
}

func TestBcUncleHeaderValidityTests(t *testing.T) {
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcUncleHeaderValiditiy.json"), []string{}, t)
}

func TestBcInvalidHeaderTests(t *testing.T) {
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcInvalidHeaderTest.json"), []string{}, t)
}

func TestBcInvalidRLPTests(t *testing.T) {
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcInvalidRLPTest.json"), []string{}, t)
}

func TestBcRPCAPITests(t *testing.T) {
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcRPC_API_Test.json"), []string{}, t)
}

func TestBcForkBlockTests(t *testing.T) {
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcForkBlockTest.json"), []string{}, t)
}

func TestBcTotalDifficulty(t *testing.T) {
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcTotalDifficultyTest.json"), []string{}, t)
}

func TestBcWallet(t *testing.T) {
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcWalletTest.json"), []string{}, t)
}
