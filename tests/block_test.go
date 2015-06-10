package tests

import (
	"path/filepath"
	"testing"
)

var baseDir = filepath.Join(".", "files")
var blockTestDir = filepath.Join(baseDir, "BlockTests")

// TODO: refactor test setup & execution to better align with vm and tx tests
func TestBcValidBlockTests(t *testing.T) {
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcValidBlockTest.json"), []string{"SimpleTx3"})
}

func TestBcUncleTests(t *testing.T) {
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcUncleTest.json"), []string{})
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcBruncleTest.json"), []string{})
}

func TestBcUncleHeaderValidityTests(t *testing.T) {
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcUncleHeaderValiditiy.json"), []string{})
}

func TestBcInvalidHeaderTests(t *testing.T) {
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcInvalidHeaderTest.json"), []string{})
}

func TestBcInvalidRLPTests(t *testing.T) {
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcInvalidRLPTest.json"), []string{})
}

func TestBcRPCAPITests(t *testing.T) {
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcRPC_API_Test.json"), []string{})
}

func TestBcForkBlockTests(t *testing.T) {
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcForkBlockTest.json"), []string{})
}

func TestBcTotalDifficulty(t *testing.T) {
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcTotalDifficultyTest.json"), []string{})
}

func TestBcWallet(t *testing.T) {
	runBlockTestsInFile(filepath.Join(blockTestDir, "bcWalletTest.json"), []string{})
}
