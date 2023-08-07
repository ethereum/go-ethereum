package countdown

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCountdownWillCallback(t *testing.T) {
	var fakeI interface{}
	called := make(chan int)
	OnTimeoutFn := func(time.Time, interface{}) error {
		called <- 1
		return nil
	}

	countdown := NewCountDown(1000 * time.Millisecond)
	countdown.OnTimeoutFn = OnTimeoutFn
	countdown.Reset(fakeI)
	<-called
	t.Log("Times up, successfully called OnTimeoutFn")
}

func TestCountdownShouldReset(t *testing.T) {
	var fakeI interface{}
	called := make(chan int)
	OnTimeoutFn := func(time.Time, interface{}) error {
		called <- 1
		return nil
	}

	countdown := NewCountDown(5000 * time.Millisecond)
	countdown.OnTimeoutFn = OnTimeoutFn
	// Check countdown did not start
	assert.False(t, countdown.isInitilised())
	countdown.Reset(fakeI)
	// Now the countdown should already started
	assert.True(t, countdown.isInitilised())
	expectedCalledTime := time.Now().Add(9000 * time.Millisecond)
	resetTimer := time.NewTimer(4000 * time.Millisecond)

firstReset:
	for {
		select {
		case <-called:
			if time.Now().After(expectedCalledTime) {
				// Make sure the countdown runs forever
				assert.True(t, countdown.isInitilised())
				t.Log("Correctly reset the countdown once")
			} else {
				t.Fatalf("Countdown did not reset correctly first time")
			}
			break firstReset
		case <-resetTimer.C:
			countdown.Reset(fakeI)
		}
	}

	// Now the countdown is paused after calling the callback function, let's reset it again
	assert.True(t, countdown.isInitilised())
	expectedTimeAfterReset := time.Now().Add(5000 * time.Millisecond)
	<-called
	// Always initilised
	assert.True(t, countdown.isInitilised())
	if time.Now().After(expectedTimeAfterReset) {
		t.Log("Correctly reset the countdown second time")
	} else {
		t.Fatalf("Countdown did not reset correctly second time")
	}
}

func TestCountdownShouldResetEvenIfErrored(t *testing.T) {
	var fakeI interface{}
	called := make(chan int)
	OnTimeoutFn := func(time.Time, interface{}) error {
		called <- 1
		return fmt.Errorf("ERROR!")
	}

	countdown := NewCountDown(5000 * time.Millisecond)
	countdown.OnTimeoutFn = OnTimeoutFn
	// Check countdown did not start
	assert.False(t, countdown.isInitilised())
	countdown.Reset(fakeI)
	// Now the countdown should already started
	assert.True(t, countdown.isInitilised())
	expectedCalledTime := time.Now().Add(9000 * time.Millisecond)
	resetTimer := time.NewTimer(4000 * time.Millisecond)

firstReset:
	for {
		select {
		case <-called:
			if time.Now().After(expectedCalledTime) {
				// Make sure the countdown runs forever
				assert.True(t, countdown.isInitilised())
				t.Log("Correctly reset the countdown once")
			} else {
				t.Fatalf("Countdown did not reset correctly first time")
			}
			break firstReset
		case <-resetTimer.C:
			countdown.Reset(fakeI)
		}
	}

	// Now the countdown is paused after calling the callback function, let's reset it again
	assert.True(t, countdown.isInitilised())
	expectedTimeAfterReset := time.Now().Add(5000 * time.Millisecond)
	<-called
	// Always initilised
	assert.True(t, countdown.isInitilised())
	if time.Now().After(expectedTimeAfterReset) {
		t.Log("Correctly reset the countdown second time")
	} else {
		t.Fatalf("Countdown did not reset correctly second time")
	}
}

func TestCountdownShouldBeAbleToStop(t *testing.T) {
	var fakeI interface{}
	called := make(chan int)
	OnTimeoutFn := func(time.Time, interface{}) error {
		called <- 1
		return nil
	}

	countdown := NewCountDown(5000 * time.Millisecond)
	countdown.OnTimeoutFn = OnTimeoutFn
	// Check countdown did not start
	assert.False(t, countdown.isInitilised())
	countdown.Reset(fakeI)
	// Now the countdown should already started
	assert.True(t, countdown.isInitilised())
	// Try manually stop the timer before it triggers the callback
	stopTimer := time.NewTimer(4000 * time.Millisecond)
	<-stopTimer.C
	countdown.StopTimer()
	assert.False(t, countdown.isInitilised())
}

func TestCountdownShouldAvoidDeadlock(t *testing.T) {
	var fakeI interface{}
	called := make(chan int)
	countdown := NewCountDown(5000 * time.Millisecond)
	OnTimeoutFn := func(time.Time, interface{}) error {
		countdown.Reset(fakeI)
		called <- 1
		return nil
	}

	countdown.OnTimeoutFn = OnTimeoutFn
	countdown.Reset(fakeI)
	<-called
}
