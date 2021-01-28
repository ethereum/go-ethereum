package misc

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func TestBlockElasticity(t *testing.T) {
	initial := new(big.Int).SetUint64(params.InitialBaseFee)
	parent := &types.Header{
		GasUsed:  10000000,
		GasLimit: 10000000,
		BaseFee:  initial,
	}
	header := &types.Header{
		GasUsed:  20000000,
		GasLimit: 10000000,
		BaseFee:  initial,
	}
	if err := VerifyEip1559Header(parent, header, false); err != nil {
		t.Errorf("Expected valid header: %s", err)
	}
	header.GasUsed += 1
	expected := fmt.Sprintf("exceeded elasticity multiplier: gasUsed %d, gasTarget*elasticityMultiplier %d", header.GasUsed, header.GasLimit*params.ElasticityMultiplier)
	if err := VerifyEip1559Header(parent, header, false); fmt.Sprint(err) != expected {
		t.Errorf("Expected invalid header")
	}
}

func TestCalcBaseFee(t *testing.T) {
	tests := []struct {
		parentBaseFee   *big.Int
		parentGasLimit  uint64
		parentGasUsed   uint64
		expectedBaseFee *big.Int
	}{
		// baseFee should remain unchaned when the gasUsed is equal to the gasTarget
		{
			new(big.Int).SetUint64(params.InitialBaseFee),
			10000000,
			10000000,
			new(big.Int).SetUint64(params.InitialBaseFee),
		},
		// baseFee should decrease when the gasUsed is below the gasTarget
		{
			new(big.Int).SetUint64(params.InitialBaseFee),
			10000000,
			9000000,
			new(big.Int).SetUint64(987500000),
		},
		// baseFee should increase when the gasUsed is below the gasTarget
		{
			new(big.Int).SetUint64(params.InitialBaseFee),
			10000000,
			11000000,
			new(big.Int).SetUint64(1012500000),
		},
	}
	for i, test := range tests {
		parent := &types.Header{
			GasLimit: test.parentGasLimit,
			GasUsed:  test.parentGasUsed,
			BaseFee:  test.parentBaseFee,
		}
		baseFee := CalcBaseFee(parent)
		if baseFee.Cmp(test.expectedBaseFee) != 0 {
			t.Errorf("Test %d: expected %d, got %d", i+1, test.expectedBaseFee.Int64(), baseFee.Int64())
		}
	}
}
