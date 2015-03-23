package natspec

import (
	"testing"
)

func TestNotice(t *testing.T) {

	tx := `
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

	abi := `
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
	`

	desc := "Will multiply `a` by 7 and return `a * 7`."

	method := "multiply"

	ns, err := New()
	if err != nil {
		t.Errorf("NewNATSpec error %v", err)
	}

	notice, err := ns.Notice(tx, abi, method, desc)

	if err != nil {
		t.Errorf("expected no error got %v", err)
	}

	expected := "Will multiply 122 by 7 and return 854."
	if notice != expected {
		t.Errorf("incorrect notice. expected %v, got %v", expected, notice)
	} else {
		t.Logf("returned notice \"%v\"", notice)
	}

	notice, err = ns.Notice(tx, abi, method, "Will multiply 122 by \"7\" and return 854.")

	expected = "natspec.js error setting expression: (anonymous): Line 1:41 Unexpected number"

	if err == nil {
		t.Errorf("expected error, got nothing (notice: '%v')", err, notice)
	} else {
		if err.Error() != expected {
			t.Errorf("expected error '%s' got '%v' (notice: '%v')", expected, err, notice)
		}
	}

	// https://github.com/ethereum/natspec.js/issues/1
	badDesc := "Will multiply `e` by 7 and return `a * 7`."
	notice, err = ns.Notice(tx, abi, method, badDesc)

	expected = "natspec.js error evaluating expression: Natspec evaluation failed, wrong input params"

	if err == nil {
		t.Errorf("expected error, got nothing (notice: '%v')", notice)
	} else {
		if err.Error() != expected {
			t.Errorf("expected error '%s' got '%v' (notice: '%v')", expected, err, notice)
		}
	}

	notice, err = ns.Notice(tx, abi, "missing_method", desc)

	expected = "natspec.js error evaluating expression: Natspec evaluation failed, method does not exist"

	if err == nil {
		t.Errorf("expected error, got nothing (notice: '%v')", notice)
	} else {
		if err.Error() != expected {
			t.Errorf("expected error '%s' got '%v' (notice: '%v')", expected, err, notice)
		}
	}

}
