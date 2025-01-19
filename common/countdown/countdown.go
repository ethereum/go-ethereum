// A countdown timer that will mostly be used by XDPoS v2 consensus engine
package countdown

import (
	"sync"
	"time"

	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/log"
)

type TimeoutDurationHelper interface {
	GetTimeoutDuration(types.Round, types.Round) time.Duration
	SetParams(time.Duration, float64, uint8) error
}

type CountdownTimer struct {
	lock           sync.RWMutex // Protects the Initilised field
	resetc         chan ResetInfo
	quitc          chan chan struct{}
	initilised     bool
	durationHelper TimeoutDurationHelper
	// Triggered when the countdown timer timeout for the `timeoutDuration` period, it will pass current timestamp to the callback function
	OnTimeoutFn func(time time.Time, i interface{}) error
}

type ResetInfo struct {
	currentRound, highestRound types.Round
}

func NewExpCountDown(duration time.Duration, base float64, max_exponent uint8) (*CountdownTimer, error) {
	durationHelper, err := NewExpTimeoutDuration(duration, base, max_exponent)
	if err != nil {
		return nil, err
	}
	return &CountdownTimer{
		resetc:         make(chan ResetInfo),
		quitc:          make(chan chan struct{}),
		initilised:     false,
		durationHelper: durationHelper,
	}, nil
}

// Completely stop the countdown timer from running.
func (t *CountdownTimer) StopTimer() {
	q := make(chan struct{})
	t.quitc <- q
	<-q
}

func (t *CountdownTimer) SetParams(duration time.Duration, base float64, maxExponent uint8) error {
	return t.durationHelper.SetParams(duration, base, maxExponent)
}

// Reset will start the countdown timer if it's already stopped, or simply reset the countdown time back to the defual `duration`
func (t *CountdownTimer) Reset(i interface{}, currentRound, highestRound types.Round) {
	if !t.isInitilised() {
		t.setInitilised(true)
		go t.startTimer(i, currentRound, highestRound)
	} else {
		t.resetc <- ResetInfo{currentRound, highestRound}
	}
}

// A long running process that
func (t *CountdownTimer) startTimer(i interface{}, currentRound, highestRound types.Round) {
	// Make sure we mark Initilised to false when we quit the countdown
	defer t.setInitilised(false)
	timer := time.NewTimer(t.durationHelper.GetTimeoutDuration(currentRound, highestRound))
	// We start with a inf loop
	for {
		select {
		case q := <-t.quitc:
			log.Debug("Quit countdown timer")
			close(q)
			return
		case <-timer.C:
			log.Debug("Countdown time reached!")
			go func() {
				err := t.OnTimeoutFn(time.Now(), i)
				if err != nil {
					log.Error("OnTimeoutFn error", "error", err)
				}
				log.Debug("OnTimeoutFn processed")
			}()
			timer.Reset(t.durationHelper.GetTimeoutDuration(currentRound, highestRound))
		case info := <-t.resetc:
			log.Debug("Reset countdown timer")
			currentRound = info.currentRound
			highestRound = info.highestRound
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(t.durationHelper.GetTimeoutDuration(currentRound, highestRound))
		}
	}
}

// Set the desired value to Initilised with lock to avoid race condition
func (t *CountdownTimer) setInitilised(value bool) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.initilised = value
}

func (t *CountdownTimer) isInitilised() bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.initilised
}
