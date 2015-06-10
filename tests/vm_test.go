package tests

import (
	"path/filepath"
	"testing"
)

var vmTestDir = filepath.Join(baseDir, "VMTests")

// I've created a new function for each tests so it's easier to identify where the problem lies if any of them fail.
func TestVMArithmetic(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmArithmeticTest.json")
	RunVmTest(fn, t)
}

func TestBitwiseLogicOperation(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmBitwiseLogicOperationTest.json")
	RunVmTest(fn, t)
}

func TestBlockInfo(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmBlockInfoTest.json")
	RunVmTest(fn, t)
}

func TestEnvironmentalInfo(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmEnvironmentalInfoTest.json")
	RunVmTest(fn, t)
}

func TestFlowOperation(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmIOandFlowOperationsTest.json")
	RunVmTest(fn, t)
}

func TestLogTest(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmLogTest.json")
	RunVmTest(fn, t)
}

func TestPerformance(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmPerformanceTest.json")
	RunVmTest(fn, t)
}

func TestPushDupSwap(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmPushDupSwapTest.json")
	RunVmTest(fn, t)
}

func TestVMSha3(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmSha3Test.json")
	RunVmTest(fn, t)
}

func TestVm(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmtests.json")
	RunVmTest(fn, t)
}

func TestVmLog(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmLogTest.json")
	RunVmTest(fn, t)
}

func TestInputLimits(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmInputLimits.json")
	RunVmTest(fn, t)
}

func TestInputLimitsLight(t *testing.T) {
	fn := filepath.Join(vmTestDir, "vmInputLimitsLight.json")
	RunVmTest(fn, t)
}

func TestVMRandom(t *testing.T) {
	fns, _ := filepath.Glob(filepath.Join(baseDir, "RandomTests", "*"))
	for _, fn := range fns {
		RunVmTest(fn, t)
	}
}
