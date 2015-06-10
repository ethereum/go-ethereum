package tests

import "testing"

func TestStateSystemOperations(t *testing.T) {
	const fn = "../files/StateTests/stSystemOperationsTest.json"
	RunVmTest(fn, t)
}

func TestStateExample(t *testing.T) {
	const fn = "../files/StateTests/stExample.json"
	RunVmTest(fn, t)
}

func TestStatePreCompiledContracts(t *testing.T) {
	const fn = "../files/StateTests/stPreCompiledContracts.json"
	RunVmTest(fn, t)
}

func TestStateRecursiveCreate(t *testing.T) {
	const fn = "../files/StateTests/stRecursiveCreate.json"
	RunVmTest(fn, t)
}

func TestStateSpecial(t *testing.T) {
	const fn = "../files/StateTests/stSpecialTest.json"
	RunVmTest(fn, t)
}

func TestStateRefund(t *testing.T) {
	const fn = "../files/StateTests/stRefundTest.json"
	RunVmTest(fn, t)
}

func TestStateBlockHash(t *testing.T) {
	const fn = "../files/StateTests/stBlockHashTest.json"
	RunVmTest(fn, t)
}

func TestStateInitCode(t *testing.T) {
	const fn = "../files/StateTests/stInitCodeTest.json"
	RunVmTest(fn, t)
}

func TestStateLog(t *testing.T) {
	const fn = "../files/StateTests/stLogTests.json"
	RunVmTest(fn, t)
}

func TestStateTransaction(t *testing.T) {
	const fn = "../files/StateTests/stTransactionTest.json"
	RunVmTest(fn, t)
}

func TestCallCreateCallCode(t *testing.T) {
	const fn = "../files/StateTests/stCallCreateCallCodeTest.json"
	RunVmTest(fn, t)
}

func TestMemory(t *testing.T) {
	const fn = "../files/StateTests/stMemoryTest.json"
	RunVmTest(fn, t)
}

func TestMemoryStress(t *testing.T) {
	if os.Getenv("TEST_VM_COMPLEX") == "" {
		t.Skip()
	}
	const fn = "../files/StateTests/stMemoryStressTest.json"
	RunVmTest(fn, t)
}

func TestQuadraticComplexity(t *testing.T) {
	if os.Getenv("TEST_VM_COMPLEX") == "" {
		t.Skip()
	}
	const fn = "../files/StateTests/stQuadraticComplexityTest.json"
	RunVmTest(fn, t)
}

func TestSolidity(t *testing.T) {
	const fn = "../files/StateTests/stSolidityTest.json"
	RunVmTest(fn, t)
}

func TestWallet(t *testing.T) {
	const fn = "../files/StateTests/stWalletTest.json"
	RunVmTest(fn, t)
}

func TestStateTestsRandom(t *testing.T) {
	fns, _ := filepath.Glob("../files/StateTests/RandomTests/*")
	for _, fn := range fns {
		RunVmTest(fn, t)
	}
}
