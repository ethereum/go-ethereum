package tests

import (
	"os"
	"path/filepath"
	"testing"
)

var stateTestDir = filepath.Join(baseDir, "StateTests")

func TestStateSystemOperations(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stSystemOperationsTest.json")
	RunVmTest(fn, t)
}

func TestStateExample(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stExample.json")
	RunVmTest(fn, t)
}

func TestStatePreCompiledContracts(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stPreCompiledContracts.json")
	RunVmTest(fn, t)
}

func TestStateRecursiveCreate(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stRecursiveCreate.json")
	RunVmTest(fn, t)
}

func TestStateSpecial(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stSpecialTest.json")
	RunVmTest(fn, t)
}

func TestStateRefund(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stRefundTest.json")
	RunVmTest(fn, t)
}

func TestStateBlockHash(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stBlockHashTest.json")
	RunVmTest(fn, t)
}

func TestStateInitCode(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stInitCodeTest.json")
	RunVmTest(fn, t)
}

func TestStateLog(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stLogTests.json")
	RunVmTest(fn, t)
}

func TestStateTransaction(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stTransactionTest.json")
	RunVmTest(fn, t)
}

func TestCallCreateCallCode(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stCallCreateCallCodeTest.json")
	RunVmTest(fn, t)
}

func TestMemory(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stMemoryTest.json")
	RunVmTest(fn, t)
}

func TestMemoryStress(t *testing.T) {
	if os.Getenv("TEST_VM_COMPLEX") == "" {
		t.Skip()
	}
	fn := filepath.Join(stateTestDir, "stMemoryStressTest.json")
	RunVmTest(fn, t)
}

func TestQuadraticComplexity(t *testing.T) {
	if os.Getenv("TEST_VM_COMPLEX") == "" {
		t.Skip()
	}
	fn := filepath.Join(stateTestDir, "stQuadraticComplexityTest.json")
	RunVmTest(fn, t)
}

func TestSolidity(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stSolidityTest.json")
	RunVmTest(fn, t)
}

func TestWallet(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stWalletTest.json")
	RunVmTest(fn, t)
}

func TestStateTestsRandom(t *testing.T) {
	fns, _ := filepath.Glob("./files/StateTests/RandomTests/*")
	for _, fn := range fns {
		RunVmTest(fn, t)
	}
}
