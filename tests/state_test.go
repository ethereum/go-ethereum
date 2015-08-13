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
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
)

func init() {
	if os.Getenv("JITVM") == "true" {
		vm.ForceJit = true
		vm.EnableJit = true
	}
}

func BenchmarkStateCall1024(b *testing.B) {
	fn := filepath.Join(stateTestDir, "stCallCreateCallCodeTest.json")
	if err := BenchVmTest(fn, bconf{"Call1024BalanceTooLow", true, os.Getenv("JITVM") == "true"}, b); err != nil {
		b.Error(err)
	}
}

func TestStateSystemOperations(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stSystemOperationsTest.json")
	if err := RunStateTest(fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateExample(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stExample.json")
	if err := RunStateTest(fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStatePreCompiledContracts(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stPreCompiledContracts.json")
	if err := RunStateTest(fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateRecursiveCreate(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stRecursiveCreate.json")
	if err := RunStateTest(fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateSpecial(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stSpecialTest.json")
	if err := RunStateTest(fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateRefund(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stRefundTest.json")
	if err := RunStateTest(fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateBlockHash(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stBlockHashTest.json")
	if err := RunStateTest(fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateInitCode(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stInitCodeTest.json")
	if err := RunStateTest(fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateLog(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stLogTests.json")
	if err := RunStateTest(fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateTransaction(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stTransactionTest.json")
	if err := RunStateTest(fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestCallCreateCallCode(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stCallCreateCallCodeTest.json")
	if err := RunStateTest(fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestMemory(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stMemoryTest.json")
	if err := RunStateTest(fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestMemoryStress(t *testing.T) {
	if os.Getenv("TEST_VM_COMPLEX") == "" {
		t.Skip()
	}
	fn := filepath.Join(stateTestDir, "stMemoryStressTest.json")
	if err := RunStateTest(fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestQuadraticComplexity(t *testing.T) {
	if os.Getenv("TEST_VM_COMPLEX") == "" {
		t.Skip()
	}
	fn := filepath.Join(stateTestDir, "stQuadraticComplexityTest.json")
	if err := RunStateTest(fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestSolidity(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stSolidityTest.json")
	if err := RunStateTest(fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestWallet(t *testing.T) {
	fn := filepath.Join(stateTestDir, "stWalletTest.json")
	if err := RunStateTest(fn, StateSkipTests); err != nil {
		t.Error(err)
	}
}

func TestStateTestsRandom(t *testing.T) {
	fns, _ := filepath.Glob("./files/StateTests/RandomTests/*")
	for _, fn := range fns {
		if err := RunStateTest(fn, StateSkipTests); err != nil {
			t.Error(err)
		}
	}
}
