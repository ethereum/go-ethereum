// Copyright 2019 The go-ethereum Authors
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

package csvlogger

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/log"
)

// Logger is a metrics/events logger that writes logged values and events into a comma separated file
type Logger struct {
	file            *os.File
	started         mclock.AbsTime
	channels        []*Channel
	period          time.Duration
	stopCh, stopped chan struct{}
	storeCh         chan string
	eventHeader     string
}

// NewLogger creates a new Logger
func NewLogger(fileName string, updatePeriod time.Duration, eventHeader string) *Logger {
	if fileName == "" {
		return nil
	}
	f, err := os.Create(fileName)
	if err != nil {
		log.Error("Error creating log file", "name", fileName, "error", err)
		return nil
	}
	return &Logger{
		file:        f,
		period:      updatePeriod,
		stopCh:      make(chan struct{}),
		storeCh:     make(chan string, 1),
		eventHeader: eventHeader,
	}
}

// NewChannel creates a new value logger channel that writes values in a single
// column. If the relative change of the value is bigger than the given threshold
// then a new line is added immediately (threshold can also be 0).
func (l *Logger) NewChannel(name string, threshold float64) *Channel {
	if l == nil {
		return nil
	}
	c := &Channel{
		logger:    l,
		name:      name,
		threshold: threshold,
	}
	l.channels = append(l.channels, c)
	return c
}

// NewMinMaxChannel creates a new value logger channel that writes the minimum and
// maximum of the tracked value in two columns. It never triggers adding a new line.
// If zeroDefault is true then 0 is written to both min and max columns if no update
// was given during the last period. If it is false then the last update will appear
// in both columns.
func (l *Logger) NewMinMaxChannel(name string, zeroDefault bool) *Channel {
	if l == nil {
		return nil
	}
	c := &Channel{
		logger:        l,
		name:          name,
		minmax:        true,
		mmZeroDefault: zeroDefault,
	}
	l.channels = append(l.channels, c)
	return c
}

func (l *Logger) store(event string) {
	s := fmt.Sprintf("%g", float64(mclock.Now()-l.started)/1000000000)
	for _, ch := range l.channels {
		s += ", " + ch.store()
	}
	if event != "" {
		s += ", " + event
	}
	l.file.WriteString(s + "\n")
}

// Start writes the header line and starts the logger
func (l *Logger) Start() {
	if l == nil {
		return
	}
	l.started = mclock.Now()
	s := "Time"
	for _, ch := range l.channels {
		s += ", " + ch.header()
	}
	if l.eventHeader != "" {
		s += ", " + l.eventHeader
	}
	l.file.WriteString(s + "\n")
	go func() {
		timer := time.NewTimer(l.period)
		for {
			select {
			case <-timer.C:
				l.store("")
				timer.Reset(l.period)
			case event := <-l.storeCh:
				l.store(event)
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(l.period)
			case <-l.stopCh:
				close(l.stopped)
				return
			}
		}
	}()
}

// Stop stops the logger and closes the file
func (l *Logger) Stop() {
	if l == nil {
		return
	}
	l.stopped = make(chan struct{})
	close(l.stopCh)
	<-l.stopped
	l.file.Close()
}

// Event immediately adds a new line and adds the given event string in the last column
func (l *Logger) Event(event string) {
	if l == nil {
		return
	}
	select {
	case l.storeCh <- event:
	case <-l.stopCh:
	}
}

// Channel represents a logger channel tracking a single value
type Channel struct {
	logger                                             *Logger
	lock                                               sync.Mutex
	name                                               string
	threshold, storeMin, storeMax, lastValue, min, max float64
	minmax, mmSet, mmZeroDefault                       bool
}

// Update updates the tracked value
func (lc *Channel) Update(value float64) {
	if lc == nil {
		return
	}
	lc.lock.Lock()
	defer lc.lock.Unlock()

	lc.lastValue = value
	if lc.minmax {
		if value > lc.max || !lc.mmSet {
			lc.max = value
		}
		if value < lc.min || !lc.mmSet {
			lc.min = value
		}
		lc.mmSet = true
	} else {
		if value < lc.storeMin || value > lc.storeMax {
			select {
			case lc.logger.storeCh <- "":
			default:
			}
		}
	}
}

func (lc *Channel) store() (s string) {
	lc.lock.Lock()
	defer lc.lock.Unlock()

	if lc.minmax {
		s = fmt.Sprintf("%g, %g", lc.min, lc.max)
		lc.mmSet = false
		if lc.mmZeroDefault {
			lc.min = 0
		} else {
			lc.min = lc.lastValue
		}
		lc.max = lc.min
	} else {
		s = fmt.Sprintf("%g", lc.lastValue)
		lc.storeMin = lc.lastValue * (1 - lc.threshold)
		lc.storeMax = lc.lastValue * (1 + lc.threshold)
		if lc.lastValue < 0 {
			lc.storeMin, lc.storeMax = lc.storeMax, lc.storeMin
		}
	}
	return
}

func (lc *Channel) header() string {
	if lc.minmax {
		return lc.name + " (min), " + lc.name + " (max)"
	}
	return lc.name
}
