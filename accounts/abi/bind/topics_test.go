// Copyright 2019 The go-ethereum Authors
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

package bind

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func TestMakeTopics(t *testing.T) {
	type args struct {
		query [][]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    [][]common.Hash
		wantErr bool
	}{
		{
			"support fixed byte types, right padded to 32 bytes",
			args{[][]interface{}{{[5]byte{1, 2, 3, 4, 5}}}},
			[][]common.Hash{{common.Hash{1, 2, 3, 4, 5}}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := makeTopics(tt.args.query...)
			if (err != nil) != tt.wantErr {
				t.Errorf("makeTopics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makeTopics() = %v, want %v", got, tt.want)
			}
		})
	}
}

type args struct {
	createObj func() interface{}
	resultObj func() interface{}
	resultMap func() map[string]interface{}
	fields    abi.Arguments
	topics    []common.Hash
}

type bytesStruct struct {
	StaticBytes [5]byte
}
type int8Struct struct {
	Int8Value int8
}
type int256Struct struct {
	Int256Value *big.Int
}

type topicTest struct {
	name    string
	args    args
	wantErr bool
}

func setupTopicsTests() []topicTest {
	bytesType, _ := abi.NewType("bytes5", "", nil)
	int8Type, _ := abi.NewType("int8", "", nil)
	int256Type, _ := abi.NewType("int256", "", nil)

	tests := []topicTest{
		{
			name: "support fixed byte types, right padded to 32 bytes",
			args: args{
				createObj: func() interface{} { return &bytesStruct{} },
				resultObj: func() interface{} { return &bytesStruct{StaticBytes: [5]byte{1, 2, 3, 4, 5}} },
				resultMap: func() map[string]interface{} {
					return map[string]interface{}{"staticBytes": [5]byte{1, 2, 3, 4, 5}}
				},
				fields: abi.Arguments{abi.Argument{
					Name:    "staticBytes",
					Type:    bytesType,
					Indexed: true,
				}},
				topics: []common.Hash{
					{1, 2, 3, 4, 5},
				},
			},
			wantErr: false,
		},
		{
			name: "int8 with negative value",
			args: args{
				createObj: func() interface{} { return &int8Struct{} },
				resultObj: func() interface{} { return &int8Struct{Int8Value: -1} },
				resultMap: func() map[string]interface{} {
					return map[string]interface{}{"int8Value": int8(-1)}
				},
				fields: abi.Arguments{abi.Argument{
					Name:    "int8Value",
					Type:    int8Type,
					Indexed: true,
				}},
				topics: []common.Hash{
					{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
						255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
				},
			},
			wantErr: false,
		},
		{
			name: "int256 with negative value",
			args: args{
				createObj: func() interface{} { return &int256Struct{} },
				resultObj: func() interface{} { return &int256Struct{Int256Value: big.NewInt(-1)} },
				resultMap: func() map[string]interface{} {
					return map[string]interface{}{"int256Value": big.NewInt(-1)}
				},
				fields: abi.Arguments{abi.Argument{
					Name:    "int256Value",
					Type:    int256Type,
					Indexed: true,
				}},
				topics: []common.Hash{
					{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
						255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
				},
			},
			wantErr: false,
		},
	}

	return tests
}

func TestParseTopics(t *testing.T) {
	tests := setupTopicsTests()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createObj := tt.args.createObj()
			if err := parseTopics(createObj, tt.args.fields, tt.args.topics); (err != nil) != tt.wantErr {
				t.Errorf("parseTopics() error = %v, wantErr %v", err, tt.wantErr)
			}
			resultObj := tt.args.resultObj()
			if !reflect.DeepEqual(createObj, resultObj) {
				t.Errorf("parseTopics() = %v, want %v", createObj, resultObj)
			}
		})
	}
}

func TestParseTopicsIntoMap(t *testing.T) {
	tests := setupTopicsTests()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outMap := make(map[string]interface{})
			if err := parseTopicsIntoMap(outMap, tt.args.fields, tt.args.topics); (err != nil) != tt.wantErr {
				t.Errorf("parseTopicsIntoMap() error = %v, wantErr %v", err, tt.wantErr)
			}
			resultMap := tt.args.resultMap()
			if !reflect.DeepEqual(outMap, resultMap) {
				t.Errorf("parseTopicsIntoMap() = %v, want %v", outMap, resultMap)
			}
		})
	}
}
