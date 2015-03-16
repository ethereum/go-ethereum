package types

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// XXX Tests doesn't really do anything. This tests exists while working on the fixed size conversions
func TestConversion(t *testing.T) {
	var (
		parent   common.Hash
		coinbase common.Address
		hash     common.Hash
	)

	NewBlock(parent, coinbase, hash, big.NewInt(0), 0, "")
}
