// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package tests

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkStateCall1024(b *testing.B) {
	fn := filepath.Join(stateTestDir, "stCallCreateCallCodeTest.json")
	if err := BenchVmTest(fn, bconf{"Call1024BalanceTooLow", true, os.Getenv("JITVM") == "true"}, b); err != nil {
		b.Error(err)
	}
}

func TestStateSystemOperations(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	fn := filepath.Join(stateTestDir, "stSystemOperationsTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateExample(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	fn := filepath.Join(stateTestDir, "stExample.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStatePreCompiledContracts(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	fn := filepath.Join(stateTestDir, "stPreCompiledContracts.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateRecursiveCreate(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	fn := filepath.Join(stateTestDir, "stRecursiveCreate.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateSpecial(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	fn := filepath.Join(stateTestDir, "stSpecialTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateRefund(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	fn := filepath.Join(stateTestDir, "stRefundTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateBlockHash(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	fn := filepath.Join(stateTestDir, "stBlockHashTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateInitCode(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	fn := filepath.Join(stateTestDir, "stInitCodeTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateLog(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	fn := filepath.Join(stateTestDir, "stLogTests.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateTransaction(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	fn := filepath.Join(stateTestDir, "stTransactionTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateTransition(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	fn := filepath.Join(stateTestDir, "stTransitionTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestCallCreateCallCode(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	fn := filepath.Join(stateTestDir, "stCallCreateCallCodeTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestCallCodes(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	fn := filepath.Join(stateTestDir, "stCallCodes.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestDelegateCall(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	fn := filepath.Join(stateTestDir, "stDelegatecallTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestMemory(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	fn := filepath.Join(stateTestDir, "stMemoryTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestMemoryStress(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	if os.Getenv("TEST_VM_COMPLEX") == "" {
		t.Skip()
	}
	fn := filepath.Join(stateTestDir, "stMemoryStressTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestQuadraticComplexity(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	if os.Getenv("TEST_VM_COMPLEX") == "" {
		t.Skip()
	}
	fn := filepath.Join(stateTestDir, "stQuadraticComplexityTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestSolidity(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	fn := filepath.Join(stateTestDir, "stSolidityTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestWallet(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	fn := filepath.Join(stateTestDir, "stWalletTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateTestsRandom(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: big.NewInt(1150000),
	}

	fns, _ := filepath.Glob("./files/StateTests/RandomTests/*")
	for _, fn := range fns {
		if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
			t.Error(fn, err)
		}
	}
}

// homestead tests
func TestHomesteadStateSystemOperations(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: new(big.Int),
	}

	fn := filepath.Join(stateTestDir, "Homestead", "stSystemOperationsTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestHomesteadStatePreCompiledContracts(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: new(big.Int),
	}

	fn := filepath.Join(stateTestDir, "Homestead", "stPreCompiledContracts.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestHomesteadStateRecursiveCreate(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: new(big.Int),
	}

	fn := filepath.Join(stateTestDir, "Homestead", "stSpecialTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestHomesteadStateRefund(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: new(big.Int),
	}

	fn := filepath.Join(stateTestDir, "Homestead", "stRefundTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestHomesteadStateInitCode(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: new(big.Int),
	}

	fn := filepath.Join(stateTestDir, "Homestead", "stInitCodeTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestHomesteadStateLog(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: new(big.Int),
	}

	fn := filepath.Join(stateTestDir, "Homestead", "stLogTests.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestHomesteadStateTransaction(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: new(big.Int),
	}

	fn := filepath.Join(stateTestDir, "Homestead", "stTransactionTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestHomesteadCallCreateCallCode(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: new(big.Int),
	}

	fn := filepath.Join(stateTestDir, "Homestead", "stCallCreateCallCodeTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestHomesteadCallCodes(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: new(big.Int),
	}

	fn := filepath.Join(stateTestDir, "Homestead", "stCallCodes.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestHomesteadMemory(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: new(big.Int),
	}

	fn := filepath.Join(stateTestDir, "Homestead", "stMemoryTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestHomesteadMemoryStress(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: new(big.Int),
	}

	if os.Getenv("TEST_VM_COMPLEX") == "" {
		t.Skip()
	}
	fn := filepath.Join(stateTestDir, "Homestead", "stMemoryStressTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestHomesteadQuadraticComplexity(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: new(big.Int),
	}

	if os.Getenv("TEST_VM_COMPLEX") == "" {
		t.Skip()
	}
	fn := filepath.Join(stateTestDir, "Homestead", "stQuadraticComplexityTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestHomesteadWallet(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: new(big.Int),
	}

	fn := filepath.Join(stateTestDir, "Homestead", "stWalletTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestHomesteadDelegateCodes(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: new(big.Int),
	}

	fn := filepath.Join(stateTestDir, "Homestead", "stCallDelegateCodes.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestHomesteadDelegateCodesCallCode(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: new(big.Int),
	}

	fn := filepath.Join(stateTestDir, "Homestead", "stCallDelegateCodesCallCode.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestHomesteadBounds(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock: new(big.Int),
	}

	fn := filepath.Join(stateTestDir, "Homestead", "stBoundsTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

// EIP150 tests
func TestEIP150Specific(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	fn := filepath.Join(stateTestDir, "EIP150", "stEIPSpecificTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEIP150SingleCodeGasPrice(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	fn := filepath.Join(stateTestDir, "EIP150", "stEIPSingleCodeGasPrices.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEIP150MemExpandingCalls(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	fn := filepath.Join(stateTestDir, "EIP150", "stMemExpandingEIPCalls.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEIP150HomesteadStateSystemOperations(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	fn := filepath.Join(stateTestDir, "EIP150", "Homestead", "stSystemOperationsTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEIP150HomesteadStatePreCompiledContracts(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	fn := filepath.Join(stateTestDir, "EIP150", "Homestead", "stPreCompiledContracts.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEIP150HomesteadStateRecursiveCreate(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	fn := filepath.Join(stateTestDir, "EIP150", "Homestead", "stSpecialTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEIP150HomesteadStateRefund(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	fn := filepath.Join(stateTestDir, "EIP150", "Homestead", "stRefundTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEIP150HomesteadStateInitCode(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	fn := filepath.Join(stateTestDir, "EIP150", "Homestead", "stInitCodeTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEIP150HomesteadStateLog(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	fn := filepath.Join(stateTestDir, "EIP150", "Homestead", "stLogTests.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEIP150HomesteadStateTransaction(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	fn := filepath.Join(stateTestDir, "EIP150", "Homestead", "stTransactionTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEIP150HomesteadCallCreateCallCode(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	fn := filepath.Join(stateTestDir, "EIP150", "Homestead", "stCallCreateCallCodeTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEIP150HomesteadCallCodes(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	fn := filepath.Join(stateTestDir, "EIP150", "Homestead", "stCallCodes.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEIP150HomesteadMemory(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	fn := filepath.Join(stateTestDir, "EIP150", "Homestead", "stMemoryTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEIP150HomesteadMemoryStress(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	if os.Getenv("TEST_VM_COMPLEX") == "" {
		t.Skip()
	}
	fn := filepath.Join(stateTestDir, "EIP150", "Homestead", "stMemoryStressTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEIP150HomesteadQuadraticComplexity(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	if os.Getenv("TEST_VM_COMPLEX") == "" {
		t.Skip()
	}
	fn := filepath.Join(stateTestDir, "EIP150", "Homestead", "stQuadraticComplexityTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEIP150HomesteadWallet(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	fn := filepath.Join(stateTestDir, "EIP150", "Homestead", "stWalletTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEIP150HomesteadDelegateCodes(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	fn := filepath.Join(stateTestDir, "EIP150", "Homestead", "stCallDelegateCodes.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEIP150HomesteadDelegateCodesCallCode(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	fn := filepath.Join(stateTestDir, "EIP150", "Homestead", "stCallDelegateCodesCallCode.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEIP150HomesteadBounds(t *testing.T) {
	ruleSet := RuleSet{
		HomesteadBlock:           new(big.Int),
		HomesteadGasRepriceBlock: big.NewInt(2457000),
	}

	fn := filepath.Join(stateTestDir, "EIP150", "Homestead", "stBoundsTest.json")
	if err := RunStateTest(ruleSet, fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}
