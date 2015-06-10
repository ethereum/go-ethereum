package tests

import (
	"testing"
)

// I've created a new function for each tests so it's easier to identify where the problem lies if any of them fail.
func TestVMArithmetic(t *testing.T) {
	const fn = "../files/VMTests/vmArithmeticTest.json"
	RunVmTest(fn, t)
}

func TestBitwiseLogicOperation(t *testing.T) {
	const fn = "../files/VMTests/vmBitwiseLogicOperationTest.json"
	RunVmTest(fn, t)
}

func TestBlockInfo(t *testing.T) {
	const fn = "../files/VMTests/vmBlockInfoTest.json"
	RunVmTest(fn, t)
}

func TestEnvironmentalInfo(t *testing.T) {
	const fn = "../files/VMTests/vmEnvironmentalInfoTest.json"
	RunVmTest(fn, t)
}

func TestFlowOperation(t *testing.T) {
	const fn = "../files/VMTests/vmIOandFlowOperationsTest.json"
	RunVmTest(fn, t)
}

func TestLogTest(t *testing.T) {
	const fn = "../files/VMTests/vmLogTest.json"
	RunVmTest(fn, t)
}

func TestPerformance(t *testing.T) {
	const fn = "../files/VMTests/vmPerformanceTest.json"
	RunVmTest(fn, t)
}

func TestPushDupSwap(t *testing.T) {
	const fn = "../files/VMTests/vmPushDupSwapTest.json"
	RunVmTest(fn, t)
}

func TestVMSha3(t *testing.T) {
	const fn = "../files/VMTests/vmSha3Test.json"
	RunVmTest(fn, t)
}

func TestVm(t *testing.T) {
	const fn = "../files/VMTests/vmtests.json"
	RunVmTest(fn, t)
}

func TestVmLog(t *testing.T) {
	const fn = "../files/VMTests/vmLogTest.json"
	RunVmTest(fn, t)
}

func TestInputLimits(t *testing.T) {
	const fn = "../files/VMTests/vmInputLimits.json"
	RunVmTest(fn, t)
}

func TestInputLimitsLight(t *testing.T) {
	const fn = "../files/VMTests/vmInputLimitsLight.json"
	RunVmTest(fn, t)
}

func TestVMRandom(t *testing.T) {
	fns, _ := filepath.Glob("../files/VMTests/RandomTests/*")
	for _, fn := range fns {
		RunVmTest(fn, t)
	}
}
