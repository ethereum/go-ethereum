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

package abi

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestUnpackV2(t *testing.T) {
	for i, test := range unpackTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			def := fmt.Sprintf(`[{ "name" : "method", "outputs": %s}]`, test.def)
			abi, err := JSON(strings.NewReader(def))
			if err != nil {
				t.Fatalf("invalid ABI definition %s: %v", def, err)
			}
			encb, err := hex.DecodeString(test.enc)
			if err != nil {
				t.Fatalf("invalid hex: %s" + test.enc)
			}
			out, err := abi.Methods["method"].Outputs.UnpackValues(encb)

			if err != nil {
				t.Fatal(err)
			}
			if len(test.err) != 0 {
				// The new stuff doesn't have these types of errors
				return
			}
			if !reflect.DeepEqual(test.want, out[0]) {
				t.Errorf("test %d (%v) failed: expected %v, got %v", i, test.def, test.want, out[0])
			}
		})
	}
}

func TestMultiReturnWithArrayV2(t *testing.T) {
	const definition = `[{"name" : "multi", "outputs": [{"type": "uint64[3]"}, {"type": "uint64"}]}]`
	abi, err := JSON(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}
	buff := new(bytes.Buffer)
	buff.Write(common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000900000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000007"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000006"))

	out, err := abi.Methods["multi"].Outputs.UnpackValues(buff.Bytes())

	ret1Exp := [3]uint64{9, 8, 7}
	ret2Exp := uint64(6)

	if !reflect.DeepEqual(out[0], ret1Exp) {
		t.Error("array result", out[0], "!= Expected", ret1Exp)
	}
	if out[1] != ret2Exp {
		t.Error("int result", out[1], "!= Expected", ret2Exp)
	}
}

func TestUnmarshalV2(t *testing.T) {
	const definition = `[
	{ "name" : "int", "constant" : false, "outputs": [ { "type": "uint256" } ] },
	{ "name" : "bool", "constant" : false, "outputs": [ { "type": "bool" } ] },
	{ "name" : "bytes", "constant" : false, "outputs": [ { "type": "bytes" } ] },
	{ "name" : "fixed", "constant" : false, "outputs": [ { "type": "bytes32" } ] },
	{ "name" : "multi", "constant" : false, "outputs": [ { "type": "bytes" }, { "type": "bytes" } ] },
	{ "name" : "intArraySingle", "constant" : false, "outputs": [ { "type": "uint256[3]" } ] },
	{ "name" : "addressSliceSingle", "constant" : false, "outputs": [ { "type": "address[]" } ] },
	{ "name" : "addressSliceDouble", "constant" : false, "outputs": [ { "name": "a", "type": "address[]" }, { "name": "b", "type": "address[]" } ] },
	{ "name" : "mixedBytes", "constant" : true, "outputs": [ { "name": "a", "type": "bytes" }, { "name": "b", "type": "bytes32" } ] }]`

	abi, err := JSON(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}
	buff := new(bytes.Buffer)

	// marshall mixed bytes (mixedBytes)
	p0Exp := common.Hex2Bytes("01020000000000000000")
	p1Exp := common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000ddeeff")

	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000040"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000ddeeff"))
	buff.Write(common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000a"))
	buff.Write(common.Hex2Bytes("0102000000000000000000000000000000000000000000000000000000000000"))

	mixedBytes, err := abi.Methods["mixedBytes"].Outputs.UnpackValues(buff.Bytes())
	if err != nil {
		t.Error(err)
	} else {
		p0 := mixedBytes[0].([]byte)
		p1 := mixedBytes[1].([32]byte)
		if !bytes.Equal(p0, p0Exp) {
			t.Errorf("unexpected value unpacked: want %x, got %x", p0Exp, p0)
		}

		if !bytes.Equal(p1[:], p1Exp) {
			t.Errorf("unexpected value unpacked: want %x, got %x", p1Exp, p1)
		}
	}

	// marshal int
	integer, err := abi.Methods["int"].Outputs.UnpackValues(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"))
	if err != nil {
		t.Error(err)
	}
	if len(integer) == 0 {
		t.Error("Expected one integer")
	}
	intval := integer[0].(*big.Int)
	if intval == nil || intval.Cmp(big.NewInt(1)) != 0 {
		t.Error("expected Int to be 1 got", intval)
	}

	// marshal bool
	boolreturns, err := abi.Methods["bool"].Outputs.UnpackValues(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"))
	if err != nil {
		t.Error(err)
	}
	boolval := boolreturns[0].(bool)
	if !boolval {
		t.Error("expected Bool to be true")
	}

	// marshal dynamic bytes max length 32
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020"))
	bytesOut := common.RightPadBytes([]byte("hello"), 32)
	buff.Write(bytesOut)

	bytesreturns, err := abi.Methods["bytes"].Outputs.UnpackValues(buff.Bytes())

	if err != nil {
		t.Error(err)
	}
	bytesval := bytesreturns[0].([]byte)
	if !bytes.Equal(bytesval, bytesOut) {
		t.Errorf("expected %x got %x", bytesOut, bytesval)
	}

	// marshall dynamic bytes max length 64
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000040"))
	bytesOut = common.RightPadBytes([]byte("hello"), 64)
	buff.Write(bytesOut)

	bytesreturns, err = abi.Methods["bytes"].Outputs.UnpackValues(buff.Bytes())
	if err != nil {
		t.Error(err)
	}
	bytesval = bytesreturns[0].([]byte)
	if !bytes.Equal(bytesval, bytesOut) {
		t.Errorf("expected %x got %x", bytesOut, bytesval)
	}

	// marshall dynamic bytes max length 64
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020"))
	buff.Write(common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000003f"))
	bytesOut = common.RightPadBytes([]byte("hello"), 64)
	buff.Write(bytesOut)

	bytesreturns, err = abi.Methods["bytes"].Outputs.UnpackValues(buff.Bytes())
	if err != nil {
		t.Error(err)
	}
	bytesval = bytesreturns[0].([]byte)

	if !bytes.Equal(bytesval, bytesOut[:len(bytesOut)-1]) {
		t.Errorf("expected %x got %x", bytesOut[:len(bytesOut)-1], bytesval)
	}
	// marshal dynamic bytes output empty  (nil)
	bytesreturns, err = abi.Methods["bytes"].Outputs.UnpackValues(nil)
	if err == nil {
		t.Error("expected error")
	}
	// marshal dynamic bytes output empty
	buff.Reset()
	bytesreturns, err = abi.Methods["bytes"].Outputs.UnpackValues(buff.Bytes())
	if err == nil {
		t.Error("expected error")
	}

	// marshal dynamic bytes length 5
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000005"))
	buff.Write(common.RightPadBytes([]byte("hello"), 32))

	bytesreturns, err = abi.Methods["bytes"].Outputs.UnpackValues(buff.Bytes())
	if err != nil {
		t.Error(err)
	}
	bytesval = bytesreturns[0].([]byte)

	if !bytes.Equal(bytesval, []byte("hello")) {
		t.Errorf("expected %x got %x", bytesOut, bytesval)
	}

	// marshal dynamic bytes length 5
	buff.Reset()
	buff.Write(common.RightPadBytes([]byte("hello"), 32))

	hashreturns, err := abi.Methods["fixed"].Outputs.UnpackValues(buff.Bytes())
	if err != nil {
		t.Error(err)
	}
	hashval := hashreturns[0].([32]byte)

	helloHash := common.BytesToHash(common.RightPadBytes([]byte("hello"), 32))
	if common.Hash(hashval) != helloHash {
		t.Errorf("Expected %x to equal %x", hashval, helloHash)
	}

	// marshal error
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020"))

	bytesreturns, err = abi.Methods["bytes"].Outputs.UnpackValues(buff.Bytes())
	if err == nil {
		// Error abi: cannot marshal in to go slice: offset 32 would go over slice boundary (len=64)
		t.Error("expected error")
	}
	bytesreturns, err = abi.Methods["multi"].Outputs.UnpackValues(make([]byte, 64))

	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000003"))
	// marshal int array

	intArrayReturns, err := abi.Methods["intArraySingle"].Outputs.UnpackValues(buff.Bytes())
	if err != nil {
		t.Error(err)
	}
	intArray := intArrayReturns[0].([3]*big.Int)

	var testAgainstIntArray = [3]*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}

	for i, intval := range intArray {
		if intval.Cmp(testAgainstIntArray[i]) != 0 {
			t.Errorf("expected %v, got %v", testAgainstIntArray[i], intval)
		}
	}
	// marshal address slice
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020")) // offset
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // size
	buff.Write(common.Hex2Bytes("0000000000000000000000000100000000000000000000000000000000000000"))

	outAddrReturns, err := abi.Methods["addressSliceSingle"].Outputs.UnpackValues(buff.Bytes())
	if err != nil {
		t.Fatal("didn't expect error:", err)
	}
	outAddr := outAddrReturns[0].([]common.Address)
	if len(outAddr) != 1 {
		t.Fatal("expected 1 item, got", len(outAddr))
	}

	if outAddr[0] != (common.Address{1}) {
		t.Errorf("expected %x, got %x", common.Address{1}, outAddr[0])
	}

	// marshal multiple address slice
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000040")) // offset
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000080")) // offset
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // size
	buff.Write(common.Hex2Bytes("0000000000000000000000000100000000000000000000000000000000000000"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002")) // size
	buff.Write(common.Hex2Bytes("0000000000000000000000000200000000000000000000000000000000000000"))
	buff.Write(common.Hex2Bytes("0000000000000000000000000300000000000000000000000000000000000000"))

	outAddrStructReturns, err := abi.Methods["addressSliceDouble"].Outputs.UnpackValues(buff.Bytes())
	if err != nil {
		t.Fatal("didn't expect error:", err)
	}
	A := outAddrStructReturns[0].([]common.Address)
	B := outAddrStructReturns[1].([]common.Address)

	if len(A) != 1 {
		t.Fatal("expected 1 item, got", len(A))
	}

	if A[0] != (common.Address{1}) {
		t.Errorf("expected %x, got %x", common.Address{1}, A[0])
	}

	if len(B) != 2 {
		t.Fatal("expected 1 item, got", len(B))
	}

	if B[0] != (common.Address{2}) {
		t.Errorf("expected %x, got %x", common.Address{2}, B[0])
	}
	if B[1] != (common.Address{3}) {
		t.Errorf("expected %x, got %x", common.Address{3}, B[1])
	}

	// marshal invalid address slice
	buff.Reset()
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000100"))

	err = abi.Unpack(&outAddr, "addressSliceSingle", buff.Bytes())
	_, err = abi.Methods["addressSliceSingle"].Outputs.UnpackValues(buff.Bytes())
	if err == nil {
		t.Fatal("expected error:", err)
	}

}
