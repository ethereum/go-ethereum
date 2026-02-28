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
	"encoding/json"
	"math"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestBlockNumberJSONUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		mustFail bool
		expected BlockNumber
	}{
		0:  {`"0x"`, true, BlockNumber(0)},
		1:  {`"0x0"`, false, BlockNumber(0)},
		2:  {`"0X1"`, false, BlockNumber(1)},
		3:  {`"0x00"`, true, BlockNumber(0)},
		4:  {`"0x01"`, true, BlockNumber(0)},
		5:  {`"0x1"`, false, BlockNumber(1)},
		6:  {`"0x12"`, false, BlockNumber(18)},
		7:  {`"0x7fffffffffffffff"`, false, BlockNumber(math.MaxInt64)},
		8:  {`"0x8000000000000000"`, true, BlockNumber(0)},
		9:  {"0", true, BlockNumber(0)},
		10: {`"ff"`, true, BlockNumber(0)},
		11: {`"pending"`, false, PendingBlockNumber},
		12: {`"latest"`, false, LatestBlockNumber},
		13: {`"earliest"`, false, EarliestBlockNumber},
		14: {`"safe"`, false, SafeBlockNumber},
		15: {`"finalized"`, false, FinalizedBlockNumber},
		16: {`someString`, true, BlockNumber(0)},
		17: {`""`, true, BlockNumber(0)},
		18: {``, true, BlockNumber(0)},
	}

	for i, test := range tests {
		var num BlockNumber
		err := json.Unmarshal([]byte(test.input), &num)
		if test.mustFail && err == nil {
			t.Errorf("Test %d should fail", i)
			continue
		}
		if !test.mustFail && err != nil {
			t.Errorf("Test %d should pass but got err: %v", i, err)
			continue
		}
		if num != test.expected {
			t.Errorf("Test %d got unexpected value, want %d, got %d", i, test.expected, num)
		}
	}
}

func TestBlockNumberOrHash_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		mustFail bool
		expected BlockNumberOrHash
	}{
		0:  {`"0x"`, true, BlockNumberOrHash{}},
		1:  {`"0x0"`, false, BlockNumberOrHashWithNumber(0)},
		2:  {`"0X1"`, false, BlockNumberOrHashWithNumber(1)},
		3:  {`"0x00"`, true, BlockNumberOrHash{}},
		4:  {`"0x01"`, true, BlockNumberOrHash{}},
		5:  {`"0x1"`, false, BlockNumberOrHashWithNumber(1)},
		6:  {`"0x12"`, false, BlockNumberOrHashWithNumber(18)},
		7:  {`"0x7fffffffffffffff"`, false, BlockNumberOrHashWithNumber(math.MaxInt64)},
		8:  {`"0x8000000000000000"`, true, BlockNumberOrHash{}},
		9:  {"0", true, BlockNumberOrHash{}},
		10: {`"ff"`, true, BlockNumberOrHash{}},
		11: {`"pending"`, false, BlockNumberOrHashWithNumber(PendingBlockNumber)},
		12: {`"latest"`, false, BlockNumberOrHashWithNumber(LatestBlockNumber)},
		13: {`"earliest"`, false, BlockNumberOrHashWithNumber(EarliestBlockNumber)},
		14: {`"safe"`, false, BlockNumberOrHashWithNumber(SafeBlockNumber)},
		15: {`"finalized"`, false, BlockNumberOrHashWithNumber(FinalizedBlockNumber)},
		16: {`someString`, true, BlockNumberOrHash{}},
		17: {`""`, true, BlockNumberOrHash{}},
		18: {``, true, BlockNumberOrHash{}},
		19: {`"0x0000000000000000000000000000000000000000000000000000000000000000"`, false, BlockNumberOrHashWithHash(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"), false)},
		20: {`{"blockHash":"0x0000000000000000000000000000000000000000000000000000000000000000"}`, false, BlockNumberOrHashWithHash(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"), false)},
		21: {`{"blockHash":"0x0000000000000000000000000000000000000000000000000000000000000000","requireCanonical":false}`, false, BlockNumberOrHashWithHash(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"), false)},
		22: {`{"blockHash":"0x0000000000000000000000000000000000000000000000000000000000000000","requireCanonical":true}`, false, BlockNumberOrHashWithHash(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"), true)},
		23: {`{"blockNumber":"0x1"}`, false, BlockNumberOrHashWithNumber(1)},
		24: {`{"blockNumber":"pending"}`, false, BlockNumberOrHashWithNumber(PendingBlockNumber)},
		25: {`{"blockNumber":"latest"}`, false, BlockNumberOrHashWithNumber(LatestBlockNumber)},
		26: {`{"blockNumber":"earliest"}`, false, BlockNumberOrHashWithNumber(EarliestBlockNumber)},
		27: {`{"blockNumber":"safe"}`, false, BlockNumberOrHashWithNumber(SafeBlockNumber)},
		28: {`{"blockNumber":"finalized"}`, false, BlockNumberOrHashWithNumber(FinalizedBlockNumber)},
		29: {`{"blockNumber":"0x1", "blockHash":"0x0000000000000000000000000000000000000000000000000000000000000000"}`, true, BlockNumberOrHash{}},
	}

	for i, test := range tests {
		var bnh BlockNumberOrHash
		err := json.Unmarshal([]byte(test.input), &bnh)
		if test.mustFail && err == nil {
			t.Errorf("Test %d should fail", i)
			continue
		}
		if !test.mustFail && err != nil {
			t.Errorf("Test %d should pass but got err: %v", i, err)
			continue
		}
		hash, hashOk := bnh.Hash()
		expectedHash, expectedHashOk := test.expected.Hash()
		num, numOk := bnh.Number()
		expectedNum, expectedNumOk := test.expected.Number()
		if bnh.RequireCanonical != test.expected.RequireCanonical ||
			hash != expectedHash || hashOk != expectedHashOk ||
			num != expectedNum || numOk != expectedNumOk {
			t.Errorf("Test %d got unexpected value, want %v, got %v", i, test.expected, bnh)
		}
	}
}

func TestBlockNumberOrHash_WithNumber_MarshalAndUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		number int64
	}{
		{"max", math.MaxInt64},
		{"pending", int64(PendingBlockNumber)},
		{"latest", int64(LatestBlockNumber)},
		{"earliest", int64(EarliestBlockNumber)},
		{"safe", int64(SafeBlockNumber)},
		{"finalized", int64(FinalizedBlockNumber)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			bnh := BlockNumberOrHashWithNumber(BlockNumber(test.number))
			marshalled, err := json.Marshal(bnh)
			if err != nil {
				t.Fatal("cannot marshal:", err)
			}
			var unmarshalled BlockNumberOrHash
			err = json.Unmarshal(marshalled, &unmarshalled)
			if err != nil {
				t.Fatal("cannot unmarshal:", err)
			}
			if !reflect.DeepEqual(bnh, unmarshalled) {
				t.Fatalf("wrong result: expected %v, got %v", bnh, unmarshalled)
			}
		})
	}
}

func TestBlockNumberOrHash_StringAndUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []BlockNumberOrHash{
		BlockNumberOrHashWithNumber(math.MaxInt64),
		BlockNumberOrHashWithNumber(PendingBlockNumber),
		BlockNumberOrHashWithNumber(LatestBlockNumber),
		BlockNumberOrHashWithNumber(EarliestBlockNumber),
		BlockNumberOrHashWithNumber(SafeBlockNumber),
		BlockNumberOrHashWithNumber(FinalizedBlockNumber),
		BlockNumberOrHashWithNumber(32),
		BlockNumberOrHashWithHash(common.Hash{0xaa}, false),
	}
	for _, want := range tests {
		marshalled, _ := json.Marshal(want.String())
		var have BlockNumberOrHash
		if err := json.Unmarshal(marshalled, &have); err != nil {
			t.Fatalf("cannot unmarshal (%v): %v", string(marshalled), err)
		}
		if !reflect.DeepEqual(want, have) {
			t.Fatalf("wrong result: have %v, want %v", have, want)
		}
	}
}
