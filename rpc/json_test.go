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

package rpc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"reflect"
	"strconv"
	"testing"
)

type RWC struct {
	*bufio.ReadWriter
}

func (rwc *RWC) Close() error {
	return nil
}

func TestJSONRequestParsing(t *testing.T) {
	server := NewServer()
	service := new(Service)

	if err := server.RegisterName("calc", service); err != nil {
		t.Fatalf("%v", err)
	}

	req := bytes.NewBufferString(`{"id": 1234, "jsonrpc": "2.0", "method": "calc_add", "params": [11, 22]}`)
	var str string
	reply := bytes.NewBufferString(str)
	rw := &RWC{bufio.NewReadWriter(bufio.NewReader(req), bufio.NewWriter(reply))}

	codec := NewJSONCodec(rw)

	requests, batch, err := codec.ReadRequestHeaders()
	if err != nil {
		t.Fatalf("%v", err)
	}

	if batch {
		t.Fatalf("Request isn't a batch")
	}

	if len(requests) != 1 {
		t.Fatalf("Expected 1 request but got %d requests - %v", len(requests), requests)
	}

	if requests[0].service != "calc" {
		t.Fatalf("Expected service 'calc' but got '%s'", requests[0].service)
	}

	if requests[0].method != "add" {
		t.Fatalf("Expected method 'Add' but got '%s'", requests[0].method)
	}

	if rawId, ok := requests[0].id.(*json.RawMessage); ok {
		id, e := strconv.ParseInt(string(*rawId), 0, 64)
		if e != nil {
			t.Fatalf("%v", e)
		}
		if id != 1234 {
			t.Fatalf("Expected id 1234 but got %s", id)
		}
	} else {
		t.Fatalf("invalid request, expected *json.RawMesage got %T", requests[0].id)
	}

	var arg int
	args := []reflect.Type{reflect.TypeOf(arg), reflect.TypeOf(arg)}

	v, err := codec.ParseRequestArguments(args, requests[0].params)
	if err != nil {
		t.Fatalf("%v", err)
	}

	if len(v) != 2 {
		t.Fatalf("Expected 2 argument values, got %d", len(v))
	}

	if v[0].Int() != 11 || v[1].Int() != 22 {
		t.Fatalf("expected %d == 11 && %d == 22", v[0].Int(), v[1].Int())
	}
}

func TestJSONRequestParamsParsing(t *testing.T) {

	var (
		stringT = reflect.TypeOf("")
		intT    = reflect.TypeOf(0)
		intPtrT = reflect.TypeOf(new(int))

		stringV = reflect.ValueOf("abc")
		i       = 1
		intV    = reflect.ValueOf(i)
		intPtrV = reflect.ValueOf(&i)
	)

	var validTests = []struct {
		input    string
		argTypes []reflect.Type
		expected []reflect.Value
	}{
		{`[]`, []reflect.Type{}, []reflect.Value{}},
		{`[]`, []reflect.Type{intPtrT}, []reflect.Value{intPtrV}},
		{`[1]`, []reflect.Type{intT}, []reflect.Value{intV}},
		{`[1,"abc"]`, []reflect.Type{intT, stringT}, []reflect.Value{intV, stringV}},
		{`[null]`, []reflect.Type{intPtrT}, []reflect.Value{intPtrV}},
		{`[null,"abc"]`, []reflect.Type{intPtrT, stringT, intPtrT}, []reflect.Value{intPtrV, stringV, intPtrV}},
		{`[null,"abc",null]`, []reflect.Type{intPtrT, stringT, intPtrT}, []reflect.Value{intPtrV, stringV, intPtrV}},
	}

	codec := jsonCodec{}

	for _, test := range validTests {
		params := (json.RawMessage)([]byte(test.input))
		args, err := codec.ParseRequestArguments(test.argTypes, params)

		if err != nil {
			t.Fatal(err)
		}

		var match []interface{}
		json.Unmarshal([]byte(test.input), &match)

		if len(args) != len(test.argTypes) {
			t.Fatalf("expected %d parsed args, got %d", len(test.argTypes), len(args))
		}

		for i, arg := range args {
			expected := test.expected[i]

			if arg.Kind() != expected.Kind() {
				t.Errorf("expected type for param %d in %s", i, test.input)
			}

			if arg.Kind() == reflect.Int && arg.Int() != expected.Int() {
				t.Errorf("expected int(%d), got int(%d) in %s", expected.Int(), arg.Int(), test.input)
			}

			if arg.Kind() == reflect.String && arg.String() != expected.String() {
				t.Errorf("expected string(%s), got string(%s) in %s", expected.String(), arg.String(), test.input)
			}
		}
	}

	var invalidTests = []struct {
		input    string
		argTypes []reflect.Type
	}{
		{`[]`, []reflect.Type{intT}},
		{`[null]`, []reflect.Type{intT}},
		{`[1]`, []reflect.Type{stringT}},
		{`[1,2]`, []reflect.Type{stringT}},
		{`["abc", null]`, []reflect.Type{stringT, intT}},
	}

	for i, test := range invalidTests {
		if _, err := codec.ParseRequestArguments(test.argTypes, test.input); err == nil {
			t.Errorf("expected test %d - %s to fail", i, test.input)
		}
	}
}
