package vm

import (
	"math/big"
	"testing"
)

func TestInit(t *testing.T) {
	jumpTable := newJumpTable(ruleSet{big.NewInt(1)}, big.NewInt(0))
	if jumpTable[DELEGATECALL].valid {
		t.Error("Expected DELEGATECALL not to be present")
	}

	for _, n := range []int64{1, 2, 100} {
		jumpTable := newJumpTable(ruleSet{big.NewInt(1)}, big.NewInt(n))
		if !jumpTable[DELEGATECALL].valid {
			t.Error("Expected DELEGATECALL to be present for block", n)
		}
	}
}
