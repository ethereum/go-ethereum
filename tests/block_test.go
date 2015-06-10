package tests

import (
	"path/filepath"
	"testing"
)

func TestBcValidBlockTests(t *testing.T) {
	err := RunBlockTest(filepath.Join(blockTestDir, "bcValidBlockTest.json"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcUncleTests(t *testing.T) {
	err := RunBlockTest(filepath.Join(blockTestDir, "bcUncleTest.json"))
	if err != nil {
		t.Fatal(err)
	}
	err = RunBlockTest(filepath.Join(blockTestDir, "bcBruncleTest.json"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcUncleHeaderValidityTests(t *testing.T) {
	err := RunBlockTest(filepath.Join(blockTestDir, "bcUncleHeaderValiditiy.json"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcInvalidHeaderTests(t *testing.T) {
	err := RunBlockTest(filepath.Join(blockTestDir, "bcInvalidHeaderTest.json"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcInvalidRLPTests(t *testing.T) {
	err := RunBlockTest(filepath.Join(blockTestDir, "bcInvalidRLPTest.json"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcRPCAPITests(t *testing.T) {
	err := RunBlockTest(filepath.Join(blockTestDir, "bcRPC_API_Test.json"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcForkBlockTests(t *testing.T) {
	err := RunBlockTest(filepath.Join(blockTestDir, "bcForkBlockTest.json"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcTotalDifficulty(t *testing.T) {
	err := RunBlockTest(filepath.Join(blockTestDir, "bcTotalDifficultyTest.json"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestBcWallet(t *testing.T) {
	err := RunBlockTest(filepath.Join(blockTestDir, "bcWalletTest.json"))
	if err != nil {
		t.Fatal(err)
	}
}
