package common

import (
	"math/big"
	"testing"
)

func TestToGwei(t *testing.T) {
	var tests = []struct {
		wei  *big.Int
		gwei string
	}{
		{big.NewInt(0), "0"},
		{big.NewInt(1), "0.000000001"},
		{big.NewInt(1000), "0.000001"},
		{big.NewInt(1000000), "0.001"},
		{big.NewInt(1000000000), "1"},
		{big.NewInt(1100000000), "1.1"},
		{big.NewInt(1000100000), "1.0001"},
	}

	for _, test := range tests {
		t.Logf("%v", test.wei)
		actual := ToGWei(test.wei)
		if actual != test.gwei {
			t.Errorf("%v != %v", actual, test.gwei)
		}
	}
}

func TestToEth(t *testing.T) {
	var tests = []struct {
		wei *big.Int
		eth string
	}{
		{big.NewInt(0), "0"},
		{big.NewInt(1), "0.000000000000000001"},
		{big.NewInt(1000), "0.000000000000001"},
		{big.NewInt(1000000), "0.000000000001"},
		{big.NewInt(1000000000), "0.000000001"},
		{big.NewInt(1000000000000), "0.000001"},
		{big.NewInt(1000000000000000), "0.001"},
		{big.NewInt(1000000000000000000), "1"},
		{big.NewInt(1100000000000000000), "1.1"},
		{big.NewInt(1000100000000000000), "1.0001"},
	}

	for _, test := range tests {
		actual := ToEth(test.wei)
		if actual != test.eth {
			t.Errorf("%v != %v", actual, test.eth)
		}
	}
}
