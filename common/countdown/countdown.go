// A countdown timer that will mostly be used by XDPoS v2 consensus engine
package countdown

import (
	"sync"
	"time"

	"github.com/XinFinOrg/XDPoSChain/log"
)

type CountdownTimer struct {
	lock            sync.RWMutex // Protects the Initilised field
	resetc          chan int
	quitc           chan chan struct{}
	initilised      bool
	timeoutDuration time.Duration
	// Triggered when the countdown timer timeout for the `timeoutDuration` period, it will pass current timestamp to the callback function
	OnTimeoutFn func(time time.Time, i interface{}) error
}

func NewCountDown(duration time.Duration) *CountdownTimer {
	return &CountdownTimer{
		resetc:          make(chan int),
		quitc:           make(chan chan struct{}),
		initilised:      false,
		timeoutDuration: duration,
	}
}

// Completely stop the countdown timer from running.
func (t *CountdownTimer) StopTimer() {
	q := make(chan struct{})
	t.quitc <- q
	<-q
}

func (t *CountdownTimer) SetTimeoutDuration(duration time.Duration) {
	t.timeoutDuration = duration
}

// Reset will start the countdown timer if it's already stopped, or simply reset the countdown time back to the defual `duration`
func (t *CountdownTimer) Reset(i interface{}) {
	if !t.isInitilised() {
		t.setInitilised(true)
		go t.startTimer(i)
	} else {
		t.resetc <- 0
	}
}

// A long running process that
func (t *CountdownTimer) startTimer(i interface{}) {
	// Make sure we mark Initilised to false when we quit the countdown
	defer t.setInitilised(false)
	timer := time.NewTimer(t.timeoutDuration)
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
			timer.Reset(t.timeoutDuration)
		case <-t.resetc:
			log.Debug("Reset countdown timer")
			timer.Reset(t.timeoutDuration)
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
