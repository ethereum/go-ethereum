// Copyright 2016 The Go-etacoin Authors
// This file is part of Go-etacoin.
//
// Go-etacoin is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Go-etacoin is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Go-etacoin.  If not, see <http://www.gnu.org/licenses/>.

package tests

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkVmAckermann32Tests(b *testing.B) {
	fn := filepath.Join(vmTestDir, "vmPerformanceTest.json")
	if err := BenchVmTest(fn, bconf{"ackermann32", os.Getenv("JITFORCE") == "true", os.Getenv("JITVM") == "true"}, b); err != nil {
		b.Error(err)
	}
}

func BenchmarkVmFibonacci16Tests(b *testing.B) {
	fn := filepath.Join(vmTestDir, "vmPerformanceTest.json")
	if err := BenchVmTest(fn, bconf{"fibonacci16", os.Getenv("JITFORCE") == "true", os.Getenv("JITVM") == "true"}, b); err != nil {
		b.Error(err)
	}
}

// I've created a new function for each tests so it's easier to identify where the problem lies if any of them fail.
func TestVMArithmetic(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmArithmeticTest.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestBitwiseLogicOperation(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmBitwiseLogicOperationTest.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestBlockInfo(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmBlockInfoTest.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestEnvironmentalInfo(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmEnvironmentalInfoTest.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestFlowOperation(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmIOandFlowOperationsTest.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestLogTest(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmLogTest.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestPerformance(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmPerformanceTest.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestPushDupSwap(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmPushDupSwapTest.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestVMSha3(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmSha3Test.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestVm(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmtests.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestVmLog(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmLogTest.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestInputLimits(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmInputLimits.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestInputLimitsLight(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmInputLimitsLight.json")
	if err := RunVmTest(fn, VmSkipTests); err != nil {
		t.Error(err)
	}
}

func TestVMRandom(t *testing.T) {
	fns, _ := filepath.Glob(filepath.Join(baseDir, "RandomTests", "*"))
	for _, fn := range fns {
		if err := RunVmTest(fn, VmSkipTests); err != nil {
			t.Error(err)
		}
	}
}
