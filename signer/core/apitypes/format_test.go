// Copyright 2026 The go-ethereum Authors
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

package apitypes

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

const arrayTypedData = `{
	"types": {
		"EIP712Domain": [{"name": "name", "type": "string"}, {"name": "chainId", "type": "uint256"}],
		"Batch": [
			{"name": "transfers", "type": "Transfer[]"},
			{"name": "groups", "type": "Transfer[][]"},
			{"name": "owners", "type": "address[]"},
			{"name": "amounts", "type": "uint256[2]"}
		],
		"Transfer": [{"name": "to", "type": "address"}, {"name": "amount", "type": "uint256"}]
	},
	"primaryType": "Batch",
	"domain": {"name": "Test", "chainId": "1"},
	"message": {
		"transfers": [
			{"to": "0x4444444444444444444444444444444444444444", "amount": "10000000000000000000"},
			{"to": "0x2222222222222222222222222222222222222222", "amount": "1"}
		],
		"groups": [
			[{"to": "0x5555555555555555555555555555555555555555", "amount": "5"}],
			[{"to": "0x6666666666666666666666666666666666666666", "amount": "6"}, {"to": "0x7777777777777777777777777777777777777777", "amount": "7"}]
		],
		"owners": ["0x8888888888888888888888888888888888888888", "0x9999999999999999999999999999999999999999"],
		"amounts": ["11", "12"]
	}
}`

// Every element of an array goes into the signed hash, so every element has to
// be shown too. Arrays of structs used to show only their last element.
func TestFormatShowsEveryArrayElement(t *testing.T) {
	var td TypedData
	if err := json.Unmarshal([]byte(arrayTypedData), &td); err != nil {
		t.Fatal(err)
	}
	if _, _, err := TypedDataAndHash(td); err != nil {
		t.Fatalf("typed data doesn't hash: %v", err)
	}
	nvts, err := td.Format()
	if err != nil {
		t.Fatalf("format: %v", err)
	}
	batch := nvts[1].Value.([]*NameValueType)

	// The display lists each element under its index, in order.
	transfers := batch[0].Value.([]*NameValueType)
	if len(transfers) != 2 {
		t.Fatalf("transfers shows %d elements, want 2", len(transfers))
	}
	for i, want := range []string{"0x4444444444444444444444444444444444444444", "0x2222222222222222222222222222222222222222"} {
		to := transfers[i].Value.([]*NameValueType)[0].Value
		if transfers[i].Name != fmt.Sprintf("[%d]", i) || transfers[i].Typ != "Transfer" || to != want {
			t.Errorf("transfers[%d] = %s %s to %v, want [%d] Transfer to %s", i, transfers[i].Name, transfers[i].Typ, to, i, want)
		}
	}
	// And every signed value ends up in the printed prompt.
	var out strings.Builder
	for _, nvt := range nvts {
		out.WriteString(nvt.Pprint(0))
	}
	for _, want := range []string{
		"0x4444444444444444444444444444444444444444", "10000000000000000000",
		"0x2222222222222222222222222222222222222222",
		"0x5555555555555555555555555555555555555555", "0x6666666666666666666666666666666666666666", "0x7777777777777777777777777777777777777777",
		"0x8888888888888888888888888888888888888888", "0x9999999999999999999999999999999999999999",
		"11 (0xb)", "12 (0xc)",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("prompt doesn't show %s:\n%s", want, out.String())
		}
	}
}
