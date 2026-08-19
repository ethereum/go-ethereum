package tests

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// TestValidateHeaderZeroBaseFee checks that a zero base fee validates
// regardless of the big.Int representation: JSON test files unmarshal "0x0"
// to a big.Int with a non-nil (empty) abs, while RLP decoding produces a nil
// abs. A reflect.DeepEqual comparison therefore rejects a numerically-equal
// zero base fee; validateHeader must compare numerically, as it already does
// for Number and Difficulty.
func TestValidateHeaderZeroBaseFee(t *testing.T) {
	// JSON side: "0x0" unmarshals to a big.Int with a non-nil abs.
	var baseFee math.HexOrDecimal256
	if err := json.Unmarshal([]byte(`"0x0"`), &baseFee); err != nil {
		t.Fatal(err)
	}

	// RLP side: a zero base fee decoded from the canonical empty string.
	var header types.Header
	header.Number = big.NewInt(1)
	header.Difficulty = big.NewInt(0)
	header.GasLimit = 30_000_000
	header.BaseFee = big.NewInt(0)
	enc, err := rlp.EncodeToBytes(&header)
	if err != nil {
		t.Fatal(err)
	}
	if err := rlp.DecodeBytes(enc, &header); err != nil {
		t.Fatal(err)
	}

	bt := &btHeader{
		Number:        big.NewInt(1),
		Difficulty:    big.NewInt(0),
		GasLimit:      30_000_000,
		BaseFeePerGas: (*big.Int)(&baseFee),
	}

	if err := validateHeader(bt, &header); err != nil {
		t.Fatalf("validateHeader rejected a numerically-equal zero base fee: %v", err)
	}
}
