package ethchain

import (
	"testing"

	"github.com/ethereum/go-ethereum/vm"
)

func TestBloom9(t *testing.T) {
	testCase := []byte("testtest")
	bin := LogsBloom([]vm.Log{vm.Log{testCase, nil, nil}}).Bytes()
	res := BloomLookup(bin, testCase)

	if !res {
		t.Errorf("Bloom lookup failed")
	}
}
