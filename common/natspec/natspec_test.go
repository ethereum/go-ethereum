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

package natspec

import (
	"testing"
)

func makeInfoDoc(desc string) []byte {
	return []byte(`
{
  "source": "contract test { }",
  "language": "Solidity",
  "compilerVersion": "1",
  "userDoc": {
    "methods": {
      "multiply(uint256)": {
        "notice":  "` + desc + `"
      },
      "balance(address)": {
        "notice": "` + "`(balanceInmGAV / 1000).fixed(0,3)`" + ` GAV is the total funds available to ` + "`who.address()`." + `"
      }
    },
    "invariants": [
      { "notice": "The sum total amount of GAV in the system is 1 million." }
    ],
    "construction": [
      { "notice": "Endows ` + "`message.caller.address()`" + ` with 1m GAV." }
    ]
  },
  "abiDefinition": [{
    "name": "multiply",
    "constant": false,
    "type": "function",
    "inputs": [{
      "name": "a",
      "type": "uint256"
    }],
    "outputs": [{
      "name": "d",
      "type": "uint256"
    }]
  }]
}`)
}

var data = "0xc6888fa1000000000000000000000000000000000000000000000000000000000000007a"

var tx = `
{
  "params": [{
      "to": "0x8521742d3f456bd237e312d6e30724960f72517a",
      "data": "0xc6888fa1000000000000000000000000000000000000000000000000000000000000007a"
  }],
}
`

func TestNotice(t *testing.T) {

	desc := "Will multiply `a` by 7 and return `a * 7`."
	expected := "Will multiply 122 by 7 and return 854."

	infodoc := makeInfoDoc(desc)
	ns, err := NewWithDocs(infodoc, tx, data)
	if err != nil {
		t.Errorf("New: error: %v", err)
		return
	}

	notice, err := ns.Notice()

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if notice != expected {
		t.Errorf("incorrect notice. expected %v, got %v", expected, notice)
	}
}

// test missing method
func TestMissingMethod(t *testing.T) {

	desc := "Will multiply `a` by 7 and return `a * 7`."
	expected := "natspec.js error evaluating expression: Natspec evaluation failed, method does not exist"

	infodoc := makeInfoDoc(desc)
	ns, err := NewWithDocs(infodoc, tx, data)
	if err != nil {
		t.Errorf("New: error: %v", err)
	}

	notice, err := ns.noticeForMethod(tx, "missing_method", "")

	if err == nil {
		t.Errorf("expected error, got nothing (notice: '%v')", notice)
	} else {
		if err.Error() != expected {
			t.Errorf("expected error '%s' got '%v' (notice: '%v')", expected, err, notice)
		}
	}
}

// test invalid desc

func TestInvalidDesc(t *testing.T) {

	desc := "Will multiply 122 by \"7\" and return 854."
	expected := "invalid character '7' after object key:value pair"

	infodoc := makeInfoDoc(desc)
	_, err := NewWithDocs(infodoc, tx, data)
	if err == nil {
		t.Errorf("expected error, got nothing", err)
	} else {
		if err.Error() != expected {
			t.Errorf("expected error '%s' got '%v'", expected, err)
		}
	}
}

// test wrong input params
func TestWrongInputParams(t *testing.T) {

	desc := "Will multiply `e` by 7 and return `a * 7`."
	expected := "natspec.js error evaluating expression: Natspec evaluation failed, wrong input params"

	infodoc := makeInfoDoc(desc)
	ns, err := NewWithDocs(infodoc, tx, data)
	if err != nil {
		t.Errorf("New: error: %v", err)
	}

	notice, err := ns.Notice()

	if err == nil {
		t.Errorf("expected error, got nothing (notice: '%v')", notice)
	} else {
		if err.Error() != expected {
			t.Errorf("expected error '%s' got '%v' (notice: '%v')", expected, err, notice)
		}
	}

}
