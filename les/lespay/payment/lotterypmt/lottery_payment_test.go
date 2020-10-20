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

package lotterypmt

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestSchema(t *testing.T) {
	var cases = []struct {
		schema    *LotteryPaymentSchema
		entries   map[string]common.Address
		sender    bool
		expectErr bool
	}{
		{
			// Sender schema
			&LotteryPaymentSchema{
				Sender:   common.HexToAddress("deadbeef"),
				Contract: common.HexToAddress("cafebabe"),
			},
			map[string]common.Address{
				"Sender":   common.HexToAddress("deadbeef"),
				"Contract": common.HexToAddress("cafebabe"),
			},
			false, false,
		},
		{
			// Receiver schema
			&LotteryPaymentSchema{
				Receiver: common.HexToAddress("deadbeef"),
				Contract: common.HexToAddress("cafebabe"),
			},
			map[string]common.Address{
				"Receiver": common.HexToAddress("deadbeef"),
				"Contract": common.HexToAddress("cafebabe"),
			},
			true, false,
		},
		{
			// Invalid schema
			&LotteryPaymentSchema{
				Receiver: common.HexToAddress("deadbeef"),
			},
			map[string]common.Address{
				"Receiver": common.HexToAddress("deadbeef"),
			}, false, true,
		},
		{
			// Invalid schema
			&LotteryPaymentSchema{
				Sender:   common.HexToAddress("deadbeef"),
				Contract: common.HexToAddress("cafebabe"),
			}, nil, true, true,
		},
	}
	for i, c := range cases {
		blob, err := rlp.EncodeToBytes(c.schema)
		if err != nil {
			t.Fatalf("Failed to encode schema: %v", err)
		}
		schema, err := ResolveSchema(blob, common.HexToAddress("cafebabe"), c.sender)
		if c.expectErr {
			if err == nil {
				t.Fatalf("Expect error, but nil")
			}
			continue
		}
		if err != nil {
			t.Fatalf("Case %d, expect no error, but got: %v", i, err)
		}
		for key, value := range c.entries {
			got, _ := schema.Load(key)
			if !reflect.DeepEqual(got.(common.Address), value) {
				t.Fatalf("Case %d, field mismatch, want: %v, got: %v", i, value, got)
			}
		}
	}
}
