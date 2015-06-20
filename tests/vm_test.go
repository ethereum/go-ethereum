package tests

import (
	"path/filepath"
	"testing"
)

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
