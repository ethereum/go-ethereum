// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package mclock is a wrapper for a monotonic clock source
package mclock

import (
	"time"

	"github.com/aristanetworks/goarista/monotime"
)

// AbsTime represents absolute monotonic time.
type AbsTime time.Duration

// Now returns the current absolute monotonic time.
func Now() AbsTime {
	return AbsTime(monotime.Now())
}

// Clock interface makes it possible to replace the monotonic system clock with
// a simulated clock
//
// Note: event loops capable of running with a simulated clock should listen to PingChannel.
// MonotonicClock also implements this function to ensure interface compatibility.
type Clock interface {
	Now() AbsTime
	Sleep(time.Duration)
	After(time.Duration) <-chan time.Time
	PingChannel() chan struct{}
}

// MonotonicClock implements Clock using the system clock
type MonotonicClock struct{}

// PingChannel implements Clock by returning a dummy nil channel
func (MonotonicClock) PingChannel() chan struct{} {
	return nil
}

// Now implements Clock
func (MonotonicClock) Now() AbsTime {
	return AbsTime(monotime.Now())
}

// Sleep implements Clock
func (MonotonicClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

// After implements Clock
func (MonotonicClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}
