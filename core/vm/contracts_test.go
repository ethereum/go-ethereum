// Copyright 2017 The go-ethereum Authors
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

package vm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// precompiledTest defines the input/output pairs for precompiled contract tests.
type precompiledTest struct {
	Input, Expected string
	Name            string
	NoBenchmark     bool // Benchmark primarily the worst-cases
}

// precompiledFailureTest defines the input/error pairs for precompiled
// contract failure tests.
type precompiledFailureTest struct {
	Input         string
	expectedError error
	Name          string
}

// EIP-152 test vectors
var blake2FMalformedInputTests = []precompiledFailureTest{
	{
		Input:         "",
		expectedError: errBlake2FInvalidInputLength,
		Name:          "vector 0: empty input",
	},
	{
		Input:         "00000c48c9bdf267e6096a3ba7ca8485ae67bb2bf894fe72f36e3cf1361d5f3af54fa5d182e6ad7f520e511f6c3e2b8c68059b6bbd41fbabd9831f79217e1319cde05b61626300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000001",
		expectedError: errBlake2FInvalidInputLength,
		Name:          "vector 1: less than 213 bytes input",
	},
	{
		Input:         "000000000c48c9bdf267e6096a3ba7ca8485ae67bb2bf894fe72f36e3cf1361d5f3af54fa5d182e6ad7f520e511f6c3e2b8c68059b6bbd41fbabd9831f79217e1319cde05b61626300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000001",
		expectedError: errBlake2FInvalidInputLength,
		Name:          "vector 2: more than 213 bytes input",
	},
	{
		Input:         "0000000c48c9bdf267e6096a3ba7ca8485ae67bb2bf894fe72f36e3cf1361d5f3af54fa5d182e6ad7f520e511f6c3e2b8c68059b6bbd41fbabd9831f79217e1319cde05b61626300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000002",
		expectedError: errBlake2FInvalidFinalFlag,
		Name:          "vector 3: malformed final block indicator flag",
	},
}

func testPrecompiled(addr string, test precompiledTest, t *testing.T) {
	p := PrecompiledContractsBerlin[common.HexToAddress(addr)]
	in := common.Hex2Bytes(test.Input)
	contract := NewContract(AccountRef(common.HexToAddress("1337")),
		nil, new(big.Int), p.RequiredGas(in))
	t.Run(fmt.Sprintf("%s-Gas=%d", test.Name, contract.Gas), func(t *testing.T) {
		if res, err := RunPrecompiledContract(p, in, contract); err != nil {
			t.Error(err)
		} else if common.Bytes2Hex(res) != test.Expected {
			t.Errorf("Expected %v, got %v", test.Expected, common.Bytes2Hex(res))
		}
		// Verify that the precompile did not touch the input buffer
		exp := common.Hex2Bytes(test.Input)
		if !bytes.Equal(in, exp) {
			t.Errorf("Precompiled %v modified input data", addr)
		}
	})
}

func testPrecompiledOOG(addr string, test precompiledTest, t *testing.T) {
	p := PrecompiledContractsBerlin[common.HexToAddress(addr)]
	in := common.Hex2Bytes(test.Input)
	contract := NewContract(AccountRef(common.HexToAddress("1337")),
		nil, new(big.Int), p.RequiredGas(in)-1)
	t.Run(fmt.Sprintf("%s-Gas=%d", test.Name, contract.Gas), func(t *testing.T) {
		_, err := RunPrecompiledContract(p, in, contract)
		if err.Error() != "out of gas" {
			t.Errorf("Expected error [out of gas], got [%v]", err)
		}
		// Verify that the precompile did not touch the input buffer
		exp := common.Hex2Bytes(test.Input)
		if !bytes.Equal(in, exp) {
			t.Errorf("Precompiled %v modified input data", addr)
		}
	})
}

func testPrecompiledFailure(addr string, test precompiledFailureTest, t *testing.T) {
	p := PrecompiledContractsBerlin[common.HexToAddress(addr)]
	in := common.Hex2Bytes(test.Input)
	contract := NewContract(AccountRef(common.HexToAddress("31337")),
		nil, new(big.Int), p.RequiredGas(in))

	t.Run(test.Name, func(t *testing.T) {
		_, err := RunPrecompiledContract(p, in, contract)
		if !reflect.DeepEqual(err, test.expectedError) {
			t.Errorf("Expected error [%v], got [%v]", test.expectedError, err)
		}
		// Verify that the precompile did not touch the input buffer
		exp := common.Hex2Bytes(test.Input)
		if !bytes.Equal(in, exp) {
			t.Errorf("Precompiled %v modified input data", addr)
		}
	})
}

func benchmarkPrecompiled(addr string, test precompiledTest, bench *testing.B) {
	if test.NoBenchmark {
		return
	}
	p := PrecompiledContractsBerlin[common.HexToAddress(addr)]
	in := common.Hex2Bytes(test.Input)
	reqGas := p.RequiredGas(in)
	contract := NewContract(AccountRef(common.HexToAddress("1337")),
		nil, new(big.Int), reqGas)

	var (
		res  []byte
		err  error
		data = make([]byte, len(in))
	)

	bench.Run(fmt.Sprintf("%s-Gas=%d", test.Name, contract.Gas), func(bench *testing.B) {
		bench.ResetTimer()
		for i := 0; i < bench.N; i++ {
			contract.Gas = reqGas
			copy(data, in)
			res, err = RunPrecompiledContract(p, data, contract)
		}
		bench.StopTimer()
		//Check if it is correct
		if err != nil {
			bench.Error(err)
			return
		}
		if common.Bytes2Hex(res) != test.Expected {
			bench.Error(fmt.Sprintf("Expected %v, got %v", test.Expected, common.Bytes2Hex(res)))
			return
		}
	})
}

// Benchmarks the sample inputs from the ECRECOVER precompile.
func BenchmarkPrecompiledEcrecover(bench *testing.B) {
	t := precompiledTest{
		Input:    "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02",
		Expected: "000000000000000000000000ceaccac640adf55b2028469bd36ba501f28b699d",
		Name:     "",
	}
	benchmarkPrecompiled("01", t, bench)
}

// Benchmarks the sample inputs from the SHA256 precompile.
func BenchmarkPrecompiledSha256(bench *testing.B) {
	t := precompiledTest{
		Input:    "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02",
		Expected: "811c7003375852fabd0d362e40e68607a12bdabae61a7d068fe5fdd1dbbf2a5d",
		Name:     "128",
	}
	benchmarkPrecompiled("02", t, bench)
}

// Benchmarks the sample inputs from the RIPEMD precompile.
func BenchmarkPrecompiledRipeMD(bench *testing.B) {
	t := precompiledTest{
		Input:    "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02",
		Expected: "0000000000000000000000009215b8d9882ff46f0dfde6684d78e831467f65e6",
		Name:     "128",
	}
	benchmarkPrecompiled("03", t, bench)
}

// Benchmarks the sample inputs from the identiy precompile.
func BenchmarkPrecompiledIdentity(bench *testing.B) {
	t := precompiledTest{
		Input:    "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02",
		Expected: "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02",
		Name:     "128",
	}
	benchmarkPrecompiled("04", t, bench)
}

// Tests the sample inputs from the ModExp EIP 198.
func TestPrecompiledModExp(t *testing.T) { testJson("modexp", "05", t) }

// Benchmarks the sample inputs from the ModExp EIP 198.
func BenchmarkPrecompiledModExp(b *testing.B) { benchJson("modexp", "05", b) }

// Tests the sample inputs from the elliptic curve addition EIP 213.
func TestPrecompiledBn256Add(t *testing.T) { testJson("bn256Add", "06", t) }

// Benchmarks the sample inputs from the elliptic curve addition EIP 213.
func BenchmarkPrecompiledBn256Add(b *testing.B) { benchJson("bn256Add", "06", b) }

// Tests OOG
func TestPrecompiledModExpOOG(t *testing.T) {
	modexpTests, err := loadJson("modexp")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range modexpTests {
		testPrecompiledOOG("05", test, t)
	}
}

// Tests the sample inputs from the elliptic curve scalar multiplication EIP 213.
func TestPrecompiledBn256ScalarMul(t *testing.T) { testJson("bn256ScalarMul", "07", t) }

// Benchmarks the sample inputs from the elliptic curve scalar multiplication EIP 213.
func BenchmarkPrecompiledBn256ScalarMul(b *testing.B) { benchJson("bn256ScalarMul", "07", b) }

// Tests the sample inputs from the elliptic curve pairing check EIP 197.
func TestPrecompiledBn256Pairing(t *testing.T) { testJson("bn256Pairing", "08", t) }

// Behcnmarks the sample inputs from the elliptic curve pairing check EIP 197.
func BenchmarkPrecompiledBn256Pairing(b *testing.B) { benchJson("bn256Pairing", "08", b) }

func TestPrecompiledBlake2F(t *testing.T) { testJson("blake2F", "09", t) }

func BenchmarkPrecompiledBlake2F(b *testing.B) { benchJson("blake2F", "09", b) }

func TestPrecompileBlake2FMalformedInput(t *testing.T) {
	for _, test := range blake2FMalformedInputTests {
		testPrecompiledFailure("09", test, t)
	}
}

func TestPrecompiledEcrecover(t *testing.T) { testJson("ecRecover", "01", t) }

func testJson(name, addr string, t *testing.T) {
	tests, err := loadJson(name)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		testPrecompiled(addr, test, t)
	}
}

func benchJson(name, addr string, b *testing.B) {
	tests, err := loadJson(name)
	if err != nil {
		b.Fatal(err)
	}
	for _, test := range tests {
		benchmarkPrecompiled(addr, test, b)
	}
}

func TestPrecompiledBLS12381G1Add(t *testing.T)      { testJson("blsG1Add", "0a", t) }
func TestPrecompiledBLS12381G1Mul(t *testing.T)      { testJson("blsG1Mul", "0b", t) }
func TestPrecompiledBLS12381G1MultiExp(t *testing.T) { testJson("blsG1MultiExp", "0c", t) }
func TestPrecompiledBLS12381G2Add(t *testing.T)      { testJson("blsG2Add", "0d", t) }
func TestPrecompiledBLS12381G2Mul(t *testing.T)      { testJson("blsG2Mul", "0e", t) }
func TestPrecompiledBLS12381G2MultiExp(t *testing.T) { testJson("blsG2MultiExp", "0f", t) }
func TestPrecompiledBLS12381Pairing(t *testing.T)    { testJson("blsPairing", "10", t) }
func TestPrecompiledBLS12381MapG1(t *testing.T)      { testJson("blsMapG1", "11", t) }
func TestPrecompiledBLS12381MapG2(t *testing.T)      { testJson("blsMapG2", "12", t) }

func BenchmarkPrecompiledBLS12381G1Add(b *testing.B)      { benchJson("blsG1Add", "0a", b) }
func BenchmarkPrecompiledBLS12381G1Mul(b *testing.B)      { benchJson("blsG1Mul", "0b", b) }
func BenchmarkPrecompiledBLS12381G1MultiExp(b *testing.B) { benchJson("blsG1MultiExp", "0c", b) }
func BenchmarkPrecompiledBLS12381G2Add(b *testing.B)      { benchJson("blsG2Add", "0d", b) }
func BenchmarkPrecompiledBLS12381G2Mul(b *testing.B)      { benchJson("blsG2Mul", "0e", b) }
func BenchmarkPrecompiledBLS12381G2MultiExp(b *testing.B) { benchJson("blsG2MultiExp", "0f", b) }
func BenchmarkPrecompiledBLS12381Pairing(b *testing.B)    { benchJson("blsPairing", "10", b) }
func BenchmarkPrecompiledBLS12381MapG1(b *testing.B)      { benchJson("blsMapG1", "11", b) }
func BenchmarkPrecompiledBLS12381MapG2(b *testing.B)      { benchJson("blsMapG2", "12", b) }

func TestPrecompiledBLS12381G1AddFail(t *testing.T) {
	for _, test := range blsG1AddFailTests {
		testPrecompiledFailure("0a", test, t)
	}
}

func TestPrecompiledBLS12381G1MulFail(t *testing.T) {
	for _, test := range blsG1MulFailTests {
		testPrecompiledFailure("0b", test, t)
	}
}

func TestPrecompiledBLS12381G1MultiExpFail(t *testing.T) {
	for _, test := range blsG1MultiExpFailTests {
		testPrecompiledFailure("0c", test, t)
	}
}

func TestPrecompiledBLS12381G2AddFail(t *testing.T) {
	for _, test := range blsG2AddFailTests {
		testPrecompiledFailure("0d", test, t)
	}
}

func TestPrecompiledBLS12381G2MulFail(t *testing.T) {
	for _, test := range blsG2MulFailTests {
		testPrecompiledFailure("0e", test, t)
	}
}

func TestPrecompiledBLS12381G2MultiExpFail(t *testing.T) {
	for _, test := range blsG2MultiExpFailTests {
		testPrecompiledFailure("0f", test, t)
	}
}

func TestPrecompiledBLS12381PairingFail(t *testing.T) {
	for _, test := range blsPairingFailTests {
		testPrecompiledFailure("10", test, t)
	}
}

func TestPrecompiledBLS12381MapG1Fail(t *testing.T) {
	for _, test := range blsMapG1FailTests {
		testPrecompiledFailure("11", test, t)
	}
}

func TestPrecompiledBLS12381MapG2Fail(t *testing.T) {
	for _, test := range blsMapG2FailTests {
		testPrecompiledFailure("12", test, t)
	}
}

func loadJson(name string) ([]precompiledTest, error) {
	data, err := ioutil.ReadFile(fmt.Sprintf("testdata/precompiles/%v.json", name))
	if err != nil {
		return nil, err
	}
	var testcases []precompiledTest
	err = json.Unmarshal(data, &testcases)
	return testcases, err
}

func writeJson(name string, testcases []precompiledTest) error {
	data, err := json.MarshalIndent(testcases, "", "  ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(fmt.Sprintf("testdata/precompiles/%v.json", name), data, 0644)
	return err
}

//func TestWrite(t *testing.T) {
//err := writeJson("blsG1Add",blsG1AddTests )
//err := writeJson("blsG1Mul",blsG1MulTests )
//err := writeJson("blsG1MultiExp",blsG1MultiExpTests )
//err := writeJson("blsG2Add",blsG2AddTests )
//err = writeJson("blsG2Mul",blsG2MulTests )
//err = writeJson("blsG2MultiExp",blsG2MultiExpTests )
//err := writeJson("blsPairing",blsPairingTests )
//err := writeJson("modexp", modexpTests)
//err = writeJson("blake2F", blake2FTests)
//err = writeJson("bn256Add", bn256AddTests)
//err = writeJson("bn256Pairing", bn256PairingTests)

//err = writeJson("bn256ScalarMul", bn256ScalarMulTests)
//err = writeJson("ecRecover", ecRecoverTests)
//if err != nil {
//	t.Fatal(err)
//}
//}

var blsG1AddFailTests = []precompiledFailureTest{
	{
		Input:         "",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_g1add_empty_input",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"00000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_g1add_short_input",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"000000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_g1add_large_input",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000108b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1",
		expectedError: errBLS12381InvalidFieldElementTopBytes,
		Name:          "bls_g1add_violate_top_bytes",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"000000000000000000000000000000001a0111ea397fe69a4b1ba7b6434bacd764774b84f38512bf6730d2a0f6b0f6241eabfffeb153ffffb9feffffffffaaac" +
			"0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1",
		expectedError: errBLS12381InvalidFieldElement,
		Name:          "bls_g1add_invalid_field_element",
	},
	{
		Input: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1",
		expectedError: errBLS12381G1PointIsNotOnCurve,
		Name:          "bls_g1add_point_not_on_curve",
	},
}
var blsG1MulFailTests = []precompiledFailureTest{
	{
		Input:         "",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_g1mul_empty_input",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"00000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"0000000000000000000000000000000000000000000000000000000000000007",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_g1mul_short_input",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"000000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"0000000000000000000000000000000000000000000000000000000000000007",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_g1mul_large_input",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000108b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"0000000000000000000000000000000000000000000000000000000000000007",
		expectedError: errBLS12381InvalidFieldElementTopBytes,
		Name:          "bls_g1mul_violate_top_bytes",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"000000000000000000000000000000001a0111ea397fe69a4b1ba7b6434bacd764774b84f38512bf6730d2a0f6b0f6241eabfffeb153ffffb9feffffffffaaac" +
			"0000000000000000000000000000000000000000000000000000000000000007",
		expectedError: errBLS12381InvalidFieldElement,
		Name:          "bls_g1mul_invalid_field_element",
	},
	{
		Input: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"0000000000000000000000000000000000000000000000000000000000000001",
		expectedError: errBLS12381G1PointIsNotOnCurve,
		Name:          "bls_g1mul_point_not_on_curve",
	},
}
var blsG1MultiExpFailTests = []precompiledFailureTest{
	{
		Input:         "",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_g1multiexp_empty_input",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"00000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"0000000000000000000000000000000000000000000000000000000000000007",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_g1multiexp_short_input",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"000000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"0000000000000000000000000000000000000000000000000000000000000007",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_g1multiexp_large_input",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"000000000000000000000000000000001a0111ea397fe69a4b1ba7b6434bacd764774b84f38512bf6730d2a0f6b0f6241eabfffeb153ffffb9feffffffffaaac" +
			"0000000000000000000000000000000000000000000000000000000000000007",
		expectedError: errBLS12381InvalidFieldElement,
		Name:          "bls_g1multiexp_invalid_field_element",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000108b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"0000000000000000000000000000000000000000000000000000000000000007",
		expectedError: errBLS12381InvalidFieldElementTopBytes,
		Name:          "bls_g1multiexp_violate_top_bytes",
	},
	{
		Input: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"0000000000000000000000000000000000000000000000000000000000000001",
		expectedError: errBLS12381G1PointIsNotOnCurve,
		Name:          "bls_g1multiexp_point_not_on_curve",
	},
}
var blsG2AddFailTests = []precompiledFailureTest{
	{
		Input:         "",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_g2add_empty_input",
	},
	{
		Input: "00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"0000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be" +
			"00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_g2add_short_input",
	},
	{
		Input: "00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"00000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be" +
			"00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_g2add_large_input",
	},
	{
		Input: "00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000010606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be" +
			"00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be",
		expectedError: errBLS12381InvalidFieldElementTopBytes,
		Name:          "bls_g2add_violate_top_bytes",
	},
	{
		Input: "00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000001a0111ea397fe69a4b1ba7b6434bacd764774b84f38512bf6730d2a0f6b0f6241eabfffeb153ffffb9feffffffffaaac" +
			"00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be",
		expectedError: errBLS12381InvalidFieldElement,
		Name:          "bls_g2add_invalid_field_element",
	},
	{
		Input: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be",
		expectedError: errBLS12381G2PointIsNotOnCurve,
		Name:          "bls_g2add_point_not_on_curve",
	},
}
var blsG2MulFailTests = []precompiledFailureTest{
	{
		Input:         "",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_g2mul_empty_input",
	},
	{
		Input: "00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"0000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be" +
			"0000000000000000000000000000000000000000000000000000000000000007",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_g2mul_short_input",
	},
	{
		Input: "00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"00000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be" +
			"0000000000000000000000000000000000000000000000000000000000000007",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_g2mul_large_input",
	},
	{
		Input: "00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000010606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be" +
			"0000000000000000000000000000000000000000000000000000000000000007",
		expectedError: errBLS12381InvalidFieldElementTopBytes,
		Name:          "bls_g2mul_violate_top_bytes",
	},
	{
		Input: "00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000001a0111ea397fe69a4b1ba7b6434bacd764774b84f38512bf6730d2a0f6b0f6241eabfffeb153ffffb9feffffffffaaac" +
			"0000000000000000000000000000000000000000000000000000000000000007",
		expectedError: errBLS12381InvalidFieldElement,
		Name:          "bls_g2mul_invalid_field_element",
	},
	{
		Input: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"0000000000000000000000000000000000000000000000000000000000000001",
		expectedError: errBLS12381G2PointIsNotOnCurve,
		Name:          "bls_g2mul_point_not_on_curve",
	},
}
var blsG2MultiExpFailTests = []precompiledFailureTest{
	{
		Input:         "",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_g2multiexp_empty_input",
	},
	{
		Input: "00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"0000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be" +
			"0000000000000000000000000000000000000000000000000000000000000007",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_g2multiexp_short_input",
	},
	{
		Input: "00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"00000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be" +
			"0000000000000000000000000000000000000000000000000000000000000007",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_g2multiexp_large_input",
	},
	{
		Input: "00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000010606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be" +
			"0000000000000000000000000000000000000000000000000000000000000007",
		expectedError: errBLS12381InvalidFieldElementTopBytes,
		Name:          "bls_g2multiexp_violate_top_bytes",
	},
	{
		Input: "00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000001a0111ea397fe69a4b1ba7b6434bacd764774b84f38512bf6730d2a0f6b0f6241eabfffeb153ffffb9feffffffffaaac" +
			"0000000000000000000000000000000000000000000000000000000000000007",
		expectedError: errBLS12381InvalidFieldElement,
		Name:          "bls_g2multiexp_invalid_field_element",
	},
	{
		Input: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"0000000000000000000000000000000000000000000000000000000000000001",
		expectedError: errBLS12381G2PointIsNotOnCurve,
		Name:          "bls_g2multiexp_point_not_on_curve",
	},
}
var blsPairingFailTests = []precompiledFailureTest{
	{
		Input:         "",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_pairing_empty_input",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be" +
			"0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"00000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_pairing_extra_data",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be" +
			"0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000001a0111ea397fe69a4b1ba7b6434bacd764774b84f38512bf6730d2a0f6b0f6241eabfffeb153ffffb9feffffffffaaac",
		expectedError: errBLS12381InvalidFieldElement,
		Name:          "bls_pairing_invalid_field_element",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be" +
			"0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000010606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be",
		expectedError: errBLS12381InvalidFieldElementTopBytes,
		Name:          "bls_pairing_top_bytes",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be",
		expectedError: errBLS12381G1PointIsNotOnCurve,
		Name:          "bls_pairing_g1_not_on_curve",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be" +
			"0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001",
		expectedError: errBLS12381G2PointIsNotOnCurve,
		Name:          "bls_pairing_g2_not_on_curve",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004" +
			"000000000000000000000000000000000a989badd40d6212b33cffc3f3763e9bc760f988c9926b26da9dd85e928483446346b8ed00e1de5d5ea93e354abe706c" +
			"00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be",
		expectedError: errBLS12381G1PointSubgroup,
		Name:          "bls_pairing_g1_not_in_correct_subgroup",
	},
	{
		Input: "0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8" +
			"0000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e" +
			"000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801" +
			"000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be" +
			"0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb" +
			"0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002" +
			"00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000" +
			"00000000000000000000000000000000013a59858b6809fca4d9a3b6539246a70051a3c88899964a42bc9a69cf9acdd9dd387cfa9086b894185b9a46a402be73" +
			"0000000000000000000000000000000002d27e0ec3356299a346a09ad7dc4ef68a483c3aed53f9139d2f929a3eecebf72082e5e58c6da24ee32e03040c406d4f",
		expectedError: errBLS12381G2PointSubgroup,
		Name:          "bls_pairing_g2_not_in_correct_subgroup",
	},
}
var blsMapG1FailTests = []precompiledFailureTest{
	{
		Input:         "",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_mapg1_empty_input",
	},
	{
		Input:         "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_mapg1_short_input",
	},
	{
		Input:         "00000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		expectedError: errBLS12381InvalidFieldElementTopBytes,
		Name:          "bls_mapg1_top_bytes",
	},
	{
		Input:         "000000000000000000000000000000001a0111ea397fe69a4b1ba7b6434bacd764774b84f38512bf6730d2a0f6b0f6241eabfffeb153ffffb9feffffffffaaac",
		expectedError: errBLS12381InvalidFieldElement,
		Name:          "bls_mapg1_invalid_fq_element",
	},
}
var blsMapG2FailTests = []precompiledFailureTest{
	{
		Input:         "",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_mapg2_empty_input",
	},
	{
		Input: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		expectedError: errBLS12381InvalidInputLength,
		Name:          "bls_mapg2_short_input",
	},
	{
		Input: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000" +
			"00000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		expectedError: errBLS12381InvalidFieldElementTopBytes,
		Name:          "bls_mapg2_top_bytes",
	},
	{
		Input: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000001a0111ea397fe69a4b1ba7b6434bacd764774b84f38512bf6730d2a0f6b0f6241eabfffeb153ffffb9feffffffffaaac",
		expectedError: errBLS12381InvalidFieldElement,
		Name:          "bls_mapg2_invalid_fq_element",
	},
}
