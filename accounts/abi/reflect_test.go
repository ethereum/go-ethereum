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

package abi

import (
	"reflect"
	"testing"
)

type reflectTest struct {
	name  string
	args  []string
	struc interface{}
	want  map[string]string
	err   string
}

var reflectTests = []reflectTest{
	{
		name: "OneToOneCorrespondance",
		args: []string{"fieldA"},
		struc: struct {
			FieldA int `abi:"fieldA"`
		}{},
		want: map[string]string{
			"fieldA": "FieldA",
		},
	},
	{
		name: "MissingFieldsInStruct",
		args: []string{"fieldA", "fieldB"},
		struc: struct {
			FieldA int `abi:"fieldA"`
		}{},
		want: map[string]string{
			"fieldA": "FieldA",
		},
	},
	{
		name: "MoreFieldsInStructThanArgs",
		args: []string{"fieldA"},
		struc: struct {
			FieldA int `abi:"fieldA"`
			FieldB int
		}{},
		want: map[string]string{
			"fieldA": "FieldA",
		},
	},
	{
		name: "MissingFieldInArgs",
		args: []string{"fieldA"},
		struc: struct {
			FieldA int `abi:"fieldA"`
			FieldB int `abi:"fieldB"`
		}{},
		err: "struct: abi tag 'fieldB' defined but not found in abi",
	},
	{
		name: "NoAbiDescriptor",
		args: []string{"fieldA"},
		struc: struct {
			FieldA int
		}{},
		want: map[string]string{
			"fieldA": "FieldA",
		},
	},
	{
		name: "NoArgs",
		args: []string{},
		struc: struct {
			FieldA int `abi:"fieldA"`
		}{},
		err: "struct: abi tag 'fieldA' defined but not found in abi",
	},
	{
		name: "DifferentName",
		args: []string{"fieldB"},
		struc: struct {
			FieldA int `abi:"fieldB"`
		}{},
		want: map[string]string{
			"fieldB": "FieldA",
		},
	},
	{
		name: "DifferentName",
		args: []string{"fieldB"},
		struc: struct {
			FieldA int `abi:"fieldB"`
		}{},
		want: map[string]string{
			"fieldB": "FieldA",
		},
	},
	{
		name: "MultipleFields",
		args: []string{"fieldA", "fieldB"},
		struc: struct {
			FieldA int `abi:"fieldA"`
			FieldB int `abi:"fieldB"`
		}{},
		want: map[string]string{
			"fieldA": "FieldA",
			"fieldB": "FieldB",
		},
	},
	{
		name: "MultipleFieldsABIMissing",
		args: []string{"fieldA", "fieldB"},
		struc: struct {
			FieldA int `abi:"fieldA"`
			FieldB int
		}{},
		want: map[string]string{
			"fieldA": "FieldA",
			"fieldB": "FieldB",
		},
	},
	{
		name: "NameConflict",
		args: []string{"fieldB"},
		struc: struct {
			FieldA int `abi:"fieldB"`
			FieldB int
		}{},
		err: "abi: multiple variables maps to the same abi field 'fieldB'",
	},
	{
		name: "Underscored",
		args: []string{"_"},
		struc: struct {
			FieldA int
		}{},
		err: "abi: purely underscored output cannot unpack to struct",
	},
	{
		name: "DoubleMapping",
		args: []string{"fieldB", "fieldC", "fieldA"},
		struc: struct {
			FieldA int `abi:"fieldC"`
			FieldB int
		}{},
		err: "abi: multiple outputs mapping to the same struct field 'FieldA'",
	},
	{
		name: "AlreadyMapped",
		args: []string{"fieldB", "fieldB"},
		struc: struct {
			FieldB int `abi:"fieldB"`
		}{},
		err: "struct: abi tag in 'FieldB' already mapped",
	},
}

func TestReflectNameToStruct(t *testing.T) {
	for _, test := range reflectTests {
		t.Run(test.name, func(t *testing.T) {
			m, err := mapArgNamesToStructFields(test.args, reflect.ValueOf(test.struc))
			if len(test.err) > 0 {
				if err == nil || err.Error() != test.err {
					t.Fatalf("Invalid error: expected %v, got %v", test.err, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				for fname := range test.want {
					if m[fname] != test.want[fname] {
						t.Fatalf("Incorrect value for field %s: expected %v, got %v", fname, test.want[fname], m[fname])
					}
				}
			}
		})
	}
}
