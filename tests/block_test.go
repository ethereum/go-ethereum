package tests

import (
	"testing"
)

// TODO: refactor test setup & execution to better align with vm and tx tests
func TestBcValidBlockTests(t *testing.T) {
	// SimpleTx3 genesis block does not validate against calculated state root
	// as of 2015-06-09. unskip once working /Gustav
	runBlockTestsInFile("files/BlockTests/bcValidBlockTest.json", []string{"SimpleTx3"}, t)
}

func TestBcUncleTests(t *testing.T) {
	runBlockTestsInFile("files/BlockTests/bcUncleTest.json", []string{}, t)
	runBlockTestsInFile("files/BlockTests/bcBruncleTest.json", []string{}, t)
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

func TestBcRPCAPITests(t *testing.T) {
	runBlockTestsInFile("files/BlockTests/bcRPC_API_Test.json", []string{}, t)
}

func TestBcForkBlockTests(t *testing.T) {
	runBlockTestsInFile("files/BlockTests/bcForkBlockTest.json", []string{}, t)
}

func TestBcTotalDifficulty(t *testing.T) {
	runBlockTestsInFile("files/BlockTests/bcTotalDifficultyTest.json", []string{}, t)
}

func TestBcWallet(t *testing.T) {
	runBlockTestsInFile("files/BlockTests/bcWalletTest.json", []string{}, t)
}
