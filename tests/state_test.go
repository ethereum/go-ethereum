package tests

import (
	"os"
	"path/filepath"
	"testing"
)

var stateTestDir = filepath.Join(baseDir, "StateTests")

func TestStateSystemOperations(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stSystemOperationsTest.json")
	if err := RunStateTest(fn); err != nil {
		t.Error(err)
	}
}

func TestStateExample(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stExample.json")
	if err := RunStateTest(fn); err != nil {
		t.Error(err)
	}
}

func TestStatePreCompiledContracts(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stPreCompiledContracts.json")
	if err := RunStateTest(fn); err != nil {
		t.Error(err)
	}
}

func TestStateRecursiveCreate(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stRecursiveCreate.json")
	if err := RunStateTest(fn); err != nil {
		t.Error(err)
	}
}

func TestStateSpecial(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stSpecialTest.json")
	if err := RunStateTest(fn); err != nil {
		t.Error(err)
	}
}

func TestStateRefund(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stRefundTest.json")
	if err := RunStateTest(fn); err != nil {
		t.Error(err)
	}
}

func TestStateBlockHash(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stBlockHashTest.json")
	if err := RunStateTest(fn); err != nil {
		t.Error(err)
	}
}

func TestStateInitCode(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stInitCodeTest.json")
	if err := RunStateTest(fn); err != nil {
		t.Error(err)
	}
}

func TestStateLog(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stLogTests.json")
	if err := RunStateTest(fn); err != nil {
		t.Error(err)
	}
}

func TestStateTransaction(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stTransactionTest.json")
	if err := RunStateTest(fn); err != nil {
		t.Error(err)
	}
}

func TestCallCreateCallCode(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stCallCreateCallCodeTest.json")
	if err := RunStateTest(fn); err != nil {
		t.Error(err)
	}
}

func TestMemory(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stMemoryTest.json")
	if err := RunStateTest(fn); err != nil {
		t.Error(err)
	}
}

func TestMemoryStress(t *testing.T) {
	if os.Getenv("TEST_VM_COMPLEX") == "" {
		t.Skip()
	}
	fn := filepath.Join(stateTestDir, "stMemoryStressTest.json")
	if err := RunStateTest(fn); err != nil {
		t.Error(err)
	}
}

func TestQuadraticComplexity(t *testing.T) {
	if os.Getenv("TEST_VM_COMPLEX") == "" {
		t.Skip()
	}
	fn := filepath.Join(stateTestDir, "stQuadraticComplexityTest.json")
	if err := RunStateTest(fn); err != nil {
		t.Error(err)
	}
}

func TestSolidity(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stSolidityTest.json")
	if err := RunStateTest(fn); err != nil {
		t.Error(err)
	}
}

func TestWallet(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stWalletTest.json")
	if err := RunStateTest(fn); err != nil {
		t.Error(err)
	}
}

func TestStateTestsRandom(t *testing.T) {
	fns, _ := filepath.Glob("./files/StateTests/RandomTests/*")
	for _, fn := range fns {
		if err := RunStateTest(fn); err != nil {
			t.Error(err)
		}
	}
}
