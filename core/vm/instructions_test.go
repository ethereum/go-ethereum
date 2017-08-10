package vm

import (
	"fmt"
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

func opBenchmark(bench *testing.B, op func(pc *uint64, evm *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error), args ...string) {
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

func precompiledBenchmark(addr, input, expected string, gas uint64, bench *testing.B) {

	contract := NewContract(AccountRef(common.HexToAddress("1337")),
		nil, new(big.Int), gas)

	p := PrecompiledContractsMetropolis[common.HexToAddress(addr)]
	in := common.Hex2Bytes(input)
	var (
		res []byte
		err error
	)
	data := make([]byte, len(in))
	bench.ResetTimer()
	for i := 0; i < bench.N; i++ {
		contract.Gas = gas
		copy(data, in)
		res, err = RunPrecompiledContract(p, data, contract)
	}
	bench.StopTimer()
	//Check if it is correct
	if err != nil {
		bench.Error(err)
		return
	}
	if common.Bytes2Hex(res) != expected {
		bench.Error(fmt.Sprintf("Expected %v, got %v", expected, common.Bytes2Hex(res)))
		return
	}
}

func BenchmarkPrecompiledECDSA(bench *testing.B) {
	var (
		addr = "01"
		inp  = "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02"
		exp  = "000000000000000000000000ceaccac640adf55b2028469bd36ba501f28b699d"
		gas  = uint64(4000000)
	)
	precompiledBenchmark(addr, inp, exp, gas, bench)
}

func BenchmarkPrecompiledSha256(bench *testing.B) {
	var (
		addr = "02"
		inp  = "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02"
		exp  = "811c7003375852fabd0d362e40e68607a12bdabae61a7d068fe5fdd1dbbf2a5d"
		gas  = uint64(4000000)
	)
	precompiledBenchmark(addr, inp, exp, gas, bench)
}

func BenchmarkPrecompiledRipeMD(bench *testing.B) {
	var (
		addr = "03"
		inp  = "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02"
		exp  = "0000000000000000000000009215b8d9882ff46f0dfde6684d78e831467f65e6"
		gas  = uint64(4000000)
	)
	precompiledBenchmark(addr, inp, exp, gas, bench)
}

func BenchmarkPrecompiledIdentity(bench *testing.B) {
	var (
		addr = "04"
		inp  = "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02"
		exp  = "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02"
		gas  = uint64(4000000)
	)
	precompiledBenchmark(addr, inp, exp, gas, bench)
}

func BenchmarkPrecompiledModExp(bench *testing.B) {
	var (
		addr = "05"
		inp  = "00000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000002003fffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2efffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2f"
		exp  = "0000000000000000000000000000000000000000000000000000000000000001"
		gas  = uint64(4000000)
	)
	precompiledBenchmark(addr, inp, exp, gas, bench)
}

func BenchmarkPrecompiledBn256Add(bench *testing.B) {
	var (
		addr = "06"
		inp  = "0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
		exp  = "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
		gas  = uint64(4000000)
	)
	precompiledBenchmark(addr, inp, exp, gas, bench)
}

func BenchmarkPrecompiledBn256ScalarMul(bench *testing.B) {
	var (
		addr = "07"
		inp  = "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
		exp  = "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
		gas  = uint64(4000000)
	)
	precompiledBenchmark(addr, inp, exp, gas, bench)
}

func BenchmarkPrecompiledBn256Pairing(bench *testing.B) {
	var (
		addr = "08"
		inp  = "1c76476f4def4bb94541d57ebba1193381ffa7aa76ada664dd31c16024c43f593034dd2920f673e204fee2811c678745fc819b55d3e9d294e45c9b03a76aef41209dd15ebff5d46c4bd888e51a93cf99a7329636c63514396b4a452003a35bf704bf11ca01483bfa8b34b43561848d28905960114c8ac04049af4b6315a416782bb8324af6cfc93537a2ad1a445cfd0ca2a71acd7ac41fadbf933c2a51be344d120a2a4cf30c1bf9845f20c6fe39e07ea2cce61f0c9bb048165fe5e4de877550111e129f1cf1097710d41c4ac70fcdfa5ba2023c6ff1cbeac322de49d1b6df7c2032c61a830e3c17286de9462bf242fca2883585b93870a73853face6a6bf411198e9393920d483a7260bfb731fb5d25f1aa493335a9e71297e485b7aef312c21800deef121f1e76426a00665e5c4479674322d4f75edadd46debd5cd992f6ed090689d0585ff075ec9e99ad690c3395bc4b313370b38ef355acdadcd122975b12c85ea5db8c6deb4aab71808dcb408fe3d1e7690c43d37b4ce6cc0166fa7daa"
		exp  = "0000000000000000000000000000000000000000000000000000000000000001"
		gas  = uint64(4000000)
	)
	precompiledBenchmark(addr, inp, exp, gas, bench)
}

func BenchmarkOpAdd(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opAdd, x, y)
}

func BenchmarkOpSub(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opSub, x, y)
}

func BenchmarkOpMul(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opMul, x, y)
}

func BenchmarkOpDiv(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opDiv, x, y)
}

func BenchmarkOpSdiv(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opSdiv, x, y)
}

func BenchmarkOpMod(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opMod, x, y)
}

func BenchmarkOpSmod(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opSmod, x, y)
}

func BenchmarkOpExp(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opExp, x, y)
}

func BenchmarkOpSignExtend(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opSignExtend, x, y)
}

func BenchmarkOpLt(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opLt, x, y)
}

func BenchmarkOpGt(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opGt, x, y)
}

func BenchmarkOpSlt(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opSlt, x, y)
}

func BenchmarkOpSgt(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opSgt, x, y)
}

func BenchmarkOpEq(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opEq, x, y)
}

func BenchmarkOpAnd(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opAnd, x, y)
}

func BenchmarkOpOr(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opOr, x, y)
}

func BenchmarkOpXor(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opXor, x, y)
}

func BenchmarkOpByte(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opByte, x, y)
}

func BenchmarkOpAddmod(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	z := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opAddmod, x, y, z)
}

func BenchmarkOpMulmod(b *testing.B) {
	x := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	y := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"
	z := "ABCDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff"

	opBenchmark(b, opMulmod, x, y, z)
}

//func BenchmarkOpSha3(b *testing.B) {
//	x := "0"
//	y := "32"
//
//	opBenchmark(b,opSha3, x, y)
//
//
//}
