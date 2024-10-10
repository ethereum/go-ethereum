package eip7783

import (
	"math/big"
	"testing"
)

func TestCalcGasLimitEIP7783Test(t *testing.T) {
	// Do multiple tests here
	tests := []struct {
		blockNum, startBlockNum                                         *big.Int
		initialGasLimit, gasIncreaseRate, gasLimitCap, expectedGasLimit uint64
	}{
		{big.NewInt(100), big.NewInt(50), 100000, 10, 200000, 100500},
		{big.NewInt(100), big.NewInt(100), 100000, 10, 200000, 100000},
		{big.NewInt(99), big.NewInt(100), 100000, 10, 200000, 100000},
	}

	for i, test := range tests {
		if have, want := CalcGasLimitEIP7783(test.blockNum, test.startBlockNum, test.initialGasLimit, test.gasIncreaseRate, test.gasLimitCap), test.expectedGasLimit; have != want {
			t.Errorf("test %d: have %d  want %d, ", i, have, want)
		}
	}
}
