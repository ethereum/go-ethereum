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
	"encoding/json"
	"fmt"
	"io"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

// Executer is an executer method for performing state executions. It takes one
// argument which is the input data and expects output data to be returned as
// multiple 32 byte word length concatenated slice
type Executer func(datain []byte) []byte

// The ABI holds information about a contract's context and available
// invokable methods. It will allow you to type check function calls and
// packs data accordingly.
type ABI struct {
	Methods map[string]Method
}

// JSON returns a parsed ABI interface and error if it failed.
func JSON(reader io.Reader) (ABI, error) {
	dec := json.NewDecoder(reader)

	var abi ABI
	if err := dec.Decode(&abi); err != nil {
		return ABI{}, err
	}

	return abi, nil
}

// tests, tests whether the given input would result in a successful
// call. Checks argument list count and matches input to `input`.
func (abi ABI) pack(name string, args ...interface{}) ([]byte, error) {
	method := abi.Methods[name]

	var ret []byte
	for i, a := range args {
		input := method.Inputs[i]

		packed, err := input.Type.pack(a)
		if err != nil {
			return nil, fmt.Errorf("`%s` %v", name, err)
		}
		ret = append(ret, packed...)

	}

	return ret, nil
}

// Pack the given method name to conform the ABI. Method call's data
// will consist of method_id, args0, arg1, ... argN. Method id consists
// of 4 bytes and arguments are all 32 bytes.
// Method ids are created from the first 4 bytes of the hash of the
// methods string signature. (signature = baz(uint32,string32))
func (abi ABI) Pack(name string, args ...interface{}) ([]byte, error) {
	method, exist := abi.Methods[name]
	if !exist {
		return nil, fmt.Errorf("method '%s' not found", name)
	}

	// start with argument count match
	if len(args) != len(method.Inputs) {
		return nil, fmt.Errorf("argument count mismatch: %d for %d", len(args), len(method.Inputs))
	}

	arguments, err := abi.pack(name, args...)
	if err != nil {
		return nil, err
	}

	// Set function id
	packed := abi.Methods[name].Id()
	packed = append(packed, arguments...)

	return packed, nil
}

// toGoType parses the input and casts it to the proper type defined by the ABI
// argument in t.
func toGoType(t Argument, input []byte) interface{} {
	switch t.Type.T {
	case IntTy:
		return common.BytesToBig(input)
	case UintTy:
		return common.BytesToBig(input)
	case BoolTy:
		return common.BytesToBig(input).Uint64() > 0
	case AddressTy:
		return common.BytesToAddress(input)
	case HashTy:
		return common.BytesToHash(input)
	}
	return nil
}

// Call executes a call and attemps to parse the return values and returns it as
// an interface. It uses the executer method to perform the actual call since
// the abi knows nothing of the lower level calling mechanism.
//
// Call supports all abi types and includes multiple return values. When only
// one item is returned a single interface{} will be returned, if a contract
// method returns multiple values an []interface{} slice is returned.
func (abi ABI) Call(executer Executer, name string, args ...interface{}) interface{} {
	callData, err := abi.Pack(name, args...)
	if err != nil {
		glog.V(logger.Debug).Infoln("pack error:", err)
		return nil
	}

	output := executer(callData)

	method := abi.Methods[name]
	ret := make([]interface{}, int(math.Max(float64(len(method.Outputs)), float64(len(output)/32))))
	for i := 0; i < len(ret); i += 32 {
		index := i / 32
		ret[index] = toGoType(method.Outputs[index], output[i:i+32])
	}

	// return single interface
	if len(ret) == 1 {
		return ret[0]
	}

	return ret
}

func (abi *ABI) UnmarshalJSON(data []byte) error {
	var methods []Method
	if err := json.Unmarshal(data, &methods); err != nil {
		return err
	}

	abi.Methods = make(map[string]Method)
	for _, method := range methods {
		abi.Methods[method.Name] = method
	}

	return nil
}
