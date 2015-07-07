// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package logger

import (
	"fmt"
	"sync"
)

type stdMsg struct {
	level LogLevel
	msg   string
}

type jsonMsg []byte

func (m jsonMsg) Level() LogLevel {
	return 0
}

func (m jsonMsg) String() string {
	return string(m)
}

type LogMsg interface {
	Level() LogLevel
	fmt.Stringer
}

func (m stdMsg) Level() LogLevel {
	return m.level
}

func (m stdMsg) String() string {
	return m.msg
}

var (
	logMessageC = make(chan LogMsg)
	addSystemC  = make(chan LogSystem)
	flushC      = make(chan chan struct{})
	resetC      = make(chan chan struct{})
)

func init() {
	go dispatchLoop()
}

// each system can buffer this many messages before
// blocking incoming log messages.
const sysBufferSize = 500

func dispatchLoop() {
	var (
		systems  []LogSystem
		systemIn []chan LogMsg
		systemWG sync.WaitGroup
	)
	bootSystem := func(sys LogSystem) {
		in := make(chan LogMsg, sysBufferSize)
		systemIn = append(systemIn, in)
		systemWG.Add(1)
		go sysLoop(sys, in, &systemWG)
	}

	for {
		select {
		case msg := <-logMessageC:
			for _, c := range systemIn {
				c <- msg
			}

		case sys := <-addSystemC:
			systems = append(systems, sys)
			bootSystem(sys)

		case waiter := <-resetC:
			// reset means terminate all systems
			for _, c := range systemIn {
				close(c)
			}
			systems = nil
			systemIn = nil
			systemWG.Wait()
			close(waiter)

		case waiter := <-flushC:
			// flush means reboot all systems
			for _, c := range systemIn {
				close(c)
			}
			systemIn = nil
			systemWG.Wait()
			for _, sys := range systems {
				bootSystem(sys)
			}
			close(waiter)
		}
	}
}

func sysLoop(sys LogSystem, in <-chan LogMsg, wg *sync.WaitGroup) {
	for msg := range in {
		sys.LogPrint(msg)
	}
	wg.Done()
}

// Reset removes all active log systems.
// It blocks until all current messages have been delivered.
func Reset() {
	waiter := make(chan struct{})
	resetC <- waiter
	<-waiter
}

// Flush waits until all current log messages have been dispatched to
// the active log systems.
func Flush() {
	waiter := make(chan struct{})
	flushC <- waiter
	<-waiter
}

// AddLogSystem starts printing messages to the given LogSystem.
func AddLogSystem(sys LogSystem) {
	addSystemC <- sys
}
