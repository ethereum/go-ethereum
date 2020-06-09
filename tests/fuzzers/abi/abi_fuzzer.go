// Copyright 2020 The go-ethereum Authors
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

// +build gofuzz

package abi

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	fuzz "github.com/google/gofuzz"
)

func unpackPack(abi abi.ABI, method string, inputType []interface{}, input []byte) bool {
	outptr := reflect.New(reflect.TypeOf(inputType))
	if err := abi.Unpack(outptr.Interface(), method, input); err == nil {
		output, err := abi.Pack(method, input)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(input, output) {
			panic(fmt.Sprintf("unpackPack is not equal, \ninput : %x\noutput: %x", input, output))
		}
		return true
	}
	return false
}

func packUnpack(abi abi.ABI, method string, input []interface{}) bool {
	if packed, err := abi.Pack(method, input); err == nil {
		outptr := reflect.New(reflect.TypeOf(input))
		err := abi.Unpack(outptr.Interface(), method, packed)
		if err != nil {
			panic(err)
		}
		out := outptr.Elem().Interface()
		if !reflect.DeepEqual(input, out) {
			panic(fmt.Sprintf("unpackPack is not equal, \ninput : %x\noutput: %x", input, out))
		}
		return true
	}
	return false
}

type args struct {
	name string
	typ  string
}

func createABI(name string, stateMutability, payable *string, inputs []args) (abi.ABI, error) {
	sig := fmt.Sprintf(`{ "type" : "function", "name" : "%v" `, name)
	if stateMutability != nil {
		sig += fmt.Sprintf(`, "stateMutability": "%v" `, *stateMutability)
	}
	if payable != nil {
		sig += fmt.Sprintf(`, "payable": %v `, *payable)
	}
	if len(inputs) > 0 {
		sig += fmt.Sprintf(`, "inputs" : [ {`)
		for i, inp := range inputs {
			sig += fmt.Sprintf(`"name" : "%v", "type" : "%v" `, inp.name, inp.typ)
			if i+1 < len(inputs) {
				sig += ","
			}
		}
		sig += "} ]"
		sig += fmt.Sprintf(`, "outputs" : [ {`)
		for i, inp := range inputs {
			sig += fmt.Sprintf(`"name" : "%v", "type" : "%v" `, inp.name, inp.typ)
			if i+1 < len(inputs) {
				sig += ","
			}
		}
		sig += "} ]"
	}
	sig += `}`

	abi, err := abi.JSON(strings.NewReader(sig))
	if err != nil {
		panic(fmt.Sprintf("err: %v, abi: %v", err.Error(), sig))
	}
	return abi, err
}

func fillStruct(structs []interface{}, data []byte) {
	fuzz.NewFromGoFuzz(data).Fuzz(&structs)
}

func createStructs(args []args) []interface{} {
	structs := make([]interface{}, len(args))
	for i, arg := range args {
		t, err := abi.NewType(arg.typ, "", nil)
		if err != nil {
			panic(err)
		}
		structs[i] = reflect.New(t.GetType()).Elem()
	}
	return structs
}

func Fuzz(input []byte) int {
	good := false

	names := []string{"", "_name", "name", "NAME", "name_", "__", "_name_", "n"}
	stateMut := []string{"", "pure", "view", "payable"}
	stateMutabilites := []*string{nil, &stateMut[0], &stateMut[1], &stateMut[2], &stateMut[3]}
	pays := []string{"", "true", "false"}
	payables := []*string{nil, &pays[0], &pays[1], &pays[2]}
	varNames := []string{"a", "b", "c", "d", "e", "f", "g"}
	varNames = append(varNames, names...)
	varTypes := []string{"bool", "address", "bytes", "string",
		"uint", "int", "uint8", "int8", "uint8", "int8", "uint16", "int16",
		"uint24", "int24", "uint32", "int32", "uint40", "int40", "uint48", "int48", "uint56", "int56",
		"uint64", "int64", "uint72", "int72", "uint80", "int80", "uint88", "int88", "uint96", "int96",
		"uint104", "int104", "uint112", "int112", "uint120", "int120", "uint128", "int128", "uint136", "int136",
		"uint144", "int144", "uint152", "int152", "uint160", "int160", "uint168", "int168", "uint176", "int176",
		"uint184", "int184", "uint192", "int192", "uint200", "int200", "uint208", "int208", "uint216", "int216",
		"uint224", "int224", "uint232", "int232", "uint240", "int240", "uint248", "int248", "uint256", "int256",
		"byte1", "byte2", "byte3", "byte4", "byte5", "byte6", "byte7", "byte8", "byte9", "byte10", "byte11",
		"byte12", "byte13", "byte14", "byte15", "byte16", "byte17", "byte18", "byte19", "byte20", "byte21",
		"byte22", "byte23", "byte24", "byte25", "byte26", "byte27", "byte28", "byte29", "byte30", "byte31", "byte32"}
	rand := rand.New(rand.NewSource(int64(input[0])))
	for _, name := range names {
		for _, stateMut := range stateMutabilites {
			for _, payable := range payables {
				var arg []args
				for i := rand.Int31n(2); i > 0; i-- {
					argName := varNames[rand.Int31n(int32(len(varNames)))]
					argTyp := varTypes[rand.Int31n(int32(len(varTypes)))]
					if rand.Int31n(10) == 0 {
						argTyp += "[]"
					} else if rand.Int31n(10) == 0 {
						arrayArgs := rand.Int31n(30)
						argTyp = fmt.Sprintf("[%d]", arrayArgs)
					}
					arg = append(arg, args{
						name: argName,
						typ:  argTyp,
					})
				}
				abi, err := createABI(name, stateMut, payable, arg)
				if err != nil {
					continue
				}
				structs := createStructs(arg)
				b := unpackPack(abi, name, structs, input)
				fillStruct(structs, input)
				c := packUnpack(abi, name, structs)
				good = good || b || c
			}
		}
	}
	if good {
		return 1
	}
	return 0
}
