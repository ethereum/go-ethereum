package fees

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateEncodedL1DataFee(t *testing.T) {
	l1BaseFee := new(big.Int).SetUint64(15000000)

	data := []byte{0, 10, 1, 0}
	overhead := new(big.Int).SetUint64(100)
	scalar := new(big.Int).SetUint64(10)

	expected := new(big.Int).SetUint64(184) // 184.2
	actual := calculateEncodedL1DataFee(data, overhead, l1BaseFee, scalar)
	assert.Equal(t, expected, actual)
}
