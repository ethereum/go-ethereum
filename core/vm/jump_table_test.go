package vm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/params"
)

func TestInit(t *testing.T) {
	params.HomesteadBlock = big.NewInt(1)

	jumpTable.init(big.NewInt(0))
	if jumpTable[DELEGATECALL].valid {
		t.Error("Expected DELEGATECALL not to be present")
	}

	for _, n := range []int64{1, 2, 100} {
		jumpTable.init(big.NewInt(n))
		if !jumpTable[DELEGATECALL].valid {
			t.Error("Expected DELEGATECALL to be present for block", n)
		}
	}
}
