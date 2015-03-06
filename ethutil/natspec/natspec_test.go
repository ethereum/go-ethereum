package natspec

import (
	"testing"
)

func TestNotice(t *testing.T) {

	ns, err := NewNATSpec(`
	{
            "jsonrpc": "2.0",
            "method": "eth_call",
            "params": [{
                "to": "0x8521742d3f456bd237e312d6e30724960f72517a",
                "data": "0xc6888fa1000000000000000000000000000000000000000000000000000000000000007a"
            }],
            "id": 6
        }
	`)

	if err != nil {
		t.Errorf("NewNATSpec error %v", err)
	}

	ns.SetABI(`
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
	ns.SetDescription("Will multiply `a` by 7 and return `a * 7`.")
	ns.SetMethod("multiply")

	notice := ns.Parse()

	expected := "Will multiply 122 by 7 and return 854."
	if notice != expected {
		t.Errorf("incorrect notice. expected %v, got %v", expected, notice)
	} else {
		t.Logf("returned notice \"%v\"", notice)
	}
}
