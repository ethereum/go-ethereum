// A countdown timer that will mostly be used by XDPoS v2 consensus engine
package countdown

import (
	"fmt"
	"math"
	"time"

	"github.com/XinFinOrg/XDPoSChain/core/types"
)

const maxExponentUpperbound uint8 = 32

type ExpTimeoutDuration struct {
	duration    time.Duration
	base        float64
	maxExponent uint8
}

func NewExpTimeoutDuration(duration time.Duration, base float64, maxExponent uint8) (*ExpTimeoutDuration, error) {
	d := &ExpTimeoutDuration{
		duration:    duration,
		base:        base,
		maxExponent: maxExponent,
	}
	err := d.sanityCheck()
	return d, err
}

func (d *ExpTimeoutDuration) sanityCheck() error {
	if d.maxExponent >= maxExponentUpperbound {
		return fmt.Errorf("max_exponent (%d)= >= max_exponent_upperbound (%d)", d.maxExponent, maxExponentUpperbound)
	}
	if math.Pow(d.base, float64(d.maxExponent)) >= float64(math.MaxUint32) {
		return fmt.Errorf("base^max_exponent (%f^%d) should be less than 2^32", d.base, d.maxExponent)
	}
	return nil
}

// The inputs should be: currentRound, highestQuorumCert's round
func (d *ExpTimeoutDuration) GetTimeoutDuration(currentRound, highestRound types.Round) time.Duration {
	power := float64(1)
	// below statement must be true, just to prevent negative result
	if highestRound < currentRound {
		exp := uint8(currentRound-highestRound) - 1
		if exp > d.maxExponent {
			exp = d.maxExponent
		}
		power = math.Pow(d.base, float64(exp))
	}
	return d.duration * time.Duration(power)
}

func (d *ExpTimeoutDuration) SetParams(duration time.Duration, base float64, maxExponent uint8) error {
	prevDuration := d.duration
	prevBase := d.base
	prevME := d.maxExponent

	d.duration = duration
	d.base = base
	d.maxExponent = maxExponent
	// if parameters are wrong, should remain instead of change or panic
	if err := d.sanityCheck(); err != nil {
		d.duration = prevDuration
		d.base = prevBase
		d.maxExponent = prevME
		return err
	}
	return nil
}
