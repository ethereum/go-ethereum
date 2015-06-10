package tests

import (
	"os"
	"path/filepath"
	"testing"
)

var stateTestDir = filepath.Join(baseDir, "StateTests")

func TestStateSystemOperations(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stSystemOperationsTest.json")
	RunStateTest(fn, t)
}

func TestStateExample(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stExample.json")
	RunStateTest(fn, t)
}

func TestStatePreCompiledContracts(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stPreCompiledContracts.json")
	RunStateTest(fn, t)
}

func TestStateRecursiveCreate(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stRecursiveCreate.json")
	RunStateTest(fn, t)
}

func TestStateSpecial(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stSpecialTest.json")
	RunStateTest(fn, t)
}

func TestStateRefund(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stRefundTest.json")
	RunStateTest(fn, t)
}

func TestStateBlockHash(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stBlockHashTest.json")
	RunStateTest(fn, t)
}

func TestStateInitCode(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stInitCodeTest.json")
	RunStateTest(fn, t)
}

func TestStateLog(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stLogTests.json")
	RunStateTest(fn, t)
}

func TestStateTransaction(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stTransactionTest.json")
	RunStateTest(fn, t)
}

func TestCallCreateCallCode(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stCallCreateCallCodeTest.json")
	RunStateTest(fn, t)
}

func TestMemory(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stMemoryTest.json")
	RunStateTest(fn, t)
}

func TestMemoryStress(t *testing.T) {
	if os.Getenv("TEST_VM_COMPLEX") == "" {
		t.Skip()
	}
	fn := filepath.Join(stateTestDir, "stMemoryStressTest.json")
	RunStateTest(fn, t)
}

func TestQuadraticComplexity(t *testing.T) {
	if os.Getenv("TEST_VM_COMPLEX") == "" {
		t.Skip()
	}
	fn := filepath.Join(stateTestDir, "stQuadraticComplexityTest.json")
	RunStateTest(fn, t)
}

func TestSolidity(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stSolidityTest.json")
	RunStateTest(fn, t)
}

func TestWallet(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stWalletTest.json")
	RunStateTest(fn, t)
}

func TestStateTestsRandom(t *testing.T) {
	fns, _ := filepath.Glob("./files/StateTests/RandomTests/*")
	for _, fn := range fns {
		RunStateTest(fn, t)
	}
}
