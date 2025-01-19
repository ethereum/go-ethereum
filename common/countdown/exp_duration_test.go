package countdown

import (
	"math"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/stretchr/testify/assert"
)

func TestExpDuration(t *testing.T) {
	base := float64(2.0)
	max_exponent := uint8(2)
	duration := time.Second * 59
	helper, err := NewExpTimeoutDuration(duration, base, max_exponent)
	assert.Nil(t, err)
	// round 10 = 9+1, normal case, should be base
	currentRound := types.Round(10)
	highestQCRound := types.Round(9)
	result := helper.GetTimeoutDuration(currentRound, highestQCRound)
	assert.Equal(t, duration, result)

	// round 11 = 9+2, already 1 round timeout, should be base*exponent
	currentRound++
	result = helper.GetTimeoutDuration(currentRound, highestQCRound)
	assert.Equal(t, duration*time.Duration(base), result)

	// round 12 = 9+3, already 2 rounds timeout, should be base*exponent^2
	currentRound++
	result = helper.GetTimeoutDuration(currentRound, highestQCRound)
	assert.Equal(t, duration*time.Duration(base)*time.Duration(base), result)

	// test SetParams
	duration++
	max_exponent++
	base++
	helper.SetParams(duration, base, max_exponent)
	result = helper.GetTimeoutDuration(currentRound, highestQCRound)
	assert.Equal(t, duration*time.Duration(base)*time.Duration(base), result)
	// round 14 = 9+5, already 4 rounds timeout, but max_exponent=3, should be base*exponent^3
	currentRound++
	currentRound++
	result = helper.GetTimeoutDuration(currentRound, highestQCRound)
	assert.Equal(t, duration*time.Duration(math.Pow(base, float64(3))), result)

	// extreme case
	helper.SetParams(duration, float64(0), uint8(0))
	result = helper.GetTimeoutDuration(currentRound, highestQCRound)
	assert.Equal(t, duration, result)
}

func TestInvalidParameter(t *testing.T) {
	base := float64(2.0)
	max_exponent := uint8(32)
	duration := time.Second * 59
	_, err := NewExpTimeoutDuration(duration, base, max_exponent)
	assert.Error(t, err)

	base = float64(3.0)
	max_exponent = uint8(21)
	duration = time.Second * 59
	_, err = NewExpTimeoutDuration(duration, base, max_exponent)
	assert.Error(t, err)
}
