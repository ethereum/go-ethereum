package vm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

func TestByteOp(t *testing.T) {
	var (
		env   = NewEVM(Context{}, nil, params.TestChainConfig, Config{EnableJit: false, ForceJit: false})
		stack = newstack()
	)
	tests := []struct {
		v        string
		th       uint64
		expected *big.Int
	}{
		{"ABCDEF0908070605040302010000000000000000000000000000000000000000", 0, big.NewInt(0xAB)},
		{"ABCDEF0908070605040302010000000000000000000000000000000000000000", 1, big.NewInt(0xCD)},
		{"00CDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff", 0, big.NewInt(0x00)},
		{"00CDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff", 1, big.NewInt(0xCD)},
		{"0000000000000000000000000000000000000000000000000000000000102030", 31, big.NewInt(0x30)},
		{"0000000000000000000000000000000000000000000000000000000000102030", 30, big.NewInt(0x20)},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 32, big.NewInt(0x0)},
		{"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 0xFFFFFFFFFFFFFFFF, big.NewInt(0x0)},
	}
	pc := uint64(0)
	for _, test := range tests {
		val := new(big.Int).SetBytes(common.Hex2Bytes(test.v))
		th := new(big.Int).SetUint64(test.th)
		stack.push(val)
		stack.push(th)
		opByte(&pc, env, nil, nil, stack)
		actual := stack.pop()
		if actual.Cmp(test.expected) != 0 {
			t.Fatalf("Expected  [%v] %v:th byte to be %v, was %v.", test.v, test.th, test.expected, actual)
		}
	}
}

func getBenchmarker(op func(pc *uint64, evm *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error), args ...string) func(b *testing.B) {
	x := func(bench *testing.B) {
		var (
			env   = NewEVM(Context{}, nil, params.TestChainConfig, Config{EnableJit: false, ForceJit: false})
			stack = newstack()
		)
		// convert args
		byteArgs := make([][]byte, len(args))
		for i, arg := range args {
			byteArgs[i] = common.Hex2Bytes(arg)
		}
		pc := uint64(0)
		bench.ResetTimer()
		for i := 0; i < bench.N; i++ {
			for _, arg := range byteArgs {
				a := new(big.Int).SetBytes(arg)
				stack.push(a)
			}
			op(&pc, env, nil, nil, stack)
			stack.pop()
		}
	}
	return x
}

func BenchmarkOpAdd(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opAdd, x, y)
	f(b)
}
func BenchmarkOpSub(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opSub, x, y)
	f(b)
}
func BenchmarkOpMul(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opMul, x, y)

	f(b)
}
func BenchmarkOpDiv(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opDiv, x, y)
	f(b)
}
func BenchmarkOpSdiv(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opSdiv, x, y)
	f(b)
}
func BenchmarkOpMod(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opMod, x, y)
	f(b)
}
func BenchmarkOpSmod(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opSmod, x, y)
	f(b)
}
func BenchmarkOpExp(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opExp, x, y)

	f(b)
}
func BenchmarkOpSignExtend(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opSignExtend, x, y)

	f(b)
}
func BenchmarkOpLt(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opLt, x, y)
	f(b)
}
func BenchmarkOpGt(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opGt, x, y)
	f(b)
}
func BenchmarkOpSlt(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opSlt, x, y)
	f(b)
}
func BenchmarkOpSgt(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opSgt, x, y)

	f(b)
}
func BenchmarkOpEq(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opEq, x, y)
	f(b)
}
func BenchmarkOpAnd(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opAnd, x, y)
	f(b)
}
func BenchmarkOpOr(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opOr, x, y)
	f(b)
}
func BenchmarkOpXor(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opXor, x, y)
	f(b)
}
func BenchmarkOpByte(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opByte, x, y)
	f(b)
}

func BenchmarkOpAddmod(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	z := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opAddmod, x, y, z)
	f(b)
}
func BenchmarkOpMulmod(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	z := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	f := getBenchmarker(opMulmod, x, y, z)
	f(b)
}

//func BenchmarkOpSha3(b *testing.B) {
//	x := "0"
//	y := "32"
//
//	f := getBenchmarker(opSha3, x, y)
//
//	f(b)
//}
