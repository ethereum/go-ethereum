// Copyright 2018 The go-ethereum Authors
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

package feed

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// KV mocks a key value store
type KV map[string]string

func (kv KV) Get(key string) string {
	return kv[key]
}
func (kv KV) Set(key, value string) {
	kv[key] = value
}

func compareByteSliceToExpectedHex(t *testing.T, variableName string, actualValue []byte, expectedHex string) {
	if hexutil.Encode(actualValue) != expectedHex {
		t.Fatalf("%s: Expected %s to be %s, got %s", t.Name(), variableName, expectedHex, hexutil.Encode(actualValue))
	}
}

func testBinarySerializerRecovery(t *testing.T, bin binarySerializer, expectedHex string) {
	name := reflect.TypeOf(bin).Elem().Name()
	serialized := make([]byte, bin.binaryLength())
	if err := bin.binaryPut(serialized); err != nil {
		t.Fatalf("%s.binaryPut error when trying to serialize structure: %s", name, err)
	}

	compareByteSliceToExpectedHex(t, name, serialized, expectedHex)

	recovered := reflect.New(reflect.TypeOf(bin).Elem()).Interface().(binarySerializer)
	if err := recovered.binaryGet(serialized); err != nil {
		t.Fatalf("%s.binaryGet error when trying to deserialize structure: %s", name, err)
	}

	if !reflect.DeepEqual(bin, recovered) {
		t.Fatalf("Expected that the recovered %s equals the marshalled %s", name, name)
	}

	serializedWrongLength := make([]byte, 1)
	copy(serializedWrongLength[:], serialized)
	if err := recovered.binaryGet(serializedWrongLength); err == nil {
		t.Fatalf("Expected %s.binaryGet to fail since data is too small", name)
	}
}

func testBinarySerializerLengthCheck(t *testing.T, bin binarySerializer) {
	name := reflect.TypeOf(bin).Elem().Name()
	// make a slice that is too small to contain the metadata
	serialized := make([]byte, bin.binaryLength()-1)

	if err := bin.binaryPut(serialized); err == nil {
		t.Fatalf("Expected %s.binaryPut to fail, since target slice is too small", name)
	}
}

func testValueSerializer(t *testing.T, v valueSerializer, expected KV) {
	name := reflect.TypeOf(v).Elem().Name()
	kv := make(KV)

	v.AppendValues(kv)
	if !reflect.DeepEqual(expected, kv) {
		expj, _ := json.Marshal(expected)
		gotj, _ := json.Marshal(kv)
		t.Fatalf("Expected %s.AppendValues to return %s, got %s", name, string(expj), string(gotj))
	}

	recovered := reflect.New(reflect.TypeOf(v).Elem()).Interface().(valueSerializer)
	err := recovered.FromValues(kv)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(recovered, v) {
		t.Fatalf("Expected recovered %s to be the same", name)
	}
}
