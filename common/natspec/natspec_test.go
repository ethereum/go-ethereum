package natspec

import (
	"testing"
)

func makeUserdoc(desc string) []byte {
	return []byte(`
{
  "source": "...",
  "language": "Solidity",
  "languageVersion": 1,
  "methods": {
    "multiply(uint256)": {
      "notice":  ` + desc + `
    },
    "balance(address)": {
      "notice": "` + "`(balanceInmGAV / 1000).fixed(0,3)`" + ` GAV is the total funds available to ` + "`who.address()`." + `
    }
  },
  "invariants": [
    { "notice": "The sum total amount of GAV in the system is 1 million." }
  ],
  "construction": [
    { "notice": "Endows ` + "`message.caller.address()`" + ` with 1m GAV." }
  ]
}
`)
}

var data = "0xc6888fa1000000000000000000000000000000000000000000000000000000000000007a"

var tx = `
{
  "jsonrpc": "2.0",
  "method": "eth_call",
  "params": [{
      "to": "0x8521742d3f456bd237e312d6e30724960f72517a",
      "data": "0xc6888fa1000000000000000000000000000000000000000000000000000000000000007a"
  }],
  "id": 6
}
`

var abi = []byte(`
[{
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
`)

func TestNotice(t *testing.T) {

	desc := "Will multiply `a` by 7 and return `a * 7`."
	expected := "Will multiply 122 by 7 and return 854."

	userdoc := makeUserdoc(desc)

	ns, err := NewWithDocs(abi, userdoc)
	if err != nil {
		t.Errorf("New: error: %v", err)
	}

	notice, err := ns.Notice(tx, desc)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if notice != expected {
		t.Errorf("incorrect notice. expected %v, got %v", expected, notice)
	} else {
		t.Logf("returned notice \"%v\"", notice)
	}
}

// test missing method
func TestMissingMethod(t *testing.T) {

	desc := "Will multiply `a` by 7 and return `a * 7`."
	userdoc := makeUserdoc(desc)
	expected := "natspec.js error evaluating expression: Natspec evaluation failed, method does not exist"

	ns, err := NewWithDocs(abi, userdoc)
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
	expected := "natspec.js error setting expression: (anonymous): Line 1:41 Unexpected number"

	userdoc := makeUserdoc(desc)

	ns, err := NewWithDocs(abi, userdoc)
	if err != nil {
		t.Errorf("New: error: %v", err)
	}
	notice, err := ns.Notice(tx, data)

	if err == nil {
		t.Errorf("expected error, got nothing (notice: '%v')", err, notice)
	} else {
		if err.Error() != expected {
			t.Errorf("expected error '%s' got '%v' (notice: '%v')", expected, err, notice)
		}
	}
}

// test wrong input params
func TestWrongInputParams(t *testing.T) {

	desc := "Will multiply `e` by 7 and return `a * 7`."
	expected := "natspec.js error evaluating expression: Natspec evaluation failed, wrong input params"

	userdoc := makeUserdoc(desc)

	ns, err := NewWithDocs(abi, userdoc)
	if err != nil {
		t.Errorf("New: error: %v", err)
	}

	notice, err := ns.Notice(tx, desc)

	if err == nil {
		t.Errorf("expected error, got nothing (notice: '%v')", notice)
	} else {
		if err.Error() != expected {
			t.Errorf("expected error '%s' got '%v' (notice: '%v')", expected, err, notice)
		}
	}

}
