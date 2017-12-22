// Copyright 2017 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"regexp"
	"reflect"
)

type decodedArgument struct {
	soltype abi.Argument
	value   interface{}
}
type decodedCallData struct {
	signature string
	name      string
	inputs    []decodedArgument
}

// String implements stringer interface, tries to use the underlying value-type
func (arg decodedArgument) String() string {
	var value string
	switch arg.value.(type) {
	case fmt.Stringer:
		value = arg.value.(fmt.Stringer).String()
	default:
		value = fmt.Sprintf("%v", arg.value)
	}
	return fmt.Sprintf("%v: %v", arg.soltype.Type.String(), value)
}

// String implements stringer interface for decodedCallData
func (cd decodedCallData) String() string {
	args := make([]string, len(cd.inputs))
	for i, arg := range cd.inputs {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", cd.name, strings.Join(args, ","))
}

// parseCallData matches the provided call data against the abi definition,
// and returns a struct containing the actual go-typed values
func parseCallData(calldata []byte, abidata string) (*decodedCallData, error) {

	if len(calldata) < 4 {
		return nil, fmt.Errorf("Invalid ABI-data, incomplete method signature of (%d bytes)", len(calldata))
	}

	sigdata, argdata := calldata[:4], calldata[4:]
	if len(argdata)%32 != 0 {
		return nil, fmt.Errorf("Not ABI-encoded data; length should be a multiple of 32 (was %d)", len(argdata))
	}

	abispec, err := abi.JSON(strings.NewReader(abidata))
	if err != nil {
		return nil, fmt.Errorf("Failed parsing JSON ABI: %v, abidata: %v", err, abidata)
	}

	method := abispec.MethodById(sigdata)
	if method == nil {
		return nil, fmt.Errorf("Supplied ABI spec does not contain method signature in data: 0x%x", sigdata)
	}
	var v interface{}
	method.Inputs.Unpack(v, argdata)

	ref := reflect.ValueOf(v)
	values := make([]interface{}, ref.NumField())

	for i := 0; i < ref.NumField(); i++ {
		values[i] = ref.Field(i).Interface()
	}
	fmt.Println(values)



	decoded := decodedCallData{signature: method.Sig(), name: method.Name}

/*
	for n, argument := range method.Inputs {
		if err != nil {
			return nil, fmt.Errorf("Failed to decode argument %d (signature %v): %v", n, method.Sig(), err)
		} else {
			decodedArg := decodedArgument{
				soltype: argument,
				value: reflect.ValueOf(v,) ,
			}
			decoded.inputs = append(decoded.inputs, decodedArg)
		}
	}

	// We're finished decoding the data. At this point, we encode the decoded data to see if it matches with the
	// original data. If we didn't do that, it would e.g. be possible to stuff extra data into the arguments, which
	// is not detected by merely decoding the data.


	var (
		encoded          []byte
	)
	encoded, err = method.Inputs.Pack(v)


	if !bytes.Equal(encoded, calldata) {
		exp := common.Bytes2Hex(encoded)
		was := common.Bytes2Hex(calldata)
		return nil, fmt.Errorf("WARNING: Supplied data is stuffed with extra data. \nWant %s\nHave %s\nfor method %v", exp, was, method.Sig())
	}
*/
	return &decoded, nil
}

// MethodSelectorToAbi converts a method selector into an ABI struct. The returned data is a valid json string
// which can be consumed by the standard abi package.
func MethodSelectorToAbi(selector string) ([]byte, error) {

	re := regexp.MustCompile("^([^\\)]+)\\(([a-z0-9,\\[\\]]*)\\)")

	type fakeArg struct {
		Type string `json:"type"`
	}
	type fakeABI struct {
		Name   string    `json:"name"`
		Type   string    `json:"type"`
		Inputs []fakeArg `json:"inputs"`
	}
	groups := re.FindStringSubmatch(selector)
	if len(groups) != 3 {
		return nil, fmt.Errorf("Did not match: %v (%v matches)", selector, len(groups))
	}
	name := groups[1]
	args := groups[2]
	arguments := make([]fakeArg, 0)
	if len(args) > 0 {
		for _, arg := range strings.Split(args, ",") {
			arguments = append(arguments, fakeArg{arg})
		}
	}
	abicheat := fakeABI{
		name, "function", arguments,
	}
	return json.Marshal([]fakeABI{abicheat})

}

type abiDb struct {
	db map[string]string
}

// NewAbiDBFromFile loads signature database from file, and
// errors if the file is not valid json. Does no other validation of contents
func NewAbiDBFromFile(path string) (*abiDb, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	db := new(abiDb)
	json.Unmarshal(raw, &db.db)
	return db, nil
}

// LookupMethodSelector checks the given 4byte-sequence against the known ABI methods.
// OBS: This method does not validate the match, it's assumed the caller will do so
func (db *abiDb) LookupMethodSelector(id []byte) (string, error) {
	if len(id) != 4 {
		return "", fmt.Errorf("Expected 4-byte id, got %d", len(id))
	}
	sig := common.ToHex(id)
	if key, exists := db.db[sig]; exists {
		return key, nil
	}
	return "", fmt.Errorf("Signature %v not found", sig)
}
func (db *abiDb) Size() int {
	return len(db.db)
}
