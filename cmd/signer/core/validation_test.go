// Copyright 2018 The go-ethereum Authors
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

package core

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func hexAddr(a string) common.Address { return common.BytesToAddress(common.FromHex(a)) }
func mixAddr(a string) (*common.MixedcaseAddress, error) {
	return common.NewMixedcaseAddressFromString(a)
}
func toHexBig(h string) hexutil.Big {
	b := big.NewInt(0).SetBytes(common.FromHex(h))
	return hexutil.Big(*b)
}
func toHexUint(h string) hexutil.Uint64 {
	b := big.NewInt(0).SetBytes(common.FromHex(h))
	return hexutil.Uint64(b.Uint64())
}
func dummyTxArgs(t txtestcase) *SendTxArgs {
	to, _ := mixAddr(t.to)
	from, _ := mixAddr(t.from)
	n := toHexUint(t.n)
	gas := toHexUint(t.g)
	gasPrice := toHexBig(t.gp)
	value := toHexBig(t.value)
	var (
		data, input *hexutil.Bytes
	)
	if t.d != "" {
		a := hexutil.Bytes(common.FromHex(t.d))
		data = &a
	}
	if t.i != "" {
		a := hexutil.Bytes(common.FromHex(t.i))
		input = &a

	}
	return &SendTxArgs{
		From:     *from,
		To:       to,
		Value:    value,
		Nonce:    n,
		GasPrice: gasPrice,
		Gas:      gas,
		Data:     data,
		Input:    input,
	}
}

type txtestcase struct {
	from, to, n, g, gp, value, d, i string
	expectErr                       bool
	numMessages                     int
}

func TestValidator(t *testing.T) {
	var (
		// use empty db, there are other tests for the abi-specific stuff
		db, _ = NewEmptyAbiDB()
		v     = NewValidator(db)
	)
	testcases := []txtestcase{
		// Invalid to checksum
		{from: "000000000000000000000000000000000000dead", to: "000000000000000000000000000000000000dead",
			n: "0x01", g: "0x20", gp: "0x40", value: "0x01", numMessages: 1},
		// valid 0x000000000000000000000000000000000000dEaD
		{from: "000000000000000000000000000000000000dead", to: "0x000000000000000000000000000000000000dEaD",
			n: "0x01", g: "0x20", gp: "0x40", value: "0x01", numMessages: 0},
		// conflicting input and data
		{from: "000000000000000000000000000000000000dead", to: "0x000000000000000000000000000000000000dEaD",
			n: "0x01", g: "0x20", gp: "0x40", value: "0x01", d: "0x01", i: "0x02", expectErr: true},
		// Data can't be parsed
		{from: "000000000000000000000000000000000000dead", to: "0x000000000000000000000000000000000000dEaD",
			n: "0x01", g: "0x20", gp: "0x40", value: "0x01", d: "0x0102", numMessages: 1},
		// Data (on Input) can't be parsed
		{from: "000000000000000000000000000000000000dead", to: "0x000000000000000000000000000000000000dEaD",
			n: "0x01", g: "0x20", gp: "0x40", value: "0x01", i: "0x0102", numMessages: 1},
		// Send to 0
		{from: "000000000000000000000000000000000000dead", to: "0x0000000000000000000000000000000000000000",
			n: "0x01", g: "0x20", gp: "0x40", value: "0x01", numMessages: 1},
		// Create empty contract (no value)
		{from: "000000000000000000000000000000000000dead", to: "",
			n: "0x01", g: "0x20", gp: "0x40", value: "0x00", numMessages: 1},
		// Create empty contract (with value)
		{from: "000000000000000000000000000000000000dead", to: "",
			n: "0x01", g: "0x20", gp: "0x40", value: "0x01", expectErr: true},
		// Small payload for create
		{from: "000000000000000000000000000000000000dead", to: "",
			n: "0x01", g: "0x20", gp: "0x40", value: "0x01", d: "0x01", numMessages: 1},
	}
	for i, test := range testcases {
		msgs, err := v.ValidateTransaction(dummyTxArgs(test), nil)
		if err == nil && test.expectErr {
			t.Errorf("Test %d, expected error", i)
			for _, msg := range msgs.Messages {
				fmt.Printf("* %s: %s\n", msg.Typ, msg.Message)
			}
		}
		if err != nil && !test.expectErr {
			t.Errorf("Test %d, unexpected error: %v", i, err)
		}
		if err == nil {
			got := len(msgs.Messages)
			if got != test.numMessages {
				for _, msg := range msgs.Messages {
					fmt.Printf("* %s: %s\n", msg.Typ, msg.Message)
				}
				t.Errorf("Test %d, expected %d messages, got %d", i, test.numMessages, got)
			} else {
				//Debug printout, remove later
				for _, msg := range msgs.Messages {
					fmt.Printf("* [%d] %s: %s\n", i, msg.Typ, msg.Message)
				}
				fmt.Println()
			}
		}
	}
}
