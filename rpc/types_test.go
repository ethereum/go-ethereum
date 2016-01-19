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
	"bytes"
	"encoding/json"
	"math/big"
	"testing"
)

func TestNewHexNumber(t *testing.T) {
	tests := []interface{}{big.NewInt(123), int64(123), uint64(123), int8(123), uint8(123)}

	for i, v := range tests {
		hn := NewHexNumber(v)
		if hn == nil {
			t.Fatalf("Unable to create hex number instance for tests[%d]", i)
		}
		if hn.Int64() != 123 {
			t.Fatalf("expected %d, got %d on value tests[%d]", 123, hn.Int64(), i)
		}
	}

	failures := []interface{}{"", nil, []byte{1, 2, 3, 4}}
	for i, v := range failures {
		hn := NewHexNumber(v)
		if hn != nil {
			t.Fatalf("Creating a nex number instance of %T should fail (failures[%d])", failures[i], i)
		}
	}
}

func TestHexNumberUnmarshalJSON(t *testing.T) {
	tests := []string{`"0x4d2"`, "1234", `"1234"`}
	for i, v := range tests {
		var hn HexNumber
		if err := json.Unmarshal([]byte(v), &hn); err != nil {
			t.Fatalf("Test %d failed - %s", i, err)
		}

		if hn.Int64() != 1234 {
			t.Fatalf("Expected %d, got %d for test[%d]", 1234, hn.Int64(), i)
		}
	}
}

func TestHexNumberMarshalJSON(t *testing.T) {
	hn := NewHexNumber(1234567890)
	got, err := json.Marshal(hn)
	if err != nil {
		t.Fatalf("Unable to marshal hex number - %s", err)
	}

	exp := []byte(`"0x499602d2"`)
	if bytes.Compare(exp, got) != 0 {
		t.Fatalf("Invalid json.Marshal, expected '%s', got '%s'", exp, got)
	}
}
