package types

import (
	"math/big"
	"testing"
)

func TestFakeExponential(t *testing.T) {
	var tests = []struct {
		factor, num, denom int64
		want               int64
	}{
		// When num==0 the return value should always equal the value of factor
		{1, 0, 1, 1},
		{38493, 0, 1000, 38493},
		{0, 1234, 2345, 0}, // should be 0
		{1, 2, 1, 6},       // approximate 7.389
		{1, 4, 2, 6},
		{1, 3, 1, 16}, // approximate 20.09
		{1, 6, 2, 18},
		{1, 4, 1, 49}, // approximate 54.60
		{1, 8, 2, 50},
		{10, 8, 2, 542}, // approximate 540.598
		{11, 8, 2, 596}, // approximate 600.58
		{1, 5, 1, 136},  // approximate 148.4
		{1, 5, 2, 11},   // approximate 12.18
		{2, 5, 2, 23},   // approximate 24.36
	}

	for _, tt := range tests {
		factor := big.NewInt(tt.factor)
		num := big.NewInt(tt.num)
		denom := big.NewInt(tt.denom)
		result := fakeExponential(factor, num, denom)
		//t.Logf("%v*e^(%v/%v): %v", factor, num, denom, result)
		if tt.want != result.Int64() {
			t.Errorf("got %v want %v", result, tt.want)
		}
	}
}
